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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
)

// TestParseAndValidateRuleYAML verifies strict YAML decoding, template checks, and rule validation errors.
func TestParseAndValidateRuleYAML(t *testing.T) {
	testCases := []struct {
		name                   string
		yaml                   string
		wantIncludeCompanionTS bool
		wantErr                string
	}{
		// Accepts a valid rule combining score, choice, and noul questions with outcome-specific fix_hints.
		{
			name: "valid rule with noul choice and score questions passes",
			yaml: `
id: demo_rule
name: Demo Rule
description: Checks alignment.
target:
  kind: go_logical_package
  include: ["pkg/**"]
questions:
  - id: q_score
    type: score
    instructions: "Rate {{.Target}}"
    levels:
      - description: "Bad"
        fix_hint: "Fix {{.File}}"
      - description: "Good"
    pass:
      min_score: 1.0
  - id: q_choice
    for_each: markdown_sections
    type: choice
    instructions: "Check {{.Heading}}"
    options:
      ok:
        description: "Matches"
      bad:
        description: "Drifted"
        fix_hint: "Update {{.Heading}} at L{{.StartLine}}"
    pass:
      expected_choice: ok
      min_probability: 0.5
  - id: q_noul
    for_each: changed_go_test_cases
    type: noul
    instructions: "Case {{.Name}} ({{.Doc}})"
    fix_hint: "Fix {{.Name}} at {{.File}}:{{.Line}}"
    pass:
      expect: true
      min_probability: 0.5
`,
			wantErr: "",
		},
		// Accepts a changed_file TS rule using include_companion_ts and all TypeScript extractors.
		{
			name: "valid TS rule with include_companion_ts and TS extractors passes",
			yaml: `
id: ts_demo
name: TS Demo
target:
  kind: changed_file
  include: ["web/src/**/*.ts"]
context:
  include_companion_ts: true
questions:
  - id: decls
    for_each: changed_ts_decls
    type: noul
    instructions: "Check {{.Kind}} {{.Name}}"
    fix_hint: "Fix {{.File}}"
    pass:
      expect: true
      min_probability: 0.5
  - id: funcs
    for_each: changed_ts_funcs
    type: noul
    instructions: "Name {{.Name}}"
    fix_hint: "Rename {{.Name}}"
    pass:
      expect: true
      min_probability: 0.5
  - id: specs
    for_each: changed_ts_specs
    type: noul
    instructions: "Spec {{.Name}}"
    fix_hint: "Fix {{.Name}}"
    pass:
      expect: true
      min_probability: 0.5
`,
			wantIncludeCompanionTS: true,
		},
		// Rejects unknown YAML keys through yaml.v3 KnownFields strict decoding.
		{
			name: "unknown YAML field rejected by strict decoder",
			yaml: `
id: bad_field
name: Bad Field
unknown_key: true
target:
  kind: changed_file
questions: []
`,
			wantErr: "field unknown_key not found",
		},
		// Rejects choice questions when a non-passing option lacks both option-level and fallback fix_hint.
		{
			name: "missing fix_hint on non-passing choice option rejected",
			yaml: `
id: missing_hint
name: Missing Hint
target:
  kind: changed_file
questions:
  - id: c1
    type: choice
    instructions: "Check {{.File}}"
    options:
      aligned:
        description: "OK"
      stale:
        description: "Stale"
    pass:
      expected_choice: aligned
      min_probability: 0.5
`,
			wantErr: "non-passing option requires fix_hint",
		},
		// Rejects score questions when a level below pass.min_score lacks a fix_hint.
		{
			name: "missing fix_hint on failing score level rejected",
			yaml: `
id: missing_score_hint
name: Missing Score Hint
target:
  kind: changed_file
questions:
  - id: s1
    type: score
    instructions: "Score {{.File}}"
    levels:
      - description: "Fail level"
      - description: "Pass level"
    pass:
      min_score: 1.0
`,
			wantErr: "failing score level requires fix_hint",
		},
		// Rejects noul questions when question-level fix_hint is omitted.
		{
			name: "missing fix_hint on noul rejected",
			yaml: `
id: missing_noul_hint
name: Missing Noul Hint
target:
  kind: changed_file
questions:
  - id: n1
    type: noul
    instructions: "Check {{.File}}"
    pass:
      expect: true
      min_probability: 0.5
`,
			wantErr: "noul question requires a non-empty fix_hint",
		},
		// Rejects templates referencing fields that do not exist on ExtractedItem.
		{
			name: "typo in template field rejected at parse time",
			yaml: `
id: bad_tmpl
name: Bad Template
target:
  kind: changed_file
questions:
  - id: n1
    type: noul
    instructions: "Check {{.NonExistentField}}"
    fix_hint: "Fix {{.File}}"
    pass:
      expect: true
      min_probability: 0.5
`,
			wantErr: "template references unknown field",
		},
		// Rejects unsupported for_each extractor names.
		{
			name: "unknown for_each extractor rejected",
			yaml: `
id: bad_ext
name: Bad Extractor
target:
  kind: changed_file
questions:
  - id: n1
    for_each: non_existent_extractor
    type: noul
    instructions: "Check {{.File}}"
    fix_hint: "Fix"
    pass:
      expect: true
      min_probability: 0.5
`,
			wantErr: "unknown for_each extractor",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseRuleYAML([]byte(tc.yaml))

			if tc.wantErr == "" && err != nil {
				t.Fatalf("ParseRuleYAML() unexpected error: %v", err)
			}
			if tc.wantErr == "" && got.Context.IncludeCompanionTS != tc.wantIncludeCompanionTS {
				t.Errorf("Context.IncludeCompanionTS = %v, want %v", got.Context.IncludeCompanionTS, tc.wantIncludeCompanionTS)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Errorf("ParseRuleYAML() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

// TestLoadRulesDir verifies sorted rule loading, non-YAML skipping, and duplicate rule ID rejection.
func TestLoadRulesDir(t *testing.T) {
	baseRule := `
id: b_rule
name: B Rule
target:
  kind: changed_file
questions:
  - id: q1
    type: noul
    instructions: "Check {{.File}}"
    fix_hint: "Fix"
    pass:
      expect: true
      min_probability: 0.5
`
	testCases := []struct {
		name    string
		files   map[string]string
		wantIDs []string
		wantErr string
	}{
		// Loads .yaml and .yml files sorted by rule ID while ignoring non-YAML documentation files.
		{
			name: "sorts rules by ID and ignores README.md",
			files: map[string]string{
				"2.yaml":    baseRule,
				"1.yml":     strings.Replace(baseRule, "id: b_rule", "id: a_rule", 1),
				"README.md": "# Ignored\n",
			},
			wantIDs: []string{"a_rule", "b_rule"},
		},
		// Rejects directories containing two YAML files that declare the same rule ID.
		{
			name: "rejects duplicate rule IDs across files",
			files: map[string]string{
				"first.yaml":  baseRule,
				"second.yaml": strings.Replace(baseRule, "name: B Rule", "name: Duplicate B Rule", 1),
			},
			wantErr: "duplicate rule id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeTempRuleFiles(t, tc.files)

			specs, err := LoadRulesDir(dir)

			assertLoadedRuleIDs(t, specs, err, tc.wantIDs, tc.wantErr)
		})
	}
}

// TestBuiltinRulesDirectoryValidatesCleanly verifies that all production YAML rules under scripts/codequality/rules parse and validate cleanly.
func TestBuiltinRulesDirectoryValidatesCleanly(t *testing.T) {
	repoRoot, err := gotestutil.ResolveRepoRoot("")
	if err != nil {
		t.Fatalf("ResolveRepoRoot() error: %v", err)
	}
	rulesDir := filepath.Join(repoRoot, "scripts", "codequality", "rules")

	specs, err := LoadRulesDir(rulesDir)

	assertLoadedRuleIDs(t, specs, err, []string{"go_code_quality", "go_refactor_safety", "go_test_quality", "ts_code_quality", "ts_test_quality"}, "")
	if !specs[4].Context.IncludeCompanionTS {
		t.Errorf("ts_test_quality.Context.IncludeCompanionTS = false, want true")
	}
}

// TestAllowedPassChoices verifies that AllowedPassChoices deduplicates ExpectedChoice and non-empty AllowedChoices entries.
func TestAllowedPassChoices(t *testing.T) {
	spec := PassSpec{
		ExpectedChoice: "ok_a",
		AllowedChoices: []string{"ok_b", "", "ok_a", "ok_c"},
	}

	got := spec.AllowedPassChoices()

	want := map[string]struct{}{
		"ok_a": {},
		"ok_b": {},
		"ok_c": {},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("AllowedPassChoices() mismatch (-want +got):\n%s", diff)
	}
}

// TestValidateRuleValidBoundaries verifies boundary-valid rule configurations including max option/level counts and zero thresholds.
func TestValidateRuleValidBoundaries(t *testing.T) {
	testCases := []struct {
		name string
		rule RuleSpec
	}{
		// Accepts a blocking choice rule with the maximum 255 options.
		{
			name: "blocking choice rule with 255 options passes",
			rule: func() RuleSpec {
				r := validChoiceRuleFixture()
				r.Enforcement = EnforcementBlocking
				r.Questions[0].Options = makeChoiceOptions(maxChoiceOrScoreCardinality)
				r.Questions[0].Pass.ExpectedChoice = "opt_0"
				return r
			}(),
		},
		// Accepts a score rule with the maximum 255 levels and pass.min_score at the lower bound 0.
		{
			name: "score rule with 255 levels and min_score zero passes",
			rule: func() RuleSpec {
				r := validScoreRuleFixture()
				r.Questions[0].Levels = makeScoreLevels(maxChoiceOrScoreCardinality, true)
				r.Questions[0].Pass.MinScore = floatPtr(0.0)
				return r
			}(),
		},
		// Accepts a score rule with pass.max_score=0 when only level 1 above max_score requires a fix_hint.
		{
			name: "score rule with max_score zero requires fix_hint only on level above zero",
			rule: func() RuleSpec {
				r := validScoreRuleFixture()
				r.Questions[0].Levels = []ScoreLevelSpec{
					{Description: "Passing"},
					{Description: "Failing", FixHint: "Fix failing level"},
				}
				r.Questions[0].Pass = PassSpec{MaxScore: floatPtr(0.0)}
				return r
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRule(tc.rule)

			if err != nil {
				t.Fatalf("ValidateRule() unexpected error: %v", err)
			}
		})
	}
}

// TestValidateRuleErrors verifies that invalid rule metadata, question definitions, probability bounds, and score thresholds are rejected.
func TestValidateRuleErrors(t *testing.T) {
	testCases := []struct {
		name    string
		rule    RuleSpec
		wantErr string
	}{
		// Rejects rules with an empty ID even when Name is provided.
		{
			name: "missing rule id rejected",
			rule: func() RuleSpec {
				r := validNoulRuleFixture()
				r.ID = ""
				return r
			}(),
			wantErr: "rule id and name are required",
		},
		// Rejects unsupported enforcement modes outside advisory and blocking.
		{
			name: "invalid enforcement rejected",
			rule: func() RuleSpec {
				r := validNoulRuleFixture()
				r.Enforcement = EnforcementMode("strict")
				return r
			}(),
			wantErr: "invalid enforcement",
		},
		// Rejects unsupported target.kind values outside go_logical_package and changed_file.
		{
			name: "invalid target kind rejected",
			rule: func() RuleSpec {
				r := validNoulRuleFixture()
				r.Target.Kind = TargetKind("unknown")
				return r
			}(),
			wantErr: "invalid target.kind",
		},
		// Rejects rules that define zero questions.
		{
			name: "empty questions slice rejected",
			rule: func() RuleSpec {
				r := validNoulRuleFixture()
				r.Questions = nil
				return r
			}(),
			wantErr: "at least one question is required",
		},
		// Rejects rules containing two questions with the same ID.
		{
			name: "duplicate question id rejected",
			rule: func() RuleSpec {
				r := validNoulRuleFixture()
				r.Questions = append(r.Questions, r.Questions[0])
				return r
			}(),
			wantErr: "duplicate question id",
		},
		// Rejects questions with blank instructions even when question ID is present.
		{
			name: "blank question instructions rejected",
			rule: func() RuleSpec {
				r := validNoulRuleFixture()
				r.Questions[0].Instructions = "   "
				return r
			}(),
			wantErr: "question id and instructions are required",
		},
		// Rejects unsupported question types outside noul, choice, and score.
		{
			name: "invalid question type rejected",
			rule: func() RuleSpec {
				r := validNoulRuleFixture()
				r.Questions[0].Type = QuestionType("freeform")
				return r
			}(),
			wantErr: "invalid question type",
		},
		// Rejects noul questions when pass.expect is omitted.
		{
			name: "noul missing pass.expect rejected",
			rule: func() RuleSpec {
				r := validNoulRuleFixture()
				r.Questions[0].Pass.Expect = nil
				return r
			}(),
			wantErr: "noul question requires pass.expect",
		},
		// Rejects min_probability equal to zero on the exclusive lower bound.
		{
			name: "min_probability zero rejected",
			rule: func() RuleSpec {
				r := validNoulRuleFixture()
				r.Questions[0].Pass.MinProbability = 0
				return r
			}(),
			wantErr: "pass.min_probability must be in (0, 1)",
		},
		// Rejects min_probability equal to one on the exclusive upper bound for a choice question.
		{
			name: "min_probability one rejected",
			rule: func() RuleSpec {
				r := validChoiceRuleFixture()
				r.Questions[0].Pass.MinProbability = 1
				return r
			}(),
			wantErr: "pass.min_probability must be in (0, 1)",
		},
		// Rejects choice questions defining fewer than two options.
		{
			name: "choice with single option rejected",
			rule: func() RuleSpec {
				r := validChoiceRuleFixture()
				r.Questions[0].Options = map[string]ChoiceOptionSpec{
					"ok": {Description: "Only one"},
				}
				return r
			}(),
			wantErr: "choice requires 2..255 options",
		},
		// Rejects choice questions defining neither expected_choice nor allowed_choices.
		{
			name: "choice missing passing options rejected",
			rule: func() RuleSpec {
				r := validChoiceRuleFixture()
				r.Questions[0].Pass.ExpectedChoice = ""
				r.Questions[0].Pass.AllowedChoices = nil
				return r
			}(),
			wantErr: "choice requires pass.expected_choice or pass.allowed_choices",
		},
		// Rejects choice questions referencing a passing option key absent from options.
		{
			name: "choice referencing undefined passing option rejected",
			rule: func() RuleSpec {
				r := validChoiceRuleFixture()
				r.Questions[0].Pass.ExpectedChoice = "missing_option"
				return r
			}(),
			wantErr: "not defined in options",
		},
		// Rejects choice options whose description is blank.
		{
			name: "choice option with blank description rejected",
			rule: func() RuleSpec {
				r := validChoiceRuleFixture()
				r.Questions[0].Options["ok"] = ChoiceOptionSpec{Description: "  "}
				return r
			}(),
			wantErr: "description is required",
		},
		// Rejects score questions defining fewer than two rubric levels.
		{
			name: "score with single level rejected",
			rule: func() RuleSpec {
				r := validScoreRuleFixture()
				r.Questions[0].Levels = []ScoreLevelSpec{{Description: "Only"}}
				return r
			}(),
			wantErr: "score requires 2..255 levels",
		},
		// Rejects score questions omitting both pass.min_score and pass.max_score.
		{
			name: "score missing min_score and max_score rejected",
			rule: func() RuleSpec {
				r := validScoreRuleFixture()
				r.Questions[0].Pass = PassSpec{}
				return r
			}(),
			wantErr: "score requires pass.min_score or pass.max_score",
		},
		// Rejects pass.min_score exceeding len(levels)-1 on a 2-level rubric.
		{
			name: "score min_score exceeding max level rejected",
			rule: func() RuleSpec {
				r := validScoreRuleFixture()
				r.Questions[0].Pass.MinScore = floatPtr(2.0)
				return r
			}(),
			wantErr: "pass.min_score out of range",
		},
		// Rejects pass.max_score exceeding len(levels)-1 on a 2-level rubric.
		{
			name: "score max_score exceeding max level rejected",
			rule: func() RuleSpec {
				r := validScoreRuleFixture()
				r.Questions[0].Pass = PassSpec{MaxScore: floatPtr(2.0)}
				return r
			}(),
			wantErr: "pass.max_score out of range",
		},
		// Rejects score rubric levels whose description is blank.
		{
			name: "score level with blank description rejected",
			rule: func() RuleSpec {
				r := validScoreRuleFixture()
				r.Questions[0].Levels[0].Description = "  "
				return r
			}(),
			wantErr: "description is required",
		},
		// Rejects score questions when a failing level above pass.max_score lacks a fix_hint.
		{
			name: "score failing level above max_score missing fix_hint rejected",
			rule: func() RuleSpec {
				r := validScoreRuleFixture()
				r.Questions[0].Levels = []ScoreLevelSpec{
					{Description: "Good"},
					{Description: "Bad"},
				}
				r.Questions[0].Pass = PassSpec{MaxScore: floatPtr(0.0)}
				return r
			}(),
			wantErr: "failing score level requires fix_hint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRule(tc.rule)

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("ValidateRule() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func writeTempRuleFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("WriteFile(%s) error: %v", name, err)
		}
	}
	return dir
}

func assertLoadedRuleIDs(t *testing.T, specs []RuleSpec, err error, wantIDs []string, wantErr string) {
	t.Helper()
	if wantErr != "" {
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("LoadRulesDir() error = %v, want substring %q", err, wantErr)
		}
		return
	}
	if err != nil {
		t.Fatalf("LoadRulesDir() unexpected error: %v", err)
	}
	gotIDs := make([]string, len(specs))
	for i, s := range specs {
		gotIDs[i] = s.ID
	}
	if diff := cmp.Diff(wantIDs, gotIDs); diff != "" {
		t.Errorf("LoadRulesDir() IDs mismatch (-want +got):\n%s", diff)
	}
}

func validNoulRuleFixture() RuleSpec {
	expect := true
	return RuleSpec{
		ID:          "demo_rule",
		Name:        "Demo Rule",
		Enforcement: EnforcementAdvisory,
		Target:      TargetSpec{Kind: TargetChangedFile},
		Questions: []QuestionSpec{{
			ID:           "q1",
			Type:         QuestionTypeNoul,
			Instructions: "Check {{.File}}",
			FixHint:      "Fix",
			Pass:         PassSpec{Expect: &expect, MinProbability: 0.5},
		}},
	}
}

func validChoiceRuleFixture() RuleSpec {
	return RuleSpec{
		ID:          "demo_choice",
		Name:        "Demo Choice",
		Enforcement: EnforcementAdvisory,
		Target:      TargetSpec{Kind: TargetChangedFile},
		Questions: []QuestionSpec{{
			ID:           "q1",
			Type:         QuestionTypeChoice,
			Instructions: "Check {{.File}}",
			Options: map[string]ChoiceOptionSpec{
				"ok":  {Description: "OK"},
				"bad": {Description: "Bad", FixHint: "Fix"},
			},
			Pass: PassSpec{ExpectedChoice: "ok", MinProbability: 0.5},
		}},
	}
}

func validScoreRuleFixture() RuleSpec {
	return RuleSpec{
		ID:          "demo_score",
		Name:        "Demo Score",
		Enforcement: EnforcementAdvisory,
		Target:      TargetSpec{Kind: TargetChangedFile},
		Questions: []QuestionSpec{{
			ID:           "q1",
			Type:         QuestionTypeScore,
			Instructions: "Rate {{.File}}",
			Levels: []ScoreLevelSpec{
				{Description: "Bad", FixHint: "Fix"},
				{Description: "Good"},
			},
			Pass: PassSpec{MinScore: floatPtr(1.0)},
		}},
	}
}

func makeChoiceOptions(count int) map[string]ChoiceOptionSpec {
	opts := make(map[string]ChoiceOptionSpec, count)
	for i := range count {
		opts[fmt.Sprintf("opt_%d", i)] = ChoiceOptionSpec{Description: "Option", FixHint: "Fix"}
	}
	return opts
}

func makeScoreLevels(count int, withFixHint bool) []ScoreLevelSpec {
	levels := make([]ScoreLevelSpec, count)
	for i := range count {
		hint := ""
		if withFixHint {
			hint = "Fix"
		}
		levels[i] = ScoreLevelSpec{Description: "Level", FixHint: hint}
	}
	return levels
}

func floatPtr(v float64) *float64 {
	return &v
}
