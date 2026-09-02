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
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LogFilterFunc defines the function signature for filtering logs. It returns true if the log should be kept.
type LogFilterFunc = func(ctx context.Context, log *log.Log) bool

// NewLogFilterTask creates a task that consumes a list of logs and returns a new list
// containing only the logs that satisfy the filter function.
func NewLogFilterTask(tid taskid.TaskImplementationID[[]*log.Log], sourceLogs taskid.TaskReference[[]*log.Log], logFilter LogFilterFunc) coretask.Task[[]*log.Log] {
	return NewLogFilterTaskWithDependencies(tid, sourceLogs, nil, logFilter)
}

// NewLogFilterTaskWithDependencies creates a task that consumes a list of logs and returns a new list
// containing only the logs that satisfy the filter function, with extra task dependencies.
func NewLogFilterTaskWithDependencies(tid taskid.TaskImplementationID[[]*log.Log], sourceLogs taskid.TaskReference[[]*log.Log], extraDependencies []taskid.UntypedTaskReference, logFilter LogFilterFunc) coretask.Task[[]*log.Log] {
	dependencies := append([]taskid.UntypedTaskReference{sourceLogs}, extraDependencies...)
	return NewProgressReportableInspectionTask(tid, dependencies, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
		if taskMode != inspectioncore_contract.TaskModeRun {
			return []*log.Log{}, nil
		}

		logs := coretask.GetTaskResult(ctx, sourceLogs)
		if len(logs) == 0 {
			return []*log.Log{}, nil
		}

		concurrency := min(runtime.GOMAXPROCS(0), len(logs))
		if concurrency <= 0 {
			concurrency = 1
		}

		var completed atomic.Int32
		workerResults := make([][]*log.Log, concurrency)

		progressUpdator := progressutil.NewProgressUpdator(progress, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := int(completed.Load())
			tp.Percentage = float32(current) / float32(len(logs))
			tp.Message = fmt.Sprintf("%d/%d", current, len(logs))
		})
		progressUpdator.Start(ctx)

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
					completed.Add(1)
				}
				workerResults[c] = workerFiltered
			})
		}
		pool.Wait()

		progressUpdator.Done()

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

		tracingActive, _ := khictx.GetValue(ctx, inspectioncore_contract.TracingActive)
		if tracingActive {
			trace.SpanFromContext(ctx).SetAttributes(
				attribute.String("log_count", fmt.Sprintf("%d -> %d", len(logs), len(filteredLogs))),
			)
		}
		return filteredLogs, nil
	})
}
