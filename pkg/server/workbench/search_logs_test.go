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
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func setupSearchLogsTestWorkbench() *Workbench {
	logs := []cel.LogData{
		{ID: 1, LogTypeID: 1, SeverityTypeID: 1, SummaryStringID: 1, Timestamp: 1000},
		{ID: 2, LogTypeID: 1, SeverityTypeID: 2, SummaryStringID: 2, Timestamp: 2000},
		{ID: 3, LogTypeID: 1, SeverityTypeID: 3, SummaryStringID: 3, Timestamp: 3000},
		{ID: 4, LogTypeID: 1, SeverityTypeID: 3, SummaryStringID: 4, Timestamp: 4000},
		{ID: 5, LogTypeID: 1, SeverityTypeID: 4, SummaryStringID: 5, Timestamp: 5000},
	}
	timelines := []*cel.TimelineData{
		{
			ID:           1,
			ParentID:     0,
			ChildrenIDs:  []uint32{2, 3},
			Name:         "default",
			TimelineType: "Namespace",
		},
		{
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
		},
		{
			ID:           3,
			ParentID:     1,
			Name:         "pod-b",
			TimelineType: "Pod",
			Events: []cel.EventInfo{
				{LogID: 3, Timestamp: 3000},
				{LogID: 5, Timestamp: 5000},
			},
		},
		{
			ID:           4,
			ParentID:     2,
			Name:         "container-a1",
			TimelineType: "Container",
			Events: []cel.EventInfo{
				{LogID: 2, Timestamp: 2000},
			},
		},
	}
	severities := []*khifilev6.Severity{testSeverityInfo, testSeverityWarning, testSeverityError, testSeverityFatal}
	wb := newTestIndexedWorkbench(logs, timelines, severities)
	wb.internPool.IngestChunk(&khifilev6.InterningPoolChunk{
		Strings: []*khifilev6.InternString{
			{Id: proto.Uint32(1), Value: proto.String("summary-1")},
			{Id: proto.Uint32(2), Value: proto.String("summary-2")},
			{Id: proto.Uint32(3), Value: proto.String("summary-3")},
			{Id: proto.Uint32(4), Value: proto.String("summary-4")},
			{Id: proto.Uint32(5), Value: proto.String("summary-5")},
		},
	})
	return wb
}

