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
	"fmt"
	"sort"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type testTaskDefinition struct {
	id           taskid.TaskImplementationID[any]
	dependencies []taskid.UntypedTaskReference
	labels       *typedmap.ReadonlyTypedMap
}

// Run implements Definition.
func (d *testTaskDefinition) Run(ctx context.Context) (any, error) {
	return nil, nil
}

func (d *testTaskDefinition) UntypedRun(ctx context.Context) (any, error) {
	return nil, nil
}

var _ Definition[any] = (*testTaskDefinition)(nil)

func (d *testTaskDefinition) ID() taskid.TaskImplementationID[any] {
	return d.id
}

func (d *testTaskDefinition) UntypedID() taskid.UntypedTaskImplementationID {
	return d.id
}

func (d *testTaskDefinition) Labels() *typedmap.ReadonlyTypedMap {
	return d.labels
}

// Dependencies implements KHITaskUnit.
func (d *testTaskDefinition) Dependencies() []taskid.UntypedTaskReference {
	return d.dependencies
}

// assertSortTaskGraph is a test helper that verifies the sortTaskGraph results
// match the expected task IDs, missing dependencies, etc.
func assertSortTaskGraph(t *testing.T, tasks []UntypedDefinition, expectedTaskIDs []string, expectedMissing []string, expectedRunnable bool, expectedHasCyclicDependency bool) {
	t.Helper() // Mark this as a helper function to improve test output

	// Create task set and run the sort
	taskSet := &DefinitionSet{definitions: tasks}
	result := taskSet.sortTaskGraph()

	// Compare actual vs expected runnable status
	if result.Runnable != expectedRunnable {
		t.Errorf("Expected runnable=%v, got %v", expectedRunnable, result.Runnable)
	}

	// Compare actual vs expected cyclic dependency status
	if result.HasCyclicDependency != expectedHasCyclicDependency {
		t.Errorf("Expected hasCyclicDependency=%v, got %v", expectedHasCyclicDependency, result.HasCyclicDependency)
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

func newDebugDefinition(id string, dependencies []string, labelOpt ...LabelOpt) *testTaskDefinition {
	labels := NewLabelSet(labelOpt...)
	dependencyReferenceIds := []taskid.UntypedTaskReference{}
	for _, id := range dependencies {
		dependencyReferenceIds = append(dependencyReferenceIds, taskid.NewTaskReference[any](id))
	}

	return &testTaskDefinition{
		id:           taskid.NewDefaultImplementationID[any](id),
		dependencies: dependencyReferenceIds,
		labels:       labels,
	}
}

func TestSortTaskGraphWithValidGraph(t *testing.T) {
	tasks := []UntypedDefinition{
		newDebugDefinition("foo", []string{"bar"}),
		newDebugDefinition("bar", []string{}),
		newDebugDefinition("qux", []string{"quux"}),
		newDebugDefinition("quux", []string{"foo", "bar"}),
	}

	// Expected order after topological sort
	expectedTaskIDs := []string{"bar", "foo", "quux", "qux"}

	// This graph is valid, so no missing dependencies, is runnable, and has no cycles
	assertSortTaskGraph(t, tasks, expectedTaskIDs, []string{}, true, false)
}

func TestSortTaskGraphReturnsTheStableResult(t *testing.T) {
	COUNT := 100
	for i := 0; i < COUNT; i++ {
		tasks := []UntypedDefinition{
			newDebugDefinition("foo", []string{}),
			newDebugDefinition("bar", []string{"foo"}),
			newDebugDefinition("qux", []string{"foo"}),
			newDebugDefinition("quux", []string{"foo"}),
		}

		// Expected order after topological sort
		expectedTaskIDs := []string{"foo", "bar", "quux", "qux"}

		// This graph is valid, so no missing dependencies, is runnable, and has no cycles
		assertSortTaskGraph(t, tasks, expectedTaskIDs, []string{}, true, false)
	}
}

func TestSortTaskGraphWithMissingDependency(t *testing.T) {
	tasks := []UntypedDefinition{
		newDebugDefinition("foo", []string{"bar", "missing-input2"}),
		newDebugDefinition("bar", []string{}),
		newDebugDefinition("qux", []string{"quux", "missing-input1"}),
		newDebugDefinition("quux", []string{"foo", "bar"}),
	}

	// Graph has missing dependencies, so we expect it to be not runnable
	expectedMissing := []string{"missing-input1", "missing-input2"}

	// When dependencies are missing, we don't have a sorted list of tasks
	assertSortTaskGraph(t, tasks, []string{}, expectedMissing, false, false)
}

func TestResolveGraphWithCircularDependency(t *testing.T) {
	tasks := []UntypedDefinition{
		newDebugDefinition("foo", []string{"bar", "qux"}),
		newDebugDefinition("bar", []string{}),
		newDebugDefinition("qux", []string{"quux"}),
		newDebugDefinition("quux", []string{"foo", "bar"}),
	}

	// This graph has a cycle, so we expect it to be not runnable
	// When there's a cycle, we don't have a sorted list of tasks or missing dependencies
	assertSortTaskGraph(t, tasks, []string{}, []string{}, false, true)
}

// assertResolveTask is a test helper that verifies the ResolveTask results
// match the expected task IDs and selection priorities.
func assertResolveTask(t *testing.T, tasks []UntypedDefinition, availableTasks []UntypedDefinition, expectedTaskIDs []string) {
	t.Helper() // Mark this as a helper function to improve test output

	// Create task sets
	taskSet := &DefinitionSet{definitions: tasks}
	availableSet, err := NewSet(availableTasks)
	if err != nil {
		t.Fatalf("Failed to create available task set: %v", err)
	}

	// Resolve tasks
	resolvedTaskSet, err := taskSet.ResolveTask(availableSet)
	if err != nil {
		t.Fatalf("ResolveTask failed: %v", err)
	}

	// Verify the task set is runnable
	if !resolvedTaskSet.runnable {
		t.Errorf("Expected resolved task set to be runnable, but it is not")
	}

	// Extract and verify the task IDs in the expected order
	actualTaskIDs := make([]string, 0, len(resolvedTaskSet.definitions))
	for _, task := range resolvedTaskSet.definitions {
		actualTaskIDs = append(actualTaskIDs, task.UntypedID().ReferenceIDString())
	}

	if diff := cmp.Diff(actualTaskIDs, expectedTaskIDs); diff != "" {
		t.Errorf("Task selection mismatch (-actual,+expected):\n%s", diff)
	}
}

func TestWrapGraph(t *testing.T) {
	testCases := []struct {
		ResolvedShape string
		Definitions   []UntypedDefinition
	}{
		{
			//https://dreampuf.github.io/GraphvizOnline/#digraph%20G%20%7B%0Astart%20%5Bshape%3D%22diamond%22%2Cfillcolor%3Dgray%2Cstyle%3Dfilled%5D%0Atest_init%20%5Bshape%3D%22circle%22%2Clabel%3D%22test-init%22%5D%0Atest_done%20%5Bshape%3D%22circle%22%2Clabel%3D%22test-done%22%5D%0Astart%20-%3E%20test_init%0Atest_init%20-%3E%20test_done%0A%7D
			ResolvedShape: `digraph G {
start [shape="diamond",fillcolor=gray,style=filled]
test_init_default [shape="circle",label="test-init#default"]
test_done_default [shape="circle",label="test-done#default"]
start -> test_init_default
test_init_default -> test_done_default
}`,
			Definitions: []UntypedDefinition{},
		},
		{
			//https://dreampuf.github.io/GraphvizOnline/#digraph%20G%20%7B%0Astart%20%5Bshape%3D%22diamond%22%2Cfillcolor%3Dgray%2Cstyle%3Dfilled%5D%0Atest_init%20%5Bshape%3D%22circle%22%2Clabel%3D%22test-init%22%5D%0Abar%20%5Bshape%3D%22circle%22%2Clabel%3D%22bar%22%5D%0Afoo%20%5Bshape%3D%22circle%22%2Clabel%3D%22foo%22%5D%0Aquux%20%5Bshape%3D%22circle%22%2Clabel%3D%22quux%22%5D%0Aquz%20%5Bshape%3D%22circle%22%2Clabel%3D%22quz%22%5D%0Atest_done%20%5Bshape%3D%22circle%22%2Clabel%3D%22test-done%22%5D%0Astart%20-%3E%20test_init%0Atest_init%20-%3E%20bar%0Atest_init%20-%3E%20foo%0Atest_init%20-%3E%20quux%0Atest_init%20-%3E%20quz%0Atest_init%20-%3E%20test_done%0Afoo%20-%3E%20test_done%0Abar%20-%3E%20test_done%0Aquz%20-%3E%20test_done%0Aquux%20-%3E%20test_done%0A%7D
			ResolvedShape: `digraph G {
start [shape="diamond",fillcolor=gray,style=filled]
test_init_default [shape="circle",label="test-init#default"]
bar_default [shape="circle",label="bar#default"]
foo_default [shape="circle",label="foo#default"]
quux_default [shape="circle",label="quux#default"]
quz_default [shape="circle",label="quz#default"]
test_done_default [shape="circle",label="test-done#default"]
start -> test_init_default
test_init_default -> bar_default
test_init_default -> foo_default
test_init_default -> quux_default
test_init_default -> quz_default
bar_default -> test_done_default
foo_default -> test_done_default
quux_default -> test_done_default
quz_default -> test_done_default
test_init_default -> test_done_default
}`,
			Definitions: []UntypedDefinition{
				newDebugDefinition("foo", []string{}),
				newDebugDefinition("bar", []string{}),
				newDebugDefinition("quz", []string{}),
				newDebugDefinition("quux", []string{}),
			},
		},
		{
			//https://dreampuf.github.io/GraphvizOnline/#digraph%20G%20%7B%0Astart%20%5Bshape%3D%22diamond%22%2Cfillcolor%3Dgray%2Cstyle%3Dfilled%5D%0Atest_init%20%5Bshape%3D%22circle%22%2Clabel%3D%22test-init%22%5D%0Afoo%20%5Bshape%3D%22circle%22%2Clabel%3D%22foo%22%5D%0Aquux%20%5Bshape%3D%22circle%22%2Clabel%3D%22quux%22%5D%0Aquz%20%5Bshape%3D%22circle%22%2Clabel%3D%22quz%22%5D%0Abar%20%5Bshape%3D%22circle%22%2Clabel%3D%22bar%22%5D%0Atest_done%20%5Bshape%3D%22circle%22%2Clabel%3D%22test-done%22%5D%0Astart%20-%3E%20test_init%0Atest_init%20-%3E%20foo%0Atest_init%20-%3E%20quux%0Atest_init%20-%3E%20quz%0Afoo%20-%3E%20bar%0Aquz%20-%3E%20bar%0Atest_init%20-%3E%20test_done%0Abar%20-%3E%20test_done%0Aquux%20-%3E%20test_done%0A%7D
			ResolvedShape: `digraph G {
start [shape="diamond",fillcolor=gray,style=filled]
test_init_default [shape="circle",label="test-init#default"]
foo_default [shape="circle",label="foo#default"]
quux_default [shape="circle",label="quux#default"]
quz_default [shape="circle",label="quz#default"]
bar_default [shape="circle",label="bar#default"]
test_done_default [shape="circle",label="test-done#default"]
start -> test_init_default
test_init_default -> foo_default
test_init_default -> quux_default
test_init_default -> quz_default
foo_default -> bar_default
quz_default -> bar_default
bar_default -> test_done_default
quux_default -> test_done_default
test_init_default -> test_done_default
}`,
			Definitions: []UntypedDefinition{
				newDebugDefinition("foo", []string{}),
				newDebugDefinition("bar", []string{"foo", "quz"}),
				newDebugDefinition("quz", []string{}),
				newDebugDefinition("quux", []string{}),
			},
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%d", i), func(t *testing.T) {
			originalSet, err := NewSet(testCase.Definitions)
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
			wrapped, err := originalSet.WrapGraph(taskid.NewDefaultImplementationID[any]("test"), []taskid.UntypedTaskReference{})
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}

			runnableSet, err := wrapped.ResolveTask(wrapped)
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}

			graphviz, err := runnableSet.DumpGraphviz()
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}

			if graphviz != testCase.ResolvedShape {
				t.Errorf("the resolved task shape is not matching with the expected shape\nExpected\n%s\n\nActual\n%s", testCase.ResolvedShape, graphviz)
			}
		})
		t.Run(fmt.Sprintf("testcase-%d-stable-check", i), func(t *testing.T) {
			COUNT := 0
			var prev *DefinitionSet
			for i := 0; i < COUNT; i++ {
				originalSet, err := NewSet(testCase.Definitions)
				if err != nil {
					t.Errorf("unexpected error %v", err)
				}
				wrapped, err := originalSet.WrapGraph(taskid.NewDefaultImplementationID[any]("test"), []taskid.UntypedTaskReference{})
				if err != nil {
					t.Errorf("unexpected error %v", err)
				}
				runnableSet, err := wrapped.ResolveTask(wrapped)
				if err != nil {
					t.Errorf("unexpected error %v", err)
				}
				if prev == nil {
					prev = runnableSet
				} else {
					if diff := cmp.Diff(prev, runnableSet); diff != "" {
						t.Errorf("the result is not same with the previous result. WrapGraph returns unstable result\n%s", diff)
					}
				}
			}
		})
	}
}

