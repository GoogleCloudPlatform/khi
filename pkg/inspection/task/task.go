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

package task

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type InspectionProcessorFunc[T any] = func(ctx context.Context, taskMode int, v *task.VariableSet, progress *progress.TaskProgress) (T, error)

// NewInspectionProcessor generates a processor task.Definition with progress reporting feature
func NewInspectionProcessor[T any](taskId taskid.TaskImplementationID[T], dependencies []taskid.UntypedTaskReference, processor InspectionProcessorFunc[T], labelOpts ...task.LabelOpt) task.Definition[T] {
	return task.NewProcessorTask(taskId, dependencies, func(ctx context.Context, taskMode int, v *task.VariableSet) (T, error) {
		md, err := GetMetadataSetFromVariable(v)
		if err != nil {
			return *new(T), err
		}
		progress, found := typedmap.Get(md, progress.ProgressMetadataKey)
		if !found {
			return *new(T), fmt.Errorf("progress metadata not found")
		}
		defer progress.ResolveTask(taskId.String())
		taskProgress, err := progress.GetTaskProgress(taskId.String())
		if err != nil {
			return *new(T), err
		}
		return processor(ctx, taskMode, v, taskProgress)

	}, append([]task.LabelOpt{&ProgressReportableTaskLabelOptImpl{}}, labelOpts...)...)
}

// NewInspectionCachedProcessor generates a cached processor task.Definition with progress reporting feature
func NewInspectionCachedProcessor[T any](taskId taskid.TaskImplementationID[T], dependencies []taskid.UntypedTaskReference, processor InspectionProcessorFunc[T], labelOpts ...task.LabelOpt) task.Definition[T] {
	return task.NewCachedProcessor(taskId, dependencies, func(ctx context.Context, taskMode int, v *task.VariableSet) (T, error) {
		md, err := GetMetadataSetFromVariable(v)
		if err != nil {
			return *new(T), err
		}
		progress, found := typedmap.Get(md, progress.ProgressMetadataKey)
		if !found {
			return *new(T), fmt.Errorf("progress metadata not found")
		}
		defer progress.ResolveTask(taskId.String())
		taskProgress, err := progress.GetTaskProgress(taskId.String())
		if err != nil {
			return *new(T), err
		}
		return processor(ctx, taskMode, v, taskProgress)

	}, append([]task.LabelOpt{&ProgressReportableTaskLabelOptImpl{}}, labelOpts...)...)
}

type InspectionProducerFunc[T any] = func(ctx context.Context, taskMode int, progress *progress.TaskProgress) (T, error)

// NewInspectionProducer generates a producer task.Definition with progress reporting feature
func NewInspectionProducer[T any](taskId taskid.TaskImplementationID[T], producer InspectionProducerFunc[T], labelOpts ...task.LabelOpt) task.Definition[T] {
	return task.NewProcessorTask(taskId, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode int, v *task.VariableSet) (T, error) {
		md, err := GetMetadataSetFromVariable(v)
		if err != nil {
			return *new(T), err
		}
		progress, found := typedmap.Get(md, progress.ProgressMetadataKey)
		if !found {
			return *new(T), fmt.Errorf("progress metadata not found")
		}
		defer progress.ResolveTask(taskId.String())
		taskProgress, err := progress.GetTaskProgress(taskId.String())
		if err != nil {
			return *new(T), err
		}
		return producer(ctx, taskMode, taskProgress)

	}, append([]task.LabelOpt{&ProgressReportableTaskLabelOptImpl{}}, labelOpts...)...)
}
