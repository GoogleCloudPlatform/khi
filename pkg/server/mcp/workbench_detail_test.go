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

var (
	testPodSegments = []workbench.TimelineSegment{
		{Type: "Namespace", Name: "default"},
		{Type: "Pod", Name: "nginx"},
	}
	testCreateRevision = workbench.ResourceRevision{
		Index:     0,
		Time:      time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
		Verb:      "CREATE",
		State:     "Pending",
		Principal: "user@example.com",
		LogID:     11,
	}
	testUpdateRevision = workbench.ResourceRevision{
		Index:     1,
		Time:      time.Date(2026, 10, 1, 10, 5, 0, 0, time.UTC),
		Verb:      "UPDATE",
		State:     "Running",
		Principal: "system:node|x",
		LogID:     12,
	}
)

func mustPaginate[T any](t *testing.T, items []T, pageSize int, pageToken string) mdtemplate.PageResult[T] {
	t.Helper()
	page, err := mdtemplate.Paginate(items, pageSize, defaultDetailPageSize, maxDetailPageSize, pageToken)
	if err != nil {
		t.Fatalf("mdtemplate.Paginate() unexpected error = %v", err)
	}
	return page
}

func TestWorkbenchDetail_Templates(t *testing.T) {
	templates := mdtemplate.MustParse(templateFS, "templates/*.md.tmpl")
	timelineLogs := &workbench.TimelineLogsResult{
		TimelineID: 10,
		Segments:   testPodSegments,
		Applied: workbench.AppliedFilter{
			LogQuery:                    "severity >= WARNING",
			ExcludeTimelinesWithoutLogs: true,
		},
		TotalOtherTimelineCount: 2,
		OtherTimelines: []workbench.OtherTimelineLogCount{
			{TimelineID: 11, Segments: []workbench.TimelineSegment{{Type: "Namespace", Name: "default"}, {Type: "Pod", Name: "other"}}, LogCount: 2},
		},
		Logs: []workbench.TimelineLogEntry{
			{LogID: 101, Time: time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC), Severity: testSeverityWarning, LogType: "k8s_audit", Summary: "update a|b", OtherTimelineIDs: []uint32{11, 12}},
			{LogID: 102, Time: time.Date(2026, 10, 1, 10, 5, 0, 0, time.UTC), Severity: testSeverityError, LogType: "k8s_audit", Summary: "delete"},
			{LogID: 103, Time: time.Date(2026, 10, 1, 10, 10, 0, 0, time.UTC), Severity: testSeverityError, LogType: "k8s_audit", Summary: "delete again"},
		},
	}
	emptyTimelineLogs := &workbench.TimelineLogsResult{
		TimelineID: 10,
		Segments:   testPodSegments,
	}

	testCases := []struct {
		name         string
		templateName string
		data         any
		want         string
	}{
		{
			name:         "get_timeline_logs.md.tmpl with other timelines and a next page",
			templateName: "get_timeline_logs.md.tmpl",
			data:         buildTimelineLogsTemplateData(timelineLogs, mustPaginate(t, timelineLogs.Logs, 2, "")),
			want: strings.Join([]string{
				"# Logs of [Namespace] default > [Pod] nginx (`10`)",
				"",
				"Showing logs 1-2 of 3, oldest first.",
				"",
				"## Applied filter",
				"Enter these values in the KHI Web UI filter to see the same data.",
				"",
				"```yaml",
				`timelineQuery: ""`,
				`timelineExclusionQuery: ""`,
				"logQuery: severity >= WARNING",
				`startTime: ""`,
				`endTime: ""`,
				"excludeTimelinesWithoutLogs: true",
				"```",
				"",
				"## Other timelines linked to these logs",
				"Counted over all 3 logs, not only this page.",
				"Showing 1 of 2 timelines, sorted by the number of logs.",
				"",
				"| Timeline | ID | Logs |",
				"| --- | --- | --- |",
				"| [Namespace] default > [Pod] other | `11` | 2 |",
				"",
				"## Logs",
				"",
				"| Time | Severity | Type | Log ID | Summary | Also on |",
				"| --- | --- | --- | --- | --- | --- |",
				"| 2026-10-01T10:00:00Z | WARNING | k8s_audit | `101` | update a\\|b | `11`, `12` |",
				"| 2026-10-01T10:05:00Z | ERROR | k8s_audit | `102` | delete |  |",
				"",
				`pageToken: "` + mdtemplate.EncodePageToken(2) + `"`,
			}, "\n"),
		},
		{
			name:         "get_timeline_logs.md.tmpl on the last page",
			templateName: "get_timeline_logs.md.tmpl",
			data:         buildTimelineLogsTemplateData(timelineLogs, mustPaginate(t, timelineLogs.Logs, 2, mdtemplate.EncodePageToken(2))),
			want: strings.Join([]string{
				"# Logs of [Namespace] default > [Pod] nginx (`10`)",
				"",
				"Showing logs 3-3 of 3, oldest first.",
				"",
				"## Applied filter",
				"Enter these values in the KHI Web UI filter to see the same data.",
				"",
				"```yaml",
				`timelineQuery: ""`,
				`timelineExclusionQuery: ""`,
				"logQuery: severity >= WARNING",
				`startTime: ""`,
				`endTime: ""`,
				"excludeTimelinesWithoutLogs: true",
				"```",
				"",
				"## Other timelines linked to these logs",
				"Counted over all 3 logs, not only this page.",
				"Showing 1 of 2 timelines, sorted by the number of logs.",
				"",
				"| Timeline | ID | Logs |",
				"| --- | --- | --- |",
				"| [Namespace] default > [Pod] other | `11` | 2 |",
				"",
				"## Logs",
				"",
				"| Time | Severity | Type | Log ID | Summary | Also on |",
				"| --- | --- | --- | --- | --- | --- |",
				"| 2026-10-01T10:10:00Z | ERROR | k8s_audit | `103` | delete again |  |",
			}, "\n"),
		},
		{
			name:         "get_timeline_logs.md.tmpl without matched logs",
			templateName: "get_timeline_logs.md.tmpl",
			data:         buildTimelineLogsTemplateData(emptyTimelineLogs, mustPaginate(t, emptyTimelineLogs.Logs, 0, "")),
			want: strings.Join([]string{
				"# Logs of [Namespace] default > [Pod] nginx (`10`)",
				"",
				"No logs on this timeline matched the filter.",
				"",
				"## Applied filter",
				"Enter these values in the KHI Web UI filter to see the same data.",
				"",
				"```yaml",
				`timelineQuery: ""`,
				`timelineExclusionQuery: ""`,
				`logQuery: ""`,
				`startTime: ""`,
				`endTime: ""`,
				"excludeTimelinesWithoutLogs: false",
				"```",
			}, "\n"),
		},
		{
			name:         "get_log.md.tmpl with linked timelines and a truncated body",
			templateName: "get_log.md.tmpl",
			data: buildLogTemplateData(&workbench.LogDetail{
				LogID:    101,
				Time:     time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
				Severity: testSeverityError,
				LogType:  "k8s_audit",
				Summary:  "update a|b",
				LinkedTimelines: []workbench.LinkedTimeline{
					{TimelineID: 10, Segments: testPodSegments},
				},
				Body: workbench.TruncatedBody{Content: "kind: Pod\n", TotalBytes: 20, NextByteOffset: 10, Truncated: true},
			}),
			want: strings.Join([]string{
				"# Log `101`",
				"",
				"- Time: 2026-10-01T10:00:00Z",
				"- Severity: ERROR",
				"- Type: k8s_audit",
				"- Summary: update a\\|b",
				"",
				"## Linked timelines",
				"",
				"| Timeline | ID |",
				"| --- | --- |",
				"| [Namespace] default > [Pod] nginx | `10` |",
				"",
				"## Body (20 bytes)",
				"",
				"```yaml",
				"kind: Pod",
				"# [KHI] truncated: showing 10 of 20 bytes. Call again with byteOffset=10 to read the rest.",
				"```",
				"",
				"byteOffset: 10",
			}, "\n"),
		},
		{
			name:         "get_log.md.tmpl without linked timelines",
			templateName: "get_log.md.tmpl",
			data: buildLogTemplateData(&workbench.LogDetail{
				LogID:    102,
				Time:     time.Date(2026, 10, 1, 10, 5, 0, 0, time.UTC),
				Severity: testSeverityInfo,
				LogType:  "event",
				Summary:  "Scheduled",
				Body:     workbench.TruncatedBody{Content: "kind: Pod\n", TotalBytes: 10},
			}),
			want: strings.Join([]string{
				"# Log `102`",
				"",
				"- Time: 2026-10-01T10:05:00Z",
				"- Severity: INFO",
				"- Type: event",
				"- Summary: Scheduled",
				"",
				"## Linked timelines",
				"",
				"This log is not linked to any timeline.",
				"",
				"## Body (10 bytes)",
				"",
				"```yaml",
				"kind: Pod",
				"```",
			}, "\n"),
		},
		{
			name:         "get_resource_revisions.md.tmpl with a next page",
			templateName: "get_resource_revisions.md.tmpl",
			data: buildResourceRevisionsTemplateData(&workbench.ResourceRevisionsResult{
				TimelineID:   10,
				Segments:     testPodSegments,
				MatchedCount: 3,
				Revisions: []workbench.ResourceRevisionSummary{
					{ResourceRevision: testCreateRevision, Changes: workbench.LineChangeCount{Added: 3}},
					{ResourceRevision: testUpdateRevision, Changes: workbench.LineChangeCount{Added: 1, Deleted: 1}},
				},
			}, 0),
			want: strings.Join([]string{
				"# Revisions of [Namespace] default > [Pod] nginx (`10`)",
				"",
				"Showing revisions 0-1 of 3, oldest first.",
				"",
				"| # | Time | Verb | State | Principal | Log ID | Diff |",
				"| --- | --- | --- | --- | --- | --- | --- |",
				"| 0 | 2026-10-01T10:00:00Z | CREATE | Pending | user@example.com | `11` | +3 -0 |",
				"| 1 | 2026-10-01T10:05:00Z | UPDATE | Running | system:node\\|x | `12` | +1 -1 |",
				"",
				`pageToken: "` + mdtemplate.EncodePageToken(2) + `"`,
			}, "\n"),
		},
		{
			name:         "get_resource_revisions.md.tmpl without matched revisions",
			templateName: "get_resource_revisions.md.tmpl",
			data: buildResourceRevisionsTemplateData(&workbench.ResourceRevisionsResult{
				TimelineID: 10,
				Segments:   testPodSegments,
			}, 0),
			want: strings.Join([]string{
				"# Revisions of [Namespace] default > [Pod] nginx (`10`)",
				"",
				"No revisions of this timeline matched the time range.",
			}, "\n"),
		},
		{
			name:         "get_resource_revisions.md.tmpl with a page token past the last revision",
			templateName: "get_resource_revisions.md.tmpl",
			data: buildResourceRevisionsTemplateData(&workbench.ResourceRevisionsResult{
				TimelineID:   10,
				Segments:     testPodSegments,
				MatchedCount: 3,
			}, 5),
			want: strings.Join([]string{
				"# Revisions of [Namespace] default > [Pod] nginx (`10`)",
				"",
				"No revisions on this page. 3 revisions matched the time range.",
			}, "\n"),
		},
		{
			name:         "get_resource_manifest.md.tmpl",
			templateName: "get_resource_manifest.md.tmpl",
			data: resourceManifestTemplateData{
				TimelineID:  "10",
				Segments:    testPodSegments,
				Revision:    toRevisionTemplateData(testCreateRevision),
				BodySection: formatBodySection("Body", "yaml", workbench.TruncatedBody{Content: "kind: Pod\nstatus:\n  phase: Pending\n", TotalBytes: 35}),
			},
			want: strings.Join([]string{
				"# Manifest of [Namespace] default > [Pod] nginx (`10`)",
				"",
				"- Revision: 0",
				"- Time: 2026-10-01T10:00:00Z",
				"- Verb: CREATE",
				"- State: Pending",
				"- Principal: user@example.com",
				"- Log ID: `11`",
				"",
				"## Body (35 bytes)",
				"",
				"```yaml",
				"kind: Pod",
				"status:",
				"  phase: Pending",
				"```",
			}, "\n"),
		},
		{
			name:         "get_resource_diff.md.tmpl against the previous revision",
			templateName: "get_resource_diff.md.tmpl",
			data: buildResourceDiffTemplateData(&workbench.ResourceDiff{
				TimelineID: 10,
				Segments:   testPodSegments,
				Revision:   testUpdateRevision,
				Previous:   &testCreateRevision,
				Changes:    workbench.LineChangeCount{Added: 1, Deleted: 1},
				Diff: workbench.TruncatedBody{
					Content:    "--- revision 0\n+++ revision 1\n@@ -3 +3 @@\n-  phase: Pending\n+  phase: Running\n",
					TotalBytes: 78,
				},
			}),
			want: strings.Join([]string{
				"# Diff of [Namespace] default > [Pod] nginx (`10`)",
				"",
				"- Revision: 1, 2026-10-01T10:05:00Z, UPDATE by system:node|x, state Running, log `12`",
				"- Previous revision: 0, 2026-10-01T10:00:00Z, CREATE by user@example.com, state Pending, log `11`",
				"- Changes: +1 -1",
				"",
				"## Diff (78 bytes)",
				"",
				"```diff",
				"--- revision 0",
				"+++ revision 1",
				"@@ -3 +3 @@",
				"-  phase: Pending",
				"+  phase: Running",
				"```",
			}, "\n"),
		},
		{
			name:         "get_resource_diff.md.tmpl for the initial revision",
			templateName: "get_resource_diff.md.tmpl",
			data: buildResourceDiffTemplateData(&workbench.ResourceDiff{
				TimelineID: 10,
				Segments:   testPodSegments,
				Revision:   testCreateRevision,
				Changes:    workbench.LineChangeCount{Added: 1},
				Diff: workbench.TruncatedBody{
					Content:    "--- none\n+++ revision 0\n@@ -0,0 +1 @@\n+kind: Pod\n",
					TotalBytes: 49,
				},
			}),
			want: strings.Join([]string{
				"# Diff of [Namespace] default > [Pod] nginx (`10`)",
				"",
				"- Revision: 0, 2026-10-01T10:00:00Z, CREATE by user@example.com, state Pending, log `11`",
				"- Previous revision: none",
				"- Changes: +1 -0",
				"",
				"## Diff (49 bytes)",
				"",
				"```diff",
				"--- none",
				"+++ revision 0",
				"@@ -0,0 +1 @@",
				"+kind: Pod",
				"```",
			}, "\n"),
		},
		{
			name:         "get_resource_diff.md.tmpl for a revision identical to the previous one",
			templateName: "get_resource_diff.md.tmpl",
			data: buildResourceDiffTemplateData(&workbench.ResourceDiff{
				TimelineID: 10,
				Segments:   testPodSegments,
				Revision:   testUpdateRevision,
				Previous:   &testCreateRevision,
			}),
			want: strings.Join([]string{
				"# Diff of [Namespace] default > [Pod] nginx (`10`)",
				"",
				"- Revision: 1, 2026-10-01T10:05:00Z, UPDATE by system:node|x, state Running, log `12`",
				"- Previous revision: 0, 2026-10-01T10:00:00Z, CREATE by user@example.com, state Pending, log `11`",
				"- Changes: +0 -0",
				"",
				"## Diff (0 bytes)",
				"",
				"The manifest is identical to the previous revision.",
			}, "\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := templates.Render(tc.templateName, tc.data)
			if err != nil {
				t.Fatalf("templates.Render(%q) unexpected error = %v", tc.templateName, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("templates.Render(%q) mismatch (-want +got):\n%s", tc.templateName, diff)
			}
		})
	}
}

