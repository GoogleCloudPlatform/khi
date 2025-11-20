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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LogGroup holds a collection of logs that belong to the same group.
type LogGroup struct {
	Group string
	Logs  []*log.Log
}

// LogGroupMap is a map of log groups, where the key is the group identifier.
type LogGroupMap = map[string]*LogGroup

// LogGrouper defines a function that returns a group key for a given log.
type LogGrouper = func(ctx context.Context, log *log.Log) string

// NewLogGrouperTask creates a task that groups logs based on a grouper function.
// It processes a list of logs and organizes them into a map of LogGroup,
// where each group contains logs with the same key.
func NewLogGrouperTask(taskId taskid.TaskImplementationID[LogGroupMap], logTask taskid.TaskReference[[]*log.Log], grouper LogGrouper) coretask.Task[LogGroupMap] {
	return NewProgressReportableInspectionTask(taskId, []taskid.UntypedTaskReference{
		logTask,
	},
		func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (LogGroupMap, error) {
			if taskMode != inspectioncore_contract.TaskModeRun {
				return LogGroupMap{}, nil
			}

			logs := coretask.GetTaskResult(ctx, logTask)
			groups := LogGroupMap{}
			completed := 0

			progressUpdator := progressutil.NewProgressUpdator(progress, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
				tp.Percentage = float32(completed) / float32(len(logs))
				tp.Message = fmt.Sprintf("%d/%d", completed, len(logs))
			})
			progressUpdator.Start(ctx)

			for _, l := range logs {
				group := grouper(ctx, l)
				if _, ok := groups[group]; !ok {
					groups[group] = &LogGroup{
						Group: group,
						Logs:  make([]*log.Log, 0),
					}
				}
				groups[group].Logs = append(groups[group].Logs, l)
				completed++
			}

			progressUpdator.Done()

			tracingActive, _ := khictx.GetValue(ctx, inspectioncore_contract.TracingActive)
			if tracingActive {
				trace.SpanFromContext(ctx).SetAttributes(
					attribute.String("log_count", fmt.Sprintf("%d", len(logs))),
					attribute.String("group_count", fmt.Sprintf("%d", len(groups))),
				)
			}

			return groups, nil
		})
}
