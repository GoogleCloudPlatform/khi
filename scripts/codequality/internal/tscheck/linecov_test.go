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

package tscheck

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

func TestAnalyzeTSCoverage(t *testing.T) {
	sampleSrc := `export interface SampleViewModel {
  readonly label: string;
}

export class SamplePresenter {
  // Uncalled computed callback inside initialized class property.
  readonly summary = computed(() => { // comment
    const prefix =
      buildPrefix();
    return prefix;
  });

  format(v: number, fallback?: string): string {
    const safeFallback = fallback ?? 'default';
    if (v < 0) {
      return 'negative';
    } else {
      v = v + 1;
    }
    if (v === 999) { return 'overflow'; }
    return v > 10
      ? 'large'
      : safeFallback;
  }
}
`
	eofSrc := "const a = 1;\nconst b = 2;"

	covByFile := map[string]gotestutil.IstanbulFileCoverage{
		"web/src/app/sample/missing.ts": {
			File:       "web/src/app/sample/missing.ts",
			Statements: []gotestutil.IstanbulSpan{{Skip: true, StartLine: 1, EndLine: 1}},
		},
		"web/src/app/sample/skipped.ts": {
			File:       "web/src/app/sample/skipped.ts",
			Statements: []gotestutil.IstanbulSpan{{Skip: true, StartLine: 1, EndLine: 1}},
		},
		"web/src/app/sample/eof.ts": {
			File: "web/src/app/sample/eof.ts",
			Statements: []gotestutil.IstanbulSpan{
				{StartLine: 1, StartCol: 0, EndLine: 1, EndCol: 12, Count: 1},
				{StartLine: 2, StartCol: 0, EndLine: 2, EndCol: 12, Count: 0},
			},
		},
		"web/src/app/sample/sample.ts": {
			File: "web/src/app/sample/sample.ts",
			Statements: []gotestutil.IstanbulSpan{
				{StartLine: 7, StartCol: 21, EndLine: 11, EndCol: 4, Count: 1},
				{StartLine: 9, StartCol: 6, EndLine: 9, EndCol: 19, Count: 0},
				{StartLine: 10, StartCol: 4, EndLine: 10, EndCol: 18, Count: 0},
				{StartLine: 14, StartCol: 25, EndLine: 14, EndCol: 46, Count: 1},
				{StartLine: 15, StartCol: 4, EndLine: 19, EndCol: 5, Count: 1},
				{StartLine: 16, StartCol: 6, EndLine: 16, EndCol: 24, Count: 0},
				{StartLine: 18, StartCol: 6, EndLine: 18, EndCol: 16, Count: 1},
				{StartLine: 20, StartCol: 4, EndLine: 20, EndCol: 41, Count: 1},
				{StartLine: 20, StartCol: 21, EndLine: 20, EndCol: 39, Count: 0},
				{StartLine: 21, StartCol: 4, EndLine: 23, EndCol: 21, Count: 1},
			},
			Functions: []gotestutil.IstanbulFnSpan{
				{Skip: true},
				{
					Decl:  gotestutil.IstanbulSpan{StartLine: 7, StartCol: 30, EndLine: 7, EndCol: 35, Count: 0},
					Body:  gotestutil.IstanbulSpan{StartLine: 7, StartCol: 36, EndLine: 11, EndCol: 3, Count: 0},
					Count: 0,
				},
				{
					Decl:  gotestutil.IstanbulSpan{StartLine: 13, StartCol: 2, EndLine: 13, EndCol: 8, Count: 1},
					Body:  gotestutil.IstanbulSpan{StartLine: 13, StartCol: 47, EndLine: 24, EndCol: 3, Count: 1},
					Count: 1,
				},
			},
			Branches: []gotestutil.IstanbulBranchSpan{
				{BranchType: "binary-expr", Arm: gotestutil.IstanbulSpan{StartLine: 14, StartCol: 25, EndLine: 14, EndCol: 33, Count: 1}},
				{BranchType: "binary-expr", Arm: gotestutil.IstanbulSpan{StartLine: 14, StartCol: 37, EndLine: 14, EndCol: 46, Count: 0}},
				{BranchType: "if", Arm: gotestutil.IstanbulSpan{StartLine: 15, StartCol: 4, EndLine: 19, EndCol: 5, Count: 0}},
				{BranchType: "if", Arm: gotestutil.IstanbulSpan{StartLine: 17, StartCol: 11, EndLine: 19, EndCol: 5, Count: 1}},
				{BranchType: "cond-expr", Arm: gotestutil.IstanbulSpan{StartLine: 22, StartCol: 8, EndLine: 22, EndCol: 15, Count: 0}},
				{BranchType: "cond-expr", Arm: gotestutil.IstanbulSpan{StartLine: 23, StartCol: 8, EndLine: 23, EndCol: 20, Count: 1}},
			},
		},
	}

	allLines := make(map[int]struct{})
	for ln := 1; ln <= 25; ln++ {
		allLines[ln] = struct{}{}
	}
	readSource := func(relPath string) ([]byte, error) {
		switch relPath {
		case "web/src/app/sample/sample.ts":
			return []byte(sampleSrc), nil
		case "web/src/app/sample/eof.ts":
			return []byte(eofSrc), nil
		case "web/src/app/sample/skipped.ts":
			return []byte("const skipped = 1;\n"), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	testCases := []struct {
		name           string
		changedLines   map[string]map[int]struct{}
		includeOverall bool
		wantSummary    rule.CoverageSummary
		wantSpanStarts []int
	}{
		{
			name: "handles interfaces callbacks single-line guards nullish coalescing and multi-line ternaries",
			changedLines: map[string]map[int]struct{}{
				"web/src/app/sample/sample.ts":  allLines,
				"web/src/app/sample/missing.ts": {1: {}},
				"web/src/app/sample/skipped.ts": {1: {}},
				"web/src/app/sample/nocov.ts":   {1: {}},
			},
			includeOverall: true,
			wantSummary: rule.CoverageSummary{
				Enabled:           true,
				HasOverall:        true,
				PatchExecutable:   14,
				PatchCovered:      8,
				OverallStatements: 12,
				OverallCovered:    7,
			},
			wantSpanStarts: []int{8, 16, 20, 22},
		},
		{
			name: "evaluates final line when file has no trailing newline and ignores empty changed line map",
			changedLines: map[string]map[int]struct{}{
				"web/src/app/sample/eof.ts":    {1: {}, 2: {}},
				"web/src/app/sample/sample.ts": {},
			},
			includeOverall: false,
			wantSummary: rule.CoverageSummary{
				Enabled:         true,
				HasOverall:      false,
				PatchExecutable: 2,
				PatchCovered:    1,
			},
			wantSpanStarts: []int{2},
		},
		{
			name: "interface-only and comment-only changes report zero uncovered findings",
			changedLines: map[string]map[int]struct{}{
				"web/src/app/sample/sample.ts": {1: {}, 2: {}, 3: {}, 6: {}, 11: {}},
			},
			includeOverall: false,
			wantSummary: rule.CoverageSummary{
				Enabled:    true,
				HasOverall: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AnalyzeTSCoverage(TSCoverageOptions{
				CovByFile:          covByFile,
				ChangedLinesByFile: tc.changedLines,
				ReadSource:         readSource,
				IncludeOverall:     tc.includeOverall,
			})
			if err != nil {
				t.Fatalf("AnalyzeTSCoverage() error: %v", err)
			}
			if diff := cmp.Diff(tc.wantSummary, got.Summary); diff != "" {
				t.Errorf("Summary mismatch (-want +got):\n%s", diff)
			}
			assertTSSpanStarts(t, got.Findings, tc.wantSpanStarts)
		})
	}

	_, err := AnalyzeTSCoverage(TSCoverageOptions{
		CovByFile:          covByFile,
		ChangedLinesByFile: map[string]map[int]struct{}{"web/src/app/sample/sample.ts": {1: {}}},
		ReadSource: func(string) ([]byte, error) {
			return nil, errors.New("io read failure")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "failed to read web/src/app/sample/sample.ts") {
		t.Errorf("AnalyzeTSCoverage(read error) = %v, want wrapped read error", err)
	}
}

func assertTSSpanStarts(t *testing.T, findings []rule.Finding, wantStarts []int) {
	t.Helper()
	var gotStarts []int
	for _, f := range findings {
		for _, sp := range f.UncoveredSpans {
			gotStarts = append(gotStarts, sp.StartLine)
		}
	}
	if diff := cmp.Diff(wantStarts, gotStarts); diff != "" {
		t.Errorf("UncoveredSpan starts mismatch (-want +got):\n%s", diff)
	}
}

func TestCollectEarliestTSStmtCols(t *testing.T) {
	fc := gotestutil.IstanbulFileCoverage{
		Statements: []gotestutil.IstanbulSpan{
			{StartLine: 0, StartCol: 1},
			{StartLine: 4, StartCol: 1},
			{StartLine: 1, StartCol: 10},
			{StartLine: 1, StartCol: 0},
			{StartLine: 1, StartCol: 4},
			{StartLine: 2, StartCol: 1, Skip: true},
			{StartLine: 2, StartCol: 6},
			{StartLine: 2, StartCol: 8},
			{StartLine: 3, StartCol: 5},
		},
		Functions: []gotestutil.IstanbulFnSpan{
			{Skip: true, Decl: gotestutil.IstanbulSpan{StartLine: 2, StartCol: 0}},
			{Decl: gotestutil.IstanbulSpan{StartLine: 2, StartCol: 2}},
		},
	}
	got := collectEarliestTSStmtCols(3, fc)
	want := []int{-1, 0, 2, 5}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("collectEarliestTSStmtCols() mismatch (-want +got):\n%s", diff)
	}
	skippedCands := collectTSCandidates([]string{"const x = 1;"}, []int{-1, 0}, gotestutil.IstanbulFileCoverage{
		Statements: []gotestutil.IstanbulSpan{{StartLine: 1, EndLine: 1, Skip: true}},
	})
	if len(skippedCands) != 0 {
		t.Errorf("collectTSCandidates(Skip) len = %d, want 0", len(skippedCands))
	}
}

func TestBuildTSCandidateCases(t *testing.T) {
	testCases := []struct {
		name     string
		rawLines []string
		earliest []int
		sp       gotestutil.IstanbulSpan
		wantOK   bool
		wantEff  [2]int
	}{
		{
			name:     "skipped span returns false",
			rawLines: []string{"const x = 1;"},
			earliest: []int{-1, 0},
			sp:       gotestutil.IstanbulSpan{StartLine: 1, EndLine: 1, EndCol: 12, Skip: true},
			wantOK:   false,
		},
		{
			name:     "single-line closing token is preserved",
			rawLines: []string{"}"},
			earliest: []int{-1, -1},
			sp:       gotestutil.IstanbulSpan{StartLine: 1, StartCol: 0, EndLine: 1, EndCol: 1},
			wantOK:   true,
			wantEff:  [2]int{1, 1},
		},
		{
			name:     "multiline closing brace on final line trims effEnd",
			rawLines: []string{"if (ok) {", "  run();", "}"},
			earliest: []int{-1, 0, 2, -1},
			sp:       gotestutil.IstanbulSpan{StartLine: 1, StartCol: 0, EndLine: 3, EndCol: 1},
			wantOK:   true,
			wantEff:  [2]int{1, 2},
		},
		{
			name:     "multiline span ending past rawLines avoids out-of-bounds",
			rawLines: []string{"const a = 1;", "const b = 2;"},
			earliest: []int{-1, 0, 0},
			sp:       gotestutil.IstanbulSpan{StartLine: 1, StartCol: 0, EndLine: 5, EndCol: 3},
			wantOK:   true,
			wantEff:  [2]int{1, 5},
		},
		{
			name:     "opening suffix on last rawLine after col-zero statement advances effStart",
			rawLines: []string{"const a = 1;", "run(() => {"},
			earliest: []int{-1, 0, 0},
			sp:       gotestutil.IstanbulSpan{StartLine: 2, StartCol: 6, EndLine: 3, EndCol: 2},
			wantOK:   true,
			wantEff:  [2]int{3, 3},
		},
		{
			name:     "start line past rawLines avoids out-of-bounds",
			rawLines: []string{"const a = 1;"},
			earliest: []int{-1, 0},
			sp:       gotestutil.IstanbulSpan{StartLine: 2, StartCol: 4, EndLine: 3, EndCol: 2},
			wantOK:   true,
			wantEff:  [2]int{2, 3},
		},
		{
			name:     "opening suffix with no earlier statement keeps effStart",
			rawLines: []string{"  {", "  run();"},
			earliest: []int{-1, -1, 2},
			sp:       gotestutil.IstanbulSpan{StartLine: 1, StartCol: 2, EndLine: 2, EndCol: 8},
			wantOK:   true,
			wantEff:  [2]int{1, 2},
		},
		{
			name:     "opening suffix at same column as earliest statement keeps effStart",
			rawLines: []string{"{", "  run();"},
			earliest: []int{-1, 0, 2},
			sp:       gotestutil.IstanbulSpan{StartLine: 1, StartCol: 0, EndLine: 2, EndCol: 8},
			wantOK:   true,
			wantEff:  [2]int{1, 2},
		},
		{
			name:     "non-opening suffix after earlier statement keeps effStart",
			rawLines: []string{"const x = call(", "  arg)"},
			earliest: []int{-1, 0, 2},
			sp:       gotestutil.IstanbulSpan{StartLine: 1, StartCol: 10, EndLine: 2, EndCol: 6},
			wantOK:   true,
			wantEff:  [2]int{1, 2},
		},
		{
			name:     "empty multiline callback collapses effStart past effEnd",
			rawLines: []string{"run(() => {", "});"},
			earliest: []int{-1, 0, -1},
			sp:       gotestutil.IstanbulSpan{StartLine: 1, StartCol: 6, EndLine: 2, EndCol: 3},
			wantOK:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cand, ok := buildTSCandidate(tc.rawLines, tc.earliest, tsCandidateStatement, "", tc.sp)
			if ok != tc.wantOK {
				t.Fatalf("buildTSCandidate() ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && ([2]int{cand.effStart, cand.effEnd} != tc.wantEff) {
				t.Errorf("eff bounds = %v, want %v", [2]int{cand.effStart, cand.effEnd}, tc.wantEff)
			}
		})
	}
}

func TestEvaluateTSLineStateAndPredicates(t *testing.T) {
	t.Run("wider uncovered sibling statement overrides narrower covered statement on same line", func(t *testing.T) {
		cands := []tsLineCandidate{
			{kind: tsCandidateStatement, effStart: 1, effEnd: 1, origStartLine: 1, origStartCol: 0, origEndLine: 1, origEndCol: 20, count: 0},
			{kind: tsCandidateStatement, effStart: 1, effEnd: 1, origStartLine: 1, origStartCol: 22, origEndLine: 1, origEndCol: 30, count: 1},
		}
		got := evaluateTSLineState(1, cands)
		if got != (tsLineState{executable: true, covered: false}) {
			t.Errorf("evaluateTSLineState() = %+v, want executable uncovered", got)
		}
	})

	t.Run("findEarliestCoveredStmtCol filters non-statements other lines and zero counts", func(t *testing.T) {
		filtered := []tsLineCandidate{
			{kind: tsCandidateFnBody, origStartLine: 1, origStartCol: 0, count: 1},
			{kind: tsCandidateStatement, origStartLine: 2, origStartCol: 1, count: 1},
			{kind: tsCandidateStatement, origStartLine: 1, origStartCol: 2, count: 0},
			{kind: tsCandidateStatement, origStartLine: 1, origStartCol: 9, count: 1},
			{kind: tsCandidateStatement, origStartLine: 1, origStartCol: 4, count: 1},
		}
		if got := findEarliestCoveredStmtCol(1, filtered); got != 4 {
			t.Errorf("findEarliestCoveredStmtCol(filtered) = %d, want 4", got)
		}
		colZero := []tsLineCandidate{
			{kind: tsCandidateStatement, origStartLine: 1, origStartCol: 0, count: 1},
			{kind: tsCandidateStatement, origStartLine: 1, origStartCol: 6, count: 1},
		}
		if got := findEarliestCoveredStmtCol(1, colZero); got != 0 {
			t.Errorf("findEarliestCoveredStmtCol(colZero) = %d, want 0", got)
		}
	})

	t.Run("shouldSkipCoveredLineSubspan", testShouldSkipCoveredLineSubspan)
	t.Run("isSingleLineUncoveredOverride", testIsSingleLineUncoveredOverride)
	t.Run("isNarrowerTSCandidate", testIsNarrowerTSCandidate)
	t.Run("column and closing helpers", testLineCovScalarHelpers)
}

func testShouldSkipCoveredLineSubspan(t *testing.T) {
	cases := []struct {
		cand        tsLineCandidate
		earliestCol int
		want        bool
	}{
		{tsLineCandidate{kind: tsCandidateBranchArm, branchType: "binary-expr", origStartLine: 1, origEndLine: 2}, 0, false},
		{tsLineCandidate{kind: tsCandidateBranchArm, branchType: "binary-expr", origStartLine: 0, origEndLine: 1}, 0, false},
		{tsLineCandidate{kind: tsCandidateBranchArm, branchType: "binary-expr", origStartLine: 1, origEndLine: 1}, -1, false},
		{tsLineCandidate{kind: tsCandidateBranchArm, branchType: "binary-expr", origStartLine: 1, origEndLine: 1}, 0, true},
		{tsLineCandidate{kind: tsCandidateBranchArm, branchType: "default-arg", origStartLine: 1, origEndLine: 1}, 0, true},
		{tsLineCandidate{kind: tsCandidateBranchArm, branchType: "cond-expr", origStartLine: 1, origEndLine: 1}, 0, false},
		{tsLineCandidate{kind: tsCandidateStatement, branchType: "binary-expr", origStartLine: 1, origEndLine: 1}, 0, false},
		{tsLineCandidate{kind: tsCandidateFnDecl, origStartLine: 1, origEndLine: 1, origStartCol: 4}, 4, false},
		{tsLineCandidate{kind: tsCandidateFnDecl, origStartLine: 1, origEndLine: 1, origStartCol: 8}, 4, true},
	}
	for i, tc := range cases {
		if got := shouldSkipCoveredLineSubspan(1, &tc.cand, tc.earliestCol); got != tc.want {
			t.Errorf("case %d: shouldSkipCoveredLineSubspan() = %v, want %v", i, got, tc.want)
		}
	}
}

func testIsSingleLineUncoveredOverride(t *testing.T) {
	cases := []struct {
		cand tsLineCandidate
		want bool
	}{
		{tsLineCandidate{kind: tsCandidateStatement, effStart: 5, effEnd: 5, count: 1}, false},
		{tsLineCandidate{kind: tsCandidateStatement, effStart: 4, effEnd: 5, count: 0}, false},
		{tsLineCandidate{kind: tsCandidateStatement, effStart: 5, effEnd: 6, count: 0}, false},
		{tsLineCandidate{kind: tsCandidateStatement, effStart: 5, effEnd: 5, count: 0}, true},
		{tsLineCandidate{kind: tsCandidateFnBody, effStart: 5, effEnd: 5, origStartLine: 5, origEndLine: 5, count: 0}, true},
		{tsLineCandidate{kind: tsCandidateFnBody, effStart: 5, effEnd: 5, origStartLine: 4, origEndLine: 5, count: 0}, false},
		{tsLineCandidate{kind: tsCandidateFnBody, effStart: 5, effEnd: 5, origStartLine: 5, origEndLine: 6, count: 0}, false},
		{tsLineCandidate{kind: tsCandidateBranchArm, effStart: 5, effEnd: 5, origStartLine: 5, origEndLine: 5, count: 0}, false},
	}
	for i, tc := range cases {
		if got := isSingleLineUncoveredOverride(5, &tc.cand); got != tc.want {
			t.Errorf("case %d: isSingleLineUncoveredOverride() = %v, want %v", i, got, tc.want)
		}
	}
}

func testIsNarrowerTSCandidate(t *testing.T) {
	cases := []struct {
		a    tsLineCandidate
		b    tsLineCandidate
		want bool
	}{
		{
			a:    tsLineCandidate{origStartLine: 5, origEndLine: 6},
			b:    tsLineCandidate{origStartLine: 1, origEndLine: 3},
			want: true,
		},
		{
			a:    tsLineCandidate{origStartLine: 1, origEndLine: 4},
			b:    tsLineCandidate{origStartLine: 2, origEndLine: 3},
			want: false,
		},
		{
			a:    tsLineCandidate{kind: tsCandidateBranchArm, origStartLine: 1, origEndLine: 3, origStartCol: 10, origEndCol: 20, count: 1},
			b:    tsLineCandidate{kind: tsCandidateStatement, origStartLine: 1, origEndLine: 3, origStartCol: 0, origEndCol: 5, count: 1},
			want: true,
		},
		{
			a:    tsLineCandidate{kind: tsCandidateStatement, origStartLine: 1, origEndLine: 1, origStartCol: 10, origEndCol: 15, count: 1},
			b:    tsLineCandidate{kind: tsCandidateStatement, origStartLine: 1, origEndLine: 1, origStartCol: 2, origEndCol: 10, count: 1},
			want: true,
		},
		{
			a:    tsLineCandidate{kind: tsCandidateStatement, origStartLine: 1, origEndLine: 1, origStartCol: 2, origEndCol: 10, count: 1},
			b:    tsLineCandidate{kind: tsCandidateStatement, origStartLine: 1, origEndLine: 1, origStartCol: 10, origEndCol: 15, count: 1},
			want: false,
		},
		{
			a:    tsLineCandidate{kind: tsCandidateStatement, origStartLine: 1, origEndLine: 1, origStartCol: 2, origEndCol: 10, count: 0},
			b:    tsLineCandidate{kind: tsCandidateBranchArm, origStartLine: 1, origEndLine: 1, origStartCol: 2, origEndCol: 10, count: 2},
			want: true,
		},
		{
			a:    tsLineCandidate{kind: tsCandidateBranchArm, origStartLine: 1, origEndLine: 1, origStartCol: 2, origEndCol: 10, count: 2},
			b:    tsLineCandidate{kind: tsCandidateStatement, origStartLine: 1, origEndLine: 1, origStartCol: 2, origEndCol: 10, count: 0},
			want: false,
		},
		{
			a:    tsLineCandidate{kind: tsCandidateStatement, origStartLine: 1, origEndLine: 1, origStartCol: 2, origEndCol: 10, count: 1},
			b:    tsLineCandidate{kind: tsCandidateBranchArm, origStartLine: 1, origEndLine: 1, origStartCol: 2, origEndCol: 10, count: 2},
			want: false,
		},
		{
			a:    tsLineCandidate{kind: tsCandidateStatement, origStartLine: 1, origEndLine: 1, origStartCol: 2, origEndCol: 10, count: 1},
			b:    tsLineCandidate{kind: tsCandidateStatement, origStartLine: 1, origEndLine: 1, origStartCol: 2, origEndCol: 10, count: 1},
			want: false,
		},
	}
	for i, tc := range cases {
		if got := isNarrowerTSCandidate(&tc.a, &tc.b); got != tc.want {
			t.Errorf("case %d: isNarrowerTSCandidate() = %v, want %v", i, got, tc.want)
		}
	}
}

func testLineCovScalarHelpers(t *testing.T) {
	if states := mapIstanbulToLines(nil, nil, gotestutil.IstanbulFileCoverage{}); len(states) != 1 {
		t.Errorf("mapIstanbulToLines(empty) len = %d, want 1", len(states))
	}
	if col := resolveTSEndCol([]string{"abc"}, 5, -1); col != 0 {
		t.Errorf("resolveTSEndCol(past end, -1) = %d, want 0", col)
	}
	if col := resolveTSEndCol([]string{"abc"}, 1, -1); col != 3 {
		t.Errorf("resolveTSEndCol(negative endCol) = %d, want 3", col)
	}
	if col := resolveTSEndCol([]string{"abc"}, 1, 0); col != 0 {
		t.Errorf("resolveTSEndCol(zero endCol) = %d, want 0", col)
	}
	if col := resolveTSEndCol([]string{"abc"}, 1, 2); col != 2 {
		t.Errorf("resolveTSEndCol(in-bounds endCol) = %d, want 2", col)
	}
	if col := resolveTSEndCol([]string{"abc"}, 1, 99); col != 3 {
		t.Errorf("resolveTSEndCol(clamp) = %d, want 3", col)
	}
	if isOnlyTSOpeningSuffix("{", -1) {
		t.Errorf("isOnlyTSOpeningSuffix(-1) = true, want false")
	}
	if !isOnlyTSOpeningSuffix("{ // trailing", 0) {
		t.Errorf("isOnlyTSOpeningSuffix(col 0 brace) = false, want true")
	}
	if isOnlyTSOpeningSuffix("{", 5) {
		t.Errorf("isOnlyTSOpeningSuffix(past len) = true, want false")
	}
	if !isOnlyTSClosingChars(" \t\r});,]") {
		t.Errorf("isOnlyTSClosingChars(valid) = false, want true")
	}
	if isOnlyTSClosingChars("  x}") {
		t.Errorf("isOnlyTSClosingChars(identifier) = true, want false")
	}
	gotFiles := sortedNonEmptyLineFiles(map[string]map[int]struct{}{
		"b.ts":     {1: {}},
		"empty.ts": {},
		"a.ts":     {2: {}},
	})
	if diff := cmp.Diff([]string{"a.ts", "b.ts"}, gotFiles); diff != "" {
		t.Errorf("sortedNonEmptyLineFiles() mismatch (-want +got):\n%s", diff)
	}
}
