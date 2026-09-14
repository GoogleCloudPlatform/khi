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

package coreinspection

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/filter"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"google.golang.org/protobuf/proto"
)

// ErrInspectionTypeNotFound indicates that the requested inspection type is not registered on the server.
var ErrInspectionTypeNotFound = errors.New("inspection type not found")

// InspectRegistry inspects the given InspectionTaskServer and returns all registered tasks
// grouped by reference identifier, as well as all registered inspection types.
func InspectRegistry(server *InspectionTaskServer) (*apiv1.GetInspectionTaskRegistryResponse, error) {
	allTasks := server.GetAllRegisteredTasks()

	// Group tasks by their task reference ID.
	tasksByRef := make(map[string][]coretask.UntypedTask)
	for _, task := range allTasks {
		refID := task.UntypedID().ReferenceIDString()
		tasksByRef[refID] = append(tasksByRef[refID], task)
	}

	// Sort reference IDs alphabetically for deterministic output.
	refIDs := make([]string, 0, len(tasksByRef))
	for refID := range tasksByRef {
		refIDs = append(refIDs, refID)
	}
	slices.Sort(refIDs)

	taskGroups := make([]*apiv1.RegisteredTaskGroupInfo, 0, len(refIDs))
	for _, refID := range refIDs {
		tasksInRef := tasksByRef[refID]
		// Sort tasks within the group: highest priority first, then implementation ID alphabetically.
		slices.SortFunc(tasksInRef, func(a, b coretask.UntypedTask) int {
			priorityA := typedmap.GetOrDefault(a.Labels(), coretask.LabelKeyTaskSelectionPriority, 0)
			priorityB := typedmap.GetOrDefault(b.Labels(), coretask.LabelKeyTaskSelectionPriority, 0)
			if priorityA != priorityB {
				return priorityB - priorityA
			}
			return strings.Compare(a.UntypedID().String(), b.UntypedID().String())
		})

		registeredTasks := make([]*apiv1.RegisteredTaskInfo, 0, len(tasksInRef))
		for _, task := range tasksInRef {
			registeredTasks = append(registeredTasks, convertToRegisteredTaskInfo(task))
		}

		taskGroups = append(taskGroups, &apiv1.RegisteredTaskGroupInfo{
			TaskReferenceId: proto.String(refID),
			Tasks:           registeredTasks,
		})
	}

	inspectionTypes := server.GetAllInspectionTypes()
	registeredTypes := make([]*apiv1.RegisteredInspectionTypeInfo, 0, len(inspectionTypes))
	for _, it := range inspectionTypes {
		registeredTypes = append(registeredTypes, &apiv1.RegisteredInspectionTypeInfo{
			Id:          proto.String(it.Id),
			Name:        proto.String(it.Name),
			Description: proto.String(it.Description),
			Icon:        proto.String(it.Icon),
			Labels:      maps.Clone(it.Labels),
		})
	}

	return &apiv1.GetInspectionTaskRegistryResponse{
		TaskGroups:      taskGroups,
		InspectionTypes: registeredTypes,
	}, nil
}

// InspectResolution evaluates task filtering against an inspection type and feature configuration,
// and resolves the execution DAG.
func InspectResolution(
	server *InspectionTaskServer,
	inspectionTypeID string,
	featureOverrides map[string]bool,
) (*apiv1.ResolveInspectionTaskGraphResponse, error) {
	currentType := server.GetInspectionType(inspectionTypeID)
	if currentType == nil {
		return nil, fmt.Errorf("%w: %s", ErrInspectionTypeNotFound, inspectionTypeID)
	}

	filterEvals, availableTaskSet, err := evaluateCandidateTasks(server.GetAllRegisteredTasks(), currentType)
	if err != nil {
		return nil, err
	}

	availableFeatures, initialTasks, disabledTasks := resolveFeatureConfiguration(availableTaskSet, featureOverrides)
	dagInfo := buildTaskDAGInfo(initialTasks, availableTaskSet.GetAll(), disabledTasks)

	return &apiv1.ResolveInspectionTaskGraphResponse{
		FilteringEvaluations: filterEvals,
		Dag:                  dagInfo,
		AvailableFeatures:    availableFeatures,
	}, nil
}

