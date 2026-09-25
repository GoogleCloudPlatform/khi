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
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed testdata/*.tmpl testdata/*.golden.md
var testdataFS embed.FS

func TestCode(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain string without backticks",
			input: "my-id-123",
			want:  "`my-id-123`",
		},
		{
			name:  "empty string",
			input: "",
			want:  "``",
		},
		{
			name:  "embedded single backtick",
			input: "foo`bar",
			want:  "``foo`bar``",
		},
		{
			name:  "leading backtick",
			input: "`foo",
			want:  "`` `foo ``",
		},
		{
			name:  "trailing backtick",
			input: "foo`",
			want:  "`` foo` ``",
		},
		{
			name:  "both leading and trailing backtick",
			input: "`foo`",
			want:  "`` `foo` ``",
		},
		{
			name:  "embedded double backticks",
			input: "foo``bar",
			want:  "```foo``bar```",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Code(tc.input)
			if got != tc.want {
				t.Errorf("Code(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCell(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "escapes pipe character",
			input: "column|value",
			want:  `column\|value`,
		},
		{
			name:  "replaces LF with space",
			input: "line1\nline2",
			want:  "line1 line2",
		},
		{
			name:  "replaces CRLF with space",
			input: "line1\r\nline2",
			want:  "line1 line2",
		},
		{
			name:  "replaces CR with space",
			input: "line1\rline2",
			want:  "line1 line2",
		},
		{
			name:  "replaces multiple pipes and newlines",
			input: "a|b\r\nc|d\ne",
			want:  `a\|b c\|d e`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Cell(tc.input)
			if got != tc.want {
				t.Errorf("Cell(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

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
			name:  "fraction rounded to integer",
			input: 0.42,
			want:  "42%",
		},
		{
			name:  "fraction rounded up",
			input: 0.426,
			want:  "43%",
		},
		{
			name:  "one hundred percent",
			input: 1.0,
			want:  "100%",
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

func TestFence(t *testing.T) {
	testCases := []struct {
		name     string
		lang     string
		body     string
		wantBody string
	}{
		{
			name:     "simple block",
			lang:     "yaml",
			body:     "key: value",
			wantBody: "```yaml\nkey: value\n```",
		},
		{
			name:     "trims single trailing newline",
			lang:     "yaml",
			body:     "key: value\n",
			wantBody: "```yaml\nkey: value\n```",
		},
		{
			name:     "trims single trailing CRLF",
			lang:     "yaml",
			body:     "key: value\r\n",
			wantBody: "```yaml\nkey: value\n```",
		},
		{
			name:     "enclosing body with backticks",
			lang:     "markdown",
			body:     "```code```",
			wantBody: "````markdown\n```code```\n````",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Fence(tc.lang, tc.body)
			if got != tc.wantBody {
				t.Errorf("Fence(%q, %q) = %q, want %q", tc.lang, tc.body, got, tc.wantBody)
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
		name              string
		items             []int
		pageSize          int
		defaultPageSize   int
		maxPageSize       int
		pageToken         string
		wantLen           int
		wantStart         int
		wantEnd           int
		wantTotal         int
		wantNextPageToken string
		wantErr           bool
	}{
		{
			name:              "page 1 of 120 items with pageSize 50",
			items:             items,
			pageSize:          50,
			defaultPageSize:   50,
			maxPageSize:       100,
			pageToken:         "",
			wantLen:           50,
			wantStart:         1,
			wantEnd:           50,
			wantTotal:         120,
			wantNextPageToken: "eyJvZmZzZXQiOjUwfQ",
		},
		{
			name:              "page 2 of 120 items with pageSize 50",
			items:             items,
			pageSize:          50,
			defaultPageSize:   50,
			maxPageSize:       100,
			pageToken:         "eyJvZmZzZXQiOjUwfQ",
			wantLen:           50,
			wantStart:         51,
			wantEnd:           100,
			wantTotal:         120,
			wantNextPageToken: "eyJvZmZzZXQiOjEwMH0",
		},
		{
			name:              "page 3 of 120 items with pageSize 50 (last page)",
			items:             items,
			pageSize:          50,
			defaultPageSize:   50,
			maxPageSize:       100,
			pageToken:         "eyJvZmZzZXQiOjEwMH0",
			wantLen:           20,
			wantStart:         101,
			wantEnd:           120,
			wantTotal:         120,
			wantNextPageToken: "",
		},
		{
			name:              "caps at maxPageSize when requested pageSize is too large",
			items:             items,
			pageSize:          200,
			defaultPageSize:   50,
			maxPageSize:       30,
			pageToken:         "",
			wantLen:           30,
			wantStart:         1,
			wantEnd:           30,
			wantTotal:         120,
			wantNextPageToken: "eyJvZmZzZXQiOjMwfQ",
		},
		{
			name:              "uses defaultPageSize when pageSize is non-positive",
			items:             items,
			pageSize:          0,
			defaultPageSize:   25,
			maxPageSize:       100,
			pageToken:         "",
			wantLen:           25,
			wantStart:         1,
			wantEnd:           25,
			wantTotal:         120,
			wantNextPageToken: "eyJvZmZzZXQiOjI1fQ",
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
			name:              "empty items slice returns zeroed ranges and no next page token",
			items:             nil,
			pageSize:          50,
			defaultPageSize:   50,
			maxPageSize:       100,
			pageToken:         "",
			wantLen:           0,
			wantStart:         0,
			wantEnd:           0,
			wantTotal:         0,
			wantNextPageToken: "",
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
			if len(got.Items) != tc.wantLen {
				t.Errorf("len(got.Items) = %d, want %d", len(got.Items), tc.wantLen)
			}
			if got.Start != tc.wantStart {
				t.Errorf("got.Start = %d, want %d", got.Start, tc.wantStart)
			}
			if got.End != tc.wantEnd {
				t.Errorf("got.End = %d, want %d", got.End, tc.wantEnd)
			}
			if got.Total != tc.wantTotal {
				t.Errorf("got.Total = %d, want %d", got.Total, tc.wantTotal)
			}
			if got.NextPageToken != tc.wantNextPageToken {
				t.Errorf("got.NextPageToken = %q, want %q", got.NextPageToken, tc.wantNextPageToken)
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
			if gotFormatted != tc.wantMessage {
				t.Errorf("FormatError() mismatch (-want +got):\n%s", cmp.Diff(tc.wantMessage, gotFormatted))
			}

			res, out, err := ErrorResult(tc.code, tc.bullets...)
			if err != nil {
				t.Fatalf("ErrorResult() returned error: %v", err)
			}
			if out != nil {
				t.Errorf("ErrorResult() out = %v, want nil", out)
			}
			if !res.IsError {
				t.Error("ErrorResult() res.IsError = false, want true")
			}
			if len(res.Content) != 1 {
				t.Fatalf("len(res.Content) = %d, want 1", len(res.Content))
			}
			textContent, ok := res.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("res.Content[0] type = %T, want *mcp.TextContent", res.Content[0])
			}
			if diff := cmp.Diff(tc.wantMessage, textContent.Text); diff != "" {
				t.Errorf("ErrorResult() text mismatch (-want +got):\n%s", diff)
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

	// Verify ToolResult wrapping
	toolResult, extra, err := ts.ToolResult("inspections.md.tmpl", data)
	if err != nil {
		t.Fatalf("ts.ToolResult() error: %v", err)
	}
	if extra != nil {
		t.Errorf("extra = %v, want nil", extra)
	}
	if len(toolResult.Content) != 1 {
		t.Fatalf("len(toolResult.Content) = %d, want 1", len(toolResult.Content))
	}
	textItem, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("toolResult.Content[0] type is %T, want *mcp.TextContent", toolResult.Content[0])
	}
	if diff := cmp.Diff(want, textItem.Text); diff != "" {
		t.Errorf("ToolResult content mismatch (-want +got):\n%s", diff)
	}

	// Verify ResourceResult wrapping
	resResult, err := ts.ResourceResult("khi://inspections", "inspections.md.tmpl", data)
	if err != nil {
		t.Fatalf("ts.ResourceResult() error: %v", err)
	}
	if len(resResult.Contents) != 1 {
		t.Fatalf("len(resResult.Contents) = %d, want 1", len(resResult.Contents))
	}
	rc := resResult.Contents[0]
	if rc.URI != "khi://inspections" {
		t.Errorf("rc.URI = %q, want %q", rc.URI, "khi://inspections")
	}
	if rc.MIMEType != "text/markdown" {
		t.Errorf("rc.MIMEType = %q, want %q", rc.MIMEType, "text/markdown")
	}
	if diff := cmp.Diff(want, rc.Text); diff != "" {
		t.Errorf("ResourceResult text mismatch (-want +got):\n%s", diff)
	}
}

func TestSet_PaginatedLogs(t *testing.T) {
	ts := MustParse(testdataFS, "testdata/paginated_logs.md.tmpl")

	testCases := []struct {
		name          string
		data          PageResult[string]
		wantLastLine  string
		wantHasFooter bool
	}{
		{
			name: "page with next page token contains pageFooter line",
			data: PageResult[string]{
				Items:         []string{"log message 1", "log message 2"},
				Start:         1,
				End:           2,
				Total:         10,
				NextPageToken: "eyJvZmZzZXQiOjJ9",
			},
			wantLastLine:  `pageToken: "eyJvZmZzZXQiOjJ9"`,
			wantHasFooter: true,
		},
		{
			name: "last page without token does not contain pageFooter line",
			data: PageResult[string]{
				Items:         []string{"log message 9", "log message 10"},
				Start:         9,
				End:           10,
				Total:         10,
				NextPageToken: "",
			},
			wantLastLine:  "- log message 10",
			wantHasFooter: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ts.Render("paginated_logs.md.tmpl", tc.data)
			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}
			lines := strings.Split(got, "\n")
			lastLine := lines[len(lines)-1]
			if lastLine != tc.wantLastLine {
				t.Errorf("lastLine = %q, want %q", lastLine, tc.wantLastLine)
			}
			hasTokenLine := strings.Contains(got, "pageToken:")
			if hasTokenLine != tc.wantHasFooter {
				t.Errorf("strings.Contains(got, %q) = %v, want %v\nFull output:\n%s", "pageToken:", hasTokenLine, tc.wantHasFooter, got)
			}
		})
	}
}
