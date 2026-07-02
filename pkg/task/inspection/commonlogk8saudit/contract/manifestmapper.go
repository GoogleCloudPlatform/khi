// Copyright 2026 Google LLC
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

package commonlogk8saudit_contract

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// ChangeEventTypeV2 is the type of the resource change event for V2 mappers.
type ChangeEventTypeV2 int

const (
	// ChangeEventTypeV2Creation is the event type when the resource is created or first observed.
	ChangeEventTypeV2Creation ChangeEventTypeV2 = iota
	// ChangeEventTypeV2Deletion is the event type when the resource is deleted.
	ChangeEventTypeV2Deletion
	// ChangeEventTypeV2Modification is the event type when the resource is modified.
	ChangeEventTypeV2Modification
)

// RelatedGroupSet is a set of log groups that should be processed together.
// For example, it groups a Pod log group and its Subresource (like status or binding) log groups.
type RelatedGroupSet struct {
	// Roles maps role names (e.g., "pod", "binding") to their corresponding log groups.
	Roles map[string]*ResourceManifestLogGroup
}

// MultiGroupLogEvent represents a single merged log event with its associated context.
type MultiGroupLogEvent struct {
	// Log is the log entry associated with the event.
	Log *log.Log
	// GroupRole is the role name of the group this log belongs to (e.g., "pod", "binding").
	GroupRole string
	// ResourceIdentity is the identity of the resource this log belongs to.
	ResourceIdentity *ResourceIdentity
	// EventType is the type of the change event.
	EventType ChangeEventTypeV2
	// GroupSet is the related group set this event belongs to.
	GroupSet RelatedGroupSet
}

// GetLastBodyReader returns the latest resource body reader for the specified role at the time of this event.
func (e *MultiGroupLogEvent) GetLastBodyReader(role string) (*structured.NodeReader, bool) {
	manifestLog, ok := e.getLastManifestLog(role)
	if !ok || manifestLog.ResourceBodyReader == nil {
		return nil, false
	}
	return manifestLog.ResourceBodyReader, true
}

// GetLastBodyYAML returns the latest resource body YAML for the specified role at the time of this event.
func (e *MultiGroupLogEvent) GetLastBodyYAML(role string) (string, bool) {
	manifestLog, ok := e.getLastManifestLog(role)
	if !ok || manifestLog.ResourceBodyYAML == "" {
		return "", false
	}
	return manifestLog.ResourceBodyYAML, true
}

// getLastManifestLog returns the latest ResourceManifestLog entry for the specified role at the time of this event.
func (e *MultiGroupLogEvent) getLastManifestLog(role string) (*ResourceManifestLog, bool) {
	group, found := e.GroupSet.Roles[role]
	if !found || group == nil {
		return nil, false
	}

	commonSet := log.MustGetFieldSet(e.Log, &log.CommonFieldSet{})
	eventTime := commonSet.Timestamp

	// Use binary search to quickly find the first log that is in the future relative to eventTime.
	idx := sort.Search(len(group.Logs), func(i int) bool {
		entryCommon := log.MustGetFieldSet(group.Logs[i].Log, &log.CommonFieldSet{})
		return entryCommon.Timestamp.After(eventTime)
	})

	// Scan backwards from the match to find the latest log entry with a valid resource body.
	var lastManifestLog *ResourceManifestLog
	for i := idx - 1; i >= 0; i-- {
		logEntry := group.Logs[i]
		if logEntry.ResourceBodyReader != nil || logEntry.ResourceBodyYAML != "" {
			lastManifestLog = logEntry
			break
		}
	}

	if lastManifestLog != nil {
		return lastManifestLog, true
	}
	return nil, false
}

