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
	"slices"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
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
func RunTask[T any](baseContext context.Context, task coretask.Task[T], taskDependencyValues ...TaskDependencyValues) (T, error) {
	taskCtx := prepareTaskContext(baseContext, task, taskDependencyValues...)
	return task.Run(taskCtx)
}

// RunTaskWithDependency runs a task as a graph. Supply the dependencies of the main task to resolve the graph correctly.
func RunTaskWithDependency[T any](baseContext context.Context, mainTask coretask.Task[T], dependencies []coretask.UntypedTask) (T, error) {
	retainedMainTask := coretask.NewTask(
		mainTask.ID(),
		mainTask.Dependencies(),
		mainTask.Run,
		append(coretask.FromLabels(mainTask.Labels()), coretask.NewTaskResultRetentionLabel(true))...,
	)
	taskCtx := prepareTaskContext(baseContext, retainedMainTask)

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

type testGraphMetadata struct {
	resultMap *typedmap.TypedMap
}

func (m *testGraphMetadata) IsBound(referenceID string) bool {
	return slices.Contains(m.resultMap.Keys(), referenceID)
}

func (m *testGraphMetadata) BoundReferenceIDsForTaskImplWithTag(taskImplementationID string, tag string) []string {
	return nil
}

var _ core_contract.TaskGraphMetadata = (*testGraphMetadata)(nil)

func prepareTaskContext(baseContext context.Context, task coretask.UntypedTask, taskDependencyValues ...TaskDependencyValues) context.Context {
	taskCtx := khictx.WithValue(baseContext, core_contract.TaskImplementationIDContextKey, task.UntypedID())

	resultMap := typedmap.NewTypedMap()
	for _, taskDependencyValue := range taskDependencyValues {
		taskDependencyValue.Register(resultMap)
	}

	taskCtx = khictx.WithValue(taskCtx, core_contract.TaskResultMapContextKey, resultMap)
	if _, err := khictx.GetValue(baseContext, core_contract.TaskGraphMetadataContextKey); err != nil {
		taskCtx = khictx.WithValue[core_contract.TaskGraphMetadata](taskCtx, core_contract.TaskGraphMetadataContextKey, &testGraphMetadata{resultMap: resultMap})
	}

	return taskCtx
}

// StubTask wraps a given task to return the constant values given without calling the original task.
func StubTask[T any](mockTarget coretask.Task[T], mockResult T, mockError error) coretask.Task[T] {
	return coretask.NewTask(mockTarget.ID(), []coretask.Dependency{}, func(ctx context.Context) (T, error) {
		return mockResult, mockError
	}, coretask.FromLabels(mockTarget.Labels())...)
}

// StubTaskFromReferenceID creates a new test task return the given constant value of its result.
func StubTaskFromReferenceID[T any](mockTargetReference taskid.TaskReference[T], mockResult T, mockError error) coretask.Task[T] {
	return coretask.NewTask(taskid.NewDefaultImplementationID[T](mockTargetReference.ReferenceIDString()), []coretask.Dependency{}, func(ctx context.Context) (T, error) {
		return mockResult, mockError
	})
}

// WithTaskResult adds task result to the given context. It's for testing a function using coretask.GetTaskResult inside.
func WithTaskResult[T any](ctx context.Context, taskRef taskid.TaskReference[T], value T) context.Context {
	resultMap, err := khictx.GetValue(ctx, core_contract.TaskResultMapContextKey)
	if err != nil {
		resultMap = typedmap.NewTypedMap()
		ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, resultMap)
	}
	typedmap.Set(resultMap, typedmap.NewTypedKey[T](taskRef.ReferenceIDString()), value)
	if _, err := khictx.GetValue(ctx, core_contract.TaskGraphMetadataContextKey); err != nil {
		ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, &testGraphMetadata{resultMap: resultMap})
	}
	return ctx
}

// WithTaskGraphMetadata adds custom TaskGraphMetadata to the given context for testing.
func WithTaskGraphMetadata(ctx context.Context, meta core_contract.TaskGraphMetadata) context.Context {
	return khictx.WithValue(ctx, core_contract.TaskGraphMetadataContextKey, meta)
}
