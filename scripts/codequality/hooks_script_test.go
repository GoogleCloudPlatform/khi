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
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type hookScriptCase struct {
	name         string
	scriptName   string
	vcs          string
	baseEnv      string
	diffLines    []string
	presentFiles []string
	wantErr      bool
	wantErrSub   string
	wantCalls    []string
}

func TestHookScripts(t *testing.T) {
	repoRoot, err := gotestutil.ResolveRepoRoot("")
	if err != nil {
		t.Fatalf("ResolveRepoRoot() error: %v", err)
	}
	// Poison inherited Make/CLI variables to verify hook script tests stay isolated
	// when invoked under `make pre-push BASE=<parent-bookmark>`.
	t.Setenv("TARGET", "leaked-target")
	t.Setenv("GIT_INDEX_FILE", "/leaked/git/index")
	t.Setenv("CODEQUALITY_VCS", "git")
	t.Setenv("CODEQUALITY_BASE", "leaked-parent-bookmark")
	t.Setenv("CODEQUALITY_TARGET", "leaked-target")
	t.Setenv("CODEQUALITY_MAX", "42")
	t.Setenv("CODEQUALITY_TIMEOUT", "1m")

	testCases := []hookScriptCase{
		{
			name:       "pre-commit fails explicitly when jj working copy is empty",
			scriptName: "pre-commit.sh",
			vcs:        "jj",
			wantErr:    true,
			wantErrSub: "working copy (@) has no changes",
			wantCalls:  nil,
		},
		{
			name:         "pre-commit runs only markdown targets for markdown-only jj changes",
			scriptName:   "pre-commit.sh",
			vcs:          "jj",
			diffLines:    []string{".agents/skills/pr_quality_ci/SKILL.md"},
			presentFiles: []string{".agents/skills/pr_quality_ci/SKILL.md"},
			wantCalls: []string{
				"format-md",
				"lint-md",
			},
		},
		{
			name:       "pre-commit scopes Go lint packages to modified directories",
			scriptName: "pre-commit.sh",
			vcs:        "jj",
			diffLines: []string{
				"pkg/api/handler.go",
				"pkg/common/flag.go",
			},
			presentFiles: []string{
				"pkg/api/handler.go",
				"pkg/common/flag.go",
			},
			wantCalls: []string{
				"add-licenses",
				"format-go LINT_PKGS=./pkg/api,./pkg/common",
				"lint-go LINT_PKGS=./pkg/api,./pkg/common",
				"codequality-fast",
			},
		},
		{
			name:       "pre-push runs Go CI-parity gates with unlimited mutants",
			scriptName: "pre-push.sh",
			vcs:        "jj",
			diffLines:  []string{"pkg/api/handler.go"},
			wantCalls: []string{
				"check-licenses",
				"lint-go",
				"codequality-fast CODEQUALITY_VCS=jj CODEQUALITY_BASE=trunk() CODEQUALITY_TARGET=@",
				"test-go",
				"build-go-binaries",
				"codequality-go CODEQUALITY_MAX=0 CODEQUALITY_TIMEOUT=10m CODEQUALITY_VCS=jj CODEQUALITY_BASE=trunk() CODEQUALITY_TARGET=@",
			},
		},
		{
			name:       "pre-push runs Web and Markdown gates with stacked BASE override",
			scriptName: "pre-push.sh",
			vcs:        "jj",
			baseEnv:    "push-parentbookmark",
			diffLines: []string{
				"README.md",
				"web/src/app/view.component.ts",
			},
			wantCalls: []string{
				"check-licenses",
				"lint-md",
				"check-format-web",
				"lint-web",
				"build-web",
				"build-web-dev",
				"build-storybook",
				"test-web",
				"codequality-web CODEQUALITY_MAX=0 CODEQUALITY_TIMEOUT=10m CODEQUALITY_VCS=jj CODEQUALITY_BASE=push-parentbookmark CODEQUALITY_TARGET=@",
			},
		},
		{
			name:       "pre-push runs proto, Go, and Web gates when proto changes",
			scriptName: "pre-push.sh",
			vcs:        "jj",
			diffLines:  []string{"proto/khifile/v6/khifile.proto"},
			wantCalls: []string{
				"check-licenses",
				"lint-proto",
				"build-proto",
				"lint-go",
				"codequality-fast CODEQUALITY_VCS=jj CODEQUALITY_BASE=trunk() CODEQUALITY_TARGET=@",
				"check-format-web",
				"lint-web",
				"build-web",
				"build-web-dev",
				"build-storybook",
				"test-web",
				"test-go",
				"build-go-binaries",
				"codequality CODEQUALITY_MAX=0 CODEQUALITY_TIMEOUT=10m CODEQUALITY_VCS=jj CODEQUALITY_BASE=trunk() CODEQUALITY_TARGET=@",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotCalls, out, runErr := executeHookScriptCase(t, repoRoot, tc)
			if (runErr != nil) != tc.wantErr {
				t.Fatalf("script error = %v, wantErr %v\noutput:\n%s", runErr, tc.wantErr, out)
			}
			if tc.wantErrSub != "" && !strings.Contains(out, tc.wantErrSub) {
				t.Errorf("output = %q, want substring %q", out, tc.wantErrSub)
			}
			if diff := cmp.Diff(tc.wantCalls, gotCalls); diff != "" {
				t.Errorf("recorded make calls mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func executeHookScriptCase(t *testing.T, realRepoRoot string, tc hookScriptCase) ([]string, string, error) {
	t.Helper()
	fakeRepo := t.TempDir()
	binDir := t.TempDir()
	callsFile := filepath.Join(binDir, "make_calls.txt")
	diffFixture := filepath.Join(binDir, "diff_files.txt")

	materializeFakeRepo(t, fakeRepo, tc.vcs, tc.presentFiles)
	writeLinesFixture(t, diffFixture, tc.diffLines)
	writeStubCLIs(t, binDir, callsFile, diffFixture)

	scriptPath := filepath.Join(realRepoRoot, "scripts", tc.scriptName)
	cmd := exec.CommandContext(context.Background(), "/bin/bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"KHI_REPO_ROOT="+fakeRepo,
		"MAKE_BIN="+filepath.Join(binDir, "make"),
		"VCS="+tc.vcs,
		"BASE="+tc.baseEnv,
		"TARGET=",
		"GIT_INDEX_FILE=",
		"CODEQUALITY_VCS=",
		"CODEQUALITY_BASE=",
		"CODEQUALITY_TARGET=",
		"CODEQUALITY_MAX=",
		"CODEQUALITY_TIMEOUT=",
	)
	outBytes, runErr := cmd.CombinedOutput()
	return readRecordedLines(t, callsFile), string(outBytes), runErr
}

func materializeFakeRepo(t *testing.T, fakeRepo string, vcs string, files []string) {
	t.Helper()
	vcsDir := ".git"
	if vcs == "jj" {
		vcsDir = ".jj"
	}
	if err := os.MkdirAll(filepath.Join(fakeRepo, vcsDir), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error: %v", vcsDir, err)
	}
	for _, rel := range files {
		full := filepath.Join(fakeRepo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error: %v", full, err)
		}
		if err := os.WriteFile(full, []byte("package stub\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error: %v", full, err)
		}
	}
}

func writeLinesFixture(t *testing.T, path string, lines []string) {
	t.Helper()
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error: %v", path, err)
	}
}

func writeStubCLIs(t *testing.T, binDir string, callsFile string, diffFixture string) {
	t.Helper()
	fakeMake := `#!/bin/bash
set -euo pipefail
echo "$*" >> "` + callsFile + `"
`
	fakeJJ := `#!/bin/bash
set -euo pipefail
if [[ "${1:-}" == "diff" ]]; then
  cat "` + diffFixture + `"
fi
`
	if err := os.WriteFile(filepath.Join(binDir, "make"), []byte(fakeMake), 0o755); err != nil {
		t.Fatalf("WriteFile(fake make) error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "jj"), []byte(fakeJJ), 0o755); err != nil {
		t.Fatalf("WriteFile(fake jj) error: %v", err)
	}
}

func readRecordedLines(t *testing.T, callsFile string) []string {
	t.Helper()
	f, err := os.Open(callsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("Open(%s) error: %v", callsFile, err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if text := strings.TrimSpace(sc.Text()); text != "" {
			lines = append(lines, text)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}
	return lines
}
