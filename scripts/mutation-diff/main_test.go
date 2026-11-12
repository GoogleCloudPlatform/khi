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

func TestRunGitEndToEndAndThreshold(t *testing.T) {
	repoDir := t.TempDir()

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = repoDir
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

	runGit("init", "-b", "main")
	goModContent := `module example.com/clitest

go 1.23
`
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte(goModContent), 0o600); err != nil {
		t.Fatalf("failed writing go.mod: %v", err)
	}

	pkgDir := filepath.Join(repoDir, "pkg", "calc")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("failed creating pkg/calc: %v", err)
	}

	baseCode := `package calc

func Clamp(v int) int {
	return v
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "calc.go"), []byte(baseCode), 0o600); err != nil {
		t.Fatalf("failed writing initial calc.go: %v", err)
	}
	runGit("add", ".")
	runGit("commit", "-m", "initial")

	changedCode := `package calc

func Clamp(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "calc.go"), []byte(changedCode), 0o600); err != nil {
		t.Fatalf("failed writing updated calc.go: %v", err)
	}
	// Weak test: checks v = -1 and v = 5, missing boundary v = 0 (so v < 0 -> v <= 0 survives!).
	testCode := `package calc

import "testing"

func TestClamp(t *testing.T) {
	if got := Clamp(-1); got != 0 {
		t.Fatalf("Clamp(-1) = %d, want 0", got)
	}
	if got := Clamp(5); got != 5 {
		t.Fatalf("Clamp(5) = %d, want 5", got)
	}
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "calc_test.go"), []byte(testCode), 0o600); err != nil {
		t.Fatalf("failed writing calc_test.go: %v", err)
	}

	summaryFile := filepath.Join(repoDir, "summary.md")
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{
		"--repo-root=" + repoDir,
		"--vcs=git",
		"--base=main",
		"--format=github",
		"--summary-file=" + summaryFile,
		"--workers=2",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() unexpected error in soft-start mode: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "::warning file=pkg/calc/calc.go,line=4") {
		t.Errorf("expected GitHub warning annotation on line 4, got: %q", stdout.String())
	}
	summaryBytes, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatalf("failed reading summary.md: %v", err)
	}
	summaryStr := string(summaryBytes)
	if !strings.Contains(summaryStr, "`50.0%` (1/2)") {
		t.Errorf("expected 50.0%% score in summary.md, got:\n%s", summaryStr)
	}
	if !strings.Contains(summaryStr, "**Patch Coverage (Changed Lines)**: **`100.0%`**") {
		t.Errorf("expected automatic 100.0%% patch coverage in summary.md, got:\n%s", summaryStr)
	}
	if strings.Contains(summaryStr, "Overall Go Statement Coverage") {
		t.Errorf("did not expect overall coverage without --coverprofile or --full, got:\n%s", summaryStr)
	}

	// Verify --fail-under triggers error when score < threshold, and passes when score <= failUnder is false (score == threshold).
	err = run(context.Background(), []string{
		"--repo-root=" + repoDir,
		"--vcs=git",
		"--base=main",
		"--format=text",
		"--fail-under=80",
		"--workers=2",
	}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "below required threshold") {
		t.Errorf("expected threshold failure error, got: %v", err)
	}

	err = run(context.Background(), []string{
		"--repo-root=" + repoDir,
		"--vcs=git",
		"--base=main",
		"--format=text",
		"--fail-under=50",
		"--workers=2",
	}, &stdout, &stderr)
	if err != nil {
		t.Errorf("expected score == fail-under (50%%) to pass, got: %v", err)
	}

	// Verify --coverprofile with 0-count block marks changed lines uncovered without failing --fail-under.
	coverPath := filepath.Join(repoDir, "go-cover.output")
	coverContent := `mode: set
example.com/clitest/pkg/calc/calc.go:3.22,6.3 2 0
`
	if err := os.WriteFile(coverPath, []byte(coverContent), 0o600); err != nil {
		t.Fatalf("failed writing coverprofile: %v", err)
	}
	stdout.Reset()
	err = run(context.Background(), []string{
		"--repo-root=" + repoDir,
		"--vcs=git",
		"--base=main",
		"--coverprofile=" + coverPath,
		"--format=text",
		"--color=always",
		"--fail-under=90",
	}, &stdout, &stderr)
	if err != nil {
		t.Errorf("expected 0-tested run with --fail-under=90 to succeed, got: %v", err)
	}
	if !strings.Contains(stdout.String(), "n/a (0 tested)") || !strings.Contains(stdout.String(), "Uncovered Mutants: 2") || !strings.Contains(stdout.String(), "\x1b[") {
		t.Errorf("expected colored 2 uncovered mutants summary from coverprofile pre-filter, got: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "(0 / 2 statements)") {
		t.Errorf("expected overall statement coverage from --coverprofile in output, got: %q", stdout.String())
	}

	// Verify --fail-under-coverage enforces patch line coverage threshold.
	err = run(context.Background(), []string{
		"--repo-root=" + repoDir,
		"--vcs=git",
		"--base=main",
		"--coverprofile=" + coverPath,
		"--format=text",
		"--fail-under-coverage=80",
	}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "patch line coverage 0.0% is below required threshold 80.0%") {
		t.Errorf("expected --fail-under-coverage error, got: %v", err)
	}
}

func TestRunRejectsInvalidFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"--format=xml"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "invalid --format") {
		t.Errorf("expected invalid --format error, got: %v", err)
	}

	err = run(context.Background(), []string{"--color=rainbow"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "invalid --color") {
		t.Errorf("expected invalid --color error, got: %v", err)
	}
}
