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
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestTimelinePathPool_Get(t *testing.T) {
	// Set up common dependencies for tests
	idGen := &IDGenerator{}
	internPool := NewInternPool(idGen)

	id1 := uint32(1)
	id2 := uint32(2)
	typeA := &pb.TimelineType{Id: &id1}
	typeB := &pb.TimelineType{Id: &id2}

	testCases := []struct {
		name string
		test func(t *testing.T, pool *TimelinePathPool)
	}{
		{
			name: "should create and deduplicate root paths",
			test: func(t *testing.T, pool *TimelinePathPool) {
				p1 := pool.Get(nil, PathSegment{Name: "root", Type: typeA})
				p2 := pool.Get(nil, PathSegment{Name: "root", Type: typeA})

				if p1 != p2 {
					t.Errorf("expected identical pointers for exact same root path")
				}
				if p1.Parent != nil {
					t.Errorf("expected nil parent for root path")
				}
				if diff := cmp.Diff("root", p1.Name.Resolve()); diff != "" {
					t.Errorf("name mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(typeA, p1.Type, protocmp.Transform()); diff != "" {
					t.Errorf("type mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "should differentiate by name",
			test: func(t *testing.T, pool *TimelinePathPool) {
				p1 := pool.Get(nil, PathSegment{Name: "root1", Type: typeA})
				p2 := pool.Get(nil, PathSegment{Name: "root2", Type: typeA})

				if p1 == p2 {
					t.Errorf("expected different pointers for different names")
				}
			},
		},
		{
			name: "should differentiate by type",
			test: func(t *testing.T, pool *TimelinePathPool) {
				p1 := pool.Get(nil, PathSegment{Name: "root", Type: typeA})
				p2 := pool.Get(nil, PathSegment{Name: "root", Type: typeB})

				if p1 == p2 {
					t.Errorf("expected different pointers for different types")
				}
			},
		},
		{
			name: "should differentiate by parent",
			test: func(t *testing.T, pool *TimelinePathPool) {
				parent1 := pool.Get(nil, PathSegment{Name: "parent1", Type: typeA})
				parent2 := pool.Get(nil, PathSegment{Name: "parent2", Type: typeA})

				child1 := pool.Get(parent1, PathSegment{Name: "child", Type: typeB})
				child2 := pool.Get(parent2, PathSegment{Name: "child", Type: typeB})

				if child1 == child2 {
					t.Errorf("expected different pointers for different parents")
				}
			},
		},
		{
			name: "should correctly append child paths",
			test: func(t *testing.T, pool *TimelinePathPool) {
				parent := pool.Get(nil, PathSegment{Name: "parent", Type: typeA})
				child := pool.Get(parent, PathSegment{Name: "child", Type: typeB})

				if child.Parent != parent {
					t.Errorf("expected child path to point to the correct parent")
				}
				if diff := cmp.Diff("child", child.Name.Resolve()); diff != "" {
					t.Errorf("name mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "should build multi-level path identical to sequential build",
			test: func(t *testing.T, pool *TimelinePathPool) {
				multi := pool.Get(nil,
					PathSegment{Name: "root", Type: typeA},
					PathSegment{Name: "child", Type: typeB},
				)

				root := pool.Get(nil, PathSegment{Name: "root", Type: typeA})
				child := pool.Get(root, PathSegment{Name: "child", Type: typeB})

				if multi != child {
					t.Errorf("expected multi-segment path to match sequential path pointers")
				}
			},
		},
		{
			name: "should handle concurrent creation safely without duplicates",
			test: func(t *testing.T, pool *TimelinePathPool) {
				const numGoroutines = 100
				var wg sync.WaitGroup
				wg.Add(numGoroutines)

				results := make([]*TimelinePath, numGoroutines)
				for i := 0; i < numGoroutines; i++ {
					go func(idx int) {
						defer wg.Done()
						results[idx] = pool.Get(nil,
							PathSegment{Name: "concurrent", Type: typeA},
							PathSegment{Name: "path", Type: typeB},
						)
					}(i)
				}
				wg.Wait()

				first := results[0]
				for i := 1; i < numGoroutines; i++ {
					if results[i] != first {
						t.Errorf("expected all concurrent calls to return the exact same pointer, mismatch at index %d", i)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use a fresh pool for each test case to avoid cross-test contamination
			pool := NewTimelinePathPool(idGen, internPool)
			tc.test(t, pool)
		})
	}
}
