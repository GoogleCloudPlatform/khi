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

package coretask

import (
	"fmt"
	"slices"
	"strings"
)

// verifyAcyclic checks if the graph formed by outgoingAdjList contains any cycle.
func verifyAcyclic(outgoingAdjList map[string][]string, inDegree map[string]int, nodeCount int) error {
	inDegreeCopy := make(map[string]int, len(inDegree))
	for k, v := range inDegree {
		inDegreeCopy[k] = v
	}
	queue := make([]string, 0, nodeCount)
	for implID, deg := range inDegreeCopy {
		if deg == 0 {
			queue = append(queue, implID)
		}
	}
	visitedCount := 0
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		visitedCount++
		for _, target := range outgoingAdjList[curr] {
			inDegreeCopy[target]--
			if inDegreeCopy[target] == 0 {
				queue = append(queue, target)
			}
		}
	}
	if visitedCount < nodeCount {
		cyclicPath := extractCyclicDependencyPath(outgoingAdjList, inDegreeCopy)
		return fmt.Errorf("failed to sort as a runnable task graph. \n The graph contains cyclic dependency\n%s", cyclicPath)
	}
	return nil
}

// isReachable checks if there is a directed path of length >= 1 from startImplID to targetImplID.
func isReachable(startImplID, targetImplID string, outgoingAdjList map[string][]string) bool {
	visited := make(map[string]bool)
	queue := make([]string, 0, len(outgoingAdjList[startImplID]))
	for _, next := range outgoingAdjList[startImplID] {
		if next == targetImplID {
			return true
		}
		visited[next] = true
		queue = append(queue, next)
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, next := range outgoingAdjList[curr] {
			if next == targetImplID {
				return true
			}
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}

// extractCyclicDependencyPath identifies and formats the cyclic path in the graph.
func extractCyclicDependencyPath(
	outgoingAdjList map[string][]string,
	inDegree map[string]int,
) string {
	unresolved := make([]string, 0)
	for implID, deg := range inDegree {
		if deg > 0 {
			unresolved = append(unresolved, implID)
		}
	}
	slices.Sort(unresolved)
	if len(unresolved) == 0 {
		return ""
	}

	cycle := findCycle(outgoingAdjList, inDegree, unresolved)
	if len(cycle) == 0 {
		return fmt.Sprintf("... -> %s -> ...", unresolved[0])
	}

	// Format matching: "... -> tail] -> [cycle -> ...] -> [head -> ..."
	return fmt.Sprintf("... -> %s] -> [%s] -> [%s -> ...", cycle[len(cycle)-1], strings.Join(cycle, " -> "), cycle[0])
}

// findCycle performs DFS over unresolved nodes to detect and return a cycle path.
func findCycle(outgoingAdjList map[string][]string, inDegree map[string]int, unresolved []string) []string {
	visitState := make(map[string]int) // 0: unvisited, 1: visiting, 2: visited
	parent := make(map[string]string)
	var cycle []string

	var dfs func(currImplID string) bool
	dfs = func(currImplID string) bool {
		visitState[currImplID] = 1
		for _, nextImplID := range outgoingAdjList[currImplID] {
			if inDegree[nextImplID] <= 0 {
				continue
			}
			if visitState[nextImplID] == 1 {
				for curr := currImplID; curr != nextImplID && curr != ""; curr = parent[curr] {
					cycle = append(cycle, curr)
				}
				cycle = append(cycle, nextImplID)
				slices.Reverse(cycle)
				return true
			}
			if visitState[nextImplID] == 0 {
				parent[nextImplID] = currImplID
				if dfs(nextImplID) {
					return true
				}
			}
		}
		visitState[currImplID] = 2
		return false
	}

	for _, start := range unresolved {
		if visitState[start] == 0 && dfs(start) {
			break
		}
	}
	return cycle
}
