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

package workbench

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
)

func TestLogTimelineCSRIndex(t *testing.T) {
	testCases := []struct {
		name      string
		maxLogID  uint32
		timelines []*cel.TimelineData
		queries   map[uint32][]uint32
	}{
		{
			name:      "empty timelines",
			maxLogID:  10,
			timelines: nil,
			queries: map[uint32][]uint32{
				1: nil,
			},
		},
		{
			name:     "multiple timelines referencing overlapping and distinct logs",
			maxLogID: 5,
			timelines: []*cel.TimelineData{
				{
					ID: 100,
					Events: []cel.EventInfo{
						{LogID: 1},
						{LogID: 2},
					},
					Revisions: []cel.RevisionInfo{
						{LogID: 1}, // Duplicate log within same timeline
						{LogID: 3},
					},
				},
				{
					ID: 200,
					Events: []cel.EventInfo{
						{LogID: 2},
					},
					Revisions: []cel.RevisionInfo{
						{LogID: 4},
						{LogID: 6}, // Out of maxLogID bounds, should be skipped
					},
				},
				{
					ID: 300,
					Events: []cel.EventInfo{
						{LogID: 2},
					},
				},
			},
			queries: map[uint32][]uint32{
				0: nil,
				1: {100},
				2: {100, 200, 300},
				3: {100},
				4: {200},
				5: nil,
				6: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idx := NewLogTimelineCSRIndex(tc.maxLogID, tc.timelines)
			for logID, want := range tc.queries {
				got := idx.GetTimelineIDs(logID)
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("GetTimelineIDs(%d) mismatch (-want +got):\n%s", logID, diff)
				}
			}
		})
	}
}

func TestLogTimelineCSRIndexLargeScale(t *testing.T) {
	// Verifies that log IDs exceeding 16M (e.g. 17,000,000) are supported without memory errors.
	maxLogID := uint32(17_000_000)
	timelines := []*cel.TimelineData{
		{
			ID: 10,
			Events: []cel.EventInfo{
				{LogID: 16_777_217},
				{LogID: 17_000_000},
			},
		},
		{
			ID: 20,
			Revisions: []cel.RevisionInfo{
				{LogID: 16_777_217},
			},
		},
	}

	idx := NewLogTimelineCSRIndex(maxLogID, timelines)

	testCases := []struct {
		logID uint32
		want  []uint32
	}{
		{
			logID: 16_777_217,
			want:  []uint32{10, 20},
		},
		{
			logID: 17_000_000,
			want:  []uint32{10},
		},
		{
			logID: 1,
			want:  nil,
		},
	}

	for _, tc := range testCases {
		got := idx.GetTimelineIDs(tc.logID)
		if diff := cmp.Diff(tc.want, got); diff != "" {
			t.Errorf("GetTimelineIDs(%d) mismatch (-want +got):\n%s", tc.logID, diff)
		}
	}
}
