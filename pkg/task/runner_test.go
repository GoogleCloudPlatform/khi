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

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func createMockTask(id string, dependencies []string, runFunc func(ctx context.Context) (any, error)) UntypedTask {
	deps := make([]taskid.UntypedTaskReference, len(dependencies))
	for i, dep := range dependencies {
		deps[i] = taskid.NewTaskReference[any](dep)
	}

	return NewTask(
		taskid.NewDefaultImplementationID[any](id),
		deps,
		runFunc,
	)
}

func TestLocalRunner_SingleTask(t *testing.T) {
	taskResult := "task_result"
	task := createMockTask("task1", nil, func(ctx context.Context) (any, error) {
		return taskResult, nil
	})

	taskSet, err := NewSet([]UntypedTask{task})
	if err != nil {
		t.Fatalf("Failed to create task set: %v", err)
	}

	sortResult := taskSet.sortTaskGraph()
	runnableSet := &TaskSet{tasks: sortResult.TopologicalSortedTasks, runnable: true}

	runner, err := NewLocalRunner(runnableSet)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	err = runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Failed to run task: %v", err)
	}

	<-runner.Wait()

	val, found := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task1"))
	if !found {
		t.Errorf("Expected task result to be found")
	}
	if val != taskResult {
		t.Errorf("Expected task result '%v', got '%v'", taskResult, val)
	}
}

func TestLocalRunner_TasksWithDependencies(t *testing.T) {
	executionOrder := []string{}
	var mu sync.Mutex

	task1 := createMockTask("task1", nil, func(ctx context.Context) (any, error) {
		mu.Lock()
		executionOrder = append(executionOrder, "task1")
		mu.Unlock()
		return "result1", nil
	})

	task2 := createMockTask("task2", []string{"task1"}, func(ctx context.Context) (any, error) {
		mu.Lock()
		executionOrder = append(executionOrder, "task2")
		mu.Unlock()

		task1Result := GetTaskResult(ctx, taskid.NewTaskReference[string]("task1"))
		if task1Result != "result1" {
			panic("task1 result is not matching")
		}
		return "result2", nil
	})

	taskSet, err := NewSet([]UntypedTask{task1, task2})
	if err != nil {
		t.Fatalf("Failed to create task set: %v", err)
	}

	sortResult := taskSet.sortTaskGraph()
	runnableSet := &TaskSet{tasks: sortResult.TopologicalSortedTasks, runnable: true}

	runner, err := NewLocalRunner(runnableSet)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	err = runner.Run(context.Background())
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

	task1Result, found := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task1"))
	if !found {
		t.Errorf("Expected task result to be found")
	}
	if task1Result != "result1" {
		t.Errorf("Expected task1 result 'result1', got '%v'", task1Result)
	}

	task2Result, found := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task2"))
	if !found {
		t.Errorf("Expected task result to be found")
	}
	if task2Result != "result2" {
		t.Errorf("Expected task2 result 'result2', got '%v'", task2Result)
	}
}

func TestLocalRunner_TaskError(t *testing.T) {
	expectedErr := errors.New("task error")

	task1 := createMockTask("task1", nil, func(ctx context.Context) (any, error) {
		return nil, expectedErr
	})

	task2Executed := false
	task2 := createMockTask("task2", []string{"task1"}, func(ctx context.Context) (any, error) {
		task2Executed = true
		return "result2", nil
	})

	taskSet, err := NewSet([]UntypedTask{task1, task2})
	if err != nil {
		t.Fatalf("Failed to create task set: %v", err)
	}

	sortResult := taskSet.sortTaskGraph()
	runnableSet := &TaskSet{tasks: sortResult.TopologicalSortedTasks, runnable: true}

	runner, err := NewLocalRunner(runnableSet)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	err = runner.Run(context.Background())
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

	task := createMockTask("task1", nil, func(ctx context.Context) (any, error) {
		close(taskStarted)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return "unexpected completion", nil
		}
	})

	taskSet, err := NewSet([]UntypedTask{task})
	if err != nil {
		t.Fatalf("Failed to create task set: %v", err)
	}

	sortResult := taskSet.sortTaskGraph()
	runnableSet := &TaskSet{tasks: sortResult.TopologicalSortedTasks, runnable: true}

	runner, err := NewLocalRunner(runnableSet)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	err = runner.Run(ctx)
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
