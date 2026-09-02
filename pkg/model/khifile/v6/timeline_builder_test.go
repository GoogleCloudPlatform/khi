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
	"testing"
	"time"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTimelineBuilder_ToProto_SortsRevisionsByTime(t *testing.T) {
	builder := &TimelineBuilder{}

	now := time.Now()
	t1 := now.Add(-2 * time.Hour)
	t2 := now.Add(-1 * time.Hour)
	t3 := now

	id1 := uint32(1)
	id2 := uint32(2)
	id3 := uint32(3)

	// Add out of chronological order
	builder.AddRevision(pendingRevision{LogID: id2, ChangedTime: t2})
	builder.AddRevision(pendingRevision{LogID: id3, ChangedTime: t3})
	builder.AddRevision(pendingRevision{LogID: id1, ChangedTime: t1})

	got := builder.ToProto()

	wantRevisions := []*pb.Revision{
		{LogId: &id1, ChangedTime: timestamppb.New(t1)},
		{LogId: &id2, ChangedTime: timestamppb.New(t2)},
		{LogId: &id3, ChangedTime: timestamppb.New(t3)},
	}

	if diff := cmp.Diff(wantRevisions, got.Revisions, protocmp.Transform()); diff != "" {
		t.Errorf("Revisions not sorted chronologically (-want +got):\n%s", diff)
	}
}

func TestTimelineBuilder_ToProto_SortsEventsByTime(t *testing.T) {
	builder := &TimelineBuilder{}

	now := time.Now()
	t1 := now.Add(-2 * time.Hour)
	t2 := now.Add(-1 * time.Hour)
	t3 := now

	id1 := uint32(1)
	id2 := uint32(2)
	id3 := uint32(3)

	// Add out of chronological order
	builder.AddEvent(pendingEvent{LogID: id2, Timestamp: t2})
	builder.AddEvent(pendingEvent{LogID: id3, Timestamp: t3})
	builder.AddEvent(pendingEvent{LogID: id1, Timestamp: t1})

	got := builder.ToProto()

	wantEvents := []*pb.Event{
		{LogId: &id1},
		{LogId: &id2},
		{LogId: &id3},
	}

	if diff := cmp.Diff(wantEvents, got.Events, protocmp.Transform()); diff != "" {
		t.Errorf("Events not sorted chronologically (-want +got):\n%s", diff)
	}
}

func TestTimelineBuilder_FindOldestTime(t *testing.T) {
	now := time.Now()
	t1 := now.Add(-2 * time.Hour)
	t2 := now.Add(-1 * time.Hour)
	t3 := now

	testCases := []struct {
		name      string
		revisions []pendingRevision
		events    []pendingEvent
		wantTime  time.Time
		wantFound bool
	}{
		{
			name:      "empty builder returns not found",
			wantFound: false,
		},
		{
			name: "finds oldest from revisions only",
			revisions: []pendingRevision{
				{ChangedTime: t2},
				{ChangedTime: t1},
				{ChangedTime: t3},
			},
			wantTime:  t1,
			wantFound: true,
		},
		{
			name: "finds oldest from events only",
			events: []pendingEvent{
				{Timestamp: t3},
				{Timestamp: t1},
				{Timestamp: t2},
			},
			wantTime:  t1,
			wantFound: true,
		},
		{
			name: "event is older than revision",
			revisions: []pendingRevision{
				{ChangedTime: t2},
			},
			events: []pendingEvent{
				{Timestamp: t1},
			},
			wantTime:  t1,
			wantFound: true,
		},
		{
			name: "revision is older than event",
			revisions: []pendingRevision{
				{ChangedTime: t1},
			},
			events: []pendingEvent{
				{Timestamp: t2},
			},
			wantTime:  t1,
			wantFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := &TimelineBuilder{}
			for _, r := range tc.revisions {
				b.AddRevision(r)
			}
			for _, e := range tc.events {
				b.AddEvent(e)
			}

			gotTime, gotFound := b.FindOldestTime()
			if gotFound != tc.wantFound {
				t.Fatalf("FindOldestTime() found mismatch: want %v, got %v", tc.wantFound, gotFound)
			}
			if tc.wantFound && !gotTime.Equal(tc.wantTime) {
				t.Errorf("FindOldestTime() time mismatch: want %v, got %v", tc.wantTime, gotTime)
			}
		})
	}
}
