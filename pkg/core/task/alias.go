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

package coretask

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// NewAliasTask generates a new task implementation that proxies the result of another task.
// This is useful for selectively overriding dependencies on a per-task basis.
func NewAliasTask[TaskResult any](taskId taskid.TaskImplementationID[TaskResult], sourceTaskReference taskid.TaskReference[TaskResult], labelOpts ...LabelOpt) *TaskImpl[TaskResult] {
	return NewTask(
		taskId,
		[]taskid.UntypedTaskReference{sourceTaskReference},
		func(ctx context.Context) (TaskResult, error) {
			return GetTaskResult(ctx, sourceTaskReference), nil
		},
		labelOpts...,
	)
}
