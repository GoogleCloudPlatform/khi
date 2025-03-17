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

package recorder

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/constants"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"

	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

type LogGroupFilterFunc = func(ctx context.Context, resourcePath string) bool

type LogFilterFunc = func(ctx context.Context, l *types.AuditLogParserInput) bool

// RecorderFunc records events/revisions...etc on the given ChangeSet. If it returns an error, then the result is ignored.
type RecorderFunc = func(ctx context.Context, resourcePath string, currentLog *types.AuditLogParserInput, prevStateInGroup any, cs *history.ChangeSet, builder *history.Builder, vs *task.VariableSet) (any, error)

// RecorderTaskManager provides the way of extending resource specific
type RecorderTaskManager struct {
	taskID        taskid.TaskImplementationId
	recorderTasks []task.Definition
}

func NewAuditRecorderTaskManager(taskID taskid.TaskImplementationId) *RecorderTaskManager {
	return &RecorderTaskManager{
		taskID:        taskID,
		recorderTasks: make([]task.Definition, 0),
	}
}

func (r *RecorderTaskManager) AddRecorder(name string, dependencies []string, recorder RecorderFunc, logGroupFilter LogGroupFilterFunc, logFilter LogFilterFunc) {
	dependenciesBase := []string{
		inspection_task.BuilderGeneratorTaskID,
		constants.LogConvertTaskID,
		constants.ManifestGenerateTaskID,
	}
	newTask := inspection_task.NewInspectionProcessor(r.GetRecorderTaskName(name), append(dependenciesBase, dependencies...), func(ctx context.Context, taskMode int, v *task.VariableSet, tp *progress.TaskProgress) (any, error) {
		if taskMode == inspection_task.TaskModeDryRun {
			return struct{}{}, nil
		}
		builder, err := inspection_task.GetHistoryBuilderFromTaskVariable(v)
		if err != nil {
			return nil, err
		}
		groupedLogs, err := task.GetTypedVariableFromTaskVariable[[]*types.TimelineGrouperResult](v, constants.ManifestGenerateTaskID, nil)
		if err != nil {
			return nil, err
		}
		filteredLogs, allCount := filterMatchedGroupedLogs(ctx, groupedLogs, logGroupFilter)
		processedLogCount := atomic.Int32{}
		updator := progress.NewProgressUpdator(tp, time.Second, func(tp *progress.TaskProgress) {
			current := processedLogCount.Load()
			tp.Percentage = float32(current) / float32(allCount)
			tp.Message = fmt.Sprintf("%d/%d", current, allCount)
		})
		updator.Start(ctx)
		defer updator.Done()
		workerPool := worker.NewPool(16)
		for _, loopGroup := range filteredLogs {
			group := loopGroup
			var prevState any = nil
			workerPool.Run(func() {
				for _, l := range group.PreParsedLogs {
					if !logFilter(ctx, l) {
						processedLogCount.Add(1)
						continue
					}
					cs := history.NewChangeSet(l.Log)
					currentState, err := recorder(ctx, group.TimelineResourcePath, l, prevState, cs, builder, v)
					if err != nil {
						processedLogCount.Add(1)
						continue
					}
					prevState = currentState
					cp, err := cs.FlushToHistory(builder)
					if err != nil {
						processedLogCount.Add(1)
						continue
					}
					for _, path := range cp {
						tb := builder.GetTimelineBuilder(path)
						tb.Sort()
					}
					processedLogCount.Add(1)
				}
			})
		}
		workerPool.Wait()
		return struct{}{}, nil
	})
	r.recorderTasks = append(r.recorderTasks, newTask)
}

func (r *RecorderTaskManager) GetRecorderTaskName(recorderName string) string {
	return fmt.Sprintf("%s/recorder/%s", r.taskID.ReferenceId(), recorderName)
}

func (r *RecorderTaskManager) Register(server *inspection.InspectionTaskServer, labelOpts ...task.LabelOpt) error {
	recorderTaskIds := []string{}
	for _, recorder := range r.recorderTasks {
		err := server.AddTaskDefinition(recorder)
		if err != nil {
			return err
		}
		recorderTaskIds = append(recorderTaskIds, recorder.ID().String())
	}
	waiterTask := inspection_task.NewInspectionProcessor(r.taskID.String(), recorderTaskIds, func(ctx context.Context, taskMode int, v *task.VariableSet, progress *progress.TaskProgress) (any, error) {
		return struct{}{}, nil
	}, labelOpts...)

	err := server.AddTaskDefinition(waiterTask)
	return err
}

// filterMatchedGroupedLogs returns the filtered grouper result array and the total count of logs inside
func filterMatchedGroupedLogs(ctx context.Context, logGroups []*types.TimelineGrouperResult, matcher LogGroupFilterFunc) ([]*types.TimelineGrouperResult, int) {
	result := []*types.TimelineGrouperResult{}
	totalLogCount := 0
	for _, group := range logGroups {
		if matcher(ctx, group.TimelineResourcePath) {
			result = append(result, group)
			totalLogCount += len(group.PreParsedLogs)
		}
	}
	return result, totalLogCount
}
