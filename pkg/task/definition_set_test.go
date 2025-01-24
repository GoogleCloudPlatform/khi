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
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/exp/slices"
)

type testTaskDefinition struct {
	id           taskid.TaskImplementationId
	dependencies []taskid.TaskReferenceId
	labels       *LabelSet
	runnable     Runnable
}

var _ Definition = (*testTaskDefinition)(nil)

func (d *testTaskDefinition) ID() taskid.TaskImplementationId {
	return d.id
}

func (d *testTaskDefinition) Labels() *LabelSet {
	return d.labels
}

// Runnable implements KHITaskUnit.
func (d *testTaskDefinition) Runnable(taskMode int) Runnable {
	return d.runnable
}

// Dependencies implements KHITaskUnit.
func (d *testTaskDefinition) Dependencies() []taskid.TaskReferenceId {
	return d.dependencies
}

func (d *testTaskDefinition) WithTestLabel(labels ...string) *testTaskDefinition {
	d.labels.Set("test-label", labels)
	return d
}

func (d *testTaskDefinition) WithThreadUnsafeLabel() *testTaskDefinition {
	d.labels.Set(LabelKeyThreadUnsafe, true)
	return d
}

func (d *testTaskDefinition) WithRunnable(runnable Runnable) *testTaskDefinition {
	d.runnable = runnable
	return d
}

func newDebugDefinition(id string, dependencies []string, labelOpt ...LabelOpt) *testTaskDefinition {
	labels := NewLabelSet()
	dependencyReferenceIds := []taskid.TaskReferenceId{}
	for _, id := range dependencies {
		dependencyReferenceIds = append(dependencyReferenceIds, taskid.NewTaskReference(id))
	}
	for _, opt := range labelOpt {
		opt.Write(labels)
	}

	return &testTaskDefinition{
		id:           taskid.NewTaskImplementationId(id),
		dependencies: dependencyReferenceIds,
		labels:       labels,
	}
}
func TestSortTaskGraphWithValidGraph(t *testing.T) {
	tasks := []Definition{
		newDebugDefinition("foo", []string{"bar"}),
		newDebugDefinition("bar", []string{}),
		newDebugDefinition("qux", []string{"quux"}),
		newDebugDefinition("quux", []string{"foo", "bar"}),
	}
	taskSet := &DefinitionSet{definitions: tasks}

	resolveResult := taskSet.sortTaskGraph()

	if diff := cmp.Diff(resolveResult, &SortTaskGraphResult{
		TopologicalSortedTasks: []Definition{
			newDebugDefinition("bar", []string{}),
			newDebugDefinition("foo", []string{"bar"}),
			newDebugDefinition("quux", []string{"foo", "bar"}),
			newDebugDefinition("qux", []string{"quux"}),
		},
		MissingDependencies: []taskid.TaskReferenceId{},
		Runnable:            true,
		HasCyclicDependency: false,
	}, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, taskid.TaskImplementationId{}, taskid.TaskReferenceId{})); diff != "" {
		t.Errorf("Generated SortTaskGraphResult is mismatched, (-actual,+expected);\n%s", diff)
	}
}

func TestSortTaskGraphReturnsTheStableResult(t *testing.T) {
	COUNT := 100
	for i := 0; i < COUNT; i++ {
		tasks := []Definition{
			newDebugDefinition("foo", []string{}),
			newDebugDefinition("bar", []string{"foo"}),
			newDebugDefinition("qux", []string{"foo"}),
			newDebugDefinition("quux", []string{"foo"}),
		}
		taskSet := &DefinitionSet{definitions: tasks}

		resolveResult := taskSet.sortTaskGraph()

		if diff := cmp.Diff(resolveResult, &SortTaskGraphResult{
			TopologicalSortedTasks: []Definition{
				newDebugDefinition("foo", []string{}),
				newDebugDefinition("bar", []string{"foo"}),
				newDebugDefinition("quux", []string{"foo"}),
				newDebugDefinition("qux", []string{"foo"}),
			},
			MissingDependencies: []taskid.TaskReferenceId{},
			Runnable:            true,
			HasCyclicDependency: false,
		}, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, taskid.TaskImplementationId{}, taskid.TaskReferenceId{})); diff != "" {
			t.Errorf("Generated SortTaskGraphResult is mismatched, (-actual,+expected);\n%s", diff)
			break
		}
	}
}

