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
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

const testLogBodyYAML = "apiVersion: v1\nkind: Event\nmessage: Container nginx was OOMKilled\n"

func setupTimelineLogsTestWorkbench(t *testing.T) *Workbench {
	t.Helper()
	wb := setupSearchLogsTestWorkbench()

	pool := khifilev6model.NewTestInternPool(id.NewGenerator())
	for i := range wb.searchIndex.Logs {
		l := &wb.searchIndex.Logs[i]
		l.SummaryStringID = pool.InternString(fmt.Sprintf("summary-%d", l.ID)).ID()
	}
	body, err := structured.FromYAML(testLogBodyYAML)
	if err != nil {
		t.Fatalf("failed to parse body yaml: %v", err)
	}
	bodyRef, err := khifilev6model.ToInternedStruct(body, pool)
	if err != nil {
		t.Fatalf("failed to intern body struct: %v", err)
	}
	// Log ID 2 is stored at index 1.
	wb.searchIndex.Logs[1].BodyStructID = bodyRef.ID()

	readonlyPool := khifilev6model.NewReadonlyInternPool()
	var strs []*khifilev6.InternString
	for sRef := range pool.SortedStringRefs() {
		strs = append(strs, sRef.ToProto())
	}
	var fieldSets []*khifilev6.InternFieldPathSet
	for fsRef := range pool.FieldSetRefs() {
		fieldSets = append(fieldSets, fsRef.ToProto())
	}
	var structs []*pb.InternedStruct
	for sRef := range pool.StructRefs() {
		structs = append(structs, sRef.ToProto())
	}
	readonlyPool.IngestChunk(&khifilev6.InterningPoolChunk{
		Strings:       strs,
		FieldPathSets: fieldSets,
		Structs:       structs,
	})
	wb.internPool = readonlyPool
	wb.searchIndex.InternPool = readonlyPool

	// Link log 1 to timeline 4 and log 4 to timeline 3 so the logs on timeline 2 are shared with
	// timeline 4 (logs 1 and 2) more often than with timeline 3 (log 4).
	tl4 := wb.searchIndex.TimelineMap[4]
	tl4.Events = append([]cel.EventInfo{{LogID: 1, Timestamp: 1000}}, tl4.Events...)
	wb.searchIndex.TimelineMap[3].Events = append(wb.searchIndex.TimelineMap[3].Events, cel.EventInfo{
		LogID:     4,
		Timestamp: 4000,
	})
	wb.searchIndex.LogTimelineIndex = NewLogTimelineCSRIndex(uint32(len(wb.searchIndex.Logs)), wb.searchIndex.Timelines)

	return wb
}

// addTimelinesLinkedToLog1 adds count Pod timelines under timeline 1, each linked to log 1, and rebuilds the log-timeline index.
func addTimelinesLinkedToLog1(wb *Workbench, count int) {
	tl1 := wb.searchIndex.TimelineMap[1]
	for i := range count {
		tl := &cel.TimelineData{
			ID:           uint32(100 + i),
			ParentID:     1,
			Name:         fmt.Sprintf("extra-pod-%d", i),
			TimelineType: "Pod",
			Events:       []cel.EventInfo{{LogID: 1, Timestamp: 1000}},
		}
		tl1.ChildrenIDs = append(tl1.ChildrenIDs, tl.ID)
		wb.searchIndex.Timelines = append(wb.searchIndex.Timelines, tl)
		wb.searchIndex.TimelineMap[tl.ID] = tl
	}
	wb.searchIndex.LogTimelineIndex = NewLogTimelineCSRIndex(uint32(len(wb.searchIndex.Logs)), wb.searchIndex.Timelines)
}

