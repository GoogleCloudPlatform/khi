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
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseCoverProfile(t *testing.T) {
	profile := `mode: set
github.com/GoogleCloudPlatform/khi/pkg/foo/bar.go:10.15,14.2 2 0
github.com/GoogleCloudPlatform/khi/pkg/foo/bar.go:10.15,14.2 2 1
github.com/GoogleCloudPlatform/khi/pkg/foo/bar.go:5.12,8.2 1 0
github.com/GoogleCloudPlatform/khi/pkg/generated/gen.go:1.1,5.2 3 1
github.com/GoogleCloudPlatform/khi/pkg/testutil/helper.go:1.1,5.2 2 1
github.com/GoogleCloudPlatform/khi/pkg/foo/zzz_codegen.go:1.1,5.2 2 1
`
	want := map[string][]CoverBlock{
		"pkg/foo/bar.go": {
			{
				StartLine:     5,
				StartCol:      12,
				EndLine:       8,
				EndCol:        2,
				NumStatements: 1,
				Count:         0,
			},
			{
				StartLine:     10,
				StartCol:      15,
				EndLine:       14,
				EndCol:        2,
				NumStatements: 2,
				Count:         1,
			},
		},
	}

	got, err := ParseCoverProfile(strings.NewReader(profile), "github.com/GoogleCloudPlatform/khi")
	if err != nil {
		t.Fatalf("ParseCoverProfile() unexpected error: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ParseCoverProfile() mismatch (-want +got):\n%s", diff)
	}
}

func TestAnalyzeCoverage(t *testing.T) {
	sampleSrc := `package sample

import "fmt"

func Process(v int) (string, error) {
	// Comment inside covered block should not count as executable.
	if v < 0 {
		return "", fmt.Errorf("negative value: %d", v)
	} else {
		v = v + 1
	}
	if v == 999 { return "", fmt.Errorf("overflow") }
	return fmt.Sprintf("%d", v), nil
}
`

	sampleBlocks := map[string][]CoverBlock{
		"pkg/sample/sample.go": {
			// Function entry up to "if v < 0".
			{StartLine: 5, StartCol: 37, EndLine: 7, EndCol: 11, NumStatements: 1, Count: 1},
			// Untaken "if v < 0" body.
			{StartLine: 7, StartCol: 11, EndLine: 9, EndCol: 3, NumStatements: 1, Count: 0},
			// Taken "else" body.
			{StartLine: 9, StartCol: 9, EndLine: 11, EndCol: 3, NumStatements: 1, Count: 1},
			// Single-line "if v == 999" condition evaluated, body untaken.
			{StartLine: 12, StartCol: 2, EndLine: 12, EndCol: 14, NumStatements: 1, Count: 1},
			{StartLine: 12, StartCol: 15, EndLine: 12, EndCol: 51, NumStatements: 1, Count: 0},
			// Final return covered.
			{StartLine: 13, StartCol: 2, EndLine: 13, EndCol: 34, NumStatements: 1, Count: 1},
		},
	}

	testCases := []struct {
		name           string
		changedLines   map[string]map[int]struct{}
		includeOverall bool
		want           *CoverageReport
	}{
		{
			name: "ignores comments and closing braces while catching multi-line and single-line uncovered branches",
			changedLines: map[string]map[int]struct{}{
				"pkg/sample/sample.go": {
					6:  {}, // Comment-only line.
					7:  {}, // "if v < 0 {" condition was evaluated (covered).
					8:  {}, // Uncovered error return inside if block.
					9:  {}, // "} else {" taken else header (covered).
					10: {}, // Covered else body.
					11: {}, // Closing brace only.
					12: {}, // Single-line "if v == 999 { return ... }" with uncovered body.
				},
			},
			includeOverall: true,
			want: &CoverageReport{
				BaseLabel:            "trunk()",
				PatchExecutableLines: 5,
				PatchCoveredLines:    3,
				OverallStatements:    6,
				OverallCovered:       4,
				HasOverallCoverage:   true,
				Files: []FileCoverageReport{
					{
						File:                   "pkg/sample/sample.go",
						ChangedExecutableLines: 5,
						ChangedCoveredLines:    3,
						TotalStatements:        6,
						CoveredStatements:      4,
						UncoveredRanges: []LineRange{
							{StartLine: 8, EndLine: 8},
							{StartLine: 12, EndLine: 12},
						},
						UncoveredSpans: []UncoveredSpan{
							{
								File:            "pkg/sample/sample.go",
								StartLine:       8,
								EndLine:         8,
								ExecutableLines: 1,
								Lines: []SourceLine{
									{Number: 7, Text: "\tif v < 0 {"},
									{Number: 8, Text: "\t\treturn \"\", fmt.Errorf(\"negative value: %d\", v)"},
								},
							},
							{
								File:            "pkg/sample/sample.go",
								StartLine:       12,
								EndLine:         12,
								ExecutableLines: 1,
								Lines: []SourceLine{
									{Number: 12, Text: "\tif v == 999 { return \"\", fmt.Errorf(\"overflow\") }"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "comment-only edit produces zero changed executable lines",
			changedLines: map[string]map[int]struct{}{
				"pkg/sample/sample.go": {
					6:  {},
					11: {},
				},
			},
			includeOverall: false,
			want: &CoverageReport{
				BaseLabel:          "trunk()",
				HasOverallCoverage: false,
			},
		},
		{
			name: "package without coverprofile blocks marks function body lines uncovered",
			changedLines: map[string]map[int]struct{}{
				"pkg/notests/logic.go": {
					4: {}, // Struct field declaration (non-executable).
					8: {}, // Function statement (executable, uncovered).
					9: {}, // Function return (executable, uncovered).
				},
			},
			includeOverall: false,
			want: &CoverageReport{
				BaseLabel:            "trunk()",
				PatchExecutableLines: 2,
				PatchCoveredLines:    0,
				HasOverallCoverage:   false,
				Files: []FileCoverageReport{
					{
						File:                   "pkg/notests/logic.go",
						ChangedExecutableLines: 2,
						ChangedCoveredLines:    0,
						UncoveredRanges: []LineRange{
							{StartLine: 8, EndLine: 9},
						},
						UncoveredSpans: []UncoveredSpan{
							{
								File:            "pkg/notests/logic.go",
								StartLine:       8,
								EndLine:         9,
								ExecutableLines: 2,
								Lines: []SourceLine{
									{Number: 7, Text: "func Compute(x int) int {"},
									{Number: 8, Text: "\ty := x * 2"},
									{Number: 9, Text: "\treturn y"},
								},
							},
						},
					},
				},
			},
		},
	}

	noTestsSrc := `package notests

type Options struct {
	Limit int
}

func Compute(x int) int {
	y := x * 2
	return y
}
`

	readSource := func(relPath string) ([]byte, error) {
		switch relPath {
		case "pkg/sample/sample.go":
			return []byte(sampleSrc), nil
		case "pkg/notests/logic.go":
			return []byte(noTestsSrc), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AnalyzeCoverage("trunk()", sampleBlocks, tc.changedLines, readSource, tc.includeOverall)
			if err != nil {
				t.Fatalf("AnalyzeCoverage() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AnalyzeCoverage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
