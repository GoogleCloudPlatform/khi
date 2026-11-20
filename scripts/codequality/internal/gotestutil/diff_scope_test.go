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
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TestCollectChangedLinesJJCurrentOnly verifies that CollectChangedLines with VCS "jj" scopes to @-..@ when CurrentOnly is true and trunk() when false.
func TestCollectChangedLinesJJCurrentOnly(t *testing.T) {
	testCases := []struct {
		name          string
		currentOnly   bool
		wantBaseLabel string
		wantFromArg   string
	}{
		// Uses @- as base revision when CurrentOnly is true and Base is empty.
		{
			name:          "current only uses parent @-",
			currentOnly:   true,
			wantBaseLabel: "@-",
			wantFromArg:   "fork_point((@) | (@-))",
		},
		// Uses trunk() as base revision when CurrentOnly is false and Base is empty.
		{
			name:          "default uses trunk()",
			currentOnly:   false,
			wantBaseLabel: "trunk()",
			wantFromArg:   "fork_point((@) | (trunk()))",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			argsLog := filepath.Join(binDir, "jj_args.log")
			installFakeJJ(t, binDir, argsLog, "pkg/alpha/demo.go")
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			got, err := CollectChangedLines(context.Background(), t.TempDir(), DiffOptions{
				VCS:         "jj",
				CurrentOnly: tc.currentOnly,
			})

			if err != nil {
				t.Fatalf("CollectChangedLines() unexpected error: %v", err)
			}
			if got.BaseLabel != tc.wantBaseLabel {
				t.Errorf("BaseLabel = %q, want %q", got.BaseLabel, tc.wantBaseLabel)
			}
			if got.ResolvedBaseRev != "jj-commit-id-123" {
				t.Errorf("ResolvedBaseRev = %q, want %q", got.ResolvedBaseRev, "jj-commit-id-123")
			}
			loggedBytes, readErr := os.ReadFile(argsLog)
			if readErr != nil {
				t.Fatalf("failed to read fake jj args log: %v", readErr)
			}
			if !strings.Contains(string(loggedBytes), tc.wantFromArg) {
				t.Errorf("jj diff args %q did not contain %q", string(loggedBytes), tc.wantFromArg)
			}
		})
	}
}

// TestCollectChangedLinesCurrentOnlyDirtyWithTarget verifies that explicit Target overrides working-tree diffing on a dirty git repo when CurrentOnly is true.
func TestCollectChangedLinesCurrentOnlyDirtyWithTarget(t *testing.T) {
	repoDir, revisionByAnchor := initContractCurrentOnlyRepo(t, true)

	got, err := CollectChangedLines(context.Background(), repoDir, DiffOptions{
		VCS:         "git",
		CurrentOnly: true,
		Target:      "HEAD",
	})

	if err != nil {
		t.Fatalf("CollectChangedLines() unexpected error: %v", err)
	}
	if got.BaseLabel != "HEAD" {
		t.Errorf("BaseLabel = %q, want %q", got.BaseLabel, "HEAD")
	}
	if got.ResolvedBaseRev != revisionByAnchor["head"] {
		t.Errorf("ResolvedBaseRev = %q, want %q", got.ResolvedBaseRev, revisionByAnchor["head"])
	}
	if len(got.ChangedFiles) != 0 {
		t.Errorf("ChangedFiles = %v, want empty when diffing HEAD..HEAD", got.ChangedFiles)
	}
}

