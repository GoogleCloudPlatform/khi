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

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
	"github.com/google/go-cmp/cmp"
)

func TestRenderGitHubAnnotationsPrioritizesLargeUncoveredSpans(t *testing.T) {
	warnings := []DirectiveWarning{
		{File: "pkg/a/single.go", Line: 3, Message: "invalid directive"},
	}
	report := BuildMutationReport("HEAD^1", nil, warnings, nil)
	AttachLineCoverage(report, &CoverageReport{
		BaseLabel:            "HEAD^1",
		PatchExecutableLines: 15,
		Files: []FileCoverageReport{
			{
				File:                   "pkg/a/single.go",
				ChangedExecutableLines: 1,
				UncoveredSpans: []UncoveredSpan{
					{File: "pkg/a/single.go", StartLine: 8, EndLine: 8, ExecutableLines: 1},
				},
			},
			{
				File:                   "pkg/b/boundary.go",
				ChangedExecutableLines: 4,
				UncoveredSpans: []UncoveredSpan{
					{File: "pkg/b/boundary.go", StartLine: 20, EndLine: 23, ExecutableLines: 4},
				},
			},
			{
				File:                   "pkg/c/large.go",
				ChangedExecutableLines: 10,
				UncoveredSpans: []UncoveredSpan{
					{File: "pkg/c/large.go", StartLine: 40, EndLine: 51, ExecutableLines: 10},
				},
			},
		},
	})

	// Cap=1 stops immediately after the warning without emitting any uncovered spans.
	var capOneBuf bytes.Buffer
	RenderGitHubAnnotations(&capOneBuf, report, 1, 4)
	wantCapOne := "::warning file=pkg/a/single.go,line=3,title=Invalid NOMUTANTS Directive::invalid directive\n"
	if diff := cmp.Diff(wantCapOne, capOneBuf.String()); diff != "" {
		t.Errorf("RenderGitHubAnnotations(cap=1) mismatch (-want +got):\n%s", diff)
	}

	// Cap=2 emits the 1 warning plus the single largest uncovered span (remain = 2 - 1 = 1).
	var capTwoBuf bytes.Buffer
	RenderGitHubAnnotations(&capTwoBuf, report, 2, 4)
	wantCapTwo := `::warning file=pkg/a/single.go,line=3,title=Invalid NOMUTANTS Directive::invalid directive
::warning file=pkg/c/large.go,line=40,endLine=51,title=Uncovered Lines in PR Diff::Changed Go block (40-51, 10 executable lines) is not covered by unit tests.
`
	if diff := cmp.Diff(wantCapTwo, capTwoBuf.String()); diff != "" {
		t.Errorf("RenderGitHubAnnotations(cap=2) mismatch (-want +got):\n%s", diff)
	}

	// Cap=3 includes exact minUncoveredLines=4 boundary (boundary.go) while still excluding single.go (1 line).
	var buf bytes.Buffer
	RenderGitHubAnnotations(&buf, report, 3, 4)
	want := `::warning file=pkg/a/single.go,line=3,title=Invalid NOMUTANTS Directive::invalid directive
::warning file=pkg/b/boundary.go,line=20,endLine=23,title=Uncovered Lines in PR Diff::Changed Go block (20-23, 4 executable lines) is not covered by unit tests.
::warning file=pkg/c/large.go,line=40,endLine=51,title=Uncovered Lines in PR Diff::Changed Go block (40-51, 10 executable lines) is not covered by unit tests.
`
	if diff := cmp.Diff(want, buf.String()); diff != "" {
		t.Errorf("RenderGitHubAnnotations(cap=3, min=4) mismatch (-want +got):\n%s", diff)
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

	redundant := []RedundantTestFinding{
		{
			GlobalIndex:     1,
			Kind:            RedundantWeakAssertion,
			File:            "pkg/a/logic_test.go",
			Line:            42,
			TestName:        "TestLogicSmoke",
			DominatedBy:     "TestLogic",
			DominatedByFile: "pkg/a/logic_test.go",
			DominatedByLine: 10,
		},
	}

	report := BuildMutationReport("HEAD^1", plans, nil, redundant)

	var annBuf bytes.Buffer
	RenderGitHubAnnotations(&annBuf, report, 1, 4)
	gotAnn := strings.TrimSpace(annBuf.String())
	if strings.Count(gotAnn, "::warning ") != 1 || !strings.Contains(gotAnn, "line=12") {
		t.Errorf("expected exactly 1 capped GitHub annotation on line 12, got: %q", gotAnn)
	}

	var allAnnBuf bytes.Buffer
	RenderGitHubAnnotations(&allAnnBuf, report, 10, 4)
	if !strings.Contains(allAnnBuf.String(), "Redundant Test in PR Diff (NO OUTPUT CHECK)") {
		t.Errorf("expected redundant test annotation in uncapped output, got: %q", allAnnBuf.String())
	}

	var zeroAnnBuf bytes.Buffer
	RenderGitHubAnnotations(&zeroAnnBuf, report, 0, 4)
	if zeroAnnBuf.Len() != 0 {
		t.Errorf("expected empty output when maxAnnotations=0, got: %q", zeroAnnBuf.String())
	}

	noSurvivingOrRedundant := BuildMutationReport("HEAD^1", []*FilePlan{{
		File: "pkg/a/logic.go",
		Mutants: []*Mutant{{
			GlobalIndex: 1,
			File:        "pkg/a/logic.go",
			Line:        20,
			Operator:    OperatorAOR,
			Status:      MutantKilled,
		}},
	}}, nil, nil)
	var cleanTextBuf, cleanSumBuf bytes.Buffer
	RenderTextReport(&cleanTextBuf, noSurvivingOrRedundant, gotestutil.TermStyle{Enabled: false})
	RenderGitHubSummary(&cleanSumBuf, noSurvivingOrRedundant)
	if strings.Contains(cleanTextBuf.String(), "Surviving Mutants") || strings.Contains(cleanTextBuf.String(), "Redundant Tests") {
		t.Errorf("unexpected failure sections in clean text report:\n%s", cleanTextBuf.String())
	}
	if strings.Contains(cleanSumBuf.String(), "Surviving Mutants") || strings.Contains(cleanSumBuf.String(), "Redundant Tests") {
		t.Errorf("unexpected failure sections in clean GitHub summary:\n%s", cleanSumBuf.String())
	}

	var sumBuf bytes.Buffer
	RenderGitHubSummary(&sumBuf, report)
	gotSum := sumBuf.String()
	for _, snippet := range []string{
		"## Go Differential Test & Mutation Report",
		"**`60.0%`** (3 killed / 5 tested)",
		"1 budget-skipped",
		"| `pkg/a/logic.go` | `60.0%` (3/5) | 3 | 2 | 0 | 0 | - |",
		"- if x < 5 {",
		"+ if x <= 5 {",
		"How to Triage Surviving Mutants (Without Test Bloat)",
		"`cmp.Diff`",
		"Redundant Tests in Diff (1)",
		"| `TestLogicSmoke` (`pkg/a/logic_test.go:42`) | `TestLogic` (line 10) | **NO OUTPUT CHECK** — Only checks 'err != nil'",
	} {
		if !strings.Contains(gotSum, snippet) {
			t.Errorf("RenderGitHubSummary() missing expected snippet %q in:\n%s", snippet, gotSum)
		}
	}
}

func TestRenderGitHubSummaryTypeScriptLanguageFences(t *testing.T) {
	plans := []*FilePlan{
		{
			File: "web/src/app/foo.ts",
			Mutants: []*Mutant{
				{
					GlobalIndex:  1,
					File:         "web/src/app/foo.ts",
					Line:         3,
					Operator:     OperatorROR,
					OriginalExpr: "x < 0",
					MutatedExpr:  "x <= 0",
					OriginalLine: "if (x < 0) {",
					MutatedLine:  "if (x <= 0) {",
					Status:       MutantSurvived,
				},
			},
		},
	}
	report := BuildMutationReportForLang(LangTS, "HEAD^1", plans, nil, nil)
	AttachLineCoverage(report, &CoverageReport{
		BaseLabel:            "HEAD^1",
		PatchExecutableLines: 4,
		PatchCoveredLines:    0,
		OverallStatements:    10,
		OverallCovered:       6,
		HasOverallCoverage:   true,
		Files: []FileCoverageReport{
			{
				File:                   "web/src/app/foo.ts",
				ChangedExecutableLines: 4,
				UncoveredRanges:        []LineRange{{StartLine: 5, EndLine: 8}},
				UncoveredSpans: []UncoveredSpan{
					{
						File:            "web/src/app/foo.ts",
						StartLine:       5,
						EndLine:         8,
						ExecutableLines: 4,
						Lines:           []SourceLine{{Number: 5, Text: "return x + 1;"}},
					},
				},
			},
		},
	})

	var annBuf, sumBuf bytes.Buffer
	RenderGitHubAnnotations(&annBuf, report, 10, 4)
	RenderGitHubSummary(&sumBuf, report)

	if !strings.Contains(annBuf.String(), "Changed TypeScript block (5-8, 4 executable lines)") {
		t.Errorf("expected TypeScript block annotation, got: %q", annBuf.String())
	}
	gotSum := sumBuf.String()
	for _, wantSub := range []string{
		"## TypeScript Differential Test & Mutation Report",
		"**Overall TypeScript Statement Coverage**: `60.0%` (6 / 10 statements)",
		"```ts",
		"   5 | return x + 1;",
		"`*.spec.ts`",
	} {
		if !strings.Contains(gotSum, wantSub) {
			t.Errorf("RenderGitHubSummary(LangTS) missing %q in:\n%s", wantSub, gotSum)
		}
	}
}
