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

package cel

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTimelineData_HasActivityInRange(t *testing.T) {
	sampleTimeline := &TimelineData{
		Events: []EventInfo{
			{LogID: 1, Timestamp: 100},
			{LogID: 2, Timestamp: 200},
			{LogID: 3, Timestamp: 200},
			{LogID: 4, Timestamp: 300},
		},
		Revisions: []RevisionInfo{
			{LogID: 5, ChangedTime: 250},
			{LogID: 6, ChangedTime: 400},
			{LogID: 7, ChangedTime: 400},
			{LogID: 8, ChangedTime: 500},
		},
	}

	testCases := []struct {
		name        string
		timeline    *TimelineData
		startTimeNs int64
		endTimeNs   int64
		want        bool
	}{
		{
			name:        "nil timeline",
			timeline:    nil,
			startTimeNs: 100,
			endTimeNs:   200,
			want:        false,
		},
		{
			name:        "empty timeline",
			timeline:    &TimelineData{},
			startTimeNs: 100,
			endTimeNs:   200,
			want:        false,
		},
		{
			name:        "invalid range where start is greater than end",
			timeline:    sampleTimeline,
			startTimeNs: 300,
			endTimeNs:   200,
			want:        false,
		},
		{
			name:        "range before any activity",
			timeline:    sampleTimeline,
			startTimeNs: 10,
			endTimeNs:   50,
			want:        false,
		},
		{
			name:        "range after all activity",
			timeline:    sampleTimeline,
			startTimeNs: 600,
			endTimeNs:   700,
			want:        false,
		},
		{
			name: "range between activities with no events or revisions",
			timeline: &TimelineData{
				Events: []EventInfo{
					{LogID: 1, Timestamp: 100},
				},
				Revisions: []RevisionInfo{
					{LogID: 2, ChangedTime: 400},
				},
			},
			startTimeNs: 200,
			endTimeNs:   300,
			want:        false,
		},
		{
			name:        "exact match on event start boundary",
			timeline:    sampleTimeline,
			startTimeNs: 100,
			endTimeNs:   150,
			want:        true,
		},
		{
			name:        "exact match on event end boundary",
			timeline:    sampleTimeline,
			startTimeNs: 50,
			endTimeNs:   100,
			want:        true,
		},
		{
			name:        "hit duplicate event timestamps",
			timeline:    sampleTimeline,
			startTimeNs: 200,
			endTimeNs:   200,
			want:        true,
		},
		{
			name:        "hit revision only",
			timeline:    sampleTimeline,
			startTimeNs: 240,
			endTimeNs:   260,
			want:        true,
		},
		{
			name:        "exact match on revision end boundary",
			timeline:    sampleTimeline,
			startTimeNs: 450,
			endTimeNs:   500,
			want:        true,
		},
		{
			name:        "broad range covering all activity",
			timeline:    sampleTimeline,
			startTimeNs: 0,
			endTimeNs:   1000,
			want:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.timeline.HasActivityInRange(tc.startTimeNs, tc.endTimeNs)
			if got != tc.want {
				t.Errorf("HasActivityInRange(%d, %d) = %v, want %v", tc.startTimeNs, tc.endTimeNs, got, tc.want)
			}
		})
	}
}

func TestTimelineData_ForEachLogIDInRange(t *testing.T) {
	sampleTimeline := &TimelineData{
		Events: []EventInfo{
			{LogID: 1, Timestamp: 100},
			{LogID: 2, Timestamp: 200},
			{LogID: 3, Timestamp: 200},
			{LogID: 4, Timestamp: 300},
		},
		Revisions: []RevisionInfo{
			{LogID: 5, ChangedTime: 200},
			{LogID: 6, ChangedTime: 250},
			{LogID: 7, ChangedTime: 400},
			{LogID: 8, ChangedTime: 400},
		},
	}

	testCases := []struct {
		name        string
		timeline    *TimelineData
		startTimeNs int64
		endTimeNs   int64
		maxCollect  int
		wantLogIDs  []uint32
	}{
		{
			name:        "nil timeline",
			timeline:    nil,
			startTimeNs: 100,
			endTimeNs:   300,
			maxCollect:  -1,
			wantLogIDs:  nil,
		},
		{
			name:        "empty timeline",
			timeline:    &TimelineData{},
			startTimeNs: 100,
			endTimeNs:   300,
			maxCollect:  -1,
			wantLogIDs:  nil,
		},
		{
			name:        "invalid range where start is greater than end",
			timeline:    sampleTimeline,
			startTimeNs: 300,
			endTimeNs:   200,
			maxCollect:  -1,
			wantLogIDs:  nil,
		},
		{
			name:        "range before all activity",
			timeline:    sampleTimeline,
			startTimeNs: 10,
			endTimeNs:   50,
			maxCollect:  -1,
			wantLogIDs:  nil,
		},
		{
			name:        "range after all activity",
			timeline:    sampleTimeline,
			startTimeNs: 500,
			endTimeNs:   600,
			maxCollect:  -1,
			wantLogIDs:  nil,
		},
		{
			name:        "range matches single point with duplicates across events and revisions",
			timeline:    sampleTimeline,
			startTimeNs: 200,
			endTimeNs:   200,
			maxCollect:  -1,
			wantLogIDs:  []uint32{2, 3, 5},
		},
		{
			name:        "inclusive boundaries collecting subset of events and revisions",
			timeline:    sampleTimeline,
			startTimeNs: 200,
			endTimeNs:   300,
			maxCollect:  -1,
			wantLogIDs:  []uint32{2, 3, 4, 5, 6},
		},
		{
			name:        "collect all logs within wide range",
			timeline:    sampleTimeline,
			startTimeNs: 50,
			endTimeNs:   500,
			maxCollect:  -1,
			wantLogIDs:  []uint32{1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			name:        "early callback termination after 2 items",
			timeline:    sampleTimeline,
			startTimeNs: 50,
			endTimeNs:   500,
			maxCollect:  2,
			wantLogIDs:  []uint32{1, 2},
		},
		{
			name:        "early callback termination stopping during revisions",
			timeline:    sampleTimeline,
			startTimeNs: 50,
			endTimeNs:   500,
			maxCollect:  5,
			wantLogIDs:  []uint32{1, 2, 3, 4, 5},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var gotLogIDs []uint32
			count := 0
			tc.timeline.ForEachLogIDInRange(tc.startTimeNs, tc.endTimeNs, func(logID uint32) bool {
				gotLogIDs = append(gotLogIDs, logID)
				count++
				if tc.maxCollect >= 0 && count >= tc.maxCollect {
					return false
				}
				return true
			})

			if diff := cmp.Diff(tc.wantLogIDs, gotLogIDs); diff != "" {
				t.Errorf("ForEachLogIDInRange() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
