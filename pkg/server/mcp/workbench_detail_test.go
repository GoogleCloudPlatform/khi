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

package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestParseIDArgument(t *testing.T) {
	testCases := []struct {
		name   string
		value  string
		want   uint32
		wantOK bool
	}{
		{name: "decimal ID", value: "10", want: 10, wantOK: true},
		{name: "maximum uint32 ID", value: "4294967295", want: 4294967295, wantOK: true},
		{name: "zero is not a valid ID", value: "0", wantOK: false},
		{name: "negative number", value: "-1", wantOK: false},
		{name: "overflows uint32", value: "4294967296", wantOK: false},
		{name: "non-numeric string", value: "pod-1", wantOK: false},
		{name: "empty string", value: "", wantOK: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotOK := parseIDArgument(tc.value)
			if got != tc.want || gotOK != tc.wantOK {
				t.Errorf("parseIDArgument(%q) = (%d, %v), want (%d, %v)", tc.value, got, gotOK, tc.want, tc.wantOK)
			}
		})
	}
}

func TestWorkbenchHandler_DetailToolErrors(t *testing.T) {
	server, inspectionID := setupTestInspectionServer(t)
	indexMgr := workbench.NewInspectionIndexManager(server, t.TempDir())
	mgr := workbench.NewWorkbenchManager(server, indexMgr, 5)
	handler := NewWorkbenchHandler(mgr)

	t.Cleanup(func() {
		indexMgr.Wait()
	})

	revisionIndex := 0
	at := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	unknownTimelineID := "4294967295"

	testCases := []struct {
		name string
		call func() (*mcpsdk.CallToolResult, any, error)
		want []string
	}{
		{
			name: "get_timeline_logs without inspectionId",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetTimelineLogs(context.Background(), nil, GetTimelineLogsInput{TimelineID: "1"})
			},
			want: []string{"Error: INVALID_ARGUMENT", "inspectionId is required."},
		},
		{
			name: "get_timeline_logs with a non-numeric timelineId",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetTimelineLogs(context.Background(), nil, GetTimelineLogsInput{InspectionID: inspectionID, TimelineID: "pod-1"})
			},
			want: []string{"Error: INVALID_ARGUMENT", "timelineId must be a positive integer ID, got `pod-1`."},
		},
		{
			name: "get_timeline_logs with an invalid log CEL",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetTimelineLogs(context.Background(), nil, GetTimelineLogsInput{
					InspectionID: inspectionID,
					TimelineID:   "1",
					Filter:       workbench.TimelineLogFilter{LogQuery: "invalid &&& syntax"},
				})
			},
			want: []string{"Error: INVALID_CEL", "Field: `filter.logQuery`"},
		},
		{
			name: "get_timeline_logs with an unknown timeline",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetTimelineLogs(context.Background(), nil, GetTimelineLogsInput{InspectionID: inspectionID, TimelineID: unknownTimelineID})
			},
			want: []string{"Error: TIMELINE_NOT_FOUND", "Use a timeline ID from `search_timelines` or `search_logs`."},
		},
		{
			name: "get_log with an unknown log",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetLog(context.Background(), nil, GetLogInput{InspectionID: inspectionID, LogID: "4294967295"})
			},
			want: []string{"Error: LOG_NOT_FOUND", "Use a log ID from `search_logs` or `get_timeline_logs`."},
		},
		{
			name: "get_log with an unknown inspection",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetLog(context.Background(), nil, GetLogInput{InspectionID: "non-existent-inspection", LogID: "1"})
			},
			want: []string{"Error: INSPECTION_NOT_FOUND"},
		},
		{
			name: "get_resource_revisions with a malformed pageToken",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceRevisions(context.Background(), nil, GetResourceRevisionsInput{InspectionID: inspectionID, TimelineID: "1", PageToken: "!!!"})
			},
			want: []string{"Error: INVALID_ARGUMENT", "pageToken is invalid"},
		},
		{
			name: "get_resource_revisions with an unknown timeline",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceRevisions(context.Background(), nil, GetResourceRevisionsInput{InspectionID: inspectionID, TimelineID: unknownTimelineID})
			},
			want: []string{"Error: TIMELINE_NOT_FOUND"},
		},
		{
			name: "get_resource_manifest with both revisionIndex and time",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceManifest(context.Background(), nil, GetResourceManifestInput{InspectionID: inspectionID, TimelineID: "1", RevisionIndex: &revisionIndex, Time: &at})
			},
			want: []string{"Error: INVALID_ARGUMENT", "Specify exactly one of revisionIndex or time."},
		},
		{
			name: "get_resource_manifest with neither revisionIndex nor time",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceManifest(context.Background(), nil, GetResourceManifestInput{InspectionID: inspectionID, TimelineID: "1"})
			},
			want: []string{"Error: INVALID_ARGUMENT", "Specify exactly one of revisionIndex or time."},
		},
		{
			name: "get_resource_manifest with an unknown timeline",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceManifest(context.Background(), nil, GetResourceManifestInput{InspectionID: inspectionID, TimelineID: unknownTimelineID, RevisionIndex: &revisionIndex})
			},
			want: []string{"Error: TIMELINE_NOT_FOUND"},
		},
		{
			name: "get_resource_diff with a zero timelineId",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceDiff(context.Background(), nil, GetResourceDiffInput{InspectionID: inspectionID, TimelineID: "0"})
			},
			want: []string{"Error: INVALID_ARGUMENT", "timelineId must be a positive integer ID, got `0`."},
		},
		{
			name: "get_resource_diff with an unknown timeline",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceDiff(context.Background(), nil, GetResourceDiffInput{InspectionID: inspectionID, TimelineID: unknownTimelineID})
			},
			want: []string{"Error: TIMELINE_NOT_FOUND"},
		},
		{
			name: "get_log with a negative byteOffset",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetLog(context.Background(), nil, GetLogInput{InspectionID: inspectionID, LogID: "1", ByteOffset: -1})
			},
			want: []string{"Error: INVALID_ARGUMENT", "byteOffset must not be negative, got `-1`."},
		},
		{
			name: "get_resource_manifest with a negative byteOffset",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceManifest(context.Background(), nil, GetResourceManifestInput{InspectionID: inspectionID, TimelineID: "1", RevisionIndex: &revisionIndex, ByteOffset: -1})
			},
			want: []string{"Error: INVALID_ARGUMENT", "byteOffset must not be negative, got `-1`."},
		},
		{
			name: "get_resource_diff with a negative byteOffset",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceDiff(context.Background(), nil, GetResourceDiffInput{InspectionID: inspectionID, TimelineID: "1", ByteOffset: -1})
			},
			want: []string{"Error: INVALID_ARGUMENT", "byteOffset must not be negative, got `-1`."},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, _, err := tc.call()
			if err != nil {
				t.Fatalf("handler unexpected error = %v", err)
			}
			if !res.IsError {
				t.Errorf("res.IsError = false, want true")
			}
			gotText := extractToolResultText(t, res)
			for _, sub := range tc.want {
				if !strings.Contains(gotText, sub) {
					t.Errorf("result text does not contain %q; got:\n%s", sub, gotText)
				}
			}
		})
	}
}

