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

func TestRenderTextReportEmpty(t *testing.T) {
	report := BuildMutationReport("trunk()", nil, nil)
	var buf bytes.Buffer
	RenderTextReport(&buf, report)

	want := "Go Differential Mutation Score (base: trunk()): no mutable Go production expressions changed.\n"
	if buf.String() != want {
		t.Errorf("RenderTextReport() = %q, want %q", buf.String(), want)
	}
}

func TestRenderTextReportWithSurvivingAndWarnings(t *testing.T) {
	plans := []*FilePlan{
		{
			File: "pkg/scene/palette.go",
			Mutants: []*Mutant{
				{
					GlobalIndex:  1,
					File:         "pkg/scene/palette.go",
					Line:         40,
					Operator:     OperatorROR,
					OriginalExpr: "idx < maxColors",
					MutatedExpr:  "idx <= maxColors",
					OriginalLine: "if idx < maxColors {",
					MutatedLine:  "if idx <= maxColors {",
					Status:       MutantSurvived,
				},
				{
					GlobalIndex:  2,
					File:         "pkg/scene/palette.go",
					Line:         40,
					Operator:     OperatorROR,
					OriginalExpr: "idx < maxColors",
					MutatedExpr:  "idx >= maxColors",
					OriginalLine: "if idx < maxColors {",
					MutatedLine:  "if idx >= maxColors {",
					Status:       MutantKilled,
				},
				{
					GlobalIndex:  3,
					File:         "pkg/scene/palette.go",
					Line:         42,
					Operator:     OperatorAOR,
					OriginalExpr: "n + 1",
					MutatedExpr:  "n",
					OriginalLine: "return n + 1",
					MutatedLine:  "return n",
					Status:       MutantKilled,
				},
				{
					GlobalIndex:  4,
					File:         "pkg/scene/palette.go",
					Line:         43,
					Operator:     OperatorAOR,
					OriginalExpr: "n - 1",
					MutatedExpr:  "n + 1",
					OriginalLine: "return n - 1",
					MutatedLine:  "return n + 1",
					Status:       MutantKilled,
				},
				{
					GlobalIndex:  5,
					File:         "pkg/scene/palette.go",
					Line:         44,
					Operator:     OperatorAOR,
					OriginalExpr: "n * 2",
					MutatedExpr:  "n / 2",
					Status:       MutantUncovered,
				},
				{
					GlobalIndex:         6,
					File:                "pkg/scene/palette.go",
					Line:                45,
					Operator:            OperatorSBR,
					OriginalExpr:        "count++",
					MutatedExpr:         "/* skipped count++ */",
					SuppressionCategory: SuppressionObservability,
					Status:              MutantSuppressed,
				},
			},
		},
	}
	warnings := []DirectiveWarning{
		{
			File:    "pkg/scene/palette.go",
			Line:    50,
			Message: "// NOMUTANTS requires a category and reason: // NOMUTANTS(<category>): <reason> (allowed categories: equivalent, perf, observability, external)",
		},
	}

	report := BuildMutationReport("trunk()", plans, warnings)
	var buf bytes.Buffer
	RenderTextReport(&buf, report)

	want := `Go Differential Mutation Score (base: trunk()): 75.0% (3 killed / 4 tested, 1 uncovered, 1 suppressed (observability: 1))

FILE                  SCORE        KILLED  SURVIVED  UNCOVERED  SUPPRESSED
pkg/scene/palette.go  75.0% (3/4)  3       1         1          1

Warnings (invalid NOMUTANTS directives):
  pkg/scene/palette.go:50: // NOMUTANTS requires a category and reason: // NOMUTANTS(<category>): <reason> (allowed categories: equivalent, perf, observability, external)

Surviving mutants (tests passed despite logic change):
  [M1] ROR pkg/scene/palette.go:40
    - if idx < maxColors {
    + if idx <= maxColors {
`
	if diff := cmp.Diff(want, buf.String()); diff != "" {
		t.Errorf("RenderTextReport() mismatch (-want +got):\n%s", diff)
	}
}

func TestRenderGitHubAnnotationsAndSummary(t *testing.T) {
	plans := []*FilePlan{
		{
			File: "pkg/a/logic.go",
			Mutants: []*Mutant{
				{
					GlobalIndex:  1,
					File:         "pkg/a/logic.go",
					Line:         12,
					Operator:     OperatorROR,
					OriginalExpr: "x < 5",
					MutatedExpr:  "x <= 5",
					OriginalLine: "if x < 5 {",
					MutatedLine:  "if x <= 5 {",
					Status:       MutantSurvived,
				},
				{
					GlobalIndex:  2,
					File:         "pkg/a/logic.go",
					Line:         15,
					Operator:     OperatorROR,
					OriginalExpr: "x > 10",
					MutatedExpr:  "x >= 10",
					OriginalLine: "if x > 10 {",
					MutatedLine:  "if x >= 10 {",
					Status:       MutantSurvived,
				},
				{
					GlobalIndex:  3,
					File:         "pkg/a/logic.go",
					Line:         18,
					Operator:     OperatorSBR,
					OriginalExpr: "s.ready = true",
					MutatedExpr:  "/* skipped assignment */",
					OriginalLine: "s.ready = true",
					MutatedLine:  "/* skipped: s.ready = true */",
					Status:       MutantTimedOut,
				},
				{
					GlobalIndex:  4,
					File:         "pkg/a/logic.go",
					Line:         20,
					Operator:     OperatorAOR,
					OriginalExpr: "x + 1",
					MutatedExpr:  "x",
					Status:       MutantKilled,
				},
				{
					GlobalIndex:  5,
					File:         "pkg/a/logic.go",
					Line:         22,
					Operator:     OperatorAOR,
					OriginalExpr: "x - 1",
					MutatedExpr:  "x + 1",
					Status:       MutantKilled,
				},
				{
					GlobalIndex:  6,
					File:         "pkg/a/logic.go",
					Line:         25,
					Operator:     OperatorAOR,
					OriginalExpr: "x * 2",
					MutatedExpr:  "x / 2",
					Status:       MutantSkippedBudget,
				},
			},
		},
	}

	report := BuildMutationReport("HEAD^1", plans, nil)

	var annBuf bytes.Buffer
	RenderGitHubAnnotations(&annBuf, report, 1)
	gotAnn := strings.TrimSpace(annBuf.String())
	if strings.Count(gotAnn, "::warning ") != 1 || !strings.Contains(gotAnn, "line=12") {
		t.Errorf("expected exactly 1 capped GitHub annotation on line 12, got: %q", gotAnn)
	}

	var zeroAnnBuf bytes.Buffer
	RenderGitHubAnnotations(&zeroAnnBuf, report, 0)
	if zeroAnnBuf.Len() != 0 {
		t.Errorf("expected empty output when maxAnnotations=0, got: %q", zeroAnnBuf.String())
	}

	var sumBuf bytes.Buffer
	RenderGitHubSummary(&sumBuf, report)
	gotSum := sumBuf.String()
	for _, snippet := range []string{
		"## Go Differential Mutation Report",
		"**`60.0%`** (3 killed / 5 tested)",
		"1 budget-skipped",
		"| `pkg/a/logic.go` | `60.0%` (3/5) | 3 | 2 | 0 | 0 |",
		"- if x < 5 {",
		"+ if x <= 5 {",
		"How to Triage Surviving Mutants (Without Test Bloat)",
	} {
		if !strings.Contains(gotSum, snippet) {
			t.Errorf("RenderGitHubSummary() missing expected snippet %q in:\n%s", snippet, gotSum)
		}
	}
}
