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

package tasktest

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

// retainedTask overlays the result retention label on a task so that its result stays readable after the graph finishes.
type retainedTask[T any] struct {
	coretask.Task[T]
}

var _ coretask.Task[any] = (*retainedTask[any])(nil)

// Labels returns the labels of the wrapped task with the result retention label set to true.
func (r *retainedTask[T]) Labels() *typedmap.ReadonlyTypedMap {
	retentionLabels := typedmap.NewTypedMap()
	coretask.NewTaskResultRetentionLabel(true).Write(retentionLabels)
	return typedmap.Merge(r.Task.Labels(), retentionLabels)
}

// RunTaskWithDependency runs a task as a graph. Supply the dependencies of the main task to resolve the graph correctly.
func RunTaskWithDependency[T any](baseContext context.Context, mainTask coretask.Task[T], dependencies []coretask.UntypedTask, interceptors ...coretask.Interceptor) (T, error) {
	retainedMainTask := &retainedTask[T]{Task: mainTask}

	availableTasks := make([]coretask.UntypedTask, 0, len(dependencies)+1)
	availableTasks = append(availableTasks, retainedMainTask)
	availableTasks = append(availableTasks, dependencies...)

	resolvedTaskSet, err := coretask.ResolveGraph([]coretask.UntypedTask{retainedMainTask}, availableTasks, nil)
	if err != nil {
		return *new(T), err
	}

	runner, err := coretask.NewLocalRunner(resolvedTaskSet)
	if err != nil {
		return *new(T), err
	}
	for _, interceptor := range interceptors {
		runner.AddInterceptor(interceptor)
	}

	err = runner.Run(baseContext)
	if err != nil {
		return *new(T), err
	}

	<-runner.Wait()

	variableMap, err := runner.Result()
	if err != nil {
		return *new(T), err
	}

	result, found := typedmap.Get(variableMap, typedmap.NewTypedKey[T](mainTask.ID().ReferenceIDString()))
	if !found {
		return *new(T), fmt.Errorf("failed to get the result from the task")
	}

	return result, nil
}
