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

package inspectiontaskbase

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
)

// ProgressReportableInspectionTaskFunc is a type for inspection task functions with progress reporting capabilities.
type ProgressReportableInspectionTaskFunc[T any] = func(ctx context.Context, taskMode inspection_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (T, error)

// InspectionTaskFunc is a type for basic inspection task functions.
type InspectionTaskFunc[T any] = func(ctx context.Context, taskMode inspection_contract.InspectionTaskModeType) (T, error)

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
func NewProgressReportableInspectionTask[T any](taskId taskid.TaskImplementationID[T], dependencies []taskid.UntypedTaskReference, taskFunc ProgressReportableInspectionTaskFunc[T], labelOpts ...coretask.LabelOpt) coretask.Task[T] {

	return NewInspectionTask(taskId, dependencies, func(ctx context.Context, taskMode inspection_contract.InspectionTaskModeType) (T, error) {
		metadataSet := khictx.MustGetValue(ctx, inspection_contract.InspectionRunMetadata)
		progress, found := typedmap.Get(metadataSet, inspectionmetadata.ProgressMetadataKey)
		if !found {
			return *new(T), fmt.Errorf("progress metadata not found")
		}
		defer progress.ResolveTask(taskId.String())
		taskProgress, err := progress.GetOrCreateTaskProgress(taskId.String())
		if err != nil {
			return *new(T), err
		}
		return taskFunc(ctx, taskMode, taskProgress)
	}, append([]coretask.LabelOpt{&inspection_contract.ProgressReportableTaskLabelOptImpl{}}, labelOpts...)...)
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
func NewInspectionTask[T any](taskId taskid.TaskImplementationID[T], dependencies []taskid.UntypedTaskReference, taskFunc InspectionTaskFunc[T], labelOpts ...coretask.LabelOpt) coretask.Task[T] {
	return coretask.NewTask(taskId, dependencies, func(ctx context.Context) (T, error) {
		taskMode := khictx.MustGetValue(ctx, inspection_contract.InspectionTaskMode)
		return taskFunc(ctx, taskMode)

	}, labelOpts...)
}
