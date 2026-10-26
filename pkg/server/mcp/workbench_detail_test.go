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
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestParseIDArgument(t *testing.T) {
	testCases := []struct {
		name  string
		value string
		want  uint32
	}{
		{name: "decimal ID", value: "10", want: 10},
		{name: "maximum uint32 ID", value: "4294967295", want: 4294967295},
		{name: "zero is not a valid ID", value: "0", want: 0},
		{name: "overflows uint32", value: "4294967296", want: 0},
		{name: "non-numeric string", value: "pod-1", want: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotOK := parseIDArgument(tc.value)
			wantOK := tc.want != 0
			if got != tc.want || gotOK != wantOK {
				t.Errorf("parseIDArgument(%q) = (%d, %v), want (%d, %v)", tc.value, got, gotOK, tc.want, wantOK)
			}
		})
	}
}

func TestWorkbenchHandler_DetailToolErrors(t *testing.T) {
	mgr, inspectionID := newTestWorkbenchManager(t)
	handler := NewWorkbenchHandler(mgr, nil)

	revisionIndex := 0
	at := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	unknownTimelineID := "4294967295"

	_, invalidCELErr := workbench.ValidateFilter(workbench.Filter{
		LogQuery: "invalid &&& syntax",
	})
	wantInvalidCELResult, _, _ := mdtemplate.CELErrorResult("filter.logQuery", "invalid &&& syntax", invalidCELErr)

	_, invalidTokenErr := mdtemplate.DecodePageToken("!!!")
	wantInvalidPageToken := wantErrorResult("INVALID_ARGUMENT",
		fmt.Sprintf("pageToken is invalid: %s.", invalidTokenErr.Error()),
		"Pass the pageToken value from the previous response as is.",
	)
	wantTimelineNotFound := wantErrorResult("TIMELINE_NOT_FOUND",
		fmt.Sprintf("No timeline with ID `%s` exists in inspection `%s`.", unknownTimelineID, inspectionID),
		"Use a timeline ID from `search_timelines` or `search_logs`.",
	)
	wantNegativeByteOffset := wantErrorResult("INVALID_ARGUMENT",
		"byteOffset must not be negative, got `-1`.",
		"Pass the byteOffset value from the last line of the previous response as is.",
	)

	testCases := []struct {
		name string
		call func() (*mcpsdk.CallToolResult, any, error)
		want *mcpsdk.CallToolResult
	}{
		{
			name: "get_timeline_logs without inspectionId",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetTimelineLogs(context.Background(), nil, GetTimelineLogsInput{TimelineID: "1"})
			},
			want: wantErrorResult("INVALID_ARGUMENT", "inspectionId is required."),
		},
		{
			name: "get_timeline_logs with a non-numeric timelineId",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetTimelineLogs(context.Background(), nil, GetTimelineLogsInput{InspectionID: inspectionID, TimelineID: "pod-1"})
			},
			want: wantErrorResult("INVALID_ARGUMENT", "timelineId must be a positive integer ID, got `pod-1`."),
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
			want: wantInvalidCELResult,
		},
		{
			name: "get_timeline_logs with an unknown timeline",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetTimelineLogs(context.Background(), nil, GetTimelineLogsInput{InspectionID: inspectionID, TimelineID: unknownTimelineID})
			},
			want: wantTimelineNotFound,
		},
		{
			name: "get_log with an unknown log",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetLog(context.Background(), nil, GetLogInput{InspectionID: inspectionID, LogID: "4294967295"})
			},
			want: wantErrorResult("LOG_NOT_FOUND",
				fmt.Sprintf("No log with ID `4294967295` exists in inspection `%s`.", inspectionID),
				"Use a log ID from `search_logs` or `get_timeline_logs`.",
			),
		},
		{
			name: "get_resource_revisions with a malformed pageToken",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceRevisions(context.Background(), nil, GetResourceRevisionsInput{InspectionID: inspectionID, TimelineID: "1", PageToken: "!!!"})
			},
			want: wantInvalidPageToken,
		},
		{
			name: "get_resource_revisions with an unknown timeline",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceRevisions(context.Background(), nil, GetResourceRevisionsInput{InspectionID: inspectionID, TimelineID: unknownTimelineID})
			},
			want: wantTimelineNotFound,
		},
		{
			name: "get_resource_manifest with both revisionIndex and time",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceManifest(context.Background(), nil, GetResourceManifestInput{InspectionID: inspectionID, TimelineID: "1", RevisionIndex: &revisionIndex, Time: &at})
			},
			want: wantErrorResult("INVALID_ARGUMENT", "Specify exactly one of revisionIndex or time."),
		},
		{
			name: "get_resource_manifest with neither revisionIndex nor time",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceManifest(context.Background(), nil, GetResourceManifestInput{InspectionID: inspectionID, TimelineID: "1"})
			},
			want: wantErrorResult("INVALID_ARGUMENT", "Specify exactly one of revisionIndex or time."),
		},
		{
			name: "get_resource_manifest with an unknown timeline",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceManifest(context.Background(), nil, GetResourceManifestInput{InspectionID: inspectionID, TimelineID: unknownTimelineID, RevisionIndex: &revisionIndex})
			},
			want: wantTimelineNotFound,
		},
		{
			name: "get_resource_diff with a zero timelineId",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceDiff(context.Background(), nil, GetResourceDiffInput{InspectionID: inspectionID, TimelineID: "0"})
			},
			want: wantErrorResult("INVALID_ARGUMENT", "timelineId must be a positive integer ID, got `0`."),
		},
		{
			name: "get_resource_diff with an unknown timeline",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceDiff(context.Background(), nil, GetResourceDiffInput{InspectionID: inspectionID, TimelineID: unknownTimelineID})
			},
			want: wantTimelineNotFound,
		},
		{
			name: "get_log with a negative byteOffset",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetLog(context.Background(), nil, GetLogInput{InspectionID: inspectionID, LogID: "1", ByteOffset: -1})
			},
			want: wantNegativeByteOffset,
		},
		{
			name: "get_resource_manifest with a negative byteOffset",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceManifest(context.Background(), nil, GetResourceManifestInput{InspectionID: inspectionID, TimelineID: "1", RevisionIndex: &revisionIndex, ByteOffset: -1})
			},
			want: wantNegativeByteOffset,
		},
		{
			name: "get_resource_diff with a negative byteOffset",
			call: func() (*mcpsdk.CallToolResult, any, error) {
				return handler.handleGetResourceDiff(context.Background(), nil, GetResourceDiffInput{InspectionID: inspectionID, TimelineID: "1", ByteOffset: -1})
			},
			want: wantNegativeByteOffset,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, _, err := tc.call()
			if err != nil {
				t.Fatalf("handler unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, res, ignoreMCPUnexported); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRevisionLookupErrorResult(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want *mcpsdk.CallToolResult
	}{
		{
			name: "timeline not found",
			err:  workbench.ErrTimelineNotFound,
			want: wantErrorResult("TIMELINE_NOT_FOUND",
				"No timeline with ID `10` exists in inspection `insp`.",
				"Use a timeline ID from `search_timelines` or `search_logs`.",
			),
		},
		{
			name: "revision not found",
			err:  workbench.ErrRevisionNotFound,
			want: wantErrorResult("REVISION_NOT_FOUND",
				"Timeline `10` has no matching revision: revision not found.",
				"Use a revision index from `get_resource_revisions`.",
			),
		},
		{
			name: "other errors are not lookup errors",
			err:  context.Canceled,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, ok := revisionLookupErrorResult(tc.err, "insp", "10")
			wantOK := tc.want != nil
			if ok != wantOK {
				t.Fatalf("revisionLookupErrorResult(%v) ok = %v, want %v", tc.err, ok, wantOK)
			}
			if diff := cmp.Diff(tc.want, res, ignoreMCPUnexported); diff != "" {
				t.Errorf("revisionLookupErrorResult(%v) result mismatch (-want +got):\n%s", tc.err, diff)
			}
		})
	}
}