func TestSortTaskGraphWithMissingDependency(t *testing.T) {
	tasks := []Definition{
		newDebugDefinition("foo", []string{"bar", "missing-input2"}),
		newDebugDefinition("bar", []string{}),
		newDebugDefinition("qux", []string{"quux", "missing-input1"}),
		newDebugDefinition("quux", []string{"foo", "bar"}),
	}
	taskSet := &DefinitionSet{definitions: tasks}

	resolveResult := taskSet.sortTaskGraph()

	if diff := cmp.Diff(resolveResult, &SortTaskGraphResult{
		TopologicalSortedTasks: nil,
		MissingDependencies:    []taskid.TaskReferenceId{taskid.NewTaskReference("missing-input1"), taskid.NewTaskReference("missing-input2")},
		Runnable:               false,
		HasCyclicDependency:    false,
	}, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, taskid.TaskReferenceId{}), cmpopts.SortSlices(func(a, b taskid.TaskReferenceId) bool {
		return strings.Compare(a.String(), b.String()) > 0
	})); diff != "" {
		t.Errorf("Generated SortTaskGraphResult is mismatched, (-actual,+expected);\n%s", diff)
	}
}

func TestResolveGraphShouldIgnoreAfterSharp(t *testing.T) {
	tasks := []Definition{
		newDebugDefinition("foo#suffix", []string{"bar"}),
		newDebugDefinition("bar", []string{}),
		newDebugDefinition("qux", []string{"quux"}),
		newDebugDefinition("quux", []string{"foo", "bar"}),
	}
	taskSet := &DefinitionSet{definitions: tasks}

	resolveResult := taskSet.sortTaskGraph()

	if diff := cmp.Diff(resolveResult, &SortTaskGraphResult{
		TopologicalSortedTasks: []Definition{
			newDebugDefinition("bar", []string{}),
			newDebugDefinition("foo#suffix", []string{"bar"}),
			newDebugDefinition("quux", []string{"foo", "bar"}),
			newDebugDefinition("qux", []string{"quux"}),
		},
		MissingDependencies: []taskid.TaskReferenceId{},
		Runnable:            true,
		HasCyclicDependency: false,
	}, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, taskid.TaskImplementationId{}, taskid.TaskReferenceId{})); diff != "" {
		t.Errorf("Generated SortTaskGraphResult is mismatched, (-actual,+expected);\n%s", diff)
	}
}

func TestResolveGraphWithCircularDependency(t *testing.T) {
	tasks := []Definition{
		newDebugDefinition("foo", []string{"bar", "qux"}),
		newDebugDefinition("bar", []string{}),
		newDebugDefinition("qux", []string{"quux"}),
		newDebugDefinition("quux", []string{"foo", "bar"}),
	}
	taskSet := &DefinitionSet{definitions: tasks}

	resolveResult := taskSet.sortTaskGraph()

	if diff := cmp.Diff(resolveResult, &SortTaskGraphResult{
		TopologicalSortedTasks: nil,
		MissingDependencies:    []taskid.TaskReferenceId{},
		Runnable:               false,
		HasCyclicDependency:    true,
	}, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, taskid.TaskReferenceId{}), cmpopts.SortSlices(func(a string, b string) bool {
		return strings.Compare(a, b) > 0
	})); diff != "" {
		t.Errorf("Generated SortTaskGraphResult is mismatched, (-actual,+expected);\n%s", diff)
	}
}

func TestResolveTaskWithSelectionPriority(t *testing.T) {
	tasks := []Definition{
		newDebugDefinition("foo", []string{"bar"}),
	}
	taskSet := &DefinitionSet{definitions: tasks}

	availableSet, err := NewSet([]Definition{
		newDebugDefinition("bar#a", []string{}, WithSelectionPriority(10)),
		newDebugDefinition("bar#b", []string{}, WithSelectionPriority(20)),
		newDebugDefinition("bar#c", []string{}),
	})
	if err != nil {
		t.Errorf("unexpected error\n%v", err)
	}
	resolveResult, err := taskSet.ResolveTask(availableSet)
	if err != nil {
		t.Errorf("unexpected error\n%v", err)
	}

	expectedSet, err := NewSet([]Definition{
		newDebugDefinition("bar#b", []string{}, WithSelectionPriority(20)),
		newDebugDefinition("foo", []string{"bar"}),
	})
	expectedSet.runnable = true
	if err != nil {
		t.Errorf("unexpected error\n%v", err)
	}

	if diff := cmp.Diff(resolveResult, expectedSet, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, taskid.TaskReferenceId{}, taskid.TaskImplementationId{}, DefinitionSet{}), cmpopts.SortSlices(func(a string, b string) bool {
		return strings.Compare(a, b) > 0
	})); diff != "" {
		t.Errorf("Generated DefinitionSet is mismatched, (-actual,+expected);\n%s", diff)
	}
}

