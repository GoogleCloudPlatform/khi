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

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

func TestRenderSimilarityTextAndMarkdown(t *testing.T) {
	testCases := []struct {
		name         string
		rep          *rule.Report
		wantText     string
		wantMarkdown string
	}{
		{
			name: "skipped similarity status",
			rep: &rule.Report{
				Language:         "Go",
				BaseLabel:        "main",
				ProdFilesChanged: 1,
				FuncsEvaluated:   2,
				Similarity: rule.SimilaritySummary{
					Enabled:           true,
					Skipped:           true,
					SkipReason:        "no credentials",
					NewUnitsEvaluated: 1,
				},
			},
			wantText: `══ Go Quality Report (base: main) ══
  Scope:      1 prod file(s), 0 test file(s), 2 unit(s) evaluated
  Similarity: SKIPPED (no credentials)
  Verdict:    PASS (0 violations)
`,
			wantMarkdown: `## Go Quality Report (` + "`main`" + `) — ✅ PASS

| Metric | Result |
| :--- | :--- |
| Changed Scope | 1 prod file(s), 0 test file(s), 2 unit(s) |
| Function Similarity | Skipped (no credentials) |

`,
		},
		{
			name: "clean similarity status with zero matches",
			rep: &rule.Report{
				Language:         "TypeScript",
				BaseLabel:        "main",
				ProdFilesChanged: 1,
				FuncsEvaluated:   1,
				Similarity: rule.SimilaritySummary{
					Enabled:           true,
					NewUnitsEvaluated: 2,
					CorpusUnitsCount:  768,
				},
			},
			wantText: `══ TypeScript Quality Report (base: main) ══
  Scope:      1 prod file(s), 0 test file(s), 1 unit(s) evaluated
  Similarity: OK (2 new unit(s) checked against 768 corpus units, 0 similar matches)
  Verdict:    PASS (0 violations)
`,
			wantMarkdown: `## TypeScript Quality Report (` + "`main`" + `) — ✅ PASS

| Metric | Result |
| :--- | :--- |
| Changed Scope | 1 prod file(s), 0 test file(s), 1 unit(s) |
| Function Similarity | OK (2 new unit(s), 0 matches across 768 corpus units) |

`,
		},
		{
			name: "advisory matches with shared and non-shared candidates",
			rep: &rule.Report{
				Language:         "TypeScript",
				BaseLabel:        "main",
				ProdFilesChanged: 1,
				FuncsEvaluated:   2,
				Similarity: rule.SimilaritySummary{
					Enabled:           true,
					NewUnitsEvaluated: 2,
					CorpusUnitsCount:  10,
					Advisories: []rule.SimilarityAdvisory{
						{
							Index: 1,
							NewUnit: rule.SimilarityUnitRef{
								FilePath:  "web/src/app/timeline/formatter.ts",
								StartLine: 15,
								EndLine:   25,
								Name:      "formatFileSize",
							},
							Candidates: []rule.SimilarityCandidate{
								{
									Unit: rule.SimilarityUnitRef{
										FilePath:  "web/src/app/utils/byte-format-util.ts",
										StartLine: 23,
										EndLine:   37,
										Name:      "formatBytes",
										Preview: []rule.SourceLine{
											{Number: 23, Text: "export function formatBytes(bytes: number): string {"},
										},
									},
									Score:        0.961,
									SharedModule: true,
									Guidance:     `Existing helper in shared module "web/src/app/utils"; inspect and reuse directly instead of duplicating logic.`,
								},
								{
									Unit: rule.SimilarityUnitRef{
										FilePath:  "web/src/app/timeline/other.ts",
										StartLine: 40,
										EndLine:   48,
										Name:      "otherSize",
									},
									Score:        0.945,
									SharedModule: false,
									Guidance:     "Existing implementation in the same module; reuse or parameterize the existing code.",
								},
							},
						},
						{
							NewUnit: rule.SimilarityUnitRef{
								FilePath:  "web/src/app/timeline/formatter.ts",
								StartLine: 30,
								EndLine:   40,
								Name:      "dedupLabels",
							},
							Candidates: []rule.SimilarityCandidate{
								{
									Unit: rule.SimilarityUnitRef{
										FilePath:  "web/src/app/log/dedup.ts",
										StartLine: 10,
										EndLine:   20,
										Name:      "dedupRows",
									},
									Score:        0.952,
									SharedModule: false,
									Guidance:     "Similar logic exists in another module; extract a shared helper or verify intentional symmetry.",
								},
							},
						},
					},
				},
			},
			wantText: `══ TypeScript Quality Report (base: main) ══
  Scope:      1 prod file(s), 0 test file(s), 2 unit(s) evaluated
  Similarity: ADVISORY (2/2 new unit(s) matched existing code across 10 corpus units)
  Verdict:    PASS (0 violations)

── Similar Existing TypeScript Functions (Advisory: 2 new unit(s) matched) ──
  #     SCORE  NEW UNIT                                               TOP EXISTING CANDIDATE
  [S1]  0.961  web/src/app/timeline/formatter.ts:15 · formatFileSize  web/src/app/utils/byte-format-util.ts:23 · formatBytes [shared]
  [S2]  0.952  web/src/app/timeline/formatter.ts:30 · dedupLabels     web/src/app/log/dedup.ts:10 · dedupRows

  [S1] formatFileSize — web/src/app/timeline/formatter.ts:15-25
      Candidate #1: formatBytes (web/src/app/utils/byte-format-util.ts:23-37) [shared] [ast-norm cosine: 0.961]
      Action:       Existing helper in shared module "web/src/app/utils"; inspect and reuse directly instead of duplicating logic.
          23 | export function formatBytes(bytes: number): string {
      Candidate #2: otherSize (web/src/app/timeline/other.ts:40-48) [ast-norm cosine: 0.945]
      Action:       Existing implementation in the same module; reuse or parameterize the existing code.
  [S2] dedupLabels — web/src/app/timeline/formatter.ts:30-40
      Candidate #1: dedupRows (web/src/app/log/dedup.ts:10-20) [ast-norm cosine: 0.952]
      Action:       Similar logic exists in another module; extract a shared helper or verify intentional symmetry.
`,
			wantMarkdown: `## TypeScript Quality Report (` + "`main`" + `) — ✅ PASS

| Metric | Result |
| :--- | :--- |
| Changed Scope | 1 prod file(s), 0 test file(s), 2 unit(s) |
| Function Similarity | Advisory (2/2 new unit(s) matched across 10 corpus units) |

### Similar Existing TypeScript Functions (Advisory)

| New Unit | Top Existing Candidate | AST-Norm Score | Action |
| :--- | :--- | :---: | :--- |
| ` + "`formatFileSize` (`web/src/app/timeline/formatter.ts:15`)" + ` | ` + "`formatBytes` (`web/src/app/utils/byte-format-util.ts:23`)" + ` | ` + "`0.961`" + ` | Existing helper in shared module "web/src/app/utils"; inspect and reuse directly instead of duplicating logic. |
| ` + "`dedupLabels` (`web/src/app/timeline/formatter.ts:30`)" + ` | ` + "`dedupRows` (`web/src/app/log/dedup.ts:10`)" + ` | ` + "`0.952`" + ` | Similar logic exists in another module; extract a shared helper or verify intentional symmetry. |

`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var textBuf bytes.Buffer
			RenderText(&textBuf, tc.rep, gotestutil.TermStyle{Enabled: false})
			if diff := cmp.Diff(tc.wantText, textBuf.String()); diff != "" {
				t.Errorf("RenderText() mismatch (-want +got):\n%s", diff)
			}

			var mdBuf bytes.Buffer
			RenderGitHubSummary(&mdBuf, tc.rep)
			if diff := cmp.Diff(tc.wantMarkdown, mdBuf.String()); diff != "" {
				t.Errorf("RenderGitHubSummary() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRenderSimilarityANSIStyling(t *testing.T) {
	rep := &rule.Report{
		Language:  "Go",
		BaseLabel: "main",
		Similarity: rule.SimilaritySummary{
			Enabled:           true,
			NewUnitsEvaluated: 1,
			CorpusUnitsCount:  5,
			Advisories: []rule.SimilarityAdvisory{{
				Index:   1,
				NewUnit: rule.SimilarityUnitRef{FilePath: "pkg/a/a.go", StartLine: 1, EndLine: 9, Name: "Foo"},
				Candidates: []rule.SimilarityCandidate{{
					Unit:         rule.SimilarityUnitRef{FilePath: "pkg/common/b.go", StartLine: 1, EndLine: 9, Name: "Bar"},
					Score:        0.970,
					SharedModule: true,
					Guidance:     "reuse Bar",
				}},
			}},
		},
	}
	var ansiBuf bytes.Buffer
	RenderText(&ansiBuf, rep, gotestutil.TermStyle{Enabled: true})
	if !strings.Contains(ansiBuf.String(), "\x1b[") {
		t.Errorf("expected ANSI escape sequences in styled RenderText output, got %q", ansiBuf.String())
	}
}
