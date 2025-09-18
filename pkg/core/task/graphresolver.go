// Copyright 2025 Google LLC
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
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// DefaultTaskGraphResolver is the default configuration of graph resolver used for constructing complete task graph.
var DefaultTaskGraphResolver = NewGraphResolver(100,
	&RequiredTaskLabelGraphResolverRule{},
	&TaskDependencyGraphResolverRule{},
	&SubsequentTaskRefsGraphResolverRule{},
)

// GraphResolverRuleResult represents the result of a single GraphResolverRule execution.
type GraphResolverRuleResult struct {
	// Changed indicates whether the rule modified the task list.
	Changed bool
	// Tasks is the updated list of tasks after the rule has been applied.
	Tasks []UntypedTask
}

// GraphResolverRule provides a rule to change the list of tasks included in the task graph from the given mandatory tasks.
type GraphResolverRule interface {
	// Name returns the name of the rule.
	Name() string
	// Resolve applies the rule to the current task graph, potentially adding, modifying tasks or removing tasks.
	Resolve(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error)
}

// GraphResolver iteratively applies a set of rules to determine the final set of tasks for the task graph.
// It starts with a set of required tasks and expands it based on the rules until a stable state is reached.
type GraphResolver struct {
	// Rules is the list of rules to be applied in each iteration.
	Rules []GraphResolverRule
	// MaxIteration is the maximum number of iterations to perform before considering the resolution failed.
	MaxIteration int
}

// NewGraphResolver returns a new instance of GraphResolver with given rules and configurations.
func NewGraphResolver(maxIteration int, rules ...GraphResolverRule) *GraphResolver {
	return &GraphResolver{
		Rules:        rules,
		MaxIteration: maxIteration,
	}
}

// Resolve determines the final set of tasks for the task graph.
// It iteratively applies the configured rules, starting with the initial `requiredTasks`.
// The process continues until no more changes are made to the task list (a stable state)
// or the `MaxIteration` limit is reached.
func (r *GraphResolver) Resolve(requiredTasks []UntypedTask, availableTasks []UntypedTask) ([]UntypedTask, error) {
	currentTasks := slices.Clone(requiredTasks)
	for iter := 0; iter < r.MaxIteration; iter++ {
		stabled := true
		for _, rule := range r.Rules {
			result, err := rule.Resolve(currentTasks, availableTasks)
			if err != nil {
				return nil, fmt.Errorf("failed to call Resolve function for the rule %s\n%s", rule.Name(), err.Error())
			}
			if result.Changed {
				stabled = false
			}
			currentTasks = result.Tasks
		}
		if stabled {
			return currentTasks, nil
		}
	}
	return nil, fmt.Errorf("failed to complete the resolution of tasks included in the task graph in given iteration count %d ", r.MaxIteration)
}

// RequiredTaskLabelGraphResolverRule is a resolver rule that adds tasks to the graph
// if they have the `LabelKeyRequiredTask` label set to true.
type RequiredTaskLabelGraphResolverRule struct{}

// Name implements GraphResolverRule.
func (r *RequiredTaskLabelGraphResolverRule) Name() string {
	return "require-task-label"
}

// Resolve implements GraphResolverRule.
func (r *RequiredTaskLabelGraphResolverRule) Resolve(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error) {
	taskMap, err := getMapOfTaskIDToUntypedTask(currentGraphTasks)
	if err != nil {
		return GraphResolverRuleResult{}, err
	}

	result := GraphResolverRuleResult{
		Tasks:   currentGraphTasks,
		Changed: false,
	}

	for _, task := range availableTasks {
		tid := task.UntypedID().String()
		if _, found := taskMap[tid]; found {
			continue
		}
		if required, found := typedmap.Get(task.Labels(), LabelKeyRequiredTask); required && found {
			result.Tasks = append(result.Tasks, task)
			result.Changed = true
		}
	}
	return result, nil
}

var _ GraphResolverRule = (*RequiredTaskLabelGraphResolverRule)(nil)

// TaskDependencyGraphResolverRule is a resolver rule that adds tasks to the graph
// to satisfy the dependencies of tasks already in the graph.
type TaskDependencyGraphResolverRule struct {
}

// Name implements GraphResolverRule.
func (d *TaskDependencyGraphResolverRule) Name() string {
	return "dependency"
}

