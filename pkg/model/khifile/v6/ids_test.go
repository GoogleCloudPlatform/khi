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

package khifilev6

import (
	"sync"
	"testing"
)

func TestIDGenerator(t *testing.T) {
	t.Run("sequential new", func(t *testing.T) {
		g := &IDGenerator{}

		steps := []struct {
			name string
			ns   IDNamespace
			want uint32
		}{
			{"first string", IDString, 1},
			{"second string", IDString, 2},
			{"first fieldset", IDFieldSet, 1},
		}

		for _, step := range steps {
			t.Run(step.name, func(t *testing.T) {
				got := g.New(step.ns)
				if got != step.want {
					t.Errorf("New(%v) = %d, want %d", step.ns, got, step.want)
				}
			})
		}
	})

	t.Run("set and new", func(t *testing.T) {
		g := &IDGenerator{}
		g.Set(IDString, 5)

		steps := []struct {
			name string
			ns   IDNamespace
			want uint32
		}{
			{"after set 5", IDString, 6},
			{"after set 6", IDString, 7},
		}

		for _, step := range steps {
			t.Run(step.name, func(t *testing.T) {
				got := g.New(step.ns)
				if got != step.want {
					t.Errorf("New(%v) = %d, want %d", step.ns, got, step.want)
				}
			})
		}
	})

	t.Run("concurrent new", func(t *testing.T) {
		g := &IDGenerator{}
		const (
			numGoroutines = 10
			numIncrements = 100
		)
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		ids := make(map[uint32]bool)
		var mu sync.Mutex

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numIncrements; j++ {
					id := g.New(IDString)
					mu.Lock()
					if ids[id] {
						t.Errorf("duplicate ID generated: %d", id)
					}
					ids[id] = true
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		expectedTotal := numGoroutines * numIncrements
		if len(ids) != expectedTotal {
			t.Errorf("total unique IDs = %d, want %d", len(ids), expectedTotal)
		}
	})
}