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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestTimeRangeFilter(t *testing.T) {
	wb := createSampleWorkbench()
	t1000 := time.Unix(0, 1000)
	t1500 := time.Unix(0, 1500)
	t2000 := time.Unix(0, 2000)
	t2500 := time.Unix(0, 2500)
	t3500 := time.Unix(0, 3500)
	t4000 := time.Unix(0, 4000)
	t5000 := time.Unix(0, 5000)

	testCases := []struct {
		name             string
		initialTimelines []uint32
		startTime        *time.Time
		endTime          *time.Time
		wantHasRange     bool
		wantTimelines    []uint32
	}{
		{
			name:             "nil boundaries matches all timelines",
			initialTimelines: []uint32{1, 2, 3, 4},
			startTime:        nil,
			endTime:          nil,
			wantHasRange:     false,
			wantTimelines:    []uint32{1, 2, 3, 4},
		},
		{
			name:             "range matching pod-a only",
			initialTimelines: []uint32{1, 2, 3, 4},
			startTime:        &t1000,
			endTime:          &t2000,
			wantHasRange:     true,
			wantTimelines:    []uint32{2},
		},
		{
			name:             "range matching pod-b and container-b",
			initialTimelines: []uint32{1, 2, 3, 4},
			startTime:        &t2500,
			endTime:          &t3500,
			wantHasRange:     true,
			wantTimelines:    []uint32{3, 4},
		},
		{
			name:             "range matching all active timelines",
			initialTimelines: []uint32{1, 2, 3, 4},
			startTime:        &t1000,
			endTime:          &t3500,
			wantHasRange:     true,
			wantTimelines:    []uint32{2, 3, 4},
		},
		{
			name:             "unbounded start matches early events",
			initialTimelines: []uint32{1, 2, 3, 4},
			startTime:        nil,
			endTime:          &t1500,
			wantHasRange:     true,
			wantTimelines:    []uint32{2},
		},
		{
			name:             "unbounded end matches late events",
			initialTimelines: []uint32{1, 2, 3, 4},
			startTime:        &t2500,
			endTime:          nil,
			wantHasRange:     true,
			wantTimelines:    []uint32{3, 4},
		},
		{
			name:             "range with no matching activity",
			initialTimelines: []uint32{1, 2, 3, 4},
			startTime:        &t4000,
			endTime:          &t5000,
			wantHasRange:     true,
			wantTimelines:    []uint32{},
		},
		{
			name:             "empty initial timelines does not resurrect timelines",
			initialTimelines: []uint32{},
			startTime:        &t1000,
			endTime:          &t3500,
			wantHasRange:     true,
			wantTimelines:    []uint32{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewTimeRangeFilter(tc.startTime, tc.endTime)
			if got := f.Name(); got != "Time range filter" {
				t.Errorf("Name() = %q, want %q", got, "Time range filter")
			}

			filterCtx := NewFilterContext()
			filterCtx.TimelineIDs.AddMany(tc.initialTimelines)
			var reported bool
			err := f.Process(context.Background(), filterCtx, wb.searchIndex, func(stage string, current, total uint32) error {
				reported = true
				return nil
			})
			if err != nil {
				t.Fatalf("Process() unexpected error = %v", err)
			}

			if filterCtx.HasTimeRange != tc.wantHasRange {
				t.Errorf("HasTimeRange = %v, want %v", filterCtx.HasTimeRange, tc.wantHasRange)
			}

			if tc.wantHasRange {
				if tc.startTime != nil && filterCtx.StartTimeNs != tc.startTime.UnixNano() {
					t.Errorf("StartTimeNs = %v, want %v", filterCtx.StartTimeNs, tc.startTime.UnixNano())
				}
				if tc.endTime != nil && filterCtx.EndTimeNs != tc.endTime.UnixNano() {
					t.Errorf("EndTimeNs = %v, want %v", filterCtx.EndTimeNs, tc.endTime.UnixNano())
				}
			}

			if !reported {
				t.Errorf("Process() expected progress report callback to be called")
			}

			got := bitmapToSlice(filterCtx.TimelineIDs)
			if diff := cmp.Diff(tc.wantTimelines, got); diff != "" {
				t.Errorf("TimelineIDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTimeRangeFilter_Cancellation(t *testing.T) {
	wb := createSampleWorkbench()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	now := time.Now()
	f := NewTimeRangeFilter(&now, &now)
	filterCtx := NewFilterContext()
	filterCtx.TimelineIDs.Add(1)

	err := f.Process(ctx, filterCtx, wb.searchIndex, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Process() error = %v, want %v", err, context.Canceled)
	}
}
