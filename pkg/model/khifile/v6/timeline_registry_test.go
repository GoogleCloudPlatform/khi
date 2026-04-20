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

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

func TestTimelineRegistry_GetBuilder(t *testing.T) {
	idGen := &IDGenerator{}
	internPool := NewInternPool(idGen)
	pool := NewTimelinePathPool(idGen, internPool)

	id := uint32(1)
	typeA := &pb.TimelineType{Id: &id}

	path1 := pool.Get(nil, PathSegment{Name: "path1", Type: typeA})
	path2 := pool.Get(nil, PathSegment{Name: "path2", Type: typeA})

	testCases := []struct {
		name string
		test func(t *testing.T, registry *TimelineRegistry)
	}{
		{
			name: "should return identical builder for same path",
			test: func(t *testing.T, registry *TimelineRegistry) {
				b1 := registry.GetBuilder(path1)
				b2 := registry.GetBuilder(path1)

				if b1 != b2 {
					t.Errorf("expected identical builder pointers for same path")
				}
				if b1.Path != path1 {
					t.Errorf("expected builder to hold the correct path")
				}
				if b1.TimelineItemsID == 0 {
					t.Errorf("expected builder to have a valid TimelineItemsID")
				}
			},
		},
		{
			name: "should return different builder for different paths",
			test: func(t *testing.T, registry *TimelineRegistry) {
				b1 := registry.GetBuilder(path1)
				b2 := registry.GetBuilder(path2)

				if b1 == b2 {
					t.Errorf("expected different builder pointers for different paths")
				}
			},
		},
		{
			name: "should handle concurrent builder creation safely",
			test: func(t *testing.T, registry *TimelineRegistry) {
				const numGoroutines = 100
				var wg sync.WaitGroup
				wg.Add(numGoroutines)

				results := make([]*TimelineBuilder, numGoroutines)
				for i := 0; i < numGoroutines; i++ {
					go func(idx int) {
						defer wg.Done()
						results[idx] = registry.GetBuilder(path1)
					}(i)
				}
				wg.Wait()

				first := results[0]
				for i := 1; i < numGoroutines; i++ {
					if results[i] != first {
						t.Errorf("expected all concurrent calls to return the exact same builder pointer, mismatch at index %d", i)
					}
				}
			},
		},
		{
			name: "should resolve alias to target builder",
			test: func(t *testing.T, registry *TimelineRegistry) {
				registry.SetAlias(path1, path2)
				b1 := registry.GetBuilder(path1)
				b2 := registry.GetBuilder(path2)

				if b1 != b2 {
					t.Errorf("expected alias path to resolve to target builder")
				}
				if b1.Path != path2 {
					t.Errorf("expected builder accessed via alias to hold target path, got %v, want %v", b1.Path, path2)
				}
			},
		},
		{
			name: "should panic if setting alias on path with attached builder",
			test: func(t *testing.T, registry *TimelineRegistry) {
				// Force attaching a builder to path1
				_ = registry.GetBuilder(path1)

				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected SetAlias to panic when alias already has a builder")
					}
				}()
				registry.SetAlias(path1, path2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use a fresh registry for each test case
			registry := NewTimelineRegistry(idGen, internPool)
			tc.test(t, registry)
		})
	}
}
