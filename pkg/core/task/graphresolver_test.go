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
	"fmt"
	"sort"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// mockUntypedTask is a mock implementation of the UntypedTask interface for testing.
type mockUntypedTask struct {
	id           taskid.UntypedTaskImplementationID
	labels       *typedmap.ReadonlyTypedMap
	dependencies []taskid.UntypedTaskReference
}

func (m *mockUntypedTask) UntypedID() taskid.UntypedTaskImplementationID { return m.id }
func (m *mockUntypedTask) Labels() *typedmap.ReadonlyTypedMap            { return m.labels }
func (m *mockUntypedTask) Dependencies() []taskid.UntypedTaskReference   { return m.dependencies }
func (m *mockUntypedTask) UntypedRun(ctx context.Context) (any, error)   { return nil, nil }

var _ UntypedTask = (*mockUntypedTask)(nil)

// newMockTaskWithRequiredLabel creates a new mockUntypedTask for testing, with an option to set the required label.
func newMockTaskWithRequiredLabel(idStr string, required bool) UntypedTask {
	id := taskid.NewDefaultImplementationID[any](idStr)
	var labels *typedmap.ReadonlyTypedMap
	if required {
		labels = NewLabelSet(WithLabelValue(LabelKeyRequiredTask, true))
	} else {
		labels = NewLabelSet()
	}
	return &mockUntypedTask{
		id:     id,
		labels: labels,
	}
}

func TestRequiredTaskLabelGraphResolverRule_Resolve(t *testing.T) {
	optionalTaskA := newMockTaskWithRequiredLabel("optional-task-a", false)
	requiredTaskB := newMockTaskWithRequiredLabel("required-task-b", true)
	requiredTaskC := newMockTaskWithRequiredLabel("required-task-c", true)

	testCases := []struct {
		name              string
		currentGraphTasks []UntypedTask
		availableTasks    []UntypedTask
		expectedTasks     []string // task IDs
		expectedChanged   bool
		expectErr         bool
	}{
		{
			name:              "should add a required task",
			currentGraphTasks: []UntypedTask{optionalTaskA},
			availableTasks:    []UntypedTask{optionalTaskA, requiredTaskB},
			expectedTasks:     []string{"optional-task-a#default", "required-task-b#default"},
			expectedChanged:   true,
			expectErr:         false,
		},
		{
			name:              "should not add an already existing task",
			currentGraphTasks: []UntypedTask{optionalTaskA, requiredTaskB},
			availableTasks:    []UntypedTask{optionalTaskA, requiredTaskB},
			expectedTasks:     []string{"optional-task-a#default", "required-task-b#default"},
			expectedChanged:   false,
			expectErr:         false,
		},
		{
			name:              "should do nothing if no required tasks are available",
			currentGraphTasks: []UntypedTask{optionalTaskA},
			availableTasks:    []UntypedTask{optionalTaskA},
			expectedTasks:     []string{"optional-task-a#default"},
			expectedChanged:   false,
			expectErr:         false,
		},
		{
			name:              "should add multiple required tasks",
			currentGraphTasks: []UntypedTask{},
			availableTasks:    []UntypedTask{optionalTaskA, requiredTaskB, requiredTaskC},
			expectedTasks:     []string{"required-task-b#default", "required-task-c#default"},
			expectedChanged:   true,
			expectErr:         false,
		},
		{
			name:              "should handle empty current tasks",
			currentGraphTasks: []UntypedTask{},
			availableTasks:    []UntypedTask{optionalTaskA, requiredTaskB},
			expectedTasks:     []string{"required-task-b#default"},
			expectedChanged:   true,
			expectErr:         false,
		},
		{
			name:              "should handle empty available tasks",
			currentGraphTasks: []UntypedTask{optionalTaskA},
			availableTasks:    []UntypedTask{},
			expectedTasks:     []string{"optional-task-a#default"},
			expectedChanged:   false,
			expectErr:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rule := &RequiredTaskLabelGraphResolverRule{}
			result, err := rule.Resolve(tc.currentGraphTasks, tc.availableTasks)

			if (err != nil) != tc.expectErr {
				t.Fatalf("Resolve() error = %v, expectErr %v", err, tc.expectErr)
			}
			if err != nil {
				return
			}

			if result.Changed != tc.expectedChanged {
				t.Errorf("Expected Changed to be %v, but got %v", tc.expectedChanged, result.Changed)
			}

			resultTaskIDs := make([]string, len(result.Tasks))
			for i, task := range result.Tasks {
				resultTaskIDs[i] = task.UntypedID().String()
			}

			sort.Strings(resultTaskIDs)
			sort.Strings(tc.expectedTasks)

			if len(resultTaskIDs) != len(tc.expectedTasks) {
				t.Fatalf("Expected %d tasks, but got %d. Expected: %v, Got: %v", len(tc.expectedTasks), len(resultTaskIDs), tc.expectedTasks, resultTaskIDs)
			}

			for i := range resultTaskIDs {
				if resultTaskIDs[i] != tc.expectedTasks[i] {
					t.Errorf("Task mismatch at index %d. Expected %v, but got %v", i, tc.expectedTasks[i], resultTaskIDs[i])
				}
			}
		})
	}
}

