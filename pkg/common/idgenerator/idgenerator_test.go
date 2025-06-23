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

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestUUIDGenerator(t *testing.T) {
	t.Run("Generate returns non-empty string", func(t *testing.T) {
		gen := NewUUIDGenerator()
		id := gen.Generate()
		if id == "" {
			t.Error("Expected a non-empty ID, but got an empty string")
		}
	})

	t.Run("Generate returns unique IDs", func(t *testing.T) {
		gen := NewUUIDGenerator()
		id1 := gen.Generate()
		id2 := gen.Generate()
		if id1 == id2 {
			t.Errorf("Expected unique IDs, but got two identical IDs: %s", id1)
		}
	})

	t.Run("Thread safety", func(t *testing.T) {
		gen := NewUUIDGenerator()
		ids := make(chan string, 10000)
		var wg sync.WaitGroup

		for i := 0; i < 10000; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ids <- gen.Generate()
			}()
		}

		wg.Wait()
		close(ids)

		idSet := make(map[string]struct{})
		for id := range ids {
			if _, exists := idSet[id]; exists {
				t.Errorf("Duplicate ID generated in concurrent test: %s", id)
			}
			idSet[id] = struct{}{}
		}
	})
}

func TestSequentialGenerator(t *testing.T) {
	t.Run("Generate returns sequential IDs", func(t *testing.T) {
		gen := NewSequentialGenerator("test-")
		if id1 := gen.Generate(); id1 != "test-1" {
			t.Errorf("Expected first ID to be 'test-1', got '%s'", id1)
		}
		if id2 := gen.Generate(); id2 != "test-2" {
			t.Errorf("Expected second ID to be 'test-2', got '%s'", id2)
		}
	})

	t.Run("Thread safety", func(t *testing.T) {
		gen := NewSequentialGenerator("concurrent-")
		ids := make(chan string, 10000)
		var wg sync.WaitGroup

		for i := 0; i < 10000; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ids <- gen.Generate()
			}()
		}

		wg.Wait()
		close(ids)

		idSet := make(map[string]struct{})
		for id := range ids {
			if _, exists := idSet[id]; exists {
				t.Errorf("Duplicate ID generated in concurrent test: %s", id)
			}
			idSet[id] = struct{}{}
		}
	})
}
