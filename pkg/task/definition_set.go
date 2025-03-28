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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"golang.org/x/exp/slices"
)

type LabelPredicate[T any] = func(v T) bool

// DefinitionSet is a collection of task definitions.
// It has several collection operation features for constructing the task graph to execute.
type DefinitionSet struct {
	definitions []UntypedDefinition
	runnable    bool
}

// SortTaskGraphResult represents result of topological sorting taks definitions.
type SortTaskGraphResult struct {
	// TopologicalSortedTasks is the list of definitions in topological order.
	TopologicalSortedTasks []UntypedDefinition
	// MissingDependencies is the list of task reference Ids missed to resolve task dependencies.
	// This must be empty array when the sorting succeeded.
	MissingDependencies []taskid.UntypedTaskReference
	// HasCyclicDependency indicates if the task graph containing any cyclic dependencies.
	HasCyclicDependency bool
	// Runnable indicate if this task graph is runnable or not. It means the definitions are sorted in topoligical order and all of input dependencies are resolved.
	Runnable bool
}

func NewSet(definitions []UntypedDefinition) (*DefinitionSet, error) {
	definitionIds := map[string]struct{}{}
	for _, def := range definitions {
		id := def.UntypedID()
		if _, exist := definitionIds[id.String()]; exist {
			return nil, fmt.Errorf("multiple definitions have the same ID %s", id)
		}
		definitionIds[id.String()] = struct{}{}
	}
	return &DefinitionSet{
		definitions: slices.Clone(definitions),
		runnable:    false,
	}, nil
}

// Add a task definiton to current DefinitionSet.
// Returns an error when duplicated task Id is assigned on the task.
func (s *DefinitionSet) Add(newTask UntypedDefinition) error {
	taskIdMap := map[string]interface{}{}
	for _, task := range s.definitions {
		taskIdMap[task.UntypedID().String()] = struct{}{}
	}
	if _, exist := taskIdMap[newTask.UntypedID().String()]; exist {
		return fmt.Errorf("task definition id:%s is duplicated. Definition ID must be unique", newTask.UntypedID())
	}
	s.definitions = append(s.definitions, newTask)
	return nil
}

func (s *DefinitionSet) GetAll() []UntypedDefinition {
	return slices.Clone(s.definitions)
}

// Get returns a task definition queried with an id of the task definition.
func (s *DefinitionSet) Get(id string) (UntypedDefinition, error) {
	for _, task := range s.definitions {
		if task.UntypedID().String() == id {
			return task, nil
		}
	}
	return nil, fmt.Errorf("task definition %s was not found", id)
}

// WrapGraph adds init task and done task to the runnable graph.
// The init task named as `subgraphId`-init has the dependency provided in the subgraphDependency argument. And the init task will be dependency of the tasks that had no dependency before calling this method.
// The done task named as `subgraphId`-done has the dependency of the tasks that were not dependent from any other tasks.
// The result task set will be resolvable with `[the init task] -> [the other tasks] -> [the done task]`
func (s *DefinitionSet) WrapGraph(subgraphId taskid.UntypedTaskImplementationID, subgraphDependency []taskid.UntypedTaskReference) (*DefinitionSet, error) {
	initTaskId := taskid.NewImplementationID(taskid.NewTaskReference[any](fmt.Sprintf("%s-init", subgraphId.ReferenceIDString())), subgraphId.GetTaskImplementationHash())
	doneTaskId := taskid.NewImplementationID(taskid.NewTaskReference[any](fmt.Sprintf("%s-done", subgraphId.ReferenceIDString())), subgraphId.GetTaskImplementationHash())
	rewiredTasks := []UntypedDefinition{}
	tasksNotDependentFromAnyMap := map[string]struct{}{}
	for _, t := range s.definitions {
		if len(t.Dependencies()) == 0 {
			capturedTask := t
			rewiredTask := &wrapGraphFirstTask{
				task:         capturedTask,
				dependencies: []taskid.UntypedTaskReference{initTaskId.GetTaskReference()},
			}
			rewiredTasks = append(rewiredTasks, rewiredTask)
		} else {
			rewiredTasks = append(rewiredTasks, t)
		}
		tasksNotDependentFromAnyMap[t.UntypedID().GetUntypedReference().String()] = struct{}{}
	}
	for _, t := range s.definitions {
		for _, dep := range t.Dependencies() {
			delete(tasksNotDependentFromAnyMap, dep.String())
		}
	}

	doneTaskDependencies := []taskid.UntypedTaskReference{
		initTaskId.GetTaskReference(),
	}
	for k := range tasksNotDependentFromAnyMap {
		doneTaskDependencies = append(doneTaskDependencies, taskid.NewTaskReference[any](k))
	}
	// Sort to make result stable
	slices.SortFunc(doneTaskDependencies, func(a, b taskid.UntypedTaskReference) int { return strings.Compare(a.String(), b.String()) })
	initTask := NewTask(initTaskId, subgraphDependency, func(ctx context.Context) (any, error) { return nil, nil })
	doneTask := NewTask(doneTaskId, doneTaskDependencies, func(ctx context.Context) (any, error) { return nil, nil })
	rewiredTasks = append(rewiredTasks, initTask, doneTask)
	return NewSet(rewiredTasks)
}

