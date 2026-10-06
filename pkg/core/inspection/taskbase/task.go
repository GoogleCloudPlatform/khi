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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// InspectionTaskFunc is a type for inspection task functions.
type InspectionTaskFunc[T any] = func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (T, error)

// DefineInspectionTask creates an inspection task whose inputs are declared through a coretask.Binder.
// bind declares the inputs with coretask.Use and related functions and returns the task function
// that receives the task mode from the context.
func DefineInspectionTask[T any](id taskid.TaskImplementationID[T], bind func(b *coretask.Binder) InspectionTaskFunc[T], labelOpts ...coretask.LabelOpt) coretask.Task[T] {
	return coretask.Define(id, func(b *coretask.Binder) func(ctx context.Context) (T, error) {
		taskFunc := bind(b)
		return func(ctx context.Context) (T, error) {
			taskMode := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskMode)
			return taskFunc(ctx, taskMode)
		}
	}, labelOpts...)
}
