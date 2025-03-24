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

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type ProcessorFunc[T any] = func(ctx context.Context, taskMode int, v *VariableSet) (T, error)

// NewProcessor returns a task definition generates a variable named the task Id from one or more variables generated from the dependency.
// A processor task set the variable that has the same name of the task Id at the end.
func NewProcessorTask[T any](taskImplementationID taskid.TaskImplementationID[T], dependencies []taskid.UntypedTaskReference, processor ProcessorFunc[T], labelOpts ...LabelOpt) Definition[T] {
	return NewDefinitionFromFunc(taskImplementationID, dependencies, func(ctx context.Context, taskMode int, v *VariableSet) (T, error) {
		return processor(ctx, taskMode, v)
	}, labelOpts...)
}