func TestSubsetFilter(t *testing.T) {
	tasks := []Definition{
		newDebugDefinition("foo", []string{"bar"}).WithTestLabel("foo"),
		newDebugDefinition("bar", []string{}).WithTestLabel("foo"),
		newDebugDefinition("qux", []string{"quux"}).WithTestLabel("bar"),
		newDebugDefinition("quux", []string{"foo", "bar"}).WithTestLabel("foo"),
	}
	taskSet := &DefinitionSet{definitions: tasks}
	hasPredict := func(v any) bool {
		return slices.Contains(v.([]string), "foo")
	}

	filteredTaskSet := taskSet.FilteredSubset("test-label", hasPredict, false)
	if diff := cmp.Diff(taskSet, filteredTaskSet, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, DefinitionSet{}, taskid.TaskImplementationId{}, taskid.TaskReferenceId{})); diff == "" {
		t.Errorf("FilteredSubset must return another instance of KHITaskUnitSet")
	}
	if diff := cmp.Diff(filteredTaskSet, &DefinitionSet{
		definitions: []Definition{
			newDebugDefinition("foo", []string{"bar"}).WithTestLabel("foo"),
			newDebugDefinition("bar", []string{}).WithTestLabel("foo"),
			newDebugDefinition("quux", []string{"foo", "bar"}).WithTestLabel("foo"),
		},
	}, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, DefinitionSet{}, taskid.TaskImplementationId{}, taskid.TaskReferenceId{})); diff != "" {
		t.Errorf("The result of FilteredSubset is mismatched, (-actual,+expected);\n%s", diff)
	}
}

func TestSubsetFilterIncludeUndefined(t *testing.T) {
	tasks := []Definition{
		newDebugDefinition("foo", []string{"bar"}),
		newDebugDefinition("bar", []string{}),
		newDebugDefinition("qux", []string{"quux"}).WithTestLabel("bar"),
		newDebugDefinition("quux", []string{"foo", "bar"}),
	}
	taskSet := &DefinitionSet{definitions: tasks}
	hasPredict := func(v any) bool {
		return slices.Contains(v.([]string), "foo")
	}

	filteredTaskSet := taskSet.FilteredSubset("test-label", hasPredict, true)
	if diff := cmp.Diff(taskSet, filteredTaskSet, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, DefinitionSet{}, taskid.TaskImplementationId{}, taskid.TaskReferenceId{})); diff == "" {
		t.Errorf("FilteredSubset must return another instance of KHITaskUnitSet")
	}
	if diff := cmp.Diff(filteredTaskSet, &DefinitionSet{
		definitions: []Definition{
			newDebugDefinition("foo", []string{"bar"}),
			newDebugDefinition("bar", []string{}),
			newDebugDefinition("quux", []string{"foo", "bar"}),
		},
	}, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, DefinitionSet{}, taskid.TaskImplementationId{}, taskid.TaskReferenceId{})); diff != "" {
		t.Errorf("The result of FilteredSubset is mismatched, (-actual,+expected);\n%s", diff)
	}
}

