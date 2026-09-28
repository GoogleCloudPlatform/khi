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
	"container/heap"
	"fmt"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// ResolveGraph resolves the task graph starting from initialTasks, drawing dependencies from availableTasks,
// and excluding disabledTasks.
// It applies a deterministic 5-phase resolution algorithm and returns a runnable TaskSet containing
// topologically sorted tasks and concrete TaskEdges.
func ResolveGraph(
	initialTasks []UntypedTask,
	availableTasks []UntypedTask,
	disabledTasks []UntypedTask,
) (*TaskSet, error) {
	availableTaskMap, disabledRefIDSet, err := verifyInput(initialTasks, availableTasks, disabledTasks)
	if err != nil {
		return nil, err
	}

	// --- Phase 1: Mandatory Closure ---
	graphTaskMap, err := resolveMandatoryClosure(initialTasks, availableTasks, availableTaskMap, disabledRefIDSet)
	if err != nil {
		return nil, err
	}

	// --- Phase 2: Active Feature Expansion & Fan-In Candidate Binding ---
	candidateFanInEdges, err := resolveActiveFeaturesAndCandidateFanInEdges(graphTaskMap, availableTasks, availableTaskMap, disabledRefIDSet)
	if err != nil {
		return nil, err
	}

	// --- Phase 3: Point-to-Point Edge Binding ---
	pointToPointEdges := resolvePointToPointEdges(graphTaskMap)

	// --- Phase 4: Fan-In Cycle Resolution & Edge Deduplication ---
	resolvedTasks, resolvedEdges, boundFanInRefIDsByTaskImplID, err := resolveFanInEdgesAndCycles(graphTaskMap, pointToPointEdges, candidateFanInEdges)
	if err != nil {
		return nil, err
	}

	// --- Phase 5: Topological Sorting (Kahn's Algorithm) ---
	return buildAndSortTaskSet(resolvedTasks, resolvedEdges, boundFanInRefIDsByTaskImplID), nil
}

// verifyInput validates the input tasks to ResolveGraph:
// 1. availableTasks must contain at most one implementation for each reference ID (no variants).
// 2. initialTasks must be a subset of availableTasks with matching implementation IDs.
// 3. disabledTasks must be a subset of availableTasks with matching implementation IDs.
// 4. No task in initialTasks can also be present in disabledTasks.
// It returns a map of reference ID to UntypedTask for availableTasks, and a set of disabled reference IDs.
func verifyInput(
	initialTasks []UntypedTask,
	availableTasks []UntypedTask,
	disabledTasks []UntypedTask,
) (map[string]UntypedTask, map[string]struct{}, error) {
	availableTaskMap := make(map[string]UntypedTask, len(availableTasks))
	for _, t := range availableTasks {
		refID := t.UntypedID().ReferenceIDString()
		implID := t.UntypedID().String()
		if existing, exists := availableTaskMap[refID]; exists {
			return nil, nil, fmt.Errorf("available task %q has conflicting implementation %q for reference %q", implID, existing.UntypedID().String(), refID)
		}
		availableTaskMap[refID] = t
	}

	initialRefIDSet, err := verifyTaskSubset(initialTasks, availableTaskMap, "initial")
	if err != nil {
		return nil, nil, err
	}

	disabledRefIDSet, err := verifyTaskSubset(disabledTasks, availableTaskMap, "disabled")
	if err != nil {
		return nil, nil, err
	}

	for refID := range disabledRefIDSet {
		if _, isInitial := initialRefIDSet[refID]; isInitial {
			return nil, nil, fmt.Errorf("initial task %q is explicitly disabled", availableTaskMap[refID].UntypedID().String())
		}
	}

	return availableTaskMap, disabledRefIDSet, nil
}

// verifyTaskSubset verifies that all tasks in the given subset exist in availableTaskMap with matching implementation IDs and no duplicate references.
func verifyTaskSubset(
	tasks []UntypedTask,
	availableTaskMap map[string]UntypedTask,
	kind string,
) (map[string]struct{}, error) {
	refIDSet := make(map[string]struct{}, len(tasks))
	for _, t := range tasks {
		refID := t.UntypedID().ReferenceIDString()
		implID := t.UntypedID().String()
		if _, exists := refIDSet[refID]; exists {
			return nil, fmt.Errorf("%s task %q has duplicate reference %q", kind, implID, refID)
		}
		avail, exists := availableTaskMap[refID]
		if !exists {
			return nil, fmt.Errorf("%s task %q is not in available tasks", kind, implID)
		}
		if avail.UntypedID().String() != implID {
			return nil, fmt.Errorf("%s task %q has conflicting implementation %q in available tasks for reference %q", kind, implID, avail.UntypedID().String(), refID)
		}
		refIDSet[refID] = struct{}{}
	}
	return refIDSet, nil
}

