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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
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
	kind      destinationKind
	featureID string
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

	isForm := make(map[string]bool)

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
				kind:      destinationFeature,
				featureID: id,
			}
		case typedmap.GetOrDefault(labels, inspectioncore.TaskLabelKeyIsFormTask, false):
			isForm[id] = true
			destinations[id] = destination{
				kind: destinationForm,
			}
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
		if _, isGated := gateTaskByTask[edge.SourceImplID]; isGated {
			// Feature-gated tasks export data to shared inventory tasks across features,
			// so their outgoing data edges must not be traversed for section resolution.
			continue
		}
		adj[edge.SourceImplID] = append(adj[edge.SourceImplID], edge.TargetImplID)
	}

	for _, task := range allTasks {
		id := task.UntypedID().String()
		if _, isFeature := features[id]; isFeature || isForm[id] {
			continue
		}

		startID := id
		if gateTaskID, isGated := gateTaskByTask[id]; isGated {
			if _, isFeature := features[gateTaskID]; isFeature {
				destinations[id] = destination{
					kind:      destinationFeatureMember,
					featureID: gateTaskID,
				}
				continue
			}
			startID = gateTaskID
		}

		if startID != "" {
			if best, ok := findNearestFeature(startID, adj, features); ok {
				destinations[id] = destination{
					kind:      destinationFeatureMember,
					featureID: best.id,
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

func findNearestFeature(startID string, adj map[string][]string, features map[string]featureInfo) (featureInfo, bool) {
	currentLevel := []string{startID}
	visited := map[string]bool{startID: true}
	var candidateFeatures []featureInfo

	for len(currentLevel) > 0 {
		var nextLevel []string
		for _, curr := range currentLevel {
			for _, next := range adj[curr] {
				if visited[next] {
					continue
				}
				visited[next] = true
				if fi, ok := features[next]; ok {
					// Feature tasks are terminal search targets and are not expanded further.
					candidateFeatures = append(candidateFeatures, fi)
					continue
				}
				nextLevel = append(nextLevel, next)
			}
		}
		if len(candidateFeatures) > 0 {
			break
		}
		currentLevel = nextLevel
	}

	if len(candidateFeatures) == 0 {
		return featureInfo{}, false
	}

	best := candidateFeatures[0]
	for _, f := range candidateFeatures[1:] {
		if f.order < best.order || (f.order == best.order && f.id < best.id) {
			best = f
		}
	}
	return best, true
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
