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

package coretask

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/google/go-cmp/cmp"
)

func createMockRunnableTask(id string, dependencies []string, runFunc func(ctx context.Context) (any, error), labelOpts ...LabelOpt) UntypedTask {
	deps := make([]Dependency, len(dependencies))
	for i, dep := range dependencies {
		deps[i] = taskid.NewTaskReference[any](dep)
	}

	return NewTask(
		taskid.NewDefaultImplementationID[any](id),
		deps,
		runFunc,
		labelOpts...,
	)
}

func TestLocalRunner_SingleTask(t *testing.T) {
	testCases := []struct {
		name       string
		taskResult string
	}{
		{
			name:       "execute single task and verify retained result",
			taskResult: "task_result",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task := createMockRunnableTask("task1", nil, func(ctx context.Context) (any, error) {
				return tc.taskResult, nil
			}, NewTaskResultRetentionLabel(true))

			runnableSet, err := ResolveGraph([]UntypedTask{task}, []UntypedTask{task}, nil)
			if err != nil {
				t.Fatalf("failed to resolve task graph: %v", err)
			}

			runner, err := NewLocalRunner(runnableSet)
			if err != nil {
				t.Fatalf("failed to create runner: %v", err)
			}

			err = runner.Run(context.Background())
			if err != nil {
				t.Fatalf("failed to run task: %v", err)
			}

			<-runner.Wait()

			val, found := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task1"))
			if !found {
				t.Errorf("expected task result to be found")
			}
			if val != tc.taskResult {
				t.Errorf("expected task result '%v', got '%v'", tc.taskResult, val)
			}
		})
	}
}

func TestLocalRunner_TasksWithDependencies(t *testing.T) {
	testCases := []struct {
		name            string
		wantExecOrder   []string
		wantTask1Result string
		wantTask2Result string
	}{
		{
			name:            "two sequential tasks with data dependency",
			wantExecOrder:   []string{"task1", "task2"},
			wantTask1Result: "result1",
			wantTask2Result: "result2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executionOrder := []string{}
			var mu sync.Mutex

			task1 := createMockRunnableTask("task1", nil, func(ctx context.Context) (any, error) {
				mu.Lock()
				executionOrder = append(executionOrder, "task1")
				mu.Unlock()
				return tc.wantTask1Result, nil
			}, NewTaskResultRetentionLabel(true))

			task2 := createMockRunnableTask("task2", []string{"task1"}, func(ctx context.Context) (any, error) {
				mu.Lock()
				executionOrder = append(executionOrder, "task2")
				mu.Unlock()

				task1Result := GetTaskResult(ctx, taskid.NewTaskReference[string]("task1"))
				if task1Result != tc.wantTask1Result {
					panic("task1 result is not matching")
				}
				return tc.wantTask2Result, nil
			}, NewTaskResultRetentionLabel(true))

			runnableSet, err := ResolveGraph([]UntypedTask{task1, task2}, []UntypedTask{task1, task2}, nil)
			if err != nil {
				t.Fatalf("failed to resolve task graph: %v", err)
			}

			runner, err := NewLocalRunner(runnableSet)
			if err != nil {
				t.Fatalf("failed to create runner: %v", err)
			}

			err = runner.Run(context.Background())
			if err != nil {
				t.Fatalf("failed to run task: %v", err)
			}

			<-runner.Wait()

			if diff := cmp.Diff(tc.wantExecOrder, executionOrder); diff != "" {
				t.Errorf("execution order mismatch (-want +got):\n%s", diff)
			}

			task1Result, found := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task1"))
			if !found {
				t.Errorf("expected task1 result to be found")
			}
			if task1Result != tc.wantTask1Result {
				t.Errorf("expected task1 result '%s', got '%v'", tc.wantTask1Result, task1Result)
			}

			task2Result, found := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task2"))
			if !found {
				t.Errorf("expected task2 result to be found")
			}
			if task2Result != tc.wantTask2Result {
				t.Errorf("expected task2 result '%s', got '%v'", tc.wantTask2Result, task2Result)
			}
		})
	}
}

