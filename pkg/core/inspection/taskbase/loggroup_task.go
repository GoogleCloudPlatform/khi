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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
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

// LogGrouperFunc defines a function that returns a group key for a given log.
type LogGrouperFunc = func(ctx context.Context, log *log.Log) string

// DefineLogGrouperTask creates a task that groups the logs provided by logTask with the grouper function returned by bind.
// bind declares the additional inputs the grouper reads, and the task declares logTask itself.
func DefineLogGrouperTask(taskID taskid.TaskImplementationID[LogGroupMap], logTask taskid.TaskReference[[]*log.Log], bind func(b *coretask.Binder) LogGrouperFunc) coretask.Task[LogGroupMap] {
	return DefineInspectionTask(taskID, func(b *coretask.Binder) InspectionTaskFunc[LogGroupMap] {
		logs := coretask.Use(b, logTask)
		grouper := bind(b)
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (LogGroupMap, error) {
			if taskMode != inspectioncore.TaskModeRun {
				return LogGroupMap{}, nil
			}
			return groupLogs(ctx, logs.Get(ctx), grouper), nil
		}
	})
}

// groupLogs organizes logs into groups keyed by the result of grouper.
func groupLogs(ctx context.Context, logs []*log.Log, grouper LogGrouperFunc) LogGroupMap {
	groups := LogGroupMap{}

	tracker := progress.NewTracker(ctx, len(logs), progress.WithUnit("logs"))
	defer tracker.Done()

	for _, l := range logs {
		group := grouper(ctx, l)
		if _, ok := groups[group]; !ok {
			groups[group] = &LogGroup{
				Group: group,
				Logs:  make([]*log.Log, 0),
			}
		}
		groups[group].Logs = append(groups[group].Logs, l)
		tracker.Inc()
	}

	tracingActive, _ := khictx.GetValue(ctx, inspectioncore.TracingActive)
	if tracingActive {
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("log_count", fmt.Sprintf("%d", len(logs))),
			attribute.String("group_count", fmt.Sprintf("%d", len(groups))),
		)
	}

	return groups
}
