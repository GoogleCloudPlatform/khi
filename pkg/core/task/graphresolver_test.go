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

type mockUntypedTask struct {
	id           taskid.UntypedTaskImplementationID
	labels       *typedmap.ReadonlyTypedMap
	dependencies []Dependency
}

func (m *mockUntypedTask) UntypedID() taskid.UntypedTaskImplementationID { return m.id }
func (m *mockUntypedTask) Labels() *typedmap.ReadonlyTypedMap            { return m.labels }
func (m *mockUntypedTask) Dependencies() []Dependency                    { return m.dependencies }
func (m *mockUntypedTask) UntypedRun(ctx context.Context) (any, error)   { return nil, nil }

var _ UntypedTask = (*mockUntypedTask)(nil)

func createMockTask(refID, implHash string, deps []Dependency, labelOpts ...LabelOpt) UntypedTask {
	id := taskid.NewImplementationID(taskid.NewTaskReference[any](refID), implHash)
	return &mockUntypedTask{
		id:           id,
		labels:       NewLabelSet(labelOpts...),
		dependencies: deps,
	}
}

func TestResolveGraph_MandatoryClosure(t *testing.T) {
	testCases := []struct {
		name           string
		initialTasks   []UntypedTask
		availableTasks []UntypedTask
		wantTaskIDs    []string
		wantErr        bool
	}{
		{
			name: "single task without dependencies",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			wantTaskIDs: []string{"task-a#default"},
			wantErr:     false,
		},
		{
			name: "transitive mandatory dependencies are resolved",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", []Dependency{
					taskid.NewTaskReference[any]("task-b"),
				}),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", []Dependency{
					taskid.NewTaskReference[any]("task-b"),
				}),
				createMockTask("task-b", "default", []Dependency{
					taskid.NewTaskReference[any]("task-c"),
				}),
				createMockTask("task-c", "default", nil),
			},
			wantTaskIDs: []string{"task-c#default", "task-b#default", "task-a#default"},
			wantErr:     false,
		},
		{
			name: "required task label automatically pulls task into graph",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
				createMockTask("required-task", "default", nil, WithLabelValue(LabelKeyRequiredTask, true)),
			},
			wantTaskIDs: []string{"required-task#default", "task-a#default"},
			wantErr:     false,
		},
		{
			name: "missing mandatory dependency returns error",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", []Dependency{
					taskid.NewTaskReference[any]("non-existent"),
				}),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", []Dependency{
					taskid.NewTaskReference[any]("non-existent"),
				}),
			},
			wantTaskIDs: nil,
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskSet, err := ResolveGraph(tc.initialTasks, tc.availableTasks, nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ResolveGraph() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			var gotTaskIDs []string
			for _, task := range taskSet.GetAll() {
				gotTaskIDs = append(gotTaskIDs, task.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantTaskIDs, gotTaskIDs); diff != "" {
				t.Errorf("ResolveGraph() task IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveGraph_OptionalDependencies(t *testing.T) {
	tagA := NewTag[any]("tag-a")

	testCases := []struct {
		name           string
		initialTasks   []UntypedTask
		availableTasks []UntypedTask
		wantTaskIDs    []string
		wantEdgeCount  int
	}{
		{
			name: "optional dependency is activated when target exists in graph",
			initialTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					taskid.NewTaskReference[any]("provider", taskid.ScopeActiveGraph),
				}),
				createMockTask("provider", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					taskid.NewTaskReference[any]("provider", taskid.ScopeActiveGraph),
				}),
				createMockTask("provider", "default", nil),
			},
			wantTaskIDs:   []string{"provider#default", "consumer#default"},
			wantEdgeCount: 1,
		},
		{
			name: "optional dependency is NOT activated when target is not in graph",
			initialTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					taskid.NewTaskReference[any]("provider", taskid.ScopeActiveGraph),
				}),
			},
			availableTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					taskid.NewTaskReference[any]("provider", taskid.ScopeActiveGraph),
				}),
				createMockTask("provider", "default", nil),
			},
			wantTaskIDs:   []string{"consumer#default"},
			wantEdgeCount: 0,
		},
		{
			name: "fan-in with ScopeActiveGraph only includes existing tasks",
			initialTasks: []UntypedTask{
				createMockTask("collector", "default", []Dependency{
					tagA.Ref(taskid.ScopeActiveGraph),
				}),
				createMockTask("prod1", "default", nil, ProvidesTag(tagA)),
			},
			availableTasks: []UntypedTask{
				createMockTask("collector", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("prod1", "default", nil, ProvidesTag(tagA)),
				createMockTask("prod2", "default", nil, ProvidesTag(tagA)), // not in graph
			},
			wantTaskIDs:   []string{"prod1#default", "collector#default"},
			wantEdgeCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskSet, err := ResolveGraph(tc.initialTasks, tc.availableTasks, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var gotTaskIDs []string
			for _, task := range taskSet.GetAll() {
				gotTaskIDs = append(gotTaskIDs, task.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantTaskIDs, gotTaskIDs); diff != "" {
				t.Errorf("ResolveGraph() task IDs mismatch (-want +got):\n%s", diff)
			}
			if len(taskSet.Edges()) != tc.wantEdgeCount {
				t.Errorf("got %d edges, want %d", len(taskSet.Edges()), tc.wantEdgeCount)
			}
		})
	}
}

