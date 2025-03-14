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
)

type InspectionProcessorFunc = func(ctx context.Context, taskMode int, v *task.VariableSet, progress *progress.TaskProgress) (any, error)

// NewInspectionProcessor generates a processor task.Definition with progress reporting feature
func NewInspectionProcessor(taskId string, dependencies []string, processor InspectionProcessorFunc, labelOpts ...task.LabelOpt) task.Definition {
	return task.NewProcessorTask(taskId, dependencies, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
		md, err := GetMetadataSetFromVariable(v)
		if err != nil {
			return nil, err
		}
		progress, found := typedmap.Get(md, progress.ProgressMetadataKey)
		if !found {
			return nil, fmt.Errorf("progress metadata not found")
		}
		defer progress.ResolveTask(taskId)
		taskProgress, err := progress.GetTaskProgress(taskId)
		if err != nil {
			return nil, err
		}
		return processor(ctx, taskMode, v, taskProgress)

	}, append([]task.LabelOpt{&ProgressReportableTaskLabelOptImpl{}}, labelOpts...)...)
}

// NewInspectionCachedProcessor generates a cached processor task.Definition with progress reporting feature
func NewInspectionCachedProcessor(taskId string, dependencies []string, processor InspectionProcessorFunc, labelOpts ...task.LabelOpt) task.Definition {
	return task.NewCachedProcessor(taskId, dependencies, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
		md, err := GetMetadataSetFromVariable(v)
		if err != nil {
			return nil, err
		}
		progress, found := typedmap.Get(md, progress.ProgressMetadataKey)
		if !found {
			return nil, fmt.Errorf("progress metadata not found")
		}
		defer progress.ResolveTask(taskId)
		taskProgress, err := progress.GetTaskProgress(taskId)
		if err != nil {
			return nil, err
		}
		return processor(ctx, taskMode, v, taskProgress)

	}, append([]task.LabelOpt{&ProgressReportableTaskLabelOptImpl{}}, labelOpts...)...)
}

type InspectionProducerFunc = func(ctx context.Context, taskMode int, progress *progress.TaskProgress) (any, error)

// NewInspectionProducer generates a producer task.Definition with progress reporting feature
func NewInspectionProducer(taskId string, producer InspectionProducerFunc, labelOpts ...task.LabelOpt) task.Definition {
	return task.NewProcessorTask(taskId, []string{}, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
		md, err := GetMetadataSetFromVariable(v)
		if err != nil {
			return nil, err
		}
		progress, found := typedmap.Get(md, progress.ProgressMetadataKey)
		if !found {
			return nil, fmt.Errorf("progress metadata not found")
		}
		defer progress.ResolveTask(taskId)
		taskProgress, err := progress.GetTaskProgress(taskId)
		if err != nil {
			return nil, err
		}
		return producer(ctx, taskMode, taskProgress)

	}, append([]task.LabelOpt{&ProgressReportableTaskLabelOptImpl{}}, labelOpts...)...)
}
