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
func extractTimelines(pool *TimelinePathPool, registry *TimelineRegistry) []*pb.Timeline {
	var timelines []*pb.Timeline

	for path := range pool.Paths() {
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

	// Sort timelines by ID to ensure deterministic output
	slices.SortFunc(timelines, func(a, b *pb.Timeline) int {
		return cmp.Compare(a.GetId(), b.GetId())
	})

	return timelines
}