// TestFilterDiffByPathsEdgeCases verifies that FilterDiffByPaths handles whitespace-only entries and catch-all root patterns.
func TestFilterDiffByPathsEdgeCases(t *testing.T) {
	testCases := []struct {
		name  string
		paths []string
		want  DiffResult
	}{
		// Ignores blank entries and matches nothing when no non-empty path patterns remain.
		{
			name:  "whitespace only paths yield empty collections",
			paths: []string{"", "   "},
			want: DiffResult{
				VCS:             "jj",
				BaseLabel:       "trunk()",
				ResolvedBaseRev: "rev-base-123",
			},
		},
		// Returns the full DiffResult unchanged when a catch-all root pattern follows a blank entry.
		{
			name:  "catch-all root pattern returns input unchanged",
			paths: []string{"", "."},
			want:  sampleFullDiffResult(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterDiffByPaths(sampleFullDiffResult(), tc.paths)

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FilterDiffByPaths() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestIntersectDiffChangedSinceJJAndErrors verifies jj diff --name-only handling and unsupported VCS error reporting.
func TestIntersectDiffChangedSinceJJAndErrors(t *testing.T) {
	testCases := []struct {
		name       string
		targetRev  string
		wantToFlag string
	}{
		// Defaults empty targetRev to @ when invoking jj diff --name-only.
		{
			name:       "empty target defaults to @",
			targetRev:  "",
			wantToFlag: "--to @",
		},
		// Passes explicit targetRev to jj diff --name-only when non-empty.
		{
			name:       "explicit target passes through",
			targetRev:  "rev-target",
			wantToFlag: "--to rev-target",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			argsLog := filepath.Join(binDir, "jj_args.log")
			installFakeJJ(t, binDir, argsLog, "pkg/a/foo.go")
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			baseRes := sampleFullDiffResult()

			got, err := IntersectDiffChangedSince(context.Background(), t.TempDir(), baseRes, "rev-since", tc.targetRev)

			if err != nil {
				t.Fatalf("IntersectDiffChangedSince() unexpected error: %v", err)
			}
			want := DiffResult{
				VCS:             "jj",
				BaseLabel:       "trunk()",
				ResolvedBaseRev: "rev-base-123",
				ChangedLines: map[string]map[int]struct{}{
					"pkg/a/foo.go": {10: {}, 12: {}},
				},
				ChangedFiles: []string{"pkg/a/foo.go"},
			}
			if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("IntersectDiffChangedSince() mismatch (-want +got):\n%s", diff)
			}
			loggedBytes, readErr := os.ReadFile(argsLog)
			if readErr != nil {
				t.Fatalf("failed to read fake jj args log: %v", readErr)
			}
			if !strings.Contains(string(loggedBytes), tc.wantToFlag) {
				t.Errorf("jj diff args %q did not contain %q", string(loggedBytes), tc.wantToFlag)
			}
		})
	}

	t.Run("unsupported vcs rejected by collectChangedFileSetBetween", func(t *testing.T) {
		_, err := collectChangedFileSetBetween(context.Background(), t.TempDir(), "svn", "r1", "r2")

		if err == nil {
			t.Errorf("collectChangedFileSetBetween() error = nil, want non-nil")
		}
	})
}

// TestIntersectDiffChangedSinceGitTargetBounds verifies that explicit targetRev prevents later working-tree edits from leaking into git since-diffs.
func TestIntersectDiffChangedSinceGitTargetBounds(t *testing.T) {
	repoDir, revisionByAnchor := initContractSinceRepo(t)
	writeContractRepoFile(t, repoDir, "pkg/demo/commit1_only.go", `package demo

func CommitOne() int { return 99 }
`)
	baseResult, err := CollectChangedLines(context.Background(), repoDir, DiffOptions{
		VCS:  "git",
		Base: "main",
	})
	if err != nil {
		t.Fatalf("failed to collect base diff: %v", err)
	}

	got, err := IntersectDiffChangedSince(context.Background(), repoDir, baseResult, revisionByAnchor["commit1"], revisionByAnchor["commit2"])

	if err != nil {
		t.Fatalf("IntersectDiffChangedSince() unexpected error: %v", err)
	}
	wantChanged := []string{
		"pkg/demo/commit2_only.go",
		"pkg/demo/commit2_only_test.go",
		"pkg/demo/multi_touch.go",
	}
	if diff := cmp.Diff(wantChanged, got.ChangedFiles); diff != "" {
		t.Errorf("IntersectDiffChangedSince().ChangedFiles mismatch (-want +got):\n%s", diff)
	}
}

func installFakeJJ(t *testing.T, binDir string, argsLog string, changedFile string) {
	t.Helper()
	script := `#!/bin/sh
echo "$*" >> "` + argsLog + `"
case "$1" in
  diff)
    for arg in "$@"; do
      if [ "$arg" = "--name-only" ]; then
        echo "` + changedFile + `"
        exit 0
      fi
    done
    cat <<'EOF'
diff --git a/` + changedFile + ` b/` + changedFile + `
index 1111111..2222222 100644
--- a/` + changedFile + `
+++ b/` + changedFile + `
@@ -0,0 +1,1 @@
+package alpha
EOF
    exit 0
    ;;
  log)
    echo "jj-commit-id-123"
    exit 0
    ;;
esac
exit 1
`
	jjPath := filepath.Join(binDir, "jj")
	if err := os.WriteFile(jjPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake jj script: %v", err)
	}
}
