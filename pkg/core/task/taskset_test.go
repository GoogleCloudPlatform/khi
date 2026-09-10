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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/google/go-cmp/cmp"
)

type testTask struct {
	id           taskid.TaskImplementationID[any]
	dependencies []Dependency
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
func (d *testTask) Dependencies() []Dependency {
	return d.dependencies
}

func newDebugTask(id string, dependencies []string, labelOpt ...LabelOpt) *testTask {
	labels := NewLabelSet(labelOpt...)
	deps := make([]Dependency, 0, len(dependencies))
	for _, depID := range dependencies {
		deps = append(deps, taskid.NewTaskReference[any](depID))
	}

	return &testTask{
		id:           taskid.NewDefaultImplementationID[any](id),
		dependencies: deps,
		labels:       labels,
	}
}

func TestNewTaskSet(t *testing.T) {
	testCases := []struct {
		name       string
		tasks      []UntypedTask
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "unique tasks succeed",
			tasks: []UntypedTask{
				newDebugTask("foo", nil),
				newDebugTask("bar", nil),
			},
			wantErr: false,
		},
		{
			name: "duplicate task IDs return error",
			tasks: []UntypedTask{
				newDebugTask("foo", nil),
				newDebugTask("foo", nil),
			},
			wantErr:    true,
			wantErrMsg: "multiple tasks have the same ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			set, err := NewTaskSet(tc.tasks)
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewTaskSet() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("expected error message containing '%s', got '%v'", tc.wantErrMsg, err)
				}
				return
			}
			if len(set.GetAll()) != len(tc.tasks) {
				t.Errorf("expected %d tasks in set, got %d", len(tc.tasks), len(set.GetAll()))
			}
		})
	}
}

func TestTaskSet_AddAndRemove(t *testing.T) {
	testCases := []struct {
		name        string
		initial     []UntypedTask
		op          func(s *TaskSet) error
		wantTaskIDs []string
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name: "add new task succeeds",
			initial: []UntypedTask{
				newDebugTask("foo", nil),
			},
			op: func(s *TaskSet) error {
				return s.Add(newDebugTask("bar", nil))
			},
			wantTaskIDs: []string{"foo#default", "bar#default"},
			wantErr:     false,
		},
		{
			name: "add duplicate task fails",
			initial: []UntypedTask{
				newDebugTask("foo", nil),
			},
			op: func(s *TaskSet) error {
				return s.Add(newDebugTask("foo", nil))
			},
			wantTaskIDs: []string{"foo#default"},
			wantErr:     true,
			wantErrMsg:  "is duplicated",
		},
		{
			name: "remove existing task succeeds",
			initial: []UntypedTask{
				newDebugTask("foo", nil),
				newDebugTask("bar", nil),
			},
			op: func(s *TaskSet) error {
				return s.Remove("foo#default")
			},
			wantTaskIDs: []string{"bar#default"},
			wantErr:     false,
		},
		{
			name: "remove non-existent task fails",
			initial: []UntypedTask{
				newDebugTask("foo", nil),
			},
			op: func(s *TaskSet) error {
				return s.Remove("bar#default")
			},
			wantTaskIDs: []string{"foo#default"},
			wantErr:     true,
			wantErrMsg:  "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewTaskSet(tc.initial)
			if err != nil {
				t.Fatalf("failed to create task set: %v", err)
			}

			opErr := tc.op(s)
			if (opErr != nil) != tc.wantErr {
				t.Fatalf("operation error = %v, wantErr = %v", opErr, tc.wantErr)
			}
			if tc.wantErr {
				if !strings.Contains(opErr.Error(), tc.wantErrMsg) {
					t.Errorf("expected error message containing '%s', got '%v'", tc.wantErrMsg, opErr)
				}
			}

			gotIDs := make([]string, 0, len(s.GetAll()))
			for _, task := range s.GetAll() {
				gotIDs = append(gotIDs, task.UntypedID().String())
			}
			if diff := cmp.Diff(tc.wantTaskIDs, gotIDs); diff != "" {
				t.Errorf("task IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskSet_Get(t *testing.T) {
	testCases := []struct {
		name       string
		taskID     string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "get existing task succeeds",
			taskID:  "foo#default",
			wantErr: false,
		},
		{
			name:       "get non-existing task returns error",
			taskID:     "missing#default",
			wantErr:    true,
			wantErrMsg: "was not found",
		},
	}

	s, err := NewTaskSet([]UntypedTask{newDebugTask("foo", nil)})
	if err != nil {
		t.Fatalf("failed to create task set: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task, err := s.Get(tc.taskID)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Get() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("expected error message containing '%s', got '%v'", tc.wantErrMsg, err)
				}
				return
			}
			if task.UntypedID().String() != tc.taskID {
				t.Errorf("expected task ID %s, got %s", tc.taskID, task.UntypedID())
			}
		})
	}
}

