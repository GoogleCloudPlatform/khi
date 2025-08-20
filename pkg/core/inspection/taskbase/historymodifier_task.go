// Copyright 2025 Google LLC
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

package inspectiontaskbase

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/errorreport"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
)

type HistoryModifer interface {
	GroupedLogTask() taskid.TaskReference[LogGroupMap]
	ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error
}

func NewHistoryModifierTask(tid taskid.TaskImplementationID[struct{}], historyModifier HistoryModifer) coretask.Task[struct{}] {
	return NewProgressReportableInspectionTask(tid, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspection_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) (struct{}, error) {
		if taskMode == inspection_contract.TaskModeDryRun {
			slog.DebugContext(ctx, "Skipping task because this is dry run mode")
			return struct{}{}, nil
		}

		builder := khictx.MustGetValue(ctx, inspection_contract.CurrentHistoryBuilder)
		groupedLogs := coretask.GetTaskResult(ctx, historyModifier.GroupedLogTask())

		totalLogCount := 0
		var processedLogCount atomic.Uint32
		for _, group := range groupedLogs {
			totalLogCount += len(group.Logs)
		}

		updator := progressutil.NewProgressUpdator(tp, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := processedLogCount.Load()
			tp.Percentage = float32(current) / float32(totalLogCount)
			tp.Message = fmt.Sprintf("%d/%d", current, totalLogCount)
		})
		updator.Start(ctx)

		pool := worker.NewPool(16)
		for _, group := range groupedLogs {
			pool.Run(func() {
				defer errorreport.CheckAndReportPanic()
				err := builder.ParseLogsByGroups(ctx, group.Logs, func(logIndex int, l *log.Log) *history.ChangeSet {
					cs := history.NewChangeSet(l)
					err := historyModifier.ModifyChangeSetFromLog(ctx, l, cs, builder)
					if err != nil {
						var yaml string
						yamlBytes, err2 := l.Serialize("", &structured.YAMLNodeSerializer{})
						if err2 != nil {
							yaml = "ERROR!! failed to dump in yaml"
						} else {
							yaml = string(yamlBytes)
						}
						slog.WarnContext(ctx, fmt.Sprintf("parser end with an error\n%s", err))
						slog.DebugContext(ctx, yaml)
						return nil
					}
					return nil
				})
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to complete parsing logs for group %s\nerr: %s", group.Group, err.Error()))
				}
			})
		}
		pool.Wait()
		updator.Done()

		return struct{}{}, nil
	})
}
