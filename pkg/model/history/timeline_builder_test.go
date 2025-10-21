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
	"testing"
	"time"
)

func TestTimelineBuilder_GetRevision(t *testing.T) {
	t0 := time.Now()
	revisions := []*ResourceRevision{
		{ChangeTime: t0},
		{ChangeTime: t0.Add(10 * time.Second)},
		{ChangeTime: t0.Add(20 * time.Second)},
		{ChangeTime: t0.Add(30 * time.Second)},
		{ChangeTime: t0.Add(40 * time.Second)},
	}

	builder := &TimelineBuilder{
		timeline: &ResourceTimeline{
			Revisions: revisions,
		},
		sorted: true,
	}

	// Test GetRevisionBefore
	beforeTestCases := []struct {
		name     string
		time     time.Time
		expected *ResourceRevision
	}{
		{"before all", t0.Add(-5 * time.Second), nil}, // Expect nil because there is no revision before the first one.
		{"exact match", t0.Add(20 * time.Second), revisions[2]},
		{"between two", t0.Add(25 * time.Second), revisions[2]},
		{"after all", t0.Add(50 * time.Second), revisions[4]},
		{"first element", t0, revisions[0]},
		{"last element", t0.Add(40 * time.Second), revisions[4]},
	}

	for _, tc := range beforeTestCases {
		t.Run("GetRevisionBefore/"+tc.name, func(t *testing.T) {
			got := builder.GetRevisionBefore(tc.time)
			if got != tc.expected {
				t.Errorf("GetRevisionBefore(%v) = %v, want %v", tc.time, got, tc.expected)
			}
		})
	}

	// Test GetRevisionAfter
	afterTestCases := []struct {
		name     string
		time     time.Time
		expected *ResourceRevision
	}{
		{"before all", t0.Add(-5 * time.Second), revisions[0]},
		{"exact match", t0.Add(20 * time.Second), revisions[2]},
		{"between two", t0.Add(25 * time.Second), revisions[3]},
		{"after all", t0.Add(50 * time.Second), nil}, // Expect nil because there is no revision after the last one.
		{"first element", t0, revisions[0]},
		{"last element", t0.Add(40 * time.Second), revisions[4]},
	}

	for _, tc := range afterTestCases {
		t.Run("GetRevisionAfter/"+tc.name, func(t *testing.T) {
			got := builder.GetRevisionAfter(tc.time)
			if got != tc.expected {
				t.Errorf("GetRevisionAfter(%v) = %v, want %v", tc.time, got, tc.expected)
			}
		})
	}

	// Test with empty revisions
	emptyBuilder := &TimelineBuilder{
		timeline: &ResourceTimeline{
			Revisions: []*ResourceRevision{},
		},
		sorted: true,
	}
	t.Run("GetRevisionBefore/empty", func(t *testing.T) {
		if got := emptyBuilder.GetRevisionBefore(t0); got != nil {
			t.Errorf("GetRevisionBefore on empty revisions should be nil, got %v", got)
		}
	})
	t.Run("GetRevisionAfter/empty", func(t *testing.T) {
		if got := emptyBuilder.GetRevisionAfter(t0); got != nil {
			t.Errorf("GetRevisionAfter on empty revisions should be nil, got %v", got)
		}
	})
}