// Resolve adds tasks from the available pool to satisfy unmet dependencies of tasks
// currently in the graph. If multiple tasks can satisfy a single dependency, the one
// with the highest priority is chosen. It returns an error if a dependency cannot be resolved.
func (d *TaskDependencyGraphResolverRule) Resolve(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error) {
	inclduedTaskReferences := getMapOfReferenceIDs(currentGraphTasks)

	missingReferences := make(map[string]taskid.UntypedTaskReference)
	for _, task := range currentGraphTasks {
		for _, dependency := range task.Dependencies() {
			refID := dependency.ReferenceIDString()
			if _, found := inclduedTaskReferences[refID]; !found {
				missingReferences[refID] = dependency
			}
		}
	}

	result := GraphResolverRuleResult{
		Tasks:   currentGraphTasks,
		Changed: false,
	}

	if len(missingReferences) > 0 {
		result.Changed = true
		for _, ref := range missingReferences {
			highest, err := findHighestPriorityUntypedTaskForTaskReference(ref, availableTasks)
			if err != nil {
				return GraphResolverRuleResult{}, err
			}
			result.Tasks = append(result.Tasks, highest)
		}
	}
	return result, nil
}

var _ GraphResolverRule = (*TaskDependencyGraphResolverRule)(nil)

// dependencyOverridenUntypedTask wraps an existing UntypedTask to dynamically override its dependencies.
// This is used by SubsequentTaskRefsGraphResolverRule to ensure that a subsequent task
// correctly depends on the task that triggered its addition to the graph.
type dependencyOverridenUntypedTask struct {
	Parent                 UntypedTask
	AdditionalDependencies []taskid.UntypedTaskReference
}

// newDependencyOverridenUntypedTask creates a new instance of dependencyOverridenUntypedTask,
// wrapping the provided parent task.
func newDependencyOverridenUntypedTask(parent UntypedTask) *dependencyOverridenUntypedTask {
	return &dependencyOverridenUntypedTask{
		Parent:                 parent,
		AdditionalDependencies: make([]taskid.UntypedTaskReference, 0),
	}
}

// AddDependency adds a new task reference to the list of additional dependencies, ignoring duplicates.
// It returns true if the dependency was added, and false if it already existed.
func (d *dependencyOverridenUntypedTask) AddDependency(newTaskRef taskid.UntypedTaskReference) bool {
	for _, ref := range d.Dependencies() {
		if ref.ReferenceIDString() == newTaskRef.ReferenceIDString() {
			return false
		}
	}
	d.AdditionalDependencies = append(d.AdditionalDependencies, newTaskRef)
	return true
}

// Dependencies implements UntypedTask by returning the parent's dependencies merged with any additional dependencies.
func (d *dependencyOverridenUntypedTask) Dependencies() []taskid.UntypedTaskReference {
	return append(d.Parent.Dependencies(), d.AdditionalDependencies...)
}

// Labels implements UntypedTask by delegating to the parent task.
func (d *dependencyOverridenUntypedTask) Labels() *typedmap.ReadonlyTypedMap {
	return d.Parent.Labels()
}

// UntypedID implements UntypedTask by delegating to the parent task.
func (d *dependencyOverridenUntypedTask) UntypedID() taskid.UntypedTaskImplementationID {
	return d.Parent.UntypedID()
}

// UntypedRun implements UntypedTask by delegating to the parent task.
func (d *dependencyOverridenUntypedTask) UntypedRun(ctx context.Context) (any, error) {
	return d.Parent.UntypedRun(ctx)
}

var _ UntypedTask = (*dependencyOverridenUntypedTask)(nil)

// SubsequentTaskRefsGraphResolverRule is a resolver rule that adds tasks to the graph
// based on the `LabelKeySubsequentTaskRefs` label of tasks already in the graph.
// This rule won't resolve dependencies of tasks added as the subsequent task. This must be used with the `dependency` resolver rule.
type SubsequentTaskRefsGraphResolverRule struct {
}

// Name implements GraphResolverRule.
func (s *SubsequentTaskRefsGraphResolverRule) Name() string {
	return "subsequent-task-label"
}

