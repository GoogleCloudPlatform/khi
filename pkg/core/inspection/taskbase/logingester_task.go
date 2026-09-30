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

// LogIngester defines the interface for ingesting log metadata into KHI v6 format.
type LogIngester interface {
	// RawLogTask returns the task reference that provides the raw logs to ingest.
	RawLogTask() taskid.TaskReference[[]*log.Log]
	// Dependencies returns additional task dependencies of the ingester.
	Dependencies() []coretask.Dependency
	// ProcessLog is called for each log entry to customize log metadata (summary, severity, timestamp, etc.).
	ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error)
}

// LogIngesterFunc converts a log into the change set of its metadata, such as summary, severity, and timestamp.
// It returns a nil change set to skip the log.
type LogIngesterFunc = func(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error)

// NewLogIngesterTask returns a task that ingests log metadata into the KHI v6 builder.
func NewLogIngesterTask(taskID taskid.TaskImplementationID[struct{}], ingester LogIngester, labels ...coretask.LabelOpt) coretask.Task[struct{}] {
	rawLogTaskID := ingester.RawLogTask()
	dependencies := append([]coretask.Dependency{rawLogTaskID}, ingester.Dependencies()...)
	allLabels := append([]coretask.LabelOpt{
		coretask.ProvidesTag(TagLogIngester),
	}, labels...)
	return NewInspectionTask(taskID, dependencies, func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (struct{}, error) {
		if taskMode == inspectioncore.TaskModeDryRun {
			return struct{}{}, nil
		}
		return struct{}{}, ingestLogs(ctx, taskID, coretask.GetTaskResult(ctx, rawLogTaskID), ingester.ProcessLog)
	}, allLabels...)
}

// DefineLogIngesterTask returns a task that ingests metadata of the logs provided by rawLogTask into the KHI v6 builder.
// bind declares the additional inputs the ingester reads and returns the function that processes each log.
// The task declares rawLogTask itself.
func DefineLogIngesterTask(taskID taskid.TaskImplementationID[struct{}], rawLogTask taskid.TaskReference[[]*log.Log], bind func(b *coretask.Binder) LogIngesterFunc, labels ...coretask.LabelOpt) coretask.DefinedTask[struct{}] {
	allLabels := append([]coretask.LabelOpt{
		coretask.ProvidesTag(TagLogIngester),
	}, labels...)
	return DefineInspectionTask(taskID, func(b *coretask.Binder) InspectionTaskFunc[struct{}] {
		logs := coretask.Use(b, rawLogTask)
		processLog := bind(b)
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (struct{}, error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return struct{}{}, nil
			}
			return struct{}{}, ingestLogs(ctx, taskID, logs.Get(ctx), processLog)
		}
	}, allLabels...)
}

// ingestLogs processes logs in parallel with processLog and flushes the resulting change sets to the builder in the context.
func ingestLogs(ctx context.Context, taskID taskid.TaskImplementationID[struct{}], logs []*log.Log, processLog LogIngesterFunc) error {
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	if err := ctx.Err(); err != nil {
		return err
	}

	concurrency := runtime.GOMAXPROCS(0)
	pool := worker.NewPool(concurrency)
	var skippedLogCount atomic.Uint32

	tracker := progress.NewTracker(ctx, len(logs), progress.WithUnit("logs"))
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
				cs, err := processLog(ctx, l)
				tracker.Inc()
				if err != nil {
					logTaskError(ctx, "failed to process log in ingester", err, l)
					setErr(err)
					return
				}
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

	slog.DebugContext(ctx, fmt.Sprintf("LogIngesterTask %s finished: processed %d logs (skipped %d logs)", taskID.String(), len(logs), skippedLogCount.Load()))

	tracingActive, _ := khictx.GetValue(ctx, inspectioncore.TracingActive)
	if tracingActive {
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("log_count", fmt.Sprintf("%d", len(logs))),
		)
	}
	return nil
}