func (s *DefinitionSet) sortTaskGraph() *SortTaskGraphResult {
	// To check if there were no cyclic task path or missing inputs,
	// perform the topological sorting algorithm known as Kahn's algorithm
	// Reference: https://en.wikipedia.org/wiki/Topological_sorting
	nonResolvedTasksMap := map[string]UntypedDefinition{}
	currentMissingTaskDependencies := map[string]map[string]interface{}{}
	currentMissingTaskSourceCount := map[string]int{}
	taskCount := 0
	for _, task := range s.definitions {
		id := task.UntypedID()
		dependencies := task.Dependencies()
		if _, found := currentMissingTaskDependencies[id.String()]; found {
			continue
		}
		currentMissingTaskDependencies[id.String()] = map[string]interface{}{}
		sourceCount := 0
		for _, dependency := range dependencies {
			currentMissingTaskDependencies[id.String()][dependency.ReferenceIDString()] = struct{}{}
			sourceCount += 1
		}
		nonResolvedTasksMap[id.String()] = task
		currentMissingTaskSourceCount[id.String()] = sourceCount
		taskCount += 1
	}

	topologicalSortedTasks := []UntypedDefinition{}
	for i := 0; i < taskCount; i++ {
		var nextResolveTaskId string = "N/A"
		nextResolvedTaskIdThreadUnsafeCandidate := "N/A"
		for _, taskId := range sortedMapKeys(nonResolvedTasksMap) { // Needs task sorting to get the same result every time.
			if currentMissingTaskSourceCount[taskId] == 0 {
				if nextResolvedTaskIdThreadUnsafeCandidate == "N/A" {
					nextResolvedTaskIdThreadUnsafeCandidate = taskId
				}
			}
		}
		if nextResolveTaskId == "N/A" {
			nextResolveTaskId = nextResolvedTaskIdThreadUnsafeCandidate
		}
		if nextResolveTaskId != "N/A" {
			nextTask := nonResolvedTasksMap[nextResolveTaskId]
			delete(nonResolvedTasksMap, nextResolveTaskId)
			removingDependencyId := nextTask.UntypedID().ReferenceIDString()
			for taskId := range nonResolvedTasksMap {
				if _, exist := currentMissingTaskDependencies[taskId][removingDependencyId]; exist {
					delete(currentMissingTaskDependencies[taskId], removingDependencyId)
					currentMissingTaskSourceCount[taskId]--
				}
			}
			topologicalSortedTasks = append(topologicalSortedTasks, nextTask)
		} else {
			// Failed to perform topological sort.
			// Gathers the cause of the failure.
			missingTaskIdsInMap := map[string]interface{}{}
			for taskId := range nonResolvedTasksMap {
				for source := range currentMissingTaskDependencies[taskId] {
					missingTaskIdsInMap[source] = struct{}{}
				}
			}
			for _, task := range nonResolvedTasksMap {
				delete(missingTaskIdsInMap, task.UntypedID().ReferenceIDString())
			}

			// When there were no task runnable only with the missing sources,
			// the task graph shape is at least cyclic.
			hasCyclicDependencies := true
			for taskId := range nonResolvedTasksMap {
				canBeNextStartingPoint := true
				for source := range currentMissingTaskDependencies[taskId] {
					if _, exist := missingTaskIdsInMap[source]; !exist {
						canBeNextStartingPoint = false
						break
					}
				}
				if canBeNextStartingPoint {
					hasCyclicDependencies = false
					break
				}
			}

			if !hasCyclicDependencies {
				for _, task := range nonResolvedTasksMap {
					delete(missingTaskIdsInMap, task.UntypedID().ReferenceIDString())
				}
			}

			missingSources := []taskid.UntypedTaskReference{}
			for source := range missingTaskIdsInMap {
				missingSources = append(missingSources, taskid.NewTaskReference[any](source))
			}

			return &SortTaskGraphResult{
				Runnable:               false,
				TopologicalSortedTasks: nil,
				HasCyclicDependency:    hasCyclicDependencies,
				MissingDependencies:    missingSources,
			}
		}
	}

	return &SortTaskGraphResult{
		Runnable:               true,
		TopologicalSortedTasks: topologicalSortedTasks,
		MissingDependencies:    []taskid.UntypedTaskReference{},
		HasCyclicDependency:    false,
	}
}

