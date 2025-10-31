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
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// HistoryModifer defines the interface for modifying the History with change sets based on log entries.
// Implementations of this interface can be used to customize how log data is transformed into
// structured history.
// To process data generated from processing the last log in the same group, the method ModifyChangeSetFromLog receive and return a variable typed T.
type HistoryModifer[T any] interface {
	// LogSerializerTask is one of prerequiste task of HistoryModifier serializes its logs to history data before processing with this modifier.
	LogSerializerTask() taskid.TaskReference[[]*log.Log]
	// Dependencies are the additional references used in history modifier.
	Dependencies() []taskid.UntypedTaskReference
	// GroupedLogTask returns a reference to the task that provides the grouped logs.
	GroupedLogTask() taskid.TaskReference[LogGroupMap]
	// ModifyChangeSetFromLog is called for each log entry to modify the corresponding ChangeSet.
	// This method allows for custom logic to be applied during the history building process.
	// The prevGroupData is the returned value from the last procesed log in the same group.
	ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData T) (T, error)
}

// NewHistoryModifierTask creates a task that modifies the history builder based on grouped logs.
// It processes logs in parallel and applies the logic from the provided HistoryModifer
// to build a comprehensive history of events.
func NewHistoryModifierTask[T any](tid taskid.TaskImplementationID[struct{}], historyModifier HistoryModifer[T], labels ...coretask.LabelOpt) coretask.Task[struct{}] {
	groupedLogTaskID := historyModifier.GroupedLogTask()
	dependencies := append([]taskid.UntypedTaskReference{historyModifier.LogSerializerTask(), historyModifier.GroupedLogTask()}, historyModifier.Dependencies()...)
	return NewProgressReportableInspectionTask(tid, dependencies, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) (struct{}, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			slog.DebugContext(ctx, "Skipping task because this is dry run mode")
			return struct{}{}, nil
		}

		builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)
		groupedLogs := coretask.GetTaskResult(ctx, groupedLogTaskID)

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

		processedLogCount.Store(0)

		pool := worker.NewPool(16)
		for _, group := range groupedLogs {
			pool.Run(func() {
				defer errorreport.CheckAndReportPanic()

				var groupData T
				err := builder.ParseLogsByGroups(ctx, group.Logs, func(logIndex int, l *log.Log) *history.ChangeSet {
					processedLogCount.Add(1)
					var err error
					cs := history.NewChangeSet(l)
					groupData, err = historyModifier.ModifyChangeSetFromLog(ctx, l, cs, builder, groupData)
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
					return cs
				})
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to complete parsing logs for group %s\nerr: %s", group.Group, err.Error()))
				}
			})
		}
		pool.Wait()
		updator.Done()

		return struct{}{}, nil
	}, append([]coretask.LabelOpt{
		// Tasks modifying history must be dependent from SerializerTask.
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref())}, labels...)...)
}