// resolveMandatoryClosure computes the initial graph tasks by expanding mandatory point-to-point dependencies
// starting from initialTasks and system-required tasks in availableTasks.
func resolveMandatoryClosure(
	initialTasks []UntypedTask,
	availableTasks []UntypedTask,
	availableTaskMap map[string]UntypedTask,
	disabledRefIDSet map[string]struct{},
) (map[string]UntypedTask, error) {
	graphTaskMap := make(map[string]UntypedTask)
	queue := make([]UntypedTask, 0)

	// Collect initial tasks.
	for _, t := range initialTasks {
		refID := t.UntypedID().ReferenceIDString()
		graphTaskMap[refID] = t
		queue = append(queue, t)
	}

	// Collect system-required tasks (LabelKeyRequiredTask).
	for _, t := range availableTasks {
		if req, found := typedmap.Get(t.Labels(), LabelKeyRequiredTask); found && req {
			refID := t.UntypedID().ReferenceIDString()
			if _, isDisabled := disabledRefIDSet[refID]; isDisabled {
				return nil, fmt.Errorf("required task %q is explicitly disabled", t.UntypedID())
			}
			if _, exists := graphTaskMap[refID]; !exists {
				graphTaskMap[refID] = t
				queue = append(queue, t)
			}
		}
	}

	// Expand mandatory point-to-point dependency closure.
	if err := expandMandatoryDependencies(queue, graphTaskMap, availableTaskMap, disabledRefIDSet); err != nil {
		return nil, err
	}

	return graphTaskMap, nil
}

// expandMandatoryDependencies expands the mandatory point-to-point dependency closure for tasks in queue.
func expandMandatoryDependencies(
	queue []UntypedTask,
	graphTaskMap map[string]UntypedTask,
	availableTaskMap map[string]UntypedTask,
	disabledRefIDSet map[string]struct{},
) error {
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, dep := range curr.Dependencies() {
			if dep.DescriptorScope() == taskid.ScopeAll && dep.DescriptorCardinality() == taskid.CardinalityPointToPoint {
				ptp, ok := dep.(taskid.PointToPointDescriptor)
				if !ok {
					continue
				}
				refID := ptp.ReferenceID()
				if _, exists := graphTaskMap[refID]; !exists {
					if _, isDisabled := disabledRefIDSet[refID]; isDisabled {
						return fmt.Errorf("required dependency %q required by %q is disabled", refID, curr.UntypedID())
					}
					targetTask, ok := availableTaskMap[refID]
					if !ok {
						return fmt.Errorf("required dependency %q required by %q not found", refID, curr.UntypedID())
					}
					graphTaskMap[refID] = targetTask
					queue = append(queue, targetTask)
				}
			}
		}
	}
	return nil
}

// isFeatureGateActive checks whether candidate is allowed to be pulled into the active graph
// under ScopeActiveFeatures. If candidate has no LabelKeyFeatureGateTaskRef label, it is
// ungated and always allowed. Otherwise, its feature gate task reference must be present in graphTaskMap.
func isFeatureGateActive(candidate UntypedTask, graphTaskMap map[string]UntypedTask) bool {
	gateRef, found := typedmap.Get(candidate.Labels(), LabelKeyFeatureGateTaskRef)
	if !found || gateRef == nil {
		return true
	}
	_, inGraph := graphTaskMap[gateRef.ReferenceIDString()]
	return inGraph
}