var (
	testTimeline3Segments = []TimelineSegment{
		{Type: "Namespace", Name: "default"},
		{Type: "Pod", Name: "pod-b"},
	}
	testTimeline4Segments = []TimelineSegment{
		{Type: "Namespace", Name: "default"},
		{Type: "Pod", Name: "pod-a"},
		{Type: "Container", Name: "container-a1"},
	}
	testTimelineLog1 = TimelineLogEntry{
		LogID:            1,
		Time:             time.Unix(0, 1000).UTC(),
		Severity:         testSeverityInfo,
		LogType:          "k8s-event",
		Summary:          "summary-1",
		OtherTimelineIDs: []uint32{4},
	}
	testTimelineLog2 = TimelineLogEntry{
		LogID:            2,
		Time:             time.Unix(0, 2000).UTC(),
		Severity:         testSeverityWarning,
		LogType:          "k8s-event",
		Summary:          "summary-2",
		OtherTimelineIDs: []uint32{4},
	}
	testTimelineLog4 = TimelineLogEntry{
		LogID:            4,
		Time:             time.Unix(0, 4000).UTC(),
		Severity:         testSeverityError,
		LogType:          "k8s-event",
		Summary:          "summary-4",
		OtherTimelineIDs: []uint32{3},
	}
)

func timeAtNs(ns int64) *time.Time {
	t := time.Unix(0, ns).UTC()
	return &t
}

func otherTimeline(timelineID uint32, segments []TimelineSegment, matchedLogCount int) OtherTimeline {
	return OtherTimeline{
		LinkedTimeline:  LinkedTimeline{TimelineID: timelineID, Segments: segments},
		MatchedLogCount: matchedLogCount,
	}
}

