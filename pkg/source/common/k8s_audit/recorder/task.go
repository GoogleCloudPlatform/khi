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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	common_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

type LogGroupFilterFunc = func(ctx context.Context, resourcePath string) bool

type LogFilterFunc = func(ctx context.Context, l *types.AuditLogParserInput) bool

// RecorderFunc records events/revisions...etc on the given ChangeSet. If it returns an error, then the result is ignored.
type RecorderFunc = func(ctx context.Context, resourcePath string, currentLog *types.AuditLogParserInput, prevStateInGroup any, cs *history.ChangeSet, builder *history.Builder) (any, error)

// RecorderTaskManager provides the way of extending resource specific
type RecorderTaskManager struct {
	taskID         taskid.TaskImplementationID[struct{}]
	recorderTasks  []coretask.UntypedTask
	recorderPrefix string
}

func NewAuditRecorderTaskManager(taskID taskid.TaskImplementationID[struct{}], recorderPrefix string) *RecorderTaskManager {
	return &RecorderTaskManager{
		taskID:         taskID,
		recorderTasks:  make([]coretask.UntypedTask, 0),
		recorderPrefix: recorderPrefix,
	}
}

func (r *RecorderTaskManager) AddRecorder(name string, dependencies []taskid.UntypedTaskReference, recorder RecorderFunc, logGroupFilter LogGroupFilterFunc, logFilter LogFilterFunc) {
	dependenciesBase := []taskid.UntypedTaskReference{
		common_k8saudit_taskid.LogConvertTaskID.Ref(),
		common_k8saudit_taskid.ManifestGenerateTaskID.Ref(),
	}
	newTask := inspectiontaskbase.NewProgressReportableInspectionTask(r.GetRecorderTaskName(name), append(dependenciesBase, dependencies...), func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) (any, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return struct{}{}, nil
		}
		builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)
		groupedLogs := coretask.GetTaskResult(ctx, common_k8saudit_taskid.ManifestGenerateTaskID.Ref())

		filteredLogs, allCount := filterMatchedGroupedLogs(ctx, groupedLogs, logGroupFilter)
		processedLogCount := atomic.Int32{}
		updator := progressutil.NewProgressUpdator(tp, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
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
					currentState, err := recorder(ctx, group.TimelineResourcePath, l, prevState, cs, builder)
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

func (r *RecorderTaskManager) GetRecorderTaskName(recorderName string) taskid.TaskImplementationID[any] {
	return taskid.NewDefaultImplementationID[any](fmt.Sprintf("%s/feature/k8s_audit/%s/recorder/%s", gcp_task.GCPPrefix, r.recorderPrefix, recorderName))
}

func (r *RecorderTaskManager) Register(registry coretask.TaskRegistry, inspectionTypes ...string) error {
	recorderTaskIds := []taskid.UntypedTaskReference{}
	for _, recorder := range r.recorderTasks {
		err := registry.AddTask(recorder)
		if err != nil {
			return err
		}
		recorderTaskIds = append(recorderTaskIds, recorder.UntypedID().GetUntypedReference())
	}
	waiterTask := inspectiontaskbase.NewInspectionTask(r.taskID, recorderTaskIds, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	}, inspectioncore_contract.FeatureTaskLabel("Kubernetes Audit Log", `Gather kubernetes audit logs and visualize resource modifications.`, enum.LogTypeAudit, true, inspectionTypes...))
	err := registry.AddTask(waiterTask)
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