// resolveActiveFeaturesAndCandidateFanInEdges resolves candidate fan-in edges and conditionally expands
// active feature tasks (both point-to-point and fan-in) for all tasks in graphTaskMap.
// First, it expands graphTaskMap to a fixed point.
// Second, once graphTaskMap has reached its fixed point, it collects rawEdges for all CardinalityFanIn dependencies.
func resolveActiveFeaturesAndCandidateFanInEdges(
	graphTaskMap map[string]UntypedTask,
	availableTasks []UntypedTask,
	availableTaskMap map[string]UntypedTask,
	disabledRefIDSet map[string]struct{},
) ([]taskid.TaskEdge, error) {
	// 1. Expand graphTaskMap to a fixed point.
	for {
		sizeBefore := len(graphTaskMap)
		tasks := make([]UntypedTask, 0, sizeBefore)
		for _, t := range graphTaskMap {
			tasks = append(tasks, t)
		}
		slices.SortFunc(tasks, compareTaskByImplementationID)

		for _, t := range tasks {
			if err := expandActiveFeaturePointToPointDependencies(t, availableTaskMap, graphTaskMap, disabledRefIDSet); err != nil {
				return nil, err
			}
			if err := expandActiveFeatureFanInProducers(t, availableTasks, availableTaskMap, graphTaskMap, disabledRefIDSet); err != nil {
				return nil, err
			}
		}

		if len(graphTaskMap) == sizeBefore {
			break
		}
	}

	// 2. Collect fan-in candidate edges once graphTaskMap has reached its fixed point.
	tasks := make([]UntypedTask, 0, len(graphTaskMap))
	for _, t := range graphTaskMap {
		tasks = append(tasks, t)
	}
	slices.SortFunc(tasks, compareTaskByImplementationID)

	var rawEdges []taskid.TaskEdge
	for _, task := range tasks {
		for _, dep := range task.Dependencies() {
			if dep.DescriptorCardinality() != taskid.CardinalityFanIn {
				continue
			}
			fanInDep, ok := dep.(taskid.FanInDescriptor)
			if !ok {
				continue
			}

			tag := fanInDep.Tag()
			var matchingProducers []UntypedTask
			switch fanInDep.DescriptorScope() {
			case taskid.ScopeActiveFeatures, taskid.ScopeActiveGraph:
				matchingProducers = findActiveGraphTasksByTag(tag, graphTaskMap)
			case taskid.ScopeAll:
				return nil, fmt.Errorf("ScopeAll is not supported for fan-in dependency on tag %q", tag)
			default:
				return nil, fmt.Errorf("unknown or unsupported dependency scope: %v", fanInDep.DescriptorScope())
			}

			for _, p := range matchingProducers {
				priority := typedmap.GetOrDefault(p.Labels(), LabelKeyProvidedTagPriority(tag), DefaultTagPriority)
				tagType := typedmap.GetOrDefault(p.Labels(), LabelKeyProvidedTagType(tag), "")
				if tagType == "" && p.ResultType() != nil {
					tagType = p.ResultType().String()
				}
				rawEdges = append(rawEdges, taskid.TaskEdge{
					SourceRefID:  p.UntypedID().ReferenceIDString(),
					SourceImplID: p.UntypedID().String(),
					TargetImplID: task.UntypedID().String(),
					Cardinality:  taskid.CardinalityFanIn,
					Tag:          tag,
					Priority:     priority,
					OutputType:   tagType,
				})
			}
		}
	}

	slices.SortFunc(rawEdges, compareTaskEdge)
	return rawEdges, nil
}

// expandActiveFeaturePointToPointDependencies expands any point-to-point dependencies of task
// configured with ScopeActiveFeatures if their feature gate task is present in the active graph.
func expandActiveFeaturePointToPointDependencies(
	task UntypedTask,
	availableTaskMap map[string]UntypedTask,
	graphTaskMap map[string]UntypedTask,
	disabledRefIDSet map[string]struct{},
) error {
	for _, dep := range task.Dependencies() {
		if dep.DescriptorCardinality() != taskid.CardinalityPointToPoint || dep.DescriptorScope() != taskid.ScopeActiveFeatures {
			continue
		}
		ptp, ok := dep.(taskid.PointToPointDescriptor)
		if !ok {
			continue
		}
		refID := ptp.ReferenceID()
		if _, inGraph := graphTaskMap[refID]; inGraph {
			continue
		}
		targetTask, ok := availableTaskMap[refID]
		if !ok {
			continue
		}
		if _, isDisabled := disabledRefIDSet[refID]; isDisabled {
			continue
		}
		if !isFeatureGateActive(targetTask, graphTaskMap) {
			continue
		}

		graphTaskMap[refID] = targetTask
		if err := expandMandatoryDependencies([]UntypedTask{targetTask}, graphTaskMap, availableTaskMap, disabledRefIDSet); err != nil {
			return err
		}
	}
	return nil
}