func TestGetTimelineLogs(t *testing.T) {
	testCases := []struct {
		name                    string
		timelineID              uint32
		filter                  TimelineLogFilter
		wantTotalOtherTimelines int
		wantOtherTimelines      []OtherTimeline
		wantLogs                []TimelineLogEntry
		wantErrTarget           error
	}{
		{
			name:                    "returns chronological logs and orders other timelines by matched log count",
			timelineID:              2,
			filter:                  TimelineLogFilter{},
			wantTotalOtherTimelines: 2,
			wantOtherTimelines: []OtherTimeline{
				otherTimeline(4, testTimeline4Segments, 2),
				otherTimeline(3, testTimeline3Segments, 1),
			},
			wantLogs: []TimelineLogEntry{testTimelineLog1, testTimelineLog2, testTimelineLog4},
		},
		{
			name:       "filters logs by logQuery and orders tied other timelines by ID",
			timelineID: 2,
			filter: TimelineLogFilter{
				LogQuery: "severity >= WARNING",
			},
			wantTotalOtherTimelines: 2,
			wantOtherTimelines: []OtherTimeline{
				otherTimeline(3, testTimeline3Segments, 1),
				otherTimeline(4, testTimeline4Segments, 1),
			},
			wantLogs: []TimelineLogEntry{testTimelineLog2, testTimelineLog4},
		},
		{
			name:       "filters logs by inclusive start and end time",
			timelineID: 2,
			filter: TimelineLogFilter{
				StartTime: timeAtNs(2000),
				EndTime:   timeAtNs(4000),
			},
			wantTotalOtherTimelines: 2,
			wantOtherTimelines: []OtherTimeline{
				otherTimeline(3, testTimeline3Segments, 1),
				otherTimeline(4, testTimeline4Segments, 1),
			},
			wantLogs: []TimelineLogEntry{testTimelineLog2, testTimelineLog4},
		},
		{
			name:       "filters logs by start time only",
			timelineID: 2,
			filter: TimelineLogFilter{
				StartTime: timeAtNs(2001),
			},
			wantTotalOtherTimelines: 1,
			wantOtherTimelines: []OtherTimeline{
				otherTimeline(3, testTimeline3Segments, 1),
			},
			wantLogs: []TimelineLogEntry{testTimelineLog4},
		},
		{
			name:          "non-existent timeline returns ErrTimelineNotFound",
			timelineID:    999,
			wantErrTarget: ErrTimelineNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupTimelineLogsTestWorkbench(t)
			got, err := wb.GetTimelineLogs(context.Background(), tc.timelineID, tc.filter)
			if tc.wantErrTarget != nil {
				if !errors.Is(err, tc.wantErrTarget) {
					t.Fatalf("GetTimelineLogs() err = %v, want %v", err, tc.wantErrTarget)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetTimelineLogs() unexpected error = %v", err)
			}
			if got.TotalOtherTimelineCount != tc.wantTotalOtherTimelines {
				t.Errorf("TotalOtherTimelineCount = %d, want %d", got.TotalOtherTimelineCount, tc.wantTotalOtherTimelines)
			}
			if diff := cmp.Diff(tc.wantOtherTimelines, got.OtherTimelines); diff != "" {
				t.Errorf("OtherTimelines mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantLogs, got.Logs, protocmp.Transform()); diff != "" {
				t.Errorf("Logs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetTimelineLogs_OtherTimelinesLimit(t *testing.T) {
	// Timeline 4 shares 2 matched logs; timeline 3 and every extra timeline share 1, so ties are ordered by ID.
	wantKeptIDs := []uint32{4, 3}
	for id := uint32(100); id <= 117; id++ {
		wantKeptIDs = append(wantKeptIDs, id)
	}

	testCases := []struct {
		name                    string
		extraTimelineCount      int
		wantTotalOtherTimelines int
		wantOtherTimelineIDs    []uint32
	}{
		{
			name:                    "keeps all other timelines at the limit",
			extraTimelineCount:      18,
			wantTotalOtherTimelines: 20,
			wantOtherTimelineIDs:    wantKeptIDs,
		},
		{
			name:                    "keeps timelines with the most matched logs and the smallest IDs when exceeding the limit",
			extraTimelineCount:      21,
			wantTotalOtherTimelines: 23,
			wantOtherTimelineIDs:    wantKeptIDs,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupTimelineLogsTestWorkbench(t)
			addTimelinesLinkedToLog1(wb, tc.extraTimelineCount)

			got, err := wb.GetTimelineLogs(context.Background(), 2, TimelineLogFilter{})
			if err != nil {
				t.Fatalf("GetTimelineLogs() unexpected error = %v", err)
			}
			if got.TotalOtherTimelineCount != tc.wantTotalOtherTimelines {
				t.Errorf("TotalOtherTimelineCount = %d, want %d", got.TotalOtherTimelineCount, tc.wantTotalOtherTimelines)
			}
			var gotIDs []uint32
			for _, other := range got.OtherTimelines {
				gotIDs = append(gotIDs, other.TimelineID)
			}
			if diff := cmp.Diff(tc.wantOtherTimelineIDs, gotIDs); diff != "" {
				t.Errorf("OtherTimelines IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetTimelineLogs_WebUIAppliedFilterParity(t *testing.T) {
	testCases := []struct {
		name       string
		timelineID uint32
		filter     TimelineLogFilter
	}{
		{
			name:       "unfiltered timeline logs match Web UI pipeline with AppliedFilter",
			timelineID: 2,
			filter:     TimelineLogFilter{},
		},
		{
			name:       "filtered timeline logs match Web UI pipeline with AppliedFilter",
			timelineID: 2,
			filter: TimelineLogFilter{
				LogQuery: "severity >= WARNING",
			},
		},
		{
			name:       "time range filtered timeline logs match Web UI pipeline with AppliedFilter",
			timelineID: 2,
			filter: TimelineLogFilter{
				StartTime: timeAtNs(2000),
				EndTime:   timeAtNs(4000),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupTimelineLogsTestWorkbench(t)
			res, err := wb.GetTimelineLogs(context.Background(), tc.timelineID, tc.filter)
			if err != nil {
				t.Fatalf("GetTimelineLogs() unexpected error = %v", err)
			}

			// Execute the Web UI filter pipeline using the exact AppliedFilter returned by GetTimelineLogs.
			pipelineRes, err := wb.FilterTimeline(context.Background(), res.Applied.ToPipelineParams(), nil)
			if err != nil {
				t.Fatalf("FilterTimeline() unexpected error = %v", err)
			}

			allTLIDs := []uint32{1, 2, 3, 4}
			uiTimelineIDs := decodeSparseBitset(pipelineRes.GetTimelineMode(), pipelineRes.GetTimelineBitset(), allTLIDs)
			foundTimeline := false
			for _, id := range uiTimelineIDs {
				if id == tc.timelineID {
					foundTimeline = true
					break
				}
			}
			if !foundTimeline {
				t.Errorf("Web UI pipeline did not include timeline %d; got %v", tc.timelineID, uiTimelineIDs)
			}

			allLogIDs := []uint32{1, 2, 3, 4, 5}
			uiLogIDs := decodeSparseBitset(pipelineRes.GetLogMode(), pipelineRes.GetLogBitset(), allLogIDs)
			uiLogSet := make(map[uint32]bool, len(uiLogIDs))
			for _, id := range uiLogIDs {
				uiLogSet[id] = true
			}

			var wantOnTimeline []uint32
			wb.searchIndex.TimelineMap[tc.timelineID].ForEachLogID(func(logID uint32) bool {
				if uiLogSet[logID] {
					wantOnTimeline = append(wantOnTimeline, logID)
				}
				return true
			})

			var gotOnTimeline []uint32
			for _, l := range res.Logs {
				gotOnTimeline = append(gotOnTimeline, l.LogID)
			}
			if diff := cmp.Diff(wantOnTimeline, gotOnTimeline); diff != "" {
				t.Errorf("logs on timeline %d mismatch against Web UI filter (-want +got):\n%s", tc.timelineID, diff)
			}
		})
	}
}

func TestGetLogDetail(t *testing.T) {
	log2LinkedTimelines := []LinkedTimeline{
		{TimelineID: 2, Segments: []TimelineSegment{{Type: "Namespace", Name: "default"}, {Type: "Pod", Name: "pod-a"}}},
		{TimelineID: 4, Segments: testTimeline4Segments},
	}
	testCases := []struct {
		name          string
		logID         uint32
		byteOffset    int
		want          *LogDetail
		wantErrTarget error
	}{
		{
			name:       "returns full log metadata linked timelines and YAML body",
			logID:      2,
			byteOffset: 0,
			want: &LogDetail{
				LogID:           2,
				Time:            time.Unix(0, 2000).UTC(),
				Severity:        testSeverityWarning,
				LogType:         "k8s-event",
				Summary:         "summary-2",
				LinkedTimelines: log2LinkedTimelines,
				Body: BodyChunk{
					Content:    testLogBodyYAML,
					TotalBytes: len(testLogBodyYAML),
				},
			},
		},
		{
			name:       "returns the YAML body from byteOffset",
			logID:      2,
			byteOffset: len("apiVersion: v1\nkind: Event\n"),
			want: &LogDetail{
				LogID:           2,
				Time:            time.Unix(0, 2000).UTC(),
				Severity:        testSeverityWarning,
				LogType:         "k8s-event",
				Summary:         "summary-2",
				LinkedTimelines: log2LinkedTimelines,
				Body: BodyChunk{
					Content:    "message: Container nginx was OOMKilled\n",
					TotalBytes: len(testLogBodyYAML),
				},
			},
		},
		{
			name:          "non-existent log ID returns ErrLogNotFound",
			logID:         999,
			wantErrTarget: ErrLogNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupTimelineLogsTestWorkbench(t)
			got, err := wb.GetLogDetail(tc.logID, tc.byteOffset)
			if tc.wantErrTarget != nil {
				if !errors.Is(err, tc.wantErrTarget) {
					t.Fatalf("GetLogDetail() err = %v, want %v", err, tc.wantErrTarget)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetLogDetail() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("GetLogDetail() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