// evaluateCandidateTasks filters and prioritizes registered tasks against the given inspection type.
func evaluateCandidateTasks(
	allTasks []coretask.UntypedTask,
	currentType *InspectionType,
) ([]*apiv1.TaskFilterEvaluation, *coretask.TaskSet, error) {
	// Sort tasks deterministically by RefID then Implementation ID.
	slices.SortFunc(allTasks, func(a, b coretask.UntypedTask) int {
		if cmp := strings.Compare(a.UntypedID().ReferenceIDString(), b.UntypedID().ReferenceIDString()); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.UntypedID().String(), b.UntypedID().String())
	})

	type evaluatedTask struct {
		task        coretask.UntypedTask
		compatible  bool
		matchReason string
	}

	evaluations := make([]evaluatedTask, 0, len(allTasks))
	var compatibleTasks []coretask.UntypedTask
	for _, task := range allTasks {
		compat, reason := EvaluateTaskCompatibility(task, currentType)
		evaluations = append(evaluations, evaluatedTask{
			task:        task,
			compatible:  compat,
			matchReason: reason,
		})
		if compat {
			compatibleTasks = append(compatibleTasks, task)
		}
	}

	// Determine winners among compatible tasks using priority deduplication.
	winnerMap := make(map[string]string)
	dedupedTasks := deduplicateTasksByPriority(compatibleTasks)
	for _, winner := range dedupedTasks {
		winnerMap[winner.UntypedID().ReferenceIDString()] = winner.UntypedID().String()
	}

	filterEvals := make([]*apiv1.TaskFilterEvaluation, 0, len(evaluations))
	for _, ev := range evaluations {
		task := ev.task
		priority := int32(typedmap.GetOrDefault(task.Labels(), coretask.LabelKeyTaskSelectionPriority, 0))
		refID := task.UntypedID().ReferenceIDString()
		implID := task.UntypedID().String()

		var isSelected bool
		var supersededBy string
		if ev.compatible {
			winnerID := winnerMap[refID]
			if implID == winnerID {
				isSelected = true
			} else {
				isSelected = false
				supersededBy = winnerID
			}
		}

		filterEvals = append(filterEvals, &apiv1.TaskFilterEvaluation{
			TaskImplementationId:             proto.String(implID),
			TaskReferenceId:                  proto.String(refID),
			IsCompatible:                     proto.Bool(ev.compatible),
			MatchReason:                      proto.String(ev.matchReason),
			IsSelected:                       proto.Bool(isSelected),
			SupersededByTaskImplementationId: proto.String(supersededBy),
			Priority:                         proto.Int32(priority),
		})
	}

	availableTaskSet, err := coretask.NewTaskSet(dedupedTasks)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build available task set: %w", err)
	}

	return filterEvals, availableTaskSet, nil
}

// resolveFeatureConfiguration resolves feature toggle states and partitions tasks into initial and disabled sets.
func resolveFeatureConfiguration(
	availableTaskSet *coretask.TaskSet,
	featureOverrides map[string]bool,
) ([]*apiv1.FeatureToggleInfo, []coretask.UntypedTask, []coretask.UntypedTask) {
	featureSet := coretask.Subset(availableTaskSet, filter.NewEnabledFilter(inspectioncore_contract.LabelKeyInspectionFeatureFlag, false))
	defaultFeatures := coretask.Subset(availableTaskSet, filter.NewEnabledFilter(inspectioncore_contract.LabelKeyInspectionDefaultFeatureFlag, false))

	enabledFeatures := make(map[string]bool)
	for _, task := range defaultFeatures.GetAll() {
		enabledFeatures[task.UntypedID().String()] = true
	}
	for featureID, enabled := range featureOverrides {
		enabledFeatures[featureID] = enabled
	}

	allFeatureTasks := featureSet.GetAll()
	slices.SortFunc(allFeatureTasks, func(a, b coretask.UntypedTask) int {
		orderA := typedmap.GetOrDefault(a.Labels(), inspectioncore_contract.LabelKeyFeatureTaskOrder, DefaultFeatureTaskOrder)
		orderB := typedmap.GetOrDefault(b.Labels(), inspectioncore_contract.LabelKeyFeatureTaskOrder, DefaultFeatureTaskOrder)
		if orderA != orderB {
			return orderA - orderB
		}
		return strings.Compare(a.UntypedID().String(), b.UntypedID().String())
	})

	availableFeatures := make([]*apiv1.FeatureToggleInfo, 0, len(allFeatureTasks))
	for _, ft := range allFeatureTasks {
		label := typedmap.GetOrDefault(ft.Labels(), inspectioncore_contract.LabelKeyFeatureTaskTitle, fmt.Sprintf("No label Set!(%s)", ft.UntypedID()))
		description := typedmap.GetOrDefault(ft.Labels(), inspectioncore_contract.LabelKeyFeatureTaskDescription, "")
		availableFeatures = append(availableFeatures, &apiv1.FeatureToggleInfo{
			TaskImplementationId: proto.String(ft.UntypedID().String()),
			Label:                proto.String(label),
			Description:          proto.String(description),
			Enabled:              proto.Bool(enabledFeatures[ft.UntypedID().String()]),
		})
	}

	var initialTasks []coretask.UntypedTask
	var disabledTasks []coretask.UntypedTask
	for _, ft := range allFeatureTasks {
		if enabledFeatures[ft.UntypedID().String()] {
			initialTasks = append(initialTasks, ft)
		} else {
			disabledTasks = append(disabledTasks, ft)
		}
	}

	return availableFeatures, initialTasks, disabledTasks
}