func TestResolveGraph_FanInScopeActiveFeatures(t *testing.T) {
	tagDiscovery := NewTag[any]("discovery")

	testCases := []struct {
		name        string
		setup       func() (initialTasks, availableTasks, disabledTasks []UntypedTask)
		wantTaskIDs []string
		wantErr     bool
	}{
		{
			name: "anchor check includes anchored provider but excludes unselected feature provider",
			setup: func() ([]UntypedTask, []UntypedTask, []UntypedTask) {
				commonAncestor := createMockTask("common-ancestor", "default", nil)
				prod1 := createMockTask("prod1", "default", []Dependency{
					taskid.NewTaskReference[any]("common-ancestor"),
				}, ProvidesTag(tagDiscovery))
				unselectedFeature := createMockTask("unselected-feature", "default", nil)
				prod2 := createMockTask("prod2", "default", []Dependency{
					taskid.NewTaskReference[any]("unselected-feature"),
				}, ProvidesTag(tagDiscovery))
				collector := createMockTask("collector", "default", []Dependency{
					tagDiscovery.Ref(FromActiveFeatures),
				})
				return []UntypedTask{collector, commonAncestor},
					[]UntypedTask{collector, commonAncestor, prod1, unselectedFeature, prod2},
					[]UntypedTask{unselectedFeature}
			},
			wantTaskIDs: []string{"common-ancestor#default", "prod1#default", "collector#default"},
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initial, available, disabled := tc.setup()
			taskSet, err := ResolveGraph(initial, available, disabled)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ResolveGraph() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			var gotTaskIDs []string
			for _, task := range taskSet.GetAll() {
				gotTaskIDs = append(gotTaskIDs, task.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantTaskIDs, gotTaskIDs); diff != "" {
				t.Errorf("ResolveGraph() task IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveGraph_PointToPointScopeActiveFeatures(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func() (initialTasks, availableTasks, disabledTasks []UntypedTask)
		wantTaskIDs []string
		wantEdges   []taskid.TaskEdge
		wantErr     bool
	}{
		{
			name: "pulls in target task and upstream dependencies when upstream merges into active graph",
			setup: func() ([]UntypedTask, []UntypedTask, []UntypedTask) {
				commonAncestor := createMockTask("common-ancestor", "default", nil)
				targetTask := createMockTask("target-task", "default", []Dependency{
					taskid.NewTaskReference[any]("common-ancestor"),
				})
				collector := createMockTask("collector", "default", []Dependency{
					taskid.NewTaskReference[any]("target-task", taskid.ScopeActiveFeatures),
				})
				return []UntypedTask{collector, commonAncestor},
					[]UntypedTask{collector, commonAncestor, targetTask},
					nil
			},
			wantTaskIDs: []string{"common-ancestor#default", "target-task#default", "collector#default"},
			wantEdges: []taskid.TaskEdge{
				{
					SourceRefID:  "target-task",
					SourceImplID: "target-task#default",
					TargetImplID: "collector#default",
					Cardinality:  taskid.CardinalityPointToPoint,
				},
				{
					SourceRefID:  "common-ancestor",
					SourceImplID: "common-ancestor#default",
					TargetImplID: "target-task#default",
					Cardinality:  taskid.CardinalityPointToPoint,
				},
			},
			wantErr: false,
		},
		{
			name: "excludes target task when upstream prerequisite is disabled or outside active graph",
			setup: func() ([]UntypedTask, []UntypedTask, []UntypedTask) {
				unselectedFeature := createMockTask("unselected-feature", "default", nil)
				targetTask := createMockTask("target-task", "default", []Dependency{
					taskid.NewTaskReference[any]("unselected-feature"),
				})
				collector := createMockTask("collector", "default", []Dependency{
					taskid.NewTaskReference[any]("target-task", taskid.ScopeActiveFeatures),
				})
				return []UntypedTask{collector},
					[]UntypedTask{collector, unselectedFeature, targetTask},
					[]UntypedTask{unselectedFeature}
			},
			wantTaskIDs: []string{"collector#default"},
			wantEdges:   nil,
			wantErr:     false,
		},
		{
			name: "chains multi-hop point-to-point dependencies with ScopeActiveFeatures",
			setup: func() ([]UntypedTask, []UntypedTask, []UntypedTask) {
				commonAncestor := createMockTask("common-ancestor", "default", nil)
				intermediateTask := createMockTask("intermediate-task", "default", []Dependency{
					taskid.NewTaskReference[any]("common-ancestor"),
				})
				targetTask := createMockTask("target-task", "default", []Dependency{
					taskid.NewTaskReference[any]("intermediate-task"),
				})
				collector := createMockTask("collector", "default", []Dependency{
					taskid.NewTaskReference[any]("target-task", taskid.ScopeActiveFeatures),
				})
				return []UntypedTask{collector, commonAncestor},
					[]UntypedTask{collector, commonAncestor, intermediateTask, targetTask},
					nil
			},
			wantTaskIDs: []string{"common-ancestor#default", "intermediate-task#default", "target-task#default", "collector#default"},
			wantEdges: []taskid.TaskEdge{
				{
					SourceRefID:  "target-task",
					SourceImplID: "target-task#default",
					TargetImplID: "collector#default",
					Cardinality:  taskid.CardinalityPointToPoint,
				},
				{
					SourceRefID:  "common-ancestor",
					SourceImplID: "common-ancestor#default",
					TargetImplID: "intermediate-task#default",
					Cardinality:  taskid.CardinalityPointToPoint,
				},
				{
					SourceRefID:  "intermediate-task",
					SourceImplID: "intermediate-task#default",
					TargetImplID: "target-task#default",
					Cardinality:  taskid.CardinalityPointToPoint,
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initial, available, disabled := tc.setup()
			taskSet, err := ResolveGraph(initial, available, disabled)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ResolveGraph() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			var gotTaskIDs []string
			for _, task := range taskSet.GetAll() {
				gotTaskIDs = append(gotTaskIDs, task.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantTaskIDs, gotTaskIDs); diff != "" {
				t.Errorf("ResolveGraph() task IDs mismatch (-want +got):\n%s", diff)
			}

			if tc.wantEdges != nil {
				if diff := cmp.Diff(tc.wantEdges, taskSet.Edges()); diff != "" {
					t.Errorf("ResolveGraph() edges mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestResolveGraph_EdgeDeduplication(t *testing.T) {
	tagA := NewTag[any]("tag-a")

	testCases := []struct {
		name         string
		initialTasks []UntypedTask
		wantEdges    []taskid.TaskEdge
	}{
		{
			name: "point-to-point dependency and fan-in dependency deduplicate with tag preserved",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil, ProvidesTag(tagA)),
				createMockTask("task-b", "default", []Dependency{
					taskid.NewTaskReference[any]("task-a"),
					tagA.Ref(),
				}),
			},
			wantEdges: []taskid.TaskEdge{
				{
					SourceRefID:  "task-a",
					SourceImplID: "task-a#default",
					TargetImplID: "task-b#default",
					Cardinality:  taskid.CardinalityPointToPoint,
					Tag:          "tag-a",
					Priority:     100,
				},
			},
		},
		{
			name: "multiple dependencies deduplicate with priority merging",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil, ProvidesTag(tagA, WithTagPriority(20))),
				createMockTask("task-b", "default", []Dependency{
					taskid.NewTaskReference[any]("task-a", taskid.ScopeActiveGraph),
					tagA.Ref(),
				}),
			},
			wantEdges: []taskid.TaskEdge{
				{
					SourceRefID:  "task-a",
					SourceImplID: "task-a#default",
					TargetImplID: "task-b#default",
					Cardinality:  taskid.CardinalityPointToPoint,
					Tag:          "tag-a",
					Priority:     20,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskSet, err := ResolveGraph(tc.initialTasks, tc.initialTasks, nil)
			if err != nil {
				t.Fatalf("ResolveGraph() unexpected error = %v", err)
			}

			edges := taskSet.Edges()
			if diff := cmp.Diff(tc.wantEdges, edges); diff != "" {
				t.Errorf("Edges() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveGraph_CyclicDependency(t *testing.T) {
	testCases := []struct {
		name         string
		initialTasks []UntypedTask
		wantErrMsg   string
	}{
		{
			name: "three-task cycle returns cyclic dependency error",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", []Dependency{
					taskid.NewTaskReference[any]("task-b"),
				}),
				createMockTask("task-b", "default", []Dependency{
					taskid.NewTaskReference[any]("task-c"),
				}),
				createMockTask("task-c", "default", []Dependency{
					taskid.NewTaskReference[any]("task-a"),
				}),
			},
			wantErrMsg: "cyclic dependency",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveGraph(tc.initialTasks, tc.initialTasks, nil)
			if err == nil {
				t.Fatal("expected error for cyclic dependency, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantErrMsg)
			}
		})
	}
}

func TestResolveGraph_DisabledTasks(t *testing.T) {
	tagA := NewTag[any]("tag-a")

	testCases := []struct {
		name           string
		initialTasks   []UntypedTask
		availableTasks []UntypedTask
		disabledTasks  []UntypedTask
		wantTaskIDs    []string
		wantErr        bool
	}{
		{
			name: "initial task is disabled returns error",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			disabledTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			wantErr: true,
		},
		{
			name: "mandatory dependency of initial task is disabled returns error",
			initialTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					taskid.NewTaskReference[any]("provider"),
				}),
			},
			availableTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					taskid.NewTaskReference[any]("provider"),
				}),
				createMockTask("provider", "default", nil),
			},
			disabledTasks: []UntypedTask{
				createMockTask("provider", "default", nil),
			},
			wantErr: true,
		},
		{
			name: "required task is disabled returns error",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
				createMockTask("task-required", "default", nil, NewRequiredTaskLabel()),
			},
			disabledTasks: []UntypedTask{
				createMockTask("task-required", "default", nil),
			},
			wantErr: true,
		},
		{
			name: "initial task has variant in initial tasks returns error",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "v1", nil),
				createMockTask("task-a", "v2", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "v1", nil),
				createMockTask("task-a", "v2", nil),
			},
			disabledTasks: nil,
			wantErr:       true,
		},
		{
			name: "initial task has variant in available tasks returns error",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
				createMockTask("task-a", "custom", nil),
			},
			disabledTasks: nil,
			wantErr:       true,
		},
		{
			name: "optional dependency is disabled is safely ignored",
			initialTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					taskid.NewTaskReference[any]("provider", taskid.ScopeActiveGraph),
				}),
			},
			availableTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					taskid.NewTaskReference[any]("provider", taskid.ScopeActiveGraph),
				}),
				createMockTask("provider", "default", nil),
			},
			disabledTasks: []UntypedTask{
				createMockTask("provider", "default", nil),
			},
			wantTaskIDs: []string{"consumer#default"},
			wantErr:     false,
		},
		{
			name: "fan-in resolves multiple producers providing same tag with different reference IDs",
			initialTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(FromActiveFeatures),
				}),
			},
			availableTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(FromActiveFeatures),
				}),
				createMockTask("provider-a", "default", nil, ProvidesTag(tagA)),
				createMockTask("provider-b", "default", nil, ProvidesTag(tagA)),
			},
			disabledTasks: nil,
			wantTaskIDs:   []string{"provider-a#default", "provider-b#default", "consumer#default"},
			wantErr:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskSet, err := ResolveGraph(tc.initialTasks, tc.availableTasks, tc.disabledTasks)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ResolveGraph() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			var gotTaskIDs []string
			for _, task := range taskSet.GetAll() {
				gotTaskIDs = append(gotTaskIDs, task.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantTaskIDs, gotTaskIDs); diff != "" {
				t.Errorf("ResolveGraph() task IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type mockUnknownScopeFanIn struct {
	tag string
}

var _ taskid.FanInDescriptor = (*mockUnknownScopeFanIn)(nil)

func (m mockUnknownScopeFanIn) DescriptorCardinality() taskid.EdgeCardinality {
	return taskid.CardinalityFanIn
}

func (m mockUnknownScopeFanIn) DescriptorScope() taskid.DependencyScope {
	return taskid.DependencyScope(999)
}

func (m mockUnknownScopeFanIn) Tag() string {
	return m.tag
}

func TestResolveGraph_UnknownScope(t *testing.T) {
	testCases := []struct {
		name       string
		task       UntypedTask
		wantErrMsg string
	}{
		{
			name: "unknown DependencyScope returns error",
			task: createMockTask("collector", "default", []Dependency{
				mockUnknownScopeFanIn{tag: "some-tag"},
			}),
			wantErrMsg: "unsupported dependency scope",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveGraph([]UntypedTask{tc.task}, []UntypedTask{tc.task}, nil)
			if err == nil {
				t.Fatal("expected error for unknown DependencyScope, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantErrMsg)
			}
		})
	}
}

func TestResolveGraph_InputValidation(t *testing.T) {
	testCases := []struct {
		name           string
		initialTasks   []UntypedTask
		availableTasks []UntypedTask
		disabledTasks  []UntypedTask
		wantErrMsg     string
		wantErr        bool
	}{
		{
			name: "available tasks has duplicate reference ID (variant)",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "v1", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "v1", nil),
				createMockTask("task-a", "v2", nil),
			},
			disabledTasks: nil,
			wantErr:       true,
			wantErrMsg:    "conflicting implementation",
		},
		{
			name: "initial task is not in available tasks",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-b", "default", nil),
			},
			disabledTasks: nil,
			wantErr:       true,
			wantErrMsg:    "not in available tasks",
		},
		{
			name: "initial task implementation does not match available task implementation",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "custom", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			disabledTasks: nil,
			wantErr:       true,
			wantErrMsg:    "conflicting implementation",
		},
		{
			name: "initial tasks has duplicate reference ID",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			disabledTasks: nil,
			wantErr:       true,
			wantErrMsg:    "duplicate reference",
		},
		{
			name: "disabled task is not in available tasks",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			disabledTasks: []UntypedTask{
				createMockTask("task-b", "default", nil),
			},
			wantErr:    true,
			wantErrMsg: "not in available tasks",
		},
		{
			name: "disabled task implementation does not match available task implementation",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
				createMockTask("task-b", "default", nil),
			},
			disabledTasks: []UntypedTask{
				createMockTask("task-b", "custom", nil),
			},
			wantErr:    true,
			wantErrMsg: "conflicting implementation",
		},
		{
			name: "disabled tasks has duplicate reference ID",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
				createMockTask("task-b", "default", nil),
			},
			disabledTasks: []UntypedTask{
				createMockTask("task-b", "default", nil),
				createMockTask("task-b", "default", nil),
			},
			wantErr:    true,
			wantErrMsg: "duplicate reference",
		},
		{
			name: "initial task is in disabled tasks",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			disabledTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			wantErr:    true,
			wantErrMsg: "explicitly disabled",
		},
		{
			name: "valid inputs succeed without error",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", nil),
				createMockTask("task-b", "default", nil),
			},
			disabledTasks: []UntypedTask{
				createMockTask("task-b", "default", nil),
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveGraph(tc.initialTasks, tc.availableTasks, tc.disabledTasks)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ResolveGraph() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr && tc.wantErrMsg != "" {
				if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("ResolveGraph() error = %q, want substring %q", err.Error(), tc.wantErrMsg)
				}
			}
		})
	}
}
