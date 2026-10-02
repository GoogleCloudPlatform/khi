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

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func setupSearchLogsTestWorkbench() *Workbench {
	wb := NewWorkbench("wb-test", "test-inspection")
	wb.searchIndex = &SearchIndex{
		TimelineMap: make(map[uint32]*cel.TimelineData),
	}

	wb.searchIndex.StyleResolver = &cel.SimpleStyleResolver{
		LogTypes: map[uint32]string{1: "k8s-event"},
		Severities: map[uint32]uint32{
			1: 1, // INFO
			2: 2, // WARNING
			3: 3, // ERROR
			4: 4, // FATAL
		},
	}

	internPool := khifilev6model.NewReadonlyInternPool()
	internPool.IngestChunk(&khifilev6.InterningPoolChunk{
		Strings: []*khifilev6.InternString{
			{Id: proto.Uint32(1), Value: proto.String("summary-1")},
			{Id: proto.Uint32(2), Value: proto.String("summary-2")},
			{Id: proto.Uint32(3), Value: proto.String("summary-3")},
			{Id: proto.Uint32(4), Value: proto.String("summary-4")},
			{Id: proto.Uint32(5), Value: proto.String("summary-5")},
		},
	})
	wb.searchIndex.InternPool = internPool

	wb.searchIndex.Logs = []cel.LogData{
		{ID: 1, LogTypeID: 1, SeverityTypeID: 1, SummaryStringID: 1, Timestamp: 1000},
		{ID: 2, LogTypeID: 1, SeverityTypeID: 2, SummaryStringID: 2, Timestamp: 2000},
		{ID: 3, LogTypeID: 1, SeverityTypeID: 3, SummaryStringID: 3, Timestamp: 3000},
		{ID: 4, LogTypeID: 1, SeverityTypeID: 3, SummaryStringID: 4, Timestamp: 4000},
		{ID: 5, LogTypeID: 1, SeverityTypeID: 4, SummaryStringID: 5, Timestamp: 5000},
	}

	tl1 := &cel.TimelineData{
		ID:           1,
		ParentID:     0,
		ChildrenIDs:  []uint32{2, 3},
		Name:         "default",
		TimelineType: "Namespace",
	}

	tl2 := &cel.TimelineData{
		ID:           2,
		ParentID:     1,
		ChildrenIDs:  []uint32{4},
		Name:         "pod-a",
		TimelineType: "Pod",
		Events: []cel.EventInfo{
			{LogID: 1, Timestamp: 1000},
			{LogID: 2, Timestamp: 2000},
			{LogID: 4, Timestamp: 4000},
		},
	}

	tl3 := &cel.TimelineData{
		ID:           3,
		ParentID:     1,
		ChildrenIDs:  nil,
		Name:         "pod-b",
		TimelineType: "Pod",
		Events: []cel.EventInfo{
			{LogID: 3, Timestamp: 3000},
			{LogID: 5, Timestamp: 5000},
		},
	}

	tl4 := &cel.TimelineData{
		ID:           4,
		ParentID:     2,
		ChildrenIDs:  nil,
		Name:         "container-a1",
		TimelineType: "Container",
		Events: []cel.EventInfo{
			{LogID: 2, Timestamp: 2000},
		},
	}

	wb.searchIndex.Timelines = []*cel.TimelineData{tl1, tl2, tl3, tl4}
	wb.searchIndex.TimelineMap[1] = tl1
	wb.searchIndex.TimelineMap[2] = tl2
	wb.searchIndex.TimelineMap[3] = tl3
	wb.searchIndex.TimelineMap[4] = tl4

	wb.searchIndex.LogTimelineIndex = NewLogTimelineCSRIndex(uint32(len(wb.searchIndex.Logs)), wb.searchIndex.Timelines)

	wb.styleChunk = &khifilev6.TimelineStyleChunk{
		Severities: []*khifilev6.Severity{testSeverityInfo, testSeverityWarning, testSeverityError, testSeverityFatal},
	}
	// Build the style maps from the style chunk as BuildBaseSearchIndex does when the workbench loads.
	wb.styles = wb.buildStyleMaps()

	return wb
}

