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

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	common_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var Task = inspection_task.NewInspectionTask(common_k8saudit_taskid.CommonLogParseTaskID, []taskid.UntypedTaskReference{
	common_k8saudit_taskid.CommonAuitLogSource,
}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, tp *progress.TaskProgress) ([]*types.AuditLogParserInput, error) {
	if taskMode == inspection_task_interface.TaskModeDryRun {
		return nil, nil
	}
	source := task.GetTaskResult(ctx, common_k8saudit_taskid.CommonAuitLogSource)

	processedCount := atomic.Int32{}
	progressUpdater := progress.NewProgressUpdator(tp, time.Second, func(tp *progress.TaskProgress) {
		current := processedCount.Load()
		tp.Percentage = float32(current) / float32(len(source.Logs))
		tp.Message = fmt.Sprintf("%d/%d", current, len(source.Logs))
	})
	err := progressUpdater.Start(ctx)
	if err != nil {
		return nil, err
	}
	defer progressUpdater.Done()
	parsedLogs := make([]*types.AuditLogParserInput, len(source.Logs))
	wg := sync.WaitGroup{}
	concurrency := 16
	for i := 0; i < concurrency; i++ {
		thread := i
		wg.Add(1)
		go func(t int) {
			for l := t; l < len(source.Logs); l += concurrency {
				log := source.Logs[l]
				prestep, err := source.Extractor.ExtractFields(ctx, log)
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
	parsedLogsWithoutError := []*types.AuditLogParserInput{}
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