// ResolveTask generate a super set of this definition set with adding required tasks from availableTaskSet.
// The returned DefinitionSet of this method will be `runnable` and topologically sorted.
func (s *DefinitionSet) ResolveTask(availableDefinitionSet *DefinitionSet) (*DefinitionSet, error) {
	sourceTaskSet := s
	sortResult := sourceTaskSet.sortTaskGraph()
	if sortResult.Runnable {
		return &DefinitionSet{definitions: sortResult.TopologicalSortedTasks, runnable: true}, nil
	} else {
		// the sourceTaskSet can't be topologically sorted with its own tasks.
		// Try to add missing dependencies from availableDefinitionSet
		complementedTask := []UntypedDefinition{}
		resolutionFailure := false
		var missingTaskId taskid.UntypedTaskReference
		for _, missingSource := range sortResult.MissingDependencies {
			matched := []UntypedDefinition{}
			for _, task := range availableDefinitionSet.definitions {
				missingSourceReference := missingSource.ReferenceIDString()
				if task.UntypedID().ReferenceIDString() == missingSourceReference {
					matched = append(matched, task)
				}
			}
			// sort matched tasks with selection priority for in case when there are 2 or more tasks can be usable for resolving required dependency
			maxPriority := -1
			var maxPriorityTaskDefinition UntypedDefinition
			for _, task := range matched {
				priority := typedmap.GetOrDefault(task.Labels(), LabelKeyTaskSelectionPriority, 0)
				if priority >= maxPriority {
					maxPriority = priority
					maxPriorityTaskDefinition = task
				}
			}

			if maxPriorityTaskDefinition != nil {
				complementedTask = append(complementedTask, maxPriorityTaskDefinition)
			} else {
				resolutionFailure = true
				missingTaskId = missingSource
			}
		}
		if !resolutionFailure {
			tasks := append(slices.Clone(sourceTaskSet.definitions), complementedTask...)
			sourceTaskSet = &DefinitionSet{
				definitions: tasks,
				runnable:    false,
			}
			return sourceTaskSet.ResolveTask(availableDefinitionSet)
		}
		return nil, fmt.Errorf("Failed to resolve the task set.\n Missing %s\nAvailable tasks:\n%v", missingTaskId.ReferenceIDString(), dumpTaskIDList(availableDefinitionSet))
	}
}

