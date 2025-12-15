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

package commonlogk8sauditv2_contract

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"golang.org/x/sync/errgroup"
)

// ChangeEventType is the type of the resource change event.
type ChangeEventType int

const (
	// ChangeEventTypeSourceCreation is the event type when the source resource is created.
	// The "creation" is not necessary means the audit log with of a request with verb "CREATE", but it is the first event of the source resource.
	// This event can happen within the same resource multiple time when the resource was completely deleted and then created again.
	ChangeEventTypeSourceCreation ChangeEventType = iota
	// ChangeEventTypeSourceDeletion is the event type when the source resource is deleted.
	// Deletion event can happen multiple times when audit logs has no information on the content and the resource had multiple deletion requests in series.
	ChangeEventTypeSourceDeletion
	// ChangeEventTypeSourceModification is the event type when the source resource is modified.
	ChangeEventTypeSourceModification
	// ChangeEventTypeTargetCreation is the event type when the target resource is created.
	// The "creation" is not necessary means the audit log with of a request with verb "CREATE", but it is the first event of the target resource.
	ChangeEventTypeTargetCreation
	// ChangeEventTypeTargetDeletion is the event type when the target resource is deleted.
	// Deletion event can happen multiple times when audit logs has no information on the content and the resource had multiple deletion requests in series.
	ChangeEventTypeTargetDeletion
	// ChangeEventTypeTargetModification is the event type when the target resource is modified.
	ChangeEventTypeTargetModification
)

// ResourcePair is the pair of the target resource and the source resource.
type ResourcePair struct {
	// TargetGroup is the resource path of the target resource.
	TargetGroup *ResourceIdentity
	// SourceGroup is the resource path of the source resource.
	SourceGroup *ResourceIdentity
}

// ResourceChangeEvent is the event of the resource change.
type ResourceChangeEvent struct {
	// EventType is the type of the event.
	EventType ChangeEventType
	// Log is the log associated with the event.
	Log *log.Log
	// EventSourceResource is the source resource of the event.
	EventSourceResource *ResourceIdentity
	// EventTargetResource is the target resource of the event.
	EventTargetResource *ResourceIdentity
	// EventSourceBodyYAML is the YAML representation of the source resource body.
	EventSourceBodyYAML string
	// EventSourceBodyReader is the reader for the source resource body.
	EventSourceBodyReader *structured.NodeReader
	// EventTargetBodyYAML is the YAML representation of the target resource body.
	EventTargetBodyYAML string
	// EventTargetBodyReader is the reader for the target resource body.
	EventTargetBodyReader *structured.NodeReader
}

// ManifestLogToTimelineMapperTaskSetting is the setting for the manifest timeline mapper task.
type ManifestLogToTimelineMapperTaskSetting[T any] interface {
	// TaskID returns the task ID.
	TaskID() taskid.TaskImplementationID[struct{}]
	// LogIngesterTask returns the task reference for the log serializer task.
	LogIngesterTask() taskid.TaskReference[[]*log.Log]
	// GroupedLogTask returns the task reference for the grouped log task.
	GroupedLogTask() taskid.TaskReference[ResourceManifestLogGroupMap]
	// Dependencies returns the dependencies of the task.
	Dependencies() []taskid.UntypedTaskReference
	// PassCount returns the number of passes.
	PassCount() int
	// ResourcePairs returns the resource pairs.
	ResourcePairs(ctx context.Context, groupedLogs ResourceManifestLogGroupMap) ([]ResourcePair, error)
	// Process processes the event.
	Process(ctx context.Context, passIndex int, event ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, prevGroupData T) (T, error)
}