// mockTaskOptions allows configuring a mock task's dependencies and priority.
type mockTaskOptions struct {
	dependencies       []taskid.UntypedTaskReference
	priority           int
	subsequentTaskRefs []taskid.UntypedTaskReference
}

// newMockTask creates a new mockUntypedTask for testing with custom options.
func newMockTask(idStr string, implID string, opts mockTaskOptions) UntypedTask {
	id := taskid.NewImplementationID(taskid.NewTaskReference[any](idStr), implID)
	labelOpts := []LabelOpt{WithLabelValue(LabelKeyTaskSelectionPriority, opts.priority)}
	if len(opts.subsequentTaskRefs) > 0 {
		labelOpts = append(labelOpts, WithLabelValue(LabelKeySubsequentTaskRefs, opts.subsequentTaskRefs))
	}
	labels := NewLabelSet(labelOpts...)
	return &mockUntypedTask{
		id:           id,
		labels:       labels,
		dependencies: opts.dependencies,
	}
}

func TestDependencyResolverGraphResolverRule_Resolve(t *testing.T) {
	providerRef := taskid.NewTaskReference[any]("provider")
	unresolvableRef := taskid.NewTaskReference[any]("unresolvable")

	providerTask := newMockTask("provider", "default", mockTaskOptions{priority: 10})
	providerTaskLowPrio := newMockTask("provider", "low", mockTaskOptions{priority: 5})
	providerTaskHighPrio := newMockTask("provider", "high", mockTaskOptions{priority: 20})

	consumerTask := newMockTask("consumer", "default", mockTaskOptions{dependencies: []taskid.UntypedTaskReference{providerRef}})
	unresolvableConsumerTask := newMockTask("unresolvable-consumer", "default", mockTaskOptions{dependencies: []taskid.UntypedTaskReference{unresolvableRef}})

	testCases := []struct {
		name              string
		currentGraphTasks []UntypedTask
		availableTasks    []UntypedTask
		expectedTasks     []string // task implementation IDs
		expectedChanged   bool
		expectErr         bool
	}{
		{
			name:              "should resolve a simple dependency",
			currentGraphTasks: []UntypedTask{consumerTask},
			availableTasks:    []UntypedTask{providerTask, consumerTask},
			expectedTasks:     []string{"consumer#default", "provider#default"},
			expectedChanged:   true,
			expectErr:         false,
		},
		{
			name:              "should do nothing if dependency is already satisfied",
			currentGraphTasks: []UntypedTask{consumerTask, providerTask},
			availableTasks:    []UntypedTask{providerTask, consumerTask},
			expectedTasks:     []string{"consumer#default", "provider#default"},
			expectedChanged:   false,
			expectErr:         false,
		},
		{
			name:              "should return an error for unresolvable dependency",
			currentGraphTasks: []UntypedTask{unresolvableConsumerTask},
			availableTasks:    []UntypedTask{providerTask},
			expectedTasks:     nil,
			expectedChanged:   false,
			expectErr:         true,
		},
		{
			name:              "should select the highest priority task",
			currentGraphTasks: []UntypedTask{consumerTask},
			availableTasks:    []UntypedTask{consumerTask, providerTask, providerTaskLowPrio, providerTaskHighPrio},
			expectedTasks:     []string{"consumer#default", "provider#high"},
			expectedChanged:   true,
			expectErr:         false,
		},
		{
			name:              "should do nothing if there are no dependencies",
			currentGraphTasks: []UntypedTask{providerTask},
			availableTasks:    []UntypedTask{providerTask, consumerTask},
			expectedTasks:     []string{"provider#default"},
			expectedChanged:   false,
			expectErr:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rule := &TaskDependencyGraphResolverRule{}
			result, err := rule.Resolve(tc.currentGraphTasks, tc.availableTasks)

			if (err != nil) != tc.expectErr {
				t.Fatalf("Resolve() error = %v, expectErr %v", err, tc.expectErr)
			}
			if err != nil {
				return
			}

			if result.Changed != tc.expectedChanged {
				t.Errorf("Expected Changed to be %v, but got %v", tc.expectedChanged, result.Changed)
			}

			resultTaskIDs := make([]string, len(result.Tasks))
			for i, task := range result.Tasks {
				resultTaskIDs[i] = task.UntypedID().String()
			}

			sort.Strings(resultTaskIDs)
			sort.Strings(tc.expectedTasks)

			if len(resultTaskIDs) != len(tc.expectedTasks) {
				t.Fatalf("Expected %d tasks, but got %d. Expected: %v, Got: %v", len(tc.expectedTasks), len(resultTaskIDs), tc.expectedTasks, resultTaskIDs)
			}

			for i := range resultTaskIDs {
				if resultTaskIDs[i] != tc.expectedTasks[i] {
					t.Errorf("Task mismatch at index %d. Expected %v, but got %v", i, tc.expectedTasks[i], resultTaskIDs[i])
				}
			}
		})
	}
}

