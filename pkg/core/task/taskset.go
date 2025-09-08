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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"golang.org/x/exp/slices"
)

type LabelPredicate[T any] = func(v T) bool

// TaskSet is a collection of tasks.
// It has several collection operation features for constructing the task graph to execute.
type TaskSet struct {
	tasks    []UntypedTask
	runnable bool
}

// sortTaskResult represents result of topological sorting tasks.
type sortTaskResult struct {
	// TopologicalSortedTasks is the list of tasks in topological order.
	TopologicalSortedTasks []UntypedTask
	// MissingDependencies is the list of task reference Ids missed to resolve task dependencies.
	// This must be empty array when the sorting succeeded.
	MissingDependencies []taskid.UntypedTaskReference
	// CyclicDependencyPath is the path of task dependencies. Runnable became false if this field is "".
	CyclicDependencyPath string
	// Runnable indicate if this task graph is runnable or not. It means the tasks are sorted in topoligical order and all of input dependencies are resolved.
	Runnable bool
}

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
		tasks:    slices.Clone(tasks),
		runnable: false,
	}, nil
}

// Add a task definiton to current TaskSet.
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

func (s *TaskSet) GetAll() []UntypedTask {
	return slices.Clone(s.tasks)
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

func (s *TaskSet) sortTaskGraph() *sortTaskResult {
	// To check if there were no cyclic task path or missing inputs,
	// perform the topological sorting algorithm known as Kahn's algorithm
	// Reference: https://en.wikipedia.org/wiki/Topological_sorting
	nonResolvedTasksMap := map[string]UntypedTask{}
	currentMissingTaskDependencies := map[string]map[string]interface{}{}
	currentMissingTaskSourceCount := map[string]int{}
	taskCount := 0

	// Initialize currentMissingTaskDependencies and currentMissingTaskSourceCount for all tasks.
	for _, task := range s.tasks {
		taskID := task.UntypedID()

		sourceCount := 0
		missingDependencies := map[string]interface{}{}
		for _, dependency := range task.Dependencies() {
			if _, found := missingDependencies[dependency.ReferenceIDString()]; !found {
				missingDependencies[dependency.ReferenceIDString()] = struct{}{}
				sourceCount += 1
			}
		}
		currentMissingTaskDependencies[taskID.String()] = missingDependencies
		nonResolvedTasksMap[taskID.String()] = task
		currentMissingTaskSourceCount[taskID.String()] = sourceCount
		taskCount += 1
	}

	topologicalSortedTasks := []UntypedTask{}
	for i := 0; i < taskCount; i++ {
		var nextTaskID string = "N/A"
		for _, taskId := range sortedMapKeys(nonResolvedTasksMap) { // Needs task sorting to get the same result every time.
			if currentMissingTaskSourceCount[taskId] == 0 {
				nextTaskID = taskId
			}
		}

		if nextTaskID != "N/A" {
			nextTask := nonResolvedTasksMap[nextTaskID]
			delete(nonResolvedTasksMap, nextTaskID)
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
				for dependency := range currentMissingTaskDependencies[taskId] {
					missingTaskIdsInMap[dependency] = struct{}{}
				}
			}
			for _, task := range nonResolvedTasksMap {
				delete(missingTaskIdsInMap, task.UntypedID().ReferenceIDString())
			}

			missingSources := []taskid.UntypedTaskReference{}
			for source := range missingTaskIdsInMap {
				missingSources = append(missingSources, taskid.NewTaskReference[any](source))
			}

			if len(missingSources) == 0 {
				// If there are no missing dependencies but still can't resolve the graph,
				// it means there is a cyclic dependency
				return getSortTaskResultWithDetailCyclicDependency(nonResolvedTasksMap, currentMissingTaskDependencies, missingSources)
			}

			return &sortTaskResult{
				Runnable:               false,
				TopologicalSortedTasks: nil,
				CyclicDependencyPath:   "",
				MissingDependencies:    missingSources,
			}
		}
	}

	return &sortTaskResult{
		Runnable:               true,
		TopologicalSortedTasks: topologicalSortedTasks,
		MissingDependencies:    []taskid.UntypedTaskReference{},
		CyclicDependencyPath:   "",
	}
}