// NewManifestLogToTimelineMapper creates a new timeline mapper task but from resource logs.
// ManifestLogToTimelineMapper is a task that generates a timeline of resource changes based on the processed manifests.
// It is designed to handle the relationship between two resources (Source and Target) and generate revisions for the Target resource based on the changes in the Source resource.
// For example, it can be used to generate a timeline of Pod status changes based on the Pod resource itself (Source=None, Target=Pod), or to generate a timeline of binding subresource but deleted when its parent Pod is deleted (Source=Pod, Target=Source pod's binding).
// The setting has ResourcePairs method that returns the resource pairs to know these pairs of target and source.
//
// The task works by iterating over the logs of the Source and Target resources in chronological order.
// It calls the Process method of the setting for each event, allowing the implementation to maintain state and generate revisions.
//
// Type Parameter T:
// The type parameter T represents the state that is passed between Process calls for the same resource pair.
// This allows the implementation to track the history of the resource and detect changes.
func NewManifestLogToTimelineMapper[T any](setting ManifestLogToTimelineMapperTaskSetting[T]) coretask.Task[struct{}] {
	dependencies := append([]taskid.UntypedTaskReference{setting.LogIngesterTask(), setting.GroupedLogTask()}, setting.Dependencies()...)
	return inspectiontaskbase.NewProgressReportableInspectionTask(setting.TaskID(), dependencies, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) (struct{}, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			slog.DebugContext(ctx, "Skipping task because this is dry run mode")
			return struct{}{}, nil
		}

		builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)
		groupedLogs := coretask.GetTaskResult(ctx, setting.GroupedLogTask())

		tp.MarkIndeterminate()
		trackingGroups, err := setting.ResourcePairs(ctx, groupedLogs)
		if err != nil {
			return struct{}{}, err
		}
		doneGroupCount := atomic.Int32{}

		updator := progressutil.NewProgressUpdator(tp, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := doneGroupCount.Load()
			tp.Percentage = float32(current) / float32(len(trackingGroups))
			tp.Message = fmt.Sprintf("%d/%d", current, len(trackingGroups))
		})
		updator.Start(ctx)

		errGrp, childCtx := errgroup.WithContext(ctx)
		errGrp.SetLimit(runtime.GOMAXPROCS(0))
		passCount := setting.PassCount()
		for _, trackingGroup := range trackingGroups {
			trackingGroup := trackingGroup
			errGrp.Go(func() error {
				defer doneGroupCount.Add(1)
				changedPaths := map[string]struct{}{}
				var sourceLogs *ResourceManifestLogGroup
				var targetLogs *ResourceManifestLogGroup
				if trackingGroup.SourceGroup != nil {
					sourceLogs = groupedLogs[trackingGroup.SourceGroup.ResourcePathString()]
				}
				if trackingGroup.TargetGroup != nil {
					targetLogs = groupedLogs[trackingGroup.TargetGroup.ResourcePathString()]
				}
				if sourceLogs == nil {
					sourceLogs = &ResourceManifestLogGroup{}
				}
				if targetLogs == nil {
					targetLogs = &ResourceManifestLogGroup{}
				}
				var prevData T
				for pass := 0; pass < passCount; pass++ {
					for event := range iterateLogGroupPair(sourceLogs, targetLogs) {
						cs := history.NewChangeSet(event.Log)
						prevData, err = setting.Process(childCtx, pass, event, cs, builder, prevData)
						if err != nil {
							return err
						}
						cp, err := cs.FlushToHistory(builder)
						if err != nil {
							slog.WarnContext(ctx, "failed to flush the changeset to history", "error", err)
						}
						for _, path := range cp {
							changedPaths[path] = struct{}{}
						}
					}
				}
				for path := range changedPaths {
					tb := builder.GetTimelineBuilder(path)
					tb.Sort()
				}
				return nil
			})
		}
		if err := errGrp.Wait(); err != nil {
			return struct{}{}, err
		}

		return struct{}{}, nil
	})
}

func iterateLogGroupPair(sourceLogs *ResourceManifestLogGroup, targetLogs *ResourceManifestLogGroup) func(func(ResourceChangeEvent) bool) {
	sResource := sourceLogs.Resource
	tResource := targetLogs.Resource
	return func(fn func(ResourceChangeEvent) bool) {
		slogIndex := 0
		tlogIndex := 0
		var pickFromSource bool
		for slogIndex < len(sourceLogs.Logs) || tlogIndex < len(targetLogs.Logs) {
			switch {
			case tlogIndex >= len(targetLogs.Logs):
				pickFromSource = true
			case slogIndex >= len(sourceLogs.Logs):
				pickFromSource = false
			default:
				sTime := log.MustGetFieldSet(sourceLogs.Logs[slogIndex].Log, &log.CommonFieldSet{})
				tTime := log.MustGetFieldSet(targetLogs.Logs[tlogIndex].Log, &log.CommonFieldSet{})
				if sTime.Timestamp.Before(tTime.Timestamp) {
					pickFromSource = true
				} else {
					pickFromSource = false
				}
			}
			var next ResourceChangeEvent
			if pickFromSource {
				sLog := sourceLogs.Logs[slogIndex]
				var tLogReader *structured.NodeReader
				var tLogBodyYAML string
				if tlogIndex-1 >= 0 {
					tLog := targetLogs.Logs[tlogIndex-1]
					tLogReader = tLog.ResourceBodyReader
					tLogBodyYAML = tLog.ResourceBodyYAML
				}
				eType := ChangeEventTypeSourceModification
				if sLog.ResourceCreated {
					eType = ChangeEventTypeSourceCreation
				}
				if sLog.ResourceDeleted {
					eType = ChangeEventTypeSourceDeletion
				}
				next = ResourceChangeEvent{
					EventType:             eType,
					Log:                   sLog.Log,
					EventSourceResource:   sResource,
					EventTargetResource:   tResource,
					EventSourceBodyYAML:   sLog.ResourceBodyYAML,
					EventSourceBodyReader: sLog.ResourceBodyReader,
					EventTargetBodyYAML:   tLogBodyYAML,
					EventTargetBodyReader: tLogReader,
				}
				slogIndex++
			} else {
				tLog := targetLogs.Logs[tlogIndex]
				var sLogReader *structured.NodeReader
				var sLogBodyYAML string
				if slogIndex-1 >= 0 {
					sLog := sourceLogs.Logs[slogIndex-1]
					sLogReader = sLog.ResourceBodyReader
					sLogBodyYAML = sLog.ResourceBodyYAML
				}
				eType := ChangeEventTypeTargetModification
				if tLog.ResourceCreated {
					eType = ChangeEventTypeTargetCreation
				}
				if tLog.ResourceDeleted {
					eType = ChangeEventTypeTargetDeletion
				}
				next = ResourceChangeEvent{
					EventType:             eType,
					Log:                   tLog.Log,
					EventSourceResource:   sResource,
					EventTargetResource:   tResource,
					EventSourceBodyYAML:   sLogBodyYAML,
					EventSourceBodyReader: sLogReader,
					EventTargetBodyYAML:   tLog.ResourceBodyYAML,
					EventTargetBodyReader: tLog.ResourceBodyReader,
				}
				tlogIndex++
			}
			if !fn(next) {
				return
			}
		}
	}
}
