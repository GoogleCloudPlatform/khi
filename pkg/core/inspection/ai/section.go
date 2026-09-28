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

package ai

import (
	"strings"
	"unicode"

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
}

func newSectionResolver(taskGraph *coretask.TaskSet) *sectionResolver {
	destinations := make(map[string]destination)
	features := make(map[string]featureInfo)

	isForm := make(map[string]bool)
	isInventory := make(map[string]bool)

	allTasks := taskGraph.GetAll()
	for _, task := range allTasks {
		id := task.UntypedID().String()
		labels := task.Labels()
		switch {
		case typedmap.GetOrDefault(labels, inspectioncore.LabelKeyInspectionFeatureFlag, false):
			features[id] = featureInfo{
				id:    id,
				title: typedmap.GetOrDefault(labels, inspectioncore.LabelKeyFeatureTaskTitle, ""),
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
		case typedmap.GetOrDefault(labels, inspectioncore.LabelKeyInventoryTask, false):
			isInventory[id] = true
			destinations[id] = destination{
				kind: destinationShared,
			}
		}
	}

	adj := make(map[string][]string)
	for _, edge := range taskGraph.Edges() {
		adj[edge.SourceImplID] = append(adj[edge.SourceImplID], edge.TargetImplID)
	}

	for _, task := range allTasks {
		id := task.UntypedID().String()
		if _, isFeature := features[id]; isFeature || isForm[id] || isInventory[id] {
			continue
		}

		if best, ok := findNearestFeature(id, adj, isInventory, features); ok {
			destinations[id] = destination{
				kind:      destinationFeatureMember,
				featureID: best.id,
			}
		} else {
			destinations[id] = destination{
				kind: destinationShared,
			}
		}
	}

	return &sectionResolver{
		destinations: destinations,
		features:     features,
	}
}

func findNearestFeature(startID string, adj map[string][]string, isInventory map[string]bool, features map[string]featureInfo) (featureInfo, bool) {
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
				if isInventory[next] {
					// Shared inventory tasks must not be traversed into or through.
					continue
				}
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
	return r.destinations[taskID]
}

// formatTaskTitle converts a task implementation or reference ID into a human-readable title.
func formatTaskTitle(taskID string) string {
	if idx := strings.LastIndex(taskID, "/"); idx != -1 {
		taskID = taskID[idx+1:]
	}
	if idx := strings.Index(taskID, "#"); idx != -1 {
		taskID = taskID[:idx]
	}
	words := strings.FieldsFunc(taskID, func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})
	for i, w := range words {
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
