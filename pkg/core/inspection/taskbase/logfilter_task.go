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

package inspectiontaskbase

import (
	"context"
	"fmt"
	"runtime"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LogFilterFunc defines the function signature for filtering logs. It returns true if the log should be kept.
type LogFilterFunc = func(ctx context.Context, log *log.Log) bool

// DefineLogFilterTask creates a task that returns only the logs provided by sourceLogs that the filter function returned by bind keeps.
// bind declares the additional inputs the filter reads, and the task declares sourceLogs itself.
func DefineLogFilterTask(taskID taskid.TaskImplementationID[[]*log.Log], sourceLogs taskid.TaskReference[[]*log.Log], bind func(b *coretask.Binder) LogFilterFunc, labels ...coretask.LabelOpt) coretask.Task[[]*log.Log] {
	return DefineInspectionTask(taskID, func(b *coretask.Binder) InspectionTaskFunc[[]*log.Log] {
		logs := coretask.Use(b, sourceLogs)
		logFilter := bind(b)
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*log.Log, error) {
			if taskMode != inspectioncore.TaskModeRun {
				return []*log.Log{}, nil
			}
			return filterLogs(ctx, logs.Get(ctx), logFilter)
		}
	}, labels...)
}

// filterLogs returns the logs that logFilter keeps in their original order, evaluating logFilter in parallel.
func filterLogs(ctx context.Context, logs []*log.Log, logFilter LogFilterFunc) ([]*log.Log, error) {
	if len(logs) == 0 {
		return []*log.Log{}, nil
	}

	concurrency := min(runtime.GOMAXPROCS(0), len(logs))
	if concurrency <= 0 {
		concurrency = 1
	}

	workerResults := make([][]*log.Log, concurrency)

	tracker := progress.NewTracker(ctx, len(logs), progress.WithUnit("logs"))
	defer tracker.Done()

	pool := worker.NewPool(concurrency)
	for c := 0; c < concurrency; c++ {
		c := c
		pool.Run(func() {
			start := c * len(logs) / concurrency
			end := (c + 1) * len(logs) / concurrency
			var workerFiltered []*log.Log
			for i := start; i < end; i++ {
				if ctx.Err() != nil {
					return
				}
				if logFilter(ctx, logs[i]) {
					workerFiltered = append(workerFiltered, logs[i])
				}
				tracker.Inc()
			}
			workerResults[c] = workerFiltered
		})
	}
	pool.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	totalCount := 0
	for _, wr := range workerResults {
		totalCount += len(wr)
	}
	filteredLogs := make([]*log.Log, 0, totalCount)
	for _, wr := range workerResults {
		filteredLogs = append(filteredLogs, wr...)
	}

	tracingActive, _ := khictx.GetValue(ctx, inspectioncore.TracingActive)
	if tracingActive {
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("log_count", fmt.Sprintf("%d -> %d", len(logs), len(filteredLogs))),
		)
	}
	return filteredLogs, nil
}