// buildTaskDAGInfo resolves the task dependency DAG and formats nodes and edges for visualization.
func buildTaskDAGInfo(
	initialTasks []coretask.UntypedTask,
	availableTasks []coretask.UntypedTask,
	disabledTasks []coretask.UntypedTask,
) *apiv1.TaskDAGInfo {
	resolvedTaskSet, err := coretask.ResolveGraph(initialTasks, availableTasks, disabledTasks)
	if err != nil {
		return &apiv1.TaskDAGInfo{
			IsSuccess:    proto.Bool(false),
			ErrorMessage: proto.String(err.Error()),
			Nodes:        []*apiv1.TaskDAGNode{},
			Edges:        []*apiv1.TaskDAGEdge{},
		}
	}
	return buildTaskDAGInfoFromTaskSet(resolvedTaskSet)
}

// buildTaskDAGInfoFromTaskSet formats nodes and edges of an already resolved TaskSet for visualization.
func buildTaskDAGInfoFromTaskSet(resolvedTaskSet *coretask.TaskSet) *apiv1.TaskDAGInfo {
	dagInfo := &apiv1.TaskDAGInfo{
		IsSuccess:    proto.Bool(true),
		ErrorMessage: proto.String(""),
	}

	resolvedTasks := resolvedTaskSet.GetAll()
	nodes := make([]*apiv1.TaskDAGNode, 0, len(resolvedTasks))
	for idx, t := range resolvedTasks {
		labels := t.Labels()
		priority := int32(typedmap.GetOrDefault(labels, coretask.LabelKeyTaskSelectionPriority, 0))
		isFeature := typedmap.GetOrDefault(labels, inspectioncore_contract.LabelKeyInspectionFeatureFlag, false)
		outputType := ""
		if t.ResultType() != nil {
			outputType = t.ResultType().String()
		}
		nodes = append(nodes, &apiv1.TaskDAGNode{
			TaskImplementationId: proto.String(t.UntypedID().String()),
			TaskReferenceId:      proto.String(t.UntypedID().ReferenceIDString()),
			IsFeature:            proto.Bool(isFeature),
			TopologicalOrder:     proto.Int32(int32(idx)),
			Priority:             proto.Int32(priority),
			Labels:               typedmap.ToStringMap(labels),
			OutputType:           proto.String(outputType),
			ProvidedTags:         extractProvidedTags(labels),
		})
	}
	dagInfo.Nodes = nodes

	resolvedEdges := resolvedTaskSet.Edges()
	edges := make([]*apiv1.TaskDAGEdge, 0, len(resolvedEdges))
	for _, edge := range resolvedEdges {
		edges = append(edges, &apiv1.TaskDAGEdge{
			SourceImplementationId:      proto.String(edge.SourceImplID),
			DestinationImplementationId: proto.String(edge.TargetImplID),
			SourceReferenceId:           proto.String(edge.SourceRefID),
			Cardinality:                 convertEdgeCardinality(edge.Cardinality),
			Tag:                         proto.String(edge.Tag),
			Priority:                    proto.Int32(int32(edge.Priority)),
			OutputType:                  proto.String(edge.OutputType),
		})
	}
	// Sort edges deterministically by source implementation, destination implementation, and tag.
	slices.SortFunc(edges, func(a, b *apiv1.TaskDAGEdge) int {
		if c := strings.Compare(a.GetSourceImplementationId(), b.GetSourceImplementationId()); c != 0 {
			return c
		}
		if c := strings.Compare(a.GetDestinationImplementationId(), b.GetDestinationImplementationId()); c != 0 {
			return c
		}
		return strings.Compare(a.GetTag(), b.GetTag())
	})
	dagInfo.Edges = edges

	return dagInfo
}

