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
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func mapToSortedSlice(m map[uint32]struct{}) []uint32 {
	res := make([]uint32, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
	return res
}

func TestTimelineCELFilter(t *testing.T) {
	wb := createSampleWorkbench()
	testCases := []struct {
		name          string
		query         string
		wantTimelines []uint32
		wantErr       bool
	}{
		{
			name:          "empty query matches all timelines",
			query:         "",
			wantTimelines: []uint32{1, 2, 3, 4},
		},
		{
			name:          "matching name",
			query:         `name == "pod-a"`,
			wantTimelines: []uint32{2},
		},
		{
			name:    "invalid syntax",
			query:   `name ==`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewTimelineCELFilter(tc.query)
			filterCtx := NewFilterContext()
			err := f.Process(context.Background(), filterCtx, wb.searchIndex, nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Process() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			got := mapToSortedSlice(filterCtx.TimelineIDs)
			if diff := cmp.Diff(tc.wantTimelines, got); diff != "" {
				t.Errorf("TimelineCELFilter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIncludeDescendantsFilter(t *testing.T) {
	wb := createSampleWorkbench()
	testCases := []struct {
		name          string
		initialIDs    []uint32
		wantTimelines []uint32
	}{
		{
			name:          "include descendants of root",
			initialIDs:    []uint32{1},
			wantTimelines: []uint32{1, 2, 3, 4},
		},
		{
			name:          "include descendants of pod-b",
			initialIDs:    []uint32{3},
			wantTimelines: []uint32{3, 4},
		},
		{
			name:          "leaf node has no new descendants",
			initialIDs:    []uint32{2},
			wantTimelines: []uint32{2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewIncludeDescendantsFilter()
			filterCtx := NewFilterContext()
			for _, id := range tc.initialIDs {
				filterCtx.TimelineIDs[id] = struct{}{}
			}
			err := f.Process(context.Background(), filterCtx, wb.searchIndex, nil)
			if err != nil {
				t.Fatalf("Process() unexpected error: %v", err)
			}
			got := mapToSortedSlice(filterCtx.TimelineIDs)
			if diff := cmp.Diff(tc.wantTimelines, got); diff != "" {
				t.Errorf("IncludeDescendantsFilter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTimelineCELExclusionFilter(t *testing.T) {
	wb := createSampleWorkbench()
	testCases := []struct {
		name          string
		initialIDs    []uint32
		query         string
		wantTimelines []uint32
		wantErr       bool
	}{
		{
			name:          "empty exclusion does nothing",
			initialIDs:    []uint32{1, 2, 3, 4},
			query:         "",
			wantTimelines: []uint32{1, 2, 3, 4},
		},
		{
			name:          "exclude pod-b removes pod-b and its container descendant",
			initialIDs:    []uint32{1, 2, 3, 4},
			query:         `name == "pod-b"`,
			wantTimelines: []uint32{1, 2},
		},
		{
			name:       "invalid syntax",
			initialIDs: []uint32{1, 2},
			query:      `== invalid`,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewTimelineCELExclusionFilter(tc.query)
			filterCtx := NewFilterContext()
			for _, id := range tc.initialIDs {
				filterCtx.TimelineIDs[id] = struct{}{}
			}
			err := f.Process(context.Background(), filterCtx, wb.searchIndex, nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Process() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			got := mapToSortedSlice(filterCtx.TimelineIDs)
			if diff := cmp.Diff(tc.wantTimelines, got); diff != "" {
				t.Errorf("TimelineCELExclusionFilter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLogCELFilter(t *testing.T) {
	wb := createSampleWorkbench()
	testCases := []struct {
		name               string
		initialTimelineIDs []uint32
		query              string
		wantLogs           []uint32
		wantErr            bool
	}{
		{
			name:               "match all candidate logs on Pod A",
			initialTimelineIDs: []uint32{2},
			query:              "",
			wantLogs:           []uint32{1, 2},
		},
		{
			name:               "filter logs by severity ERROR",
			initialTimelineIDs: []uint32{1, 2, 3, 4},
			query:              `severity >= ERROR`,
			wantLogs:           []uint32{2},
		},
		{
			name:               "invalid log query syntax",
			initialTimelineIDs: []uint32{2},
			query:              `severity ==`,
			wantErr:            true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewLogCELFilter(tc.query)
			filterCtx := NewFilterContext()
			for _, id := range tc.initialTimelineIDs {
				filterCtx.TimelineIDs[id] = struct{}{}
			}
			err := f.Process(context.Background(), filterCtx, wb.searchIndex, nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Process() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			got := mapToSortedSlice(filterCtx.LogIDs)
			if diff := cmp.Diff(tc.wantLogs, got); diff != "" {
				t.Errorf("LogCELFilter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIncludeAncestorsFilter(t *testing.T) {
	wb := createSampleWorkbench()
	testCases := []struct {
		name          string
		initialIDs    []uint32
		wantTimelines []uint32
	}{
		{
			name:          "include ancestors of container-b (4 -> 3 -> 1)",
			initialIDs:    []uint32{4},
			wantTimelines: []uint32{1, 3, 4},
		},
		{
			name:          "root node has no parent",
			initialIDs:    []uint32{1},
			wantTimelines: []uint32{1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewIncludeAncestorsFilter()
			filterCtx := NewFilterContext()
			for _, id := range tc.initialIDs {
				filterCtx.TimelineIDs[id] = struct{}{}
			}
			err := f.Process(context.Background(), filterCtx, wb.searchIndex, nil)
			if err != nil {
				t.Fatalf("Process() unexpected error: %v", err)
			}
			got := mapToSortedSlice(filterCtx.TimelineIDs)
			if diff := cmp.Diff(tc.wantTimelines, got); diff != "" {
				t.Errorf("IncludeAncestorsFilter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExcludeNoLogsFilter(t *testing.T) {
	wb := createSampleWorkbench()
	testCases := []struct {
		name               string
		enabled            bool
		initialTimelineIDs []uint32
		initialLogIDs      []uint32
		wantTimelines      []uint32
	}{
		{
			name:               "disabled preserves all timelines",
			enabled:            false,
			initialTimelineIDs: []uint32{1, 2, 3, 4},
			initialLogIDs:      []uint32{2},
			wantTimelines:      []uint32{1, 2, 3, 4},
		},
		{
			name:               "enabled retains only timelines with matching logs",
			enabled:            true,
			initialTimelineIDs: []uint32{1, 2, 3, 4},
			initialLogIDs:      []uint32{2},
			wantTimelines:      []uint32{2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewExcludeNoLogsFilter(tc.enabled)
			filterCtx := NewFilterContext()
			for _, id := range tc.initialTimelineIDs {
				filterCtx.TimelineIDs[id] = struct{}{}
			}
			for _, id := range tc.initialLogIDs {
				filterCtx.LogIDs[id] = struct{}{}
			}
			err := f.Process(context.Background(), filterCtx, wb.searchIndex, nil)
			if err != nil {
				t.Fatalf("Process() unexpected error: %v", err)
			}
			got := mapToSortedSlice(filterCtx.TimelineIDs)
			if diff := cmp.Diff(tc.wantTimelines, got); diff != "" {
				t.Errorf("ExcludeNoLogsFilter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
