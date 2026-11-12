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

func TestRenderTextReportEmpty(t *testing.T) {
	testCases := []struct {
		name     string
		coverage *CoverageReport
		want     string
	}{
		{
			name:     "nil coverage omits overall line",
			coverage: nil,
			want:     "Go Differential Test Report (base: trunk()): no executable Go production lines changed.\n",
		},
		{
			name: "disabled overall coverage omits overall line",
			coverage: &CoverageReport{
				BaseLabel:          "trunk()",
				OverallStatements:  100,
				OverallCovered:     75,
				HasOverallCoverage: false,
			},
			want: "Go Differential Test Report (base: trunk()): no executable Go production lines changed.\n",
		},
		{
			name: "zero overall statements omits overall line",
			coverage: &CoverageReport{
				BaseLabel:          "trunk()",
				OverallStatements:  0,
				HasOverallCoverage: true,
			},
			want: "Go Differential Test Report (base: trunk()): no executable Go production lines changed.\n",
		},
		{
			name: "positive overall coverage renders statement summary",
			coverage: &CoverageReport{
				BaseLabel:          "trunk()",
				OverallStatements:  100,
				OverallCovered:     75,
				HasOverallCoverage: true,
			},
			want: `Go Differential Test Report (base: trunk()): no executable Go production lines changed.
Overall Go Statement Coverage: 75.0% (75/100 statements)
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			report := BuildMutationReport("trunk()", nil, nil, nil)
			if tc.coverage != nil {
				AttachLineCoverage(report, tc.coverage)
			}
			var buf bytes.Buffer
			RenderTextReport(&buf, report, gotestutil.TermStyle{Enabled: false})
			if diff := cmp.Diff(tc.want, buf.String()); diff != "" {
				t.Errorf("RenderTextReport() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRenderTextReportWithSurvivingWarningsAndGroupedRedundantTests(t *testing.T) {
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

	redundant := []RedundantTestFinding{
		{
			GlobalIndex:     1,
			Kind:            RedundantDuplicate,
			File:            "pkg/scene/palette_test.go",
			Line:            28,
			TestName:        "TestPalette/default_scale",
			DominatedBy:     "TestPalette/boundary_scale",
			DominatedByFile: "pkg/scene/palette_test.go",
			DominatedByLine: 14,
			CandidateKills:  1,
			DominatorKills:  2,
		},
		{
			GlobalIndex:     2,
			Kind:            RedundantDuplicate,
			File:            "pkg/scene/palette_test.go",
			Line:            42,
			TestName:        "TestPalette/preset_red",
			DominatedBy:     "TestPalette/preset_blue",
			DominatedByFile: "pkg/scene/palette_test.go",
			DominatedByLine: 35,
			CandidateKills:  3,
			DominatorKills:  3,
		},
		{
			GlobalIndex:     3,
			Kind:            RedundantDuplicate,
			File:            "pkg/scene/palette_test.go",
			Line:            49,
			TestName:        "TestPalette/preset_green",
			DominatedBy:     "TestPalette/preset_blue",
			DominatedByFile: "pkg/scene/palette_test.go",
			DominatedByLine: 35,
			CandidateKills:  3,
			DominatorKills:  3,
		},
	}

	report := BuildMutationReport("trunk()", plans, warnings, redundant)
	AttachLineCoverage(report, &CoverageReport{
		BaseLabel:            "trunk()",
		PatchExecutableLines: 8,
		PatchCoveredLines:    6,
		Files: []FileCoverageReport{
			{
				File:                   "pkg/scene/palette.go",
				ChangedExecutableLines: 6,
				ChangedCoveredLines:    5,
				UncoveredRanges:        []LineRange{{StartLine: 44, EndLine: 44}},
				UncoveredSpans: []UncoveredSpan{
					{
						File:            "pkg/scene/palette.go",
						StartLine:       44,
						EndLine:         44,
						ExecutableLines: 1,
						Lines:           []SourceLine{{Number: 44, Text: "return n * 2"}},
					},
				},
			},
			{
				File:                   "pkg/scene/wiring.go",
				ChangedExecutableLines: 2,
				ChangedCoveredLines:    1,
				UncoveredRanges:        []LineRange{{StartLine: 12, EndLine: 12}},
				UncoveredSpans: []UncoveredSpan{
					{
						File:            "pkg/scene/wiring.go",
						StartLine:       12,
						EndLine:         12,
						ExecutableLines: 1,
						Lines: []SourceLine{
							{Number: 11, Text: "if err != nil {"},
							{Number: 12, Text: "    return err"},
						},
					},
				},
			},
		},
	})

	var buf bytes.Buffer
	RenderTextReport(&buf, report, gotestutil.TermStyle{Enabled: false})

	want := `══ Go Differential Test Report (base: trunk()) ══
  Patch Coverage: 75.0% (6 / 8 changed lines covered)
  Mutation Score: 75.0% (3 killed / 4 tested)  |  Uncovered Mutants: 1  |  Suppressed: 1 (observability: 1)

  FILE                    PATCH COV     MUTATION  KILLED  SURVIVED  SUPPRESSED  UNCOVERED LINES
  pkg/scene/palette.go  83.3% (5/6)  75.0% (3/4)       3         1           1  44
  pkg/scene/wiring.go   50.0% (1/2)          n/a       0         0           0  12

  Redundant Tests Summary (3):
  ID    LOCATION                      REDUNDANT TEST             KIND       COVERED BY
  [R1]  pkg/scene/palette_test.go:28  TestPalette/default_scale  SAME PATH  line 14 (boundary_scale)
  [R2]  pkg/scene/palette_test.go:42  TestPalette/preset_red     SAME PATH  line 35 (preset_blue)
  [R3]  pkg/scene/palette_test.go:49  TestPalette/preset_green   SAME PATH  line 35 (preset_blue)

── Invalid NOMUTANTS Directives (1) ──
  pkg/scene/palette.go:50: // NOMUTANTS requires a category and reason: // NOMUTANTS(<category>): <reason> (allowed categories: equivalent, perf, observability, external)

── Surviving Mutants (1) — tests still passed when logic changed ──
  pkg/scene/palette.go
    [M1] line 40  (boundary/comparison: ROR)
      - if idx < maxColors {
      + if idx <= maxColors {

  Tip: Delete redundant/unreachable branches first, or strengthen assertions (cmp.Diff / boundary inputs) rather than adding mock-heavy tests.

── Uncovered Changed Code (2 block(s)) — never executed by tests ──
  pkg/scene/palette.go
    lines 44
        44 | return n * 2

  pkg/scene/wiring.go
    lines 12
        11 | if err != nil {
        12 |     return err

── Redundant Tests in Diff (3) — already covered by another test ──
  pkg/scene/palette_test.go  (in TestPalette)
    [R1] line 28: default_scale  [SAME PATH]
         Covered by line 14: boundary_scale
         Why: Runs the exact same if/else path, and catches fewer logic bugs (1 vs 2) than line 14.
         Fix: Delete this duplicate case unless it documents a distinct regression input.
    Covered by line 35: preset_blue
      - [R2] line 42: preset_red  [SAME PATH]
      - [R3] line 49: preset_green  [SAME PATH]
      Why: All 2 cases run the exact same code path and check the exact same 3 logic condition(s) as line 35 (no new branches tested).
      Fix: Delete duplicate cases unless keeping a distinct regression input.
`
	if diff := cmp.Diff(want, buf.String()); diff != "" {
		t.Errorf("RenderTextReport(plain) mismatch (-want +got):\n%s", diff)
	}

	var colorBuf bytes.Buffer
	RenderTextReport(&colorBuf, report, gotestutil.TermStyle{Enabled: true})
	gotColor := colorBuf.String()
	for _, wantSeq := range []string{
		"\x1b[1;36m══ Go Differential Test Report (base: trunk()) ══\x1b[0m",
		"\x1b[1;31m<\x1b[0m",
		"\x1b[1;32m<=\x1b[0m",
		"\x1b[33m[SAME PATH]\x1b[0m",
	} {
		if !strings.Contains(gotColor, wantSeq) {
			t.Errorf("RenderTextReport(color) missing %q in:\n%s", wantSeq, gotColor)
		}
	}
}
