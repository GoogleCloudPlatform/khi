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
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
	"github.com/google/go-cmp/cmp"
)

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

	sampleBlocks := map[string][]gotestutil.CoverBlock{
		"pkg/sample/sample.go": {
			{StartLine: 5, StartCol: 37, EndLine: 7, EndCol: 11, NumStatements: 1, Count: 1},
			{StartLine: 7, StartCol: 11, EndLine: 9, EndCol: 3, NumStatements: 1, Count: 0},
			{StartLine: 9, StartCol: 9, EndLine: 11, EndCol: 3, NumStatements: 1, Count: 1},
			{StartLine: 12, StartCol: 2, EndLine: 12, EndCol: 14, NumStatements: 1, Count: 1},
			{StartLine: 12, StartCol: 15, EndLine: 12, EndCol: 51, NumStatements: 1, Count: 0},
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
					6:  {},
					7:  {},
					8:  {},
					9:  {},
					10: {},
					11: {},
					12: {},
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
	}

	readSource := func(relPath string) ([]byte, error) {
		if relPath == "pkg/sample/sample.go" {
			return []byte(sampleSrc), nil
		}
		return nil, os.ErrNotExist
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

func TestPlanCoverageExecution(t *testing.T) {
	tmpRepo := t.TempDir()
	mustCreateDummyGoFile(t, filepath.Join(tmpRepo, "pkg", "common", "set", "set.go"))
	mustCreateDummyGoFile(t, filepath.Join(tmpRepo, "pkg", "task", "inspection", "sample", "extractor.go"))
	mustCreateDummyGoFile(t, filepath.Join(tmpRepo, "pkg", "task", "inspection", "sample", "impl", "mapper.go"))
	mustCreateDummyGoFile(t, filepath.Join(tmpRepo, "scripts", "mutation-diff", "runner.go"))

	testCases := []struct {
		name         string
		changedFiles []string
		full         bool
		want         CoverageTestPlan
	}{
		{
			name:         "no changed files skips execution in targeted mode",
			changedFiles: nil,
			full:         false,
			want: CoverageTestPlan{
				Skip: true,
			},
		},
		{
			name:         "full mode runs entire repository regardless of changed files",
			changedFiles: nil,
			full:         true,
			want: CoverageTestPlan{
				Packages: []string{"./..."},
			},
		},
		{
			name: "targeted mode runs packages containing changed production files including scripts",
			changedFiles: []string{
				"pkg/common/set/set.go",
				"pkg/task/inspection/sample/extractor.go",
				"pkg/generated/ignored.go",
				"scripts/mutation-diff/runner.go",
			},
			full: false,
			want: CoverageTestPlan{
				Packages: []string{
					"./pkg/common/set",
					"./pkg/task/inspection/sample",
					"./scripts/mutation-diff",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := PlanCoverageExecution(tmpRepo, tc.changedFiles, tc.full)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("PlanCoverageExecution() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func mustCreateDummyGoFile(t *testing.T, fullPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("package dummy\n"), 0o644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}
}
