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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

// LogGroupFilterFunc defines a function signature for filtering log groups based on their resource path.
type LogGroupFilterFunc = func(ctx context.Context, resourcePath string) bool

// LogFilterFunc defines a function signature for filtering individual audit logs.
type LogFilterFunc = func(ctx context.Context, l *commonlogk8saudit_contract.AuditLogParserInput) bool

// RecorderRequest holds the context and data for a recorder function to process.
type RecorderRequest struct {
	TimelineResourceStringPath string
	LogParseResult             *commonlogk8saudit_contract.AuditLogParserInput
	LastLogParseResult         *commonlogk8saudit_contract.AuditLogParserInput
	// PreviousState is any type value returned from the last recorder call. Recorder can have state gathered until its current log here.
	PreviousState           any
	ParentTimelineRevisions []*history.ResourceRevision
	ChangeSet               *history.ChangeSet
	Builder                 *history.Builder
	IsFirstLog              bool
	IsLastLog               bool
}

// RecorderFunc records events/revisions...etc on the given ChangeSet. If it returns an error, then the result is ignored.
// Recoder has responsibility to determine where the revisions or events are placed regarding the request generated for each logs.
type RecorderFunc = func(ctx context.Context, req *RecorderRequest) (any, error)

// hierarchicalTimelineGroupWorker represents a node in a hierarchical tree structure of timeline groups.
// It is used to organize and process timeline groups based on their resource paths to process them from ancestor to children.
type hierarchicalTimelineGroupWorker struct {
	children map[string]*hierarchicalTimelineGroupWorker
	group    *commonlogk8saudit_contract.TimelineGrouperResult
}

// Run traverses the hierarchical timeline group structure and executes a given function `f`
// for each `TimelineGrouperResult` found. It uses a worker pool to process groups concurrently.
func (h *hierarchicalTimelineGroupWorker) Run(ctx context.Context, f func(group *commonlogk8saudit_contract.TimelineGrouperResult)) {
	if h.group != nil {
		f(h.group)
	}
	wg := sync.WaitGroup{}
	for _, child := range h.children {
		wg.Add(1)
		go func(child *hierarchicalTimelineGroupWorker) {
			defer wg.Done()
			child.Run(ctx, f)
		}(child)
	}
	wg.Wait()
}

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

// AddRecorder adds a new recorder task to the manager.
// Each recorder task processes grouped audit logs and applies custom recording logic.
func (r *RecorderTaskManager) AddRecorder(name string, dependencies []taskid.UntypedTaskReference, recorder RecorderFunc, logGroupFilter LogGroupFilterFunc, logFilter LogFilterFunc) {
	dependenciesBase := []taskid.UntypedTaskReference{
		commonlogk8saudit_contract.LogConvertTaskID.Ref(),
		commonlogk8saudit_contract.ManifestGenerateTaskID.Ref(),
	}
	newTask := inspectiontaskbase.NewProgressReportableInspectionTask(r.GetRecorderTaskName(name), append(dependenciesBase, dependencies...), func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) (any, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return struct{}{}, nil
		}
		builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)
		groupedLogs := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ManifestGenerateTaskID.Ref())

		filteredLogs, allCount := filterMatchedGroupedLogs(ctx, groupedLogs, logGroupFilter)
		processedLogCount := atomic.Int32{}
		updator := progressutil.NewProgressUpdator(tp, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := processedLogCount.Load()
			tp.Percentage = float32(current) / float32(allCount)
			tp.Message = fmt.Sprintf("%d/%d", current, allCount)
		})
		updator.Start(ctx)
		defer updator.Done()

		hierarchicalGroupedLogs := convertTimelineGroupListToHierarcicalTimelineGroup(filteredLogs)

		hierarchicalGroupedLogs.Run(ctx, func(group *commonlogk8saudit_contract.TimelineGrouperResult) {
			var prevState any = nil

			groupPath := resourcepath.ResourcePath{
				Path:               group.TimelineResourcePath,
				ParentRelationship: enum.RelationshipChild,
			}
			tb := builder.GetTimelineBuilder(groupPath.GetParentPathString())
			parentTimelineRevisions := tb.GetRevisions()

			for i, l := range group.PreParsedLogs {
				if !logFilter(ctx, l) {
					processedLogCount.Add(1)
					continue
				}
				cs := history.NewChangeSet(l.Log)
				var prevLog *commonlogk8saudit_contract.AuditLogParserInput
				if i > 0 {
					prevLog = group.PreParsedLogs[i-1]
				}
				currentState, err := recorder(ctx, &RecorderRequest{
					TimelineResourceStringPath: group.TimelineResourcePath,
					LogParseResult:             l,
					LastLogParseResult:         prevLog,
					PreviousState:              prevState,
					ParentTimelineRevisions:    parentTimelineRevisions,
					ChangeSet:                  cs,
					Builder:                    builder,
					IsFirstLog:                 i == 0,
					IsLastLog:                  i == len(group.PreParsedLogs)-1,
				})
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

		return struct{}{}, nil
	})
	r.recorderTasks = append(r.recorderTasks, newTask)
}

func (r *RecorderTaskManager) GetRecorderTaskName(recorderName string) taskid.TaskImplementationID[any] {
	return taskid.NewDefaultImplementationID[any](fmt.Sprintf("%s/feature/k8s_audit/%s/recorder/%s", commonlogk8saudit_contract.CommonK8sAuditLogTaskIDPrefix, r.recorderPrefix, recorderName))
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
	}, inspectioncore_contract.FeatureTaskLabel("Kubernetes Audit Log", `Gather kubernetes audit logs and visualize resource modifications.`, enum.LogTypeAudit, 1000, true, inspectionTypes...), coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()))
	err := registry.AddTask(waiterTask)
	return err
}

// filterMatchedGroupedLogs returns the filtered grouper result array and the total count of logs inside
func filterMatchedGroupedLogs(ctx context.Context, logGroups []*commonlogk8saudit_contract.TimelineGrouperResult, matcher LogGroupFilterFunc) ([]*commonlogk8saudit_contract.TimelineGrouperResult, int) {
	result := []*commonlogk8saudit_contract.TimelineGrouperResult{}
	totalLogCount := 0
	for _, group := range logGroups {
		if matcher(ctx, group.TimelineResourcePath) {
			result = append(result, group)
			totalLogCount += len(group.PreParsedLogs)
		}
	}
	return result, totalLogCount
}

// convertTimelineGroupListToHierarcicalTimelineGroup converts a flat list of `TimelineGrouperResult`
// into a hierarchical tree structure represented by `hierarcicalTimelineGroupWorker`.
// logGroup must be sorted by its path because this assume parent must appear before its children.
func convertTimelineGroupListToHierarcicalTimelineGroup(logGroup []*commonlogk8saudit_contract.TimelineGrouperResult) *hierarchicalTimelineGroupWorker {
	root := &hierarchicalTimelineGroupWorker{
		children: map[string]*hierarchicalTimelineGroupWorker{},
		group:    nil,
	}
	for _, group := range logGroup {
		current := root
		segments := strings.Split(group.TimelineResourcePath, "#")
		for i, segment := range segments {
			if _, ok := current.children[segment]; !ok {
				current.children[segment] = &hierarchicalTimelineGroupWorker{
					children: map[string]*hierarchicalTimelineGroupWorker{},
					group:    nil,
				}
			}
			current = current.children[segment]
			if i == len(segments)-1 {
				current.group = group
			}
		}
	}
	return root
}
