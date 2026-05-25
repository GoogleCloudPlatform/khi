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
	"cmp"
	"slices"
	"strings"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

// ExtractTimelinesAndItemsChunkSource extracts the arrays of Timeline and TimelineItems protobuf messages
// from the given TimelinePathPool and TimelineRegistry.
// Empty TimelineBuilders (with no events or revisions) are omitted to save space.
func ExtractTimelinesAndItemsChunkSource(pool *TimelinePathPool, registry *TimelineRegistry) ([]*pb.Timeline, []*pb.TimelineItems) {
	items := extractTimelineItems(registry)
	timelines := extractTimelines(pool, registry)
	return timelines, items
}

// extractTimelineItems iterates over the TimelineRegistry to generate TimelineItems protobufs.
func extractTimelineItems(registry *TimelineRegistry) []*pb.TimelineItems {
	var items []*pb.TimelineItems

	for builder := range registry.Builders() {
		if proto := builder.ToProto(); proto != nil {
			items = append(items, proto)
		}
	}

	// Sort items by ID to ensure deterministic output
	slices.SortFunc(items, func(a, b *pb.TimelineItems) int {
		return cmp.Compare(a.GetId(), b.GetId())
	})

	return items
}

// extractTimelines iterates over the TimelinePathPool to generate Timeline protobufs.
// It assigns TimelineItemsId appropriately by checking if the builder exists.
// extractTimelines iterates over the TimelinePathPool to generate Timeline protobufs
// sorted in parent-to-child (pre-order tree traversal) order.
func extractTimelines(pool *TimelinePathPool, registry *TimelineRegistry) []*pb.Timeline {
	// 1. Build parent-to-children adjacency list
	parentToChildren := make(map[*TimelinePath][]*TimelinePath)
	var roots []*TimelinePath

	for path := range pool.Paths() {
		if path.Parent == nil {
			roots = append(roots, path)
		} else {
			parentToChildren[path.Parent] = append(parentToChildren[path.Parent], path)
		}
	}

	// 2. Traverse tree in pre-order (depth-first, parent-first)
	var sortedPaths []*TimelinePath
	var traverse func(p *TimelinePath)

	traverse = func(p *TimelinePath) {
		sortedPaths = append(sortedPaths, p)
		children := parentToChildren[p]
		// Sort sibling timelines deterministically by ID
		slices.SortFunc(children, func(a, b *TimelinePath) int {
			priorityDiff := int(*a.Type.SortPriority - *b.Type.SortPriority)
			if priorityDiff != 0 {
				return priorityDiff
			}
			return strings.Compare(a.Name.Resolve(), b.Name.Resolve())
		})
		for _, child := range children {
			traverse(child)
		}
	}

	// Sort root timelines deterministically by ID
	slices.SortFunc(roots, func(a, b *TimelinePath) int {
		priorityDiff := int(*a.Type.SortPriority - *b.Type.SortPriority)
		if priorityDiff != 0 {
			return priorityDiff
		}
		return strings.Compare(a.Name.Resolve(), b.Name.Resolve())
	})
	for _, r := range roots {
		traverse(r)
	}

	// 3. Generate pb.Timeline messages in the sorted order
	var timelines []*pb.Timeline
	for _, path := range sortedPaths {
		var itemsID *uint32
		if b, ok := registry.GetBuilderIfExists(path); ok && b.HasItems() {
			id := b.TimelineItemsID
			itemsID = &id
		}

		var parentID *uint32
		if path.Parent != nil {
			id := path.Parent.ID
			parentID = &id
		}

		var typeID *uint32
		if path.Type != nil {
			typeID = path.Type.Id
		}

		id := path.ID
		nameID := path.Name.id

		timelines = append(timelines, &pb.Timeline{
			Id:               &id,
			TimelineType:     typeID,
			NameStringId:     &nameID,
			TimelineItemsId:  itemsID,
			ParentTimelineId: parentID,
		})
	}

	return timelines
}
