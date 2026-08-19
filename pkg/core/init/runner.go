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

package coreinit

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

var (
	registryMu   sync.RWMutex
	initializers []*Initializer
)

// RegisterInitializer registers an Initializer into the global registry.
func RegisterInitializer(initializer *Initializer) {
	if initializer == nil || strings.TrimSpace(string(initializer.ID)) == "" {
		panic("initializer must have a non-empty ID")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	initializers = append(initializers, initializer)
}

// ResetInitializersForTest clears registered initializers for unit testing.
func ResetInitializersForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()
	initializers = nil
}

func sortInitializers(inits []*Initializer) ([]*Initializer, error) {
	nodeMap := make(map[InitializerID]*Initializer, len(inits))
	inDegree := make(map[InitializerID]int, len(inits))
	adjList := make(map[InitializerID][]InitializerID, len(inits))

	for _, init := range inits {
		if _, exists := nodeMap[init.ID]; exists {
			return nil, fmt.Errorf("duplicate initializer registered: %s", init.ID)
		}
		nodeMap[init.ID] = init
		inDegree[init.ID] = 0
	}

	// Build edges from Dependencies (dep -> init) and Before (init -> beforeTarget)
	for _, init := range inits {
		for _, dep := range init.Dependencies {
			if _, exists := nodeMap[dep]; !exists {
				return nil, fmt.Errorf("initializer %s requires missing dependency: %s", init.ID, dep)
			}
			adjList[dep] = append(adjList[dep], init.ID)
			inDegree[init.ID]++
		}
		for _, beforeTarget := range init.Before {
			if _, exists := nodeMap[beforeTarget]; !exists {
				return nil, fmt.Errorf("initializer %s references missing before target: %s", init.ID, beforeTarget)
			}
			adjList[init.ID] = append(adjList[init.ID], beforeTarget)
			inDegree[beforeTarget]++
		}
	}

	// Kahn's algorithm with deterministic tie-breaking by sorting keys
	var queue []InitializerID
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	slices.Sort(queue)

	var sorted []*Initializer
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		sorted = append(sorted, nodeMap[curr])

		var nextReady []InitializerID
		for _, neighbor := range adjList[curr] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				nextReady = append(nextReady, neighbor)
			}
		}

		slices.Sort(nextReady)
		queue = append(queue, nextReady...)
	}

	if len(sorted) != len(inits) {
		return nil, fmt.Errorf("circular dependency detected among initializers")
	}

	return sorted, nil
}
