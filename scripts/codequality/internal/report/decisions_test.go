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

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func TestRenderDecisionsTextGitHubAndAnnotations(t *testing.T) {
	testCases := []struct {
		name           string
		rep            *rule.Report
		style          gotestutil.TermStyle
		maxAnnotations int
		wantText       string
		wantMD         string
		wantAnnot      string
	}{
		// Renders skip status in text and Markdown without emitting GitHub Actions annotations when credentials are absent.
		{
			name: "skipped report renders skip reason cleanly",
			rep: &rule.Report{
				Language:  "Decisions",
				BaseLabel: "origin/main",
				Decisions: rule.DecisionSummary{
					Enabled:    true,
					Skipped:    true,
					SkipReason: "no API key configured",
				},
			},
			maxAnnotations: 20,
			wantText: `══ Decision Rules Report (base: origin/main) ══
  Status:     SKIPPED (no API key configured)
`,
			wantMD: `## Decision Rules Report (` + "`origin/main`" + `) — ⚪ SKIPPED

| Metric | Result |
| :--- | :--- |
| Status | Skipped (no API key configured) |

`,
			wantAnnot: "",
		},
		// Renders PASS status with zero findings, omitting finding tables and workflow annotations.
		{
			name: "passing report renders zero findings",
			rep: &rule.Report{
				Language:  "Decisions",
				BaseLabel: "origin/main",
				Decisions: rule.DecisionSummary{
					Enabled:            true,
					RulesEvaluated:     1,
					TargetsEvaluated:   2,
					QuestionsEvaluated: 14,
					CacheHits:          2,
				},
			},
			maxAnnotations: 20,
			wantText: `══ Decision Rules Report (base: origin/main) ══
  Scope:      1 rule(s), 2 target(s), 14 question(s) evaluated (cache: 2 hit(s), 0 API call(s))
  Verdict:    PASS (0 findings)
`,
			wantMD: `## Decision Rules Report (` + "`origin/main`" + `) — ✅ PASS

| Metric | Result |
| :--- | :--- |
| Evaluated Scope | 1 rule(s), 2 target(s), 14 question(s) |
| Cache & API | 2 cache hit(s), 0 live API call(s) |

`,
			wantAnnot: "",
		},
		// Renders FAIL status, fallback/explicit indices, single-vs-range spans, and capped workflow annotations for mixed findings.
		{
			name: "enforced blocking and advisory findings render tables details and capped annotations",
			rep: &rule.Report{
				Language:  "Decisions",
				BaseLabel: "origin/main",
				Decisions: rule.DecisionSummary{
					Enabled:            true,
					RulesEvaluated:     1,
					TargetsEvaluated:   1,
					QuestionsEvaluated: 4,
					APICalls:           1,
					Findings: []rule.DecisionItemFinding{
						{
							RuleID:     "pkg_doc_alignment",
							QuestionID: "section_claim",
							Blocking:   true,
							File:       "pkg/demo/README.md",
							Line:       10,
							EndLine:    18,
							Subject:    "Overview",
							Outcome:    "stale_symbol (p=0.88)",
							Why:        "Symbol removed.",
							Fix:        "Update Overview.",
						},
						{
							Index:      7,
							RuleID:     "pkg_doc_alignment",
							QuestionID: "unit",
							Blocking:   false,
							File:       "pkg/demo/demo.go",
							Line:       5,
							EndLine:    5,
							Subject:    "NewTask",
							Outcome:    "p_true=0.05",
							Why:        "Undocumented task.",
							Fix:        "Document NewTask.",
						},
					},
				},
			},
			maxAnnotations: 1,
			wantText: `══ Decision Rules Report (base: origin/main) ══
  Scope:      1 rule(s), 1 target(s), 4 question(s) evaluated (cache: 0 hit(s), 1 API call(s))
  Verdict:    FAIL (1 blocking, 1 advisory)

── Decision Rule Findings (2) ──
  #     RULE / QUESTION                  SEVERITY  OUTCOME                LOCATION
  [D1]  pkg_doc_alignment/section_claim  BLOCKING  stale_symbol (p=0.88)  pkg/demo/README.md:10-18 · Overview
  [D7]  pkg_doc_alignment/unit           ADVISORY  p_true=0.05            pkg/demo/demo.go:5 · NewTask

  [D1] pkg_doc_alignment/section_claim — pkg/demo/README.md:10-18 ("Overview")
      Outcome: stale_symbol (p=0.88)
      Why:     Symbol removed.
      Fix:     Update Overview.
  [D7] pkg_doc_alignment/unit — pkg/demo/demo.go:5 ("NewTask")
      Outcome: p_true=0.05
      Why:     Undocumented task.
      Fix:     Document NewTask.
`,
			wantMD: `## Decision Rules Report (` + "`origin/main`" + `) — ❌ FAIL (1 blocking, 1 advisory)

| Metric | Result |
| :--- | :--- |
| Evaluated Scope | 1 rule(s), 1 target(s), 4 question(s) |
| Cache & API | 0 cache hit(s), 1 live API call(s) |

### Decision Rule Findings

| # | Rule / Question | Severity | Outcome | Location | Subject |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ` + "`[D1]`" + ` | ` + "`pkg_doc_alignment/section_claim`" + ` | ` + "`Blocking`" + ` | ` + "`stale_symbol (p=0.88)`" + ` | ` + "`pkg/demo/README.md:10-18`" + ` | ` + "`Overview`" + ` |
| ` + "`[D7]`" + ` | ` + "`pkg_doc_alignment/unit`" + ` | ` + "`Advisory`" + ` | ` + "`p_true=0.05`" + ` | ` + "`pkg/demo/demo.go:5`" + ` | ` + "`NewTask`" + ` |

<details>
<summary><strong>[D1] pkg_doc_alignment/section_claim</strong> — <code>pkg/demo/README.md:10-18</code> (<code>Overview</code>)</summary>

- **Outcome**: ` + "`stale_symbol (p=0.88)`" + `
- **Why**: Symbol removed.
- **Fix**: Update Overview.

</details>

<details>
<summary><strong>[D7] pkg_doc_alignment/unit</strong> — <code>pkg/demo/demo.go:5</code> (<code>NewTask</code>)</summary>

- **Outcome**: ` + "`p_true=0.05`" + `
- **Why**: Undocumented task.
- **Fix**: Document NewTask.

</details>

`,
			wantAnnot: "::error file=pkg/demo/README.md,line=10,title=Decision [D1] (pkg_doc_alignment/section_claim)::Overview [stale_symbol (p=0.88)] — Why: Symbol removed. | Fix: Update Overview.\n",
		},
		// Downgrades blocking findings in advisory mode to advisory verdict, warning annotations, and styled ANSI output with zero-line clamping.
		{
			name: "advisory mode downgrades blocking findings and emits ANSI styling",
			rep: &rule.Report{
				Language:  "Decisions",
				BaseLabel: "main",
				Advisory:  true,
				Decisions: rule.DecisionSummary{
					Enabled:            true,
					RulesEvaluated:     1,
					TargetsEvaluated:   2,
					QuestionsEvaluated: 2,
					CacheHits:          1,
					APICalls:           1,
					Findings: []rule.DecisionItemFinding{
						{
							RuleID:     "go_doc",
							QuestionID: "claim",
							Blocking:   true,
							File:       "pkg/a.go",
							Line:       0,
							EndLine:    0,
							Subject:    "Foo",
							Outcome:    "false",
							Why:        "Mismatch.",
							Fix:        "Fix Foo.",
						},
						{
							RuleID:     "go_doc",
							QuestionID: "name",
							Blocking:   false,
							File:       "pkg/b.go",
							Line:       4,
							EndLine:    4,
							Subject:    "Bar",
							Outcome:    "unclear",
							Why:        "Vague.",
							Fix:        "Rename Bar.",
						},
					},
				},
			},
			style:          gotestutil.TermStyle{Enabled: true},
			maxAnnotations: 10,
			wantText: "\x1b[1;36m══ Decision Rules Report (base: main) ══\x1b[0m\n" +
				"  Scope:      1 rule(s), 2 target(s), 2 question(s) evaluated (cache: 1 hit(s), 1 API call(s))\n" +
				"  Verdict:    \x1b[33mADVISORY (2 finding(s))\x1b[0m\n\n" +
				"\x1b[1;33m── Decision Rule Findings (2) ──\x1b[0m\n" +
				"  \x1b[1m#\x1b[0m     \x1b[1mRULE / QUESTION\x1b[0m  \x1b[1mSEVERITY\x1b[0m  \x1b[1mOUTCOME\x1b[0m  \x1b[1mLOCATION\x1b[0m\n" +
				"  \x1b[1;31m[D1]\x1b[0m  \x1b[1mgo_doc/claim\x1b[0m     \x1b[31mBLOCKING\x1b[0m  false    pkg/a.go:1 · Foo\n" +
				"  \x1b[1;33m[D2]\x1b[0m  \x1b[1mgo_doc/name\x1b[0m      \x1b[33mADVISORY\x1b[0m  unclear  pkg/b.go:4 · Bar\n\n" +
				"  \x1b[1;31m[D1]\x1b[0m \x1b[1mgo_doc\x1b[0m/claim — pkg/a.go:1 (\"Foo\")\n" +
				"      Outcome: false\n" +
				"      Why:     \x1b[2mMismatch.\x1b[0m\n" +
				"      Fix:     Fix Foo.\n" +
				"  \x1b[1;33m[D2]\x1b[0m \x1b[1mgo_doc\x1b[0m/name — pkg/b.go:4 (\"Bar\")\n" +
				"      Outcome: unclear\n" +
				"      Why:     \x1b[2mVague.\x1b[0m\n" +
				"      Fix:     Rename Bar.\n",
			wantMD: `## Decision Rules Report (` + "`main`" + `) — ⚠️ ADVISORY (2 finding(s))

| Metric | Result |
| :--- | :--- |
| Evaluated Scope | 1 rule(s), 2 target(s), 2 question(s) |
| Cache & API | 1 cache hit(s), 1 live API call(s) |

### Decision Rule Findings

| # | Rule / Question | Severity | Outcome | Location | Subject |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ` + "`[D1]`" + ` | ` + "`go_doc/claim`" + ` | ` + "`Blocking`" + ` | ` + "`false`" + ` | ` + "`pkg/a.go:1`" + ` | ` + "`Foo`" + ` |
| ` + "`[D2]`" + ` | ` + "`go_doc/name`" + ` | ` + "`Advisory`" + ` | ` + "`unclear`" + ` | ` + "`pkg/b.go:4`" + ` | ` + "`Bar`" + ` |

<details>
<summary><strong>[D1] go_doc/claim</strong> — <code>pkg/a.go:1</code> (<code>Foo</code>)</summary>

- **Outcome**: ` + "`false`" + `
- **Why**: Mismatch.
- **Fix**: Fix Foo.

</details>

<details>
<summary><strong>[D2] go_doc/name</strong> — <code>pkg/b.go:4</code> (<code>Bar</code>)</summary>

- **Outcome**: ` + "`unclear`" + `
- **Why**: Vague.
- **Fix**: Rename Bar.

</details>

`,
			wantAnnot: `::warning file=pkg/a.go,line=1,title=Decision [D1] (go_doc/claim)::Foo [false] — Why: Mismatch. | Fix: Fix Foo.
::warning file=pkg/b.go,line=4,title=Decision [D2] (go_doc/name)::Bar [unclear] — Why: Vague. | Fix: Rename Bar.
`,
		},
		// Renders advisory verdict and warning workflow annotation when enforced mode encounters only non-blocking findings.
		{
			name: "enforced mode with advisory-only findings reports advisory verdict",
			rep: &rule.Report{
				Language:  "Decisions",
				BaseLabel: "main",
				Decisions: rule.DecisionSummary{
					Enabled:            true,
					RulesEvaluated:     1,
					TargetsEvaluated:   1,
					QuestionsEvaluated: 1,
					CacheHits:          1,
					Findings: []rule.DecisionItemFinding{
						{
							RuleID:     "go_doc",
							QuestionID: "hint",
							Blocking:   false,
							File:       "pkg/c.go",
							Line:       12,
							EndLine:    15,
							Subject:    "Baz",
							Outcome:    "vague",
							Why:        "Add detail.",
							Fix:        "Expand doc.",
						},
					},
				},
			},
			maxAnnotations: 5,
			wantText: `══ Decision Rules Report (base: main) ══
  Scope:      1 rule(s), 1 target(s), 1 question(s) evaluated (cache: 1 hit(s), 0 API call(s))
  Verdict:    ADVISORY (1 finding(s))

── Decision Rule Findings (1) ──
  #     RULE / QUESTION  SEVERITY  OUTCOME  LOCATION
  [D1]  go_doc/hint      ADVISORY  vague    pkg/c.go:12-15 · Baz

  [D1] go_doc/hint — pkg/c.go:12-15 ("Baz")
      Outcome: vague
      Why:     Add detail.
      Fix:     Expand doc.
`,
			wantMD: `## Decision Rules Report (` + "`main`" + `) — ⚠️ ADVISORY (1 finding(s))

| Metric | Result |
| :--- | :--- |
| Evaluated Scope | 1 rule(s), 1 target(s), 1 question(s) |
| Cache & API | 1 cache hit(s), 0 live API call(s) |

### Decision Rule Findings

| # | Rule / Question | Severity | Outcome | Location | Subject |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ` + "`[D1]`" + ` | ` + "`go_doc/hint`" + ` | ` + "`Advisory`" + ` | ` + "`vague`" + ` | ` + "`pkg/c.go:12-15`" + ` | ` + "`Baz`" + ` |

<details>
<summary><strong>[D1] go_doc/hint</strong> — <code>pkg/c.go:12-15</code> (<code>Baz</code>)</summary>

- **Outcome**: ` + "`vague`" + `
- **Why**: Add detail.
- **Fix**: Expand doc.

</details>

`,
			wantAnnot: "::warning file=pkg/c.go,line=12,title=Decision [D1] (go_doc/hint)::Baz [vague] — Why: Add detail. | Fix: Expand doc.\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var txtBuf, mdBuf, annBuf bytes.Buffer

			RenderText(&txtBuf, tc.rep, tc.style)
			RenderGitHubSummary(&mdBuf, tc.rep)
			RenderGitHubAnnotations(&annBuf, tc.rep, tc.maxAnnotations)

			if diff := cmp.Diff(tc.wantText, txtBuf.String()); diff != "" {
				t.Errorf("RenderText() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantMD, mdBuf.String()); diff != "" {
				t.Errorf("RenderGitHubSummary() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantAnnot, annBuf.String()); diff != "" {
				t.Errorf("RenderGitHubAnnotations() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