func TestWrapGraph(t *testing.T) {
	testCases := []struct {
		ResolvedShape string
		Definitions   []Definition
	}{
		{
			//https://dreampuf.github.io/GraphvizOnline/#digraph%20G%20%7B%0Astart%20%5Bshape%3D%22diamond%22%2Cfillcolor%3Dgray%2Cstyle%3Dfilled%5D%0Atest_init%20%5Bshape%3D%22circle%22%2Clabel%3D%22test-init%22%5D%0Atest_done%20%5Bshape%3D%22circle%22%2Clabel%3D%22test-done%22%5D%0Astart%20-%3E%20test_init%0Atest_init%20-%3E%20test_done%0A%7D
			ResolvedShape: `digraph G {
start [shape="diamond",fillcolor=gray,style=filled]
test_init [shape="circle",label="test-init"]
test_done [shape="circle",label="test-done"]
start -> test_init
test_init -> test_done
}`,
			Definitions: []Definition{},
		},
		{
			//https://dreampuf.github.io/GraphvizOnline/#digraph%20G%20%7B%0Astart%20%5Bshape%3D%22diamond%22%2Cfillcolor%3Dgray%2Cstyle%3Dfilled%5D%0Atest_init%20%5Bshape%3D%22circle%22%2Clabel%3D%22test-init%22%5D%0Abar%20%5Bshape%3D%22circle%22%2Clabel%3D%22bar%22%5D%0Afoo%20%5Bshape%3D%22circle%22%2Clabel%3D%22foo%22%5D%0Aquux%20%5Bshape%3D%22circle%22%2Clabel%3D%22quux%22%5D%0Aquz%20%5Bshape%3D%22circle%22%2Clabel%3D%22quz%22%5D%0Atest_done%20%5Bshape%3D%22circle%22%2Clabel%3D%22test-done%22%5D%0Astart%20-%3E%20test_init%0Atest_init%20-%3E%20bar%0Atest_init%20-%3E%20foo%0Atest_init%20-%3E%20quux%0Atest_init%20-%3E%20quz%0Atest_init%20-%3E%20test_done%0Afoo%20-%3E%20test_done%0Abar%20-%3E%20test_done%0Aquz%20-%3E%20test_done%0Aquux%20-%3E%20test_done%0A%7D
			ResolvedShape: `digraph G {
start [shape="diamond",fillcolor=gray,style=filled]
test_init [shape="circle",label="test-init"]
bar [shape="circle",label="bar"]
foo [shape="circle",label="foo"]
quux [shape="circle",label="quux"]
quz [shape="circle",label="quz"]
test_done [shape="circle",label="test-done"]
start -> test_init
test_init -> bar
test_init -> foo
test_init -> quux
test_init -> quz
bar -> test_done
foo -> test_done
quux -> test_done
quz -> test_done
test_init -> test_done
}`,
			Definitions: []Definition{
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
test_init [shape="circle",label="test-init"]
foo [shape="circle",label="foo"]
quux [shape="circle",label="quux"]
quz [shape="circle",label="quz"]
bar [shape="circle",label="bar"]
test_done [shape="circle",label="test-done"]
start -> test_init
test_init -> foo
test_init -> quux
test_init -> quz
foo -> bar
quz -> bar
bar -> test_done
quux -> test_done
test_init -> test_done
}`,
			Definitions: []Definition{
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
			wrapped, err := originalSet.WrapGraph(taskid.NewTaskImplementationId("test"), []taskid.TaskReferenceId{})
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
				wrapped, err := originalSet.WrapGraph(taskid.NewTaskImplementationId("test"), []taskid.TaskReferenceId{})
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
	featureTasks := []Definition{
		newDebugDefinition("foo", []string{"bar"}),
		newDebugDefinition("bar", []string{"qux"}),
	}
	featureTaskSet := DefinitionSet{definitions: featureTasks, runnable: false}
	availableTasks := []Definition{
		newDebugDefinition("qux", []string{"quux"}),
		newDebugDefinition("quux", []string{}),
		newDebugDefinition("hoge", []string{"fuga"}),
	}
	availableTaskSet := DefinitionSet{definitions: availableTasks, runnable: false}

	resolvedTaskSet, err := featureTaskSet.ResolveTask(&availableTaskSet)

	if err != nil {
		t.Errorf("unexpected err:%s", err.Error())
	}
	if diff := cmp.Diff(resolvedTaskSet, &DefinitionSet{
		runnable: true,
		definitions: []Definition{
			newDebugDefinition("quux", []string{}),
			newDebugDefinition("qux", []string{"quux"}),
			newDebugDefinition("bar", []string{"qux"}),
			newDebugDefinition("foo", []string{"bar"}),
		},
	}, cmp.AllowUnexported(testTaskDefinition{}, LabelSet{}, DefinitionSet{}, taskid.TaskImplementationId{}, taskid.TaskReferenceId{})); diff != "" {
		t.Errorf("The result of ResolveTask is mismatched, (-actual,+expected);\n%s", diff)
	}
}

func TestDumpGraphviz(t *testing.T) {
	featureTasks := []Definition{
		newDebugDefinition("foo", []string{"bar"}),
		newDebugDefinition("bar", []string{"qux", "quux"}),
	}
	featureTaskSet := DefinitionSet{definitions: featureTasks, runnable: false}
	availableTasks := []Definition{
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
quux [shape="circle",label="quux"]
qux [shape="circle",label="qux"]
bar [shape="circle",label="bar"]
foo [shape="circle",label="foo"]
start -> quux
start -> qux
qux -> bar
quux -> bar
bar -> foo
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
		featureTasks := []Definition{
			newDebugDefinition("foo", []string{"qux", "quux", "hoge"}),
		}
		featureTaskSet := DefinitionSet{definitions: featureTasks, runnable: false}
		availableTasks := []Definition{
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
fuga [shape="circle",label="fuga"]
hoge [shape="circle",label="hoge"]
quux [shape="circle",label="quux"]
qux [shape="circle",label="qux"]
foo [shape="circle",label="foo"]
start -> fuga
start -> quux
start -> qux
fuga -> hoge
qux -> foo
quux -> foo
hoge -> foo
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
	ds, err := NewSet([]Definition{})
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
	_, err := NewSet([]Definition{
		newDebugDefinition("bar", []string{"qux", "quux"}),
		newDebugDefinition("bar", []string{"qux", "quux"}),
	})
	if err == nil {
		t.Errorf("expected error, but returned no error")
	}
}
