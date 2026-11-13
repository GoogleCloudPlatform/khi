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
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

func TestRenderText(t *testing.T) {
	testCases := []struct {
		name string
		rep  *rule.Report
		want string
	}{
		{
			name: "renders clean fast pass report",
			rep: &rule.Report{
				BaseLabel:        "trunk()",
				FastMode:         true,
				ProdFilesChanged: 2,
				FuncsEvaluated:   5,
			},
			want: `══ Go Quality Report (base: trunk()) [fast AST mode] ══
  Scope:      2 prod file(s), 0 test file(s), 5 unit(s) evaluated
  Verdict:    PASS (0 violations)
`,
		},
		{
			name: "renders advisory report with baseline metrics and boundary hotspots",
			rep: &rule.Report{
				Language:         "TypeScript",
				BaseLabel:        "main",
				Advisory:         true,
				ProdFilesChanged: 1,
				FuncsEvaluated:   2,
				Coverage:         rule.CoverageSummary{Enabled: true, PatchCovered: 3, PatchExecutable: 4},
				Mutation:         rule.MutationSummary{Enabled: true, Killed: 4, Survived: 0},
				Findings: []rule.Finding{
					{
						Rule:    rule.RuleID("custom_ts_rule"),
						Kind:    rule.ViolationHardLimit,
						File:    "web/src/a.ts",
						Line:    5,
						Subject: "<file>",
						After:   2,
						Limit:   0,
					},
					{
						GlobalIndex: 7,
						Rule:        rule.RuleFuncUncoveredLines,
						Kind:        rule.ViolationHardLimit,
						File:        "web/src/b.ts",
						Line:        9,
						Subject:     "render",
						After:       1,
						Limit:       0,
						Hotspots: []rule.Hotspot{
							{Line: 1, Summary: "+1"},
							{Line: 2, Summary: "+1"},
							{Line: 3, Summary: "+1"},
							{Line: 4, Summary: "+1"},
							{Line: 5, Summary: "+1"},
						},
					},
				},
			},
			want: `══ TypeScript Quality Report (base: main) ══
  Scope:      1 prod file(s), 0 test file(s), 2 unit(s) evaluated
  Coverage:   Patch 75.0% (3/4 changed executable lines)
  Mutation:   100.0% (4/4 killed, 0 survived)
  Verdict:    ADVISORY (2 finding(s), 0 directive error(s))

── Quality Rule Violations (2) ──
  #     RULE                  STATUS      VALUE (LIMIT)  TARGET
  [Q1]  custom_ts_rule        HARD_LIMIT  2 (<= 0)       web/src/a.ts:5 · <file>
  [Q7]  func_uncovered_lines  HARD_LIMIT  1 (<= 0)       web/src/b.ts:9 · render

  [Q1] custom_ts_rule — web/src/a.ts:5 (<file>) [HARD_LIMIT: 2 (<= 0)]
  [Q7] func_uncovered_lines — web/src/b.ts:9 (render) [HARD_LIMIT: 1 (<= 0)]
      Why:    Substantive branches in touched functions were never executed by unit tests (boilerplate error guards are already excluded).
      Fix:    Add focused table-driven test cases covering the unexecuted branch, or delete unreachable production code.
      Hotspots: L1 +1; L2 +1; L3 +1; L4 +1; L5 +1
`,
		},
		{
			name: "renders enforced failure with full metrics evidence and suppressions",
			rep: &rule.Report{
				BaseLabel:        "origin/main",
				ProdFilesChanged: 1,
				TestFilesChanged: 1,
				FuncsEvaluated:   3,
				Coverage:         rule.CoverageSummary{Enabled: true, PatchCovered: 4, PatchExecutable: 5, HasOverall: true, OverallCovered: 8, OverallStatements: 10},
				Mutation:         rule.MutationSummary{Enabled: true, Killed: 3, Survived: 1, Suppressed: 1, SkippedBudget: 2},
				DirectiveErrors:  []rule.DirectiveError{{File: "pkg/a/a.go", Line: 2, Message: "bad directive"}},
				Findings: []rule.Finding{
					{
						GlobalIndex:      1,
						Rule:             rule.RuleFuncCognitive,
						Kind:             rule.ViolationRatchetRegression,
						File:             "pkg/a/a.go",
						Line:             10,
						Subject:          "Process",
						Before:           16,
						HasBefore:        true,
						After:            18,
						Limit:            15,
						Detail:           "nested loop",
						Hotspots:         []rule.Hotspot{{Line: 11, Delta: 1, Summary: "+1 (if)"}, {Line: 12, Delta: 2, Summary: "+2"}, {Line: 13, Delta: 3, Summary: "+3"}, {Line: 14, Delta: 4, Summary: "+4"}, {Line: 15, Delta: 5, Summary: "+5"}, {Line: 16, Delta: 3, Summary: "+3"}},
						UncoveredSpans:   []rule.UncoveredSpan{{StartLine: 20, EndLine: 20, Lines: []rule.SourceLine{{Number: 20, Text: "return -1"}}}},
						SurvivingMutants: []rule.MutantDetail{{Operator: rule.OperatorROR, Line: 22, OriginalLine: "if x >= 0 {", MutatedLine: "if x > 0 {"}},
						RedundantTests:   []rule.RedundantDetail{{Kind: rule.RedundantSamePath, Line: 30, TestName: "TestA/dup", DominatedBy: "TestA/base", DominatedByFile: "pkg/a/a_test.go", DominatedByLine: 25, CandidateKills: 1, DominatorKills: 2}},
					},
				},
				Suppressed: []rule.SuppressedItem{{Rule: rule.RuleFuncParams, File: "pkg/a/a.go", Line: 40, Subject: "Handler", Category: rule.CategoryProtocol, Reason: "rpc contract"}},
			},
			want: `══ Go Quality Report (base: origin/main) ══
  Scope:      1 prod file(s), 1 test file(s), 3 unit(s) evaluated
  Coverage:   Patch 80.0% (4/5 changed executable lines)  |  Overall 80.0% (8/10 stmts)
  Mutation:   75.0% (3/4 killed, 1 survived, 1 suppressed, 2 budget-skipped)
  Verdict:    FAIL (1 violation(s), 1 directive error(s))

── Invalid or Unused CODEQUALITY_ALLOW Directives (1) ──
  pkg/a/a.go:2: bad directive

── Quality Rule Violations (1) ──
  #     RULE            STATUS              VALUE (LIMIT)     TARGET
  [Q1]  func_cognitive  RATCHET_REGRESSION  16 -> 18 (<= 15)  pkg/a/a.go:10 · Process

  [Q1] func_cognitive — pkg/a/a.go:10 (Process) [RATCHET_REGRESSION: 16 -> 18 (<= 15)]
      Detail: nested loop
      Why:    Deeply nested branching and mixed boolean chains exceed human and agent reasoning capacity.
      Fix:    Use early-return guard clauses, flatten switch dispatch, or extract cohesive domain methods (keep params <= 5).
      Hotspots: L11 +1 (if); L12 +2; L13 +3; L14 +4; L15 +5 (+1 more)
      Uncovered L20-20:
          20 | return -1
      Mutant L22 [ROR — boundary/comparison]:
        - if x >= 0 {
        + if x > 0 {
      [SAME PATH] L30 TestA/dup -> dominated by TestA/base (pkg/a/a_test.go:25, kills 1 vs 2)

── Suppressed via CODEQUALITY_ALLOW (1) ──
  pkg/a/a.go:40 (Handler) [func_params / protocol]: rpc contract
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			RenderText(&buf, tc.rep, gotestutil.TermStyle{})
			if diff := cmp.Diff(tc.want, buf.String()); diff != "" {
				t.Errorf("RenderText() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
