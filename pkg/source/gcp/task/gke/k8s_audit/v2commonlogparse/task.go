// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v2commonlogparse

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/k8s"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/k8saudittask"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/rtype"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

// ParseResourceSpecificParserInputWithoutResourceBody returns ResourceSpecificParserInput from a single LogEntity.
// This function only parses data independent to the other logs.
// ResourceBody must be always empty and it's expected to be filled later.
func ParseResourceSpecificParserInputWithoutResourceBody(ctx context.Context, l *log.LogEntity) (*types.ResourceSpecificParserInput, error) {
	resourceName, err := l.GetString("protoPayload.resourceName")
	if err != nil {
		return nil, err
	}

	methodName, err := l.GetString("protoPayload.methodName")
	if err != nil {
		return nil, err
	}

	principalEmail := l.GetStringOrDefault("protoPayload.authenticationInfo.principalEmail", "")

	operation := k8s.ParseKubernetesOperation(resourceName, methodName)
	// /status subresource contains the actual content of the parent.
	// It's easier to see timeline merged with the parent timeline instead of showing status as the single subresource timeline.
	// TODO: There would be the other subresources needed to be cared like this.
	if operation.SubResourceName == "status" {
		operation.SubResourceName = ""
	}

	code := l.GetIntOrDefault("protoPayload.status.code", 0)

	requestType := rtype.RTypeUnknown
	request, _ := l.Fields.ReaderSingle("protoPayload.request")
	if request != nil && request.Has("@type") {
		rtypeInStr := request.ReadStringOrDefault("@type", "")
		if rt, found := rtype.Types[rtypeInStr]; found {
			requestType = rt
		}
	}

	responseType := rtype.RTypeUnknown
	response, _ := l.Fields.ReaderSingle("protoPayload.response")
	if response != nil && response.Has("@type") {
		rtypeInStr := response.ReadStringOrDefault("@type", "")
		if rt, found := rtype.Types[rtypeInStr]; found {
			responseType = rt
		}
	}

	return &types.ResourceSpecificParserInput{
		Log:            l,
		ResourceName:   resourceName,
		MethodName:     methodName,
		PrincipalEmail: principalEmail,
		Operation:      operation,
		Code:           code,
		Request:        request,
		RequestType:    requestType,
		Response:       response,
		ResponseType:   responseType,
	}, nil
}

var Task = inspection_task.NewInspectionProcessor(k8saudittask.CommonLogParseTaskID, []string{
	k8saudittask.K8sAuditQueryTaskID,
}, func(ctx context.Context, taskMode int, v *task.VariableSet, tp *progress.TaskProgress) (any, error) {
	if taskMode == inspection_task.TaskModeDryRun {
		return struct{}{}, nil
	}
	logs, err := task.GetTypedVariableFromTaskVariable[[]*log.LogEntity](v, k8saudittask.K8sAuditQueryTaskID, nil)
	if err != nil {
		return nil, err
	}
	processedCount := atomic.Int32{}
	progressUpdater := progress.NewProgressUpdator(tp, time.Second, func(tp *progress.TaskProgress) {
		current := processedCount.Load()
		tp.Percentage = float32(current) / float32(len(logs))
		tp.Message = fmt.Sprintf("%d/%d", current, len(logs))
	})
	err = progressUpdater.Start(ctx)
	if err != nil {
		return nil, err
	}
	defer progressUpdater.Done()
	parsedLogs := make([]*types.ResourceSpecificParserInput, len(logs))
	wg := sync.WaitGroup{}
	concurrency := 16
	for i := 0; i < concurrency; i++ {
		thread := i
		wg.Add(1)
		go func(t int) {
			for l := t; l < len(logs); l += concurrency {
				log := logs[l]
				prestep, err := ParseResourceSpecificParserInputWithoutResourceBody(ctx, log)
				if err != nil {
					continue
				}
				parsedLogs[l] = prestep
				processedCount.Add(1)
			}
			wg.Done()
		}(thread)
	}
	wg.Wait()
	parsedLogsWithoutError := []*types.ResourceSpecificParserInput{}
	for _, parsed := range parsedLogs {
		if parsed == nil {
			continue
		}
		parsedLogsWithoutError = append(parsedLogsWithoutError, parsed)
	}
	if len(parsedLogsWithoutError) < len(parsedLogs) {
		slog.WarnContext(ctx, fmt.Sprintf("Failed to parse %d count of logs in the prestep phase", len(parsedLogs)-len(parsedLogsWithoutError)))
	}
	return parsedLogsWithoutError, nil
})