// ManifestLogToTimelineMapper defines the interface for V2 manifest timeline mappers.
type ManifestLogToTimelineMapper[T any] interface {
	// TaskID returns the task ID.
	TaskID() taskid.TaskImplementationID[inspectiontaskbase.TimelineMapperResult]
	// LogIngesterTask returns the task reference for the log ingester task.
	LogIngesterTask() taskid.TaskReference[[]*log.Log]
	// GroupedLogTask returns the task reference for the grouped log task.
	GroupedLogTask() taskid.TaskReference[ResourceManifestLogGroupMap]
	// Dependencies returns additional task dependencies of the mapper.
	Dependencies() []taskid.UntypedTaskReference
	// PassCount returns the number of pre-processing passes.
	PassCount() int
	// ResolveRelatedGroupSets resolves log groups into related group sets to be processed together.
	ResolveRelatedGroupSets(ctx context.Context, groupedLogs ResourceManifestLogGroupMap) ([]RelatedGroupSet, error)
	// PreProcessLog is called during pre-processing passes for each log event.
	PreProcessLog(ctx context.Context, passIndex int, event MultiGroupLogEvent, prevGroupData T) (T, error)
	// ProcessLog is called for each log event to stage timeline changes.
	ProcessLog(ctx context.Context, event MultiGroupLogEvent, prevGroupData T) (*khifilev6.TimelineChangeSet, T, error)
}

// ManifestSinglePassMapperBase provides a base implementation for V2 mappers that require only a single pass.
type ManifestSinglePassMapperBase[T any] struct{}

// PassCount returns 0 as no pre-processing pass is required.
func (ManifestSinglePassMapperBase[T]) PassCount() int {
	return 0
}

// PreProcessLog is a no-op pre-processor that returns the state as-is.
func (ManifestSinglePassMapperBase[T]) PreProcessLog(ctx context.Context, passIndex int, event MultiGroupLogEvent, prevGroupData T) (T, error) {
	return prevGroupData, nil
}

// ManifestStatelessMapperBase provides a base implementation for V2 mappers that are both stateless and require a single pass.
type ManifestStatelessMapperBase struct{}

// PassCount returns 0 as no pre-processing pass is required.
func (ManifestStatelessMapperBase) PassCount() int {
	return 0
}

// PreProcessLog is a no-op pre-processor that returns an empty struct.
func (ManifestStatelessMapperBase) PreProcessLog(ctx context.Context, passIndex int, event MultiGroupLogEvent, prevGroupData struct{}) (struct{}, error) {
	return struct{}{}, nil
}

// NewManifestLogToTimelineMapper creates a new timeline mapper task utilizing the V2 mapper interface.
func NewManifestLogToTimelineMapper[T any](setting ManifestLogToTimelineMapper[T]) coretask.Task[inspectiontaskbase.TimelineMapperResult] {
	groupedLogTaskID := setting.GroupedLogTask()
	dependencies := append([]taskid.UntypedTaskReference{setting.LogIngesterTask(), setting.GroupedLogTask()}, setting.Dependencies()...)

	return inspectiontaskbase.NewProgressReportableInspectionTask(setting.TaskID(), dependencies, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) (inspectiontaskbase.TimelineMapperResult, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			slog.DebugContext(ctx, "Skipping task because this is dry run mode")
			return inspectiontaskbase.NewTimelineMapperResult(), nil
		}

		builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
		groupedLogs := coretask.GetTaskResult(ctx, groupedLogTaskID)

		tp.MarkIndeterminate()
		relatedGroupSets, err := setting.ResolveRelatedGroupSets(ctx, groupedLogs)
		if err != nil {
			return inspectiontaskbase.NewTimelineMapperResult(), err
		}

		var processedGroupCount atomic.Int32
		totalGroups := len(relatedGroupSets)

		updator := progressutil.NewProgressUpdator(tp, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := processedGroupCount.Load()
			if totalGroups > 0 {
				tp.Percentage = float32(current) / float32(totalGroups)
			}
			tp.Message = fmt.Sprintf("%d/%d", current, totalGroups)
		})
		updator.Start(ctx)

		var sharedErr error
		var errMu sync.Mutex

		setErr := func(err error) {
			errMu.Lock()
			defer errMu.Unlock()
			if sharedErr == nil {
				sharedErr = err
			}
		}

		hasErr := func() bool {
			errMu.Lock()
			defer errMu.Unlock()
			return sharedErr != nil
		}

		var resultMu sync.Mutex
		finalResult := inspectiontaskbase.NewTimelineMapperResult()

		pool := worker.NewPool(runtime.GOMAXPROCS(0))
		passCount := setting.PassCount()

		for _, groupSet := range relatedGroupSets {
			pool.Run(func() {
				defer processedGroupCount.Add(1)
				if hasErr() {
					return
				}

				var state T

				// 1. Pre-processing passes
				for passIdx := 0; passIdx < passCount; passIdx++ {
					for event := range iterateMultiGroupLog(groupSet) {
						if hasErr() {
							return
						}
						nextState, err := setting.PreProcessLog(ctx, passIdx, event, state)
						if err != nil {
							setErr(err)
							return
						}
						state = nextState
					}
				}

				localResult := inspectiontaskbase.NewTimelineMapperResult()

				// 2. Final processing pass
				for event := range iterateMultiGroupLog(groupSet) {
					if hasErr() {
						return
					}
					cs, nextState, err := setting.ProcessLog(ctx, event, state)
					if err != nil {
						setErr(err)
						return
					}
					state = nextState

					if cs != nil {
						err := cs.Flush(builder.TimelineAccumulator)
						if err != nil {
							setErr(err)
							return
						}
						for p := range cs.Events {
							localResult.Events[p]++
						}
						for p, revs := range cs.Revisions {
							localResult.Revisions[p] += len(revs)
						}
						for alias, target := range cs.Aliases {
							localResult.Aliases[alias] = target
						}
					}
				}

				resultMu.Lock()
				finalResult.Merge(localResult)
				resultMu.Unlock()
			})
		}

		pool.Wait()
		updator.Done()

		if sharedErr != nil {
			return inspectiontaskbase.NewTimelineMapperResult(), sharedErr
		}

		return finalResult, nil
	})
}

