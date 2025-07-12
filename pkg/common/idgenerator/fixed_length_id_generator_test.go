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

package idgenerator

import (
	"sync"
	"testing"
)

func TestFixedLengthIDGenerator_Generate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "Test with length 10",
			length: 10,
		},
		{
			name:   "Test with length 32",
			length: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewFixedLengthIDGenerator(tt.length)
			id := g.Generate()
			if len(id) != tt.length {
				t.Errorf("Generate() length = %v, want %v", len(id), tt.length)
			}
		})
	}
}

func TestFixedLengthIDGenerator_Generate_Concurrent(t *testing.T) {
	t.Parallel()
	g := NewFixedLengthIDGenerator(16)
	numGoroutines := 100
	idsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	generatedIDs := make(map[string]bool)
	var mu sync.Mutex

	// Probability colliding generated IDs are low enough. (3.54*E-27 %) Duplicated IDs indicate problems on its logic.
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id := g.Generate()
				mu.Lock()
				if _, exists := generatedIDs[id]; exists {
					t.Errorf("Duplicate ID generated: %s", id)
				}
				generatedIDs[id] = true
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	expectedNumIDs := numGoroutines * idsPerGoroutine
	if len(generatedIDs) != expectedNumIDs {
		t.Errorf("Expected %d unique IDs, but got %d", expectedNumIDs, len(generatedIDs))
	}
}
