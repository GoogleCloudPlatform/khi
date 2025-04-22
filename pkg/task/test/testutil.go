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

package task_test

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/task/contextkey"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type TaskDependencyValues interface {
	Register(resultMap *typedmap.TypedMap)
}

type taskDependencyValuePair[T any] struct {
	Value T
	Key   taskid.TaskReference[T]
}

// Register implements TaskDependencyValuePair.
func (t *taskDependencyValuePair[T]) Register(resultMap *typedmap.TypedMap) {
	typedmap.Set(resultMap, typedmap.NewTypedKey[T](t.Key.ReferenceIDString()), t.Value)
}

var _ TaskDependencyValues = (*taskDependencyValuePair[any])(nil)

// NewTaskDependencyValuePair returns a new pair of a task reference and its value.
func NewTaskDependencyValuePair[T any](key taskid.TaskReference[T], value T) TaskDependencyValues {
	return &taskDependencyValuePair[T]{
		Value: value,
		Key:   key,
	}
}

// RunTask runs a single task.
func RunTask[T any](baseContext context.Context, task task.Task[T], taskDependencyValues ...TaskDependencyValues) (T, error) {
	taskCtx := prepareTaskContext(baseContext, task, taskDependencyValues...)
	return task.Run(taskCtx)
}

// RunTaskWithDependency runs a task as a graph. Supply the dependencies of the main task to resolve the graph correctly.
func RunTaskWithDependency[T any](baseContext context.Context, mainTask task.Task[T], dependencies []task.UntypedTask) (T, error) {
	taskCtx := prepareTaskContext(baseContext, mainTask)

	taskSet, err := task.NewTaskSet([]task.UntypedTask{mainTask})
	if err != nil {
		return *new(T), err
	}
	allTaskSet, err := task.NewTaskSet(dependencies)
	if err != nil {
		return *new(T), err
	}
	resolvedTaskSet, err := taskSet.ResolveTask(allTaskSet)
	if err != nil {
		return *new(T), err
	}

	runner, err := task.NewLocalRunner(resolvedTaskSet)
	if err != nil {
		return *new(T), err
	}

	err = runner.Run(taskCtx)
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

func prepareTaskContext(baseContext context.Context, task task.UntypedTask, taskDependencyValues ...TaskDependencyValues) context.Context {
	taskCtx := khictx.WithValue(baseContext, task_contextkey.TaskImplementationIDContextKey, task.UntypedID())

	resultMap := typedmap.NewTypedMap()
	for _, taskDependencyValue := range taskDependencyValues {
		taskDependencyValue.Register(resultMap)
	}

	taskCtx = khictx.WithValue(taskCtx, task_contextkey.TaskResultMapContextKey, resultMap)

	return taskCtx
}

// StubTask wraps a given task to return the constant values given without calling the original task.
func StubTask[T any](mockTarget task.Task[T], mockResult T, mockError error) task.Task[T] {
	return task.NewTask(mockTarget.ID(), []taskid.UntypedTaskReference{}, func(ctx context.Context) (T, error) {
		return mockResult, mockError
	}, task.FromLabels(mockTarget.Labels())...)
}

// StubTaskFromReferenceID creates a new test task return the given constant value of its result.
func StubTaskFromReferenceID[T any](mockTargetReference taskid.TaskReference[T], mockResult T, mockError error) task.Task[T] {
	return task.NewTask(taskid.NewDefaultImplementationID[T](mockTargetReference.ReferenceIDString()), []taskid.UntypedTaskReference{}, func(ctx context.Context) (T, error) {
		return mockResult, mockError
	})
}