func TestLocalRunner_ResultCleanup(t *testing.T) {
	testCases := []struct {
		name       string
		setupTasks func() []UntypedTask
		verify     func(t *testing.T, runner *LocalRunner)
	}{
		{
			name: "single task without retention is deleted immediately after completion",
			setupTasks: func() []UntypedTask {
				task := createMockRunnableTask("task1", nil, func(ctx context.Context) (any, error) {
					return "result1", nil
				})
				return []UntypedTask{task}
			},
			verify: func(t *testing.T, runner *LocalRunner) {
				_, found := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task1"))
				if found {
					t.Errorf("expected task1 result to be deleted, but it was found")
				}
			},
		},
		{
			name: "single task with retention is kept after completion",
			setupTasks: func() []UntypedTask {
				task := createMockRunnableTask("task1", nil, func(ctx context.Context) (any, error) {
					return "result1", nil
				}, NewTaskResultRetentionLabel(true))
				return []UntypedTask{task}
			},
			verify: func(t *testing.T, runner *LocalRunner) {
				val, found := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task1"))
				if !found {
					t.Errorf("expected task1 result to be found, but it was not")
				}
				if val != "result1" {
					t.Errorf("expected task1 result 'result1', got '%v'", val)
				}
			},
		},
		{
			name: "dependency chain without retention deletes all intermediate results",
			setupTasks: func() []UntypedTask {
				task1 := createMockRunnableTask("task1", nil, func(ctx context.Context) (any, error) {
					return "result1", nil
				})
				task2 := createMockRunnableTask("task2", []string{"task1"}, func(ctx context.Context) (any, error) {
					task1Val := GetTaskResult(ctx, taskid.NewTaskReference[string]("task1"))
					if task1Val != "result1" {
						return nil, errors.New("unexpected task1 result")
					}
					return "result2", nil
				})
				return []UntypedTask{task1, task2}
			},
			verify: func(t *testing.T, runner *LocalRunner) {
				_, found1 := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task1"))
				if found1 {
					t.Errorf("expected task1 result to be deleted after task2 completes")
				}
				_, found2 := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task2"))
				if found2 {
					t.Errorf("expected task2 result to be deleted because it has no dependents and no retention")
				}
			},
		},
		{
			name: "dependency chain with upstream retained keeps upstream and deletes downstream",
			setupTasks: func() []UntypedTask {
				task1 := createMockRunnableTask("task1", nil, func(ctx context.Context) (any, error) {
					return "result1", nil
				}, NewTaskResultRetentionLabel(true))
				task2 := createMockRunnableTask("task2", []string{"task1"}, func(ctx context.Context) (any, error) {
					task1Val := GetTaskResult(ctx, taskid.NewTaskReference[string]("task1"))
					if task1Val != "result1" {
						return nil, errors.New("unexpected task1 result")
					}
					return "result2", nil
				})
				return []UntypedTask{task1, task2}
			},
			verify: func(t *testing.T, runner *LocalRunner) {
				val1, found1 := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task1"))
				if !found1 {
					t.Errorf("expected task1 result to be kept due to retention label")
				}
				if val1 != "result1" {
					t.Errorf("expected task1 result 'result1', got '%v'", val1)
				}
				_, found2 := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task2"))
				if found2 {
					t.Errorf("expected task2 result to be deleted")
				}
			},
		},
		{
			name: "multi-dependent tasks all receive upstream result before it is deleted",
			setupTasks: func() []UntypedTask {
				task1 := createMockRunnableTask("task1", nil, func(ctx context.Context) (any, error) {
					return "shared_result", nil
				})
				task2 := createMockRunnableTask("task2", []string{"task1"}, func(ctx context.Context) (any, error) {
					val := GetTaskResult(ctx, taskid.NewTaskReference[string]("task1"))
					if val != "shared_result" {
						return nil, errors.New("task2: invalid task1 result")
					}
					time.Sleep(10 * time.Millisecond)
					return "result2", nil
				})
				task3 := createMockRunnableTask("task3", []string{"task1"}, func(ctx context.Context) (any, error) {
					val := GetTaskResult(ctx, taskid.NewTaskReference[string]("task1"))
					if val != "shared_result" {
						return nil, errors.New("task3: invalid task1 result")
					}
					time.Sleep(30 * time.Millisecond)
					return "result3", nil
				})
				return []UntypedTask{task1, task2, task3}
			},
			verify: func(t *testing.T, runner *LocalRunner) {
				_, found := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("task1"))
				if found {
					t.Errorf("expected task1 result to be deleted after all dependents finish")
				}
			},
		},
		{
			name: "diamond dependency with final task retained",
			setupTasks: func() []UntypedTask {
				taskA := createMockRunnableTask("taskA", nil, func(ctx context.Context) (any, error) {
					return "resA", nil
				})
				taskB := createMockRunnableTask("taskB", []string{"taskA"}, func(ctx context.Context) (any, error) {
					resA := GetTaskResult(ctx, taskid.NewTaskReference[string]("taskA"))
					return resA + "->B", nil
				})
				taskC := createMockRunnableTask("taskC", []string{"taskA"}, func(ctx context.Context) (any, error) {
					resA := GetTaskResult(ctx, taskid.NewTaskReference[string]("taskA"))
					return resA + "->C", nil
				})
				taskD := createMockRunnableTask("taskD", []string{"taskB", "taskC"}, func(ctx context.Context) (any, error) {
					resB := GetTaskResult(ctx, taskid.NewTaskReference[string]("taskB"))
					resC := GetTaskResult(ctx, taskid.NewTaskReference[string]("taskC"))
					return resB + "+" + resC + "->D", nil
				}, NewTaskResultRetentionLabel(true))
				return []UntypedTask{taskA, taskB, taskC, taskD}
			},
			verify: func(t *testing.T, runner *LocalRunner) {
				_, foundA := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("taskA"))
				if foundA {
					t.Errorf("expected taskA result to be deleted")
				}
				_, foundB := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("taskB"))
				if foundB {
					t.Errorf("expected taskB result to be deleted")
				}
				_, foundC := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("taskC"))
				if foundC {
					t.Errorf("expected taskC result to be deleted")
				}
				valD, foundD := GetTaskResultFromLocalRunner(runner, taskid.NewTaskReference[string]("taskD"))
				if !foundD {
					t.Errorf("expected taskD result to be retained")
				}
				expectedD := "resA->B+resA->C->D"
				if valD != expectedD {
					if diff := cmp.Diff(expectedD, valD); diff != "" {
						t.Errorf("taskD result mismatch (-want +got):\n%s", diff)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tasks := tc.setupTasks()
			runnableSet, err := ResolveGraph(tasks, tasks, nil)
			if err != nil {
				t.Fatalf("failed to resolve graph: %v", err)
			}

			runner, err := NewLocalRunner(runnableSet)
			if err != nil {
				t.Fatalf("failed to create runner: %v", err)
			}

			err = runner.Run(context.Background())
			if err != nil {
				t.Fatalf("failed to run task: %v", err)
			}

			<-runner.Wait()

			if _, err := runner.Result(); err != nil {
				t.Fatalf("runner completed with error: %v", err)
			}

			tc.verify(t, runner)
		})
	}
}

