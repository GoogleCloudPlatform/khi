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

package summary

import (
	"fmt"
	"slices"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

type destinationKind int

const (
	destinationShared destinationKind = iota
	destinationForm
	destinationFeature
	destinationFeatureMember
)

type destination struct {
	kind       destinationKind
	featureIDs []string
}

type featureInfo struct {
	id    string
	title string
	order int
}

type sectionResolver struct {
	destinations map[string]destination
	features     map[string]featureInfo
	taskTitles   map[string]string
}

func newSectionResolver(taskGraph *coretask.TaskSet) *sectionResolver {
	destinations := make(map[string]destination)
	features := make(map[string]featureInfo)
	isBoundary := make(map[string]bool)

	allTasks := taskGraph.GetAll()
	refToImplID := make(map[string]string, len(allTasks))
	taskTitles := make(map[string]string, len(allTasks))
	for _, task := range allTasks {
		id := task.UntypedID().String()
		refToImplID[task.UntypedID().ReferenceIDString()] = id
		labels := task.Labels()
		title := typedmap.GetOrDefault(labels, coretask.LabelKeyTaskTitle, "")
		if title == "" {
			title = id
		}
		taskTitles[id] = title
		switch {
		case typedmap.GetOrDefault(labels, inspectioncore.LabelKeyInspectionFeatureFlag, false):
			features[id] = featureInfo{
				id:    id,
				title: title,
				order: typedmap.GetOrDefault(labels, inspectioncore.LabelKeyFeatureTaskOrder, 0),
			}
			destinations[id] = destination{
				kind:       destinationFeature,
				featureIDs: []string{id},
			}
			isBoundary[id] = true
		case typedmap.GetOrDefault(labels, inspectioncore.TaskLabelKeyIsFormTask, false):
			destinations[id] = destination{
				kind: destinationForm,
			}
			isBoundary[id] = true
		case hasFanInInput(task):
			// Fan-in tasks aggregate shared data across features and belong to Shared & Filter Context.
			destinations[id] = destination{
				kind: destinationShared,
			}
			isBoundary[id] = true
		}
	}

	gateTaskByTask := make(map[string]string)
	for _, task := range allTasks {
		id := task.UntypedID().String()
		if gateRef, hasGate := typedmap.Get(task.Labels(), coretask.LabelKeyFeatureGateTaskRef); hasGate {
			gateTaskByTask[id] = refToImplID[gateRef.ReferenceIDString()]
		}
	}

	adj := make(map[string][]string)
	for _, edge := range taskGraph.Edges() {
		if edge.Cardinality != taskid.CardinalityPointToPoint || edge.Scope != taskid.ScopeAll {
			// Optional and fan-in dependencies do not pull upstream tasks into a consumer feature's ownership.
			continue
		}
		if isBoundary[edge.SourceImplID] {
			// Outgoing edges from feature, form, and fan-in boundary tasks must not propagate ownership downstream.
			continue
		}
		if _, isGated := gateTaskByTask[edge.SourceImplID]; isGated {
			// Feature-gated tasks export data to shared inventory tasks across features,
			// so their outgoing data edges must not be traversed for section resolution.
			continue
		}
		adj[edge.SourceImplID] = append(adj[edge.SourceImplID], edge.TargetImplID)
	}

	for _, task := range allTasks {
		id := task.UntypedID().String()
		if isBoundary[id] {
			continue
		}

		startID := id
		if gateTaskID, isGated := gateTaskByTask[id]; isGated {
			startID = gateTaskID
		}

		if startID != "" {
			if reached := findReachableFeatures(startID, adj, features); len(reached) > 0 {
				destinations[id] = destination{
					kind:       destinationFeatureMember,
					featureIDs: reached,
				}
				continue
			}
		}
		destinations[id] = destination{
			kind: destinationShared,
		}
	}

	return &sectionResolver{
		destinations: destinations,
		features:     features,
		taskTitles:   taskTitles,
	}
}

func hasFanInInput(task coretask.UntypedTask) bool {
	for _, input := range task.Inputs() {
		if input.Dependency.DescriptorCardinality() == taskid.CardinalityFanIn {
			return true
		}
	}
	return false
}

func findReachableFeatures(startID string, adj map[string][]string, features map[string]featureInfo) []string {
	if _, isFeature := features[startID]; isFeature {
		return []string{startID}
	}

	queue := []string{startID}
	visited := map[string]bool{startID: true}
	var reached []string

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, next := range adj[curr] {
			if visited[next] {
				continue
			}
			visited[next] = true
			if _, isFeature := features[next]; isFeature {
				// Feature tasks are terminal search targets and are not expanded further.
				reached = append(reached, next)
				continue
			}
			queue = append(queue, next)
		}
	}

	slices.Sort(reached)
	return reached
}

func (r *sectionResolver) destinationOf(taskID string) destination {
	dest, ok := r.destinations[taskID]
	if !ok {
		panic(fmt.Sprintf("unknown task ID %q in sectionResolver", taskID))
	}
	return dest
}

func (r *sectionResolver) titleOf(taskID string) string {
	title, ok := r.taskTitles[taskID]
	if !ok {
		panic(fmt.Sprintf("unknown task ID %q in sectionResolver", taskID))
	}
	return title
}