// mockGraphResolverRule is a mock implementation of GraphResolverRule for testing GraphResolver.
type mockGraphResolverRule struct {
	name        string
	resolveFunc func(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error)
}

func (m *mockGraphResolverRule) Name() string { return m.name }
func (m *mockGraphResolverRule) Resolve(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error) {
	return m.resolveFunc(currentGraphTasks, availableTasks)
}

func TestGraphResolver_Resolve(t *testing.T) {
	taskA := newMockTask("task-a", "default", mockTaskOptions{})
	taskB := newMockTask("task-b", "default", mockTaskOptions{})
	taskC := newMockTask("task-c", "default", mockTaskOptions{})

	testCases := []struct {
		name          string
		rules         []GraphResolverRule
		requiredTasks []UntypedTask
		maxIteration  int
		expectedTasks []string
		expectErr     bool
	}{
		{
			name: "should reach a stable state",
			rules: []GraphResolverRule{
				&mockGraphResolverRule{
					name: "add-b-once",
					resolveFunc: func(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error) {
						// Add taskB only if it's not present
						for _, task := range currentGraphTasks {
							if task.UntypedID().String() == taskB.UntypedID().String() {
								return GraphResolverRuleResult{Changed: false, Tasks: currentGraphTasks}, nil
							}
						}
						return GraphResolverRuleResult{Changed: true, Tasks: append(currentGraphTasks, taskB)}, nil
					},
				},
				&mockGraphResolverRule{
					name: "do-nothing",
					resolveFunc: func(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error) {
						return GraphResolverRuleResult{Changed: false, Tasks: currentGraphTasks}, nil
					},
				},
			},
			requiredTasks: []UntypedTask{taskA},
			maxIteration:  5,
			expectedTasks: []string{"task-a#default", "task-b#default"},
			expectErr:     false,
		},
		{
			name: "should return error if max iterations are reached",
			rules: []GraphResolverRule{
				&mockGraphResolverRule{
					name: "always-change",
					resolveFunc: func(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error) {
						// Always report change, but add a new task instance to avoid duplicates error
						newTask := newMockTask("new-task", "default", mockTaskOptions{})
						return GraphResolverRuleResult{Changed: true, Tasks: append(currentGraphTasks, newTask)}, nil
					},
				},
			},
			requiredTasks: []UntypedTask{taskA},
			maxIteration:  3,
			expectedTasks: nil,
			expectErr:     true,
		},
		{
			name: "should propagate an error from a rule",
			rules: []GraphResolverRule{
				&mockGraphResolverRule{
					name: "return-error",
					resolveFunc: func(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error) {
						return GraphResolverRuleResult{}, fmt.Errorf("internal rule error")
					},
				},
			},
			requiredTasks: []UntypedTask{taskA},
			maxIteration:  5,
			expectedTasks: nil,
			expectErr:     true,
		},
		{
			name:          "should handle no rules",
			rules:         []GraphResolverRule{},
			requiredTasks: []UntypedTask{taskA},
			maxIteration:  5,
			expectedTasks: []string{"task-a#default"},
			expectErr:     false,
		},
		{
			name: "should finish in one iteration if no changes",
			rules: []GraphResolverRule{
				&mockGraphResolverRule{
					name: "do-nothing",
					resolveFunc: func(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error) {
						return GraphResolverRuleResult{Changed: false, Tasks: currentGraphTasks}, nil
					},
				},
			},
			requiredTasks: []UntypedTask{taskA, taskC},
			maxIteration:  5,
			expectedTasks: []string{"task-a#default", "task-c#default"},
			expectErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolver := &GraphResolver{
				Rules:        tc.rules,
				MaxIteration: tc.maxIteration,
			}
			resultTasks, err := resolver.Resolve(tc.requiredTasks, nil) // availableTasks is not used by mock rules

			if (err != nil) != tc.expectErr {
				t.Fatalf("Resolve() error = %v, expectErr %v", err, tc.expectErr)
			}
			if err != nil {
				return
			}

			resultTaskIDs := make([]string, len(resultTasks))
			for i, task := range resultTasks {
				resultTaskIDs[i] = task.UntypedID().String()
			}

			sort.Strings(resultTaskIDs)
			sort.Strings(tc.expectedTasks)

			if len(resultTaskIDs) != len(tc.expectedTasks) {
				t.Fatalf("Expected %d tasks, but got %d. Expected: %v, Got: %v", len(tc.expectedTasks), len(resultTaskIDs), tc.expectedTasks, resultTaskIDs)
			}

			for i := range resultTaskIDs {
				if resultTaskIDs[i] != tc.expectedTasks[i] {
					t.Errorf("Task mismatch at index %d. Expected %v, but got %v", i, tc.expectedTasks[i], resultTaskIDs[i])
				}
			}
		})
	}
}

