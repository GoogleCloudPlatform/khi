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

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
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
		wantStderr string
	}{
		{
			name:       "fast mode passes clean git diff",
			args:       []string{"check", "--repo-root=" + repoDir, "--vcs=git", "--base=HEAD~1", "--speed=fast", "--filter=must", "--color=never"},
			wantCode:   0,
			wantStdout: "PASS (0 violations)",
			wantStderr: "[1/2]",
		},
		{
			name:       "full dynamic mode passes Go and writes github summary",
			args:       []string{"check", "--repo-root=" + repoDir, "--vcs=git", "--base=HEAD~1", "--skip-sources=decisions,similarity", "--format=github", "--summary-file=" + summaryPath},
			wantCode:   0,
			wantStdout: "",
			wantStderr: "[1/4]",
		},
		{
			name:       "lang ts on Go-only diff passes cleanly",
			args:       []string{"check", "--repo-root=" + repoDir, "--lang=ts", "--vcs=git", "--base=HEAD~1", "--skip-sources=decisions,similarity", "--color=never"},
			wantCode:   0,
			wantStdout: "TypeScript Quality Report",
		},
		{
			name:       "lang ts with fast skips dynamic TS stage cleanly",
			args:       []string{"check", "--repo-root=" + repoDir, "--lang=ts", "--speed=fast", "--filter=must", "--vcs=git", "--base=HEAD~1", "--summary-file=" + summaryPath},
			wantCode:   0,
			wantStdout: "no Go or TypeScript files changed",
		},
		{
			name:     "invalid lang flag returns code 2",
			args:     []string{"check", "--lang=python"},
			wantCode: 2,
			wantErr:  true,
		},
		{
			name:     "invalid format flag returns code 2",
			args:     []string{"check", "--format=xml"},
			wantCode: 2,
			wantErr:  true,
		},
		{
			name:     "unknown flag returns code 2",
			args:     []string{"check", "--unknown-flag"},
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
				wantStderr: tc.wantStderr,
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
	wantErrSub string
	stdout     string
	stderr     string
	wantStdout string
	wantStderr string
}

func assertCLIOutcome(t *testing.T, c cliOutcomeCheck) {
	t.Helper()
	if c.code != c.wantCode {
		t.Errorf("code = %d, want %d (err=%v, stderr=%s, stdout=%s)", c.code, c.wantCode, c.err, c.stderr, c.stdout)
	}
	if (c.err != nil) != c.wantErr {
		t.Errorf("err = %v, wantErr %v", c.err, c.wantErr)
	}
	if c.wantErrSub != "" && (c.err == nil || !strings.Contains(c.err.Error(), c.wantErrSub)) {
		t.Errorf("err = %v, want substring %q", c.err, c.wantErrSub)
	}
	if c.wantStdout != "" && !strings.Contains(c.stdout, c.wantStdout) {
		t.Errorf("stdout missing %q:\n%s", c.wantStdout, c.stdout)
	}
	if c.wantStderr != "" && !strings.Contains(c.stderr, c.wantStderr) {
		t.Errorf("stderr missing %q:\n%s", c.wantStderr, c.stderr)
	}
}

func TestRunCLIConventionsMode(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"go.mod": "module example.com/convmod\n\ngo 1.24\n",
		"pkg/clean/clean.go": `package clean

// PublicType is documented.
type PublicType struct {
	// Value holds the value.
	Value int
}
`,
		"pkg/bad/bad.go": `package bad

type BadType struct {
	Missing int
}
`,
		"pkg/badshort/badshort_test.go": `package badshort

import "testing"

func TestSkipShort(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow")
	}
}
`,
		"pkg/broken/broken.go": "package broken\nfunc (",
	})

	testCases := []struct {
		name       string
		args       []string
		wantCode   int
		wantErr    bool
		wantErrSub string
		wantStdout string
	}{
		// Passes with exit code 0 when the package satisfies all conventions.
		{
			name:     "clean package passes with exit code 0",
			args:     []string{"conventions", "--repo-root=" + dir, "--pkgs=./pkg/clean"},
			wantCode: 0,
		},
		// Exits with code 1 and prints the exported-member-doc violation line.
		{
			name:       "violation prints formatted line and exits 1",
			args:       []string{"conventions", "--repo-root=" + dir, "--pkgs=./pkg/bad"},
			wantCode:   1,
			wantStdout: "pkg/bad/bad.go:4:2: exported struct field BadType.Missing must have a GoDoc comment (exported-member-doc)",
		},
		// Exits with code 1 and prints the no-short-skip violation line for a test file skipping under testing.Short().
		{
			name:       "no-short-skip violation prints formatted line and exits 1",
			args:       []string{"conventions", "--repo-root=" + dir, "--pkgs=./pkg/badshort"},
			wantCode:   1,
			wantStdout: "pkg/badshort/badshort_test.go:7:3: do not call Skip/Skipf/SkipNow inside testing.Short(); scale down iteration counts instead of skipping tests (no-short-skip)",
		},
		// Returns exit code 2 and propagates a parse error for syntactically invalid Go files.
		{
			name:       "parse error returns exit code 2",
			args:       []string{"conventions", "--repo-root=" + dir, "--pkgs=./pkg/broken"},
			wantCode:   2,
			wantErr:    true,
			wantErrSub: "failed to parse pkg/broken/broken.go",
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
				wantErrSub: tc.wantErrSub,
				stdout:     stdout.String(),
				stderr:     stderr.String(),
				wantStdout: tc.wantStdout,
			})
		})
	}
}