// expandActiveFeatureFanInProducers expands any fan-in dependencies of task configured with ScopeActiveFeatures.
func expandActiveFeatureFanInProducers(
	task UntypedTask,
	availableTasks []UntypedTask,
	availableTaskMap map[string]UntypedTask,
	graphTaskMap map[string]UntypedTask,
	disabledRefIDSet map[string]struct{},
) error {
	for _, dep := range task.Dependencies() {
		if dep.DescriptorCardinality() != taskid.CardinalityFanIn || dep.DescriptorScope() != taskid.ScopeActiveFeatures {
			continue
		}
		fanInDep, ok := dep.(taskid.FanInDescriptor)
		if !ok {
			continue
		}
		if err := findAndConditionallyExpandActiveFeatureTasks(fanInDep.Tag(), availableTasks, availableTaskMap, graphTaskMap, disabledRefIDSet); err != nil {
			return err
		}
	}
	return nil
}

// findActiveGraphTasksByTag returns all tasks currently in graphTaskMap that provide the given tag.
func findActiveGraphTasksByTag(tag string, graphTaskMap map[string]UntypedTask) []UntypedTask {
	var matching []UntypedTask
	for _, t := range graphTaskMap {
		if slices.Contains(getProvidedTags(t), tag) {
			matching = append(matching, t)
		}
	}
	slices.SortFunc(matching, compareTaskByImplementationID)
	return matching
}

// compareTaskEdge compares two task edges deterministically by SourceImplID, TargetImplID, Tag, and Priority.
func compareTaskEdge(a, b taskid.TaskEdge) int {
	if c := strings.Compare(a.SourceImplID, b.SourceImplID); c != 0 {
		return c
	}
	if c := strings.Compare(a.TargetImplID, b.TargetImplID); c != 0 {
		return c
	}
	if c := strings.Compare(a.Tag, b.Tag); c != 0 {
		return c
	}
	return a.Priority - b.Priority
}

// resolvePointToPointEdges creates point-to-point edges for all tasks in graphTaskMap
// whose source (upstream dependency) exists in the graph (both required and optional).
func resolvePointToPointEdges(graphTaskMap map[string]UntypedTask) []taskid.TaskEdge {
	tasks := make([]UntypedTask, 0, len(graphTaskMap))
	for _, t := range graphTaskMap {
		tasks = append(tasks, t)
	}
	slices.SortFunc(tasks, compareTaskByImplementationID)

	var rawEdges []taskid.TaskEdge
	for _, t := range tasks {
		for _, dep := range t.Dependencies() {
			if dep.DescriptorCardinality() == taskid.CardinalityPointToPoint {
				ptp, ok := dep.(taskid.PointToPointDescriptor)
				if !ok {
					continue
				}
				refID := ptp.ReferenceID()
				if sourceTask, exists := graphTaskMap[refID]; exists {
					outputType := ""
					if sourceTask.ResultType() != nil {
						outputType = sourceTask.ResultType().String()
					}
					rawEdges = append(rawEdges, taskid.TaskEdge{
						SourceRefID:  refID,
						SourceImplID: sourceTask.UntypedID().String(),
						TargetImplID: t.UntypedID().String(),
						Cardinality:  taskid.CardinalityPointToPoint,
						OutputType:   outputType,
					})
				}
			}
		}
	}
	return rawEdges
}

