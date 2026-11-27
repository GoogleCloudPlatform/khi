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

package advisorycheck

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestAnalyzeTSTestDiffSignals verifies TS spec removal, case count reduction, assertion weakening, and skip detection (ignoring comments/strings).
func TestAnalyzeTSTestDiffSignals(t *testing.T) {
	testCases := []struct {
		name         string
		targetFiles  map[string]string
		baseFiles    map[string]string
		changedFiles []string
		deletedFiles []string
		changedLines map[string]map[int]struct{}
		want         []advisoryCheckSnapshot
	}{
		// Removing a spec and reducing expect/toEqual counts flags test_removed, test_cases_reduced, and test_assertion_weakened.
		{
			name: "removed spec and weakened assertion in spec.ts",
			targetFiles: map[string]string{
				"web/src/app/a.spec.ts": `describe('A', () => {
  it('keeps', () => {
    expect(1).toBe(1);
  });
});
`,
			},
			baseFiles: map[string]string{
				"web/src/app/a.spec.ts": `describe('A', () => {
  it('keeps', () => {
    expect(1).toEqual(1);
  });
  it('drops', () => {
    expect(2).toBe(2);
  });
});
`,
			},
			changedFiles: []string{"web/src/app/a.spec.ts"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "web/src/app/a.spec.ts", Symbol: "a.spec.ts"},
				{Check: checkTestCasesReduced, File: "web/src/app/a.spec.ts", Symbol: "a.spec.ts"},
				{Check: checkTestRemoved, File: "web/src/app/a.spec.ts", Symbol: "drops"},
			},
		},
		// Adding xit/it.skip in code triggers test_skip_added while string literals containing xit do not.
		{
			name: "xit in code triggers while string literal does not",
			targetFiles: map[string]string{
				"web/src/app/b.spec.ts": `describe('B', () => {
  it('checks literal', () => {
    const s = 'xit("not a skip")';
    expect(s).toBeTruthy();
  });
  xit('skipped', () => {});
});
`,
			},
			baseFiles: map[string]string{
				"web/src/app/b.spec.ts": `describe('B', () => {
  it('checks literal', () => {
    expect('ok').toBeTruthy();
  });
});
`,
			},
			changedFiles: []string{"web/src/app/b.spec.ts"},
			changedLines: map[string]map[int]struct{}{"web/src/app/b.spec.ts": {3: {}, 6: {}}},
			want: []advisoryCheckSnapshot{
				{Check: checkTestSkipAdded, File: "web/src/app/b.spec.ts", Symbol: "xit"},
			},
		},
		// Deleting a spec.ts file reports test_removed for its removed spec entries.
		{
			name:        "deleted spec file emits test_removed",
			targetFiles: map[string]string{},
			baseFiles: map[string]string{
				"web/src/app/gone.spec.ts": "describe('G', () => { it('works', () => {}); });\n",
			},
			deletedFiles: []string{"web/src/app/gone.spec.ts"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestRemoved, File: "web/src/app/gone.spec.ts", Symbol: "works"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeRepoFiles(t, repoRoot, tc.targetFiles)
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: tc.changedFiles,
				DeletedFiles: tc.deletedFiles,
				ChangedLines: tc.changedLines,
				ReadBaseFile: func(rel string) ([]byte, bool, error) {
					s, ok := tc.baseFiles[rel]
					return []byte(s), ok, nil
				},
			}

			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			assertAdvisorySnapshots(t, items, tc.want)
		})
	}
}

// TestAnalyzeTSProdSuppressionAndDoubleCast verifies lexer-backed TS suppression_added and ts_double_cast_added with string/comment immunity.
func TestAnalyzeTSProdSuppressionAndDoubleCast(t *testing.T) {
	testCases := []struct {
		name      string
		baseSrc   string
		targetSrc string
		changed   map[int]struct{}
		want      []advisoryCheckSnapshot
	}{
		// String literals and prose comments mentioning suppression keywords or `as unknown as` produce zero findings.
		{
			name:    "string literals and prose comments do not trigger false positives",
			baseSrc: "export function demo(): string { return ''; }\n",
			targetSrc: `export function demo(): string {
  // Mentioning as unknown as, eslint-disable, @ts-ignore, @ts-expect-error, and CODEQUALITY_ALLOW in prose is safe.
  // eslint-disablement and @ts-ignored field docs are not directives.
  /*
   * Discussing CODEQUALITY_ALLOW directives inside a block comment is safe.
   */
  const text = 'eslint-disable-next-line @ts-ignore as unknown as Foo';
  return text;
}
`,
			changed: map[int]struct{}{2: {}, 3: {}, 4: {}, 5: {}, 6: {}, 7: {}},
			want:    nil,
		},
		// Unclosed quoted string ending with a trailing backslash at EOF does not panic while lexing.
		{
			name:      "unclosed string ending with trailing backslash does not panic",
			baseSrc:   "export const ok = 1;\n",
			targetSrc: "export const bad = \"unterminated\\",
			changed:   map[int]struct{}{1: {}},
			want:      nil,
		},
		// Real suppression comments and `as unknown as` in TS production code trigger suppression_added and ts_double_cast_added.
		{
			name:    "real suppression comment and double cast trigger advisories",
			baseSrc: "export function demo(x: object): string { return String(x); }\n",
			targetSrc: `export function demo(x: object): string {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const y = x as unknown as string;
  /* @ts-expect-error legacy */
  // @ts-ignore
  /*
   * CODEQUALITY_ALLOW(ts_func_lines, generated): table
   */
  /// @ts-ignore
  // @ts-ignore--reason
  // @ts-expect-error;reason
  /*@ts-ignore*/
  // CODEQUALITY_ALLOW (ts_func_lines, generated): spaced
  return y;
}
`,
			changed: map[int]struct{}{2: {}, 3: {}, 4: {}, 5: {}, 6: {}, 7: {}, 8: {}, 9: {}, 10: {}, 11: {}, 12: {}, 13: {}},
			want: []advisoryCheckSnapshot{
				{Check: checkSuppressionAdded, File: "web/src/app/demo.ts", Symbol: "<comment>"},
				{Check: checkTSDoubleCastAdded, File: "web/src/app/demo.ts", Symbol: "<type-cast>"},
				{Check: checkSuppressionAdded, File: "web/src/app/demo.ts", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "web/src/app/demo.ts", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "web/src/app/demo.ts", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "web/src/app/demo.ts", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "web/src/app/demo.ts", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "web/src/app/demo.ts", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "web/src/app/demo.ts", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "web/src/app/demo.ts", Symbol: "<comment>"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeRepoFiles(t, repoRoot, map[string]string{"web/src/app/demo.ts": tc.targetSrc})
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: []string{"web/src/app/demo.ts"},
				ChangedLines: map[string]map[int]struct{}{"web/src/app/demo.ts": tc.changed},
				ReadBaseFile: func(rel string) ([]byte, bool, error) {
					if rel == "web/src/app/demo.ts" {
						return []byte(tc.baseSrc), true, nil
					}
					return nil, false, nil
				},
			}

			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			assertAdvisorySnapshots(t, items, tc.want)
		})
	}
}
