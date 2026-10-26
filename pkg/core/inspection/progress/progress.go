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

package progress

import (
	"context"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

type taskProgressContextKey struct{}

var noopProgress = inspectionmetadata.NewNoopTaskProgressMetadata()

// ResolveTitle determines the display title for a task using its explicit title label or its shortened task ID.
func ResolveTitle(task coretask.UntypedTask) string {
	if title, found := typedmap.Get(task.Labels(), coretask.LabelKeyTaskTitle); found && title != "" {
		return title
	}
	id := task.UntypedID().String()
	if after, found := strings.CutPrefix(id, "khi.google.com/inspection/"); found {
		id = after
	} else if after, found := strings.CutPrefix(id, "khi.google.com/"); found {
		id = after
	}
	id = strings.TrimSuffix(id, "#default")
	return id
}

// WithContext embeds a TaskProgressMetadata instance into the provided context.
func WithContext(ctx context.Context, tp *inspectionmetadata.TaskProgressMetadata) context.Context {
	return context.WithValue(ctx, taskProgressContextKey{}, tp)
}

// FromContext retrieves the TaskProgressMetadata bound to the context, returning a no-op instance if absent.
func FromContext(ctx context.Context) *inspectionmetadata.TaskProgressMetadata {
	if tp, ok := ctx.Value(taskProgressContextKey{}).(*inspectionmetadata.TaskProgressMetadata); ok && tp != nil {
		return tp
	}
	return noopProgress
}

// TaskInterceptor attaches a TaskProgressMetadata to the task execution context and resolves it when the task completes.
// It also wraps task execution in a task-scoped cancelable context so any background progress trackers are cleanly stopped upon task exit.
func TaskInterceptor(ctx context.Context, task coretask.UntypedTask, next func(context.Context) (any, error)) (any, error) {
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if metadataSet, err := khictx.GetValue(taskCtx, inspectionmetadata.MapContextKey); err == nil {
		if progressMeta, found := typedmap.Get(metadataSet, inspectionmetadata.ProgressMetadataKey); found {
			taskID := task.UntypedID().String()
			if tp, err := progressMeta.GetOrCreateTaskProgress(taskID); err == nil {
				tp.SetLabel(ResolveTitle(task))
				taskCtx = WithContext(taskCtx, tp)
				defer func() {
					_ = progressMeta.ResolveTask(taskID)
				}()
			}
		}
	}
	return next(taskCtx)
}

// Report updates the task progress completion ratio (0.0 to 1.0) and message directly from context.
func Report(ctx context.Context, ratio float32, message string) {
	FromContext(ctx).Update(ratio, message)
}

// ReportIndeterminate marks the current task progress as indeterminate with the given status message.
func ReportIndeterminate(ctx context.Context, message string) {
	FromContext(ctx).UpdateIndeterminate(message)
}