func TestSearchLogs_Aggregation(t *testing.T) {
	testCases := []struct {
		name                string
		filter              Filter
		maxTimelines        int
		maxSampleLogs       int
		wantMatchedLogCount int
		wantMatchedTLCount  int
		wantFirstMatchTime  time.Time
		wantLastMatchTime   time.Time
		wantSeverityCounts  []SeverityCount
		wantTimelineGroups  []TimelineLogGroup
		wantSampleLogs      []SampleLogEntry
	}{
		{
			name:                "full aggregation across multiple timelines with shared log",
			filter:              Filter{},
			maxTimelines:        10,
			maxSampleLogs:       10,
			wantMatchedLogCount: 5,
			wantMatchedTLCount:  3,
			wantFirstMatchTime:  time.Unix(0, 1000).UTC(),
			wantLastMatchTime:   time.Unix(0, 5000).UTC(),
			wantSeverityCounts: []SeverityCount{
				{Severity: testSeverityFatal, Count: 1},
				{Severity: testSeverityError, Count: 2},
				{Severity: testSeverityWarning, Count: 1},
				{Severity: testSeverityInfo, Count: 1},
			},
			wantTimelineGroups: []TimelineLogGroup{
				{
					TimelineID: 2,
					Segments: []TimelineSegment{
						{Type: "Namespace", Name: "default"},
						{Type: "Pod", Name: "pod-a"},
					},
					MatchedLogCount: 3,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 1},
						{Severity: testSeverityWarning, Count: 1},
						{Severity: testSeverityInfo, Count: 1},
					},
					FirstMatchTime: time.Unix(0, 1000).UTC(),
					LastMatchTime:  time.Unix(0, 4000).UTC(),
				},
				{
					TimelineID: 3,
					Segments: []TimelineSegment{
						{Type: "Namespace", Name: "default"},
						{Type: "Pod", Name: "pod-b"},
					},
					MatchedLogCount: 2,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityFatal, Count: 1},
						{Severity: testSeverityError, Count: 1},
					},
					FirstMatchTime: time.Unix(0, 3000).UTC(),
					LastMatchTime:  time.Unix(0, 5000).UTC(),
				},
				{
					TimelineID: 4,
					Segments: []TimelineSegment{
						{Type: "Namespace", Name: "default"},
						{Type: "Pod", Name: "pod-a"},
						{Type: "Container", Name: "container-a1"},
					},
					MatchedLogCount: 1,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityWarning, Count: 1},
					},
					FirstMatchTime: time.Unix(0, 2000).UTC(),
					LastMatchTime:  time.Unix(0, 2000).UTC(),
				},
			},
			wantSampleLogs: []SampleLogEntry{
				{
					LogID:       1,
					Time:        time.Unix(0, 1000).UTC(),
					Severity:    testSeverityInfo,
					LogType:     "k8s-event",
					Summary:     "summary-1",
					TimelineIDs: []uint32{2},
				},
				{
					LogID:       2,
					Time:        time.Unix(0, 2000).UTC(),
					Severity:    testSeverityWarning,
					LogType:     "k8s-event",
					Summary:     "summary-2",
					TimelineIDs: []uint32{2, 4},
				},
				{
					LogID:       3,
					Time:        time.Unix(0, 3000).UTC(),
					Severity:    testSeverityError,
					LogType:     "k8s-event",
					Summary:     "summary-3",
					TimelineIDs: []uint32{3},
				},
				{
					LogID:       4,
					Time:        time.Unix(0, 4000).UTC(),
					Severity:    testSeverityError,
					LogType:     "k8s-event",
					Summary:     "summary-4",
					TimelineIDs: []uint32{2},
				},
				{
					LogID:       5,
					Time:        time.Unix(0, 5000).UTC(),
					Severity:    testSeverityFatal,
					LogType:     "k8s-event",
					Summary:     "summary-5",
					TimelineIDs: []uint32{3},
				},
			},
		},
		{
			name:                "zero matching logs returns empty result",
			filter:              Filter{LogQuery: "severity >= 99"},
			wantMatchedLogCount: 0,
			wantMatchedTLCount:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupSearchLogsTestWorkbench()
			got, err := wb.SearchLogs(context.Background(), tc.filter, tc.maxTimelines, tc.maxSampleLogs)
			if err != nil {
				t.Fatalf("SearchLogs() unexpected error = %v", err)
			}
			if got.MatchedLogCount != tc.wantMatchedLogCount {
				t.Errorf("MatchedLogCount = %d, want %d", got.MatchedLogCount, tc.wantMatchedLogCount)
			}
			if got.MatchedTimelineCount != tc.wantMatchedTLCount {
				t.Errorf("MatchedTimelineCount = %d, want %d", got.MatchedTimelineCount, tc.wantMatchedTLCount)
			}
			if !got.FirstMatchTime.Equal(tc.wantFirstMatchTime) {
				t.Errorf("FirstMatchTime = %v, want %v", got.FirstMatchTime, tc.wantFirstMatchTime)
			}
			if !got.LastMatchTime.Equal(tc.wantLastMatchTime) {
				t.Errorf("LastMatchTime = %v, want %v", got.LastMatchTime, tc.wantLastMatchTime)
			}
			if diff := cmp.Diff(tc.wantSeverityCounts, got.SeverityCounts, protocmp.Transform()); diff != "" {
				t.Errorf("SeverityCounts mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantTimelineGroups, got.TimelineGroups, protocmp.Transform()); diff != "" {
				t.Errorf("TimelineGroups mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantSampleLogs, got.SampleLogs, protocmp.Transform()); diff != "" {
				t.Errorf("SampleLogs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSearchLogs_TimelineFiltering(t *testing.T) {
	testCases := []struct {
		name                string
		filter              Filter
		wantMatchedLogCount int
		wantMatchedTLCount  int
		wantTimelineIDs     []uint32
		wantLog2TimelineIDs []uint32
	}{
		{
			name: "timeline exclusion query excludes timeline from groups and sample logs",
			filter: Filter{
				TimelineExclusionQuery: "name == 'container-a1'",
			},
			wantMatchedLogCount: 5,
			wantMatchedTLCount:  2,
			wantTimelineIDs:     []uint32{2, 3},
			wantLog2TimelineIDs: []uint32{2},
		},
		{
			name: "timeline include query filters to matching timeline",
			filter: Filter{
				TimelineQuery: "name == 'pod-b'",
			},
			wantMatchedLogCount: 2,
			wantMatchedTLCount:  1,
			wantTimelineIDs:     []uint32{3},
			wantLog2TimelineIDs: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupSearchLogsTestWorkbench()
			got, err := wb.SearchLogs(context.Background(), tc.filter, 10, 10)
			if err != nil {
				t.Fatalf("SearchLogs() unexpected error = %v", err)
			}
			if got.MatchedLogCount != tc.wantMatchedLogCount {
				t.Errorf("MatchedLogCount = %d, want %d", got.MatchedLogCount, tc.wantMatchedLogCount)
			}
			if got.MatchedTimelineCount != tc.wantMatchedTLCount {
				t.Errorf("MatchedTimelineCount = %d, want %d", got.MatchedTimelineCount, tc.wantMatchedTLCount)
			}

			var gotTLIDs []uint32
			for _, g := range got.TimelineGroups {
				gotTLIDs = append(gotTLIDs, g.TimelineID)
			}
			if diff := cmp.Diff(tc.wantTimelineIDs, gotTLIDs); diff != "" {
				t.Errorf("TimelineGroups IDs mismatch (-want +got):\n%s", diff)
			}

			var log2TLs []uint32
			for _, s := range got.SampleLogs {
				if s.LogID == 2 {
					log2TLs = s.TimelineIDs
					break
				}
			}
			if diff := cmp.Diff(tc.wantLog2TimelineIDs, log2TLs); diff != "" {
				t.Errorf("Log 2 TimelineIDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSearchLogs_TruncationAndSampling(t *testing.T) {
	testCases := []struct {
		name                string
		setupWorkbench      func() *Workbench
		maxTimelines        int
		maxSampleLogs       int
		wantMatchedLogCount int
		wantMatchedTLCount  int
		wantGroupCount      int
		wantSampleLogCount  int
		wantSampleLogIDs    []uint32
	}{
		{
			name:                "truncation by maxTimelines retains total count while limiting groups",
			maxTimelines:        2,
			maxSampleLogs:       10,
			wantMatchedLogCount: 5,
			wantMatchedTLCount:  3,
			wantGroupCount:      2,
			wantSampleLogCount:  5,
			wantSampleLogIDs:    []uint32{1, 2, 3, 4, 5},
		},
		{
			name:                "evenly spaced sampling picks first middle and last logs",
			maxTimelines:        10,
			maxSampleLogs:       3,
			wantMatchedLogCount: 5,
			wantMatchedTLCount:  3,
			wantGroupCount:      3,
			wantSampleLogCount:  3,
			wantSampleLogIDs:    []uint32{1, 3, 5},
		},
		{
			name: "time-based sampling avoids clustering in an initial log burst",
			setupWorkbench: func() *Workbench {
				wb := setupSearchLogsTestWorkbench()
				// Place logs 1, 2, and 3 in a tight burst near t=1000, log 4 at the midpoint t=3000, and log 5 at t=5000.
				wb.searchIndex.Logs[0].Timestamp = 1000
				wb.searchIndex.Logs[1].Timestamp = 1001
				wb.searchIndex.Logs[2].Timestamp = 1002
				wb.searchIndex.Logs[3].Timestamp = 3000
				wb.searchIndex.Logs[4].Timestamp = 5000
				return wb
			},
			maxTimelines:        10,
			maxSampleLogs:       3,
			wantMatchedLogCount: 5,
			wantMatchedTLCount:  3,
			wantGroupCount:      3,
			wantSampleLogCount:  3,
			wantSampleLogIDs:    []uint32{1, 4, 5},
		},
		{
			name: "fallback to index-based sampling when all logs have identical timestamps",
			setupWorkbench: func() *Workbench {
				wb := setupSearchLogsTestWorkbench()
				for i := range wb.searchIndex.Logs {
					wb.searchIndex.Logs[i].Timestamp = 1000
				}
				return wb
			},
			maxTimelines:        10,
			maxSampleLogs:       3,
			wantMatchedLogCount: 5,
			wantMatchedTLCount:  3,
			wantGroupCount:      3,
			wantSampleLogCount:  3,
			wantSampleLogIDs:    []uint32{1, 3, 5},
		},
		{
			name: "time-based sampling returns strictly increasing indices with duplicate timestamps",
			setupWorkbench: func() *Workbench {
				wb := setupSearchLogsTestWorkbench()
				// Duplicate timestamps across intermediate logs.
				wb.searchIndex.Logs[0].Timestamp = 1000
				wb.searchIndex.Logs[1].Timestamp = 3000
				wb.searchIndex.Logs[2].Timestamp = 3000
				wb.searchIndex.Logs[3].Timestamp = 3000
				wb.searchIndex.Logs[4].Timestamp = 5000
				return wb
			},
			maxTimelines:        10,
			maxSampleLogs:       4,
			wantMatchedLogCount: 5,
			wantMatchedTLCount:  3,
			wantGroupCount:      3,
			wantSampleLogCount:  4,
			wantSampleLogIDs:    []uint32{1, 2, 4, 5},
		},
		{
			name:                "sampling single log picks the first log",
			maxTimelines:        10,
			maxSampleLogs:       1,
			wantMatchedLogCount: 5,
			wantMatchedTLCount:  3,
			wantGroupCount:      3,
			wantSampleLogCount:  1,
			wantSampleLogIDs:    []uint32{1},
		},
		{
			name:                "defaults used when max values are non-positive",
			maxTimelines:        0,
			maxSampleLogs:       -1,
			wantMatchedLogCount: 5,
			wantMatchedTLCount:  3,
			wantGroupCount:      3,
			wantSampleLogCount:  5,
			wantSampleLogIDs:    []uint32{1, 2, 3, 4, 5},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupSearchLogsTestWorkbench()
			if tc.setupWorkbench != nil {
				wb = tc.setupWorkbench()
			}
			got, err := wb.SearchLogs(context.Background(), Filter{}, tc.maxTimelines, tc.maxSampleLogs)
			if err != nil {
				t.Fatalf("SearchLogs() unexpected error = %v", err)
			}
			if got.MatchedLogCount != tc.wantMatchedLogCount {
				t.Errorf("MatchedLogCount = %d, want %d", got.MatchedLogCount, tc.wantMatchedLogCount)
			}
			if got.MatchedTimelineCount != tc.wantMatchedTLCount {
				t.Errorf("MatchedTimelineCount = %d, want %d", got.MatchedTimelineCount, tc.wantMatchedTLCount)
			}
			if len(got.TimelineGroups) != tc.wantGroupCount {
				t.Errorf("len(TimelineGroups) = %d, want %d", len(got.TimelineGroups), tc.wantGroupCount)
			}
			if len(got.SampleLogs) != tc.wantSampleLogCount {
				t.Errorf("len(SampleLogs) = %d, want %d", len(got.SampleLogs), tc.wantSampleLogCount)
			}

			var gotLogIDs []uint32
			for _, s := range got.SampleLogs {
				gotLogIDs = append(gotLogIDs, s.LogID)
			}
			if diff := cmp.Diff(tc.wantSampleLogIDs, gotLogIDs); diff != "" {
				t.Errorf("SampleLogs IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSearchLogs_Errors(t *testing.T) {
	testCases := []struct {
		name           string
		filter         Filter
		setupWorkbench func() *Workbench
		wantErr        bool
		wantErrTarget  error
	}{
		{
			name: "invalid CEL in log query returns validation error",
			filter: Filter{
				LogQuery: "invalid &&",
			},
			setupWorkbench: setupSearchLogsTestWorkbench,
			wantErr:        true,
		},
		{
			name: "invalid CEL in timeline query returns validation error",
			filter: Filter{
				TimelineQuery: "invalid &&",
			},
			setupWorkbench: setupSearchLogsTestWorkbench,
			wantErr:        true,
		},
		{
			name:   "closed workbench returns ErrWorkbenchClosed",
			filter: Filter{},
			setupWorkbench: func() *Workbench {
				wb := setupSearchLogsTestWorkbench()
				wb.Close()
				return wb
			},
			wantErr:       true,
			wantErrTarget: ErrWorkbenchClosed,
		},
		{
			name:   "unready search index returns error",
			filter: Filter{},
			setupWorkbench: func() *Workbench {
				wb := setupSearchLogsTestWorkbench()
				wb.searchIndex = nil
				return wb
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := tc.setupWorkbench()
			_, err := wb.SearchLogs(context.Background(), tc.filter, 10, 10)
			if (err != nil) != tc.wantErr {
				t.Errorf("SearchLogs() err = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("SearchLogs() err = %v, wantErrTarget = %v", err, tc.wantErrTarget)
			}
		})
	}
}
