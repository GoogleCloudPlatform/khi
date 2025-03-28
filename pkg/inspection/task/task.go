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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type InspectionProcessorFunc[T any] = func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) (T, error)

// NewInspectionTask generates a processor task.Definition with progress reporting feature
func NewInspectionTask[T any](taskId taskid.TaskImplementationID[T], dependencies []taskid.UntypedTaskReference, processor InspectionProcessorFunc[T], labelOpts ...task.LabelOpt) task.Definition[T] {
	return task.NewTask(taskId, dependencies, func(ctx context.Context) (T, error) {
		taskMode := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionTaskMode)
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
		return processor(ctx, taskMode, taskProgress)

	}, append([]task.LabelOpt{&ProgressReportableTaskLabelOptImpl{}}, labelOpts...)...)
}
