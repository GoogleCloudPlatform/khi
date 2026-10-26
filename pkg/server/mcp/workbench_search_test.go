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

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/proto"
)

// The following severity definitions emulate the style chunk of a loaded inspection in the tests of this file.
var (
	testSeverityInfo    = &khifilev6.Severity{Id: proto.Uint32(1), Label: proto.String("INFO"), Order: proto.Int32(1)}
	testSeverityWarning = &khifilev6.Severity{Id: proto.Uint32(2), Label: proto.String("WARNING"), Order: proto.Int32(2)}
	testSeverityError   = &khifilev6.Severity{Id: proto.Uint32(3), Label: proto.String("ERROR"), Order: proto.Int32(3)}
)

func TestWorkbenchHandler_HandleSearchTimelines(t *testing.T) {
	mgr, inspectionID := newTestWorkbenchManager(t)
	handler := NewWorkbenchHandler(mgr, nil)

	_, invalidCELErr := workbench.ValidateFilter(workbench.Filter{
		TimelineQuery: "invalid &&& syntax",
	})
	wantInvalidCELResult, _, _ := mdtemplate.CELErrorResult("filter.timelineQuery", "invalid &&& syntax", invalidCELErr)

	testCases := []struct {
		name  string
		input SearchTimelinesInput
		want  *mcpsdk.CallToolResult
	}{
		{
			name: "zero matching timelines still returns Filter vocabulary",
			input: SearchTimelinesInput{
				InspectionID: inspectionID,
				Filter: workbench.Filter{
					TimelineQuery: `name == "no-such-timeline"`,
				},
			},
			want: wantTextResult(strings.Join([]string{
				"# Timeline tree",
				"",
				"Matched 0 timelines. Showing 0 nodes (maxNodes 200).",
				"",
				mdtemplate.FormatAppliedFilter(workbench.AppliedFilter{
					TimelineQuery: `name == "no-such-timeline"`,
				}),
				"",
				"## Tree",
				"",
				"No timelines matched the filter.",
				"",
				"## Filter vocabulary",
				"These are the values that filter CEL can match in this inspection.",
				"- path keys (lowercased timelineType): none",
				"- timelineType: none",
				"- logType: none",
				"- severity: `UNKNOWN`, `INFO`, `WARNING`, `ERROR`, `FATAL` (reassigned by KHI per logType for investigation importance; does not match Cloud Logging severity — query `body(\"severity\", \"ERROR\")` for the raw Cloud Logging field)",
			}, "\n")),
		},
		{
			name: "missing inspectionId returns INVALID_ARGUMENT",
			input: SearchTimelinesInput{
				InspectionID: "",
			},
			want: wantErrorResult("INVALID_ARGUMENT", "inspectionId is required."),
		},
		{
			name: "invalid CEL expression in filter.timelineQuery returns INVALID_CEL",
			input: SearchTimelinesInput{
				InspectionID: inspectionID,
				Filter: workbench.Filter{
					TimelineQuery: "invalid &&& syntax",
				},
			},
			want: wantInvalidCELResult,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, _, err := handler.handleSearchTimelines(context.Background(), nil, tc.input)
			if err != nil {
				t.Fatalf("handleSearchTimelines() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, res, ignoreMCPUnexported); diff != "" {
				t.Errorf("handleSearchTimelines() result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWorkbenchHandler_HandleSearchLogs(t *testing.T) {
	mgr, inspectionID := newTestWorkbenchManager(t)
	handler := NewWorkbenchHandler(mgr, nil)

	_, invalidCELErr := workbench.ValidateFilter(workbench.Filter{
		LogQuery: "invalid &&& syntax",
	})
	wantInvalidCELResult, _, _ := mdtemplate.CELErrorResult("filter.logQuery", "invalid &&& syntax", invalidCELErr)

	testCases := []struct {
		name  string
		input SearchLogsInput
		want  *mcpsdk.CallToolResult
	}{
		{
			name: "zero matching logs still returns Filter vocabulary",
			input: SearchLogsInput{
				InspectionID: inspectionID,
				Filter: workbench.Filter{
					LogQuery: `logType == "no-such-log-type"`,
				},
			},
			want: wantTextResult(strings.Join([]string{
				"# Logs matching the filter",
				"",
				"No logs matched the filter.",
				"",
				mdtemplate.FormatAppliedFilter(workbench.AppliedFilter{
					LogQuery:                    `logType == "no-such-log-type"`,
					ExcludeTimelinesWithoutLogs: true,
				}),
				"",
				"## Filter vocabulary",
				"These are the values that filter CEL can match in this inspection.",
				"- path keys (lowercased timelineType): none",
				"- timelineType: none",
				"- logType: none",
				"- severity: `UNKNOWN`, `INFO`, `WARNING`, `ERROR`, `FATAL` (reassigned by KHI per logType for investigation importance; does not match Cloud Logging severity — query `body(\"severity\", \"ERROR\")` for the raw Cloud Logging field)",
			}, "\n")),
		},
		{
			name: "missing inspectionId returns INVALID_ARGUMENT",
			input: SearchLogsInput{
				InspectionID: "",
			},
			want: wantErrorResult("INVALID_ARGUMENT", "inspectionId is required."),
		},
		{
			name: "invalid CEL expression in filter.logQuery returns INVALID_CEL",
			input: SearchLogsInput{
				InspectionID: inspectionID,
				Filter: workbench.Filter{
					LogQuery: "invalid &&& syntax",
				},
			},
			want: wantInvalidCELResult,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, _, err := handler.handleSearchLogs(context.Background(), nil, tc.input)
			if err != nil {
				t.Fatalf("handleSearchLogs() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, res, ignoreMCPUnexported); diff != "" {
				t.Errorf("handleSearchLogs() result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatTimelineTreeLine(t *testing.T) {
	testCases := []struct {
		name    string
		node    workbench.TimelineTreeNode
		sameDay bool
		want    string
	}{
		{
			name: "same day with all fields",
			node: workbench.TimelineTreeNode{
				ID:            10,
				Depth:         1,
				Type:          "Pod",
				Name:          "test-pod",
				EventCount:    2,
				RevisionCount: 1,
				SeverityCounts: []workbench.SeverityCount{
					{Severity: testSeverityError, Count: 1},
					{Severity: testSeverityWarning, Count: 3},
					{Severity: testSeverityInfo, Count: 5},
				},
				OmittedChildren: 4,
				FirstMatchTime:  time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC),
				LastMatchTime:   time.Date(2026, 10, 1, 10, 45, 0, 0, time.UTC),
			},
			sameDay: true,
			want:    "  [Pod] test-pod id=10 children=4 ev=2 rev=1 error=1 warning=3 info=5 09:30:00..10:45:00",
		},
		{
			name: "multi day with RFC3339 format",
			node: workbench.TimelineTreeNode{
				ID:             1,
				Depth:          0,
				Type:           "Node",
				Name:           "node-1",
				FirstMatchTime: time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC),
				LastMatchTime:  time.Date(2026, 10, 2, 10, 45, 0, 0, time.UTC),
			},
			sameDay: false,
			want:    "[Node] node-1 id=1 2026-10-01T09:30:00Z..2026-10-02T10:45:00Z",
		},
		{
			name: "minimal node with zero counts and zero times",
			node: workbench.TimelineTreeNode{
				ID:    5,
				Depth: 0,
				Type:  "Cluster",
				Name:  "my-cluster",
			},
			sameDay: false,
			want:    "[Cluster] my-cluster id=5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatTimelineTreeLine(tc.node, tc.sameDay)
			if got != tc.want {
				t.Errorf("formatTimelineTreeLine() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatSeveritySummary(t *testing.T) {
	testCases := []struct {
		name   string
		counts []workbench.SeverityCount
		want   string
	}{
		{
			name:   "no counts return empty string",
			counts: nil,
			want:   "",
		},
		{
			name: "counts are joined with severity labels",
			counts: []workbench.SeverityCount{
				{Severity: testSeverityError, Count: 2},
				{Severity: testSeverityWarning, Count: 3},
				{Severity: testSeverityInfo, Count: 5},
			},
			want: "ERROR 2, WARNING 3, INFO 5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSeveritySummary(tc.counts)
			if got != tc.want {
				t.Errorf("formatSeveritySummary() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWorkbenchSearch_Templates(t *testing.T) {
	templates := mdtemplate.MustParse(templateFS, "templates/*.md.tmpl")

	testCases := []struct {
		name         string
		templateName string
		data         any
		want         string
	}{
		{
			name:         "search_timelines.md.tmpl with non-empty tree lines and types",
			templateName: "search_timelines.md.tmpl",
			data: searchTimelinesTemplateData{
				MatchedTimelineCount: 5,
				DirectMatchCount:     3,
				AncestorNodeCount:    2,
				ReturnedNodeCount:    2,
				MaxDepth:             2,
				MaxNodes:             100,
				Applied: workbench.AppliedFilter{
					TimelineQuery:               "has(resource.labels.pod)",
					ExcludeTimelinesWithoutLogs: true,
				},
				TimelineTypes: []workbench.TimelineTypeDescription{
					{
						Type:        "pod",
						Description: "Kubernetes Pod",
					},
				},
				TreeLines: []string{
					"[pod] test-pod id=1 ev=2 rev=1",
					"  [container] main id=2 ev=1 rev=0",
				},
				FilterVocabulary: workbench.FilterVocabulary{
					PathKeys:      []string{"container", "k8scluster", "pod"},
					TimelineTypes: []string{"container", "k8sCluster", "pod"},
					LogTypes:      []string{"k8s audit", "k8s event"},
				},
			},
			want: strings.Join([]string{
				"# Timeline tree",
				"",
				"Matched 5 timelines (3 with matched events/revisions + 2 ancestor nodes). Showing 2 nodes (depth <= 2, maxNodes 100).",
				"Narrow with `filter.timelineQuery` (using path keys from Filter vocabulary below) or `filter.logQuery` (`severity`, `logType`) to inspect omitted nodes.",
				"",
				mdtemplate.FormatAppliedFilter(workbench.AppliedFilter{
					TimelineQuery:               "has(resource.labels.pod)",
					ExcludeTimelinesWithoutLogs: true,
				}),
				"",
				"## Timeline types",
				"- pod: Kubernetes Pod",
				"",
				"## Tree",
				"",
				"```text",
				"[pod] test-pod id=1 ev=2 rev=1",
				"  [container] main id=2 ev=1 rev=0",
				"```",
				"",
				"## Filter vocabulary",
				"These are the values that filter CEL can match in this inspection.",
				"- path keys (lowercased timelineType): `container`, `k8scluster`, `pod`",
				"- timelineType: `container`, `k8sCluster`, `pod`",
				"- logType: `k8s audit`, `k8s event`",
				"- severity: `UNKNOWN`, `INFO`, `WARNING`, `ERROR`, `FATAL` (reassigned by KHI per logType for investigation importance; does not match Cloud Logging severity — query `body(\"severity\", \"ERROR\")` for the raw Cloud Logging field)",
			}, "\n"),
		},
		{
			name:         "search_logs.md.tmpl with non-empty groups and samples",
			templateName: "search_logs.md.tmpl",
			data: searchLogsTemplateData{
				MatchedLogCount:      10,
				MatchedTimelineCount: 2,
				FirstMatchTime:       time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
				LastMatchTime:        time.Date(2026, 10, 1, 11, 0, 0, 0, time.UTC),
				SeveritySummary:      "ERROR 2, WARNING 3, INFO 5",
				Applied: workbench.AppliedFilter{
					LogQuery: "severity >= ERROR",
				},
				TimelineGroups: []timelineLogGroupTemplateData{
					{
						TimelineID: "1",
						Segments: []workbench.TimelineSegment{
							{Type: "pod", Name: "my-pod"},
						},
						MatchedLogCount: 6,
						SeveritySummary: "ERROR 2, INFO 4",
						FirstMatchTime:  time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
						LastMatchTime:   time.Date(2026, 10, 1, 10, 30, 0, 0, time.UTC),
					},
				},
				SampleLogs: []sampleLogTemplateData{
					{
						LogID:         "101",
						Time:          time.Date(2026, 10, 1, 10, 15, 0, 0, time.UTC),
						Severity:      "ERROR",
						LogType:       "container",
						Summary:       "CrashLoopBackOff",
						TimelinesCell: "`1`",
					},
				},
				FilterVocabulary: workbench.FilterVocabulary{
					PathKeys:      []string{"container", "pod"},
					TimelineTypes: []string{"container", "pod"},
					LogTypes:      []string{"container"},
				},
			},
			want: strings.Join([]string{
				"# Logs matching the filter",
				"",
				"Matched 10 logs linked to 2 timelines, from 2026-10-01T10:00:00Z to 2026-10-01T11:00:00Z.",
				"Severity: ERROR 2, WARNING 3, INFO 5.",
				"",
				mdtemplate.FormatAppliedFilter(workbench.AppliedFilter{
					LogQuery: "severity >= ERROR",
				}),
				"",
				"## Timelines linked to the matched logs",
				"Showing 1 of 2 timelines, sorted by the number of matched logs.",
				"One log can link to several timelines, so the counts can add up to more than 10.",
				"",
				"| Timeline | ID | Logs | Severity | First | Last |",
				"| --- | --- | --- | --- | --- | --- |",
				"| [pod] my-pod | `1` | 6 | ERROR 2, INFO 4 | 2026-10-01T10:00:00Z | 2026-10-01T10:30:00Z |",
				"",
				"## Sample logs",
				"Showing 1 of 10 logs, spread evenly over time.",
				"",
				"| Time | Severity | Type | Log ID | Summary | Timelines |",
				"| --- | --- | --- | --- | --- | --- |",
				"| 2026-10-01T10:15:00Z | ERROR | container | `101` | CrashLoopBackOff | `1` |",
				"",
				"## Filter vocabulary",
				"These are the values that filter CEL can match in this inspection.",
				"- path keys (lowercased timelineType): `container`, `pod`",
				"- timelineType: `container`, `pod`",
				"- logType: `container`",
				"- severity: `UNKNOWN`, `INFO`, `WARNING`, `ERROR`, `FATAL` (reassigned by KHI per logType for investigation importance; does not match Cloud Logging severity — query `body(\"severity\", \"ERROR\")` for the raw Cloud Logging field)",
			}, "\n"),
		},
		{
			name:         "search_logs.md.tmpl with zero matched logs",
			templateName: "search_logs.md.tmpl",
			data: searchLogsTemplateData{
				MatchedLogCount: 0,
				Applied: workbench.AppliedFilter{
					ExcludeTimelinesWithoutLogs: false,
				},
				FilterVocabulary: workbench.FilterVocabulary{
					PathKeys:      []string{"kind", "namespace"},
					TimelineTypes: []string{"kind", "namespace"},
					LogTypes:      []string{"k8s audit"},
				},
			},
			want: strings.Join([]string{
				"# Logs matching the filter",
				"",
				"No logs matched the filter.",
				"",
				mdtemplate.FormatAppliedFilter(workbench.AppliedFilter{}),
				"",
				"## Filter vocabulary",
				"These are the values that filter CEL can match in this inspection.",
				"- path keys (lowercased timelineType): `kind`, `namespace`",
				"- timelineType: `kind`, `namespace`",
				"- logType: `k8s audit`",
				"- severity: `UNKNOWN`, `INFO`, `WARNING`, `ERROR`, `FATAL` (reassigned by KHI per logType for investigation importance; does not match Cloud Logging severity — query `body(\"severity\", \"ERROR\")` for the raw Cloud Logging field)",
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