// deduplicateAndNormalizeEdges merges duplicate edges between the same source and target.
// Minimum Priority takes precedence.
func deduplicateAndNormalizeEdges(rawEdges []taskid.TaskEdge) []taskid.TaskEdge {
	type edgeKey struct {
		sourceImplID string
		targetImplID string
	}
	edgeMap := make(map[edgeKey]taskid.TaskEdge)
	order := make([]edgeKey, 0, len(rawEdges))

	for _, e := range rawEdges {
		key := edgeKey{sourceImplID: e.SourceImplID, targetImplID: e.TargetImplID}
		if existing, exists := edgeMap[key]; exists {
			if existing.Priority == 0 || (e.Priority > 0 && e.Priority < existing.Priority) {
				existing.Priority = e.Priority
			}
			if existing.Tag == "" && e.Tag != "" {
				existing.Tag = e.Tag
			}
			if existing.OutputType == "" && e.OutputType != "" {
				existing.OutputType = e.OutputType
			}
			if existing.Cardinality == taskid.CardinalityPointToPoint || e.Cardinality == taskid.CardinalityPointToPoint {
				existing.Cardinality = taskid.CardinalityPointToPoint
			}
			edgeMap[key] = existing
		} else {
			edgeMap[key] = e
			order = append(order, key)
		}
	}

	deduped := make([]taskid.TaskEdge, 0, len(order))
	for _, k := range order {
		deduped = append(deduped, edgeMap[k])
	}
	return deduped
}

// buildAndSortTaskSet executes Kahn's algorithm with a min-heap for deterministic ordering.
func buildAndSortTaskSet(
	tasks []UntypedTask,
	edges []taskid.TaskEdge,
	boundFanInRefIDsByTaskImplID map[string]map[string][]string,
) *TaskSet {
	inDegree := make(map[string]int, len(tasks))
	outgoing := make(map[string][]string) // key: source task implementation ID -> []target task implementation ID

	// Index tasks by ImplementationID for O(1) ready queue push.
	implToTask := make(map[string]UntypedTask, len(tasks))
	for _, task := range tasks {
		implID := task.UntypedID().String()
		inDegree[implID] = 0
		implToTask[implID] = task
	}

	for _, e := range edges {
		outgoing[e.SourceImplID] = append(outgoing[e.SourceImplID], e.TargetImplID)
		inDegree[e.TargetImplID]++
	}

	h := &taskMinHeap{}
	heap.Init(h)
	for _, task := range tasks {
		if inDegree[task.UntypedID().String()] == 0 {
			heap.Push(h, task)
		}
	}

	sortedTasks := make([]UntypedTask, 0, len(tasks))
	for h.Len() > 0 {
		curr := heap.Pop(h).(UntypedTask)
		sortedTasks = append(sortedTasks, curr)

		for _, targetImplID := range outgoing[curr.UntypedID().String()] {
			inDegree[targetImplID]--
			if inDegree[targetImplID] == 0 {
				if task, ok := implToTask[targetImplID]; ok {
					heap.Push(h, task)
				}
			}
		}
	}

	return NewResolvedTaskSet(sortedTasks, edges, boundFanInRefIDsByTaskImplID)
}

// getProvidedTags extracts all tag IDs that the task provides.
func getProvidedTags(task UntypedTask) []string {
	var tags []string
	for _, key := range task.Labels().Keys() {
		if strings.HasPrefix(key, LabelKeyProvidedTagPrefix) {
			provided := typedmap.GetOrDefault(task.Labels(), typedmap.NewTypedKey[bool](key), false)
			if provided {
				tags = append(tags, strings.TrimPrefix(key, LabelKeyProvidedTagPrefix))
			}
		}
	}
	slices.Sort(tags)
	return tags
}

// findAndConditionallyExpandActiveFeatureTasks expands candidate producers for ScopeActiveFeatures.
func findAndConditionallyExpandActiveFeatureTasks(
	tag string,
	availableTasks []UntypedTask,
	availableTaskMap map[string]UntypedTask,
	graphTaskMap map[string]UntypedTask,
	disabledRefIDSet map[string]struct{},
) error {
	for _, t := range availableTasks {
		refID := t.UntypedID().ReferenceIDString()
		if _, isDisabled := disabledRefIDSet[refID]; isDisabled {
			continue
		}
		if !slices.Contains(getProvidedTags(t), tag) {
			continue
		}

		if _, inGraph := graphTaskMap[refID]; inGraph {
			continue
		}

		if !isFeatureGateActive(t, graphTaskMap) {
			continue
		}

		graphTaskMap[refID] = t
		if err := expandMandatoryDependencies([]UntypedTask{t}, graphTaskMap, availableTaskMap, disabledRefIDSet); err != nil {
			return err
		}
	}
	return nil
}

// compareTaskByImplementationID compares two UntypedTasks lexicographically by their implementation ID string.
func compareTaskByImplementationID(a, b UntypedTask) int {
	return strings.Compare(a.UntypedID().String(), b.UntypedID().String())
}
