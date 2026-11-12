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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
	"github.com/google/go-cmp/cmp"
)

func TestAnalyzeTSCoverage(t *testing.T) {
	sampleSrc := `export interface SampleViewModel {
  readonly label: string;
}

export class SamplePresenter {
  // Uncalled computed callback inside initialized class property.
  readonly summary = computed(() => {
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

	covByFile := map[string]gotestutil.IstanbulFileCoverage{
		"web/src/app/sample/sample.ts": {
			File: "web/src/app/sample/sample.ts",
			Statements: []gotestutil.IstanbulSpan{
				// Class property "readonly summary = computed(() => { ... });" initialized on instance creation.
				{StartLine: 7, StartCol: 21, EndLine: 11, EndCol: 4, Count: 1},
				// Uncalled statements inside computed callback.
				{StartLine: 9, StartCol: 6, EndLine: 9, EndCol: 19, Count: 0},
				{StartLine: 10, StartCol: 4, EndLine: 10, EndCol: 18, Count: 0},
				// Single-line nullish coalescing statement executed with non-null fallback.
				{StartLine: 14, StartCol: 25, EndLine: 14, EndCol: 46, Count: 1},
				// "if (v < 0) { ... } else { ... }" outer statement evaluated.
				{StartLine: 15, StartCol: 4, EndLine: 19, EndCol: 5, Count: 1},
				// Untaken "if (v < 0)" return statement.
				{StartLine: 16, StartCol: 6, EndLine: 16, EndCol: 24, Count: 0},
				// Taken "else" body statement.
				{StartLine: 18, StartCol: 6, EndLine: 18, EndCol: 16, Count: 1},
				// Single-line "if (v === 999) { return 'overflow'; }" condition evaluated, return untaken.
				{StartLine: 20, StartCol: 4, EndLine: 20, EndCol: 41, Count: 1},
				{StartLine: 20, StartCol: 21, EndLine: 20, EndCol: 39, Count: 0},
				// Multi-line ternary return statement evaluated (v > 10 was false).
				{StartLine: 21, StartCol: 4, EndLine: 23, EndCol: 21, Count: 1},
			},
			Functions: []gotestutil.IstanbulFnSpan{
				// Uncalled computed arrow callback.
				{
					Decl:  gotestutil.IstanbulSpan{StartLine: 7, StartCol: 30, EndLine: 7, EndCol: 35, Count: 0},
					Body:  gotestutil.IstanbulSpan{StartLine: 7, StartCol: 36, EndLine: 11, EndCol: 3, Count: 0},
					Count: 0,
				},
				// Called format() method.
				{
					Decl:  gotestutil.IstanbulSpan{StartLine: 13, StartCol: 2, EndLine: 13, EndCol: 8, Count: 1},
					Body:  gotestutil.IstanbulSpan{StartLine: 13, StartCol: 47, EndLine: 24, EndCol: 3, Count: 1},
					Count: 1,
				},
			},
			Branches: []gotestutil.IstanbulBranchSpan{
				// Single-line "fallback ?? 'default'" where fallback was provided.
				{
					BranchType: "binary-expr",
					Arm:        gotestutil.IstanbulSpan{StartLine: 14, StartCol: 25, EndLine: 14, EndCol: 33, Count: 1},
					Count:      1,
				},
				{
					BranchType: "binary-expr",
					Arm:        gotestutil.IstanbulSpan{StartLine: 14, StartCol: 37, EndLine: 14, EndCol: 46, Count: 0},
					Count:      0,
				},
				// "if (v < 0)" branches: Istanbul sets locations[0] to the entire IfStatement loc (col 4).
				{
					BranchType: "if",
					Arm:        gotestutil.IstanbulSpan{StartLine: 15, StartCol: 4, EndLine: 19, EndCol: 5, Count: 0},
					Count:      0,
				},
				{
					BranchType: "if",
					Arm:        gotestutil.IstanbulSpan{StartLine: 17, StartCol: 11, EndLine: 19, EndCol: 5, Count: 1},
					Count:      1,
				},
				// Multi-line ternary: '? large' untaken, ': safeFallback' taken.
				{
					BranchType: "cond-expr",
					Arm:        gotestutil.IstanbulSpan{StartLine: 22, StartCol: 8, EndLine: 22, EndCol: 15, Count: 0},
					Count:      0,
				},
				{
					BranchType: "cond-expr",
					Arm:        gotestutil.IstanbulSpan{StartLine: 23, StartCol: 8, EndLine: 23, EndCol: 20, Count: 1},
					Count:      1,
				},
			},
		},
	}

	allLines := make(map[int]struct{})
	for l := 1; l <= 25; l++ {
		allLines[l] = struct{}{}
	}

	testCases := []struct {
		name           string
		changedLines   map[string]map[int]struct{}
		includeOverall bool
		want           *CoverageReport
	}{
		{
			name: "handles interfaces callbacks single-line guards nullish coalescing and multi-line ternaries",
			changedLines: map[string]map[int]struct{}{
				"web/src/app/sample/sample.ts": allLines,
			},
			includeOverall: true,
			want: &CoverageReport{
				BaseLabel:            "trunk()",
				PatchExecutableLines: 14,
				PatchCoveredLines:    8,
				OverallStatements:    10,
				OverallCovered:       6,
				HasOverallCoverage:   true,
				Files: []FileCoverageReport{
					{
						File:                   "web/src/app/sample/sample.ts",
						ChangedExecutableLines: 14,
						ChangedCoveredLines:    8,
						TotalStatements:        10,
						CoveredStatements:      6,
						UncoveredRanges: []LineRange{
							{StartLine: 8, EndLine: 10},
							{StartLine: 16, EndLine: 16},
							{StartLine: 20, EndLine: 20},
							{StartLine: 22, EndLine: 22},
						},
						UncoveredSpans: []UncoveredSpan{
							{
								File:            "web/src/app/sample/sample.ts",
								StartLine:       8,
								EndLine:         10,
								ExecutableLines: 3,
								Lines: []SourceLine{
									{Number: 7, Text: "  readonly summary = computed(() => {"},
									{Number: 8, Text: "    const prefix ="},
									{Number: 9, Text: "      buildPrefix();"},
									{Number: 10, Text: "    return prefix;"},
								},
							},
							{
								File:            "web/src/app/sample/sample.ts",
								StartLine:       16,
								EndLine:         16,
								ExecutableLines: 1,
								Lines: []SourceLine{
									{Number: 15, Text: "    if (v < 0) {"},
									{Number: 16, Text: "      return 'negative';"},
								},
							},
							{
								File:            "web/src/app/sample/sample.ts",
								StartLine:       20,
								EndLine:         20,
								ExecutableLines: 1,
								Lines: []SourceLine{
									{Number: 20, Text: "    if (v === 999) { return 'overflow'; }"},
								},
							},
							{
								File:            "web/src/app/sample/sample.ts",
								StartLine:       22,
								EndLine:         22,
								ExecutableLines: 1,
								Lines: []SourceLine{
									{Number: 22, Text: "      ? 'large'"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "interface-only and comment-only changes report zero changed executable lines",
			changedLines: map[string]map[int]struct{}{
				"web/src/app/sample/sample.ts": {
					1:  {},
					2:  {},
					3:  {},
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
		if relPath == "web/src/app/sample/sample.ts" {
			return []byte(sampleSrc), nil
		}
		return nil, os.ErrNotExist
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AnalyzeTSCoverage("trunk()", covByFile, tc.changedLines, readSource, tc.includeOverall)
			if err != nil {
				t.Fatalf("AnalyzeTSCoverage() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AnalyzeTSCoverage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClassifyExecutableTSLines(t *testing.T) {
	src := `// leading line comment
/* multi-line
   block comment */
const a = 'escaped \' // not comment'; // trailing comment
const b = "escaped \" /* not block */";
const c = ` + "`" + `template
literal` + "`" + `;
  });