func TestRunCLIUnusedExportsMode(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"go.mod": "module example.com/unusedmod\n\ngo 1.24\n",
		"pkg/unused/unused.go": `package unused

const Dead = 2
`,
	})

	var stdout, stderr bytes.Buffer
	code, err := runCLI(context.Background(), cliEnv{
		args:   []string{"unused-exports", "--repo-root=" + dir, "--pkgs=./pkg/unused"},
		stdout: &stdout,
		stderr: &stderr,
		getenv: func(string) string { return "" },
	})
	assertCLIOutcome(t, cliOutcomeCheck{
		code:       code,
		wantCode:   1,
		err:        err,
		stdout:     stdout.String(),
		stderr:     stderr.String(),
		wantStdout: "pkg/unused/unused.go:3:7: exported const Dead has no references in the repository (unused-exported-decl)",
	})
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
			extraArgs: []string{"--lang=go", "--skip-sources=decisions,similarity", "--fail-on-violations=true", "--summary-file=" + summaryPath},
			wantCode:  1,
		},
		{
			name:      "fast advisory mode exits 0 when fail-on-violations is false",
			extraArgs: []string{"--speed=fast", "--filter=must", "--fail-on-violations=false"},
			wantCode:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := append([]string{"check", "--repo-root=" + repoDir, "--vcs=git", "--base=HEAD~1"}, tc.extraArgs...)
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

// TestRunCLIDecisionsEvaluationError verifies that runCLI returns exit code 2 and propagates errors when decision rule loading fails.
func TestRunCLIDecisionsEvaluationError(t *testing.T) {
	repoDir := initTempGitRepo(t)
	var stdout, stderr bytes.Buffer

	code, err := runCLI(context.Background(), cliEnv{
		args: []string{
			"check",
			"--repo-root=" + repoDir,
			"--vcs=git",
			"--base=HEAD~1",
			"--speed=fast",
			"--filter=advisory",
			"--decisions",
			"--decisions-rules-dir=" + filepath.Join(repoDir, "nonexistent-rules"),
		},
		stdout: &stdout,
		stderr: &stderr,
		getenv: func(string) string { return "" },
	})

	if code != 2 {
		t.Errorf("runCLI() code = %d, want 2", code)
	}
	if err == nil {
		t.Errorf("runCLI() error = nil, want non-nil error for nonexistent rules directory")
	}
}

func TestShouldRunGo(t *testing.T) {
	testCases := []struct {
		name    string
		flags   cliFlags
		speed   rule.Speed
		filter  rule.Filter
		diffRes gotestutil.DiffResult
		want    bool
	}{
		// Returns false in fast advisory mode even when --lang=go, --full, --similarity, and changed Go lines are present.
		{
			name:   "fast advisory disables Go pipeline",
			flags:  cliFlags{lang: "go", full: true, similarity: true},
			speed:  rule.SpeedFast,
			filter: rule.FilterAdvisory,
			diffRes: gotestutil.DiffResult{
				ChangedLines: map[string]map[int]struct{}{"pkg/a/a.go": {1: {}}},
			},
			want: false,
		},
		// Returns false when --lang=all has no full/coverprofile flags and no Go files changed.
		{
			name:    "lang all with empty diff and no flags skips Go pipeline",
			flags:   cliFlags{lang: "all"},
			speed:   rule.SpeedAll,
			filter:  rule.FilterAll,
			diffRes: gotestutil.DiffResult{},
			want:    false,
		},
		// Returns true when --lang=go and --speed=fast even on an empty diff.
		{
			name:    "lang go with fast speed runs Go pipeline on empty diff",
			flags:   cliFlags{lang: "go"},
			speed:   rule.SpeedFast,
			filter:  rule.FilterMust,
			diffRes: gotestutil.DiffResult{},
			want:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRunGo(tc.flags, tc.speed, tc.filter, tc.diffRes)

			if got != tc.want {
				t.Errorf("shouldRunGo() = %v, want %v", got, tc.want)
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

// TestParseFlagsDefaultsAndErrors verifies subcommand defaults and FlagSet error prefixes in parseFlags.
func TestParseFlagsDefaultsAndErrors(t *testing.T) {
	var errBuf bytes.Buffer
	if _, err := parseFlags(nil, &errBuf, func(string) string { return "" }); err == nil || !strings.Contains(err.Error(), "missing subcommand") {
		t.Fatalf("parseFlags(nil) err = %v, want missing subcommand", err)
	}

	for _, sub := range []string{"conventions", "unused-exports", "qa", "check"} {
		errBuf.Reset()
		if _, err := parseFlags([]string{sub, "--unknown-test-flag"}, &errBuf, func(string) string { return "" }); err == nil {
			t.Errorf("parseFlags(%s, --unknown-test-flag) expected error", sub)
		} else if !strings.Contains(errBuf.String(), "codequality "+sub) {
			t.Errorf("errOut = %q, want prefix %q", errBuf.String(), "codequality "+sub)
		}
	}

	qaFlags, err := parseFlags([]string{"qa"}, &errBuf, func(string) string { return "" })
	if err != nil {
		t.Fatalf("parseFlags(qa) error: %v", err)
	}
	if qaFlags.speed != "fast" || qaFlags.decisionsMaxTargets != 0 || qaFlags.maxMutants != 0 || qaFlags.decisions || qaFlags.similarity {
		t.Errorf("qa defaults = (speed=%q, decisionsMaxTargets=%d, maxMutants=%d, decisions=%v, similarity=%v), want (fast, 0, 0, false, false)",
			qaFlags.speed, qaFlags.decisionsMaxTargets, qaFlags.maxMutants, qaFlags.decisions, qaFlags.similarity)
	}

	checkFlags, err := parseFlags([]string{"check"}, &errBuf, func(string) string { return "" })
	if err != nil {
		t.Fatalf("parseFlags(check) error: %v", err)
	}
	if checkFlags.speed != "all" || checkFlags.decisionsMaxTargets != 0 || checkFlags.maxMutants != 0 || checkFlags.decisions || checkFlags.similarity {
		t.Errorf("check defaults = (speed=%q, decisionsMaxTargets=%d, maxMutants=%d, decisions=%v, similarity=%v), want (all, 0, 0, false, false)",
			checkFlags.speed, checkFlags.decisionsMaxTargets, checkFlags.maxMutants, checkFlags.decisions, checkFlags.similarity)
	}

	envEnabled, err := parseFlags([]string{"qa"}, &errBuf, mapGetenv(map[string]string{
		"CODEQUALITY_DECISIONS":  "true",
		"CODEQUALITY_SIMILARITY": "1",
	}))
	if err != nil || !envEnabled.decisions || !envEnabled.similarity {
		t.Errorf("parseFlags(qa env opt-in) = (decisions=%v, similarity=%v, err=%v), want (true, true, nil)",
			envEnabled.decisions, envEnabled.similarity, err)
	}

	envOverridden, err := parseFlags([]string{"qa", "--decisions=false", "--similarity=false"}, &errBuf, mapGetenv(map[string]string{
		"CODEQUALITY_DECISIONS":  "true",
		"CODEQUALITY_SIMILARITY": "true",
	}))
	if err != nil || envOverridden.decisions || envOverridden.similarity {
		t.Errorf("parseFlags(qa CLI override) = (decisions=%v, similarity=%v, err=%v), want (false, false, nil)",
			envOverridden.decisions, envOverridden.similarity, err)
	}
}

// TestPipelineHelpersAndStageSelection verifies shouldRunGo, shouldRunTS, and changedGoFiles stage selection.
func TestPipelineHelpersAndStageSelection(t *testing.T) {
	t.Run("changedGoFiles", func(t *testing.T) {
		got := changedGoFiles(gotestutil.DiffResult{
			ChangedLines:     map[string]map[int]struct{}{"pkg/a/a.go": {1: {}}},
			ChangedTestLines: map[string]map[int]struct{}{"pkg/a/a_test.go": {2: {}}},
		})
		if len(got) != 2 || got[0] != "pkg/a/a.go" || got[1] != "pkg/a/a_test.go" {
			t.Errorf("changedGoFiles() = %v, want [pkg/a/a.go pkg/a/a_test.go]", got)
		}
	})

	t.Run("shouldRunGo", func(t *testing.T) {
		goDiff := gotestutil.DiffResult{ChangedLines: map[string]map[int]struct{}{"pkg/a/a.go": {1: {}}}}
		if !shouldRunGo(cliFlags{lang: "all", coverprofile: "cov.out"}, rule.SpeedSlow, rule.FilterMust, gotestutil.DiffResult{}) {
			t.Errorf("shouldRunGo(coverprofile) = false, want true")
		}
		if shouldRunGo(cliFlags{lang: "all"}, rule.SpeedSlow, rule.FilterAdvisory, goDiff) {
			t.Errorf("shouldRunGo(slow advisory without similarity) = true, want false")
		}
		if !shouldRunGo(cliFlags{lang: "all", similarity: true}, rule.SpeedSlow, rule.FilterAdvisory, goDiff) {
			t.Errorf("shouldRunGo(slow advisory with similarity) = false, want true")
		}
		if shouldRunGo(cliFlags{lang: "all", similarity: true}, rule.SpeedSlow, rule.FilterAdvisory, gotestutil.DiffResult{}) {
			t.Errorf("shouldRunGo(slow advisory empty) = true, want false")
		}
	})

	t.Run("shouldRunTS", func(t *testing.T) {
		cases := []struct {
			name    string
			flags   cliFlags
			filter  rule.Filter
			diffRes gotestutil.DiffResult
			want    bool
		}{
			{name: "prod lines", flags: cliFlags{lang: "all"}, filter: rule.FilterMust, diffRes: gotestutil.DiffResult{TSChangedLines: map[string]map[int]struct{}{"a.ts": {1: {}}}}, want: true},
			{name: "test lines", flags: cliFlags{lang: "all"}, filter: rule.FilterMust, diffRes: gotestutil.DiffResult{TSChangedTestLines: map[string]map[int]struct{}{"a.spec.ts": {1: {}}}}, want: true},
			{name: "tsCoverage", flags: cliFlags{lang: "all", tsCoverage: "cov.json"}, filter: rule.FilterMust, want: true},
			{name: "full", flags: cliFlags{lang: "all", full: true}, filter: rule.FilterMust, want: true},
			{name: "empty must", flags: cliFlags{lang: "all"}, filter: rule.FilterMust, want: false},
			{name: "advisory prod without similarity", flags: cliFlags{lang: "all"}, filter: rule.FilterAdvisory, diffRes: gotestutil.DiffResult{TSChangedLines: map[string]map[int]struct{}{"a.ts": {1: {}}}}, want: false},
			{name: "advisory prod with similarity", flags: cliFlags{lang: "all", similarity: true}, filter: rule.FilterAdvisory, diffRes: gotestutil.DiffResult{TSChangedLines: map[string]map[int]struct{}{"a.ts": {1: {}}}}, want: true},
			{name: "advisory test only", flags: cliFlags{lang: "all", similarity: true}, filter: rule.FilterAdvisory, diffRes: gotestutil.DiffResult{TSChangedTestLines: map[string]map[int]struct{}{"a.spec.ts": {1: {}}}}, want: false},
		}
		for _, tc := range cases {
			if got := shouldRunTS(tc.flags, rule.SpeedSlow, tc.filter, tc.diffRes); got != tc.want {
				t.Errorf("shouldRunTS(%s) = %v, want %v", tc.name, got, tc.want)
			}
		}
	})
}
