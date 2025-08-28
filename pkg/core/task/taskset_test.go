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
	"sort"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/google/go-cmp/cmp"
)

type testTask struct {
	id           taskid.TaskImplementationID[any]
	dependencies []taskid.UntypedTaskReference
	labels       *typedmap.ReadonlyTypedMap
}

// Run implements Task.
func (d *testTask) Run(ctx context.Context) (any, error) {
	return nil, nil
}

func (d *testTask) UntypedRun(ctx context.Context) (any, error) {
	return nil, nil
}

var _ Task[any] = (*testTask)(nil)

func (d *testTask) ID() taskid.TaskImplementationID[any] {
	return d.id
}

func (d *testTask) UntypedID() taskid.UntypedTaskImplementationID {
	return d.id
}

func (d *testTask) Labels() *typedmap.ReadonlyTypedMap {
	return d.labels
}

// Dependencies implements KHITaskUnit.
func (d *testTask) Dependencies() []taskid.UntypedTaskReference {
	return d.dependencies
}

// assertSortTaskGraph is a test helper that verifies the sortTaskGraph results
// match the expected task IDs, missing dependencies, etc.
func assertSortTaskGraph(t *testing.T, tasks []UntypedTask, expectedTaskIDs []string, expectedMissing []string, expectedRunnable bool, expectedCyclicDependencyPath string) {
	t.Helper() // Mark this as a helper function to improve test output

	// Create task set and run the sort
	taskSet := &TaskSet{tasks: tasks}
	result := taskSet.sortTaskGraph()

	// Compare actual vs expected runnable status
	if result.Runnable != expectedRunnable {
		t.Errorf("Expected runnable=%v, got %v", expectedRunnable, result.Runnable)
	}

	// Compare actual vs expected cyclic dependency status
	if result.CyclicDependencyPath != expectedCyclicDependencyPath {
		t.Errorf("Expected cyclicDependencyPath=%v, got %v", expectedCyclicDependencyPath, result.CyclicDependencyPath)
	}

	// If not runnable and expected not runnable with specific reasons, check missing dependencies
	if !expectedRunnable {
		// Check missing dependencies match expected
		actualMissing := make([]string, 0, len(result.MissingDependencies))
		for _, dep := range result.MissingDependencies {
			actualMissing = append(actualMissing, dep.ReferenceIDString())
		}

		// Sort both slices to ensure consistent comparison
		sort.Strings(actualMissing)
		sort.Strings(expectedMissing)

		if diff := cmp.Diff(actualMissing, expectedMissing); diff != "" {
			t.Errorf("Missing dependencies mismatch (-actual,+expected):\n%s", diff)
		}
		return
	}

	// If expected runnable, check task IDs in the expected order
	if len(result.TopologicalSortedTasks) != len(expectedTaskIDs) {
		t.Errorf("Expected %d tasks, got %d", len(expectedTaskIDs), len(result.TopologicalSortedTasks))
		return
	}

	actualTaskIDs := make([]string, 0, len(result.TopologicalSortedTasks))
	for _, task := range result.TopologicalSortedTasks {
		actualTaskIDs = append(actualTaskIDs, task.UntypedID().ReferenceIDString())
	}

	if diff := cmp.Diff(actualTaskIDs, expectedTaskIDs); diff != "" {
		t.Errorf("Task ordering mismatch (-actual,+expected):\n%s", diff)
	}
}

func newDebugTask(id string, dependencies []string, labelOpt ...LabelOpt) *testTask {
	labels := NewLabelSet(labelOpt...)
	dependencyReferenceIds := []taskid.UntypedTaskReference{}
	for _, id := range dependencies {
		dependencyReferenceIds = append(dependencyReferenceIds, taskid.NewTaskReference[any](id))
	}

	return &testTask{
		id:           taskid.NewDefaultImplementationID[any](id),
		dependencies: dependencyReferenceIds,
		labels:       labels,
	}
}

func TestSortTaskGraphWithValidGraph(t *testing.T) {
	tasks := []UntypedTask{
		newDebugTask("foo", []string{"bar"}),
		newDebugTask("bar", []string{}),
		newDebugTask("qux", []string{"quux"}),
		newDebugTask("quux", []string{"foo", "bar"}),
	}

	// Expected order after topological sort
	expectedTaskIDs := []string{"bar", "foo", "quux", "qux"}

	// This graph is valid, so no missing dependencies, is runnable, and has no cycles
	assertSortTaskGraph(t, tasks, expectedTaskIDs, []string{}, true, "")
}

func TestSortTaskGraphReturnsTheStableResult(t *testing.T) {
	COUNT := 100
	for i := 0; i < COUNT; i++ {
		tasks := []UntypedTask{
			newDebugTask("foo", []string{}),
			newDebugTask("bar", []string{"foo"}),
			newDebugTask("qux", []string{"foo"}),
			newDebugTask("quux", []string{"foo"}),
		}

		// Expected order after topological sort
		expectedTaskIDs := []string{"foo", "qux", "quux", "bar"}

		// This graph is valid, so no missing dependencies, is runnable, and has no cycles
		assertSortTaskGraph(t, tasks, expectedTaskIDs, []string{}, true, "")
	}
}

func TestSortTaskGraphWithMissingDependency(t *testing.T) {
	tasks := []UntypedTask{
		newDebugTask("foo", []string{"bar", "missing-input2"}),
		newDebugTask("bar", []string{}),
		newDebugTask("qux", []string{"quux", "missing-input1"}),
		newDebugTask("quux", []string{"foo", "bar"}),
	}

	// Graph has missing dependencies, so we expect it to be not runnable
	expectedMissing := []string{"missing-input1", "missing-input2"}

	// When dependencies are missing, we don't have a sorted list of tasks
	assertSortTaskGraph(t, tasks, []string{}, expectedMissing, false, "")
}