// EvaluateTaskCompatibility checks whether a task is compatible with the given inspection type
// and returns the compatibility status along with an explanatory diagnosis.
func EvaluateTaskCompatibility(task coretask.UntypedTask, currentType *InspectionType) (bool, string) {
	labels := task.Labels()

	if selector, ok := typedmap.Get(labels, inspectioncore_contract.LabelKeyInspectionTypeLabelSelector); ok {
		keys := make([]string, 0, len(selector))
		for k := range selector {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		var missingOrMismatched []string
		for _, k := range keys {
			wantV := selector[k]
			gotV, ok := currentType.Labels[k]
			if !ok {
				missingOrMismatched = append(missingOrMismatched, fmt.Sprintf("%s=%q (missing on inspection type)", k, wantV))
			} else if gotV != wantV {
				missingOrMismatched = append(missingOrMismatched, fmt.Sprintf("%s=%q (inspection type has %q)", k, wantV, gotV))
			}
		}

		if len(missingOrMismatched) == 0 {
			return true, fmt.Sprintf("Matched label selector (%s)", formatSelectorString(selector))
		}
		return false, fmt.Sprintf("Mismatched label selector: %s", strings.Join(missingOrMismatched, ", "))
	}

	return true, "Global task (compatible with all inspection types)"
}

// extractProvidedTags extracts provided tag definitions from task labels and returns them sorted by tag ID.
func extractProvidedTags(labels *typedmap.ReadonlyTypedMap) []*apiv1.ProvidedTagInfo {
	var tags []*apiv1.ProvidedTagInfo
	for _, key := range labels.Keys() {
		if strings.HasPrefix(key, coretask.LabelKeyProvidedTagPrefix) {
			tagID := strings.TrimPrefix(key, coretask.LabelKeyProvidedTagPrefix)
			priority := int32(typedmap.GetOrDefault(labels, coretask.LabelKeyProvidedTagPriority(tagID), coretask.DefaultTagPriority))
			outputType := typedmap.GetOrDefault(labels, coretask.LabelKeyProvidedTagType(tagID), "")
			tags = append(tags, &apiv1.ProvidedTagInfo{
				Tag:        proto.String(tagID),
				Priority:   proto.Int32(priority),
				OutputType: proto.String(outputType),
			})
		}
	}
	slices.SortFunc(tags, func(a, b *apiv1.ProvidedTagInfo) int {
		return strings.Compare(a.GetTag(), b.GetTag())
	})
	return tags
}

func convertToRegisteredTaskInfo(task coretask.UntypedTask) *apiv1.RegisteredTaskInfo {
	labels := task.Labels()
	priority := typedmap.GetOrDefault(labels, coretask.LabelKeyTaskSelectionPriority, 0)
	isFeature := typedmap.GetOrDefault(labels, inspectioncore_contract.LabelKeyInspectionFeatureFlag, false)
	isDefaultFeature := typedmap.GetOrDefault(labels, inspectioncore_contract.LabelKeyInspectionDefaultFeatureFlag, false)
	featureLabel := typedmap.GetOrDefault(labels, inspectioncore_contract.LabelKeyFeatureTaskTitle, "")
	featureDesc := typedmap.GetOrDefault(labels, inspectioncore_contract.LabelKeyFeatureTaskDescription, "")

	dependencies := convertTaskDependencies(task.Dependencies())

	selector, hasSelector := typedmap.Get(labels, inspectioncore_contract.LabelKeyInspectionTypeLabelSelector)
	var selectorRequirements []*apiv1.LabelSelectorRequirementInfo
	if hasSelector {
		selectorRequirements = convertSelectorRequirements(selector)
	}

	outputType := ""
	if task.ResultType() != nil {
		outputType = task.ResultType().String()
	}

	return &apiv1.RegisteredTaskInfo{
		TaskImplementationId: proto.String(task.UntypedID().String()),
		TaskReferenceId:      proto.String(task.UntypedID().ReferenceIDString()),
		Priority:             proto.Int32(int32(priority)),
		IsFeature:            proto.Bool(isFeature),
		IsDefaultFeature:     proto.Bool(isDefaultFeature),
		FeatureLabel:         proto.String(featureLabel),
		FeatureDescription:   proto.String(featureDesc),
		Dependencies:         dependencies,
		SelectorRequirements: selectorRequirements,
		Labels:               typedmap.ToStringMap(labels),
		OutputType:           proto.String(outputType),
		ProvidedTags:         extractProvidedTags(labels),
	}
}

func convertTaskDependencies(deps []coretask.Dependency) []*apiv1.TaskDependencyInfo {
	dependencies := make([]*apiv1.TaskDependencyInfo, 0, len(deps))
	for _, dep := range deps {
		depInfo := &apiv1.TaskDependencyInfo{
			Cardinality: convertEdgeCardinality(dep.DescriptorCardinality()),
			Scope:       convertDependencyScope(dep.DescriptorScope()),
		}
		if dep.ResultType() != nil {
			depInfo.OutputType = proto.String(dep.ResultType().String())
		}
		if p, ok := dep.(taskid.PointToPointDescriptor); ok {
			depInfo.TargetReferenceId = proto.String(p.ReferenceID())
		}
		if f, ok := dep.(taskid.FanInDescriptor); ok {
			depInfo.TargetTag = proto.String(f.Tag())
		}
		dependencies = append(dependencies, depInfo)
	}
	return dependencies
}

func convertSelectorRequirements(selector inspectioncore_contract.LabelSelector) []*apiv1.LabelSelectorRequirementInfo {
	keys := make([]string, 0, len(selector))
	for k := range selector {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	requirements := make([]*apiv1.LabelSelectorRequirementInfo, 0, len(keys))
	for _, k := range keys {
		requirements = append(requirements, &apiv1.LabelSelectorRequirementInfo{
			Key:      proto.String(k),
			Operator: proto.String("=="),
			Values:   []string{selector[k]},
		})
	}
	return requirements
}

func formatSelectorString(selector inspectioncore_contract.LabelSelector) string {
	keys := make([]string, 0, len(selector))
	for k := range selector {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, selector[k]))
	}
	return strings.Join(pairs, ", ")
}

