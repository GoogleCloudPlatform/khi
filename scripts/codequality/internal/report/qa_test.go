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

package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func TestConvertReportsToQAReportFilteringAndDetails(t *testing.T) {
	testCases := []struct {
		name      string
		baseLabel string
		speed     rule.Speed
		filter    rule.Filter
		reports   []*rule.Report
		want      *rule.QAReport
	}{
		// Strips advisory items when filter is FilterMust and preserves enriched finding details.
		{
			name:      "FilterMust drops advisory items and enriches finding details",
			baseLabel: "main",
			speed:     rule.SpeedSlow,
			filter:    rule.FilterMust,
			reports: []*rule.Report{
				nil,
				{
					Language: "",
					DirectiveErrors: []rule.DirectiveError{
						{File: "pkg/a/a.go", Line: 2, Message: "unused directive"},
					},
					Decisions: rule.DecisionSummary{
						Enabled: true,
						Findings: []rule.DecisionItemFinding{
							{RuleID: "r1", QuestionID: "q1", Blocking: true, File: "pkg/a/a.go", Line: 5, Subject: "Decide", Outcome: "fail", Why: "bad", Fix: "fix"},
						},
					},
					Findings: []rule.Finding{
						{
							Rule:    rule.RuleFuncUncoveredLines,
							Kind:    rule.ViolationHardLimit,
							File:    "pkg/a/a.go",
							Line:    10,
							Subject: "Run",
							After:   2,
							Limit:   0,
							UncoveredSpans: []rule.UncoveredSpan{
								{StartLine: 12, EndLine: 12},
								{StartLine: 15, EndLine: 16},
							},
						},
						{
							Rule:    rule.RuleFuncSurvivingMutants,
							Kind:    rule.ViolationHardLimit,
							File:    "pkg/a/a.go",
							Line:    20,
							Subject: "Check",
							After:   1,
							Limit:   0,
							SurvivingMutants: []rule.MutantDetail{
								{Operator: rule.OperatorROR, Line: 22, OriginalExpr: "a == b", MutatedExpr: "a != b"},
							},
						},
						{
							Rule:    rule.RuleTestRedundancy,
							Kind:    rule.ViolationHardLimit,
							File:    "pkg/a/a_test.go",
							Line:    30,
							Subject: "TestRun",
							After:   1,
							Limit:   0,
							RedundantTests: []rule.RedundantDetail{
								{Kind: rule.RedundantSamePath, Line: 32, TestName: "sub1", DominatedBy: "sub2"},
							},
						},
						{
							Rule:    rule.RuleFuncCognitive,
							Kind:    rule.ViolationHardLimit,
							File:    "pkg/a/a.go",
							Line:    40,
							Subject: "Complex",
							After:   16,
							Limit:   15,
							Detail:  "too complex",
							Hotspots: []rule.Hotspot{
								{Line: 41, Delta: 1, Summary: "+1 (if)"},
							},
						},
					},
				},
				{
					Language: "Go",
					Advisory: true,
					Findings: []rule.Finding{
						{Rule: rule.RuleFuncCognitive, Kind: rule.ViolationHardLimit, File: "pkg/b/b.go", Line: 1, Subject: "Ignored"},
					},
				},
			},
			want: &rule.QAReport{
				BaseLabel: "main",
				Speed:     rule.SpeedSlow,
				Filter:    rule.FilterMust,
				MustFix: []rule.QAItem{
					{
						ID:      "codequality:directive:pkg/a/a.go:2:1",
						Source:  "codequality",
						Check:   "directive",
						File:    "pkg/a/a.go",
						Line:    2,
						Summary: "unused directive",
						Detail:  "unused directive",
					},
					{
						ID:           "decisions:r1/q1:pkg/a/a.go:5:1",
						Source:       "decisions",
						Check:        "r1/q1",
						File:         "pkg/a/a.go",
						Line:         5,
						Symbol:       "Decide",
						Summary:      "bad",
						Detail:       "fail",
						ReviewPrompt: "fix",
					},
					{
						ID:       "codequality:func_uncovered_lines:pkg/a/a.go:10:1",
						Source:   "codequality",
						Check:    "func_uncovered_lines",
						File:     "pkg/a/a.go",
						Line:     10,
						Symbol:   "Run",
						Summary:  "Run [HARD_LIMIT]: 2 (<= 0)",
						Detail:   "uncovered: L12, L15-L16",
						Evidence: map[string]any{"after": 2, "kind": "HARD_LIMIT", "limit": 0},
					},
					{
						ID:       "codequality:func_surviving_mutants:pkg/a/a.go:20:2",
						Source:   "codequality",
						Check:    "func_surviving_mutants",
						File:     "pkg/a/a.go",
						Line:     20,
						Symbol:   "Check",
						Summary:  "Check [HARD_LIMIT]: 1 (<= 0)",
						Detail:   "survived: L22 (ROR): a == b -> a != b",
						Evidence: map[string]any{"after": 1, "kind": "HARD_LIMIT", "limit": 0},
					},
					{
						ID:       "codequality:func_cognitive:pkg/a/a.go:40:4",
						Source:   "codequality",
						Check:    "func_cognitive",
						File:     "pkg/a/a.go",
						Line:     40,
						Symbol:   "Complex",
						Summary:  "too complex",
						Detail:   "too complex; hotspots: L41 +1 (if)",
						Evidence: map[string]any{"after": 16, "kind": "HARD_LIMIT", "limit": 15},
					},
					{
						ID:       "codequality:test_redundancy:pkg/a/a_test.go:30:3",
						Source:   "codequality",
						Check:    "test_redundancy",
						File:     "pkg/a/a_test.go",
						Line:     30,
						Symbol:   "TestRun",
						Summary:  "TestRun [HARD_LIMIT]: 1 (<= 0)",
						Detail:   "redundant: [SAME PATH] sub1 (L32) dominated by sub2",
						Evidence: map[string]any{"after": 1, "kind": "HARD_LIMIT", "limit": 0},
					},
				},
				Advisory:       []rule.QAItem{},
				SkippedSources: []rule.SkippedSource{},
			},
		},
		// Strips mustfix items when filter is FilterAdvisory while preserving advisory items.
		{
			name:      "FilterAdvisory drops mustfix items",
			baseLabel: "@-",
			speed:     rule.SpeedFast,
			filter:    rule.FilterAdvisory,
			reports: []*rule.Report{
				{
					Language: "Go",
					Findings: []rule.Finding{
						{Rule: rule.RuleFuncCognitive, Kind: rule.ViolationHardLimit, File: "pkg/a/a.go", Line: 5, Subject: "MustDrop"},
					},
					Similarity: rule.SimilaritySummary{
						Enabled: true,
						Advisories: []rule.SimilarityAdvisory{
							{
								NewUnit: rule.SimilarityUnitRef{FilePath: "pkg/a/a.go", StartLine: 10, Name: "KeepAdv"},
								Candidates: []rule.SimilarityCandidate{
									{
										Unit:     rule.SimilarityUnitRef{FilePath: "pkg/common/c.go", StartLine: 4, Name: "Shared"},
										Score:    0.925,
										Guidance: "reuse Shared",
									},
								},
							},
							{NewUnit: rule.SimilarityUnitRef{FilePath: "pkg/a/b.go", StartLine: 20, Name: "EmptyCand"}},
						},
					},
				},
			},
			want: &rule.QAReport{
				BaseLabel: "@-",
				Speed:     rule.SpeedFast,
				Filter:    rule.FilterAdvisory,
				MustFix:   []rule.QAItem{},
				Advisory: []rule.QAItem{
					{
						ID:           "similarity:pkg/a/a.go:10:1",
						Source:       "similarity",
						Check:        "function_similarity",
						File:         "pkg/a/a.go",
						Line:         10,
						Symbol:       "KeepAdv",
						Summary:      "similar to Shared (pkg/common/c.go:4, score 0.925)",
						ReviewPrompt: "reuse Shared",
					},
					{
						ID:      "similarity:pkg/a/b.go:20:2",
						Source:  "similarity",
						Check:   "function_similarity",
						File:    "pkg/a/b.go",
						Line:    20,
						Symbol:  "EmptyCand",
						Summary: "similar existing function found",
					},
				},
				SkippedSources: []rule.SkippedSource{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ConvertReportsToQAReport(tc.baseLabel, tc.speed, tc.filter, tc.reports)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ConvertReportsToQAReport() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRenderQATextAndGitHubEdgeCases(t *testing.T) {
	testCases := []struct {
		name             string
		report           *rule.QAReport
		wantTextTokens   []string
		forbidTextTokens []string
		wantGHTokens     []string
		forbidGHTokens   []string
	}{
		// Emits mustfix error annotation without clean summary when only MustFix is populated.
		{
			name: "mustfix-only report omits clean GitHub notice",
			report: &rule.QAReport{
				BaseLabel: "main",
				Speed:     rule.SpeedFast,
				Filter:    rule.FilterMust,
				MustFix: []rule.QAItem{
					{ID: "m1", Source: "gofmt", File: "pkg/a/a.go", Line: 1, Summary: "unformatted"},
				},
			},
			wantTextTokens:   []string{"MUSTFIX (1)", "unformatted"},
			forbidTextTokens: []string{"ADVISORY", "Skipped Sources"},
			wantGHTokens:     []string{"::error file=pkg/a/a.go,line=1,title=MUSTFIX (gofmt)::unformatted"},
			forbidGHTokens:   []string{"::notice title=KHI QA"},
		},
		// Emits skipped notice without clean summary when only SkippedSources is populated.
		{
			name: "skipped-only report omits clean GitHub notice",
			report: &rule.QAReport{
				BaseLabel:      "main",
				Speed:          rule.SpeedSlow,
				Filter:         rule.FilterAll,
				SkippedSources: []rule.SkippedSource{{Source: "similarity", Reason: "no project"}},
			},
			wantTextTokens:   []string{"Skipped Sources (1)", "similarity: no project"},
			forbidTextTokens: []string{"MUSTFIX", "ADVISORY"},
			wantGHTokens:     []string{"::notice title=Skipped (similarity)::no project"},
			forbidGHTokens:   []string{"::notice title=KHI QA"},
		},
		// Renders items with empty Symbol, empty Check, empty Source, duplicate Detail, and ReviewPrompt without panicking.
		{
			name: "renders optional fields and partial labels",
			report: &rule.QAReport{
				BaseLabel: "main",
				Speed:     rule.SpeedAll,
				Filter:    rule.FilterAll,
				MustFix: []rule.QAItem{
					{
						ID:           "m1",
						Source:       "gofmt",
						Check:        "",
						File:         "pkg/a/a.go",
						Line:         1,
						Symbol:       "",
						Summary:      "file is not gofmt-formatted",
						Detail:       "run gofmt -w pkg/a/a.go",
						ReviewPrompt: "Format with gofmt.",
					},
					{
						ID:      "m2",
						Source:  "astcheck",
						Check:   "func_cognitive",
						File:    "pkg/a/b.go",
						Line:    10,
						Symbol:  "Proc",
						Summary: "same message",
						Detail:  "same message",
					},
				},
				Advisory: []rule.QAItem{
					{
						ID:      "a1",
						Source:  "",
						Check:   "doc_check",
						File:    "pkg/b/b.go",
						Line:    5,
						Symbol:  "Helper",
						Summary: "missing doc",
					},
				},
			},
			wantTextTokens: []string{
				"[gofmt] pkg/a/a.go:1 — file is not gofmt-formatted",
				"Detail: run gofmt -w pkg/a/a.go",
				"Prompt: Format with gofmt.",
				"[astcheck/func_cognitive] pkg/a/b.go:10 (Proc) — same message",
				"[doc_check] pkg/b/b.go:5 (Helper) — missing doc",
			},
			forbidTextTokens: []string{
				"Detail: same message",
			},
			wantGHTokens: []string{
				"::error file=pkg/a/a.go,line=1,title=MUSTFIX (gofmt)::file is not gofmt-formatted | Fix: Format with gofmt.",
				"::error file=pkg/a/b.go,line=10,title=MUSTFIX (astcheck/func_cognitive)::Proc: same message",
				"::warning file=pkg/b/b.go,line=5,title=ADVISORY (doc_check)::Helper: missing doc",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var textBuf bytes.Buffer
			var ghBuf bytes.Buffer

			RenderQAText(&textBuf, tc.report, gotestutil.TermStyle{})
			RenderQAGitHub(&ghBuf, tc.report)

			assertRenderedTokens(t, "RenderQAText", textBuf.String(), tc.wantTextTokens, tc.forbidTextTokens)
			assertRenderedTokens(t, "RenderQAGitHub", ghBuf.String(), tc.wantGHTokens, tc.forbidGHTokens)
		})
	}
}

func assertRenderedTokens(t *testing.T, label string, got string, want []string, forbid []string) {
	t.Helper()
	for _, tok := range want {
		if !strings.Contains(got, tok) {
			t.Errorf("%s() missing %q in:\n%s", label, tok, got)
		}
	}
	for _, tok := range forbid {
		if strings.Contains(got, tok) {
			t.Errorf("%s() unexpectedly contained %q in:\n%s", label, tok, got)
		}
	}
}