func TestResolveGraphWithCircularDependency(t *testing.T) {
	tasks := []UntypedTask{
		newDebugTask("foo", []string{"bar", "qux"}),
		newDebugTask("bar", []string{}),
		newDebugTask("qux", []string{"quux"}),
		newDebugTask("quux", []string{"foo", "bar"}),
	}
	for i := 0; i < 100; i++ { // to check the stability
		// This graph has a cycle, so we expect it to be not runnable
		// When there's a cycle, we don't have a sorted list of tasks or missing dependencies
		assertSortTaskGraph(t, tasks, []string{}, []string{}, false, "... -> foo#default] -> [quux#default -> qux#default -> foo#default] -> [quux#default -> ...")
	}
}

func TestDumpGraphviz(t *testing.T) {
	inputTasks := []UntypedTask{
		newDebugTask("foo", []string{"bar"}),
		newDebugTask("bar", []string{"qux", "quux"}),
		newDebugTask("qux", []string{}),
		newDebugTask("quux", []string{}),
	}
	ts, err := NewTaskSet(inputTasks)
	if err != nil {
		t.Fatalf("unexpected err:%s", err.Error())
	}
	resolvedTaskSet, err := ts.ToRunnableTaskSet()
	if err != nil {
		t.Errorf("unexpected err:%s", err.Error())
	}

	expected := `digraph G {
start [shape="diamond",fillcolor=gray,style=filled]
qux_default [shape="circle",label="qux#default"]
quux_default [shape="circle",label="quux#default"]
bar_default [shape="circle",label="bar#default"]
foo_default [shape="circle",label="foo#default"]
start -> qux_default
start -> quux_default
qux_default -> bar_default
quux_default -> bar_default
bar_default -> foo_default
}`
	graphViz, err := resolvedTaskSet.DumpGraphviz()
	if err != nil {
		t.Errorf("unexpected err:%s", err.Error())
	}
	if diff := cmp.Diff(graphViz, expected); diff != "" {
		t.Errorf("generated graph is not matching with the expected result\n%s", diff)
	}
}

func TestDumpGraphvizReturnsStableResult(t *testing.T) {
	COUNT := 100
	for i := 0; i < COUNT; i++ {
		featureTasks := []UntypedTask{
			newDebugTask("foo", []string{"qux", "quux", "hoge"}),
			newDebugTask("qux", []string{}),
			newDebugTask("quux", []string{}),
			newDebugTask("hoge", []string{"fuga"}),
			newDebugTask("fuga", []string{}),
		}
		ts, err := NewTaskSet(featureTasks)
		if err != nil {
			t.Fatalf("unexpected err:%s", err.Error())
		}
		resolvedTaskSet, err := ts.ToRunnableTaskSet()
		if err != nil {
			t.Errorf("unexpected err:%s", err.Error())
			break
		}

		expected := `digraph G {
start [shape="diamond",fillcolor=gray,style=filled]
qux_default [shape="circle",label="qux#default"]
quux_default [shape="circle",label="quux#default"]
fuga_default [shape="circle",label="fuga#default"]
hoge_default [shape="circle",label="hoge#default"]
foo_default [shape="circle",label="foo#default"]
start -> qux_default
start -> quux_default
start -> fuga_default
fuga_default -> hoge_default
qux_default -> foo_default
quux_default -> foo_default
hoge_default -> foo_default
}`
		graphViz, err := resolvedTaskSet.DumpGraphviz()
		if err != nil {
			t.Errorf("unexpected err:%s", err.Error())
			break
		}
		if diff := cmp.Diff(graphViz, expected); diff != "" {
			t.Errorf("generated graph is not matching with the expected result at %d\n%s", i, diff)
			break
		}
	}
}

func TestAddDefinitionToSet(t *testing.T) {
	ds, err := NewTaskSet([]UntypedTask{})
	if err != nil {
		t.Errorf("unexpected err:%s", err)
	}

	err = ds.Add(newDebugTask("bar", []string{"qux", "quux"}))
	if err != nil {
		t.Errorf("unexpected err:%s", err)
	}

	// Add a task with same ID
	err = ds.Add(newDebugTask("bar", []string{"qux2", "quux2"}))
	if err == nil {
		t.Errorf("expected error, but returned no error")
	}
}

func TestRemoveDefinitionFromSet(t *testing.T) {
	ds, err := NewTaskSet([]UntypedTask{
		newDebugTask("bar", []string{"qux", "quux"}),
		newDebugTask("foo", []string{"qux", "quux"}),
	})
	if err != nil {
		t.Errorf("unexpected err:%s", err)
	}

	err = ds.Remove("bar#default")
	if err != nil {
		t.Errorf("unexpected err:%s", err)
	}

	// Remove a task with non-existent ID
	err = ds.Remove("bar#default")
	if err == nil {
		t.Errorf("expected error, but returned no error")
	}
}

func TestNewSetWithDuplicatedID(t *testing.T) {
	_, err := NewTaskSet([]UntypedTask{
		newDebugTask("bar", []string{"qux", "quux"}),
		newDebugTask("bar", []string{"qux", "quux"}),
	})
	if err == nil {
		t.Errorf("expected error, but returned no error")
	}
}