func TestResolveTaskWithValidTaskSet(t *testing.T) {
	tasks := []UntypedDefinition{
		newDebugDefinition("foo", []string{"bar"}),
		newDebugDefinition("bar", []string{"qux"}),
	}

	availableTasks := []UntypedDefinition{
		newDebugDefinition("qux", []string{"quux"}),
		newDebugDefinition("quux", []string{}),
		newDebugDefinition("hoge", []string{"fuga"}),
	}

	// Expected resolved tasks in topological order
	expectedTaskIDs := []string{"quux", "qux", "bar", "foo"}

	assertResolveTask(t, tasks, availableTasks, expectedTaskIDs)
}

func TestDumpGraphviz(t *testing.T) {
	featureTasks := []UntypedDefinition{
		newDebugDefinition("foo", []string{"bar"}),
		newDebugDefinition("bar", []string{"qux", "quux"}),
	}
	featureTaskSet := DefinitionSet{definitions: featureTasks, runnable: false}
	availableTasks := []UntypedDefinition{
		newDebugDefinition("qux", []string{}),
		newDebugDefinition("quux", []string{}),
		newDebugDefinition("hoge", []string{"fuga"}),
	}
	availableTaskSet := DefinitionSet{definitions: availableTasks, runnable: false}

	resolvedTaskSet, err := featureTaskSet.ResolveTask(&availableTaskSet)
	if err != nil {
		t.Errorf("unexpected err:%s", err.Error())
	}

	expected := `digraph G {
start [shape="diamond",fillcolor=gray,style=filled]
quux_default [shape="circle",label="quux#default"]
qux_default [shape="circle",label="qux#default"]
bar_default [shape="circle",label="bar#default"]
foo_default [shape="circle",label="foo#default"]
start -> quux_default
start -> qux_default
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
		featureTasks := []UntypedDefinition{
			newDebugDefinition("foo", []string{"qux", "quux", "hoge"}),
		}
		featureTaskSet := DefinitionSet{definitions: featureTasks, runnable: false}
		availableTasks := []UntypedDefinition{
			newDebugDefinition("qux", []string{}),
			newDebugDefinition("quux", []string{}),
			newDebugDefinition("hoge", []string{"fuga"}),
			newDebugDefinition("fuga", []string{}),
		}
		availableTaskSet := DefinitionSet{definitions: availableTasks, runnable: false}

		resolvedTaskSet, err := featureTaskSet.ResolveTask(&availableTaskSet)
		if err != nil {
			t.Errorf("unexpected err:%s", err.Error())
			break
		}

		expected := `digraph G {
start [shape="diamond",fillcolor=gray,style=filled]
fuga_default [shape="circle",label="fuga#default"]
hoge_default [shape="circle",label="hoge#default"]
quux_default [shape="circle",label="quux#default"]
qux_default [shape="circle",label="qux#default"]
foo_default [shape="circle",label="foo#default"]
start -> fuga_default
start -> quux_default
start -> qux_default
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
	ds, err := NewSet([]UntypedDefinition{})
	if err != nil {
		t.Errorf("unexpected err:%s", err)
	}

	err = ds.Add(newDebugDefinition("bar", []string{"qux", "quux"}))
	if err != nil {
		t.Errorf("unexpected err:%s", err)
	}

	// Add a task with same ID
	err = ds.Add(newDebugDefinition("bar", []string{"qux2", "quux2"}))
	if err == nil {
		t.Errorf("expected error, but returned no error")
	}
}

func TestNewSetWithDuplicatedID(t *testing.T) {
	_, err := NewSet([]UntypedDefinition{
		newDebugDefinition("bar", []string{"qux", "quux"}),
		newDebugDefinition("bar", []string{"qux", "quux"}),
	})
	if err == nil {
		t.Errorf("expected error, but returned no error")
	}
}
