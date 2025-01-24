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
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

type NameSortStrategy struct {
	PrioritizedKeys []string
	Layer           int
}

// SortChunk implements ResourceChunkSortStrategy.
func (n *NameSortStrategy) SortChunk(builder *Builder, parents []*Resource, groupedRelationship enum.ParentRelationship, chunk []*Resource) ([]*Resource, error) {
	if len(parents) != n.Layer {
		return nil, ErrorSortSkipped
	}
	sortResult := slices.Clone(chunk)
	keyToIndex := make(map[string]int)
	for index, key := range n.PrioritizedKeys {
		keyToIndex[key] = index
	}
	priority := func(key string) int {
		index, ok := keyToIndex[key]
		if ok {
			return index
		}
		return len(n.PrioritizedKeys)
	}
	sort.Slice(sortResult, func(i, j int) bool {
		a := priority(sortResult[i].ResourceName)
		b := priority(sortResult[j].ResourceName)
		reltypeA := prorityByRelationship(sortResult[i].Relationship)
		relTypeB := prorityByRelationship(sortResult[j].Relationship)
		switch {
		case a != b:
			return a < b
		case reltypeA != relTypeB:
			return reltypeA < relTypeB
		default:
			return sortResult[i].ResourceName < sortResult[j].ResourceName
		}
	})
	return sortResult, nil
}

func prorityByRelationship(rel enum.ParentRelationship) int {
	return enum.ParentRelationships[rel].SortPriority
}

var _ ResourceChunkSortStrategy = (*NameSortStrategy)(nil)

func NewNameSortStrategy(layer int, prioritizedKeys []string) *NameSortStrategy {
	return &NameSortStrategy{
		Layer:           layer,
		PrioritizedKeys: prioritizedKeys,
	}
}

// UnreachableSortStrategy is the default sort strategy catches all.
// Hitting this sorter is unexpected but implemented not to crush because of bad output from parsers.
type UnreachableSortStrategy struct {
}

// SortChunk implements ResourceChunkSortStrategy.
func (u *UnreachableSortStrategy) SortChunk(builder *Builder, parents []*Resource, groupedRelationship enum.ParentRelationship, chunk []*Resource) ([]*Resource, error) {
	cloned := slices.Clone(chunk)
	slices.SortFunc(cloned, func(a, b *Resource) int {
		return strings.Compare(a.ResourceName, b.ResourceName)
	})
	for _, c := range chunk {
		slog.Warn(fmt.Sprintf("hitting unreachable sorter: %s", c.FullResourcePath))
	}
	return cloned, nil
}

var _ ResourceChunkSortStrategy = (*UnreachableSortStrategy)(nil)

type FirstRevisionTimeSortStrategy struct {
	TargetRelationship enum.ParentRelationship
}

// SortChunk implements ResourceChunkSortStrategy.
func (b *FirstRevisionTimeSortStrategy) SortChunk(builder *Builder, parents []*Resource, groupedRelationship enum.ParentRelationship, chunk []*Resource) ([]*Resource, error) {
	if groupedRelationship != b.TargetRelationship {
		return nil, ErrorSortSkipped
	}
	cloned := slices.Clone(chunk)
	slices.SortFunc(cloned, func(a, b *Resource) int {
		abuilder := builder.GetTimelineBuilder(a.FullResourcePath)
		bbuilder := builder.GetTimelineBuilder(b.FullResourcePath)
		arevs := abuilder.timeline.Revisions
		brevs := bbuilder.timeline.Revisions
		if len(arevs) == 0 {
			return -1
		}
		if len(brevs) == 0 {
			return 1
		}
		return int(arevs[0].ChangeTime.Sub(brevs[0].ChangeTime))
	})
	return cloned, nil
}

var _ ResourceChunkSortStrategy = (*FirstRevisionTimeSortStrategy)(nil)