`
	got := classifyExecutableTSLines([]byte(src), 8)
	want := []bool{
		false, // index 0 unused
		false, // line 1: line comment
		false, // line 2: block comment start
		false, // line 3: block comment end
		true,  // line 4: single-quoted string + trailing comment
		true,  // line 5: double-quoted string
		true,  // line 6: template start
		true,  // line 7: template continuation
		false, // line 8: pure closing delimiters
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("classifyExecutableTSLines() mismatch (-want +got):\n%s", diff)
	}
}

func TestSupplementMissingTSCoverage(t *testing.T) {
	realRepoRoot := mustFindRepoRootForTSMutationTest(t)
	if _, err := os.Stat(filepath.Join(realRepoRoot, "web", "node_modules", "typescript")); err != nil {
		t.Skipf("skipping zero-hit integration test: web/node_modules not installed (%v)", err)
	}

	tmpRepo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpRepo, "web"), 0o755); err != nil {
		t.Fatalf("failed to create web dir: %v", err)
	}
	if err := os.Symlink(filepath.Join(realRepoRoot, "web", "package.json"), filepath.Join(tmpRepo, "web", "package.json")); err != nil {
		t.Fatalf("failed to symlink web/package.json: %v", err)
	}
	if err := os.Symlink(filepath.Join(realRepoRoot, "web", "node_modules"), filepath.Join(tmpRepo, "web", "node_modules")); err != nil {
		t.Fatalf("failed to symlink web/node_modules: %v", err)
	}

	typesOnlyRel := "web/src/app/sample/types.ts"
	untestedImplRel := "web/src/app/sample/untested.ts"

	typesSrc := `export interface OnlyTypes {
  readonly id: string;
  readonly count: number;
}

