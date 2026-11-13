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

package covcheck

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

func TestAnalyzeFileLinesAndMeasureFuncUncovered(t *testing.T) {
	src := `package demo

import "fmt"

func Compute(v int, inErr error) (int, error) {
	if inErr != nil {
		return 0, inErr
	}
	// CODEQUALITY_ALLOW(func_uncovered_lines, external): os interactive branch
	if v == 99 {
		return 99, nil
	}
	if v < 0 { // trailing brace comment
		x := -v
		y := x + 1
		// comment bridging uncovered lines within 3 lines
		return y, fmt.Errorf("negative: %d", v)
	}
	if v == 42 {
		return 42, nil
	}
	a := v + 1
	b := a + 1
	c := b + 1
	if c == 9999 {
		return 9999, nil
	}
	return c, nil
}
`
	blocks := []gotestutil.CoverBlock{
		{StartLine: 5, StartCol: 46, EndLine: 6, EndCol: 18, NumStatements: 1, Count: 1},
		{StartLine: 6, StartCol: 18, EndLine: 8, EndCol: 3, NumStatements: 1, Count: 0},
		{StartLine: 10, StartCol: 2, EndLine: 10, EndCol: 13, NumStatements: 1, Count: 1},
		{StartLine: 10, StartCol: 13, EndLine: 12, EndCol: 3, NumStatements: 1, Count: 0},
		{StartLine: 13, StartCol: 2, EndLine: 13, EndCol: 11, NumStatements: 1, Count: 1},
		{StartLine: 13, StartCol: 11, EndLine: 18, EndCol: 3, NumStatements: 3, Count: 0},
		{StartLine: 19, StartCol: 2, EndLine: 19, EndCol: 13, NumStatements: 1, Count: 1},
		{StartLine: 19, StartCol: 13, EndLine: 21, EndCol: 3, NumStatements: 1, Count: 0},
		{StartLine: 22, StartCol: 2, EndLine: 25, EndCol: 15, NumStatements: 4, Count: 1},
		{StartLine: 25, StartCol: 15, EndLine: 27, EndCol: 3, NumStatements: 1, Count: 0},
		{StartLine: 28, StartCol: 2, EndLine: 28, EndCol: 15, NumStatements: 1, Count: 1},
	}

	fm, err := astcheck.AnalyzeFile("pkg/demo/demo.go", []byte(src), 3)
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}
	fa := AnalyzeFileLines("pkg/demo/demo.go", []byte(src), blocks)

	testCases := []struct {
		name           string
		changed        map[int]struct{}
		wantPatchCov   int
		wantPatchExec  int
		wantUncovCount int
		wantSpanStarts []int
		wantSpanEnds   []int
	}{
		{
			name:           "merges close uncovered lines across comments and splits across covered code or >3 line gaps",
			changed:        map[int]struct{}{1: {}, 2: {}, 6: {}, 7: {}, 10: {}, 11: {}, 13: {}, 14: {}, 22: {}, 28: {}},
			wantPatchCov:   5,
			wantPatchExec:  8,
			wantUncovCount: 5,
			wantSpanStarts: []int{14, 20, 26},
			wantSpanEnds:   []int{17, 20, 26},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cov, exec := MeasurePatchTotals(fa, tc.changed)
			if cov != tc.wantPatchCov || exec != tc.wantPatchExec {
				t.Errorf("MeasurePatchTotals() = (%d, %d), want (%d, %d)", cov, exec, tc.wantPatchCov, tc.wantPatchExec)
			}
			uncov, spans := MeasureFuncUncovered("pkg/demo/demo.go", fa, fm.Funcs[0], fm.Directives)
			if uncov != tc.wantUncovCount {
				t.Errorf("MeasureFuncUncovered() count = %d, want %d", uncov, tc.wantUncovCount)
			}
			var gotStarts, gotEnds []int
			for _, sp := range spans {
				gotStarts = append(gotStarts, sp.StartLine)
				gotEnds = append(gotEnds, sp.EndLine)
			}
			if diff := cmp.Diff(tc.wantSpanStarts, gotStarts); diff != "" {
				t.Errorf("span starts mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantSpanEnds, gotEnds); diff != "" {
				t.Errorf("span ends mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLineCovEdgeCases(t *testing.T) {
	testCases := []struct {
		name          string
		src           string
		blocks        []gotestutil.CoverBlock
		wantRawLen    int
		checkLine     int
		wantExec      bool
		wantCovered   bool
		wantGapRanges []rule.LineRange
	}{
		{
			name:       "empty source returns nil lines",
			src:        "\n",
			wantRawLen: 0,
		},
		{
			name: "multi-block same line marks uncovered when second block count is zero and handles out-of-range startCol",
			src:  "package p\r\nfunc f(a, b bool) {\r\n\tif a { _ = 1 }; if b { _ = 2 }\r\n\x00}\r\n",
			blocks: []gotestutil.CoverBlock{
				{StartLine: 3, StartCol: 2, EndLine: 3, EndCol: 16, NumStatements: 2, Count: 1},
				{StartLine: 3, StartCol: 18, EndLine: 3, EndCol: 32, NumStatements: 2, Count: 0},
				{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10, NumStatements: 1, Count: 1},
				{StartLine: 2, StartCol: 999, EndLine: 3, EndCol: 2, NumStatements: 1, Count: 1},
			},
			wantRawLen:  4,
			checkLine:   3,
			wantExec:    true,
			wantCovered: false,
			wantGapRanges: []rule.LineRange{
				{StartLine: 1, EndLine: 1},
				{StartLine: 6, EndLine: 6},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fa := AnalyzeFileLines("pkg/p/p.go", []byte(tc.src), tc.blocks)
			if len(fa.RawLines) != tc.wantRawLen {
				t.Fatalf("len(RawLines) = %d, want %d", len(fa.RawLines), tc.wantRawLen)
			}
			if tc.checkLine > 0 {
				st := fa.States[tc.checkLine]
				if st.Executable != tc.wantExec || st.Covered != tc.wantCovered {
					t.Errorf("States[%d] = %+v, want Executable=%v Covered=%v", tc.checkLine, st, tc.wantExec, tc.wantCovered)
				}
			}
			if tc.wantGapRanges != nil {
				gotRanges := rule.BuildLineRanges([]int{1, 6}, make([]bool, 10))
				if diff := cmp.Diff(tc.wantGapRanges, gotRanges); diff != "" {
					t.Errorf("BuildLineRanges(>3 gap) mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
