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

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRenderTextReport(t *testing.T) {
	testCases := []struct {
		name   string
		report *CoverageReport
		want   string
	}{
		{
			name: "no changed executable lines",
			report: &CoverageReport{
				BaseLabel:          "trunk()",
				OverallStatements:  100,
				OverallCovered:     75,
				HasOverallCoverage: true,
			},
			want: `Go Differential Coverage (base: trunk()): no executable Go production lines changed.
Overall Go Statement Coverage: 75.0% (75/100 statements)
`,
		},
		{
			name: "renders file table and uncovered code snippets",
			report: &CoverageReport{
				BaseLabel:            "trunk()",
				PatchExecutableLines: 10,
				PatchCoveredLines:    8,
				Files: []FileCoverageReport{
					{
						File:                   "pkg/foo/bar.go",
						ChangedExecutableLines: 10,
						ChangedCoveredLines:    8,
						TotalStatements:        20,
						CoveredStatements:      15,
						UncoveredRanges: []LineRange{
							{StartLine: 42, EndLine: 43},
						},
						UncoveredSpans: []UncoveredSpan{
							{
								File:            "pkg/foo/bar.go",
								StartLine:       42,
								EndLine:         43,
								ExecutableLines: 2,
								Lines: []SourceLine{
									{Number: 41, Text: "if err != nil {"},
									{Number: 42, Text: "    return err"},
									{Number: 43, Text: "    unreachable()"},
								},
							},
						},
					},
				},
			},
			want: `Go Differential Coverage (base: trunk()): 80.0% (8/10 executable changed lines covered)

FILE            PATCH COV     FILE COV  UNCOVERED DIFF LINES
pkg/foo/bar.go  80.0% (8/10)  75.0%     42-43

Uncovered changed code:
  pkg/foo/bar.go:42-43
      41 | if err != nil {
      42 |     return err
      43 |     unreachable()
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			RenderTextReport(&buf, tc.report)
			if diff := cmp.Diff(tc.want, buf.String()); diff != "" {
				t.Errorf("RenderTextReport() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRenderGitHubAnnotations(t *testing.T) {
	testCases := []struct {
		name               string
		report             *CoverageReport
		maxAnnotations     int
		minAnnotationLines int
		want               string
	}{
		{
			name: "skips small 1-3 line guards and prioritizes larger uncovered blocks",
			report: &CoverageReport{
				Files: []FileCoverageReport{
					{
						File: "pkg/a/guard.go",
						UncoveredSpans: []UncoveredSpan{
							{
								File:            "pkg/a/guard.go",
								StartLine:       12,
								EndLine:         13,
								ExecutableLines: 2,
							},
						},
					},
					{
						File: "pkg/b/medium.go",
						UncoveredSpans: []UncoveredSpan{
							{
								File:            "pkg/b/medium.go",
								StartLine:       20,
								EndLine:         24,
								ExecutableLines: 5,
							},
						},
					},
					{
						File: "pkg/c/large.go",
						UncoveredSpans: []UncoveredSpan{
							{
								File:            "pkg/c/large.go",
								StartLine:       40,
								EndLine:         51,
								ExecutableLines: 10,
							},
						},
					},
				},
			},
			maxAnnotations:     1,
			minAnnotationLines: 4,
			want: `::warning file=pkg/c/large.go,line=40,endLine=51,title=Uncovered Lines in PR Diff::Changed Go block (40-51, 10 executable lines) is not covered by unit tests.
`,
		},
		{
			name: "emits nothing when only trivial guards are uncovered",
			report: &CoverageReport{
				Files: []FileCoverageReport{
					{
						File: "pkg/a/guard.go",
						UncoveredSpans: []UncoveredSpan{
							{
								File:            "pkg/a/guard.go",
								StartLine:       10,
								EndLine:         10,
								ExecutableLines: 1,
							},
						},
					},
				},
			},
			maxAnnotations:     10,
			minAnnotationLines: 4,
			want:               "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			RenderGitHubAnnotations(&buf, tc.report, tc.maxAnnotations, tc.minAnnotationLines)
			if diff := cmp.Diff(tc.want, buf.String()); diff != "" {
				t.Errorf("RenderGitHubAnnotations() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRenderGitHubSummary(t *testing.T) {
	testCases := []struct {
		name           string
		report         *CoverageReport
		wantSubstrings []string
	}{
		{
			name: "no changed executable lines",
			report: &CoverageReport{
				BaseLabel:          "HEAD^1",
				OverallStatements:  100,
				OverallCovered:     80,
				HasOverallCoverage: true,
			},
			wantSubstrings: []string{
				"## Go Differential Coverage Report",
				"No executable Go production lines were added or modified in this change.",
				"**Overall Go Statement Coverage**: `80.0%` (80 / 100 statements)",
			},
		},
		{
			name: "changed lines with uncovered snippet",
			report: &CoverageReport{
				BaseLabel:            "HEAD^1",
				PatchExecutableLines: 4,
				PatchCoveredLines:    3,
				OverallStatements:    200,
				OverallCovered:       150,
				HasOverallCoverage:   true,
				Files: []FileCoverageReport{
					{
						File:                   "pkg/foo/bar.go",
						ChangedExecutableLines: 4,
						ChangedCoveredLines:    3,
						TotalStatements:        10,
						CoveredStatements:      8,
						UncoveredRanges: []LineRange{
							{StartLine: 18, EndLine: 18},
						},
						UncoveredSpans: []UncoveredSpan{
							{
								File:            "pkg/foo/bar.go",
								StartLine:       18,
								EndLine:         18,
								ExecutableLines: 1,
								Lines: []SourceLine{
									{Number: 18, Text: "return err"},
								},
							},
						},
					},
				},
			},
			wantSubstrings: []string{
				"## Go Differential Coverage Report",
				"**Patch Coverage (Changed Lines)**: **`75.0%`** (3 / 4 executable lines covered)",
				"**Overall Go Statement Coverage**: `75.0%` (150 / 200 statements)",
				"| `pkg/foo/bar.go` | `75.0%` (3/4) | `80.0%` | `18` |",
				"<summary>Uncovered Changed Code Snippets (1)</summary>",
				"**`pkg/foo/bar.go:18`** (1 executable lines)",
				"  18 | return err",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			RenderGitHubSummary(&buf, tc.report)
			got := buf.String()
			for _, sub := range tc.wantSubstrings {
				if !strings.Contains(got, sub) {
					t.Errorf("RenderGitHubSummary() output missing %q in:\n%s", sub, got)
				}
			}
		})
	}
}
