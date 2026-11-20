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

package gotestutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TestCollectChangedLinesCurrentOnlyContract verifies that DiffOptions.CurrentOnly scopes git diffs to the active change unless an explicit Base is given.
func TestCollectChangedLinesCurrentOnlyContract(t *testing.T) {
	testCases := []struct {
		name           string
		makeDirty      bool
		opts           DiffOptions
		wantBaseAnchor string
		wantDiff       DiffResult
	}{
		// Collects only uncommitted working-tree edits against HEAD when Base is empty and CurrentOnly is true.
		{
			name:           "dirty working tree collects uncommitted edits vs HEAD",
			makeDirty:      true,
			opts:           DiffOptions{VCS: "git", CurrentOnly: true},
			wantBaseAnchor: "head",
			wantDiff: DiffResult{
				VCS: "git",
				ChangedLines: map[string]map[int]struct{}{
					"pkg/alpha/tracked.go": {2: {}, 3: {}},
				},
				ChangedFiles: []string{"pkg/alpha/tracked.go"},
			},
		},
		// Collects only HEAD~1..HEAD when the working tree is clean, Base is empty, and CurrentOnly is true.
		{
			name:           "clean working tree collects HEAD~1..HEAD",
			makeDirty:      false,
			opts:           DiffOptions{VCS: "git", CurrentOnly: true},
			wantBaseAnchor: "headParent",
			wantDiff: DiffResult{
				VCS: "git",
				ChangedLines: map[string]map[int]struct{}{
					"pkg/alpha/latest.go": {1: {}, 2: {}, 3: {}},
				},
				ChangedFiles: []string{"pkg/alpha/latest.go"},
			},
		},
		// Diffs against the explicit Base revision even when CurrentOnly is true.
		{
			name:           "explicit Base overrides CurrentOnly",
			makeDirty:      false,
			opts:           DiffOptions{VCS: "git", Base: "main", CurrentOnly: true},
			wantBaseAnchor: "main",
			wantDiff: DiffResult{
				VCS:       "git",
				BaseLabel: "main",
				ChangedLines: map[string]map[int]struct{}{
					"pkg/alpha/earlier.go": {1: {}, 2: {}, 3: {}},
					"pkg/alpha/latest.go":  {1: {}, 2: {}, 3: {}},
				},
				ChangedFiles: []string{"pkg/alpha/earlier.go", "pkg/alpha/latest.go"},
			},
		},
		// Preserves default base branch resolution when Base is empty and CurrentOnly is false.
		{
			name:           "CurrentOnly false keeps default base diff",
			makeDirty:      false,
			opts:           DiffOptions{VCS: "git", CurrentOnly: false},
			wantBaseAnchor: "main",
			wantDiff: DiffResult{
				VCS:       "git",
				BaseLabel: "main",
				ChangedLines: map[string]map[int]struct{}{
					"pkg/alpha/earlier.go": {1: {}, 2: {}, 3: {}},
					"pkg/alpha/latest.go":  {1: {}, 2: {}, 3: {}},
				},
				ChangedFiles: []string{"pkg/alpha/earlier.go", "pkg/alpha/latest.go"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoDir, revisionByAnchor := initContractCurrentOnlyRepo(t, tc.makeDirty)
			wantBaseSHA := revisionByAnchor[tc.wantBaseAnchor]

			got, err := CollectChangedLines(context.Background(), repoDir, tc.opts)

			if err != nil {
				t.Fatalf("CollectChangedLines() unexpected error: %v", err)
			}
			gotResolvedSHA := runContractGit(t, repoDir, "rev-parse", "--verify", got.ResolvedBaseRev)
			if gotResolvedSHA != wantBaseSHA {
				t.Errorf("CollectChangedLines().ResolvedBaseRev %q resolved to %q, want %q (%s)", got.ResolvedBaseRev, gotResolvedSHA, wantBaseSHA, tc.wantBaseAnchor)
			}
			if got.BaseLabel == "" {
				t.Errorf("CollectChangedLines().BaseLabel = %q, want non-empty", got.BaseLabel)
			}
			if tc.wantDiff.BaseLabel != "" && got.BaseLabel != tc.wantDiff.BaseLabel {
				t.Errorf("CollectChangedLines().BaseLabel = %q, want %q", got.BaseLabel, tc.wantDiff.BaseLabel)
			}
			if diff := cmp.Diff(tc.wantDiff, got, cmpopts.IgnoreFields(DiffResult{}, "BaseLabel", "ResolvedBaseRev"), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("CollectChangedLines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestFilterDiffByPathsContract verifies that FilterDiffByPaths narrows all DiffResult maps and slices by path patterns while preserving VCS metadata.
func TestFilterDiffByPathsContract(t *testing.T) {
	testCases := []struct {
		name  string
		paths []string
		want  DiffResult
	}{
		// Returns the original DiffResult unchanged when paths is empty.
		{
			name:  "empty paths returns input unchanged",
			paths: nil,
			want:  sampleFullDiffResult(),
		},
		// Keeps only the exact matching relative file path across all DiffResult maps and slices while preserving metadata.
		{
			name:  "exact relative file path",
			paths: []string{"pkg/a/foo.go"},
			want: DiffResult{
				VCS:             "jj",
				BaseLabel:       "trunk()",
				ResolvedBaseRev: "rev-base-123",
				ChangedLines: map[string]map[int]struct{}{
					"pkg/a/foo.go": {10: {}, 12: {}},
				},
				ChangedFiles: []string{"pkg/a/foo.go"},
			},
		},
		// Yields empty maps and slices without panicking or leaking files when no paths match.
		{
			name:  "non-matching path yields empty collections",
			paths: []string{"pkg/doesnotexist"},
			want: DiffResult{
				VCS:             "jj",
				BaseLabel:       "trunk()",
				ResolvedBaseRev: "rev-base-123",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputDiff := sampleFullDiffResult()

			got := FilterDiffByPaths(inputDiff, tc.paths)

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FilterDiffByPaths() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	// Filters Go, TypeScript, and normalized recursive/trailing-slash prefixes while excluding sibling prefix directories.
	t.Run("directory prefixes and normalized patterns isolate Go and TypeScript siblings", func(t *testing.T) {
		prefixChecks := []struct {
			paths []string
			want  DiffResult
		}{
			{
				paths: []string{"pkg/a"},
				want: DiffResult{
					VCS:             "jj",
					BaseLabel:       "trunk()",
					ResolvedBaseRev: "rev-base-123",
					ChangedLines: map[string]map[int]struct{}{
						"pkg/a/foo.go":     {10: {}, 12: {}},
						"pkg/a/sub/bar.go": {4: {}},
					},
					ChangedTestLines: map[string]map[int]struct{}{
						"pkg/a/foo_test.go": {20: {}},
					},
					DeletedGoFiles:      []string{"pkg/a/deleted.go"},
					OtherChangedGoFiles: []string{"pkg/a/zzz_generated.go"},
					ChangedFiles: []string{
						"pkg/a/README.md",
						"pkg/a/foo.go",
						"pkg/a/foo_test.go",
						"pkg/a/sub/bar.go",
						"pkg/a/zzz_generated.go",
					},
					DeletedFiles: []string{
						"pkg/a/OLD.md",
						"pkg/a/deleted.go",
					},
				},
			},
			{
				paths: []string{"web/src/app/feature"},
				want: DiffResult{
					VCS:             "jj",
					BaseLabel:       "trunk()",
					ResolvedBaseRev: "rev-base-123",
					TSChangedLines: map[string]map[int]struct{}{
						"web/src/app/feature/panel.ts": {15: {}, 16: {}},
					},
					TSChangedTestLines: map[string]map[int]struct{}{
						"web/src/app/feature/panel.spec.ts": {30: {}},
					},
					ChangedFiles: []string{
						"web/src/app/feature/panel.spec.ts",
						"web/src/app/feature/panel.ts",
						"web/src/app/feature/styles.scss",
					},
					DeletedFiles: []string{
						"web/src/app/feature/legacy.ts",
					},
				},
			},
			{
				paths: []string{"./pkg/a/...", "web/src/app/feature/"},
				want: DiffResult{
					VCS:             "jj",
					BaseLabel:       "trunk()",
					ResolvedBaseRev: "rev-base-123",
					ChangedLines: map[string]map[int]struct{}{
						"pkg/a/foo.go":     {10: {}, 12: {}},
						"pkg/a/sub/bar.go": {4: {}},
					},
					ChangedTestLines: map[string]map[int]struct{}{
						"pkg/a/foo_test.go": {20: {}},
					},
					TSChangedLines: map[string]map[int]struct{}{
						"web/src/app/feature/panel.ts": {15: {}, 16: {}},
					},
					TSChangedTestLines: map[string]map[int]struct{}{
						"web/src/app/feature/panel.spec.ts": {30: {}},
					},
					DeletedGoFiles:      []string{"pkg/a/deleted.go"},
					OtherChangedGoFiles: []string{"pkg/a/zzz_generated.go"},
					ChangedFiles: []string{
						"pkg/a/README.md",
						"pkg/a/foo.go",
						"pkg/a/foo_test.go",
						"pkg/a/sub/bar.go",
						"pkg/a/zzz_generated.go",
						"web/src/app/feature/panel.spec.ts",
						"web/src/app/feature/panel.ts",
						"web/src/app/feature/styles.scss",
					},
					DeletedFiles: []string{
						"pkg/a/OLD.md",
						"pkg/a/deleted.go",
						"web/src/app/feature/legacy.ts",
					},
				},
			},
		}

		for _, check := range prefixChecks {
			got := FilterDiffByPaths(sampleFullDiffResult(), check.paths)
			if diff := cmp.Diff(check.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FilterDiffByPaths(%v) mismatch (-want +got):\n%s", check.paths, diff)
			}
		}
	})
}

// TestIntersectDiffChangedSinceContract verifies that IntersectDiffChangedSince filters target files to sinceRev..targetRev while keeping BaseLabel and ResolvedBaseRev anchored to base.
func TestIntersectDiffChangedSinceContract(t *testing.T) {
	repoDir, revisionByAnchor := initContractSinceRepo(t)
	baseResult, err := CollectChangedLines(context.Background(), repoDir, DiffOptions{
		VCS:  "git",
		Base: "main",
	})
	if err != nil {
		t.Fatalf("failed to collect base diff for setup: %v", err)
	}

	testCases := []struct {
		name         string
		sinceAnchor  string
		targetAnchor string
		want         DiffResult
	}{
		// Returns baseResult unchanged when sinceRev is empty.
		{
			name:         "empty sinceRev returns baseResult unchanged",
			sinceAnchor:  "",
			targetAnchor: "",
			want: DiffResult{
				VCS:       "git",
				BaseLabel: "main",
				ChangedLines: map[string]map[int]struct{}{
					"pkg/demo/commit1_only.go": {1: {}, 2: {}, 3: {}},
					"pkg/demo/commit2_only.go": {1: {}, 2: {}, 3: {}},
					"pkg/demo/multi_touch.go":  {2: {}, 3: {}, 4: {}},
				},
				ChangedTestLines: map[string]map[int]struct{}{
					"pkg/demo/commit2_only_test.go": {1: {}, 2: {}, 3: {}, 4: {}, 5: {}},
				},
				DeletedGoFiles: []string{
					"pkg/demo/del_c1.go",
					"pkg/demo/del_c2.go",
				},
				ChangedFiles: []string{
					"pkg/demo/commit1_only.go",
					"pkg/demo/commit2_only.go",
					"pkg/demo/commit2_only_test.go",
					"pkg/demo/multi_touch.go",
				},
				DeletedFiles: []string{
					"pkg/demo/del_c1.go",
					"pkg/demo/del_c2.go",
				},
			},
		},
		// Excludes files changed only in commit1 while retaining files touched in commit2 and keeping base anchor.
		{
			name:         "since commit1 with explicit commit2 target excludes commit1-only files",
			sinceAnchor:  "commit1",
			targetAnchor: "commit2",
			want: DiffResult{
				VCS:       "git",
				BaseLabel: "main",
				ChangedLines: map[string]map[int]struct{}{
					"pkg/demo/commit2_only.go": {1: {}, 2: {}, 3: {}},
					"pkg/demo/multi_touch.go":  {2: {}, 3: {}, 4: {}},
				},
				ChangedTestLines: map[string]map[int]struct{}{
					"pkg/demo/commit2_only_test.go": {1: {}, 2: {}, 3: {}, 4: {}, 5: {}},
				},
				DeletedGoFiles: []string{
					"pkg/demo/del_c2.go",
				},
				ChangedFiles: []string{
					"pkg/demo/commit2_only.go",
					"pkg/demo/commit2_only_test.go",
					"pkg/demo/multi_touch.go",
				},
				DeletedFiles: []string{
					"pkg/demo/del_c2.go",
				},
			},
		},
		// Uses HEAD or working tree when targetRev is empty while excluding commit1-only files and preserving base anchor.
		{
			name:         "since commit1 with empty target uses current HEAD",
			sinceAnchor:  "commit1",
			targetAnchor: "",
			want: DiffResult{
				VCS:       "git",
				BaseLabel: "main",
				ChangedLines: map[string]map[int]struct{}{
					"pkg/demo/commit2_only.go": {1: {}, 2: {}, 3: {}},
					"pkg/demo/multi_touch.go":  {2: {}, 3: {}, 4: {}},
				},
				ChangedTestLines: map[string]map[int]struct{}{
					"pkg/demo/commit2_only_test.go": {1: {}, 2: {}, 3: {}, 4: {}, 5: {}},
				},
				DeletedGoFiles: []string{
					"pkg/demo/del_c2.go",
				},
				ChangedFiles: []string{
					"pkg/demo/commit2_only.go",
					"pkg/demo/commit2_only_test.go",
					"pkg/demo/multi_touch.go",
				},
				DeletedFiles: []string{
					"pkg/demo/del_c2.go",
				},
			},
		},
		// Excludes files changed before base even when sinceRev precedes base.
		{
			name:         "sinceRev before base does not add pre-base files",
			sinceAnchor:  "beforeBase",
			targetAnchor: "commit2",
			want: DiffResult{
				VCS:       "git",
				BaseLabel: "main",
				ChangedLines: map[string]map[int]struct{}{
					"pkg/demo/commit1_only.go": {1: {}, 2: {}, 3: {}},
					"pkg/demo/commit2_only.go": {1: {}, 2: {}, 3: {}},
					"pkg/demo/multi_touch.go":  {2: {}, 3: {}, 4: {}},
				},
				ChangedTestLines: map[string]map[int]struct{}{
					"pkg/demo/commit2_only_test.go": {1: {}, 2: {}, 3: {}, 4: {}, 5: {}},
				},
				DeletedGoFiles: []string{
					"pkg/demo/del_c1.go",
					"pkg/demo/del_c2.go",
				},
				ChangedFiles: []string{
					"pkg/demo/commit1_only.go",
					"pkg/demo/commit2_only.go",
					"pkg/demo/commit2_only_test.go",
					"pkg/demo/multi_touch.go",
				},
				DeletedFiles: []string{
					"pkg/demo/del_c1.go",
					"pkg/demo/del_c2.go",
				},
			},
		},
		// Yields empty collections while keeping BaseLabel and ResolvedBaseRev anchored to base when nothing changed since sinceRev.
		{
			name:         "since commit2 at commit2 yields empty collections anchored to base",
			sinceAnchor:  "commit2",
			targetAnchor: "commit2",
			want: DiffResult{
				VCS:       "git",
				BaseLabel: "main",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sinceRev := revisionByAnchor[tc.sinceAnchor]
			targetRev := revisionByAnchor[tc.targetAnchor]

			got, err := IntersectDiffChangedSince(context.Background(), repoDir, baseResult, sinceRev, targetRev)

			if err != nil {
				t.Fatalf("IntersectDiffChangedSince() unexpected error: %v", err)
			}
			if got.ResolvedBaseRev != revisionByAnchor["base"] {
				t.Errorf("IntersectDiffChangedSince().ResolvedBaseRev = %q, want %q", got.ResolvedBaseRev, revisionByAnchor["base"])
			}
			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreFields(DiffResult{}, "ResolvedBaseRev"), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("IntersectDiffChangedSince() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestIntersectDiffChangedSinceInvalidRevisionContract verifies that IntersectDiffChangedSince returns an error when sinceRev cannot be resolved.
func TestIntersectDiffChangedSinceInvalidRevisionContract(t *testing.T) {
	repoDir, _ := initContractSinceRepo(t)
	baseResult, err := CollectChangedLines(context.Background(), repoDir, DiffOptions{
		VCS:  "git",
		Base: "main",
	})
	if err != nil {
		t.Fatalf("failed to collect base diff for setup: %v", err)
	}

	_, gotErr := IntersectDiffChangedSince(context.Background(), repoDir, baseResult, "refs/heads/does-not-exist", "HEAD")

	if gotErr == nil {
		t.Errorf("IntersectDiffChangedSince() error = nil, want non-nil for invalid sinceRev")
	}
}

func sampleFullDiffResult() DiffResult {
	return DiffResult{
		VCS:             "jj",
		BaseLabel:       "trunk()",
		ResolvedBaseRev: "rev-base-123",
		ChangedLines: map[string]map[int]struct{}{
			"pkg/a/foo.go":      {10: {}, 12: {}},
			"pkg/a/sub/bar.go":  {4: {}},
			"pkg/ab/sibling.go": {8: {}},
		},
		ChangedTestLines: map[string]map[int]struct{}{
			"pkg/a/foo_test.go": {20: {}},
			"pkg/b/baz_test.go": {5: {}},
		},
		TSChangedLines: map[string]map[int]struct{}{
			"web/src/app/feature/panel.ts":       {15: {}, 16: {}},
			"web/src/app/feature_extra/other.ts": {3: {}},
		},
		TSChangedTestLines: map[string]map[int]struct{}{
			"web/src/app/feature/panel.spec.ts": {30: {}},
			"web/src/app/other/other.spec.ts":   {7: {}},
		},
		DeletedGoFiles: []string{
			"pkg/a/deleted.go",
			"pkg/b/deleted.go",
		},
		OtherChangedGoFiles: []string{
			"pkg/a/zzz_generated.go",
			"pkg/generated/service.pb.go",
		},
		ChangedFiles: []string{
			"pkg/a/README.md",
			"pkg/a/foo.go",
			"pkg/a/foo_test.go",
			"pkg/a/sub/bar.go",
			"pkg/a/zzz_generated.go",
			"pkg/ab/sibling.go",
			"pkg/b/baz_test.go",
			"pkg/generated/service.pb.go",
			"web/src/app/feature/panel.spec.ts",
			"web/src/app/feature/panel.ts",
			"web/src/app/feature/styles.scss",
			"web/src/app/feature_extra/other.ts",
			"web/src/app/other/other.spec.ts",
		},
		DeletedFiles: []string{
			"pkg/a/OLD.md",
			"pkg/a/deleted.go",
			"pkg/b/deleted.go",
			"web/src/app/feature/legacy.ts",
			"web/src/app/other/legacy.ts",
		},
	}
}

func initContractCurrentOnlyRepo(t *testing.T, makeDirty bool) (string, map[string]string) {
	t.Helper()
	repoDir := t.TempDir()
	runContractGit(t, repoDir, "init", "-b", "main")

	writeContractRepoFile(t, repoDir, "pkg/alpha/tracked.go", `package alpha
`)
	writeContractRepoFile(t, repoDir, "pkg/alpha/base.go", `package alpha
`)
	mainSHA := commitContractChanges(t, repoDir, "base")

	runContractGit(t, repoDir, "checkout", "-b", "feat")
	writeContractRepoFile(t, repoDir, "pkg/alpha/earlier.go", `package alpha

func Earlier() int { return 1 }
`)
	headParentSHA := commitContractChanges(t, repoDir, "commit1")

	writeContractRepoFile(t, repoDir, "pkg/alpha/latest.go", `package alpha

func Latest() int { return 2 }
`)
	headSHA := commitContractChanges(t, repoDir, "commit2")

	if makeDirty {
		writeContractRepoFile(t, repoDir, "pkg/alpha/tracked.go", `package alpha

func Dirty() int { return 3 }
`)
	}

	return repoDir, map[string]string{
		"main":       mainSHA,
		"headParent": headParentSHA,
		"head":       headSHA,
	}
}

func initContractSinceRepo(t *testing.T) (string, map[string]string) {
	t.Helper()
	repoDir := t.TempDir()
	runContractGit(t, repoDir, "init", "-b", "main")

	writeContractRepoFile(t, repoDir, "pkg/demo/before_base.go", `package demo
`)
	writeContractRepoFile(t, repoDir, "pkg/demo/del_c1.go", `package demo
`)
	writeContractRepoFile(t, repoDir, "pkg/demo/del_c2.go", `package demo
`)
	writeContractRepoFile(t, repoDir, "pkg/demo/multi_touch.go", `package demo
`)
	beforeBaseSHA := commitContractChanges(t, repoDir, "before base")

	writeContractRepoFile(t, repoDir, "pkg/demo/before_base.go", `package demo

func BeforeBase() int { return 0 }
`)
	baseSHA := commitContractChanges(t, repoDir, "base")

	runContractGit(t, repoDir, "checkout", "-b", "feat")
	writeContractRepoFile(t, repoDir, "pkg/demo/commit1_only.go", `package demo

func CommitOne() int { return 1 }
`)
	removeContractRepoFile(t, repoDir, "pkg/demo/del_c1.go")
	writeContractRepoFile(t, repoDir, "pkg/demo/multi_touch.go", `package demo

func MultiOne() int { return 1 }
`)
	commit1SHA := commitContractChanges(t, repoDir, "commit1")

	writeContractRepoFile(t, repoDir, "pkg/demo/commit2_only.go", `package demo

func CommitTwo() int { return 2 }
`)
	writeContractRepoFile(t, repoDir, "pkg/demo/commit2_only_test.go", `package demo

import "testing"

func TestCommitTwo(t *testing.T) {}
`)
	removeContractRepoFile(t, repoDir, "pkg/demo/del_c2.go")
	writeContractRepoFile(t, repoDir, "pkg/demo/multi_touch.go", `package demo

func MultiOne() int { return 1 }
func MultiTwo() int { return 2 }
`)
	commit2SHA := commitContractChanges(t, repoDir, "commit2")

	return repoDir, map[string]string{
		"beforeBase": beforeBaseSHA,
		"base":       baseSHA,
		"commit1":    commit1SHA,
		"commit2":    commit2SHA,
	}
}

func writeContractRepoFile(t *testing.T, repoDir string, relativePath string, content string) {
	t.Helper()
	fullPath := filepath.Join(repoDir, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file %s: %v", relativePath, err)
	}
}

func removeContractRepoFile(t *testing.T, repoDir string, relativePath string) {
	t.Helper()
	fullPath := filepath.Join(repoDir, filepath.FromSlash(relativePath))
	if err := os.Remove(fullPath); err != nil {
		t.Fatalf("failed to remove file %s: %v", relativePath, err)
	}
}

func commitContractChanges(t *testing.T, repoDir string, message string) string {
	t.Helper()
	runContractGit(t, repoDir, "add", "-A")
	runContractGit(t, repoDir, "commit", "-m", message)
	return runContractGit(t, repoDir, "rev-parse", "HEAD")
}

func runContractGit(t *testing.T, repoDir string, args ...string) string {
	t.Helper()
	gitArgs := append([]string{"-c", "commit.gpgsign=false"}, args...)
	command := exec.CommandContext(context.Background(), "git", gitArgs...)
	command.Dir = repoDir
	command.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	outputBytes, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(outputBytes))
	}
	return strings.TrimSpace(string(outputBytes))
}
