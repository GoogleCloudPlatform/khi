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

package mdtemplate

import (
	"embed"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/*.tmpl testdata/*.golden.md
var testdataFS embed.FS

var ignoreMCPUnexported = cmpopts.IgnoreUnexported(
	mcp.CallToolResult{},
	mcp.TextContent{},
	mcp.ReadResourceResult{},
	mcp.ResourceContents{},
)

func TestFormatTime(t *testing.T) {
	testCases := []struct {
		name  string
		input time.Time
		want  string
	}{
		{
			name:  "zero time returns empty string",
			input: time.Time{},
			want:  "",
		},
		{
			name:  "UTC time formatted as RFC3339",
			input: time.Date(2026, time.September, 24, 1, 0, 0, 0, time.UTC),
			want:  "2026-09-24T01:00:00Z",
		},
		{
			name:  "timezone converted to UTC",
			input: time.Date(2026, time.September, 24, 10, 0, 0, 0, time.FixedZone("JST", 9*3600)),
			want:  "2026-09-24T01:00:00Z",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatTime(tc.input)
			if got != tc.want {
				t.Errorf("FormatTime(%v) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestPercent(t *testing.T) {
	testCases := []struct {
		name  string
		input float64
		want  string
	}{
		{
			name:  "zero percent",
			input: 0.0,
			want:  "0%",
		},
		{
			name:  "fraction rounded to nearest integer",
			input: 0.426,
			want:  "43%",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Percent(tc.input)
			if got != tc.want {
				t.Errorf("Percent(%f) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestPageFooter(t *testing.T) {
	testCases := []struct {
		name  string
		token string
		want  string
	}{
		{
			name:  "empty token",
			token: "",
			want:  "",
		},
		{
			name:  "non-empty token",
			token: "token123",
			want:  `pageToken: "token123"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := PageFooter(tc.token)
			if got != tc.want {
				t.Errorf("PageFooter(%q) = %q, want %q", tc.token, got, tc.want)
			}
		})
	}
}

func TestPaginate(t *testing.T) {
	items := make([]int, 120)
	for i := 0; i < 120; i++ {
		items[i] = i + 1
	}

	testCases := []struct {
		name            string
		items           []int
		pageSize        int
		defaultPageSize int
		maxPageSize     int
		pageToken       string
		want            PageResult[int]
		wantErr         bool
	}{
		{
			name:            "page 1 of 120 items with pageSize 50",
			items:           items,
			pageSize:        50,
			defaultPageSize: 50,
			maxPageSize:     100,
			pageToken:       "",
			want: PageResult[int]{
				Items:         items[:50],
				Start:         1,
				End:           50,
				Total:         120,
				NextPageToken: "eyJvZmZzZXQiOjUwfQ",
			},
		},
		{
			name:            "page 2 of 120 items with pageSize 50",
			items:           items,
			pageSize:        50,
			defaultPageSize: 50,
			maxPageSize:     100,
			pageToken:       "eyJvZmZzZXQiOjUwfQ",
			want: PageResult[int]{
				Items:         items[50:100],
				Start:         51,
				End:           100,
				Total:         120,
				NextPageToken: "eyJvZmZzZXQiOjEwMH0",
			},
		},
		{
			name:            "page 3 of 120 items with pageSize 50 (last page)",
			items:           items,
			pageSize:        50,
			defaultPageSize: 50,
			maxPageSize:     100,
			pageToken:       "eyJvZmZzZXQiOjEwMH0",
			want: PageResult[int]{
				Items:         items[100:120],
				Start:         101,
				End:           120,
				Total:         120,
				NextPageToken: "",
			},
		},
		{
			name:            "caps at maxPageSize when requested pageSize is too large",
			items:           items,
			pageSize:        200,
			defaultPageSize: 50,
			maxPageSize:     30,
			pageToken:       "",
			want: PageResult[int]{
				Items:         items[:30],
				Start:         1,
				End:           30,
				Total:         120,
				NextPageToken: "eyJvZmZzZXQiOjMwfQ",
			},
		},
		{
			name:            "uses defaultPageSize when pageSize is non-positive",
			items:           items,
			pageSize:        0,
			defaultPageSize: 25,
			maxPageSize:     100,
			pageToken:       "",
			want: PageResult[int]{
				Items:         items[:25],
				Start:         1,
				End:           25,
				Total:         120,
				NextPageToken: "eyJvZmZzZXQiOjI1fQ",
			},
		},
		{
			name:            "returns error for invalid pageToken",
			items:           items,
			pageSize:        50,
			defaultPageSize: 50,
			maxPageSize:     100,
			pageToken:       "not-base64-json",
			wantErr:         true,
		},
		{
			name:            "empty items slice returns zeroed ranges and no next page token",
			items:           nil,
			pageSize:        50,
			defaultPageSize: 50,
			maxPageSize:     100,
			pageToken:       "",
			want: PageResult[int]{
				Items: []int(nil),
			},
		},
		{
			name:            "returns error for negative offset pageToken",
			items:           items,
			pageSize:        50,
			defaultPageSize: 50,
			maxPageSize:     100,
			pageToken:       EncodePageToken(-1),
			wantErr:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Paginate(tc.items, tc.pageSize, tc.defaultPageSize, tc.maxPageSize, tc.pageToken)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Paginate() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Paginate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatErrorAndErrorResult(t *testing.T) {
	testCases := []struct {
		name        string
		code        string
		bullets     []string
		wantMessage string
	}{
		{
			name:        "error code without bullets",
			code:        "NOT_FOUND",
			bullets:     nil,
			wantMessage: "Error: NOT_FOUND",
		},
		{
			name:        "error code with bullets",
			code:        "INVALID_ARGUMENT",
			bullets:     []string{"field foo is required", "must be positive"},
			wantMessage: "Error: INVALID_ARGUMENT\n\n- field foo is required\n- must be positive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotFormatted := FormatError(tc.code, tc.bullets...)
			if diff := cmp.Diff(tc.wantMessage, gotFormatted); diff != "" {
				t.Errorf("FormatError() mismatch (-want +got):\n%s", diff)
			}

			res, out, err := ErrorResult(tc.code, tc.bullets...)
			if err != nil {
				t.Fatalf("ErrorResult() returned error: %v", err)
			}
			if out != nil {
				t.Errorf("ErrorResult() out = %v, want nil", out)
			}
			wantRes := &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: tc.wantMessage}},
			}
			if diff := cmp.Diff(wantRes, res, ignoreMCPUnexported); diff != "" {
				t.Errorf("ErrorResult() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type inspectionTestItem struct {
	ID     string
	Name   string
	Type   string
	Status string
	Start  time.Time
	End    time.Time
	Labels string
}

type inspectionTestData struct {
	Inspections []inspectionTestItem
}

func TestSet_InspectionsGolden(t *testing.T) {
	ts := MustParse(testdataFS, "testdata/inspections.md.tmpl")

	data := inspectionTestData{
		Inspections: []inspectionTestItem{
			{
				ID:     "2026-09-24-0130-a1b2",
				Name:   "prod-cluster-1 restart investigation",
				Type:   "Google Kubernetes Engine",
				Status: "DONE",
				Start:  time.Date(2026, time.September, 24, 1, 0, 0, 0, time.UTC),
				End:    time.Date(2026, time.September, 24, 2, 0, 0, 0, time.UTC),
				Labels: "projectId=my-gcp-project, clusterName=prod-cluster-1, location=us-central1",
			},
			{
				ID:     "2026-09-23-2210-c3d4",
				Name:   "staging upgrade check",
				Type:   "Google Kubernetes Engine",
				Status: "RUNNING",
				Start:  time.Date(2026, time.September, 23, 20, 0, 0, 0, time.UTC),
				End:    time.Date(2026, time.September, 23, 22, 0, 0, 0, time.UTC),
				Labels: "projectId=my-gcp-project, clusterName=staging-cluster",
			},
		},
	}

	got, err := ts.Render("inspections.md.tmpl", data)
	if err != nil {
		t.Fatalf("ts.Render() error: %v", err)
	}

	goldenBytes, err := testdataFS.ReadFile("testdata/inspections.golden.md")
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}
	want := strings.TrimRight(string(goldenBytes), "\r\n")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Render(inspections.md.tmpl) mismatch (-want +got):\n%s", diff)
	}

	toolResult, extra, err := ts.ToolResult("inspections.md.tmpl", data)
	if err != nil {
		t.Fatalf("ts.ToolResult() error: %v", err)
	}
	if extra != nil {
		t.Errorf("extra = %v, want nil", extra)
	}
	wantToolResult := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: want}},
	}
	if diff := cmp.Diff(wantToolResult, toolResult, ignoreMCPUnexported); diff != "" {
		t.Errorf("ToolResult() mismatch (-want +got):\n%s", diff)
	}

	errToolResult, errExtra, err := ts.ErrorToolResult("inspections.md.tmpl", data)
	if err != nil {
		t.Fatalf("ts.ErrorToolResult() error: %v", err)
	}
	if errExtra != nil {
		t.Errorf("errExtra = %v, want nil", errExtra)
	}
	wantErrToolResult := &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: want}},
	}
	if diff := cmp.Diff(wantErrToolResult, errToolResult, ignoreMCPUnexported); diff != "" {
		t.Errorf("ErrorToolResult() mismatch (-want +got):\n%s", diff)
	}

	resResult, err := ts.ResourceResult("khi://inspections", "inspections.md.tmpl", data)
	if err != nil {
		t.Fatalf("ts.ResourceResult() error: %v", err)
	}
	wantResResult := &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "khi://inspections",
				MIMEType: "text/markdown",
				Text:     want,
			},
		},
	}
	if diff := cmp.Diff(wantResResult, resResult, ignoreMCPUnexported); diff != "" {
		t.Errorf("ResourceResult() mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatAppliedFilter(t *testing.T) {
	testCases := []struct {
		name  string
		input workbench.AppliedFilter
		want  string
	}{
		{
			name: "full filter with timestamps",
			input: workbench.AppliedFilter{
				TimelineQuery:               `t.path.namespace == "kube-system"`,
				TimelineExclusionQuery:      `t.name.contains("pause")`,
				LogQuery:                    `severity >= WARNING`,
				StartTime:                   time.Date(2026, time.September, 24, 1, 0, 0, 0, time.UTC),
				EndTime:                     time.Date(2026, time.September, 24, 2, 0, 0, 0, time.UTC),
				ExcludeTimelinesWithoutLogs: true,
			},
			want: "## Applied filter\n" +
				"Enter these values in the KHI Web UI filter to see the same data.\n\n" +
				"```yaml\n" +
				"timelineQuery: |-\n  t.path.namespace == \"kube-system\"\n" +
				"timelineExclusionQuery: |-\n  t.name.contains(\"pause\")\n" +
				"logQuery: |-\n  severity >= WARNING\n" +
				"startTime: 2026-09-24T01:00:00Z\n" +
				"endTime: 2026-09-24T02:00:00Z\n" +
				"excludeTimelinesWithoutLogs: true\n" +
				"```",
		},
		{
			name: "queries that are not valid plain YAML scalars",
			input: workbench.AppliedFilter{
				TimelineQuery:          "!has(t.path.namespace)",
				TimelineExclusionQuery: "t.name == \"a\" ||\nt.name.startsWith(\"b: c\")",
				LogQuery:               "  l.summary.contains(\"#\")",
			},
			want: "## Applied filter\n" +
				"Enter these values in the KHI Web UI filter to see the same data.\n\n" +
				"```yaml\n" +
				"timelineQuery: |-\n  !has(t.path.namespace)\n" +
				"timelineExclusionQuery: |-\n  t.name == \"a\" ||\n  t.name.startsWith(\"b: c\")\n" +
				"logQuery: |2-\n    l.summary.contains(\"#\")\n" +
				"startTime: \"\"\n" +
				"endTime: \"\"\n" +
				"excludeTimelinesWithoutLogs: false\n" +
				"```",
		},
		{
			name: "queries led by tabs or blank lines",
			input: workbench.AppliedFilter{
				TimelineQuery:          "\tt.name == \"a\"",
				TimelineExclusionQuery: "\n\tt.name == \"b\"",
				LogQuery:               "\n  severity >= WARNING",
			},
			want: "## Applied filter\n" +
				"Enter these values in the KHI Web UI filter to see the same data.\n\n" +
				"```yaml\n" +
				"timelineQuery: |2-\n  \tt.name == \"a\"\n" +
				"timelineExclusionQuery: |2-\n  \n  \tt.name == \"b\"\n" +
				"logQuery: |2-\n  \n    severity >= WARNING\n" +
				"startTime: \"\"\n" +
				"endTime: \"\"\n" +
				"excludeTimelinesWithoutLogs: false\n" +
				"```",
		},
		{
			name: "queries with CRLF and CR line endings are normalized to LF",
			input: workbench.AppliedFilter{
				TimelineQuery:          "t.name == \"a\"\r\nt.name == \"b\"",
				TimelineExclusionQuery: "\r\n  t.name == \"c\"",
				LogQuery:               "severity >= WARNING\rl.summary.contains(\"err\")",
			},
			want: "## Applied filter\n" +
				"Enter these values in the KHI Web UI filter to see the same data.\n\n" +
				"```yaml\n" +
				"timelineQuery: |-\n  t.name == \"a\"\n  t.name == \"b\"\n" +
				"timelineExclusionQuery: |2-\n  \n    t.name == \"c\"\n" +
				"logQuery: |-\n  severity >= WARNING\n  l.summary.contains(\"err\")\n" +
				"startTime: \"\"\n" +
				"endTime: \"\"\n" +
				"excludeTimelinesWithoutLogs: false\n" +
				"```",
		},
		{
			name: "empty filter with zero timestamps",
			input: workbench.AppliedFilter{
				ExcludeTimelinesWithoutLogs: false,
			},
			want: "## Applied filter\n" +
				"Enter these values in the KHI Web UI filter to see the same data.\n\n" +
				"```yaml\n" +
				"timelineQuery: \"\"\n" +
				"timelineExclusionQuery: \"\"\n" +
				"logQuery: \"\"\n" +
				"startTime: \"\"\n" +
				"endTime: \"\"\n" +
				"excludeTimelinesWithoutLogs: false\n" +
				"```",
		},
	}

	type parsedQueries struct {
		TimelineQuery          string `yaml:"timelineQuery"`
		TimelineExclusionQuery string `yaml:"timelineExclusionQuery"`
		LogQuery               string `yaml:"logQuery"`
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatAppliedFilter(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FormatAppliedFilter() mismatch (-want +got):\n%s", diff)
			}

			_, yamlBody, found := strings.Cut(got, "```yaml\n")
			if !found {
				t.Fatalf("FormatAppliedFilter() output has no yaml block: %q", got)
			}
			var gotQueries parsedQueries
			if err := yaml.Unmarshal([]byte(strings.TrimSuffix(yamlBody, "```")), &gotQueries); err != nil {
				t.Fatalf("failed to parse the yaml block: %v", err)
			}
			normalizeLF := func(s string) string {
				return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
			}
			wantQueries := parsedQueries{
				TimelineQuery:          normalizeLF(tc.input.TimelineQuery),
				TimelineExclusionQuery: normalizeLF(tc.input.TimelineExclusionQuery),
				LogQuery:               normalizeLF(tc.input.LogQuery),
			}
			if diff := cmp.Diff(wantQueries, gotQueries); diff != "" {
				t.Errorf("parsed yaml queries mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatCELErrorAndCELErrorResult(t *testing.T) {
	wantMessage := "Error: INVALID_CEL\n\n" +
		"- Field: `filter.timelineQuery`\n" +
		"- Expression: `path.kind`\n" +
		"- Message: expression must evaluate to bool, got string"

	gotFormatted := FormatCELError("filter.timelineQuery", "path.kind", errors.New("expression must evaluate to bool, got string"))
	if diff := cmp.Diff(wantMessage, gotFormatted); diff != "" {
		t.Errorf("FormatCELError() mismatch (-want +got):\n%s", diff)
	}

	res, out, err := CELErrorResult("filter.timelineQuery", "path.kind", errors.New("expression must evaluate to bool, got string"))
	if err != nil {
		t.Fatalf("CELErrorResult() returned error: %v", err)
	}
	if out != nil {
		t.Errorf("CELErrorResult() out = %v, want nil", out)
	}
	wantRes := &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: wantMessage}},
	}
	if diff := cmp.Diff(wantRes, res, ignoreMCPUnexported); diff != "" {
		t.Errorf("CELErrorResult() mismatch (-want +got):\n%s", diff)
	}
}
