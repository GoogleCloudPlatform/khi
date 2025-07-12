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
	"fmt"
	"sync"
	"testing"
)

func TestPrefixIDGenerator_Generate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		prefix string
		want   []string
	}{
		{
			name:   "Test with prefix",
			prefix: "test-",
			want:   []string{"test-1", "test-2", "test-3"},
		},
		{
			name:   "Test with empty prefix",
			prefix: "",
			want:   []string{"1", "2", "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewPrefixIDGenerator(tt.prefix)
			for _, want := range tt.want {
				if got := g.Generate(); got != want {
					t.Errorf("Generate() = %v, want %v", got, want)
				}
			}
		})
	}
}

func TestPrefixIDGenerator_Generate_Concurrent(t *testing.T) {
	t.Parallel()
	g := NewPrefixIDGenerator("concurrent-")
	numGoroutines := 100
	idsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	generatedIDs := make(map[string]bool)
	var mu sync.Mutex

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

	// Check if all numbers from 1 to expectedNumIDs are present
	for i := 1; i <= expectedNumIDs; i++ {
		expectedID := fmt.Sprintf("concurrent-%d", i)
		if !generatedIDs[expectedID] {
			t.Errorf("Expected ID %s was not generated", expectedID)
		}
	}
}
