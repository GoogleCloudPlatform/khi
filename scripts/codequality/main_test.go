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
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCLIEndToEndModes(t *testing.T) {
	repoDir := initTempGitRepo(t)
	summaryPath := filepath.Join(t.TempDir(), "step_summary.md")

	testCases := []struct {
		name       string
		args       []string
		wantCode   int
		wantErr    bool
		wantStdout string
	}{
		{
			name:       "fast mode passes clean git diff",
			args:       []string{"--repo-root=" + repoDir, "--vcs=git", "--base=HEAD~1", "--fast", "--color=never"},
			wantCode:   0,
			wantStdout: "PASS (0 violations)",
		},
		{
			name:       "full dynamic mode passes Go and writes github summary",
			args:       []string{"--repo-root=" + repoDir, "--vcs=git", "--base=HEAD~1", "--format=github", "--summary-file=" + summaryPath},
			wantCode:   0,
			wantStdout: "",
		},
		{
			name:       "lang ts on Go-only diff passes cleanly",
			args:       []string{"--repo-root=" + repoDir, "--lang=ts", "--vcs=git", "--base=HEAD~1", "--color=never"},
			wantCode:   0,
			wantStdout: "TypeScript Quality Report",
		},
		{
			name:       "lang ts with fast skips dynamic TS stage cleanly",
			args:       []string{"--repo-root=" + repoDir, "--lang=ts", "--fast", "--vcs=git", "--base=HEAD~1", "--summary-file=" + summaryPath},
			wantCode:   0,
			wantStdout: "no Go or TypeScript files changed",
		},
		{
			name:     "invalid lang flag returns code 2",
			args:     []string{"--lang=python"},
			wantCode: 2,
			wantErr:  true,
		},
		{
			name:     "invalid format flag returns code 2",
			args:     []string{"--format=xml"},
			wantCode: 2,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code, err := runCLI(context.Background(), cliEnv{
				args:   tc.args,
				stdout: &stdout,
				stderr: &stderr,
				getenv: func(string) string { return "" },
			})
			assertCLIOutcome(t, cliOutcomeCheck{
				code:       code,
				wantCode:   tc.wantCode,
				err:        err,
				wantErr:    tc.wantErr,
				stdout:     stdout.String(),
				stderr:     stderr.String(),
				wantStdout: tc.wantStdout,
			})
		})
	}

	summaryBytes, err := os.ReadFile(summaryPath)
	if err != nil || !strings.Contains(string(summaryBytes), "✅ PASS") {
		t.Errorf("expected PASS in summary file, got (%q, %v)", string(summaryBytes), err)
	}
}

type cliOutcomeCheck struct {
	code       int
	wantCode   int
	err        error
	wantErr    bool
	stdout     string
	stderr     string
	wantStdout string
}

func assertCLIOutcome(t *testing.T, c cliOutcomeCheck) {
	t.Helper()
	if c.code != c.wantCode {
		t.Errorf("code = %d, want %d (err=%v, stderr=%s, stdout=%s)", c.code, c.wantCode, c.err, c.stderr, c.stdout)
	}
	if (c.err != nil) != c.wantErr {
		t.Errorf("err = %v, wantErr %v", c.err, c.wantErr)
	}
	if c.wantStdout != "" && !strings.Contains(c.stdout, c.wantStdout) {
		t.Errorf("stdout missing %q:\n%s", c.wantStdout, c.stdout)
	}
}

func TestRunCLIReturnsExitCodeOneOnViolation(t *testing.T) {
	repoDir := initTempGitRepoWithViolation(t)
	summaryPath := filepath.Join(t.TempDir(), "viol_summary.md")

	testCases := []struct {
		name      string
		extraArgs []string
		wantCode  int
	}{
		{
			name:      "dynamic mode evaluates lazy base overlay sorts multiple violations and exits 1",
			extraArgs: []string{"--lang=go", "--fail-on-violations=true", "--summary-file=" + summaryPath},
			wantCode:  1,
		},
		{
			name:      "fast advisory mode exits 0 when fail-on-violations is false",
			extraArgs: []string{"--fast", "--fail-on-violations=false"},
			wantCode:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := append([]string{"--repo-root=" + repoDir, "--vcs=git", "--base=HEAD~1"}, tc.extraArgs...)
			code, err := runCLI(context.Background(), cliEnv{
				args:   args,
				stdout: &stdout,
				stderr: &stderr,
				getenv: func(string) string { return "" },
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code != tc.wantCode {
				t.Errorf("code = %d, want %d (stdout=%s)", code, tc.wantCode, stdout.String())
			}
		})
	}
}

func initTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitCmd(t, dir, "init", "-b", "main")
	writeFiles(t, dir, map[string]string{
		"go.mod": "module example.com/climod\n\ngo 1.24\n",
		"pkg/svc/svc.go": `package svc

func Base() int { return 1 }
`,
	})
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "base")

	writeFiles(t, dir, map[string]string{
		"pkg/svc/svc.go": `package svc

func Base() int { return 1 }

func Double(x int) int { return x * 2 }
`,
		"pkg/svc/svc_test.go": `package svc

import "testing"

func TestDouble(t *testing.T) {
	if Double(3) != 6 {
		t.Fatal("bad")
	}
}
`,
	})
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "clean change")
	return dir
}

func initTempGitRepoWithViolation(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitCmd(t, dir, "init", "-b", "main")
	writeFiles(t, dir, map[string]string{
		"go.mod": "module example.com/violmod\n\ngo 1.24\n",
		"pkg/svc/svc.go": `package svc

func Legacy(x int) int { return x }
`,
		"pkg/svc/svc_test.go": `package svc

import "testing"

func TestLegacy(t *testing.T) {
	if Legacy(2) != 2 {
		t.Fatal("bad")
	}
}
`,
	})
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "base")

	writeFiles(t, dir, map[string]string{
		"pkg/svc/svc.go": `package svc

func Legacy(x int) int {
	if x < 0 {
		return -1
	}
	return x
}
`,
		"pkg/svc/util.go": `package svc

func Helper() int { return 1 }
`,
	})
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "add violations")
	return dir
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}