// DumpGraphviz returns definition graph as graphviz string for debugging purpose.
// The generated string can be converted to DAG graph using `dot` command.
func (s *DefinitionSet) DumpGraphviz() (string, error) {
	if !s.runnable {
		return "", fmt.Errorf("can't draw a graph for non runnable graph")
	}
	result := "digraph G {\n"
	result += "start [shape=\"diamond\",fillcolor=gray,style=filled]\n"
	for _, definition := range s.definitions {
		// concept of the feature is not defined in task level, but it's better to be included in the dumpped graph.
		// The ID can't be referenced directly because of the circular dependency issue, thus this code define the ID with NewLabelKey
		feature := typedmap.GetOrDefault(definition.Labels(), NewTaskLabelKey[bool]("khi.google.com/inspection/feature"), false)
		shape := "circle"
		if feature {
			shape = "doublecircle"
		}
		result += fmt.Sprintf("%s [shape=\"%s\",label=\"%s\"]\n", graphVizValidId(definition.UntypedID().String()), shape, definition.UntypedID())
	}

	for _, definition := range s.definitions {
		if len(definition.Dependencies()) == 0 {
			result += fmt.Sprintf("start -> %s\n", graphVizValidId(definition.UntypedID().String()))
		}
	}
	sourceRelation := map[string]UntypedDefinition{}
	for _, definition := range s.definitions {
		sources := definition.Dependencies()
		for _, source := range sources {
			sourceDefinition := sourceRelation[source.ReferenceIDString()]
			result += fmt.Sprintf("%s -> %s\n", graphVizValidId(sourceDefinition.UntypedID().String()), graphVizValidId(definition.UntypedID().String()))
		}
		sourceRelation[definition.UntypedID().ReferenceIDString()] = definition
	}
	result += "}"
	return result, nil
}

func sortedMapKeys[T any](inputMap map[string]T) []string {
	result := []string{}
	for key := range inputMap {
		result = append(result, key)
	}
	slices.SortFunc(result, strings.Compare)
	return result
}

func graphVizValidId(id string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(id, "-", "_"), "/", "_"), ".", "_"), "#", "_")
}

func dumpTaskIDList(taskSet *DefinitionSet) string {
	taskIDs := []string{}
	for _, task := range taskSet.definitions {
		taskIDs = append(taskIDs, task.UntypedID().String())
	}
	slices.SortFunc(taskIDs, strings.Compare)
	result := ""
	for _, taskID := range taskIDs {
		result += fmt.Sprintf("- %s\n", taskID)
	}
	return result
}

// wrapGraphFirstTask is an implementation of Task to rewrite its dependency for wrapping graphs as a sub graph.
// This is only used in the WrapGraph method.
type wrapGraphFirstTask struct {
	task         UntypedDefinition
	dependencies []taskid.UntypedTaskReference
}

// Dependencies implements Definition.
func (w *wrapGraphFirstTask) Dependencies() []taskid.UntypedTaskReference {
	return w.dependencies
}

// ID implements Definition.
func (w *wrapGraphFirstTask) ID() taskid.TaskImplementationID[any] {
	untypedID := w.task.UntypedID()
	return taskid.NewImplementationID(taskid.NewTaskReference[any](untypedID.GetUntypedReference().String()), untypedID.GetTaskImplementationHash())
}

// Labels implements Definition.
func (w *wrapGraphFirstTask) Labels() *typedmap.ReadonlyTypedMap {
	return w.task.Labels()
}

// Run implements Definition.
func (w *wrapGraphFirstTask) Run(ctx context.Context) (any, error) {
	return w.task.UntypedRun(ctx)
}

func (w *wrapGraphFirstTask) UntypedRun(ctx context.Context) (any, error) {
	return w.Run(ctx)
}

func (w *wrapGraphFirstTask) UntypedID() taskid.UntypedTaskImplementationID {
	return w.task.UntypedID()
}

var _ Definition[any] = (*wrapGraphFirstTask)(nil)