func TestSearchLogs_Aggregation(t *testing.T) {
	testCases := []struct {
		name          string
		filter        Filter
		maxTimelines  int
		maxSampleLogs int
		want          *LogSearchResult
	}{
		{
			name:          "full aggregation across multiple timelines with shared log",
			filter:        Filter{},
			maxTimelines:  10,
			maxSampleLogs: 10,
			want: &LogSearchResult{
				MatchedLogCount:      5,
				MatchedTimelineCount: 3,
				FirstMatchTime:       time.Unix(0, 1000).UTC(),
				LastMatchTime:        time.Unix(0, 5000).UTC(),
				SeverityCounts: []SeverityCount{
					{Severity: testSeverityFatal, Count: 1},
					{Severity: testSeverityError, Count: 2},
					{Severity: testSeverityWarning, Count: 1},
					{Severity: testSeverityInfo, Count: 1},
				},
				TimelineGroups: []TimelineLogGroup{
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
				SampleLogs: []SampleLogEntry{
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
		},
		{
			name:   "zero matching logs returns empty result",
			filter: Filter{LogQuery: "severity >= 99"},
			want:   &LogSearchResult{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupSearchLogsTestWorkbench()
			got, err := wb.SearchLogs(context.Background(), tc.filter, tc.maxTimelines, tc.maxSampleLogs)
			if err != nil {
				t.Fatalf("SearchLogs() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got, protocmp.Transform(), cmpopts.IgnoreFields(LogSearchResult{}, "Applied", "FilterVocabulary")); diff != "" {
				t.Errorf("SearchLogs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSearchLogs_TimelineFiltering(t *testing.T) {
	type wantTimelineFilterResult struct {
		MatchedLogCount int
		MatchedTLCount  int
		TimelineIDs     []uint32
		Log2TimelineIDs []uint32
	}

	testCases := []struct {
		name   string
		filter Filter
		want   wantTimelineFilterResult
	}{
		{
			name: "timeline exclusion query excludes timeline from groups and sample logs",
			filter: Filter{
				TimelineExclusionQuery: "name == 'container-a1'",
			},
			want: wantTimelineFilterResult{
				MatchedLogCount: 5,
				MatchedTLCount:  2,
				TimelineIDs:     []uint32{2, 3},
				Log2TimelineIDs: []uint32{2},
			},
		},
		{
			name: "timeline include query filters to matching timeline",
			filter: Filter{
				TimelineQuery: "name == 'pod-b'",
			},
			want: wantTimelineFilterResult{
				MatchedLogCount: 2,
				MatchedTLCount:  1,
				TimelineIDs:     []uint32{3},
				Log2TimelineIDs: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupSearchLogsTestWorkbench()
			got, err := wb.SearchLogs(context.Background(), tc.filter, 10, 10)
			if err != nil {
				t.Fatalf("SearchLogs() unexpected error = %v", err)
			}

			var gotTLIDs []uint32
			for _, g := range got.TimelineGroups {
				gotTLIDs = append(gotTLIDs, g.TimelineID)
			}
			var log2TLs []uint32
			for _, s := range got.SampleLogs {
				if s.LogID == 2 {
					log2TLs = s.TimelineIDs
					break
				}
			}
			gotSummary := wantTimelineFilterResult{
				MatchedLogCount: got.MatchedLogCount,
				MatchedTLCount:  got.MatchedTimelineCount,
				TimelineIDs:     gotTLIDs,
				Log2TimelineIDs: log2TLs,
			}
			if diff := cmp.Diff(tc.want, gotSummary); diff != "" {
				t.Errorf("SearchLogs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSearchLogs_TruncationAndSampling(t *testing.T) {
	type wantSamplingResult struct {
		MatchedTLCount int
		GroupCount     int
		SampleLogIDs   []uint32
	}

	testCases := []struct {
		name          string
		modify        func(*Workbench)
		maxTimelines  int
		maxSampleLogs int
		want          wantSamplingResult
	}{
		{
			name:          "truncation by maxTimelines retains total count while limiting groups",
			maxTimelines:  2,
			maxSampleLogs: 10,
			want: wantSamplingResult{
				MatchedTLCount: 3,
				GroupCount:     2,
				SampleLogIDs:   []uint32{1, 2, 3, 4, 5},
			},
		},
		{
			name:          "evenly spaced sampling picks first middle and last logs",
			maxTimelines:  10,
			maxSampleLogs: 3,
			want: wantSamplingResult{
				MatchedTLCount: 3,
				GroupCount:     3,
				SampleLogIDs:   []uint32{1, 3, 5},
			},
		},
		{
			name: "time-based sampling avoids clustering in an initial log burst",
			modify: func(wb *Workbench) {
				// Place logs 1, 2, and 3 in a tight burst near t=1000, log 4 at the midpoint t=3000, and log 5 at t=5000.
				wb.searchIndex.Logs[0].Timestamp = 1000
				wb.searchIndex.Logs[1].Timestamp = 1001
				wb.searchIndex.Logs[2].Timestamp = 1002
				wb.searchIndex.Logs[3].Timestamp = 3000
				wb.searchIndex.Logs[4].Timestamp = 5000
			},
			maxTimelines:  10,
			maxSampleLogs: 3,
			want: wantSamplingResult{
				MatchedTLCount: 3,
				GroupCount:     3,
				SampleLogIDs:   []uint32{1, 4, 5},
			},
		},
		{
			name: "fallback to index-based sampling when all logs have identical timestamps",
			modify: func(wb *Workbench) {
				for i := range wb.searchIndex.Logs {
					wb.searchIndex.Logs[i].Timestamp = 1000
				}
			},
			maxTimelines:  10,
			maxSampleLogs: 3,
			want: wantSamplingResult{
				MatchedTLCount: 3,
				GroupCount:     3,
				SampleLogIDs:   []uint32{1, 3, 5},
			},
		},
		{
			name: "time-based sampling returns strictly increasing indices with duplicate timestamps",
			modify: func(wb *Workbench) {
				// Duplicate timestamps across intermediate logs.
				wb.searchIndex.Logs[0].Timestamp = 1000
				wb.searchIndex.Logs[1].Timestamp = 3000
				wb.searchIndex.Logs[2].Timestamp = 3000
				wb.searchIndex.Logs[3].Timestamp = 3000
				wb.searchIndex.Logs[4].Timestamp = 5000
			},
			maxTimelines:  10,
			maxSampleLogs: 4,
			want: wantSamplingResult{
				MatchedTLCount: 3,
				GroupCount:     3,
				SampleLogIDs:   []uint32{1, 2, 4, 5},
			},
		},
		{
			name:          "sampling single log picks the first log",
			maxTimelines:  10,
			maxSampleLogs: 1,
			want: wantSamplingResult{
				MatchedTLCount: 3,
				GroupCount:     3,
				SampleLogIDs:   []uint32{1},
			},
		},
		{
			name:          "defaults used when max values are non-positive",
			maxTimelines:  0,
			maxSampleLogs: -1,
			want: wantSamplingResult{
				MatchedTLCount: 3,
				GroupCount:     3,
				SampleLogIDs:   []uint32{1, 2, 3, 4, 5},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupSearchLogsTestWorkbench()
			if tc.modify != nil {
				tc.modify(wb)
			}
			got, err := wb.SearchLogs(context.Background(), Filter{}, tc.maxTimelines, tc.maxSampleLogs)
			if err != nil {
				t.Fatalf("SearchLogs() unexpected error = %v", err)
			}

			var gotLogIDs []uint32
			for _, s := range got.SampleLogs {
				gotLogIDs = append(gotLogIDs, s.LogID)
			}
			gotSummary := wantSamplingResult{
				MatchedTLCount: got.MatchedTimelineCount,
				GroupCount:     len(got.TimelineGroups),
				SampleLogIDs:   gotLogIDs,
			}
			if diff := cmp.Diff(tc.want, gotSummary); diff != "" {
				t.Errorf("SearchLogs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSearchLogs_Errors(t *testing.T) {
	testCases := []struct {
		name          string
		filter        Filter
		modify        func(*Workbench)
		wantErrTarget error
	}{
		{
			name: "invalid CEL in log query returns validation error",
			filter: Filter{
				LogQuery: "invalid &&",
			},
		},
		{
			name: "invalid CEL in timeline query returns validation error",
			filter: Filter{
				TimelineQuery: "invalid &&",
			},
		},
		{
			name:          "closed workbench returns ErrWorkbenchClosed",
			modify:        func(wb *Workbench) { wb.Close() },
			wantErrTarget: ErrWorkbenchClosed,
		},
		{
			name:   "unready search index returns error",
			modify: func(wb *Workbench) { wb.searchIndex = nil },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupSearchLogsTestWorkbench()
			if tc.modify != nil {
				tc.modify(wb)
			}
			_, err := wb.SearchLogs(context.Background(), tc.filter, 10, 10)
			if err == nil {
				t.Fatalf("SearchLogs() unexpected nil error")
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("SearchLogs() err = %v, want %v", err, tc.wantErrTarget)
			}
		})
	}
}