// iterateMultiGroupLog merges and yields log events from all roles in chronological order.
func iterateMultiGroupLog(groupSet RelatedGroupSet) func(func(MultiGroupLogEvent) bool) {
	return func(fn func(MultiGroupLogEvent) bool) {
		// Track the current scanning cursor index for each active role.
		// Each role's logs slice is already sorted chronologically.
		indices := make(map[string]int)
		roles := make([]string, 0, len(groupSet.Roles))
		for role, group := range groupSet.Roles {
			if group != nil && len(group.Logs) > 0 {
				indices[role] = 0
				roles = append(roles, role)
			}
		}

		for {
			var nextRole string
			var nextTimestamp time.Time
			hasAny := false

			// Find the oldest log across all active roles at the current cursor positions.
			// This performs a multi-way merge sort to yield logs in global chronological order.
			for _, role := range roles {
				idx := indices[role]
				group := groupSet.Roles[role]
				if idx >= len(group.Logs) {
					continue
				}

				logEntry := group.Logs[idx]
				commonSet, err := log.GetFieldSet(logEntry.Log, &log.CommonFieldSet{})
				if err != nil {
					continue
				}

				if !hasAny || commonSet.Timestamp.Before(nextTimestamp) {
					nextRole = role
					nextTimestamp = commonSet.Timestamp
					hasAny = true
				}
			}

			if !hasAny {
				break // All logs across all groups have been successfully consumed.
			}

			idx := indices[nextRole]
			targetGroup := groupSet.Roles[nextRole]
			logEntry := targetGroup.Logs[idx]

			// Map raw manifest-level lifetime flags to higher-level change event types.
			eType := ChangeEventTypeV2Modification
			if logEntry.ResourceCreated {
				eType = ChangeEventTypeV2Creation
			} else if logEntry.ResourceDeleted {
				eType = ChangeEventTypeV2Deletion
			}

			event := MultiGroupLogEvent{
				Log:              logEntry.Log,
				GroupRole:        nextRole,
				ResourceIdentity: targetGroup.Resource,
				EventType:        eType,
				GroupSet:         groupSet,
			}

			// Advance the cursor for the role we just consumed.
			indices[nextRole]++

			// Yield the merged event to the iteration callback.
			if !fn(event) {
				return
			}
		}
	}
}