// ToRunnableTaskSet sorts given task list as topological order.
func (s *TaskSet) ToRunnableTaskSet() (*TaskSet, error) {
	sourceTaskSet := s
	sortResult := sourceTaskSet.sortTaskGraph()
	if sortResult.Runnable {
		return &TaskSet{tasks: sortResult.TopologicalSortedTasks, runnable: true}, nil
	} else {
		if sortResult.CyclicDependencyPath != "" {
			return nil, fmt.Errorf("failed to sort as a runnable task graph. \n The graph contains cyclic dependency\n%s", sortResult.CyclicDependencyPath)
		}

		if len(sortResult.MissingDependencies) > 0 {
			slices.SortFunc(sortResult.MissingDependencies, func(a, b taskid.UntypedTaskReference) int {
				return strings.Compare(a.ReferenceIDString(), b.ReferenceIDString())
			})
			missingDependenciesStr := &strings.Builder{}
			for _, missingRef := range sortResult.MissingDependencies {
				missingDependenciesStr.WriteString(fmt.Sprintf("* %s\n", missingRef.ReferenceIDString()))
			}
			return nil, fmt.Errorf("missing dependency found int he given task set.\n Missing %s", missingDependenciesStr.String())
		}
		return nil, fmt.Errorf("failed to sort as a runnable task graph. unreachable")
	}
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
		// concept of the feature is not defined in task level, but it's better to be included in the dumpped graph.
		// The ID can't be referenced directly because of the circular dependency issue, thus this code define the ID with NewLabelKey
		feature := typedmap.GetOrDefault(task.Labels(), NewTaskLabelKey[bool]("khi.google.com/inspection/feature"), false)
		shape := "circle"
		if feature {
			shape = "doublecircle"
		}
		result += fmt.Sprintf("%s [shape=\"%s\",label=\"%s\"]\n", graphVizValidId(task.UntypedID().String()), shape, task.UntypedID())
	}

	for _, task := range s.tasks {
		if len(task.Dependencies()) == 0 {
			result += fmt.Sprintf("start -> %s\n", graphVizValidId(task.UntypedID().String()))
		}
	}
	sourceRelation := map[string]UntypedTask{}
	for _, task := range s.tasks {
		sources := task.Dependencies()
		for _, source := range sources {
			sourceTask := sourceRelation[source.ReferenceIDString()]
			result += fmt.Sprintf("%s -> %s\n", graphVizValidId(sourceTask.UntypedID().String()), graphVizValidId(task.UntypedID().String()))
		}
		sourceRelation[task.UntypedID().ReferenceIDString()] = task
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

// getSortTaskResultWithDetailCyclicDependency detects and reports cyclic dependencies in the task graph.
// It returns a sortTaskResult with the details of the cyclic dependency.
func getSortTaskResultWithDetailCyclicDependency(
	nonResolvedTasksMap map[string]UntypedTask,
	currentMissingTaskDependencies map[string]map[string]interface{},
	missingSources []taskid.UntypedTaskReference,
) *sortTaskResult {
	for _, taskID := range sortedMapKeys(nonResolvedTasksMap) {
		dependentFrom := map[string]string{} // A map tracks the path where the task depended from.
		dependentFrom[taskID] = "START"
		queue := map[string]struct{}{}
		queue[taskID] = struct{}{}

		for len(queue) > 0 {
			nextTaskID := sortedMapKeys(queue)[0]
			delete(queue, nextTaskID)
			for dependency := range currentMissingTaskDependencies[nextTaskID] {
				prevParent := ""
				for visitedTask := range dependentFrom {
					// The task ID contains implementation hash(#default), it should match with the prefix.
					if strings.HasPrefix(visitedTask, dependency) {
						prevParent = dependentFrom[visitedTask]
						break
					}
				}
				if prevParent != "" {
					if prevParent == "START" {
						// now we found the path to loop back to the START. trace back the cyclic path.
						path := []string{}
						queue := map[string]struct{}{}
						queue[nextTaskID] = struct{}{}
						for len(queue) > 0 {
							nextTaskID := sortedMapKeys(queue)[0]
							if nextTaskID == "START" {
								break
							}
							delete(queue, nextTaskID)
							path = append(path, nextTaskID)
							queue[dependentFrom[nextTaskID]] = struct{}{}
						}

						return &sortTaskResult{
							Runnable:               false,
							TopologicalSortedTasks: nil,
							CyclicDependencyPath:   fmt.Sprintf("... -> %s] -> [%s] -> [%s -> ...", path[len(path)-1], strings.Join(path, " -> "), path[0]),
							MissingDependencies:    missingSources,
						}
					}
				} else {
					for taskID := range nonResolvedTasksMap {
						if strings.HasPrefix(taskID, dependency) {
							dependentFrom[taskID] = nextTaskID
							queue[taskID] = struct{}{}
							break
						}
					}
				}
			}
		}
	}
	nonResolvedTaskKeys := sortedMapKeys(nonResolvedTasksMap)
	missingSourceDependencyInfo := []string{}
	for missingDependencyKey, missingDependency := range currentMissingTaskDependencies {

		missingSourceDependencyInfo = append(missingSourceDependencyInfo, fmt.Sprintf("%s -> %v", missingDependencyKey, sortedMapKeys(missingDependency)))
	}
	// This should be unreachable if the graph has a cyclic dependency
	panic(fmt.Sprintf("unreachable. findCyclicDependency was called on a task graph with a task graph without any cyclic dependency. \n debug info: \n non resolved tasks: %v \n missing dependencies: %s", nonResolvedTaskKeys, missingSourceDependencyInfo))
}

// wrapGraphFirstTask is an implementation of Task to rewrite its dependency for wrapping graphs as a sub graph.
// This is only used in the WrapGraph method.
type wrapGraphFirstTask struct {
	task         UntypedTask
	dependencies []taskid.UntypedTaskReference
}

// Dependencies implements Task.
func (w *wrapGraphFirstTask) Dependencies() []taskid.UntypedTaskReference {
	return w.dependencies
}

// ID implements Task.
func (w *wrapGraphFirstTask) ID() taskid.TaskImplementationID[any] {
	untypedID := w.task.UntypedID()
	return taskid.NewImplementationID(taskid.NewTaskReference[any](untypedID.GetUntypedReference().String()), untypedID.GetTaskImplementationHash())
}

// Labels implements Task.
func (w *wrapGraphFirstTask) Labels() *typedmap.ReadonlyTypedMap {
	return w.task.Labels()
}

// Run implements Task.
func (w *wrapGraphFirstTask) Run(ctx context.Context) (any, error) {
	return w.task.UntypedRun(ctx)
}

func (w *wrapGraphFirstTask) UntypedRun(ctx context.Context) (any, error) {
	return w.Run(ctx)
}

func (w *wrapGraphFirstTask) UntypedID() taskid.UntypedTaskImplementationID {
	return w.task.UntypedID()
}

var _ Task[any] = (*wrapGraphFirstTask)(nil)