func TestBuildResourceRevisionsTemplateData_NextPageToken(t *testing.T) {
	testCases := []struct {
		name          string
		offset        int
		revisionCount int
		matchedCount  int
		want          string
	}{
		{
			name:          "first page with remaining revisions",
			offset:        0,
			revisionCount: 2,
			matchedCount:  5,
			want:          mdtemplate.EncodePageToken(2),
		},
		{
			name:          "middle page with remaining revisions",
			offset:        2,
			revisionCount: 2,
			matchedCount:  5,
			want:          mdtemplate.EncodePageToken(4),
		},
		{
			name:          "last page reaching the matched count",
			offset:        4,
			revisionCount: 1,
			matchedCount:  5,
			want:          "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := &workbench.ResourceRevisionsResult{MatchedCount: tc.matchedCount}
			for i := range tc.revisionCount {
				res.Revisions = append(res.Revisions, workbench.ResourceRevisionSummary{
					ResourceRevision: workbench.ResourceRevision{Index: tc.offset + i},
				})
			}
			got := buildResourceRevisionsTemplateData(res, tc.offset).NextPageToken
			if got != tc.want {
				t.Errorf("buildResourceRevisionsTemplateData(offset=%d).NextPageToken = %q, want %q", tc.offset, got, tc.want)
			}
		})
	}
}

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
