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

package inspectiontaskbase

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TimelineMapperResult holds the summary of writes performed by a timeline mapper.
type TimelineMapperResult struct {
	// Events maps timeline paths to the number of events written.
	Events map[*khifilev6.TimelinePath]int
	// Revisions maps timeline paths to the number of revisions written.
	Revisions map[*khifilev6.TimelinePath]int
	// Aliases maps alias paths to their target paths.
	Aliases map[*khifilev6.TimelinePath]*khifilev6.TimelinePath
}

// NewTimelineMapperResult creates an initialized TimelineMapperResult.
func NewTimelineMapperResult() TimelineMapperResult {
	return TimelineMapperResult{
		Events:    make(map[*khifilev6.TimelinePath]int),
		Revisions: make(map[*khifilev6.TimelinePath]int),
		Aliases:   make(map[*khifilev6.TimelinePath]*khifilev6.TimelinePath),
	}
}

// Merge merges another TimelineMapperResult into this one.
func (r *TimelineMapperResult) Merge(other TimelineMapperResult) {
	for p, count := range other.Events {
		r.Events[p] += count
	}
	for p, count := range other.Revisions {
		r.Revisions[p] += count
	}
	for alias, target := range other.Aliases {
		r.Aliases[alias] = target
	}
}

// LogToTimelineMapper defines the interface for mapping logs to timeline elements (events or revisions) in KHI file v6 format.
type LogToTimelineMapper[T any] interface {
	// LogIngesterTask is one of prerequisite task of LogToTimelineMapper ingesting logs before processing with this mapper.
	LogIngesterTask() taskid.TaskReference[[]*log.Log]
	// Dependencies are the additional references used in timeline mapper.
	Dependencies() []taskid.UntypedTaskReference
	// GroupedLogTask returns a reference to the task that provides the grouped logs.
	GroupedLogTask() taskid.TaskReference[LogGroupMap]
	// PassCount returns the number of pre-processing passes to perform on each group.
	PassCount() int
	// PreProcessLogByGroup is called during a pre-processing pass for each log in a group.
	// The passIndex is 0-indexed and ranges from 0 to PassCount()-1.
	PreProcessLogByGroup(ctx context.Context, passIndex int, l *log.Log, prevGroupData T) (T, error)
	// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
	// The prevGroupData is the returned value from the last processed log in the same group.
	ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData T) (*khifilev6.TimelineChangeSet, T, error)
}

// SinglePassMapperBase provides a base implementation of LogToTimelineMapper
// for mappers that only require a single pass over the logs.
type SinglePassMapperBase[T any] struct{}

// PassCount returns 0 as no pre-processing pass is required.
func (SinglePassMapperBase[T]) PassCount() int {
	return 0
}

// PreProcessLogByGroup is a no-op pre-processor that returns the state as-is.
func (SinglePassMapperBase[T]) PreProcessLogByGroup(ctx context.Context, passIndex int, l *log.Log, prevGroupData T) (T, error) {
	return prevGroupData, nil
}

// StatelessMapperBase provides a base implementation of LogToTimelineMapper
// for mappers that are both stateless and only require a single pass.
type StatelessMapperBase struct{}

// PassCount returns 0 as no pre-processing pass is required.
func (StatelessMapperBase) PassCount() int {
	return 0
}

// PreProcessLogByGroup is a no-op pre-processor that returns an empty struct.
func (StatelessMapperBase) PreProcessLogByGroup(ctx context.Context, passIndex int, l *log.Log, prevGroupData struct{}) (struct{}, error) {
	return struct{}{}, nil
}

