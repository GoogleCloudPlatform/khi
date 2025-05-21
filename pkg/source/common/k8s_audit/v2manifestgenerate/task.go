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

package v2manifestgenerate

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structurev2"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/rtype"
	common_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var bodyPlaceholderForMetadataLevelAuditLog = "# Resource data is unavailable. Audit logs for this resource is recorded at metadata level."

var Task = inspection_task.NewProgressReportableInspectionTask(common_k8saudit_taskid.ManifestGenerateTaskID, []taskid.UntypedTaskReference{
	common_k8saudit_taskid.TimelineGroupingTaskID.Ref(),
	gcp_task.K8sResourceMergeConfigTaskID.Ref(),
}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, tp *progress.TaskProgress) ([]*types.TimelineGrouperResult, error) {
	if taskMode == inspection_task_interface.TaskModeDryRun {
		return nil, nil
	}
	groups := task.GetTaskResult(ctx, common_k8saudit_taskid.TimelineGroupingTaskID.Ref())
	mergeConfigRegistry := task.GetTaskResult(ctx, gcp_task.GCPDefaultK8sResourceMergeConfigTask.ID().Ref())

	totalLogCount := 0
	for _, group := range groups {
		totalLogCount += len(group.PreParsedLogs)
	}
	processedCount := atomic.Int32{}
	updator := progress.NewProgressUpdator(tp, time.Second, func(tp *progress.TaskProgress) {
		current := processedCount.Load()
		tp.Percentage = float32(current) / float32(totalLogCount)
		tp.Message = fmt.Sprintf("%d/%d", current, totalLogCount)
	})
	err := updator.Start(ctx)
	if err != nil {
		return nil, err
	}
	defer updator.Done()
	workerPool := worker.NewPool(16)
	for _, group := range groups {
		currentGroup := group
		workerPool.Run(func() {
			prevRevisionBody := ""
			prevRevisionReader := structurev2.NewNodeReader(structurev2.NewEmptyMapNode())
			for _, log := range currentGroup.PreParsedLogs {
				var currentRevisionBodyType rtype.Type
				if log.IsErrorResponse || log.GeneratedFromDeleteCollectionOperation {
					log.ResourceBodyYaml = prevRevisionBody
					log.ResourceBodyReader = prevRevisionReader
					processedCount.Add(1)
					continue
				}
				currentRevisionReader := log.Response
				currentRevisionBodyType = log.ResponseType
				if currentRevisionReader == nil || log.ResponseType != rtype.RTypeUnknown {
					currentRevisionReader = log.Request
					currentRevisionBodyType = log.RequestType
				}

				// Manifest is unknown because it doesn't contain request or response in the body.
				if currentRevisionReader == nil {
					log.ResourceBodyYaml = bodyPlaceholderForMetadataLevelAuditLog
					processedCount.Add(1)
					continue
				}

				isPartial := currentRevisionBodyType == rtype.RTypePatch
				currentRevisionBodyRaw, err := currentRevisionReader.Serialize("", &structurev2.YAMLNodeSerializer{})
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to serialize resource body to yaml\n%s", err.Error()))
					processedCount.Add(1)
					continue
				}
				currentRevisionBody := string(currentRevisionBodyRaw)
				currentRevisionBody = removeAtType(currentRevisionBody)

				if isPartial {
					mergeConfigResolver := mergeConfigRegistry.Get(log.Operation.APIVersion, log.Operation.GetSingularKindName())
					mergedNode, err := structurev2.MergeNode(prevRevisionReader.Node, currentRevisionReader.Node, structurev2.MergeConfiguration{
						MergeMapOrderStrategy:    &structurev2.DefaultMergeMapOrderStrategy{},
						ArrayMergeConfigResolver: mergeConfigResolver,
					})
					if err != nil {
						slog.WarnContext(ctx, fmt.Sprintf("failed to merge resource body\n%s", err.Error()))
						processedCount.Add(1)
						continue
					}
					mergedNodeReader := structurev2.NewNodeReader(mergedNode)
					mergedYaml, err := mergedNodeReader.Serialize("", &structurev2.YAMLNodeSerializer{})
					if err != nil {
						slog.WarnContext(ctx, fmt.Sprintf("failed to read the merged resource body\n%s", err.Error()))
						processedCount.Add(1)
						continue
					}
					log.ResourceBodyYaml = removeAtType(string(mergedYaml))
					log.ResourceBodyReader = mergedNodeReader
				} else {
					if currentRevisionBodyType == rtype.RTypeDeleteOptions {
						log.ResourceBodyYaml = prevRevisionBody
						log.ResourceBodyReader = prevRevisionReader
						processedCount.Add(1)
						continue
					}
					log.ResourceBodyYaml = currentRevisionBody
					log.ResourceBodyReader = currentRevisionReader
				}
				prevRevisionBody = log.ResourceBodyYaml
				prevRevisionReader = log.ResourceBodyReader
				processedCount.Add(1)
			}
		})
	}
	workerPool.Wait()
	return groups, nil
})

// Remove @type in response or request payload
func removeAtType(yamlString string) string {
	if strings.Contains(yamlString, "'@type'") {
		index := strings.Index(yamlString, "\n")
		return yamlString[index+1:]
	}
	return yamlString
}