func TestSubsequentTaskRefsGraphResolverRule_Resolve(t *testing.T) {
	// Task and Reference Definitions
	taskBRef := taskid.NewTaskReference[any]("task-b")
	taskDRef := taskid.NewTaskReference[any]("task-d")

	taskA := newMockTask("task-a", "default", mockTaskOptions{subsequentTaskRefs: []taskid.UntypedTaskReference{taskBRef}})
	taskB := newMockTask("task-b", "default", mockTaskOptions{})
	taskC := newMockTask("task-c", "default", mockTaskOptions{subsequentTaskRefs: []taskid.UntypedTaskReference{taskBRef}})
	taskD := newMockTask("task-d", "default", mockTaskOptions{}) // This task is available but not requested initially
	taskE := newMockTask("task-e", "default", mockTaskOptions{subsequentTaskRefs: []taskid.UntypedTaskReference{taskDRef}})

	testCases := []struct {
		name                 string
		currentGraphTasks    []UntypedTask
		availableTasks       []UntypedTask
		expectedTaskIDs      []string
		expectedChanged      bool
		expectErr            bool
		dependencyValidation func(t *testing.T, tasks []UntypedTask)
	}{
		{
			name:              "should add a subsequent task",
			currentGraphTasks: []UntypedTask{taskA},
			availableTasks:    []UntypedTask{taskA, taskB},
			expectedTaskIDs:   []string{"task-a#default", "task-b#default"},
			expectedChanged:   true,
			expectErr:         false,
			dependencyValidation: func(t *testing.T, tasks []UntypedTask) {
				taskMap := tasksToMap(tasks)
				subsequentTask, ok := taskMap["task-b#default"]
				if !ok {
					t.Fatal("Subsequent task B not found in result")
				}
				assertIsWrappedAndDependsOn(t, subsequentTask, "task-a")
			},
		},
		{
			name:              "should update dependency of an existing subsequent task",
			currentGraphTasks: []UntypedTask{taskA, taskB},
			availableTasks:    []UntypedTask{taskA, taskB},
			expectedTaskIDs:   []string{"task-a#default", "task-b#default"},
			expectedChanged:   true,
			expectErr:         false,
			dependencyValidation: func(t *testing.T, tasks []UntypedTask) {
				taskMap := tasksToMap(tasks)
				subsequentTask, ok := taskMap["task-b#default"]
				if !ok {
					t.Fatal("Subsequent task B not found in result")
				}
				assertIsWrappedAndDependsOn(t, subsequentTask, "task-a")
			},
		},
		{
			name:              "should add multiple dependencies to a subsequent task",
			currentGraphTasks: []UntypedTask{taskA, taskC},
			availableTasks:    []UntypedTask{taskA, taskB, taskC},
			expectedTaskIDs:   []string{"task-a#default", "task-b#default", "task-c#default"},
			expectedChanged:   true,
			expectErr:         false,
			dependencyValidation: func(t *testing.T, tasks []UntypedTask) {
				taskMap := tasksToMap(tasks)
				subsequentTask, ok := taskMap["task-b#default"]
				if !ok {
					t.Fatal("Subsequent task B not found in result")
				}
				assertIsWrappedAndDependsOn(t, subsequentTask, "task-a", "task-c")
			},
		},
		{
			name:              "should return an error for unresolvable subsequent task",
			currentGraphTasks: []UntypedTask{taskE},
			availableTasks:    []UntypedTask{taskA, taskB, taskE}, // taskD is not available
			expectedTaskIDs:   nil,
			expectedChanged:   false,
			expectErr:         true,
		},
		{
			name:              "should do nothing if no subsequent tasks are defined",
			currentGraphTasks: []UntypedTask{taskB, taskD},
			availableTasks:    []UntypedTask{taskB, taskD},
			expectedTaskIDs:   []string{"task-b#default", "task-d#default"},
			expectedChanged:   false,
			expectErr:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rule := &SubsequentTaskRefsGraphResolverRule{}
			result, err := rule.Resolve(tc.currentGraphTasks, tc.availableTasks)

			if (err != nil) != tc.expectErr {
				t.Fatalf("Resolve() error = %v, expectErr %v", err, tc.expectErr)
			}
			if err != nil {
				return
			}

			if result.Changed != tc.expectedChanged {
				t.Errorf("Expected Changed to be %v, but got %v", tc.expectedChanged, result.Changed)
			}

			resultTaskIDs := make([]string, len(result.Tasks))
			for i, task := range result.Tasks {
				resultTaskIDs[i] = task.UntypedID().String()
			}
			sort.Strings(resultTaskIDs)
			sort.Strings(tc.expectedTaskIDs)
			if fmt.Sprintf("%v", resultTaskIDs) != fmt.Sprintf("%v", tc.expectedTaskIDs) {
				t.Errorf("Expected task IDs %v, but got %v", tc.expectedTaskIDs, resultTaskIDs)
			}

			if tc.dependencyValidation != nil {
				tc.dependencyValidation(t, result.Tasks)
			}
		})
	}
}

func tasksToMap(tasks []UntypedTask) map[string]UntypedTask {
	m := make(map[string]UntypedTask)
	for _, task := range tasks {
		m[task.UntypedID().String()] = task
	}
	return m
}

func assertIsWrappedAndDependsOn(t *testing.T, task UntypedTask, expectedDepIDs ...string) {
	t.Helper()
	wrappedTask, ok := task.(*dependencyOverridenUntypedTask)
	if !ok {
		t.Fatalf("Task %s is not a wrapped dependency-overridden task", task.UntypedID())
	}

	depsMap := make(map[string]bool)
	for _, dep := range wrappedTask.Dependencies() {
		depsMap[dep.String()] = true
	}

	if len(depsMap) != len(expectedDepIDs) {
		t.Errorf("Expected %d dependencies, but got %d for task %s", len(expectedDepIDs), len(depsMap), task.UntypedID())
	}

	for _, depID := range expectedDepIDs {
		if !depsMap[depID] {
			t.Errorf("Expected task %s to have dependency %s, but it was not found", task.UntypedID(), depID)
		}
	}
}