export type Identifier = string | number;
`
	untestedSrc := `export function computeScore(v: number): number {
  if (v < 0) {
    return 0;
  }
  return v * 2;
}
`
	mustWriteTSLineCovFile(t, filepath.Join(tmpRepo, filepath.FromSlash(typesOnlyRel)), typesSrc)
	mustWriteTSLineCovFile(t, filepath.Join(tmpRepo, filepath.FromSlash(untestedImplRel)), untestedSrc)

	alreadyCoveredRel := "web/src/app/sample/already-covered.ts"
	deletedRel := "web/src/app/sample/deleted.ts"
	mustWriteTSLineCovFile(t, filepath.Join(tmpRepo, filepath.FromSlash(alreadyCoveredRel)), `export const hit = 1;`)

	covByFile := map[string]gotestutil.IstanbulFileCoverage{
		alreadyCoveredRel: {
			File: alreadyCoveredRel,
			Statements: []gotestutil.IstanbulSpan{
				{StartLine: 1, StartCol: 13, EndLine: 1, EndCol: 21, Count: 5},
			},
		},
	}
	err := SupplementMissingTSCoverage(context.Background(), tmpRepo, []string{typesOnlyRel, untestedImplRel, alreadyCoveredRel, deletedRel}, covByFile)
	if err != nil {
		t.Fatalf("SupplementMissingTSCoverage() unexpected error: %v", err)
	}
	if got := covByFile[alreadyCoveredRel].Statements[0].Count; got != 5 {
		t.Errorf("existing coverage count = %d, want 5", got)
	}

	if _, hasTypes := covByFile[typesOnlyRel]; hasTypes {
		t.Errorf("covByFile unexpectedly contained pure type-only file %s", typesOnlyRel)
	}
	if _, hasImpl := covByFile[untestedImplRel]; !hasImpl {
		t.Fatalf("covByFile missing unimported implementation file %s", untestedImplRel)
	}

	changed := map[string]map[int]struct{}{
		typesOnlyRel:    {1: {}, 2: {}, 3: {}, 6: {}},
		untestedImplRel: {1: {}, 2: {}, 3: {}, 4: {}, 5: {}, 6: {}},
	}
	readSource := func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(tmpRepo, filepath.FromSlash(relPath)))
	}

	report, err := AnalyzeTSCoverage("trunk()", covByFile, changed, readSource, true)
	if err != nil {
		t.Fatalf("AnalyzeTSCoverage() unexpected error: %v", err)
	}

	want := &CoverageReport{
		BaseLabel:            "trunk()",
		PatchExecutableLines: 4,
		PatchCoveredLines:    0,
		OverallStatements:    4,
		OverallCovered:       1,
		HasOverallCoverage:   true,
		Files: []FileCoverageReport{
			{
				File:                   untestedImplRel,
				ChangedExecutableLines: 4,
				ChangedCoveredLines:    0,
				TotalStatements:        3,
				CoveredStatements:      0,
				UncoveredRanges: []LineRange{
					{StartLine: 1, EndLine: 5},
				},
				UncoveredSpans: []UncoveredSpan{
					{
						File:            untestedImplRel,
						StartLine:       1,
						EndLine:         5,
						ExecutableLines: 4,
						Lines: []SourceLine{
							{Number: 1, Text: "export function computeScore(v: number): number {"},
							{Number: 2, Text: "  if (v < 0) {"},
							{Number: 3, Text: "    return 0;"},
							{Number: 4, Text: "  }"},
							{Number: 5, Text: "  return v * 2;"},
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(want, report); diff != "" {
		t.Errorf("AnalyzeTSCoverage() on zero-hit supplemented files mismatch (-want +got):\n%s", diff)
	}
}

func mustWriteTSLineCovFile(t *testing.T, absPath string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", absPath, err)
	}
}