func TestRevisionLookupErrorResult(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		wantOK   bool
		wantText string
	}{
		{
			name:     "timeline not found",
			err:      workbench.ErrTimelineNotFound,
			wantOK:   true,
			wantText: "Error: TIMELINE_NOT_FOUND\n\n- No timeline with ID `10` exists in inspection `insp`.\n- Use a timeline ID from `search_timelines` or `search_logs`.",
		},
		{
			name:     "revision not found",
			err:      workbench.ErrRevisionNotFound,
			wantOK:   true,
			wantText: "Error: REVISION_NOT_FOUND\n\n- Timeline `10` has no matching revision: revision not found.\n- Use a revision index from `get_resource_revisions`.",
		},
		{
			name:   "other errors are not lookup errors",
			err:    context.Canceled,
			wantOK: false,
		},
		{
			name:   "nil error",
			err:    nil,
			wantOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, ok := revisionLookupErrorResult(tc.err, "insp", "10")
			if ok != tc.wantOK {
				t.Fatalf("revisionLookupErrorResult(%v) ok = %v, want %v", tc.err, ok, tc.wantOK)
			}
			if !ok {
				return
			}
			gotText := extractToolResultText(t, res)
			if diff := cmp.Diff(tc.wantText, gotText); diff != "" {
				t.Errorf("revisionLookupErrorResult(%v) text mismatch (-want +got):\n%s", tc.err, diff)
			}
		})
	}
}

func TestByteOffsetBeyondBodyResult(t *testing.T) {
	testCases := []struct {
		name       string
		byteOffset int
		totalBytes int
		wantOK     bool
		wantText   string
	}{
		{
			name:       "offset within the body is accepted",
			byteOffset: 10,
			totalBytes: 20,
			wantOK:     false,
		},
		{
			name:       "offset at the end of the body is accepted",
			byteOffset: 20,
			totalBytes: 20,
			wantOK:     false,
		},
		{
			name:       "zero offset on an empty body is accepted",
			byteOffset: 0,
			totalBytes: 0,
			wantOK:     false,
		},
		{
			name:       "offset past the end of the body is rejected",
			byteOffset: 21,
			totalBytes: 20,
			wantOK:     true,
			wantText: mdtemplate.FormatError("INVALID_ARGUMENT",
				"byteOffset `21` exceeds the body size of 20 bytes.",
				"Pass the byteOffset value from the last line of the previous response as is."),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, ok := byteOffsetBeyondBodyResult(tc.byteOffset, tc.totalBytes)
			if ok != tc.wantOK {
				t.Fatalf("byteOffsetBeyondBodyResult(%d, %d) ok = %v, want %v", tc.byteOffset, tc.totalBytes, ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if !res.IsError {
				t.Errorf("res.IsError = false, want true")
			}
			gotText := extractToolResultText(t, res)
			if diff := cmp.Diff(tc.wantText, gotText); diff != "" {
				t.Errorf("byteOffsetBeyondBodyResult(%d, %d) text mismatch (-want +got):\n%s", tc.byteOffset, tc.totalBytes, diff)
			}
		})
	}
}
