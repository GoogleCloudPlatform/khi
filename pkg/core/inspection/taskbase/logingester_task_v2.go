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

// LogIngesterV2 defines the interface for ingesting log metadata into KHI v6 format.
type LogIngesterV2 interface {
	// RawLogTask returns the task reference that provides the raw logs to ingest.
	RawLogTask() taskid.TaskReference[[]*log.Log]
	// Dependencies returns additional task dependencies of the ingester.
	Dependencies() []taskid.UntypedTaskReference
	// ProcessLog is called for each log entry to customize log metadata (summary, severity, timestamp, etc.).
	ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error)
}

// NewLogIngesterTaskV2 returns a task that ingests log metadata into the KHI v6 builder.
func NewLogIngesterTaskV2(taskID taskid.TaskImplementationID[[]*log.Log], ingester LogIngesterV2, labels ...coretask.LabelOpt) coretask.Task[[]*log.Log] {
	rawLogTaskID := ingester.RawLogTask()
	dependencies := append([]taskid.UntypedTaskReference{rawLogTaskID}, ingester.Dependencies()...)
	return NewProgressReportableInspectionTask(taskID, dependencies, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return []*log.Log{}, nil
		}
		logs := coretask.GetTaskResult(ctx, rawLogTaskID)
		builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		concurrency := runtime.GOMAXPROCS(0)
		pool := worker.NewPool(concurrency)
		var processedLogCount atomic.Uint32
		var skippedLogCount atomic.Uint32

		progressUpdator := progressutil.NewProgressUpdator(progress, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := processedLogCount.Load()
			if len(logs) > 0 {
				tp.Percentage = float32(current) / float32(len(logs))
			}
			tp.Message = fmt.Sprintf("%d/%d", current, len(logs))
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

		for c := 0; c < concurrency; c++ {
			if ctx.Err() != nil {
				break
			}
			pool.Run(func() {
				for i := c; i < len(logs); i += concurrency {
					if hasErr() {
						return
					}
					if err := ctx.Err(); err != nil {
						setErr(err)
						return
					}
					l := logs[i]
					cs, err := ingester.ProcessLog(ctx, l)
					processedLogCount.Add(1)
					if err != nil {
						logTaskError(ctx, "failed to process log in V2 ingester", err, l)
						setErr(err)
						return
					}
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

		slog.DebugContext(ctx, fmt.Sprintf("LogIngesterTaskV2 %s finished: processed %d logs (skipped %d logs)", taskID.String(), len(logs), skippedLogCount.Load()))

		tracingActive, _ := khictx.GetValue(ctx, inspectioncore_contract.TracingActive)
		if tracingActive {
			trace.SpanFromContext(ctx).SetAttributes(
				attribute.String("log_count", fmt.Sprintf("%d", len(logs))),
			)
		}
		return logs, nil
	}, append([]coretask.LabelOpt{
		// Tasks modifying history must be dependent from SerializerTask.
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref())}, labels...)...)
}
