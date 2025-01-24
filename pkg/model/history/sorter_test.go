// Copyright 2024 Google LLC
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

package history

import (
	"slices"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/google/go-cmp/cmp"
)

type TestResourceChunkSortStrategy struct {
}

// SortChunk implements ResourceChunkSortStrategy.
func (t *TestResourceChunkSortStrategy) SortChunk(builder *Builder, parents []*Resource, groupedRelationship enum.ParentRelationship, chunk []*Resource) ([]*Resource, error) {
	slices.SortFunc(
		chunk, func(a, b *Resource) int {
			return strings.Compare(a.ResourceName, b.ResourceName)
		},
	)
	return chunk, nil
}

var _ ResourceChunkSortStrategy = (*TestResourceChunkSortStrategy)(nil)

type testAllSkipChunkSortStrategy struct {
}

// SortChunk implements ResourceChunkSortStrategy.
func (t *testAllSkipChunkSortStrategy) SortChunk(builder *Builder, parents []*Resource, groupedRelationship enum.ParentRelationship, chunk []*Resource) ([]*Resource, error) {
	return nil, ErrorSortSkipped
}

var _ ResourceChunkSortStrategy = (*testAllSkipChunkSortStrategy)(nil)

func newResourceForTesting(name string, rel enum.ParentRelationship, children ...*Resource) *Resource {
	if children == nil {
		children = make([]*Resource, 0)
	}
	return &Resource{
		ResourceName: name,
		Relationship: rel,
		Children:     children,
	}
}

func TestResourceSort(t *testing.T) {
	type testCase struct {
		name     string
		input    []*Resource
		expected []*Resource
		strategy []ResourceChunkSortStrategy
	}
	testCases := []testCase{
		{
			name: "single layer with name",
			input: []*Resource{
				newResourceForTesting("c", enum.RelationshipChild),
				newResourceForTesting("b", enum.RelationshipChild),
				newResourceForTesting("a", enum.RelationshipChild),
			},
			expected: []*Resource{
				newResourceForTesting("a", enum.RelationshipChild),
				newResourceForTesting("b", enum.RelationshipChild),
				newResourceForTesting("c", enum.RelationshipChild),
			},
			strategy: []ResourceChunkSortStrategy{&TestResourceChunkSortStrategy{}},
		},
		{
			name: "single layer with name and skipped sorter",
			input: []*Resource{
				newResourceForTesting("c", enum.RelationshipChild),
				newResourceForTesting("b", enum.RelationshipChild),
				newResourceForTesting("a", enum.RelationshipChild),
			},
			expected: []*Resource{
				newResourceForTesting("a", enum.RelationshipChild),
				newResourceForTesting("b", enum.RelationshipChild),
				newResourceForTesting("c", enum.RelationshipChild),
			},
			strategy: []ResourceChunkSortStrategy{&TestResourceChunkSortStrategy{}},
		},
		{
			name: "single layer with relationship",
			input: []*Resource{
				newResourceForTesting("c", enum.RelationshipChild),
				newResourceForTesting("b", enum.RelationshipContainer),
				newResourceForTesting("a", enum.RelationshipChild),
			},
			expected: []*Resource{
				newResourceForTesting("a", enum.RelationshipChild),
				newResourceForTesting("c", enum.RelationshipChild),
				newResourceForTesting("b", enum.RelationshipContainer),
			},
			strategy: []ResourceChunkSortStrategy{&testAllSkipChunkSortStrategy{}, &TestResourceChunkSortStrategy{}},
		},
		{
			name: "multiple layer with name",
			input: []*Resource{
				newResourceForTesting("c", enum.RelationshipChild,
					newResourceForTesting("c", enum.RelationshipChild),
					newResourceForTesting("b", enum.RelationshipChild),
					newResourceForTesting("a", enum.RelationshipChild)),
				newResourceForTesting("b", enum.RelationshipChild,
					newResourceForTesting("c", enum.RelationshipChild),
					newResourceForTesting("b", enum.RelationshipChild),
					newResourceForTesting("a", enum.RelationshipChild)),
				newResourceForTesting("a", enum.RelationshipChild,
					newResourceForTesting("c", enum.RelationshipChild),
					newResourceForTesting("b", enum.RelationshipChild),
					newResourceForTesting("a", enum.RelationshipChild)),
			},
			expected: []*Resource{
				newResourceForTesting("a", enum.RelationshipChild,
					newResourceForTesting("a", enum.RelationshipChild),
					newResourceForTesting("b", enum.RelationshipChild),
					newResourceForTesting("c", enum.RelationshipChild)),
				newResourceForTesting("b", enum.RelationshipChild,
					newResourceForTesting("a", enum.RelationshipChild),
					newResourceForTesting("b", enum.RelationshipChild),
					newResourceForTesting("c", enum.RelationshipChild)),
				newResourceForTesting("c", enum.RelationshipChild,
					newResourceForTesting("a", enum.RelationshipChild),
					newResourceForTesting("b", enum.RelationshipChild),
					newResourceForTesting("c", enum.RelationshipChild)),
			},
			strategy: []ResourceChunkSortStrategy{&TestResourceChunkSortStrategy{}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sorter := NewResourceSorter(tc.strategy...)
			result, err := sorter.SortAll(nil, tc.input)
			if err != nil {
				t.Errorf("unexpected error %s", err.Error())
			}
			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf("sort result mismatch\n%s", diff)
			}
		})
	}
}
