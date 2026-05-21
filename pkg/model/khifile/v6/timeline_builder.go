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
	"sync"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

// TimelineBuilder accumulates parsed data (like logs, revisions, and events) for a specific TimelinePath.
// This struct will be shared across multiple goroutines processing different chunks, so all modifications
// to its internal state must be protected by its Mutex.
type TimelineBuilder struct {
	// Path is the canonical TimelinePath associated with this builder.
	// If accessed via an alias, this field remains the target path that originally created the builder.
	Path *TimelinePath
	// TimelineItemsID is the unique identifier for the accumulated timeline items.
	TimelineItemsID uint32
	// internPool is the pool used to intern strings when accumulating data.
	internPool *InternPool
	// mu protects the revisions and events slices from concurrent modification.
	mu sync.Mutex
	// revisions accumulates the history of resource changes.
	revisions []*pb.Revision
	// events accumulates the logs or events associated with the timeline.
	events []*pb.Event
}

// AddEvent adds a parsed event to the builder in a thread-safe manner.
func (b *TimelineBuilder) AddEvent(e *pb.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, e)
}

// AddRevision adds a parsed revision to the builder in a thread-safe manner.
func (b *TimelineBuilder) AddRevision(r *pb.Revision) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.revisions = append(b.revisions, r)
}

// HasItems returns true if the builder has accumulated any events or revisions.
func (b *TimelineBuilder) HasItems() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.events) > 0 || len(b.revisions) > 0
}

// ToProto converts the accumulated data into a TimelineItems protobuf message.
// It returns nil if there are no events and no revisions.
func (b *TimelineBuilder) ToProto() *pb.TimelineItems {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.events) == 0 && len(b.revisions) == 0 {
		return nil
	}

	var events []*pb.Event
	if len(b.events) > 0 {
		events = slices.Clone(b.events)
		slices.SortStableFunc(events, func(a, b *pb.Event) int {
			// TODO: Sort events by time of their associated log after implementing the log registry.
			return cmp.Compare(a.GetLogId(), b.GetLogId())
		})
	}

	var revisions []*pb.Revision
	if len(b.revisions) > 0 {
		revisions = slices.Clone(b.revisions)
		slices.SortStableFunc(revisions, func(a, b *pb.Revision) int {
			return a.GetChangedTime().AsTime().Compare(b.GetChangedTime().AsTime())
		})
	}

	itemsID := b.TimelineItemsID
	return &pb.TimelineItems{
		Id:        &itemsID,
		Events:    events,
		Revisions: revisions,
	}
}
