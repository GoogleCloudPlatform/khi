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
	"strings"
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

func setupTimelineLogsTestWorkbench(t *testing.T) *Workbench {
	t.Helper()
	wb := setupSearchLogsTestWorkbench()

	pool := khifilev6model.NewTestInternPool(id.NewGenerator())
	for i := range wb.searchIndex.Logs {
		l := &wb.searchIndex.Logs[i]
		l.SummaryStringID = pool.InternString(fmt.Sprintf("summary-%d", l.ID)).ID()
	}
	body, err := structured.FromYAML("apiVersion: v1\nkind: Event\nmessage: Container nginx was OOMKilled\n")
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

	// Also link log 4 to timeline 3 so timeline 2 has two other timelines (tl 4 with 1 log, tl 3 with 1 log).
	wb.searchIndex.TimelineMap[3].Events = append(wb.searchIndex.TimelineMap[3].Events, cel.EventInfo{
		LogID:     4,
		Timestamp: 4000,
	})
	wb.searchIndex.LogTimelineIndex = NewLogTimelineCSRIndex(uint32(len(wb.searchIndex.Logs)), wb.searchIndex.Timelines)

	return wb
}

func TestGetTimelineLogs(t *testing.T) {
	testCases := []struct {
		name                    string
		timelineID              uint32
		filter                  TimelineLogFilter
		maxOtherTimelines       int
		wantTotalOtherTimelines int
		wantOtherTimelines      []OtherTimelineLogCount
		wantLogs                []TimelineLogEntry
		wantErrTarget           error
	}{
		{
			name:                    "returns chronological logs on timeline and aggregates other linked timelines",
			timelineID:              2,
			filter:                  TimelineLogFilter{},
			maxOtherTimelines:       10,
			wantTotalOtherTimelines: 2,
			wantOtherTimelines: []OtherTimelineLogCount{
				{
					TimelineID: 3,
					Segments: []TimelineSegment{
						{Type: "Namespace", Name: "default"},
						{Type: "Pod", Name: "pod-b"},
					},
					LogCount: 1,
				},
				{
					TimelineID: 4,
					Segments: []TimelineSegment{
						{Type: "Namespace", Name: "default"},
						{Type: "Pod", Name: "pod-a"},
						{Type: "Container", Name: "container-a1"},
					},
					LogCount: 1,
				},
			},
			wantLogs: []TimelineLogEntry{
				{
					LogID:            1,
					Time:             time.Unix(0, 1000).UTC(),
					Severity:         testSeverityInfo,
					LogType:          "k8s-event",
					Summary:          "summary-1",
					OtherTimelineIDs: nil,
				},
				{
					LogID:            2,
					Time:             time.Unix(0, 2000).UTC(),
					Severity:         testSeverityWarning,
					LogType:          "k8s-event",
					Summary:          "summary-2",
					OtherTimelineIDs: []uint32{4},
				},
				{
					LogID:            4,
					Time:             time.Unix(0, 4000).UTC(),
					Severity:         testSeverityError,
					LogType:          "k8s-event",
					Summary:          "summary-4",
					OtherTimelineIDs: []uint32{3},
				},
			},
		},
		{
			name:       "filters logs by logQuery and truncates other timelines when exceeding limit",
			timelineID: 2,
			filter: TimelineLogFilter{
				LogQuery: "severity >= WARNING",
			},
			maxOtherTimelines:       1,
			wantTotalOtherTimelines: 2,
			wantOtherTimelines: []OtherTimelineLogCount{
				{
					TimelineID: 3,
					Segments: []TimelineSegment{
						{Type: "Namespace", Name: "default"},
						{Type: "Pod", Name: "pod-b"},
					},
					LogCount: 1,
				},
			},
			wantLogs: []TimelineLogEntry{
				{
					LogID:            2,
					Time:             time.Unix(0, 2000).UTC(),
					Severity:         testSeverityWarning,
					LogType:          "k8s-event",
					Summary:          "summary-2",
					OtherTimelineIDs: []uint32{4},
				},
				{
					LogID:            4,
					Time:             time.Unix(0, 4000).UTC(),
					Severity:         testSeverityError,
					LogType:          "k8s-event",
					Summary:          "summary-4",
					OtherTimelineIDs: []uint32{3},
				},
			},
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
			got, err := wb.GetTimelineLogs(context.Background(), tc.timelineID, tc.filter, tc.maxOtherTimelines)
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupTimelineLogsTestWorkbench(t)
			res, err := wb.GetTimelineLogs(context.Background(), tc.timelineID, tc.filter, 20)
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
	testCases := []struct {
		name            string
		logID           uint32
		byteOffset      int
		byteLimit       int
		wantLinkedTLIDs []uint32
		wantTruncated   bool
		wantBodyContain string
		wantErrTarget   error
	}{
		{
			name:            "returns full log metadata linked timelines and YAML body",
			logID:           2,
			byteOffset:      0,
			byteLimit:       1024,
			wantLinkedTLIDs: []uint32{2, 4},
			wantTruncated:   false,
			wantBodyContain: "message: Container nginx was OOMKilled",
		},
		{
			name:            "truncates log body at line boundary when exceeding byteLimit",
			logID:           2,
			byteOffset:      0,
			byteLimit:       30,
			wantLinkedTLIDs: []uint32{2, 4},
			wantTruncated:   true,
			wantBodyContain: "apiVersion: v1\nkind: Event\n",
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
			got, err := wb.GetLogDetail(tc.logID, tc.byteOffset, tc.byteLimit)
			if tc.wantErrTarget != nil {
				if !errors.Is(err, tc.wantErrTarget) {
					t.Fatalf("GetLogDetail() err = %v, want %v", err, tc.wantErrTarget)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetLogDetail() unexpected error = %v", err)
			}
			if got.Body.Truncated != tc.wantTruncated {
				t.Errorf("Body.Truncated = %v, want %v", got.Body.Truncated, tc.wantTruncated)
			}
			if !strings.Contains(got.Body.Content, tc.wantBodyContain) {
				t.Errorf("Body.Content = %q, want substring %q", got.Body.Content, tc.wantBodyContain)
			}
			var gotTLIDs []uint32
			for _, lt := range got.LinkedTimelines {
				gotTLIDs = append(gotTLIDs, lt.TimelineID)
			}
			if diff := cmp.Diff(tc.wantLinkedTLIDs, gotTLIDs); diff != "" {
				t.Errorf("LinkedTimelines IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
