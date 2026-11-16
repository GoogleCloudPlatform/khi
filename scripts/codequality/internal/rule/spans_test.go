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

package rule

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSplitSourceLinesAndBuildUncoveredSpans(t *testing.T) {
	if got := SplitSourceLines("\n"); got != nil {
		t.Errorf("SplitSourceLines(empty) = %v, want nil", got)
	}
	raw := SplitSourceLines("x0()\r\nif cond {\r\n  x1()\r\n  // note 1\r\n  // note 2\r\n  // note 3\r\n  y1()\r\na()\r\n// { note\r\nz()\r\n")
	// Omit index 9 from codeMask so merging lines 8 and 10 also exercises ln == len(codeLines).
	codeMask := []bool{false, true, true, true, false, false, false, true, true}

	testCases := []struct {
		name      string
		uncovered []int
		want      []UncoveredSpan
	}{
		{
			name:      "empty uncovered returns nil",
			uncovered: nil,
			want:      nil,
		},
		{
			name:      "merges 3-line non-code gap splits across code line and reaches EOF",
			uncovered: []int{3, 6, 8, 10},
			want: []UncoveredSpan{
				{
					File:            "demo.ts",
					StartLine:       3,
					EndLine:         6,
					ExecutableLines: 2,
					Lines: []SourceLine{
						{Number: 2, Text: "if cond {"},
						{Number: 3, Text: "  x1()"},
						{Number: 4, Text: "  // note 1"},
						{Number: 5, Text: "  // note 2"},
						{Number: 6, Text: "  // note 3"},
					},
				},
				{
					File:            "demo.ts",
					StartLine:       8,
					EndLine:         10,
					ExecutableLines: 2,
					Lines: []SourceLine{
						{Number: 8, Text: "a()"},
						{Number: 9, Text: "// { note"},
						{Number: 10, Text: "z()"},
					},
				},
			},
		},
		{
			name:      "merges adjacent from line 1 splits 4-line non-code gap and ignores commented brace",
			uncovered: []int{1, 2, 3, 7, 10},
			want: []UncoveredSpan{
				{
					File:            "demo.ts",
					StartLine:       1,
					EndLine:         3,
					ExecutableLines: 3,
					Lines: []SourceLine{
						{Number: 1, Text: "x0()"},
						{Number: 2, Text: "if cond {"},
						{Number: 3, Text: "  x1()"},
					},
				},
				{
					File:            "demo.ts",
					StartLine:       7,
					EndLine:         7,
					ExecutableLines: 1,
					Lines:           []SourceLine{{Number: 7, Text: "  y1()"}},
				},
				{
					File:            "demo.ts",
					StartLine:       10,
					EndLine:         10,
					ExecutableLines: 1,
					Lines:           []SourceLine{{Number: 10, Text: "z()"}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildUncoveredSpans("demo.ts", raw, codeMask, tc.uncovered)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("BuildUncoveredSpans() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
