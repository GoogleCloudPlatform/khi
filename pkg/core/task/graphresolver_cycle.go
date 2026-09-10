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
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

type consumerTagKey struct {
	consumerImplID string
	tag            string
}

// resolveFanInEdgesAndCycles resolves candidate fan-in edges against the fixed DAG formed by pointToPointEdges.
// Any candidate fan-in edge that would create a cycle (self-loop or reachability cycle) is pruned.
// It returns:
// - resolvedTasks: all tasks in the resolved graph.
// - resolvedEdges: deduplicated point-to-point and accepted fan-in edges between resolved tasks.
// - boundFanInRefIDsByTaskImplID: map of consumer task implementation ID -> tag -> bound producer reference IDs.
func resolveFanInEdgesAndCycles(
	graphTaskMap map[string]UntypedTask,
	pointToPointEdges []taskid.TaskEdge,
	candidateFanInEdges []taskid.TaskEdge,
) (
	resolvedTasks []UntypedTask,
	resolvedEdges []taskid.TaskEdge,
	boundFanInRefIDsByTaskImplID map[string]map[string][]string,
	err error,
) {
	outgoingAdjList, err := buildPointToPointOutgoingAdjacencyList(graphTaskMap, pointToPointEdges)
	if err != nil {
		return nil, nil, nil, err
	}

	sortedKeys, candidateEdgesByKey := groupCandidateFanInEdgesByConsumerTag(candidateFanInEdges)
	acceptedEdgesByKey := resolveCandidateFanInEdges(sortedKeys, candidateEdgesByKey, outgoingAdjList)

	resolvedTasks, resolvedEdges, boundFanInRefIDsByTaskImplID = assembleResolvedResult(graphTaskMap, pointToPointEdges, sortedKeys, acceptedEdgesByKey)
	return resolvedTasks, resolvedEdges, boundFanInRefIDsByTaskImplID, nil
}

// buildPointToPointOutgoingAdjacencyList constructs the outgoing adjacency and in-degree maps from point-to-point edges and validates acyclicity.
func buildPointToPointOutgoingAdjacencyList(
	graphTaskMap map[string]UntypedTask,
	pointToPointEdges []taskid.TaskEdge,
) (map[string][]string, error) {
	outgoingAdjList := make(map[string][]string, len(graphTaskMap))
	inDegree := make(map[string]int, len(graphTaskMap))
	for _, task := range graphTaskMap {
		implID := task.UntypedID().String()
		outgoingAdjList[implID] = nil
		inDegree[implID] = 0
	}

	for _, e := range pointToPointEdges {
		outgoingAdjList[e.SourceImplID] = append(outgoingAdjList[e.SourceImplID], e.TargetImplID)
		inDegree[e.TargetImplID]++
	}

	if err := verifyAcyclic(outgoingAdjList, inDegree, len(graphTaskMap)); err != nil {
		return nil, err
	}
	return outgoingAdjList, nil
}

// resolveCandidateFanInEdges filters candidate fan-in edges per consumer and tag, pruning any cyclic edges.
func resolveCandidateFanInEdges(
	sortedKeys []consumerTagKey,
	candidateEdgesByKey map[consumerTagKey][]taskid.TaskEdge,
	outgoingAdjList map[string][]string,
) map[consumerTagKey][]taskid.TaskEdge {
	acceptedEdgesByKey := make(map[consumerTagKey][]taskid.TaskEdge, len(sortedKeys))
	for _, key := range sortedKeys {
		candidates := candidateEdgesByKey[key]
		accepted := resolveCandidateFanInForConsumer(key.consumerImplID, candidates, outgoingAdjList)
		acceptedEdgesByKey[key] = accepted
	}
	return acceptedEdgesByKey
}

// assembleResolvedResult builds the resolved tasks, deduplicated edges, and bound fan-in refs.
func assembleResolvedResult(
	graphTaskMap map[string]UntypedTask,
	pointToPointEdges []taskid.TaskEdge,
	sortedKeys []consumerTagKey,
	acceptedEdgesByKey map[consumerTagKey][]taskid.TaskEdge,
) ([]UntypedTask, []taskid.TaskEdge, map[string]map[string][]string) {
	var allFanInEdges []taskid.TaskEdge
	for _, key := range sortedKeys {
		allFanInEdges = append(allFanInEdges, acceptedEdgesByKey[key]...)
	}
	allEdges := make([]taskid.TaskEdge, 0, len(pointToPointEdges)+len(allFanInEdges))
	allEdges = append(allEdges, pointToPointEdges...)
	allEdges = append(allEdges, allFanInEdges...)
	dedupedEdges := deduplicateAndNormalizeEdges(allEdges)

	boundFanInRefIDsByTaskImplID := collectBoundFanInRefIDsByTaskImplID(allFanInEdges)
	tasks := make([]UntypedTask, 0, len(graphTaskMap))
	for _, t := range graphTaskMap {
		tasks = append(tasks, t)
	}
	slices.SortFunc(tasks, compareTaskByImplementationID)
	return tasks, dedupedEdges, boundFanInRefIDsByTaskImplID
}

