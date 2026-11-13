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
)

func TestRenderGitHubAnnotations(t *testing.T) {
	enforcedRep := &rule.Report{
		BaseLabel: "HEAD^1",
		DirectiveErrors: []rule.DirectiveError{
			{File: "pkg/a/a.go", Line: 3, Message: "invalid"},
			{File: "pkg/a/a.go", Line: 4, Message: "second invalid"},
		},
		Findings: []rule.Finding{
			{
				Rule:    rule.RuleFuncSurvivingMutants,
				Kind:    rule.ViolationHardLimit,
				File:    "pkg/a/a.go",
				Line:    12,
				Subject: "Calc",
				After:   1,
				Limit:   0,
				Detail:  "1 mutant survived",
			},
			{
				GlobalIndex: 9,
				Rule:        rule.RuleFuncUncoveredLines,
				Kind:        rule.ViolationHardLimit,
				File:        "pkg/a/a.go",
				Line:        20,
				Subject:     "Helper",
				After:       2,
				Limit:       0,
			},
		},
	}

	advisoryRep := &rule.Report{
		Language: "TypeScript",
		Advisory: true,
		Findings: []rule.Finding{
			{
				GlobalIndex: 7,
				Rule:        rule.RuleFuncUncoveredLines,
				Kind:        rule.ViolationHardLimit,
				File:        "web/src/app/x.ts",
				Line:        0,
				Subject:     "<file>",
				After:       1,
				Limit:       0,
			},
			{
				Rule:      rule.RuleFuncCognitive,
				Kind:      rule.ViolationRatchetRegression,
				File:      "web/src/app/x.ts",
				Line:      9,
				Subject:   "render",
				Before:    16,
				HasBefore: true,
				After:     18,
				Limit:     15,
				Detail:    "regressed",
			},
		},
	}

	testCases := []struct {
		name      string
		rep       *rule.Report
		maxAnnots int
		want      string
	}{
		{
			name:      "zero cap returns without emitting",
			rep:       enforcedRep,
			maxAnnots: 0,
			want:      "",
		},
		{
			name:      "cap truncates directive errors",
			rep:       enforcedRep,
			maxAnnots: 1,
			want:      "::error file=pkg/a/a.go,line=3,title=Invalid CODEQUALITY_ALLOW Directive::invalid\n",
		},
		{
			name:      "cap truncates findings",
			rep:       enforcedRep,
			maxAnnots: 3,
			want: `::error file=pkg/a/a.go,line=3,title=Invalid CODEQUALITY_ALLOW Directive::invalid
::error file=pkg/a/a.go,line=4,title=Invalid CODEQUALITY_ALLOW Directive::second invalid
::error file=pkg/a/a.go,line=12,title=Go Quality [Q1] (func_surviving_mutants)::Calc [HARD_LIMIT]: 1 (<= 0) — 1 mutant survived
`,
		},
		{
			name:      "advisory report emits all warnings",
			rep:       advisoryRep,
			maxAnnots: 10,
			want: `::warning file=web/src/app/x.ts,line=1,title=TypeScript Quality [Q7] (func_uncovered_lines)::<file> [HARD_LIMIT]: 1 (<= 0)
::warning file=web/src/app/x.ts,line=9,title=TypeScript Quality [Q2] (func_cognitive)::render [RATCHET_REGRESSION]: 16 -> 18 (<= 15) — regressed
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			RenderGitHubAnnotations(&buf, tc.rep, tc.maxAnnots)
			if diff := cmp.Diff(tc.want, buf.String()); diff != "" {
				t.Errorf("RenderGitHubAnnotations() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRenderGitHubSummary(t *testing.T) {
	testCases := []struct {
		name string
		rep  *rule.Report
		want string
	}{
		{
			name: "clean pass without optional metrics",
			rep: &rule.Report{
				BaseLabel:        "HEAD^1",
				ProdFilesChanged: 2,
				TestFilesChanged: 1,
				FuncsEvaluated:   4,
			},
			want: `## Go Quality Report (` + "`HEAD^1`" + `) — ✅ PASS

| Metric | Result |
| :--- | :--- |
| Changed Scope | 2 prod file(s), 1 test file(s), 4 unit(s) |

`,
		},
		{
			name: "enforced failure with metrics directives and finding evidence",
			rep: &rule.Report{
				BaseLabel:        "HEAD^1",
				ProdFilesChanged: 1,
				FuncsEvaluated:   2,
				Coverage:         rule.CoverageSummary{Enabled: true, PatchCovered: 1, PatchExecutable: 2},
				Mutation:         rule.MutationSummary{Enabled: true, Killed: 1, Survived: 1},
				DirectiveErrors: []rule.DirectiveError{
					{File: "pkg/a/a.go", Line: 3, Message: "invalid"},
				},
				Findings: []rule.Finding{
					{
						Rule:             rule.RuleFuncSurvivingMutants,
						Kind:             rule.ViolationHardLimit,
						File:             "pkg/a/a.go",
						Line:             12,
						Subject:          "Calc",
						After:            1,
						Limit:            0,
						Detail:           "1 mutant survived",
						SurvivingMutants: []rule.MutantDetail{{Operator: rule.OperatorROR, Line: 12, OriginalLine: "a < b", MutatedLine: "a <= b"}},
						UncoveredSpans:   []rule.UncoveredSpan{{StartLine: 15, EndLine: 15, Lines: []rule.SourceLine{{Number: 15, Text: "return 0"}}}},
						RedundantTests:   []rule.RedundantDetail{{Kind: rule.RedundantNoOutputCheck, Line: 20, TestName: "TestCalc/weak", DominatedBy: "TestCalc/full", DominatedByFile: "pkg/a/a_test.go", DominatedByLine: 10}},
					},
				},
			},
			want: `## Go Quality Report (` + "`HEAD^1`" + `) — ❌ FAIL (1 violation(s), 1 directive error(s))

| Metric | Result |
| :--- | :--- |
| Changed Scope | 1 prod file(s), 0 test file(s), 2 unit(s) |
| Patch Coverage | 50.0% (1/2 lines) |
| Mutation Score | 50.0% (1/2 killed, 1 survived) |

### Invalid or Unused ` + "`CODEQUALITY_ALLOW`" + ` Directives

- ` + "`pkg/a/a.go:3`" + ` — invalid

### Quality Rule Violations

| # | Rule | Status | Value (Limit) | Location | Subject |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ` + "`[Q1]`" + ` | ` + "`func_surviving_mutants`" + ` | ` + "`HARD_LIMIT`" + ` | ` + "`1 (<= 0)`" + ` | ` + "`pkg/a/a.go:12`" + ` | ` + "`Calc`" + ` |

<details>
<summary><strong>[Q1] func_surviving_mutants</strong> — <code>pkg/a/a.go:12</code> (<code>Calc</code>)</summary>

- **Why**: Mutating operators or branches in this touched function did not cause any unit test to fail.
- **Fix**: Delete redundant guards first, or strengthen assertions (cmp.Diff on full outputs and boundary inputs).
- **Detail**: 1 mutant survived

` + "```diff" + `
- a < b
+ a <= b
` + "```" + `

` + "```go" + `
  15 | return 0
` + "```" + `
- ` + "`[NO OUTPUT CHECK]`" + ` ` + "`TestCalc/weak`" + ` (L20) dominated by ` + "`TestCalc/full`" + ` (` + "`pkg/a/a_test.go:10`" + `)

</details>

`,
		},
		{
			name: "advisory TypeScript summary with unknown rule and empty detail",
			rep: &rule.Report{
				Language:         "TypeScript",
				BaseLabel:        "main",
				Advisory:         true,
				ProdFilesChanged: 1,
				FuncsEvaluated:   1,
				Findings: []rule.Finding{{
					GlobalIndex:    5,
					Rule:           rule.RuleID("custom_ts_rule"),
					Kind:           rule.ViolationRatchetRegression,
					File:           "web/src/app/x.ts",
					Line:           9,
					Subject:        "<file>",
					Before:         10,
					HasBefore:      true,
					After:          12,
					Limit:          8,
					UncoveredSpans: []rule.UncoveredSpan{{StartLine: 9, EndLine: 9, Lines: []rule.SourceLine{{Number: 9, Text: "return 0;"}}}},
				}},
			},
			want: `## TypeScript Quality Report (` + "`main`" + `) — ⚠️ ADVISORY (1 finding(s), 0 directive error(s))

| Metric | Result |
| :--- | :--- |
| Changed Scope | 1 prod file(s), 0 test file(s), 1 unit(s) |

### Quality Rule Violations

| # | Rule | Status | Value (Limit) | Location | Subject |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ` + "`[Q5]`" + ` | ` + "`custom_ts_rule`" + ` | ` + "`RATCHET_REGRESSION`" + ` | ` + "`10 -> 12 (<= 8)`" + ` | ` + "`web/src/app/x.ts:9`" + ` | ` + "`<file>`" + ` |

<details>
<summary><strong>[Q5] custom_ts_rule</strong> — <code>web/src/app/x.ts:9</code> (<code><file></code>)</summary>


` + "```ts" + `
   9 | return 0;
` + "```" + `

</details>

`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			RenderGitHubSummary(&buf, tc.rep)
			if diff := cmp.Diff(tc.want, buf.String()); diff != "" {
				t.Errorf("RenderGitHubSummary() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
