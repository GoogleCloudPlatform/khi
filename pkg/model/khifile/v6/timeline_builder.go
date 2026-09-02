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
	"time"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// pendingEvent represents a lightweight, value-type event accumulated in TimelineBuilder pending serialization.
// It stores primitive values rather than protobuf heap pointers to reduce memory footprint and GC overhead.
type pendingEvent struct {
	// Timestamp is the time of the event copied from the associated log.
	Timestamp time.Time
	// LogID is the unique identifier of the log associated with this event.
	LogID uint32
}

// pendingFieldAnnotation represents staged field annotation IDs before serialization to protobuf.
type pendingFieldAnnotation struct {
	// FieldPathStringID is the interned string ID for the field path.
	FieldPathStringID uint32
	// MutatingWebhook contains metadata for mutating webhook annotations, or nil if none.
	MutatingWebhook *pendingMutatingWebhookInfo
}

// pendingMutatingWebhookInfo represents staged mutating webhook annotation metadata.
type pendingMutatingWebhookInfo struct {
	// ConfigurationStringID is the interned string ID of the webhook configuration name.
	ConfigurationStringID uint32
	// WebhookStringID is the interned string ID of the webhook name.
	WebhookStringID uint32
	// Round indicates the mutation round index.
	Round int32
	// Index indicates the mutation index within the round.
	Index int32
}

// pendingRevision represents a lightweight, value-type resource revision accumulated in TimelineBuilder pending serialization.
// It stores primitive IDs and values rather than protobuf heap pointers to reduce memory footprint and GC overhead.
type pendingRevision struct {
	// ChangedTime is the timestamp of the resource modification.
	ChangedTime time.Time
	// FieldAnnotations stores the list of staged field annotations.
	FieldAnnotations []pendingFieldAnnotation
	// LogID is the unique identifier of the log associated with this revision.
	LogID uint32
	// ResourceBodyStructID is the interned ID of the resource body struct (0 if none).
	ResourceBodyStructID uint32
	// PrincipalStringID is the interned string ID of the principal (0 if none).
	PrincipalStringID uint32
	// VerbType is the style ID of the revision verb (0 if none).
	VerbType uint32
	// StateType is the style ID of the revision state (0 if none).
	StateType uint32
}

// TimelineBuilder accumulates parsed data (like logs, revisions, and events) for a specific TimelinePath.
// This struct will be shared across multiple goroutines processing different chunks, so all modifications
// to its internal state must be protected by its Mutex.
type TimelineBuilder struct {
	// Path is the canonical TimelinePath associated with this builder.
	// If accessed via an alias, this field remains the target path that originally created the builder.
	Path *TimelinePath
	// TimelineItemsID is the unique identifier for the accumulated timeline items.
	TimelineItemsID uint32
	// mu protects the revisions and events slices from concurrent modification.
	mu sync.Mutex
	// revisions accumulates the history of resource changes.
	revisions []pendingRevision
	// events accumulates the logs or events associated with the timeline.
	events []pendingEvent
}

// AddEvent adds a parsed event to the builder in a thread-safe manner.
func (b *TimelineBuilder) AddEvent(e pendingEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, e)
}

// AddRevision adds a parsed revision to the builder in a thread-safe manner.
func (b *TimelineBuilder) AddRevision(r pendingRevision) {
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

// FindOldestTime returns the oldest timestamp among all accumulated events and revisions in this builder.
func (b *TimelineBuilder) FindOldestTime() (time.Time, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var oldest time.Time
	found := false

	for _, rev := range b.revisions {
		if rev.ChangedTime.IsZero() {
			continue
		}
		if !found || rev.ChangedTime.Before(oldest) {
			oldest = rev.ChangedTime
			found = true
		}
	}
	for _, ev := range b.events {
		if ev.Timestamp.IsZero() {
			continue
		}
		if !found || ev.Timestamp.Before(oldest) {
			oldest = ev.Timestamp
			found = true
		}
	}
	return oldest, found
}

// ToProto converts the accumulated data into a TimelineItems protobuf message.
// It returns nil if there are no events and no revisions.
func (b *TimelineBuilder) ToProto() *pb.TimelineItems {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.events) == 0 && len(b.revisions) == 0 {
		return nil
	}

	itemsID := b.TimelineItemsID
	return &pb.TimelineItems{
		Id:        &itemsID,
		Events:    b.convertEventsToProtoLocked(),
		Revisions: b.convertRevisionsToProtoLocked(),
	}
}

