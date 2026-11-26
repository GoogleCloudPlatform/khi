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
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type convertedItemSnapshot struct {
	HasID      bool
	HasSource  bool
	HasSummary bool
	File       string
	Line       int
	Symbol     string
}

type convertedQASnapshot struct {
	BaseLabel      string
	Speed          rule.Speed
	Filter         rule.Filter
	HasMustFix     bool
	MustFixNonNil  bool
	AdvisoryNonNil bool
	SkippedNonNil  bool
	MustFix        []convertedItemSnapshot
	Advisory       []convertedItemSnapshot
	SkippedSources []rule.SkippedSource
}

func TestRenderQAJSON(t *testing.T) {
	testCases := []struct {
		name          string
		report        *rule.QAReport
		wantTopKeys   []string
		wantRoundTrip rule.QAReport
	}{
		// Emits a single JSON object with exact schema keys and empty arrays instead of null when report is nil.
		{
			name:        "nil report serializes as empty JSON arrays",
			report:      nil,
			wantTopKeys: []string{"advisory", "base_label", "filter", "mustfix", "skipped_sources", "speed"},
			wantRoundTrip: rule.QAReport{
				MustFix:        []rule.QAItem{},
				Advisory:       []rule.QAItem{},
				SkippedSources: []rule.SkippedSource{},
			},
		},
		// Serializes populated mustfix, advisory, and skipped_sources items accurately with snake_case JSON keys.
		{
			name: "populated report serializes all sections",
			report: &rule.QAReport{
				BaseLabel: "main",
				Speed:     rule.SpeedAll,
				Filter:    rule.FilterAll,
				MustFix: []rule.QAItem{
					{
						ID:           "must-1",
						Source:       "astcheck",
						Check:        "func_cognitive",
						File:         "pkg/a/a.go",
						Line:         10,
						Symbol:       "Process",
						Summary:      "cognitive complexity exceeded",
						Detail:       "complexity 18 > 15",
						Evidence:     map[string]any{"after": float64(18)},
						ReviewPrompt: "Reduce branching in Process.",
					},
				},
				Advisory: []rule.QAItem{
					{
						ID:      "adv-1",
						Source:  "decisions",
						Check:   "go_code_quality",
						File:    "pkg/b/b.go",
						Line:    22,
						EndLine: 30,
						Symbol:  "Helper",
						Summary: "comment mismatch",
					},
				},
				SkippedSources: []rule.SkippedSource{
					{
						Source: "go_test_short",
						Reason: "compilation blocked outside target scope: pkg/common/timeseries.go:42",
					},
				},
			},
			wantTopKeys: []string{"advisory", "base_label", "filter", "mustfix", "skipped_sources", "speed"},
			wantRoundTrip: rule.QAReport{
				BaseLabel: "main",
				Speed:     rule.SpeedAll,
				Filter:    rule.FilterAll,
				MustFix: []rule.QAItem{
					{
						ID:           "must-1",
						Source:       "astcheck",
						Check:        "func_cognitive",
						File:         "pkg/a/a.go",
						Line:         10,
						Symbol:       "Process",
						Summary:      "cognitive complexity exceeded",
						Detail:       "complexity 18 > 15",
						Evidence:     map[string]any{"after": float64(18)},
						ReviewPrompt: "Reduce branching in Process.",
					},
				},
				Advisory: []rule.QAItem{
					{
						ID:      "adv-1",
						Source:  "decisions",
						Check:   "go_code_quality",
						File:    "pkg/b/b.go",
						Line:    22,
						EndLine: 30,
						Symbol:  "Helper",
						Summary: "comment mismatch",
					},
				},
				SkippedSources: []rule.SkippedSource{
					{
						Source: "go_test_short",
						Reason: "compilation blocked outside target scope: pkg/common/timeseries.go:42",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := RenderQAJSON(&buf, tc.report)

			if err != nil {
				t.Fatalf("RenderQAJSON() unexpected error: %v", err)
			}
			assertSingleJSONSchemaAndRoundTrip(t, buf.Bytes(), tc.wantTopKeys, tc.wantRoundTrip)
		})
	}
}

func TestConvertReportsToQAReport(t *testing.T) {
	testCases := []struct {
		name      string
		baseLabel string
		speed     rule.Speed
		filter    rule.Filter
		reports   []*rule.Report
		want      convertedQASnapshot
	}{
		// Normalizes nil reports into an initialized empty QAReport with empty non-nil slices.
		{
			name:      "nil reports slice",
			baseLabel: "main",
			speed:     rule.SpeedFast,
			filter:    rule.FilterAll,
			reports:   nil,
			want: convertedQASnapshot{
				BaseLabel:      "main",
				Speed:          rule.SpeedFast,
				Filter:         rule.FilterAll,
				HasMustFix:     false,
				MustFixNonNil:  true,
				AdvisoryNonNil: true,
				SkippedNonNil:  true,
				MustFix:        []convertedItemSnapshot{},
				Advisory:       []convertedItemSnapshot{},
				SkippedSources: []rule.SkippedSource{},
			},
		},
		// Maps blocking Go findings and directive errors to MustFix while keeping Advisory empty.
		{
			name:      "Go blocking findings and directive errors map to MustFix",
			baseLabel: "@-",
			speed:     rule.SpeedFast,
			filter:    rule.FilterMust,
			reports: []*rule.Report{
				{
					Language: "Go",
					DirectiveErrors: []rule.DirectiveError{
						{File: "pkg/a/a.go", Line: 3, Message: "unused directive"},
					},
					Findings: []rule.Finding{
						{
							Rule:    rule.RuleFuncCognitive,
							Kind:    rule.ViolationHardLimit,
							File:    "pkg/a/a.go",
							Line:    10,
							Subject: "Process",
							After:   18,
							Limit:   15,
							Detail:  "too complex",
						},
					},
				},
			},
			want: convertedQASnapshot{
				BaseLabel:      "@-",
				Speed:          rule.SpeedFast,
				Filter:         rule.FilterMust,
				HasMustFix:     true,
				MustFixNonNil:  true,
				AdvisoryNonNil: true,
				SkippedNonNil:  true,
				MustFix: []convertedItemSnapshot{
					{HasID: true, HasSource: true, HasSummary: true, File: "pkg/a/a.go", Line: 3},
					{HasID: true, HasSource: true, HasSummary: true, File: "pkg/a/a.go", Line: 10, Symbol: "Process"},
				},
				Advisory:       []convertedItemSnapshot{},
				SkippedSources: []rule.SkippedSource{},
			},
		},
		// Maps non-blocking decisions, similarity advisories, and advisory TypeScript findings to Advisory.
		{
			name:      "non-blocking decisions similarity and advisory TS map to Advisory",
			baseLabel: "main",
			speed:     rule.SpeedAll,
			filter:    rule.FilterAll,
			reports: []*rule.Report{
				{
					Language: "Decisions",
					Decisions: rule.DecisionSummary{
						Enabled: true,
						Findings: []rule.DecisionItemFinding{
							{
								RuleID:     "go_code_quality",
								QuestionID: "comment_logic_alignment",
								Blocking:   false,
								File:       "pkg/b/b.go",
								Line:       5,
								Subject:    "Helper",
								Outcome:    "stale",
								Why:        "comment stale",
								Fix:        "update comment",
							},
						},
					},
				},
				{
					Language: "Go",
					Similarity: rule.SimilaritySummary{
						Enabled: true,
						Advisories: []rule.SimilarityAdvisory{
							{
								NewUnit: rule.SimilarityUnitRef{FilePath: "pkg/c/c.go", StartLine: 20, EndLine: 30, Name: "DupHelper"},
								Candidates: []rule.SimilarityCandidate{
									{
										Unit:     rule.SimilarityUnitRef{FilePath: "pkg/common/h.go", StartLine: 5, Name: "SharedHelper"},
										Score:    0.95,
										Guidance: "reuse SharedHelper",
									},
								},
							},
						},
					},
				},
				{
					Language: "TypeScript",
					Advisory: true,
					Findings: []rule.Finding{
						{
							Rule:    rule.RuleFuncUncoveredLines,
							Kind:    rule.ViolationHardLimit,
							File:    "web/src/a.ts",
							Line:    8,
							Subject: "render",
							After:   1,
							Limit:   0,
						},
					},
				},
			},
			want: convertedQASnapshot{
				BaseLabel:      "main",
				Speed:          rule.SpeedAll,
				Filter:         rule.FilterAll,
				HasMustFix:     false,
				MustFixNonNil:  true,
				AdvisoryNonNil: true,
				SkippedNonNil:  true,
				MustFix:        []convertedItemSnapshot{},
				Advisory: []convertedItemSnapshot{
					{HasID: true, HasSource: true, HasSummary: true, File: "pkg/b/b.go", Line: 5, Symbol: "Helper"},
					{HasID: true, HasSource: true, HasSummary: true, File: "pkg/c/c.go", Line: 20, Symbol: "DupHelper"},
					{HasID: true, HasSource: true, HasSummary: true, File: "web/src/a.ts", Line: 8, Symbol: "render"},
				},
				SkippedSources: []rule.SkippedSource{},
			},
		},
		// Maps skipped Decisions and Similarity evaluations into SkippedSources with their skip reasons.
		{
			name:      "skipped decisions and similarity map to SkippedSources",
			baseLabel: "main",
			speed:     rule.SpeedAll,
			filter:    rule.FilterAdvisory,
			reports: []*rule.Report{
				{
					Language: "Decisions",
					Decisions: rule.DecisionSummary{
						Enabled:    true,
						Skipped:    true,
						SkipReason: "missing API key",
					},
				},
				{
					Language: "Go",
					Similarity: rule.SimilaritySummary{
						Enabled:    true,
						Skipped:    true,
						SkipReason: "missing similarity project",
					},
				},
			},
			want: convertedQASnapshot{
				BaseLabel:      "main",
				Speed:          rule.SpeedAll,
				Filter:         rule.FilterAdvisory,
				HasMustFix:     false,
				MustFixNonNil:  true,
				AdvisoryNonNil: true,
				SkippedNonNil:  true,
				MustFix:        []convertedItemSnapshot{},
				Advisory:       []convertedItemSnapshot{},
				SkippedSources: []rule.SkippedSource{
					{Source: "decisions", Reason: "missing API key"},
					{Source: "similarity", Reason: "missing similarity project"},
				},
			},
		},
		// Routes blocking DecisionItemFindings to MustFix while keeping non-blocking ones in Advisory.
		{
			name:      "splits blocking and non-blocking decisions findings",
			baseLabel: "main",
			speed:     rule.SpeedFast,
			filter:    rule.FilterAll,
			reports: []*rule.Report{
				{
					Language: "Decisions",
					Decisions: rule.DecisionSummary{
						Enabled: true,
						Findings: []rule.DecisionItemFinding{
							{RuleID: "r1", QuestionID: "q1", Blocking: true, File: "pkg/d/d.go", Line: 12, Subject: "MustItem", Outcome: "fail", Why: "bad", Fix: "fix"},
							{RuleID: "r1", QuestionID: "q2", Blocking: false, File: "pkg/d/d.go", Line: 25, Subject: "AdvItem", Outcome: "warn", Why: "hint", Fix: "consider"},
						},
					},
				},
			},
			want: convertedQASnapshot{
				BaseLabel:      "main",
				Speed:          rule.SpeedFast,
				Filter:         rule.FilterAll,
				HasMustFix:     true,
				MustFixNonNil:  true,
				AdvisoryNonNil: true,
				SkippedNonNil:  true,
				MustFix: []convertedItemSnapshot{
					{HasID: true, HasSource: true, HasSummary: true, File: "pkg/d/d.go", Line: 12, Symbol: "MustItem"},
				},
				Advisory: []convertedItemSnapshot{
					{HasID: true, HasSource: true, HasSummary: true, File: "pkg/d/d.go", Line: 25, Symbol: "AdvItem"},
				},
				SkippedSources: []rule.SkippedSource{},
			},
		},
		// Ignores nil entries inside reports without panicking.
		{
			name:      "nil report elements are ignored",
			baseLabel: "main",
			speed:     rule.SpeedSlow,
			filter:    rule.FilterAll,
			reports:   []*rule.Report{nil},
			want: convertedQASnapshot{
				BaseLabel:      "main",
				Speed:          rule.SpeedSlow,
				Filter:         rule.FilterAll,
				HasMustFix:     false,
				MustFixNonNil:  true,
				AdvisoryNonNil: true,
				SkippedNonNil:  true,
				MustFix:        []convertedItemSnapshot{},
				Advisory:       []convertedItemSnapshot{},
				SkippedSources: []rule.SkippedSource{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotReport := ConvertReportsToQAReport(tc.baseLabel, tc.speed, tc.filter, tc.reports)

			got := snapshotQAReport(gotReport)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ConvertReportsToQAReport() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRenderQATextAndGitHub(t *testing.T) {
	testCases := []struct {
		name            string
		report          *rule.QAReport
		wantSubstrs     []string
		wantFoldSubstrs []string
		forbidSubstrs   []string
	}{
		// Handles a nil QAReport without panicking in both text and GitHub renderers.
		{
			name:   "nil report does not panic",
			report: nil,
		},
		// Handles an empty clean QAReport without panicking in both text and GitHub renderers.
		{
			name: "empty report renders cleanly",
			report: &rule.QAReport{
				BaseLabel: "main",
				Speed:     rule.SpeedFast,
				Filter:    rule.FilterAll,
			},
			wantSubstrs:   []string{"main"},
			forbidSubstrs: []string{"MUSTFIX", "ADVISORY", "Skipped Sources"},
		},
		// Renders MUSTFIX, ADVISORY, and skipped source entries clearly in both text and GitHub outputs.
		{
			name: "renders MUSTFIX, ADVISORY, and skipped sources",
			report: &rule.QAReport{
				BaseLabel: "@- (fe749d34)",
				Speed:     rule.SpeedFast,
				Filter:    rule.FilterAll,
				MustFix: []rule.QAItem{
					{
						ID:      "must-1",
						Source:  "astcheck",
						Check:   "func_cognitive",
						File:    "pkg/a/a.go",
						Line:    10,
						Symbol:  "Process",
						Summary: "cognitive complexity exceeded",
					},
				},
				Advisory: []rule.QAItem{
					{
						ID:      "adv-1",
						Source:  "decisions",
						Check:   "go_code_quality",
						File:    "pkg/b/b.go",
						Line:    22,
						EndLine: 30,
						Symbol:  "Helper",
						Summary: "comment mismatch",
					},
				},
				SkippedSources: []rule.SkippedSource{
					{
						Source: "go_test_short",
						Reason: "compilation blocked outside scope",
					},
				},
			},
			wantSubstrs: []string{
				"MUSTFIX",
				"ADVISORY",
				"pkg/a/a.go",
				"cognitive complexity exceeded",
				"pkg/b/b.go",
				"comment mismatch",
				"go_test_short",
				"compilation blocked outside scope",
			},
			wantFoldSubstrs: []string{"skipped"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var textBuf bytes.Buffer
			var ghBuf bytes.Buffer

			RenderQAText(&textBuf, tc.report, gotestutil.TermStyle{})
			RenderQAGitHub(&ghBuf, tc.report)

			if missing := missingSubstrings(textBuf.String(), tc.wantSubstrs, tc.wantFoldSubstrs); len(missing) > 0 {
				t.Errorf("RenderQAText() missing tokens %v in:\n%s", missing, textBuf.String())
			}
			for _, tok := range tc.forbidSubstrs {
				if strings.Contains(textBuf.String(), tok) {
					t.Errorf("RenderQAText() unexpectedly contained %q in:\n%s", tok, textBuf.String())
				}
			}
			if missing := missingSubstrings(ghBuf.String(), tc.wantSubstrs, tc.wantFoldSubstrs); len(missing) > 0 {
				t.Errorf("RenderQAGitHub() missing tokens %v in:\n%s", missing, ghBuf.String())
			}
		})
	}
}

func assertSingleJSONSchemaAndRoundTrip(t *testing.T, payload []byte, wantTopKeys []string, wantReport rule.QAReport) {
	t.Helper()
	var rawMap map[string]any
	if err := json.Unmarshal(payload, &rawMap); err != nil {
		t.Fatalf("json.Unmarshal(map) error: %v\npayload:\n%s", err, string(payload))
	}
	gotKeys := slices.Sorted(maps.Keys(rawMap))
	if diff := cmp.Diff(wantTopKeys, gotKeys); diff != "" {
		t.Errorf("JSON top-level keys mismatch (-want +got):\n%s", diff)
	}
	for _, arrayKey := range []string{"mustfix", "advisory", "skipped_sources"} {
		if _, ok := rawMap[arrayKey].([]any); !ok {
			t.Errorf("JSON field %q = %T (%v), want JSON array [] (not null)", arrayKey, rawMap[arrayKey], rawMap[arrayKey])
		}
	}

	var gotReport rule.QAReport
	if err := json.Unmarshal(payload, &gotReport); err != nil {
		t.Fatalf("json.Unmarshal(QAReport) error: %v", err)
	}
	if diff := cmp.Diff(wantReport, gotReport); diff != "" {
		t.Errorf("RenderQAJSON() round-trip mismatch (-want +got):\n%s", diff)
	}
	assertNestedItemKeys(t, rawMap)
}

func assertNestedItemKeys(t *testing.T, rawMap map[string]any) {
	t.Helper()
	assertFirstMapKeys(t, rawMap["mustfix"], []string{"check", "detail", "evidence", "file", "id", "line", "review_prompt", "source", "summary", "symbol"}, "mustfix[0]")
	assertFirstMapKeys(t, rawMap["advisory"], []string{"check", "detail", "end_line", "evidence", "file", "id", "line", "review_prompt", "source", "summary", "symbol"}, "advisory[0]")
	assertFirstMapKeys(t, rawMap["skipped_sources"], []string{"reason", "source"}, "skipped_sources[0]")
}

func assertFirstMapKeys(t *testing.T, rawSlice any, wantKeys []string, label string) {
	t.Helper()
	items, ok := rawSlice.([]any)
	if !ok || len(items) == 0 {
		return
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("%s = %T, want JSON object", label, items[0])
	}
	if diff := cmp.Diff(wantKeys, slices.Sorted(maps.Keys(first))); diff != "" {
		t.Errorf("%s JSON keys mismatch (-want +got):\n%s", label, diff)
	}
}

func snapshotQAReport(rep *rule.QAReport) convertedQASnapshot {
	if rep == nil {
		return convertedQASnapshot{}
	}
	return convertedQASnapshot{
		BaseLabel:      rep.BaseLabel,
		Speed:          rep.Speed,
		Filter:         rep.Filter,
		HasMustFix:     rep.HasMustFix(),
		MustFixNonNil:  rep.MustFix != nil,
		AdvisoryNonNil: rep.Advisory != nil,
		SkippedNonNil:  rep.SkippedSources != nil,
		MustFix:        snapshotQAItems(rep.MustFix),
		Advisory:       snapshotQAItems(rep.Advisory),
		SkippedSources: snapshotSkippedSources(rep.SkippedSources),
	}
}

func snapshotQAItems(items []rule.QAItem) []convertedItemSnapshot {
	out := make([]convertedItemSnapshot, 0, len(items))
	for _, item := range items {
		out = append(out, convertedItemSnapshot{
			HasID:      item.ID != "",
			HasSource:  item.Source != "",
			HasSummary: item.Summary != "",
			File:       item.File,
			Line:       item.Line,
			Symbol:     item.Symbol,
		})
	}
	return out
}

func snapshotSkippedSources(skipped []rule.SkippedSource) []rule.SkippedSource {
	out := make([]rule.SkippedSource, 0, len(skipped))
	out = append(out, skipped...)
	return out
}

func missingSubstrings(rendered string, exact []string, fold []string) []string {
	var missing []string
	for _, token := range exact {
		if !strings.Contains(rendered, token) {
			missing = append(missing, token)
		}
	}
	lower := strings.ToLower(rendered)
	for _, token := range fold {
		if !strings.Contains(lower, strings.ToLower(token)) {
			missing = append(missing, token)
		}
	}
	return missing
}