func TestTaskSet_MetadataQueries(t *testing.T) {
	taskA := newDebugTask("taskA", nil)
	taskB := newDebugTask("taskB", nil)
	taskC := newDebugTask("taskC", nil)
	tasks := []UntypedTask{taskA, taskB, taskC}

	edges := []taskid.TaskEdge{
		{
			SourceRefID:  "taskA",
			SourceImplID: "taskA#default",
			TargetImplID: "taskB#default",
		},
		{
			SourceRefID:  "taskC",
			SourceImplID: "taskC#default",
			TargetImplID: "taskB#default",
		},
	}

	boundFanInRefIDsByTaskImplID := map[string]map[string][]string{
		"taskB#default": {
			"tag-sample": {"taskA"},
		},
	}

	resolved := NewResolvedTaskSet(tasks, edges, boundFanInRefIDsByTaskImplID)

	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "IncomingEdges returns all edges targeting task",
			test: func(t *testing.T) {
				got := resolved.IncomingEdges("taskB#default")
				if len(got) != 2 {
					t.Errorf("expected 2 incoming edges, got %d", len(got))
				}
			},
		},
		{
			name: "IsBound returns true for bound tasks and false for unbound",
			test: func(t *testing.T) {
				if !resolved.IsBound("taskA") {
					t.Errorf("expected taskA to be bound")
				}
				if !resolved.IsBound("taskB") {
					t.Errorf("expected taskB to be bound")
				}
				if resolved.IsBound("unbound") {
					t.Errorf("expected unbound task to not be bound")
				}
			},
		},
		{
			name: "BoundReferenceIDsForTaskImplWithTag returns task-specific bound ref IDs",
			test: func(t *testing.T) {
				got := resolved.BoundReferenceIDsForTaskImplWithTag("taskB#default", "tag-sample")
				want := []string{"taskA"}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("BoundReferenceIDsForTaskImplWithTag mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff([]string(nil), resolved.BoundReferenceIDsForTaskImplWithTag("taskUnknown#default", "tag-sample")); diff != "" {
					t.Errorf("BoundReferenceIDsForTaskImplWithTag mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff([]string(nil), resolved.BoundReferenceIDsForTaskImplWithTag("taskB#default", "non-existent-tag")); diff != "" {
					t.Errorf("BoundReferenceIDsForTaskImplWithTag mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "Edges returns copy of all edges",
			test: func(t *testing.T) {
				got := resolved.Edges()
				if len(got) != 2 {
					t.Errorf("expected 2 edges, got %d", len(got))
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func TestTaskSet_DumpGraphviz(t *testing.T) {
	testCases := []struct {
		name       string
		tasks      []UntypedTask
		wantDOT    string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid DAG produces expected Graphviz DOT",
			tasks: []UntypedTask{
				newDebugTask("foo", []string{"bar"}),
				newDebugTask("bar", []string{"qux", "quux"}),
				newDebugTask("qux", []string{}),
				newDebugTask("quux", []string{}),
			},
			wantDOT: `digraph G {
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
}`,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolvedTaskSet, err := ResolveGraph(tc.tasks, tc.tasks, nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ResolveGraph() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			graphViz, err := resolvedTaskSet.DumpGraphviz()
			if err != nil {
				t.Fatalf("DumpGraphviz() error = %v", err)
			}
			if diff := cmp.Diff(tc.wantDOT, graphViz); diff != "" {
				t.Errorf("DumpGraphviz() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskSet_DumpGraphvizNonRunnable(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
	}{
		{
			name:       "unresolved task set returns error for DumpGraphviz",
			wantErrMsg: "can't draw a graph for non runnable graph",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			unresolved, err := NewTaskSet([]UntypedTask{newDebugTask("foo", nil)})
			if err != nil {
				t.Fatalf("failed to create task set: %v", err)
			}

			_, err = unresolved.DumpGraphviz()
			if err == nil {
				t.Fatal("expected error for non-runnable graph, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Errorf("expected error message containing '%s', got '%v'", tc.wantErrMsg, err)
			}
		})
	}
}

func TestTaskSet_DumpGraphvizReturnsStableResult(t *testing.T) {
	COUNT := 100
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

	featureTasks := []UntypedTask{
		newDebugTask("foo", []string{"qux", "quux", "hoge"}),
		newDebugTask("qux", []string{}),
		newDebugTask("quux", []string{}),
		newDebugTask("hoge", []string{"fuga"}),
		newDebugTask("fuga", []string{}),
	}

	for i := 0; i < COUNT; i++ {
		resolvedTaskSet, err := ResolveGraph(featureTasks, featureTasks, nil)
		if err != nil {
			t.Fatalf("iteration %d: ResolveGraph() error = %v", i, err)
		}

		graphViz, err := resolvedTaskSet.DumpGraphviz()
		if err != nil {
			t.Fatalf("iteration %d: DumpGraphviz() error = %v", i, err)
		}
		if diff := cmp.Diff(expected, graphViz); diff != "" {
			t.Fatalf("iteration %d: DumpGraphviz() mismatch (-want +got):\n%s", i, diff)
		}
	}
}
