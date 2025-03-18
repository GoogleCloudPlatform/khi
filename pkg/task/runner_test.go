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
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func createMockTask(id string, dependencies []string, runFunc func(ctx context.Context, taskMode int, v *VariableSet) (any, error)) UntypedDefinition {
	deps := make([]taskid.UntypedTaskReference, len(dependencies))
	for i, dep := range dependencies {
		deps[i] = taskid.NewTaskReference[any](dep)
	}

	return NewDefinitionFromFunc(
		taskid.NewDefaultImplementationID[any](id),
		deps,
		runFunc,
	)
}

func TestLocalRunner_SingleTask(t *testing.T) {
	taskResult := "task_result"
	task := createMockTask("task1", nil, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		return taskResult, nil
	})

	definitionSet, err := NewSet([]UntypedDefinition{task})
	if err != nil {
		t.Fatalf("Failed to create definition set: %v", err)
	}

	sortResult := definitionSet.sortTaskGraph()
	runnableSet := &DefinitionSet{definitions: sortResult.TopologicalSortedTasks, runnable: true}

	runner, err := NewLocalRunner(runnableSet)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	err = runner.Run(context.Background(), 0, nil)
	if err != nil {
		t.Fatalf("Failed to run task: %v", err)
	}

	<-runner.Wait()

	result, err := runner.Result()
	if err != nil {
		t.Fatalf("Failed to get result: %v", err)
	}

	val, err := GetTypedVariableFromTaskVariable(result, "task1", "")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if val != taskResult {
		t.Errorf("Expected task result '%v', got '%v'", taskResult, val)
	}
}

func TestLocalRunner_TasksWithDependencies(t *testing.T) {
	executionOrder := []string{}
	var mu sync.Mutex

	task1 := createMockTask("task1", nil, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		mu.Lock()
		executionOrder = append(executionOrder, "task1")
		mu.Unlock()
		return "result1", nil
	})

	task2 := createMockTask("task2", []string{"task1"}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		mu.Lock()
		executionOrder = append(executionOrder, "task2")
		mu.Unlock()

		_, err := GetTypedVariableFromTaskVariable(v, "task1", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		return "result2", nil
	})

	definitionSet, err := NewSet([]UntypedDefinition{task1, task2})
	if err != nil {
		t.Fatalf("Failed to create definition set: %v", err)
	}

	sortResult := definitionSet.sortTaskGraph()
	runnableSet := &DefinitionSet{definitions: sortResult.TopologicalSortedTasks, runnable: true}

	runner, err := NewLocalRunner(runnableSet)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	err = runner.Run(context.Background(), 0, nil)
	if err != nil {
		t.Fatalf("Failed to run task: %v", err)
	}

	<-runner.Wait()

	if len(executionOrder) != 2 {
		t.Errorf("Expected 2 tasks to be executed, got %d", len(executionOrder))
	}
	if executionOrder[0] != "task1" {
		t.Errorf("Expected task1 to be executed first, got %s", executionOrder[0])
	}
	if executionOrder[1] != "task2" {
		t.Errorf("Expected task2 to be executed second, got %s", executionOrder[1])
	}

	result, err := runner.Result()
	if err != nil {
		t.Fatalf("Failed to get result: %v", err)
	}

	val1, err := GetTypedVariableFromTaskVariable(result, "task1", "")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if val1 != "result1" {
		t.Errorf("Expected task1 result 'result1', got '%v'", val1)
	}

	val2, err := GetTypedVariableFromTaskVariable(result, "task2", "")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if val2 != "result2" {
		t.Errorf("Expected task2 result 'result2', got '%v'", val2)
	}
}

func TestLocalRunner_TaskError(t *testing.T) {
	expectedErr := errors.New("task error")

	task1 := createMockTask("task1", nil, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		return nil, expectedErr
	})

	task2Executed := false
	task2 := createMockTask("task2", []string{"task1"}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		task2Executed = true
		return "result2", nil
	})

	definitionSet, err := NewSet([]UntypedDefinition{task1, task2})
	if err != nil {
		t.Fatalf("Failed to create definition set: %v", err)
	}

	sortResult := definitionSet.sortTaskGraph()
	runnableSet := &DefinitionSet{definitions: sortResult.TopologicalSortedTasks, runnable: true}

	runner, err := NewLocalRunner(runnableSet)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	err = runner.Run(context.Background(), 0, nil)
	if err != nil {
		t.Fatalf("Failed to run task: %v", err)
	}

	<-runner.Wait()

	_, err = runner.Result()
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedErr.Error(), err.Error())
	}

	if task2Executed {
		t.Error("Dependent task should not be executed when a dependency fails")
	}
}

func TestLocalRunner_ContextCancellation(t *testing.T) {
	taskStarted := make(chan struct{})

	task := createMockTask("task1", nil, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		close(taskStarted)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return "unexpected completion", nil
		}
	})

	definitionSet, err := NewSet([]UntypedDefinition{task})
	if err != nil {
		t.Fatalf("Failed to create definition set: %v", err)
	}

	sortResult := definitionSet.sortTaskGraph()
	runnableSet := &DefinitionSet{definitions: sortResult.TopologicalSortedTasks, runnable: true}

	runner, err := NewLocalRunner(runnableSet)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	err = runner.Run(ctx, 0, nil)
	if err != nil {
		t.Fatalf("Failed to run task: %v", err)
	}

	<-taskStarted

	cancel()

	<-runner.Wait()

	_, err = runner.Result()
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Errorf("Expected error containing '%s', got '%s'", context.Canceled.Error(), err.Error())
	}
}