func convertEdgeCardinality(c taskid.EdgeCardinality) *apiv1.TaskDependencyCardinality {
	var val apiv1.TaskDependencyCardinality
	switch c {
	case taskid.CardinalityPointToPoint:
		val = apiv1.TaskDependencyCardinality_TASK_DEPENDENCY_CARDINALITY_POINT_TO_POINT
	case taskid.CardinalityFanIn:
		val = apiv1.TaskDependencyCardinality_TASK_DEPENDENCY_CARDINALITY_FAN_IN
	default:
		val = apiv1.TaskDependencyCardinality_TASK_DEPENDENCY_CARDINALITY_UNSPECIFIED
	}
	return val.Enum()
}

func convertDependencyScope(s taskid.DependencyScope) *apiv1.TaskDependencyScope {
	var val apiv1.TaskDependencyScope
	switch s {
	case taskid.ScopeAll:
		val = apiv1.TaskDependencyScope_TASK_DEPENDENCY_SCOPE_ALL
	case taskid.ScopeActiveFeatures:
		val = apiv1.TaskDependencyScope_TASK_DEPENDENCY_SCOPE_ACTIVE_FEATURES
	case taskid.ScopeActiveGraph:
		val = apiv1.TaskDependencyScope_TASK_DEPENDENCY_SCOPE_ACTIVE_GRAPH
	default:
		val = apiv1.TaskDependencyScope_TASK_DEPENDENCY_SCOPE_UNSPECIFIED
	}
	return val.Enum()
}