func TestByteOffsetBeyondBodyResult(t *testing.T) {
	testCases := []struct {
		name       string
		byteOffset int
		totalBytes int
		want       *mcpsdk.CallToolResult
	}{
		{
			name:       "offset within the body is accepted",
			byteOffset: 10,
			totalBytes: 20,
		},
		{
			name:       "offset at the end of the body is accepted",
			byteOffset: 20,
			totalBytes: 20,
		},
		{
			name:       "offset past the end of the body is rejected",
			byteOffset: 21,
			totalBytes: 20,
			want: wantErrorResult("INVALID_ARGUMENT",
				"byteOffset `21` exceeds the content size of 20 bytes.",
				"Pass the byteOffset value from the last line of the previous response as is.",
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, ok := byteOffsetBeyondBodyResult(tc.byteOffset, tc.totalBytes)
			wantOK := tc.want != nil
			if ok != wantOK {
				t.Fatalf("byteOffsetBeyondBodyResult(%d, %d) ok = %v, want %v", tc.byteOffset, tc.totalBytes, ok, wantOK)
			}
			if diff := cmp.Diff(tc.want, res, ignoreMCPUnexported); diff != "" {
				t.Errorf("byteOffsetBeyondBodyResult(%d, %d) result mismatch (-want +got):\n%s", tc.byteOffset, tc.totalBytes, diff)
			}
		})
	}
}
