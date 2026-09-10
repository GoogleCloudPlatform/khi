// Copyright 2026 Google LLC
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
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/google/go-cmp/cmp"
)

func TestResolveGraph_FanInCycle_PriorityDifference(t *testing.T) {
	tagA := NewTag[any]("tag-a")

	testCases := []struct {
		name                        string
		initialTasks                []UntypedTask
		availableTasks              []UntypedTask
		wantTaskIDs                 []string
		wantBoundRefIDsByTaskImplID map[string][]string
	}{
		// Mermaid task graph:
		// ```mermaid
		// graph TD
		//   prod_high["prod-high (priority=10)"] -->|"FanIn (tag-a)"| consumer["consumer"]
		//   consumer -->|"PointToPoint"| prod_low["prod-low (priority=100)"]
		//   prod_low -.->|"FanIn (tag-a, cycle pruned)"| consumer
		// ```
		{
			name: "higher priority fan-in edge is accepted and cyclic lower priority edge is pruned",
			initialTasks: []UntypedTask{
				createMockTask("prod-high", "default", nil, ProvidesTag(tagA, WithTagPriority(10))),
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("prod-low", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			availableTasks: []UntypedTask{
				createMockTask("prod-high", "default", nil, ProvidesTag(tagA, WithTagPriority(10))),
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("prod-low", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			wantTaskIDs: []string{"prod-high#default", "consumer#default", "prod-low#default"},
			wantBoundRefIDsByTaskImplID: map[string][]string{
				"consumer#default": {"prod-high"},
			},
		},
		// Mermaid task graph:
		// ```mermaid
		// graph TD
		//   prod_high["prod-high (priority=10)"] -->|"FanIn (tag-a)"| consumer["consumer"]
		//   consumer -.->|"FanIn (tag-a, self-loop pruned)"| consumer
		// ```
		{
			name: "higher priority external producer is accepted and lower priority self-loop edge is pruned",
			initialTasks: []UntypedTask{
				createMockTask("prod-high", "default", nil, ProvidesTag(tagA, WithTagPriority(10))),
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			availableTasks: []UntypedTask{
				createMockTask("prod-high", "default", nil, ProvidesTag(tagA, WithTagPriority(10))),
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			wantTaskIDs: []string{"prod-high#default", "consumer#default"},
			wantBoundRefIDsByTaskImplID: map[string][]string{
				"consumer#default": {"prod-high"},
			},
		},
		// Mermaid task graph:
		// ```mermaid
		// graph TD
		//   prod_high["prod-high (priority=10)"] -->|"FanIn (tag-a)"| consumer["consumer"]
		//   consumer -->|"PointToPoint"| task_mid["task-mid"]
		//   task_mid -->|"PointToPoint"| prod_low["prod-low (priority=100)"]
		//   prod_low -.->|"FanIn (tag-a, cycle pruned)"| consumer
		// ```
		{
			name: "multi-hop transitive cycle with lower priority producer is pruned",
			initialTasks: []UntypedTask{
				createMockTask("prod-high", "default", nil, ProvidesTag(tagA, WithTagPriority(10))),
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("task-mid", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}),
				createMockTask("prod-low", "default", []Dependency{
					taskid.NewTaskReference[any]("task-mid"),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			availableTasks: []UntypedTask{
				createMockTask("prod-high", "default", nil, ProvidesTag(tagA, WithTagPriority(10))),
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("task-mid", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}),
				createMockTask("prod-low", "default", []Dependency{
					taskid.NewTaskReference[any]("task-mid"),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			wantTaskIDs: []string{"prod-high#default", "consumer#default", "task-mid#default", "prod-low#default"},
			wantBoundRefIDsByTaskImplID: map[string][]string{
				"consumer#default": {"prod-high"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskSet, err := ResolveGraph(tc.initialTasks, tc.availableTasks, nil)
			if err != nil {
				t.Fatalf("ResolveGraph() unexpected error = %v", err)
			}

			var gotTaskIDs []string
			for _, task := range taskSet.GetAll() {
				gotTaskIDs = append(gotTaskIDs, task.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantTaskIDs, gotTaskIDs); diff != "" {
				t.Errorf("ResolveGraph() task IDs mismatch (-want +got):\n%s", diff)
			}

			for taskImplID, wantRefs := range tc.wantBoundRefIDsByTaskImplID {
				gotRefs := taskSet.BoundReferenceIDsForTaskImplWithTag(taskImplID, tagA.ID())
				if diff := cmp.Diff(wantRefs, gotRefs); diff != "" {
					t.Errorf("BoundReferenceIDsForTaskImplWithTag(%q) mismatch (-want +got):\n%s", taskImplID, diff)
				}
			}
		})
	}
}

func TestResolveGraph_FanInCycle_EqualPriority(t *testing.T) {
	tagA := NewTag[any]("tag-a")

	testCases := []struct {
		name                        string
		initialTasks                []UntypedTask
		availableTasks              []UntypedTask
		wantTaskIDs                 []string
		wantBoundRefIDsByTaskImplID map[string][]string
	}{
		// Mermaid task graph:
		// ```mermaid
		// graph TD
		//   prod_a["prod-a (priority=100)"] -->|"FanIn (tag-a)"| consumer["consumer"]
		//   consumer -->|"PointToPoint"| prod_b["prod-b (priority=100)"]
		//   prod_b -.->|"FanIn (tag-a, cycle pruned)"| consumer
		// ```
		{
			name: "equal priority cyclic producer is pruned while acyclic producer is accepted",
			initialTasks: []UntypedTask{
				createMockTask("prod-a", "default", nil, ProvidesTag(tagA, WithTagPriority(100))),
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("prod-b", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			availableTasks: []UntypedTask{
				createMockTask("prod-a", "default", nil, ProvidesTag(tagA, WithTagPriority(100))),
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("prod-b", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			wantTaskIDs: []string{"prod-a#default", "consumer#default", "prod-b#default"},
			wantBoundRefIDsByTaskImplID: map[string][]string{
				"consumer#default": {"prod-a"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskSet, err := ResolveGraph(tc.initialTasks, tc.availableTasks, nil)
			if err != nil {
				t.Fatalf("ResolveGraph() unexpected error = %v", err)
			}

			var gotTaskIDs []string
			for _, task := range taskSet.GetAll() {
				gotTaskIDs = append(gotTaskIDs, task.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantTaskIDs, gotTaskIDs); diff != "" {
				t.Errorf("ResolveGraph() task IDs mismatch (-want +got):\n%s", diff)
			}

			for taskImplID, wantRefs := range tc.wantBoundRefIDsByTaskImplID {
				gotRefs := taskSet.BoundReferenceIDsForTaskImplWithTag(taskImplID, tagA.ID())
				if diff := cmp.Diff(wantRefs, gotRefs); diff != "" {
					t.Errorf("BoundReferenceIDsForTaskImplWithTag(%q) mismatch (-want +got):\n%s", taskImplID, diff)
				}
			}
		})
	}
}

func TestResolveGraph_FanInCycle_Pruning(t *testing.T) {
	tagA := NewTag[any]("tag-a")

	testCases := []struct {
		name                        string
		initialTasks                []UntypedTask
		availableTasks              []UntypedTask
		wantTaskIDs                 []string
		wantBoundRefIDsByTaskImplID map[string][]string
	}{
		// Mermaid task graph:
		// ```mermaid
		// graph TD
		//   consumer["consumer"] -->|"PointToPoint"| prod_low["prod-low (priority=100)"]
		//   prod_low -.->|"FanIn (tag-a, cycle pruned)"| consumer
		// ```
		{
			name: "sole cyclic producer is pruned and consumer resolves with empty fan-in bindings",
			initialTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("prod-low", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			availableTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("prod-low", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			wantTaskIDs: []string{"consumer#default", "prod-low#default"},
			wantBoundRefIDsByTaskImplID: map[string][]string{
				"consumer#default": nil,
			},
		},
		// Mermaid task graph:
		// ```mermaid
		// graph TD
		//   consumer["consumer"] -.->|"FanIn (tag-a, self-loop pruned)"| consumer
		// ```
		{
			name: "self-loop sole producer is pruned and consumer resolves with empty fan-in bindings",
			initialTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			availableTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			wantTaskIDs: []string{"consumer#default"},
			wantBoundRefIDsByTaskImplID: map[string][]string{
				"consumer#default": nil,
			},
		},
		// Mermaid task graph:
		// ```mermaid
		// graph TD
		//   consumer["consumer"] -->|"PointToPoint"| prod_cyclic_high["prod-cyclic-high (priority=10)"]
		//   prod_cyclic_high -.->|"FanIn (tag-a, cyclic pruned)"| consumer
		//   prod_acyclic_low["prod-acyclic-low (priority=100)"] -->|"FanIn (tag-a, acyclic accepted)"| consumer
		// ```
		{
			name: "cyclic producer having higher priority is pruned while lower priority acyclic producer is accepted",
			initialTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("prod-cyclic-high", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}, ProvidesTag(tagA, WithTagPriority(10))),
				createMockTask("prod-acyclic-low", "default", nil, ProvidesTag(tagA, WithTagPriority(100))),
			},
			availableTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("prod-cyclic-high", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}, ProvidesTag(tagA, WithTagPriority(10))),
				createMockTask("prod-acyclic-low", "default", nil, ProvidesTag(tagA, WithTagPriority(100))),
			},
			wantTaskIDs: []string{"prod-acyclic-low#default", "consumer#default", "prod-cyclic-high#default"},
			wantBoundRefIDsByTaskImplID: map[string][]string{
				"consumer#default": {"prod-acyclic-low"},
			},
		},
		// Mermaid task graph:
		// ```mermaid
		// graph TD
		//   consumer["consumer"] -->|"PointToPoint"| task_mid["task-mid"]
		//   task_mid -->|"PointToPoint"| prod_transitive["prod-transitive (priority=100)"]
		//   prod_transitive -.->|"FanIn (tag-a, transitive cycle pruned)"| consumer
		// ```
		{
			name: "transitive multi-hop cycle is pruned and consumer resolves with empty fan-in bindings",
			initialTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("task-mid", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}),
				createMockTask("prod-transitive", "default", []Dependency{
					taskid.NewTaskReference[any]("task-mid"),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			availableTasks: []UntypedTask{
				createMockTask("consumer", "default", []Dependency{
					tagA.Ref(),
				}),
				createMockTask("task-mid", "default", []Dependency{
					taskid.NewTaskReference[any]("consumer"),
				}),
				createMockTask("prod-transitive", "default", []Dependency{
					taskid.NewTaskReference[any]("task-mid"),
				}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			wantTaskIDs: []string{"consumer#default", "task-mid#default", "prod-transitive#default"},
			wantBoundRefIDsByTaskImplID: map[string][]string{
				"consumer#default": nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskSet, err := ResolveGraph(tc.initialTasks, tc.availableTasks, nil)
			if err != nil {
				t.Fatalf("ResolveGraph() unexpected error = %v", err)
			}

			var gotTaskIDs []string
			for _, task := range taskSet.GetAll() {
				gotTaskIDs = append(gotTaskIDs, task.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantTaskIDs, gotTaskIDs); diff != "" {
				t.Errorf("ResolveGraph() task IDs mismatch (-want +got):\n%s", diff)
			}

			for taskImplID, wantRefs := range tc.wantBoundRefIDsByTaskImplID {
				gotRefs := taskSet.BoundReferenceIDsForTaskImplWithTag(taskImplID, tagA.ID())
				if diff := cmp.Diff(wantRefs, gotRefs); diff != "" {
					t.Errorf("BoundReferenceIDsForTaskImplWithTag(%q) mismatch (-want +got):\n%s", taskImplID, diff)
				}
			}
		})
	}
}

func TestResolveGraph_FanInCycle_ShuffleInvariance(t *testing.T) {
	tagA := NewTag[any]("tag-a")
	tagB := NewTag[any]("tag-b")

	baseInitialTasks := []UntypedTask{
		createMockTask("prod-high", "default", nil, ProvidesTag(tagA, WithTagPriority(10))),
		createMockTask("consumer", "default", []Dependency{
			tagA.Ref(),
			tagB.Ref(),
		}),
		createMockTask("prod-low", "default", []Dependency{
			taskid.NewTaskReference[any]("consumer"),
		}, ProvidesTag(tagA, WithTagPriority(100))),
		createMockTask("tag-b-prod", "default", nil, ProvidesTag(tagB, WithTagPriority(50))),
	}
	baseAvailableTasks := []UntypedTask{
		createMockTask("prod-high", "default", nil, ProvidesTag(tagA, WithTagPriority(10))),
		createMockTask("consumer", "default", []Dependency{
			tagA.Ref(),
			tagB.Ref(),
		}),
		createMockTask("prod-low", "default", []Dependency{
			taskid.NewTaskReference[any]("consumer"),
		}, ProvidesTag(tagA, WithTagPriority(100))),
		createMockTask("tag-b-prod", "default", nil, ProvidesTag(tagB, WithTagPriority(50))),
	}

	baselineSet, err := ResolveGraph(baseInitialTasks, baseAvailableTasks, nil)
	if err != nil {
		t.Fatalf("baseline ResolveGraph() failed: %v", err)
	}

	var wantTaskIDs []string
	for _, task := range baselineSet.GetAll() {
		wantTaskIDs = append(wantTaskIDs, task.UntypedID().String())
	}

	r := rand.New(rand.NewPCG(42, 1024))
	for i := range 100 {
		shuffledInitialTasks := slices.Clone(baseInitialTasks)
		r.Shuffle(len(shuffledInitialTasks), func(i, j int) {
			shuffledInitialTasks[i], shuffledInitialTasks[j] = shuffledInitialTasks[j], shuffledInitialTasks[i]
		})

		shuffledAvailableTasks := slices.Clone(baseAvailableTasks)
		r.Shuffle(len(shuffledAvailableTasks), func(i, j int) {
			shuffledAvailableTasks[i], shuffledAvailableTasks[j] = shuffledAvailableTasks[j], shuffledAvailableTasks[i]
		})

		shuffledSet, err := ResolveGraph(shuffledInitialTasks, shuffledAvailableTasks, nil)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}

		var gotTaskIDs []string
		for _, task := range shuffledSet.GetAll() {
			gotTaskIDs = append(gotTaskIDs, task.UntypedID().String())
		}

		if diff := cmp.Diff(wantTaskIDs, gotTaskIDs); diff != "" {
			t.Fatalf("iteration %d: task order mismatch (-want +got):\n%s", i, diff)
		}
	}
}

func TestResolveGraph_FanInCycle_MutualFanIn(t *testing.T) {
	tagA := NewTag[any]("tag-a")
	tagB := NewTag[any]("tag-b")

	testCases := []struct {
		name                        string
		initialTasks                []UntypedTask
		availableTasks              []UntypedTask
		wantTaskIDs                 []string
		wantBoundRefIDsByTaskImplID map[string]map[string][]string
	}{
		// Mermaid task graph:
		// ```mermaid
		// graph TD
		//   task_a["task-a (consumes tag-b, provides tag-a)"]
		//   task_b["task-b (consumes tag-a, provides tag-b)"]
		//   task_b -.->|"FanIn (tag-b, accepted)"| task_a
		//   task_a -.->|"FanIn (tag-a, cycle pruned)"| task_b
		// ```
		{
			name: "mutual fan-in cycle between two consumers is deterministically broken by pruning the second candidate edge",
			initialTasks: []UntypedTask{
				createMockTask("task-a", "default", []Dependency{tagB.Ref()}, ProvidesTag(tagA, WithTagPriority(100))),
				createMockTask("task-b", "default", []Dependency{tagA.Ref()}, ProvidesTag(tagB, WithTagPriority(100))),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-a", "default", []Dependency{tagB.Ref()}, ProvidesTag(tagA, WithTagPriority(100))),
				createMockTask("task-b", "default", []Dependency{tagA.Ref()}, ProvidesTag(tagB, WithTagPriority(100))),
			},
			wantTaskIDs: []string{"task-b#default", "task-a#default"},
			wantBoundRefIDsByTaskImplID: map[string]map[string][]string{
				"task-a#default": {
					tagB.ID(): {"task-b"},
				},
				"task-b#default": {
					tagA.ID(): nil,
				},
			},
		},
		{
			name: "mutual fan-in cycle resolution is invariant to input task slice order",
			initialTasks: []UntypedTask{
				createMockTask("task-b", "default", []Dependency{tagA.Ref()}, ProvidesTag(tagB, WithTagPriority(100))),
				createMockTask("task-a", "default", []Dependency{tagB.Ref()}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			availableTasks: []UntypedTask{
				createMockTask("task-b", "default", []Dependency{tagA.Ref()}, ProvidesTag(tagB, WithTagPriority(100))),
				createMockTask("task-a", "default", []Dependency{tagB.Ref()}, ProvidesTag(tagA, WithTagPriority(100))),
			},
			wantTaskIDs: []string{"task-b#default", "task-a#default"},
			wantBoundRefIDsByTaskImplID: map[string]map[string][]string{
				"task-a#default": {
					tagB.ID(): {"task-b"},
				},
				"task-b#default": {
					tagA.ID(): nil,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskSet, err := ResolveGraph(tc.initialTasks, tc.availableTasks, nil)
			if err != nil {
				t.Fatalf("ResolveGraph() unexpected error = %v", err)
			}

			var gotTaskIDs []string
			for _, task := range taskSet.GetAll() {
				gotTaskIDs = append(gotTaskIDs, task.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantTaskIDs, gotTaskIDs); diff != "" {
				t.Errorf("ResolveGraph() task IDs mismatch (-want +got):\n%s", diff)
			}

			for taskImplID, tags := range tc.wantBoundRefIDsByTaskImplID {
				for tagID, wantRefs := range tags {
					gotRefs := taskSet.BoundReferenceIDsForTaskImplWithTag(taskImplID, tagID)
					if diff := cmp.Diff(wantRefs, gotRefs); diff != "" {
						t.Errorf("BoundReferenceIDsForTaskImplWithTag(%q, %q) mismatch (-want +got):\n%s", taskImplID, tagID, diff)
					}
				}
			}
		})
	}
}
