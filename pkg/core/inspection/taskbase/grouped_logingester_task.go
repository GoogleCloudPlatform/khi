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

// GroupedLogIngester defines the interface for ingesting log metadata into KHI v6 format using group-sequential processing.
type GroupedLogIngester[T any] interface {
	// PassCount returns the number of pre-processing passes to perform on each group.
	PassCount() int
	// PreProcessLogByGroup is called during a pre-processing pass for each log in a group.
	// The passIndex is 0-indexed and ranges from 0 to PassCount()-1.
	PreProcessLogByGroup(ctx context.Context, passIndex int, l *log.Log, prevGroupData T) (T, error)
	// ProcessLogByGroup is called for each log entry in a group to customize log metadata.
	// The prevGroupData is the returned value from the last processed log in the same group.
	ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData T) (*khifilev6.LogChangeSet, T, error)
}

// SinglePassGroupedIngesterBase provides a base implementation of GroupedLogIngester
// for ingesters that only require a single pass over the logs.
type SinglePassGroupedIngesterBase[T any] struct{}

// PassCount returns 0 as no pre-processing pass is required.
func (SinglePassGroupedIngesterBase[T]) PassCount() int {
	return 0
}

// PreProcessLogByGroup is a no-op pre-processor that returns the state as-is.
func (SinglePassGroupedIngesterBase[T]) PreProcessLogByGroup(ctx context.Context, passIndex int, l *log.Log, prevGroupData T) (T, error) {
	return prevGroupData, nil
}

// DefineGroupedLogIngesterTask returns a task that ingests metadata of the logs grouped by groupedLogTask into the KHI v6 builder using group-sequential processing.
//
// The task registers groupedLogTask on the Binder before calling bind.
// bind is called once at task definition time to declare additional inputs and return the GroupedLogIngester[T] instance.
// Because the returned GroupedLogIngester[T] is shared across inspections and concurrent worker goroutines, any per-group mutable state must be stored in T rather than on the ingester struct.
func DefineGroupedLogIngesterTask[T any](taskID taskid.TaskImplementationID[struct{}], groupedLogTask taskid.TaskReference[LogGroupMap], bind func(b *coretask.Binder) GroupedLogIngester[T], labelOpts ...coretask.LabelOpt) coretask.Task[struct{}] {
	allLabels := append([]coretask.LabelOpt{
		coretask.ProvidesTag(TagLogIngester),
	}, labelOpts...)
	return DefineInspectionTask(taskID, func(b *coretask.Binder) InspectionTaskFunc[struct{}] {
		groupedLogs := coretask.Use(b, groupedLogTask)
		ingester := bind(b)
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (struct{}, error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return struct{}{}, nil
			}
			return struct{}{}, ingestGroupedLogs(ctx, taskID, groupedLogs.Get(ctx), ingester)
		}
	}, allLabels...)
}

// ingestGroupedLogs processes each log group in parallel with ingester and flushes the resulting change sets to the builder in the context.
// Logs in the same group are processed sequentially so that ingester can pass state from one log to the next.
func ingestGroupedLogs[T any](ctx context.Context, taskID taskid.TaskImplementationID[struct{}], groupedLogs LogGroupMap, ingester GroupedLogIngester[T]) error {
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	totalLogCount := 0
	var skippedLogCount atomic.Uint32
	for _, group := range groupedLogs {
		totalLogCount += len(group.Logs)
	}

	passCount := ingester.PassCount()
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
			passCount := ingester.PassCount()
			for passIdx := 0; passIdx < passCount; passIdx++ {
				for _, l := range group.Logs {
					if hasErr() {
						return
					}
					nextGroupData, err := ingester.PreProcessLogByGroup(ctx, passIdx, l, groupData)
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
				cs, nextGroupData, err := ingester.ProcessLogByGroup(ctx, l, groupData)
				tracker.Inc()
				if err != nil {
					logTaskError(ctx, "parser ended with an error", err, l)
					setErr(err)
					return
				}
				groupData = nextGroupData

				if cs != nil {
					err = cs.Flush(builder.LogAccumulator)
					if err != nil {
						logTaskError(ctx, "failed to flush log changeset in ingester", err, l)
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

	slog.DebugContext(ctx, fmt.Sprintf("GroupedLogIngesterTask %s finished: processed %d logs (skipped %d logs)", taskID.String(), totalLogCount, skippedLogCount.Load()))

	tracingActive, _ := khictx.GetValue(ctx, inspectioncore.TracingActive)
	if tracingActive {
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("log_count", fmt.Sprintf("%d", totalLogCount)),
		)
	}
	return nil
}
