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

package commonlogk8saudit_impl

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var CommonLogParserTask = inspectiontaskbase.NewProgressReportableInspectionTask(commonlogk8saudit_contract.CommonLogParseTaskID, []taskid.UntypedTaskReference{
	commonlogk8saudit_contract.CommonAuitLogSource,
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) ([]*commonlogk8saudit_contract.AuditLogParserInput, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return nil, nil
	}
	source := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.CommonAuitLogSource)

	processedCount := atomic.Int32{}
	progressUpdater := progressutil.NewProgressUpdator(tp, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
		current := processedCount.Load()
		tp.Percentage = float32(current) / float32(len(source.Logs))
		tp.Message = fmt.Sprintf("%d/%d", current, len(source.Logs))
	})
	err := progressUpdater.Start(ctx)
	if err != nil {
		return nil, err
	}
	defer progressUpdater.Done()
	parsedLogs := make([]*commonlogk8saudit_contract.AuditLogParserInput, len(source.Logs))
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
	parsedLogsWithoutError := []*commonlogk8saudit_contract.AuditLogParserInput{}
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