func TestLocalRunner_TaskError(t *testing.T) {
	testCases := []struct {
		name        string
		expectedErr error
	}{
		{
			name:        "dependent task is skipped when dependency fails",
			expectedErr: errors.New("task error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task1 := createMockRunnableTask("task1", nil, func(ctx context.Context) (any, error) {
				return nil, tc.expectedErr
			})

			task2Executed := false
			task2 := createMockRunnableTask("task2", []string{"task1"}, func(ctx context.Context) (any, error) {
				task2Executed = true
				return "result2", nil
			})

			tasks := []UntypedTask{task1, task2}
			runnableSet, err := ResolveGraph(tasks, tasks, nil)
			if err != nil {
				t.Fatalf("failed to resolve graph: %v", err)
			}

			runner, err := NewLocalRunner(runnableSet)
			if err != nil {
				t.Fatalf("failed to create runner: %v", err)
			}

			err = runner.Run(context.Background())
			if err != nil {
				t.Fatalf("failed to run task: %v", err)
			}

			<-runner.Wait()

			_, err = runner.Result()
			if err == nil {
				t.Error("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tc.expectedErr.Error()) {
				t.Errorf("expected error containing '%s', got '%s'", tc.expectedErr.Error(), err.Error())
			}

			if task2Executed {
				t.Error("dependent task should not be executed when a dependency fails")
			}
		})
	}
}

