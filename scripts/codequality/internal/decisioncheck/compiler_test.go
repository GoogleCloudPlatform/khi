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

package decisioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestCompileQuestionsAndEvaluateBatchResolvesOutcomeSpecificFixHints verifies template expansion and per-option/per-level fix_hint resolution.
func TestCompileQuestionsAndEvaluateBatchResolvesOutcomeSpecificFixHints(t *testing.T) {
	yamlText := `
id: pkg_doc_alignment
name: Package Doc Alignment
enforcement: advisory
target:
  kind: go_logical_package
questions:
  - id: overall
    type: score
    instructions: "Rate {{.Target}}"
    levels:
      - description: "Major drift in {{.Target}}"
        fix_hint: "Rewrite {{.File}} from scratch."
      - description: "Minor drift in {{.Target}}"
        fix_hint: "Patch {{.File}} sections."
      - description: "Aligned"
    pass:
      min_score: 1.5
  - id: sec
    for_each: markdown_sections
    type: choice
    instructions: "Check {{.Heading}}"
    options:
      aligned:
        description: "Aligned"
      stale_symbol:
        description: "Symbol in {{.Heading}} no longer exists."
        fix_hint: "Remove stale symbol from {{.File}}:L{{.StartLine}}-L{{.EndLine}}."
      contradiction:
        description: "Behavior in {{.Heading}} contradicts code."
        fix_hint: "Fix contradicted behavior in {{.Heading}}."
    pass:
      expected_choice: aligned
      min_probability: 0.60
  - id: unit
    for_each: go_substantive_units
    type: noul
    instructions: "Is {{.Name}} covered?"
    criteria:
      true_description: "Covered"
      false_description: "{{.Name}} is undocumented."
    fix_hint: "Document {{.Name}} ({{.File}}:{{.Line}}) in README.md."
    pass:
      expect: true
      min_probability: 0.50
`
	spec, err := ParseRuleYAML([]byte(yamlText))
	if err != nil {
		t.Fatalf("ParseRuleYAML() error: %v", err)
	}

	bundle := TargetBundle{
		ID:          "pkg/demo",
		PrimaryFile: "pkg/demo/README.md",
		Files: []BundledFile{
			{RelPath: "pkg/demo/README.md", Content: []byte("# Title\n\nIntro.\n\n## Details\n\nBody.\n")},
		},
		SubstantiveGoUnits: []ExtractedItem{
			{Target: "pkg/demo", File: "pkg/demo/impl/module.go", Line: 12, StartLine: 12, EndLine: 12, Name: "secretTask", Kind: "registered DAG task"},
		},
	}

	batches, err := CompileQuestions("gemini-decision-1", "shared-ctx", spec, bundle)
	if err != nil {
		t.Fatalf("CompileQuestions() error: %v", err)
	}
	if len(batches) != 1 || len(batches[0].Order) != 4 {
		t.Fatalf("got %d batches / %d questions, want 1 batch / 4 questions", len(batches), len(batches[0].Order))
	}

	scoreVal := 0.85
	noulFail := 0.12
	resp := DecisionResponse{
		Answers: map[string]WireAnswer{
			"overall": {
				Score:         &scoreVal,
				Probabilities: map[string]float64{"0": 0.20, "1": 0.75, "2": 0.05},
				Confidence:    0.80,
			},
			"sec_0": {
				Choice:        "aligned",
				Probabilities: map[string]float64{"aligned": 0.92, "stale_symbol": 0.05, "contradiction": 0.03},
				Confidence:    0.92,
			},
			"sec_1": {
				Choice:        "stale_symbol",
				Probabilities: map[string]float64{"aligned": 0.10, "stale_symbol": 0.82, "contradiction": 0.08},
				Confidence:    0.88,
			},
			"unit_0": {
				Noul:       &noulFail,
				Confidence: 0.94,
			},
		},
	}

	got, err := EvaluateBatch(spec, false, batches[0], resp)
	if err != nil {
		t.Fatalf("EvaluateBatch() error: %v", err)
	}

	want := []rule.DecisionItemFinding{
		{
			RuleID:     "pkg_doc_alignment",
			RuleName:   "Package Doc Alignment",
			QuestionID: "overall",
			Blocking:   false,
			Target:     "pkg/demo",
			File:       "pkg/demo/README.md",
			Line:       1,
			EndLine:    1,
			Subject:    "pkg/demo",
			Outcome:    "score=0.85/2 (< 1.50, conf=0.80)",
			Why:        "Minor drift in pkg/demo",
			Fix:        "Patch pkg/demo/README.md sections.",
		},
		{
			RuleID:     "pkg_doc_alignment",
			RuleName:   "Package Doc Alignment",
			QuestionID: "sec",
			Blocking:   false,
			Target:     "pkg/demo",
			File:       "pkg/demo/README.md",
			Line:       5,
			EndLine:    7,
			Subject:    "Details",
			Outcome:    "stale_symbol (p=0.82, pass_p=0.10, conf=0.88)",
			Why:        "Symbol in Details no longer exists.",
			Fix:        "Remove stale symbol from pkg/demo/README.md:L5-L7.",
		},
		{
			RuleID:     "pkg_doc_alignment",
			RuleName:   "Package Doc Alignment",
			QuestionID: "unit",
			Blocking:   false,
			Target:     "pkg/demo",
			File:       "pkg/demo/impl/module.go",
			Line:       12,
			EndLine:    12,
			Subject:    "secretTask",
			Outcome:    "p_true=0.12 (want true >= 0.50, conf=0.94)",
			Why:        "secretTask is undocumented.",
			Fix:        "Document secretTask (pkg/demo/impl/module.go:12) in README.md.",
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("EvaluateBatch() mismatch (-want +got):\n%s", diff)
	}
}

// TestCompileQuestions verifies question expansion, wire criteria rendering, and batch payload construction across extractors.
func TestCompileQuestions(t *testing.T) {
	baseBundle := TargetBundle{
		ID:          "pkg/demo",
		PrimaryFile: "pkg/demo/code.go",
		Files: []BundledFile{
			{
				RelPath: "pkg/demo/code.go",
				Content: []byte(`package demo

type Engine struct{}

func Run() {}
`),
			},
			{
				RelPath: "pkg/demo/code_test.go",
				Content: []byte(`package demo

import "testing"

func TestRun(t *testing.T) {}
`),
			},
			{
				RelPath: "pkg/demo/notes.txt",
				Content: []byte(`# Ignored heading

Plain text body.
`),
			},
		},
	}
	testCases := []struct {
		name   string
		rule   RuleSpec
		bundle TargetBundle
		want   []CompiledBatch
	}{
		// Returns nil batches when markdown_sections finds no .md files even if non-Markdown files contain headings.
		{
			name: "skips non-markdown files and returns nil when no items match",
			rule: RuleSpec{
				ID:   "md_rule",
				Name: "MD Rule",
				Questions: []QuestionSpec{{
					ID: "sec", ForEach: ExtractorMarkdownSections, Type: QuestionTypeNoul,
					Instructions: "Check {{.Heading}}", FixHint: "Fix",
					Pass: PassSpec{Expect: ptr(true), MinProbability: 0.5},
				}},
			},
			bundle: baseBundle,
			want:   nil,
		},
		// Compiles a bundle-level noul question with empty criteria descriptions into a wire question with nil Criteria.
		{
			name: "bundle extractor with empty noul criteria omits criteria map",
			rule: RuleSpec{
				ID:   "bundle_rule",
				Name: "Bundle Rule",
				Questions: []QuestionSpec{{
					ID: "q_bundle", ForEach: ExtractorBundle, Type: QuestionTypeNoul,
					Instructions: "Check {{.Target}}", FixHint: "Fix",
					Pass: PassSpec{Expect: ptr(true), MinProbability: 0.5},
				}},
			},
			bundle: baseBundle,
			want: []CompiledBatch{{
				Request: DecisionRequest{
					Model: "gemini-decision-1",
					Input: "shared-ctx",
					Questions: map[string]WireQuestion{
						"q_bundle": {Type: QuestionTypeNoul, Instructions: "Check pkg/demo", Criteria: nil},
					},
				},
				Order: []string{"q_bundle"},
			}},
		},
		// Expands changed_go_decls and renders only true_description when false_description is omitted.
		{
			name: "changed_go_decls extractor renders noul true_description only",
			rule: RuleSpec{
				ID:   "decl_rule",
				Name: "Decl Rule",
				Questions: []QuestionSpec{{
					ID: "q_decl", ForEach: ExtractorChangedGoDecls, Type: QuestionTypeNoul,
					Instructions: "Inspect {{.Kind}} {{.Name}}",
					Criteria:     NoulCriteriaSpec{TrueDescription: "Valid {{.Name}}"},
					FixHint:      "Fix",
					Pass:         PassSpec{Expect: ptr(true), MinProbability: 0.5},
				}},
			},
			bundle: baseBundle,
			want: []CompiledBatch{{
				Request: DecisionRequest{
					Model: "gemini-decision-1",
					Input: "shared-ctx",
					Questions: map[string]WireQuestion{
						"q_decl_0": {Type: QuestionTypeNoul, Instructions: "Inspect type Engine", Criteria: map[string]string{"true_description": "Valid Engine"}},
						"q_decl_1": {Type: QuestionTypeNoul, Instructions: "Inspect func Run", Criteria: map[string]string{"true_description": "Valid Run"}},
					},
				},
				Order: []string{"q_decl_0", "q_decl_1"},
			}},
		},
		// Expands changed_go_funcs and renders only false_description when true_description is omitted.
		{
			name: "changed_go_funcs extractor renders noul false_description only",
			rule: RuleSpec{
				ID:   "func_rule",
				Name: "Func Rule",
				Questions: []QuestionSpec{{
					ID: "q_func", ForEach: ExtractorChangedGoFuncs, Type: QuestionTypeNoul,
					Instructions: "Inspect func {{.Name}}",
					Criteria:     NoulCriteriaSpec{FalseDescription: "Invalid {{.Name}}"},
					FixHint:      "Fix",
					Pass:         PassSpec{Expect: ptr(true), MinProbability: 0.5},
				}},
			},
			bundle: baseBundle,
			want: []CompiledBatch{{
				Request: DecisionRequest{
					Model: "gemini-decision-1",
					Input: "shared-ctx",
					Questions: map[string]WireQuestion{
						"q_func_0": {Type: QuestionTypeNoul, Instructions: "Inspect func Run", Criteria: map[string]string{"false_description": "Invalid Run"}},
					},
				},
				Order: []string{"q_func_0"},
			}},
		},
		// Expands changed_go_test_funcs and renders choice option criteria descriptions.
		{
			name: "changed_go_test_funcs extractor renders choice criteria",
			rule: RuleSpec{
				ID:   "test_func_rule",
				Name: "Test Func Rule",
				Questions: []QuestionSpec{{
					ID: "q_tf", ForEach: ExtractorChangedGoTestFuncs, Type: QuestionTypeChoice,
					Instructions: "Check {{.Name}}",
					Options: map[string]ChoiceOptionSpec{
						"ok":  {Description: "{{.Name}} good"},
						"bad": {Description: "{{.Name}} bad", FixHint: "Fix"},
					},
					Pass: PassSpec{ExpectedChoice: "ok", MinProbability: 0.5},
				}},
			},
			bundle: baseBundle,
			want: []CompiledBatch{{
				Request: DecisionRequest{
					Model: "gemini-decision-1",
					Input: "shared-ctx",
					Questions: map[string]WireQuestion{
						"q_tf_0": {Type: QuestionTypeChoice, Instructions: "Check TestRun", Criteria: map[string]string{"ok": "TestRun good", "bad": "TestRun bad"}},
					},
				},
				Order: []string{"q_tf_0"},
			}},
		},
		// Expands changed_go_test_cases and renders ordered score level criteria descriptions.
		{
			name: "changed_go_test_cases extractor renders score criteria",
			rule: RuleSpec{
				ID:   "test_case_rule",
				Name: "Test Case Rule",
				Questions: []QuestionSpec{{
					ID: "q_tc", ForEach: ExtractorChangedGoTestCases, Type: QuestionTypeScore,
					Instructions: "Score {{.Name}}",
					Levels:       []ScoreLevelSpec{{Description: "{{.Name}} low", FixHint: "Fix"}, {Description: "{{.Name}} high"}},
					Pass:         PassSpec{MinScore: ptr(1.0)},
				}},
			},
			bundle: baseBundle,
			want: []CompiledBatch{{
				Request: DecisionRequest{
					Model: "gemini-decision-1",
					Input: "shared-ctx",
					Questions: map[string]WireQuestion{
						"q_tc_0": {Type: QuestionTypeScore, Instructions: "Score TestRun", Criteria: []string{"TestRun low", "TestRun high"}},
					},
				},
				Order: []string{"q_tc_0"},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CompileQuestions("gemini-decision-1", "shared-ctx", tc.rule, tc.bundle)

			if err != nil {
				t.Fatalf("CompileQuestions() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreFields(CompiledBatch{}, "Meta")); diff != "" {
				t.Errorf("CompileQuestions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestCompileQuestionsTypeScript verifies question expansion across TypeScript declaration, function, and spec AST extractors when TypeScript is available.
func TestCompileQuestionsTypeScript(t *testing.T) {
	webDir, err := gotestutil.ResolveWebDir("")
	if err != nil {
		t.Skipf("skipping TypeScript compiler test: could not locate web directory (%v)", err)
	}
	if _, statErr := os.Stat(filepath.Join(webDir, "node_modules", "typescript")); statErr != nil {
		t.Skipf("skipping TypeScript compiler test: web/node_modules/typescript not installed (%v)", statErr)
	}

	tsBundle := TargetBundle{
		ID:          "web/src/app/demo.ts",
		PrimaryFile: "web/src/app/demo.ts",
		WebDir:      webDir,
		Files: []BundledFile{
			{
				RelPath: "web/src/app/demo.ts",
				Content: []byte(`export interface DemoConfig {
  readonly enabled: boolean;
}

export function runDemo(): void {}
`),
			},
			{
				RelPath: "web/src/app/demo.spec.ts",
				Content: []byte(`describe('Demo', () => {
  it('runs cleanly', () => {});
});
`),
			},
		},
	}

	testCases := []struct {
		name string
		rule RuleSpec
		want []CompiledBatch
	}{
		// Expands changed_ts_decls across production TypeScript interfaces and functions.
		{
			name: "changed_ts_decls extractor renders noul criteria for TS declarations",
			rule: RuleSpec{
				ID:   "ts_decl_rule",
				Name: "TS Decl Rule",
				Questions: []QuestionSpec{{
					ID: "q_ts_decl", ForEach: ExtractorChangedTSDecls, Type: QuestionTypeNoul,
					Instructions: "Inspect {{.Kind}} {{.Name}}",
					Criteria:     NoulCriteriaSpec{TrueDescription: "Valid {{.Name}}"},
					FixHint:      "Fix",
					Pass:         PassSpec{Expect: ptr(true), MinProbability: 0.5},
				}},
			},
			want: []CompiledBatch{{
				Request: DecisionRequest{
					Model: "gemini-decision-1",
					Input: "shared-ctx",
					Questions: map[string]WireQuestion{
						"q_ts_decl_0": {Type: QuestionTypeNoul, Instructions: "Inspect interface DemoConfig", Criteria: map[string]string{"true_description": "Valid DemoConfig"}},
						"q_ts_decl_1": {Type: QuestionTypeNoul, Instructions: "Inspect function runDemo", Criteria: map[string]string{"true_description": "Valid runDemo"}},
					},
				},
				Order: []string{"q_ts_decl_0", "q_ts_decl_1"},
			}},
		},
		// Expands changed_ts_funcs and renders choice criteria for production TypeScript functions.
		{
			name: "changed_ts_funcs extractor renders choice criteria for TS functions",
			rule: RuleSpec{
				ID:   "ts_func_rule",
				Name: "TS Func Rule",
				Questions: []QuestionSpec{{
					ID: "q_ts_func", ForEach: ExtractorChangedTSFuncs, Type: QuestionTypeChoice,
					Instructions: "Check {{.Name}}",
					Options: map[string]ChoiceOptionSpec{
						"ok":  {Description: "{{.Name}} good"},
						"bad": {Description: "{{.Name}} bad", FixHint: "Fix"},
					},
					Pass: PassSpec{ExpectedChoice: "ok", MinProbability: 0.5},
				}},
			},
			want: []CompiledBatch{{
				Request: DecisionRequest{
					Model: "gemini-decision-1",
					Input: "shared-ctx",
					Questions: map[string]WireQuestion{
						"q_ts_func_0": {Type: QuestionTypeChoice, Instructions: "Check runDemo", Criteria: map[string]string{"ok": "runDemo good", "bad": "runDemo bad"}},
					},
				},
				Order: []string{"q_ts_func_0"},
			}},
		},
		// Expands changed_ts_specs and renders ordered score level criteria for Jasmine specs.
		{
			name: "changed_ts_specs extractor renders score criteria for TS specs",
			rule: RuleSpec{
				ID:   "ts_spec_rule",
				Name: "TS Spec Rule",
				Questions: []QuestionSpec{{
					ID: "q_ts_spec", ForEach: ExtractorChangedTSSpecs, Type: QuestionTypeScore,
					Instructions: "Score {{.Name}}",
					Levels:       []ScoreLevelSpec{{Description: "{{.Name}} low", FixHint: "Fix"}, {Description: "{{.Name}} high"}},
					Pass:         PassSpec{MinScore: ptr(1.0)},
				}},
			},
			want: []CompiledBatch{{
				Request: DecisionRequest{
					Model: "gemini-decision-1",
					Input: "shared-ctx",
					Questions: map[string]WireQuestion{
						"q_ts_spec_0": {Type: QuestionTypeScore, Instructions: `Score Demo > "runs cleanly"`, Criteria: []string{`Demo > "runs cleanly" low`, `Demo > "runs cleanly" high`}},
					},
				},
				Order: []string{"q_ts_spec_0"},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CompileQuestions("gemini-decision-1", "shared-ctx", tc.rule, tsBundle)

			if err != nil {
				t.Fatalf("CompileQuestions() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreFields(CompiledBatch{}, "Meta")); diff != "" {
				t.Errorf("CompileQuestions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestCompileQuestionsErrors verifies that unsupported extractors and question types return descriptive errors.
func TestCompileQuestionsErrors(t *testing.T) {
	bundle := TargetBundle{ID: "pkg/demo", PrimaryFile: "pkg/demo/a.go"}
	testCases := []struct {
		name    string
		rule    RuleSpec
		wantErr string
	}{
		// Rejects an unknown for_each extractor kind during compilation.
		{
			name: "unsupported extractor returns error",
			rule: RuleSpec{
				ID:        "bad_ext",
				Questions: []QuestionSpec{{ID: "q1", ForEach: ExtractorKind("unknown_extractor"), Type: QuestionTypeNoul, Instructions: "Check"}},
			},
			wantErr: "unsupported extractor",
		},
		// Rejects an unknown question type when rendering wire criteria.
		{
			name: "unsupported question type returns error",
			rule: RuleSpec{
				ID:        "bad_type",
				Questions: []QuestionSpec{{ID: "q1", Type: QuestionType("unknown_type"), Instructions: "Check"}},
			},
			wantErr: "unsupported question type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompileQuestions("gemini-decision-1", "shared-ctx", tc.rule, bundle)

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("CompileQuestions() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

// TestEvaluateBatch verifies confidence gating, pass boundaries, fallback choice selection, and failing score level selection.
func TestEvaluateBatch(t *testing.T) {
	item := ExtractedItem{Target: "pkg/demo", File: "pkg/demo/a.go", Line: 10, StartLine: 10, EndLine: 12, Name: "demo"}

	testCases := []struct {
		name        string
		enforcement EnforcementMode
		question    QuestionSpec
		resp        DecisionResponse
		want        []rule.DecisionItemFinding
	}{
		// Suppresses finding when answer confidence is strictly below pass.min_confidence.
		{
			name:        "confidence below min_confidence suppresses finding",
			enforcement: EnforcementAdvisory,
			question: QuestionSpec{
				ID: "q_conf", Type: QuestionTypeNoul, Instructions: "Check {{.Name}}", FixHint: "Fix {{.Name}}.",
				Pass: PassSpec{Expect: ptr(true), MinProbability: 0.50, MinConfidence: 0.55},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_conf": {Noul: ptr(0.10), Confidence: 0.54}}},
			want: nil,
		},
		// Emits a blocking finding at exact min_confidence when expect=false fails and renders TrueDescription as Why.
		{
			name:        "expect false noul emits finding at exact min_confidence using true_description",
			enforcement: EnforcementBlocking,
			question: QuestionSpec{
				ID: "q_noul_false", Type: QuestionTypeNoul, Instructions: "Has bug in {{.Name}}?",
				Criteria: NoulCriteriaSpec{TrueDescription: "Found issue in {{.Name}}.", FalseDescription: "Clean."},
				FixHint:  "Fix {{.Name}}.",
				Pass:     PassSpec{Expect: ptr(false), MinProbability: 0.75, MinConfidence: 0.55},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_noul_false": {Noul: ptr(0.30), Confidence: 0.55}}},
			want: []rule.DecisionItemFinding{{
				RuleID: "demo_rule", RuleName: "Demo Rule", QuestionID: "q_noul_false", Blocking: true,
				Target: "pkg/demo", File: "pkg/demo/a.go", Line: 10, EndLine: 12, Subject: "demo",
				Outcome: "p_true=0.30 (want false >= 0.75, conf=0.55)", Why: "Found issue in demo.", Fix: "Fix demo.",
			}},
		},
		// Passes when expect=false satisfies 1 - pTrue == min_probability at the exact boundary.
		{
			name:        "expect false noul passes at exact min_probability boundary",
			enforcement: EnforcementAdvisory,
			question: QuestionSpec{
				ID: "q_noul_pass", Type: QuestionTypeNoul, Instructions: "Has bug?", FixHint: "Fix.",
				Pass: PassSpec{Expect: ptr(false), MinProbability: 0.75},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_noul_pass": {Noul: ptr(0.25), Confidence: 0.90}}},
			want: nil,
		},
		// Passes when summed allowed_choices probability equals pass.min_probability at the exact boundary.
		{
			name:        "choice passes at exact min_probability across allowed_choices",
			enforcement: EnforcementAdvisory,
			question: QuestionSpec{
				ID: "q_choice_pass", Type: QuestionTypeChoice, Instructions: "Check",
				Options: map[string]ChoiceOptionSpec{
					"ok_a": {Description: "OK A"}, "ok_b": {Description: "OK B"}, "bad": {Description: "Bad", FixHint: "Fix"},
				},
				Pass: PassSpec{AllowedChoices: []string{"ok_a", "ok_b"}, MinProbability: 0.50},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_choice_pass": {
				Choice: "ok_a", Probabilities: map[string]float64{"ok_a": 0.25, "ok_b": 0.25, "bad": 0.50}, Confidence: 0.85,
			}}},
			want: nil,
		},
		// Selects highest-probability non-passing option when model picks passing option with insufficient probability.
		{
			name:        "choice selects top failing option when allowed option fails min_probability",
			enforcement: EnforcementAdvisory,
			question: QuestionSpec{
				ID: "q_choice_fallback", Type: QuestionTypeChoice, Instructions: "Check",
				Options: map[string]ChoiceOptionSpec{
					"aligned": {Description: "Aligned"}, "err_a": {Description: "Err A", FixHint: "Fix A"},
					"err_b": {Description: "Err B", FixHint: "Fix B"}, "err_c": {Description: "Err C", FixHint: "Fix C"},
				},
				Pass: PassSpec{ExpectedChoice: "aligned", MinProbability: 0.60},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_choice_fallback": {
				Choice: "aligned", Probabilities: map[string]float64{"aligned": 0.45, "err_a": 0.10, "err_b": 0.30, "err_c": 0.15}, Confidence: 0.90,
			}}},
			want: []rule.DecisionItemFinding{{
				RuleID: "demo_rule", RuleName: "Demo Rule", QuestionID: "q_choice_fallback",
				Target: "pkg/demo", File: "pkg/demo/a.go", Line: 10, EndLine: 12, Subject: "demo",
				Outcome: "err_b (p=0.30, pass_p=0.45, conf=0.90)", Why: "Err B", Fix: "Fix B",
			}},
		},
		// Breaks equal-probability ties alphabetically when selected choice is absent from configured options.
		{
			name:        "choice fallback breaks ties alphabetically for unknown choice",
			enforcement: EnforcementAdvisory,
			question: QuestionSpec{
				ID: "q_choice_tie", Type: QuestionTypeChoice, Instructions: "Check",
				Options: map[string]ChoiceOptionSpec{
					"aligned": {Description: "Aligned"}, "alpha_err": {Description: "Alpha", FixHint: "Fix Alpha"}, "beta_err": {Description: "Beta", FixHint: "Fix Beta"},
				},
				Pass: PassSpec{ExpectedChoice: "aligned", MinProbability: 0.60},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_choice_tie": {
				Choice: "unknown_opt", Probabilities: map[string]float64{"aligned": 0.30, "alpha_err": 0.35, "beta_err": 0.35}, Confidence: 0.80,
			}}},
			want: []rule.DecisionItemFinding{{
				RuleID: "demo_rule", RuleName: "Demo Rule", QuestionID: "q_choice_tie",
				Target: "pkg/demo", File: "pkg/demo/a.go", Line: 10, EndLine: 12, Subject: "demo",
				Outcome: "alpha_err (p=0.35, pass_p=0.30, conf=0.80)", Why: "Alpha", Fix: "Fix Alpha",
			}},
		},
		// Preserves explicitly selected non-passing option even when another non-passing option has higher probability.
		{
			name:        "choice preserves valid selected non-passing option",
			enforcement: EnforcementAdvisory,
			question: QuestionSpec{
				ID: "q_choice_selected", Type: QuestionTypeChoice, Instructions: "Check",
				Options: map[string]ChoiceOptionSpec{
					"aligned": {Description: "Aligned"}, "minor_err": {Description: "Minor", FixHint: "Fix Minor"}, "major_err": {Description: "Major", FixHint: "Fix Major"},
				},
				Pass: PassSpec{ExpectedChoice: "aligned", MinProbability: 0.60},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_choice_selected": {
				Choice: "minor_err", Probabilities: map[string]float64{"aligned": 0.10, "minor_err": 0.30, "major_err": 0.60}, Confidence: 0.85,
			}}},
			want: []rule.DecisionItemFinding{{
				RuleID: "demo_rule", RuleName: "Demo Rule", QuestionID: "q_choice_selected",
				Target: "pkg/demo", File: "pkg/demo/a.go", Line: 10, EndLine: 12, Subject: "demo",
				Outcome: "minor_err (p=0.30, pass_p=0.10, conf=0.85)", Why: "Minor", Fix: "Fix Minor",
			}},
		},
		// Passes when numeric score equals pass.min_score at the exact boundary.
		{
			name:        "score passes at exact min_score boundary",
			enforcement: EnforcementAdvisory,
			question: QuestionSpec{
				ID: "q_score_min_pass", Type: QuestionTypeScore, Instructions: "Rate",
				Levels: []ScoreLevelSpec{{Description: "L0", FixHint: "Fix 0"}, {Description: "L1", FixHint: "Fix 1"}, {Description: "L2"}},
				Pass:   PassSpec{MinScore: ptr(1.5)},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_score_min_pass": {Score: ptr(1.5), Confidence: 0.80}}},
			want: nil,
		},
		// Selects failing level 0 on equal-probability tie across failing levels even when score rounds to 1.
		{
			name:        "score selects first tied failing level at index 0",
			enforcement: EnforcementAdvisory,
			question: QuestionSpec{
				ID: "q_score_tie0", Type: QuestionTypeScore, Instructions: "Rate",
				Levels: []ScoreLevelSpec{
					{Description: "Severe", FixHint: "Fix Severe"}, {Description: "Moderate", FixHint: "Fix Moderate"},
					{Description: "Minor", FixHint: "Fix Minor"}, {Description: "Good"},
				},
				Pass: PassSpec{MinScore: ptr(2.5)},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_score_tie0": {
				Score: ptr(1.1), Probabilities: map[string]float64{"0": 0.40, "1": 0.40, "2": 0.15, "3": 0.05}, Confidence: 0.85,
			}}},
			want: []rule.DecisionItemFinding{{
				RuleID: "demo_rule", RuleName: "Demo Rule", QuestionID: "q_score_tie0",
				Target: "pkg/demo", File: "pkg/demo/a.go", Line: 10, EndLine: 12, Subject: "demo",
				Outcome: "score=1.10/3 (< 2.50, conf=0.85)", Why: "Severe", Fix: "Fix Severe",
			}},
		},
		// Passes when numeric score equals pass.max_score with pass.min_score nil.
		{
			name:        "score passes at exact max_score boundary",
			enforcement: EnforcementAdvisory,
			question: QuestionSpec{
				ID: "q_score_max_pass", Type: QuestionTypeScore, Instructions: "Rate",
				Levels: []ScoreLevelSpec{{Description: "Clean"}, {Description: "Risky"}, {Description: "Broken", FixHint: "Fix"}},
				Pass:   PassSpec{MaxScore: ptr(1.0)},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_score_max_pass": {Score: ptr(1.0), Confidence: 0.90}}},
			want: nil,
		},
		// Emits max_score outcome format and clamps rounded score to last level when no level is marked failing.
		{
			name:        "score above max_score formats max_score outcome and clamps fallback level",
			enforcement: EnforcementAdvisory,
			question: QuestionSpec{
				ID: "q_score_max_fail", Type: QuestionTypeScore, Instructions: "Rate", FixHint: "Fallback fix.",
				Levels: []ScoreLevelSpec{{Description: "L0"}, {Description: "L1"}, {Description: "L2"}},
				Pass:   PassSpec{MaxScore: ptr(2.0)},
			},
			resp: DecisionResponse{Answers: map[string]WireAnswer{"q_score_max_fail": {Score: ptr(4.0), Confidence: 0.85}}},
			want: []rule.DecisionItemFinding{{
				RuleID: "demo_rule", RuleName: "Demo Rule", QuestionID: "q_score_max_fail",
				Target: "pkg/demo", File: "pkg/demo/a.go", Line: 10, EndLine: 12, Subject: "demo",
				Outcome: "score=4.00/2 (> 2.00, conf=0.85)", Why: "L2", Fix: "Fallback fix.",
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ruleSpec := RuleSpec{ID: "demo_rule", Name: "Demo Rule", Enforcement: tc.enforcement, Questions: []QuestionSpec{tc.question}}
			batch := CompiledBatch{Order: []string{tc.question.ID}, Meta: map[string]compiledQuestionMeta{tc.question.ID: {spec: tc.question, item: item}}}

			got, err := EvaluateBatch(ruleSpec, false, batch, tc.resp)

			if err != nil {
				t.Fatalf("EvaluateBatch() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("EvaluateBatch() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestEvaluateBatchErrors verifies that EvaluateBatch rejects responses missing an expected question key.
func TestEvaluateBatchErrors(t *testing.T) {
	ruleSpec := RuleSpec{ID: "demo_rule", Name: "Demo Rule"}
	batch := CompiledBatch{
		Order: []string{"q_missing"},
		Meta: map[string]compiledQuestionMeta{
			"q_missing": {spec: QuestionSpec{ID: "q_missing", Type: QuestionTypeNoul}, item: ExtractedItem{Target: "pkg/demo"}},
		},
	}
	resp := DecisionResponse{Answers: map[string]WireAnswer{}}

	_, err := EvaluateBatch(ruleSpec, false, batch, resp)

	if err == nil || !strings.Contains(err.Error(), "missing answer") {
		t.Errorf("EvaluateBatch() error = %v, want substring %q", err, "missing answer")
	}
}

// TestEvaluateSingleAnswerErrors verifies that invalid question types and missing answer payloads return failed=false with a zero finding and error.
func TestEvaluateSingleAnswerErrors(t *testing.T) {
	testCases := []struct {
		name     string
		question QuestionSpec
		ans      WireAnswer
		wantErr  string
	}{
		// Rejects an unsupported question type during single-answer evaluation.
		{
			name:     "unsupported question type returns error",
			question: QuestionSpec{ID: "q_unknown", Type: QuestionType("unknown_type")},
			ans:      WireAnswer{Confidence: 0.90},
			wantErr:  "unsupported question type",
		},
		// Rejects a noul answer whose Noul probability pointer is nil.
		{
			name:     "noul answer missing probability returns error",
			question: QuestionSpec{ID: "q_noul_nil", Type: QuestionTypeNoul, Pass: PassSpec{Expect: ptr(true), MinProbability: 0.50}},
			ans:      WireAnswer{Confidence: 0.90},
			wantErr:  "noul answer missing probability",
		},
		// Rejects a score answer whose Score pointer is nil.
		{
			name:     "score answer missing numeric score returns error",
			question: QuestionSpec{ID: "q_score_nil", Type: QuestionTypeScore, Pass: PassSpec{MinScore: ptr(1.0)}},
			ans:      WireAnswer{Confidence: 0.90},
			wantErr:  "score answer missing numeric score",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ruleSpec := RuleSpec{ID: "demo_rule", Name: "Demo Rule"}
			meta := compiledQuestionMeta{spec: tc.question, item: ExtractedItem{Target: "pkg/demo"}}

			finding, failed, err := evaluateSingleAnswer(ruleSpec, false, meta, tc.ans)

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("evaluateSingleAnswer() error = %v, want substring %q", err, tc.wantErr)
			}
			if failed {
				t.Errorf("evaluateSingleAnswer() failed = true, want false on error")
			}
			if diff := cmp.Diff(rule.DecisionItemFinding{}, finding); diff != "" {
				t.Errorf("evaluateSingleAnswer() finding mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
