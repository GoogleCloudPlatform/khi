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
	"fmt"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
)

type LabelPredicate[T any] = func(v T) bool

// TaskSet is a collection of tasks and resolved dependency edges.
// It implements core_contract.TaskGraphMetadata and provides querying for execution order and edges.
type TaskSet struct {
	tasks                        []UntypedTask
	edges                        []taskid.TaskEdge
	runnable                     bool
	incomingEdges                map[string][]taskid.TaskEdge   // key: target task implementation ID
	boundRefIDs                  map[string]struct{}            // set of task reference IDs bound to the graph
	boundFanInRefIDsByTaskImplID map[string]map[string][]string // targetImplID -> tag -> []sourceRefID
}

var _ core_contract.TaskGraphMetadata = (*TaskSet)(nil)

// NewTaskSet creates a new TaskSet with the given tasks.
// Returns an error if there are duplicate task IDs.
func NewTaskSet(tasks []UntypedTask) (*TaskSet, error) {
	taskIDs := map[string]struct{}{}
	for _, def := range tasks {
		id := def.UntypedID()
		if _, exist := taskIDs[id.String()]; exist {
			return nil, fmt.Errorf("multiple tasks have the same ID %s", id)
		}
		taskIDs[id.String()] = struct{}{}
	}
	return &TaskSet{
		tasks:                        slices.Clone(tasks),
		runnable:                     false,
		incomingEdges:                make(map[string][]taskid.TaskEdge),
		boundRefIDs:                  make(map[string]struct{}),
		boundFanInRefIDsByTaskImplID: make(map[string]map[string][]string),
	}, nil
}

// NewResolvedTaskSet creates a new runnable TaskSet with resolved tasks, edges, and metadata.
func NewResolvedTaskSet(
	tasks []UntypedTask,
	edges []taskid.TaskEdge,
	boundFanInRefIDsByTaskImplID map[string]map[string][]string,
) *TaskSet {
	incomingEdges := make(map[string][]taskid.TaskEdge)
	boundRefIDs := make(map[string]struct{})

	for _, t := range tasks {
		boundRefIDs[t.UntypedID().ReferenceIDString()] = struct{}{}
	}

	for _, e := range edges {
		incomingEdges[e.TargetImplID] = append(incomingEdges[e.TargetImplID], e)
	}

	copiedBoundFanInRefIDsByTaskImplID := make(map[string]map[string][]string)
	for targetImplID, byTag := range boundFanInRefIDsByTaskImplID {
		copiedBoundFanInRefIDsByTaskImplID[targetImplID] = make(map[string][]string)
		for tag, refIDs := range byTag {
			copiedBoundFanInRefIDsByTaskImplID[targetImplID][tag] = slices.Clone(refIDs)
		}
	}

	return &TaskSet{
		tasks:                        slices.Clone(tasks),
		edges:                        slices.Clone(edges),
		runnable:                     true,
		incomingEdges:                incomingEdges,
		boundRefIDs:                  boundRefIDs,
		boundFanInRefIDsByTaskImplID: copiedBoundFanInRefIDsByTaskImplID,
	}
}

// Add a task definition to current TaskSet.
// Returns an error when duplicated task Id is assigned on the task.
func (s *TaskSet) Add(newTask UntypedTask) error {
	taskIdMap := map[string]interface{}{}
	for _, task := range s.tasks {
		taskIdMap[task.UntypedID().String()] = struct{}{}
	}
	if _, exist := taskIdMap[newTask.UntypedID().String()]; exist {
		return fmt.Errorf("task id:%s is duplicated. Task ID must be unique", newTask.UntypedID())
	}
	s.tasks = append(s.tasks, newTask)
	return nil
}

// GetAll returns a copy of all tasks in the set.
func (s *TaskSet) GetAll() []UntypedTask {
	return slices.Clone(s.tasks)
}

// Edges returns a copy of all resolved edges in the set.
func (s *TaskSet) Edges() []taskid.TaskEdge {
	return slices.Clone(s.edges)
}

// IncomingEdges returns incoming edges for the given task implementation ID.
func (s *TaskSet) IncomingEdges(taskImplID string) []taskid.TaskEdge {
	return s.incomingEdges[taskImplID]
}

// IsBound returns true if the task reference was bound to the graph.
func (s *TaskSet) IsBound(refID string) bool {
	_, found := s.boundRefIDs[refID]
	return found
}

// BoundReferenceIDsForTaskImplWithTag returns the list of task reference IDs providing the tag bound specifically to the given task implementation ID.
func (s *TaskSet) BoundReferenceIDsForTaskImplWithTag(taskImplementationID string, tag string) []string {
	if byTag, ok := s.boundFanInRefIDsByTaskImplID[taskImplementationID]; ok {
		if refIDs, ok := byTag[tag]; ok {
			return slices.Clone(refIDs)
		}
	}
	return nil
}

// Remove a task definition from current DefinitionSet.
// Returns error if the definition does not exist
func (s *TaskSet) Remove(id string) error {
	taskIdMap := map[string]interface{}{}
	for _, task := range s.tasks {
		taskIdMap[task.UntypedID().String()] = struct{}{}
	}
	if _, exist := taskIdMap[id]; !exist {
		return fmt.Errorf("task definition id:%s is not found in this set", id)
	}
	n := 0
	for _, task := range s.tasks {
		if task.UntypedID().String() != id {
			s.tasks[n] = task
			n++
		}
	}
	s.tasks = s.tasks[:n]
	return nil
}

// Get returns a task with the given string task ID notation.
func (s *TaskSet) Get(id string) (UntypedTask, error) {
	for _, task := range s.tasks {
		if task.UntypedID().String() == id {
			return task, nil
		}
	}
	return nil, fmt.Errorf("task %s was not found", id)
}

// DumpGraphviz returns task graph as graphviz string for debugging purpose.
// The generated string can be converted to DAG graph using `dot` command.
func (s *TaskSet) DumpGraphviz() (string, error) {
	if !s.runnable {
		return "", fmt.Errorf("can't draw a graph for non runnable graph")
	}
	result := "digraph G {\n"
	result += "start [shape=\"diamond\",fillcolor=gray,style=filled]\n"
	for _, task := range s.tasks {
		feature := typedmap.GetOrDefault(task.Labels(), NewTaskLabelKey[bool]("khi.google.com/inspection/feature"), false)
		shape := "circle"
		if feature {
			shape = "doublecircle"
		}
		result += fmt.Sprintf("%s [shape=\"%s\",label=\"%s\"]\n", graphVizValidId(task.UntypedID().String()), shape, task.UntypedID())
	}

	for _, task := range s.tasks {
		if len(s.IncomingEdges(task.UntypedID().String())) == 0 {
			result += fmt.Sprintf("start -> %s\n", graphVizValidId(task.UntypedID().String()))
		}
	}
	for _, task := range s.tasks {
		sources := s.IncomingEdges(task.UntypedID().String())
		for _, edge := range sources {
			result += fmt.Sprintf("%s -> %s\n", graphVizValidId(edge.SourceImplID), graphVizValidId(task.UntypedID().String()))
		}
	}
	result += "}"
	return result, nil
}
func graphVizValidId(id string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(id, "-", "_"), "/", "_"), ".", "_"), "#", "_")
}
