// Copyright 2024 Google LLC
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

package inspection_task

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// ProgressReportableInspectionTaskFunc is a type for inspection task functions with progress reporting capabilities.
type ProgressReportableInspectionTaskFunc[T any] = func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) (T, error)

// InspectionTaskFunc is a type for basic inspection task functions.
type InspectionTaskFunc[T any] = func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode) (T, error)

// NewProgressReportableInspectionTask generates a task with progress reporting capabilities.
// This task can report its progress during execution through the TaskProgress object.
// Use NewInspectionTask instead for tasks immediately ends.
// Parameters:
//   - taskId: Unique identifier for the task
//   - dependencies: List of task references this task depends on
//   - taskFunc: Task execution function with progress reporting capability
//   - labelOpts: Label options to apply to the task
//
// Returns: A task with progress reporting capabilities
func NewProgressReportableInspectionTask[T any](taskId taskid.TaskImplementationID[T], dependencies []taskid.UntypedTaskReference, taskFunc ProgressReportableInspectionTaskFunc[T], labelOpts ...task.LabelOpt) task.Task[T] {

	return NewInspectionTask(taskId, dependencies, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode) (T, error) {
		metadataSet := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionRunMetadata)
		progress, found := typedmap.Get(metadataSet, progress.ProgressMetadataKey)
		if !found {
			return *new(T), fmt.Errorf("progress metadata not found")
		}
		defer progress.ResolveTask(taskId.String())
		taskProgress, err := progress.GetTaskProgress(taskId.String())
		if err != nil {
			return *new(T), err
		}
		return taskFunc(ctx, taskMode, taskProgress)
	}, append([]task.LabelOpt{&ProgressReportableTaskLabelOptImpl{}}, labelOpts...)...)
}

// NewInspectionTask creates a basic inspection task.
// The task is executed based on the task mode retrieved from the context.
// Parameters:
//   - taskId: Unique identifier for the task
//   - dependencies: List of task references this task depends on
//   - taskFunc: Task execution function
//   - labelOpts: Label options to apply to the task
//
// Returns: An inspection task
func NewInspectionTask[T any](taskId taskid.TaskImplementationID[T], dependencies []taskid.UntypedTaskReference, taskFunc InspectionTaskFunc[T], labelOpts ...task.LabelOpt) task.Task[T] {
	return task.NewTask(taskId, dependencies, func(ctx context.Context) (T, error) {
		taskMode := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionTaskMode)
		return taskFunc(ctx, taskMode)

	}, labelOpts...)
}