// NewLogToTimelineMapperTask creates a task that modifies the KHI v6 TimelineRegistry based on grouped logs.
// It processes logs in parallel and applies the logic from the provided LogToTimelineMapper.
func NewLogToTimelineMapperTask[T any](tid taskid.TaskImplementationID[TimelineMapperResult], mapper LogToTimelineMapper[T], labels ...coretask.LabelOpt) coretask.Task[TimelineMapperResult] {
	groupedLogTaskID := mapper.GroupedLogTask()
	dependencies := append([]taskid.UntypedTaskReference{mapper.LogIngesterTask(), mapper.GroupedLogTask()}, mapper.Dependencies()...)
	return NewProgressReportableInspectionTask(tid, dependencies, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) (TimelineMapperResult, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			slog.DebugContext(ctx, "Skipping task because this is dry run mode")
			return NewTimelineMapperResult(), nil
		}

		builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
		groupedLogs := coretask.GetTaskResult(ctx, groupedLogTaskID)

		totalLogCount := 0
		var processedLogCount atomic.Uint32
		var skippedLogCount atomic.Uint32
		for _, group := range groupedLogs {
			totalLogCount += len(group.Logs)
		}

		passCount := mapper.PassCount()
		totalSteps := totalLogCount * (passCount + 1)

		updator := progressutil.NewProgressUpdator(tp, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := processedLogCount.Load()
			if totalSteps > 0 {
				tp.Percentage = float32(current) / float32(totalSteps)
			}
			tp.Message = fmt.Sprintf("%d/%d", current, totalSteps)
		})
		updator.Start(ctx)

		processedLogCount.Store(0)

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
			if ctx.Err() != nil {
				return true
			}
			errMu.Lock()
			defer errMu.Unlock()
			return sharedErr != nil
		}

		var resultMu sync.Mutex
		finalResult := NewTimelineMapperResult()

		pool := worker.NewPool(runtime.GOMAXPROCS(0))
		for _, group := range groupedLogs {
			if ctx.Err() != nil {
				break
			}
			pool.Run(func() {
				if hasErr() {
					return
				}
				var groupData T

				// 1. Pre-processing passes
				passCount := mapper.PassCount()
				for passIdx := 0; passIdx < passCount; passIdx++ {
					for _, l := range group.Logs {
						if hasErr() {
							return
						}
						nextGroupData, err := mapper.PreProcessLogByGroup(ctx, passIdx, l, groupData)
						processedLogCount.Add(1)
						if err != nil {
							logTaskError(ctx, fmt.Sprintf("pre-processor ended with an error at passIndex %d", passIdx), err, l)
							setErr(err)
							return
						}
						groupData = nextGroupData
					}
				}

				localResult := NewTimelineMapperResult()

				// 2. Final processing pass
				for _, l := range group.Logs {
					if hasErr() {
						return
					}
					cs, nextGroupData, err := mapper.ProcessLogByGroup(ctx, l, groupData)
					processedLogCount.Add(1)
					if err != nil {
						logTaskError(ctx, "parser ended with an error", err, l)
						setErr(err)
						return
					}
					groupData = nextGroupData

					if cs != nil {
						err := cs.Flush(builder.TimelineAccumulator)
						if err != nil {
							logTaskError(ctx, "failed to flush the changeset to timeline registry", err, l)
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
					} else {
						skippedLogCount.Add(1)
					}
				}

				resultMu.Lock()
				finalResult.Merge(localResult)
				resultMu.Unlock()
			})
		}
		pool.Wait()
		updator.Done()

		if ctx.Err() != nil {
			return NewTimelineMapperResult(), ctx.Err()
		}
		if sharedErr != nil {
			return NewTimelineMapperResult(), sharedErr
		}

		slog.DebugContext(ctx, fmt.Sprintf("LogToTimelineMapperTaskV2 %s finished: processed %d logs (skipped %d logs)", tid.String(), totalLogCount, skippedLogCount.Load()))

		tracingActive, _ := khictx.GetValue(ctx, inspectioncore_contract.TracingActive)
		if tracingActive {
			trace.SpanFromContext(ctx).SetAttributes(
				attribute.String("log_count", fmt.Sprintf("%d", totalLogCount)),
			)
		}

		return finalResult, nil
	}, append([]coretask.LabelOpt{
		// Tasks modifying history must be dependent from SerializerTask.
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref())}, labels...)...)
}
