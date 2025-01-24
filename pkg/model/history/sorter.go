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
	"errors"
	"slices"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

var ErrorSortSkipped = errors.New("skipping this sorter for the given resource chunk")

type ResourceChunkSortStrategy interface {
	SortChunk(builder *Builder, parents []*Resource, groupedRelationship enum.ParentRelationship, chunk []*Resource) ([]*Resource, error)
}

type ResourceSorter struct {
	Strategies []ResourceChunkSortStrategy
}

func NewResourceSorter(strategy ...ResourceChunkSortStrategy) *ResourceSorter {
	return &ResourceSorter{Strategies: strategy}
}

func (r *ResourceSorter) SortAll(builder *Builder, resources []*Resource) ([]*Resource, error) {
	return r.sortLayer(builder, nil, resources)
}

func (r *ResourceSorter) sortLayer(builder *Builder, parents []*Resource, children []*Resource) ([]*Resource, error) {
	result := make([]*Resource, 0)
	grouped := map[enum.ParentRelationship][]*Resource{}
	for _, child := range children {
		if _, found := grouped[child.Relationship]; !found {
			grouped[child.Relationship] = make([]*Resource, 0)
		}
		group := grouped[child.Relationship]
		grouped[child.Relationship] = append(group, child)
	}
	// gen sorted keys
	keys := make([]enum.ParentRelationship, 0, len(grouped))
	for prs := range grouped {
		keys = append(keys, prs)
	}
	slices.SortFunc(keys, func(a, b enum.ParentRelationship) int {
		return enum.ParentRelationships[a].SortPriority - enum.ParentRelationships[b].SortPriority
	})

	for _, key := range keys {
		resource, err := r.sortGroupInLayer(builder, parents, key, grouped[key])
		if err != nil {
			return nil, err
		}
		result = append(result, resource...)
	}

	for _, child := range children {
		parents := slices.Clone(parents)
		sortedChild, err := r.sortLayer(builder, append(parents, child), child.Children)
		if err != nil {
			return nil, err
		}
		child.Children = sortedChild
	}
	return result, nil
}

func (r *ResourceSorter) sortGroupInLayer(builder *Builder, parents []*Resource, groupedRelationship enum.ParentRelationship, children []*Resource) ([]*Resource, error) {
	for _, strategy := range r.Strategies {
		sortedResource, err := strategy.SortChunk(builder, parents, groupedRelationship, children)
		if errors.Is(err, ErrorSortSkipped) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return sortedResource, nil
	}
	return nil, errors.New("all sorters decided skipping")
}