// groupCandidateFanInEdgesByConsumerTag groups candidate fan-in edges by (consumerImplID, tag) and sorts keys deterministically.
// It returns:
// - sortedKeys: deterministically sorted slice of consumerTagKey (sorted by consumerImplID then tag).
// - candidateEdgesByKey: map of consumerTagKey to candidate fan-in edges for that consumer and tag.
func groupCandidateFanInEdgesByConsumerTag(
	candidateFanInEdges []taskid.TaskEdge,
) (
	sortedKeys []consumerTagKey,
	candidateEdgesByKey map[consumerTagKey][]taskid.TaskEdge,
) {
	candidateEdgesByKey = make(map[consumerTagKey][]taskid.TaskEdge)
	for _, e := range candidateFanInEdges {
		key := consumerTagKey{
			consumerImplID: e.TargetImplID,
			tag:            e.Tag,
		}
		candidateEdgesByKey[key] = append(candidateEdgesByKey[key], e)
	}

	sortedKeys = make([]consumerTagKey, 0, len(candidateEdgesByKey))
	for k := range candidateEdgesByKey {
		sortedKeys = append(sortedKeys, k)
	}
	slices.SortFunc(sortedKeys, func(a, b consumerTagKey) int {
		if a.consumerImplID != b.consumerImplID {
			return strings.Compare(a.consumerImplID, b.consumerImplID)
		}
		return strings.Compare(a.tag, b.tag)
	})
	return sortedKeys, candidateEdgesByKey
}

// resolveCandidateFanInForConsumer resolves candidate fan-in edges for a specific consumer,
// pruning any edges that would create a cycle (self-loop or reachability cycle).
func resolveCandidateFanInForConsumer(
	consumerImplID string,
	candidates []taskid.TaskEdge,
	outgoingAdjList map[string][]string,
) []taskid.TaskEdge {
	candidatesByPriority := make(map[int][]taskid.TaskEdge)
	for _, e := range candidates {
		candidatesByPriority[e.Priority] = append(candidatesByPriority[e.Priority], e)
	}

	priorities := make([]int, 0, len(candidatesByPriority))
	for p := range candidatesByPriority {
		priorities = append(priorities, p)
	}
	slices.Sort(priorities)

	var acceptedEdges []taskid.TaskEdge
	for _, priority := range priorities {
		candidateEdgesAtPriority := candidatesByPriority[priority]
		slices.SortFunc(candidateEdgesAtPriority, func(a, b taskid.TaskEdge) int {
			if c := strings.Compare(a.SourceRefID, b.SourceRefID); c != 0 {
				return c
			}
			return strings.Compare(a.SourceImplID, b.SourceImplID)
		})

		for _, candidateEdge := range candidateEdgesAtPriority {
			producerImplID := candidateEdge.SourceImplID
			createsCycle := (producerImplID == consumerImplID) || isReachable(consumerImplID, producerImplID, outgoingAdjList)
			if createsCycle {
				continue
			}

			outgoingAdjList[producerImplID] = append(outgoingAdjList[producerImplID], consumerImplID)
			acceptedEdges = append(acceptedEdges, candidateEdge)
		}
	}
	return acceptedEdges
}

// collectBoundFanInRefIDsByTaskImplID aggregates unique source reference IDs for accepted fan-in edges per consumer task implementation ID.
func collectBoundFanInRefIDsByTaskImplID(acceptedFanInEdges []taskid.TaskEdge) map[string]map[string][]string {
	boundFanInRefIDsByTaskImplID := make(map[string]map[string][]string)
	for _, e := range acceptedFanInEdges {
		if e.Tag == "" {
			continue
		}
		if boundFanInRefIDsByTaskImplID[e.TargetImplID] == nil {
			boundFanInRefIDsByTaskImplID[e.TargetImplID] = make(map[string][]string)
		}
		boundFanInRefIDsByTaskImplID[e.TargetImplID][e.Tag] = append(boundFanInRefIDsByTaskImplID[e.TargetImplID][e.Tag], e.SourceRefID)
	}
	for targetImplID, byTag := range boundFanInRefIDsByTaskImplID {
		for tag, refIDs := range byTag {
			slices.Sort(refIDs)
			boundFanInRefIDsByTaskImplID[targetImplID][tag] = slices.Compact(refIDs)
		}
	}
	return boundFanInRefIDsByTaskImplID
}
