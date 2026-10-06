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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TimelineMapper maps the logs of each log group to timeline elements (events or revisions) in KHI file v6 format.
// T is the state passed from one log to the next log in the same group.
type TimelineMapper[T any] interface {
	// PassCount returns the number of pre-processing passes to perform on each group.
	PassCount() int
	// PreProcessLogByGroup is called during a pre-processing pass for each log in a group.
	// The passIndex is 0-indexed and ranges from 0 to PassCount()-1.
	PreProcessLogByGroup(ctx context.Context, passIndex int, l *log.Log, prevGroupData T) (T, error)
	// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
	// The prevGroupData is the returned value from the last processed log in the same group.
	ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData T) (*khifilev6.TimelineChangeSet, T, error)
}

// TimelineMapperInputs specifies the log ingester the mapper waits for and the grouped log task it reads.
// DefineLogToTimelineMapperTask registers both on the Binder before calling bind.
type TimelineMapperInputs struct {
	// LogIngester is the task that must ingest the log metadata before the mapper runs. The mapper does not read its value.
	LogIngester taskid.TaskReference[struct{}]
	// GroupedLogs is the task that provides the logs to map, grouped by the unit processed sequentially.
	GroupedLogs taskid.TaskReference[LogGroupMap]
}

// SinglePassMapperBase provides a base implementation of TimelineMapper
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

// StatelessMapperBase provides a base implementation of TimelineMapper
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

// DefineLogToTimelineMapperTask creates a task that modifies the KHI v6 TimelineRegistry based on the logs grouped by inputs.GroupedLogs.
//
// inputs specifies the log ingester the mapper waits for and the grouped log task it reads; DefineLogToTimelineMapperTask registers both on the Binder before calling bind.
// bind is called once at task definition time to declare additional inputs and return the TimelineMapper[T] instance.
// Because the returned TimelineMapper[T] is shared across inspections and concurrent worker goroutines, any per-group mutable state must be stored in T rather than on the mapper struct.
func DefineLogToTimelineMapperTask[T any](tid taskid.TaskImplementationID[struct{}], inputs TimelineMapperInputs, bind func(b *coretask.Binder) TimelineMapper[T], labelOpts ...coretask.LabelOpt) coretask.Task[struct{}] {
	allLabels := append([]coretask.LabelOpt{
		coretask.ProvidesTag(TagTimelineMapper),
	}, labelOpts...)
	return DefineInspectionTask(tid, func(b *coretask.Binder) InspectionTaskFunc[struct{}] {
		coretask.After(b, inputs.LogIngester)
		groupedLogs := coretask.Use(b, inputs.GroupedLogs)
		mapper := bind(b)
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (struct{}, error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				slog.DebugContext(ctx, "Skipping task because this is dry run mode")
				return struct{}{}, nil
			}
			return struct{}{}, mapGroupedLogs(ctx, tid, groupedLogs.Get(ctx), mapper)
		}
	}, allLabels...)
}

// mapGroupedLogs processes each log group in parallel with mapper and flushes the resulting change sets to the builder in the context.
// Logs in the same group are processed sequentially so that mapper can pass state from one log to the next.
func mapGroupedLogs[T any](ctx context.Context, tid taskid.TaskImplementationID[struct{}], groupedLogs LogGroupMap, mapper TimelineMapper[T]) error {
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	totalLogCount := 0
	var skippedLogCount atomic.Uint32
	for _, group := range groupedLogs {
		totalLogCount += len(group.Logs)
	}

	passCount := mapper.PassCount()
	totalSteps := totalLogCount * (passCount + 1)

	tracker := progress.NewTracker(ctx, totalSteps, progress.WithUnit("steps"))
	defer tracker.Done()

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
					tracker.Inc()
					if err != nil {
						logTaskError(ctx, fmt.Sprintf("pre-processor ended with an error at passIndex %d", passIdx), err, l)
						setErr(err)
						return
					}
					groupData = nextGroupData
				}
			}

			// 2. Final processing pass
			for _, l := range group.Logs {
				if hasErr() {
					return
				}
				cs, nextGroupData, err := mapper.ProcessLogByGroup(ctx, l, groupData)
				tracker.Inc()
				if err != nil {
					logTaskError(ctx, "parser ended with an error", err, l)
					setErr(err)
					return
				}
				groupData = nextGroupData

				if cs != nil {
					err := cs.Flush(builder.TimelineAccumulator, builder.LogAccumulator)
					cs.Release()
					if err != nil {
						logTaskError(ctx, "failed to flush the changeset to timeline registry", err, l)
						setErr(err)
						return
					}
				} else {
					skippedLogCount.Add(1)
				}
			}
		})
	}
	pool.Wait()

	if ctx.Err() != nil {
		return ctx.Err()
	}
	if sharedErr != nil {
		return sharedErr
	}

	slog.DebugContext(ctx, fmt.Sprintf("LogToTimelineMapperTask %s finished: processed %d logs (skipped %d logs)", tid.String(), totalLogCount, skippedLogCount.Load()))

	tracingActive, _ := khictx.GetValue(ctx, inspectioncore.TracingActive)
	if tracingActive {
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("log_count", fmt.Sprintf("%d", totalLogCount)),
		)
	}

	return nil
}