func TestLocalRunner_ContextCancellation(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "runner cancels task execution on context cancellation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskStarted := make(chan struct{})

			task := createMockRunnableTask("task1", nil, func(ctx context.Context) (any, error) {
				close(taskStarted)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(5 * time.Second):
					return "unexpected completion", nil
				}
			})

			tasks := []UntypedTask{task}
			runnableSet, err := ResolveGraph(tasks, tasks, nil)
			if err != nil {
				t.Fatalf("failed to resolve graph: %v", err)
			}

			runner, err := NewLocalRunner(runnableSet)
			if err != nil {
				t.Fatalf("failed to create runner: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())

			err = runner.Run(ctx)
			if err != nil {
				t.Fatalf("failed to run task: %v", err)
			}

			<-taskStarted

			cancel()

			<-runner.Wait()

			_, err = runner.Result()
			if err == nil {
				t.Error("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), context.Canceled.Error()) {
				t.Errorf("expected error containing '%s', got '%s'", context.Canceled.Error(), err.Error())
			}
		})
	}
}

func TestLocalRunner_AddInterceptor(t *testing.T) {
	testCases := []struct {
		name          string
		expectedOrder []string
	}{
		{
			name: "interceptors wrap task execution in order",
			expectedOrder: []string{
				"interceptor1_start",
				"interceptor2_start",
				"task_execution",
				"interceptor2_end",
				"interceptor1_end",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executionOrder := []string{}

			interceptor1 := func(ctx context.Context, task UntypedTask, next func(context.Context) (any, error)) (any, error) {
				executionOrder = append(executionOrder, "interceptor1_start")
				res, err := next(ctx)
				executionOrder = append(executionOrder, "interceptor1_end")
				return res, err
			}

			interceptor2 := func(ctx context.Context, task UntypedTask, next func(context.Context) (any, error)) (any, error) {
				executionOrder = append(executionOrder, "interceptor2_start")
				res, err := next(ctx)
				executionOrder = append(executionOrder, "interceptor2_end")
				return res, err
			}

			task := createMockRunnableTask("task1", nil, func(ctx context.Context) (any, error) {
				executionOrder = append(executionOrder, "task_execution")
				return "result", nil
			})

			tasks := []UntypedTask{task}
			runnableSet, err := ResolveGraph(tasks, tasks, nil)
			if err != nil {
				t.Fatalf("failed to resolve graph: %v", err)
			}

			runner, err := NewLocalRunner(runnableSet)
			if err != nil {
				t.Fatalf("failed to create runner: %v", err)
			}

			runner.AddInterceptor(interceptor1)
			runner.AddInterceptor(interceptor2)

			err = runner.Run(context.Background())
			if err != nil {
				t.Fatalf("failed to run task: %v", err)
			}

			<-runner.Wait()

			if diff := cmp.Diff(tc.expectedOrder, executionOrder); diff != "" {
				t.Errorf("execution order mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