// Resolve ensures that subsequent tasks specified by the `LabelKeySubsequentTaskRefs` label
// are included in the graph. It dynamically updates dependencies to ensure that the
// subsequent task runs after the task that requested it.
func (s *SubsequentTaskRefsGraphResolverRule) Resolve(currentGraphTasks []UntypedTask, availableTasks []UntypedTask) (GraphResolverRuleResult, error) {
	result := GraphResolverRuleResult{
		Changed: false,
		Tasks:   currentGraphTasks,
	}
	missingSubsequentTaskReferences := make(map[string]taskid.UntypedTaskReference)
	missingSubsequentTaskReqquestedBy := make(map[string][]UntypedTask)
	for _, task := range currentGraphTasks {
		taskRefs := typedmap.GetOrDefault(task.Labels(), LabelKeySubsequentTaskRefs, []taskid.UntypedTaskReference{})
		for _, subsequentTaskRef := range taskRefs {
			found := false
			// try finding subsequent tasks from already included tasks
			for i, t := range currentGraphTasks {
				if t.UntypedID().ReferenceIDString() == subsequentTaskRef.ReferenceIDString() {
					if _, isDependencyOverridable := t.(*dependencyOverridenUntypedTask); !isDependencyOverridable {
						currentGraphTasks[i] = newDependencyOverridenUntypedTask(t)
						t = currentGraphTasks[i]
					}
					if t.(*dependencyOverridenUntypedTask).AddDependency(task.UntypedID().GetUntypedReference()) {
						result.Changed = true
					}
					found = true
				}
			}
			if !found {
				missingSubsequentTaskReferences[subsequentTaskRef.ReferenceIDString()] = subsequentTaskRef
				missingSubsequentTaskReqquestedBy[subsequentTaskRef.ReferenceIDString()] = append(missingSubsequentTaskReqquestedBy[subsequentTaskRef.ReferenceIDString()], task)
			}
		}
	}

	for idStr, taskRef := range missingSubsequentTaskReferences {
		highest, err := findHighestPriorityUntypedTaskForTaskReference(taskRef, availableTasks)
		if err != nil {
			return GraphResolverRuleResult{}, err
		}
		overridenHighest := newDependencyOverridenUntypedTask(highest)
		for _, requestedSuccessor := range missingSubsequentTaskReqquestedBy[idStr] {
			overridenHighest.AddDependency(requestedSuccessor.UntypedID().GetUntypedReference())
		}
		result.Tasks = append(result.Tasks, overridenHighest)
		result.Changed = true
	}
	return result, nil
}

var _ GraphResolverRule = (*SubsequentTaskRefsGraphResolverRule)(nil)

// getMapOfTaskIDToUntypedTask creates a map from task ID string to UntypedTask.
// It returns an error if duplicate task IDs are found.
func getMapOfTaskIDToUntypedTask(tasks []UntypedTask) (map[string]UntypedTask, error) {
	includedTaskIDs := map[string]UntypedTask{}
	for _, task := range tasks {
		tid := task.UntypedID().String()
		if _, found := includedTaskIDs[tid]; found {
			return nil, fmt.Errorf("getMapOfTaskIDToUntypedTask: failed to generate map of taskIDs. multiple tasks with task ID '%s' found", tid)
		}
		includedTaskIDs[tid] = task
	}
	return includedTaskIDs, nil
}

// getMapOfReferenceIDs creates a map from a task's reference ID string to its UntypedTaskReference.
// This map is used to quickly check if a task satisfying a certain reference is already in the graph.
// It safely ignores duplicate references, as multiple task implementations can satisfy the same reference.
func getMapOfReferenceIDs(tasks []UntypedTask) map[string]taskid.UntypedTaskReference {
	taskReferenceMap := map[string]taskid.UntypedTaskReference{}
	for _, task := range tasks {
		refID := task.UntypedID().ReferenceIDString()
		// A task graph can contain tasks shareing a same task reference. Duplication is safely ignored.
		taskReferenceMap[refID] = task.UntypedID().GetUntypedReference()
	}
	return taskReferenceMap
}

// findHighestPriorityUntypedTaskForTaskReference searches the list of available tasks for all tasks
// matching the given reference and returns the one with the highest `LabelKeyTaskSelectionPriority`.
// It returns an error if no matching task is found.
func findHighestPriorityUntypedTaskForTaskReference(ref taskid.UntypedTaskReference, availableTasks []UntypedTask) (UntypedTask, error) {
	var matched []UntypedTask
	for _, candidateTask := range availableTasks {
		candidateRefID := candidateTask.UntypedID().GetUntypedReference()
		if ref.ReferenceIDString() == candidateRefID.ReferenceIDString() {
			matched = append(matched, candidateTask)
		}
	}

	if len(matched) == 0 {
		var availableTaskIDs []string
		for _, task := range availableTasks {
			availableTaskIDs = append(availableTaskIDs, "*"+task.UntypedID().String())
		}
		return nil, fmt.Errorf("failed to resolve task dependency. No available task can be referenced as '%s'.\nAvailable tasks:\n%s", ref.ReferenceIDString(), strings.Join(availableTaskIDs, "\n"))
	}

	// pick one of matched task with the highest priority
	slices.SortFunc(matched, func(a, b UntypedTask) int {
		priorityA := typedmap.GetOrDefault(a.Labels(), LabelKeyTaskSelectionPriority, 0)
		priorityB := typedmap.GetOrDefault(b.Labels(), LabelKeyTaskSelectionPriority, 0)
		return priorityA - priorityB
	})

	return matched[len(matched)-1], nil
}
