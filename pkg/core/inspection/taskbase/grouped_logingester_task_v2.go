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

// GroupedLogIngesterV2 defines the interface for ingesting log metadata into KHI v6 format using group-sequential processing.
type GroupedLogIngesterV2[T any] interface {
	// RawLogTask returns the task reference that provides the raw logs to ingest.
	RawLogTask() taskid.TaskReference[[]*log.Log]
	// GroupedLogTask returns a reference to the task that provides the grouped logs.
	GroupedLogTask() taskid.TaskReference[LogGroupMap]
	// Dependencies returns additional task dependencies of the ingester.
	Dependencies() []taskid.UntypedTaskReference
	// PassCount returns the number of pre-processing passes to perform on each group.
	PassCount() int
	// PreProcessLogByGroup is called during a pre-processing pass for each log in a group.
	// The passIndex is 0-indexed and ranges from 0 to PassCount()-1.
	PreProcessLogByGroup(ctx context.Context, passIndex int, l *log.Log, prevGroupData T) (T, error)
	// ProcessLogByGroup is called for each log entry in a group to customize log metadata.
	// The prevGroupData is the returned value from the last processed log in the same group.
	ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData T) (*khifilev6.LogChangeSet, T, error)
}

// SinglePassGroupedIngesterBase provides a base implementation of GroupedLogIngesterV2
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

// NewGroupedLogIngesterTaskV2 returns a task that ingests log metadata into the KHI v6 builder using group-sequential processing.
func NewGroupedLogIngesterTaskV2[T any](taskID taskid.TaskImplementationID[[]*log.Log], ingester GroupedLogIngesterV2[T], labels ...coretask.LabelOpt) coretask.Task[[]*log.Log] {
	rawLogTaskID := ingester.RawLogTask()
	groupedLogTaskID := ingester.GroupedLogTask()
	dependencies := append([]taskid.UntypedTaskReference{rawLogTaskID, groupedLogTaskID}, ingester.Dependencies()...)
	return NewProgressReportableInspectionTask(taskID, dependencies, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return []*log.Log{}, nil
		}
		logs := coretask.GetTaskResult(ctx, rawLogTaskID)
		groupedLogs := coretask.GetTaskResult(ctx, groupedLogTaskID)
		builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)

		totalLogCount := 0
		var processedLogCount atomic.Uint32
		var skippedLogCount atomic.Uint32
		for _, group := range groupedLogs {
			totalLogCount += len(group.Logs)
		}

		passCount := ingester.PassCount()
		totalSteps := totalLogCount * (passCount + 1)

		progressUpdator := progressutil.NewProgressUpdator(progress, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := processedLogCount.Load()
			if totalSteps > 0 {
				tp.Percentage = float32(current) / float32(totalSteps)
			}
			tp.Message = fmt.Sprintf("%d/%d", current, totalSteps)
		})
		progressUpdator.Start(ctx)

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
						processedLogCount.Add(1)
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
					processedLogCount.Add(1)
					if err != nil {
						logTaskError(ctx, "parser ended with an error", err, l)
						setErr(err)
						return
					}
					groupData = nextGroupData

					if cs != nil {
						err = cs.Flush(builder.LogAccumulator)
						if err != nil {
							logTaskError(ctx, "failed to flush log changeset in V2 ingester", err, l)
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
		progressUpdator.Done()

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if sharedErr != nil {
			return nil, sharedErr
		}

		slog.DebugContext(ctx, fmt.Sprintf("GroupedLogIngesterTaskV2 %s finished: processed %d logs (skipped %d logs)", taskID.String(), totalLogCount, skippedLogCount.Load()))

		tracingActive, _ := khictx.GetValue(ctx, inspectioncore_contract.TracingActive)
		if tracingActive {
			trace.SpanFromContext(ctx).SetAttributes(
				attribute.String("log_count", fmt.Sprintf("%d", totalLogCount)),
			)
		}
		return logs, nil
	}, append([]coretask.LabelOpt{
		// Tasks modifying history must be dependent from SerializerTask.
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref())}, labels...)...)
}