func (b *TimelineBuilder) convertEventsToProtoLocked() []*pb.Event {
	if len(b.events) == 0 {
		return nil
	}
	sortedEvents := slices.Clone(b.events)
	slices.SortStableFunc(sortedEvents, func(a, b pendingEvent) int {
		if cmpResult := a.Timestamp.Compare(b.Timestamp); cmpResult != 0 {
			return cmpResult
		}
		return cmp.Compare(a.LogID, b.LogID)
	})
	events := make([]*pb.Event, len(sortedEvents))
	for i, ev := range sortedEvents {
		id := ev.LogID
		events[i] = &pb.Event{
			LogId: &id,
		}
	}
	return events
}

func (b *TimelineBuilder) convertRevisionsToProtoLocked() []*pb.Revision {
	if len(b.revisions) == 0 {
		return nil
	}
	sortedRevs := slices.Clone(b.revisions)
	slices.SortStableFunc(sortedRevs, func(a, b pendingRevision) int {
		return a.ChangedTime.Compare(b.ChangedTime)
	})
	revisions := make([]*pb.Revision, len(sortedRevs))
	for i, r := range sortedRevs {
		revisions[i] = convertRevisionToProto(r)
	}
	return revisions
}

func convertRevisionToProto(r pendingRevision) *pb.Revision {
	var bodyStructID *uint32
	if r.ResourceBodyStructID != 0 {
		id := r.ResourceBodyStructID
		bodyStructID = &id
	}
	var principalID *uint32
	if r.PrincipalStringID != 0 {
		id := r.PrincipalStringID
		principalID = &id
	}
	var verbID *uint32
	if r.VerbType != 0 {
		id := r.VerbType
		verbID = &id
	}
	var stateID *uint32
	if r.StateType != 0 {
		id := r.StateType
		stateID = &id
	}
	var changedTime *timestamppb.Timestamp
	if !r.ChangedTime.IsZero() {
		changedTime = timestamppb.New(r.ChangedTime)
	}
	logID := r.LogID
	return &pb.Revision{
		LogId:                &logID,
		ChangedTime:          changedTime,
		ResourceBodyStructId: bodyStructID,
		PrincipalStringId:    principalID,
		VerbType:             verbID,
		StateType:            stateID,
		FieldAnnotations:     convertFieldAnnotationsToProto(r.FieldAnnotations),
	}
}

func convertFieldAnnotationsToProto(annotations []pendingFieldAnnotation) []*pb.FieldAnnotation {
	if len(annotations) == 0 {
		return nil
	}
	pbAnnotations := make([]*pb.FieldAnnotation, len(annotations))
	for i, fa := range annotations {
		fieldPathID := fa.FieldPathStringID
		pbAnn := &pb.FieldAnnotation{
			FieldPathStringId: &fieldPathID,
		}
		if fa.MutatingWebhook != nil {
			configID := fa.MutatingWebhook.ConfigurationStringID
			webhookID := fa.MutatingWebhook.WebhookStringID
			round := fa.MutatingWebhook.Round
			index := fa.MutatingWebhook.Index
			pbAnn.Payload = &pb.FieldAnnotation_MutatingWebhook{
				MutatingWebhook: &pb.MutatingWebhookInfo{
					ConfigurationStringId: &configID,
					WebhookStringId:       &webhookID,
					Round:                 &round,
					Index:                 &index,
				},
			}
		}
		pbAnnotations[i] = pbAnn
	}
	return pbAnnotations
}
