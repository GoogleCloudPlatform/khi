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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/decisioncheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type stubDecider struct {
	resp decisioncheck.DecisionResponse
}

func (s stubDecider) Decide(context.Context, decisioncheck.DecisionRequest) (decisioncheck.DecisionResponse, error) {
	return s.resp, nil
}

// TestDecisionsOnlyCLIMode verifies that --decisions-only evaluates decision rules without running Go/TS checks and reuses the SQLite cache.
func TestDecisionsOnlyCLIMode(t *testing.T) {
	repoDir := initDecisionsTestGitRepo(t)
	rulesDir := filepath.Join(t.TempDir(), "rules")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeSimpleDocRule(t, rulesDir, "pkg/**")

	pFail := 0.08
	pPass := 0.95
	stubFactory := func(_ context.Context, _ decisioncheck.DeciderOptions) (decisioncheck.Decider, string, error) {
		return stubDecider{
			resp: decisioncheck.DecisionResponse{
				Answers: map[string]decisioncheck.WireAnswer{
					"unit_0": {Noul: &pFail, Confidence: 0.92},
					"unit_1": {Noul: &pPass, Confidence: 0.95},
				},
			},
		}, "", nil
	}
	getenv := func(key string) string {
		if key == "CODEQUALITY_DECISIONS_CACHE_DIR" {
			return cacheDir
		}
		return ""
	}

	testCases := []struct {
		name       string
		extraArgs  []string
		factory    deciderFactory
		wantCode   int
		wantSubstr string
	}{
		// Exits with code 0 and reports advisory findings when --decisions-fail-on-violations is omitted.
		{
			name:       "advisory mode exits zero and prints finding",
			factory:    stubFactory,
			wantCode:   0,
			wantSubstr: "ADVISORY (1 finding(s))",
		},
		// Exits with code 1 and serves cached responses when --decisions-fail-on-violations is enabled.
		{
			name:       "fail-on-violations exits one using SQLite cache hit",
			extraArgs:  []string{"--decisions-fail-on-violations"},
			factory:    nil,
			wantCode:   1,
			wantSubstr: "cache: 1 hit(s), 0 API call(s)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdoutBuf, stderrBuf bytes.Buffer
			args := append([]string{
				"--repo-root=" + repoDir,
				"--vcs=git",
				"--base=HEAD~1",
				"--decisions-only",
				"--decisions-rules-dir=" + rulesDir,
				"--color=never",
			}, tc.extraArgs...)

			exitCode, err := runCLI(context.Background(), cliEnv{
				args:           args,
				stdout:         &stdoutBuf,
				stderr:         &stderrBuf,
				getenv:         getenv,
				deciderFactory: tc.factory,
			})

			if err != nil {
				t.Fatalf("runCLI() unexpected error: %v (stderr=%s)", err, stderrBuf.String())
			}
			if exitCode != tc.wantCode {
				t.Errorf("runCLI() code = %d, want %d\nstdout:\n%s", exitCode, tc.wantCode, stdoutBuf.String())
			}
			if !strings.Contains(stdoutBuf.String(), tc.wantSubstr) {
				t.Errorf("stdout missing %q:\n%s", tc.wantSubstr, stdoutBuf.String())
			}
			if strings.Contains(stdoutBuf.String(), "Go Quality Report") || strings.Contains(stdoutBuf.String(), "TypeScript Quality Report") {
				t.Errorf("decisions-only mode unexpectedly ran Go/TS report:\n%s", stdoutBuf.String())
			}
		})
	}
}

// TestDecisionsEnvVarConfiguration verifies that environment variables enable decisions, forward the GCP project ID, and promote advisory findings to blocking failures.
func TestDecisionsEnvVarConfiguration(t *testing.T) {
	repoDir := initDecisionsTestGitRepo(t)
	rulesDir := filepath.Join(t.TempDir(), "rules")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeSimpleDocRule(t, rulesDir, "pkg/**")

	pFail := 0.08
	pPass := 0.95
	var gotOpts decisioncheck.DeciderOptions
	factory := func(_ context.Context, opts decisioncheck.DeciderOptions) (decisioncheck.Decider, string, error) {
		gotOpts = opts
		return stubDecider{
			resp: decisioncheck.DecisionResponse{
				Answers: map[string]decisioncheck.WireAnswer{
					"unit_0": {Noul: &pFail, Confidence: 0.92},
					"unit_1": {Noul: &pPass, Confidence: 0.95},
				},
			},
		}, "", nil
	}
	getenv := mapGetenv(map[string]string{
		"CODEQUALITY_DECISIONS":                    "1",
		"CODEQUALITY_DECISIONS_PROJECT":            " env-dec-proj ",
		"CODEQUALITY_DECISIONS_FAIL_ON_VIOLATIONS": "true",
		"CODEQUALITY_DECISIONS_CACHE_DIR":          cacheDir,
	})
	var stdoutBuf, stderrBuf bytes.Buffer

	exitCode, err := runCLI(context.Background(), cliEnv{
		args: []string{
			"--repo-root=" + repoDir,
			"--vcs=git",
			"--base=HEAD~1",
			"--lang=ts",
			"--decisions-rules-dir=" + rulesDir,
			"--color=never",
		},
		stdout:         &stdoutBuf,
		stderr:         &stderrBuf,
		getenv:         getenv,
		deciderFactory: factory,
	})

	if err != nil {
		t.Fatalf("runCLI() unexpected error: %v (stderr=%s)", err, stderrBuf.String())
	}
	if exitCode != 1 {
		t.Errorf("runCLI() code = %d, want 1\nstdout:\n%s", exitCode, stdoutBuf.String())
	}
	if gotOpts.ProjectID != "env-dec-proj" {
		t.Errorf("DeciderOptions.ProjectID = %q, want %q", gotOpts.ProjectID, "env-dec-proj")
	}
	if !strings.Contains(stdoutBuf.String(), "FAIL (1 blocking, 0 advisory)") {
		t.Errorf("stdout missing blocking failure verdict:\n%s", stdoutBuf.String())
	}
}

func TestParseBoolEnv(t *testing.T) {
	testCases := []struct {
		name   string
		getenv func(string) string
		key    string
		want   bool
	}{
		// Returns false when the getenv lookup callback is nil.
		{
			name:   "nil getenv returns false",
			getenv: nil,
			key:    "CODEQUALITY_DECISIONS",
			want:   false,
		},
		// Trims whitespace and treats case-insensitive true as enabled.
		{
			name:   "trimmed uppercase true returns true",
			getenv: mapGetenv(map[string]string{"CODEQUALITY_DECISIONS": " TRUE "}),
			key:    "CODEQUALITY_DECISIONS",
			want:   true,
		},
		// Treats numeric 1 as enabled.
		{
			name:   "numeric one returns true",
			getenv: mapGetenv(map[string]string{"CODEQUALITY_DECISIONS": "1"}),
			key:    "CODEQUALITY_DECISIONS",
			want:   true,
		},
		// Returns false when the environment variable is explicitly false.
		{
			name:   "explicit false returns false",
			getenv: mapGetenv(map[string]string{"CODEQUALITY_DECISIONS": "false"}),
			key:    "CODEQUALITY_DECISIONS",
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseBoolEnv(tc.getenv, tc.key)

			if got != tc.want {
				t.Errorf("parseBoolEnv(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestParseDecisionsProjectEnv(t *testing.T) {
	testCases := []struct {
		name   string
		getenv func(string) string
		want   string
	}{
		// Returns an empty string when the getenv lookup callback is nil.
		{
			name:   "nil getenv returns empty string",
			getenv: nil,
			want:   "",
		},
		// Trims leading and trailing whitespace from CODEQUALITY_DECISIONS_PROJECT.
		{
			name:   "trims configured project ID",
			getenv: mapGetenv(map[string]string{"CODEQUALITY_DECISIONS_PROJECT": "  khi-dev-project  "}),
			want:   "khi-dev-project",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDecisionsProjectEnv(tc.getenv)

			if got != tc.want {
				t.Errorf("parseDecisionsProjectEnv() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveTrustedRoot(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	testCases := []struct {
		name        string
		repoRoot    string
		trustedFlag string
		want        string
	}{
		// Returns repoRoot unchanged when the --trusted-root flag is blank.
		{
			name:        "blank flag defaults to repoRoot",
			repoRoot:    "/repo/root",
			trustedFlag: "   ",
			want:        "/repo/root",
		},
		// Resolves a relative --trusted-root path to an absolute directory path.
		{
			name:        "relative flag resolves to absolute path",
			repoRoot:    "/repo/root",
			trustedFlag: "trusted-sub",
			want:        filepath.Join(workDir, "trusted-sub"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveTrustedRoot(tc.repoRoot, tc.trustedFlag)

			if err != nil {
				t.Fatalf("resolveTrustedRoot() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("resolveTrustedRoot(%q, %q) = %q, want %q", tc.repoRoot, tc.trustedFlag, got, tc.want)
			}
		})
	}
}

func TestParseCSVSet(t *testing.T) {
	testCases := []struct {
		name string
		raw  string
		want map[string]struct{}
	}{
		// Returns nil when the input string contains only whitespace.
		{
			name: "blank input returns nil",
			raw:  "   ",
			want: nil,
		},
		// Splits comma-separated tokens, trims whitespace, and ignores empty elements.
		{
			name: "splits and trims non-empty rule IDs",
			raw:  " go_code_quality , , go_test_quality ",
			want: map[string]struct{}{
				"go_code_quality": {},
				"go_test_quality": {},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseCSVSet(tc.raw)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseCSVSet(%q) mismatch (-want +got):\n%s", tc.raw, diff)
			}
		})
	}
}

func TestMergeGoChangedLines(t *testing.T) {
	testCases := []struct {
		name  string
		prod  map[string]map[int]struct{}
		tests map[string]map[int]struct{}
		want  map[string]map[int]struct{}
	}{
		// Returns nil when both production and test changed-line maps are empty.
		{
			name:  "empty inputs return nil",
			prod:  nil,
			tests: map[string]map[int]struct{}{},
			want:  nil,
		},
		// Preserves production entries when only production Go files changed.
		{
			name:  "production only preserves prod map",
			prod:  map[string]map[int]struct{}{"pkg/a/a.go": {10: {}}},
			tests: nil,
			want:  map[string]map[int]struct{}{"pkg/a/a.go": {10: {}}},
		},
		// Preserves test entries when only Go test files changed.
		{
			name:  "tests only preserves test map",
			prod:  nil,
			tests: map[string]map[int]struct{}{"pkg/a/a_test.go": {20: {}}},
			want:  map[string]map[int]struct{}{"pkg/a/a_test.go": {20: {}}},
		},
		// Combines production and test changed-line maps into one merged map.
		{
			name:  "merges both production and test entries",
			prod:  map[string]map[int]struct{}{"pkg/a/a.go": {1: {}}},
			tests: map[string]map[int]struct{}{"pkg/a/a_test.go": {2: {}}},
			want: map[string]map[int]struct{}{
				"pkg/a/a.go":      {1: {}},
				"pkg/a/a_test.go": {2: {}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeGoChangedLines(tc.prod, tc.tests)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mergeGoChangedLines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectPathsUnifiedDiffGuards(t *testing.T) {
	testCases := []struct {
		name     string
		diffRes  gotestutil.DiffResult
		relPaths []string
	}{
		// Returns an empty diff without executing VCS when relPaths is empty.
		{
			name:     "empty relPaths returns empty diff",
			diffRes:  gotestutil.DiffResult{VCS: "git", ResolvedBaseRev: "HEAD~1"},
			relPaths: nil,
		},
		// Returns an empty diff without executing VCS when ResolvedBaseRev is empty.
		{
			name:     "empty base revision returns empty diff",
			diffRes:  gotestutil.DiffResult{VCS: "git", ResolvedBaseRev: ""},
			relPaths: []string{"pkg/foo/extra.go"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := collectPathsUnifiedDiff(context.Background(), t.TempDir(), tc.diffRes, tc.relPaths)

			if err != nil {
				t.Fatalf("collectPathsUnifiedDiff() unexpected error: %v", err)
			}
			if got != "" {
				t.Errorf("collectPathsUnifiedDiff() = %q, want empty", got)
			}
		})
	}
}

// TestCollectPathsUnifiedDiffGit verifies that collectPathsUnifiedDiff invokes git diff scoped to the requested paths.
func TestCollectPathsUnifiedDiffGit(t *testing.T) {
	repoDir := initDecisionsTestGitRepo(t)
	diffRes := gotestutil.DiffResult{VCS: "git", ResolvedBaseRev: "HEAD~1"}

	got, err := collectPathsUnifiedDiff(context.Background(), repoDir, diffRes, []string{"pkg/foo/extra.go"})

	if err != nil {
		t.Fatalf("collectPathsUnifiedDiff(git) unexpected error: %v", err)
	}
	if !strings.Contains(got, "diff --git a/pkg/foo/extra.go") || !strings.Contains(got, "AddedUndocumented") {
		t.Errorf("collectPathsUnifiedDiff(git) unexpected diff output:\n%s", got)
	}
}

// TestCollectPathsUnifiedDiffJJ verifies that collectPathsUnifiedDiff passes jj diff --git --from arguments when VCS is jj.
func TestCollectPathsUnifiedDiffJJ(t *testing.T) {
	fakeBin := t.TempDir()
	fakeJJ := filepath.Join(fakeBin, "jj")
	script := `#!/bin/sh
echo "FAKE_JJ_DIFF:$*"
`
	if err := os.WriteFile(fakeJJ, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(fake jj) failed: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	diffRes := gotestutil.DiffResult{VCS: "jj", ResolvedBaseRev: "rev123"}

	got, err := collectPathsUnifiedDiff(context.Background(), fakeBin, diffRes, []string{"pkg/a/a.go", "pkg/a/b.go"})

	if err != nil {
		t.Fatalf("collectPathsUnifiedDiff(jj) unexpected error: %v", err)
	}
	want := "FAKE_JJ_DIFF:diff --git --from rev123 -- pkg/a/a.go pkg/a/b.go\n"
	if got != want {
		t.Errorf("collectPathsUnifiedDiff(jj) = %q, want %q", got, want)
	}
}

// TestDecisionsSessionCloseNilReceiver verifies that Close returns nil safely on a nil receiver.
func TestDecisionsSessionCloseNilReceiver(t *testing.T) {
	var session *decisionsSession

	err := session.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

// TestDecisionsSessionCloseClosesAndClearsCache verifies that Close closes the opened SQLite cache and clears s.cache.
func TestDecisionsSessionCloseClosesAndClearsCache(t *testing.T) {
	ctx := context.Background()
	cacheDir := t.TempDir()
	session := newDecisionsSession(ctx, cliFlags{decisions: true}, nil, mapGetenv(map[string]string{
		"CODEQUALITY_DECISIONS_CACHE_DIR": cacheDir,
	}))
	openedCache, err := session.ensureCache(ctx)
	if err != nil {
		t.Fatalf("ensureCache() setup failed: %v", err)
	}

	closeErr := session.Close()

	if closeErr != nil {
		t.Fatalf("Close() unexpected error: %v", closeErr)
	}
	if session.cache != nil {
		t.Errorf("session.cache = %v, want nil after Close()", session.cache)
	}
	if _, _, lookupErr := openedCache.Lookup(ctx, decisioncheck.CacheKey{}); lookupErr == nil {
		t.Errorf("openedCache.Lookup() error = nil after session.Close(), want closed DB error")
	}
}

// TestDecisionsSessionEnsureCacheReusesInstance verifies that repeated ensureCache calls return the cached DecisionCache pointer.
func TestDecisionsSessionEnsureCacheReusesInstance(t *testing.T) {
	ctx := context.Background()
	cacheDir := t.TempDir()
	session := newDecisionsSession(ctx, cliFlags{decisions: true}, nil, mapGetenv(map[string]string{
		"CODEQUALITY_DECISIONS_CACHE_DIR": cacheDir,
	}))
	t.Cleanup(func() { _ = session.Close() })

	first, errFirst := session.ensureCache(ctx)
	second, errSecond := session.ensureCache(ctx)

	if errFirst != nil || errSecond != nil {
		t.Fatalf("ensureCache() errors = (%v, %v), want nil", errFirst, errSecond)
	}
	if first == nil || second != first || session.cache != first {
		t.Errorf("ensureCache() returned (%p, %p) with session.cache=%p, want identical non-nil pointer", first, second, session.cache)
	}
}

// TestDecisionsSessionEvaluateSkipsEmptyDiffWhenNotFull verifies that evaluate returns nil before loading rules when not in full mode and no files changed.
func TestDecisionsSessionEvaluateSkipsEmptyDiffWhenNotFull(t *testing.T) {
	ctx := context.Background()
	session := newDecisionsSession(ctx, cliFlags{
		decisions:         true,
		decisionsRulesDir: filepath.Join(t.TempDir(), "missing-rules"),
	}, nil, nil)
	t.Cleanup(func() { _ = session.Close() })

	rep, err := session.evaluate(ctx, t.TempDir(), gotestutil.DiffResult{})

	if err != nil {
		t.Fatalf("evaluate() unexpected error: %v", err)
	}
	if rep != nil {
		t.Errorf("evaluate() = %+v, want nil for empty diff", rep)
	}
}

// TestDecisionsSessionEvaluateFullModeWithEmptyDiff verifies that evaluate inspects repository targets in --full mode even when ChangedFiles and DeletedFiles are empty.
func TestDecisionsSessionEvaluateFullModeWithEmptyDiff(t *testing.T) {
	ctx := context.Background()
	repoDir := initDecisionsTestGitRepo(t)
	rulesDir := filepath.Join(t.TempDir(), "rules")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeSimpleDocRule(t, rulesDir, "pkg/**")
	pPass := 0.95
	session := newDecisionsSession(ctx, cliFlags{
		decisions:         true,
		full:              true,
		decisionsRulesDir: rulesDir,
	}, func(context.Context, decisioncheck.DeciderOptions) (decisioncheck.Decider, string, error) {
		return stubDecider{
			resp: decisioncheck.DecisionResponse{
				Answers: map[string]decisioncheck.WireAnswer{
					"unit_0": {Noul: &pPass, Confidence: 0.95},
					"unit_1": {Noul: &pPass, Confidence: 0.95},
				},
			},
		}, "", nil
	}, mapGetenv(map[string]string{"CODEQUALITY_DECISIONS_CACHE_DIR": cacheDir}))
	t.Cleanup(func() { _ = session.Close() })

	rep, err := session.evaluate(ctx, repoDir, gotestutil.DiffResult{BaseLabel: "main"})

	if err != nil {
		t.Fatalf("evaluate(full) unexpected error: %v", err)
	}
	want := &rule.Report{
		Language:  "Decisions",
		BaseLabel: "main",
		Decisions: rule.DecisionSummary{
			Enabled:            true,
			RulesEvaluated:     1,
			TargetsEvaluated:   1,
			QuestionsEvaluated: 2,
			APICalls:           1,
		},
	}
	if diff := cmp.Diff(want, rep); diff != "" {
		t.Errorf("evaluate(full) report mismatch (-want +got):\n%s", diff)
	}
}

// TestDecisionsSessionEvaluateDefaultRulesDiffAndNilFactory verifies that evaluate loads rules from trustedRoot by default, invokes ReadDiff, and returns a skipped report when deciderFactory is nil.
func TestDecisionsSessionEvaluateDefaultRulesDiffAndNilFactory(t *testing.T) {
	ctx := context.Background()
	repoDir := initDecisionsTestGitRepo(t)
	trustedRoot := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	defaultRulesDir := filepath.Join(trustedRoot, "scripts", "codequality", "rules")
	writeFiles(t, defaultRulesDir, map[string]string{
		"diff_rule.yaml": `
id: diff_check
name: Diff Check
enforcement: advisory
target:
  kind: changed_file
  include: ["pkg/**"]
context:
  include_diff: true
questions:
  - id: ok
    type: noul
    instructions: "Is the diff clean?"
    criteria:
      false_description: "Diff issue."
    fix_hint: "Fix the diff issue."
    pass:
      expect: true
      min_probability: 0.50
`,
	})
	diffRes, err := gotestutil.CollectChangedLines(ctx, repoDir, gotestutil.DiffOptions{VCS: "git", Base: "HEAD~1"})
	if err != nil {
		t.Fatalf("CollectChangedLines() failed: %v", err)
	}
	session := newDecisionsSession(ctx, cliFlags{
		decisions:   true,
		trustedRoot: trustedRoot,
	}, nil, mapGetenv(map[string]string{"CODEQUALITY_DECISIONS_CACHE_DIR": cacheDir}))
	t.Cleanup(func() { _ = session.Close() })

	rep, err := session.evaluate(ctx, repoDir, diffRes)

	if err != nil {
		t.Fatalf("evaluate() unexpected error: %v", err)
	}
	want := &rule.Report{
		Language:  "Decisions",
		BaseLabel: "HEAD~1",
		Decisions: rule.DecisionSummary{
			Enabled:    true,
			Skipped:    true,
			SkipReason: "decision client not configured",
		},
	}
	if diff := cmp.Diff(want, rep); diff != "" {
		t.Errorf("evaluate() skipped report mismatch (-want +got):\n%s", diff)
	}
}

// TestDecisionsSessionEvaluateReturnsNilWhenNoTargetsMatch verifies that evaluate returns nil when enabled rules match none of the changed files.
func TestDecisionsSessionEvaluateReturnsNilWhenNoTargetsMatch(t *testing.T) {
	ctx := context.Background()
	repoDir := initDecisionsTestGitRepo(t)
	rulesDir := filepath.Join(t.TempDir(), "rules")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeSimpleDocRule(t, rulesDir, "docs/**")
	session := newDecisionsSession(ctx, cliFlags{
		decisions:         true,
		decisionsRulesDir: rulesDir,
	}, nil, mapGetenv(map[string]string{"CODEQUALITY_DECISIONS_CACHE_DIR": cacheDir}))
	t.Cleanup(func() { _ = session.Close() })

	rep, err := session.evaluate(ctx, repoDir, gotestutil.DiffResult{
		VCS:          "git",
		BaseLabel:    "HEAD~1",
		ChangedFiles: []string{"pkg/foo/extra.go"},
	})

	if err != nil {
		t.Fatalf("evaluate() unexpected error: %v", err)
	}
	if rep != nil {
		t.Errorf("evaluate() = %+v, want nil when zero targets match", rep)
	}
}

// TestDecisionsSessionEvaluatePropagatesFactoryError verifies that evaluate returns the error produced by deciderFactory on a cache miss.
func TestDecisionsSessionEvaluatePropagatesFactoryError(t *testing.T) {
	ctx := context.Background()
	repoDir := initDecisionsTestGitRepo(t)
	rulesDir := filepath.Join(t.TempDir(), "rules")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeSimpleDocRule(t, rulesDir, "pkg/**")
	session := newDecisionsSession(ctx, cliFlags{
		decisions:         true,
		decisionsRulesDir: rulesDir,
	}, func(context.Context, decisioncheck.DeciderOptions) (decisioncheck.Decider, string, error) {
		return nil, "", errors.New("factory boom")
	}, mapGetenv(map[string]string{"CODEQUALITY_DECISIONS_CACHE_DIR": cacheDir}))
	t.Cleanup(func() { _ = session.Close() })

	rep, err := session.evaluate(ctx, repoDir, gotestutil.DiffResult{
		VCS:          "git",
		BaseLabel:    "HEAD~1",
		ChangedFiles: []string{"pkg/foo/extra.go"},
	})

	if err == nil || !strings.Contains(err.Error(), "factory boom") {
		t.Fatalf("evaluate() error = %v, want substring %q", err, "factory boom")
	}
	if rep != nil {
		t.Errorf("evaluate() report = %+v, want nil on error", rep)
	}
}

func initDecisionsTestGitRepo(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init", "-b", "main")
	writeFiles(t, repoDir, map[string]string{
		"go.mod":            "module example.com/decmod\n\ngo 1.24\n",
		"pkg/foo/README.md": "# Foo\n\nInitial overview.\n",
		"pkg/foo/store.go":  "package foo\n\ntype Store struct{}\n",
	})
	runGitCmd(t, repoDir, "add", ".")
	runGitCmd(t, repoDir, "commit", "-m", "base")

	writeFiles(t, repoDir, map[string]string{
		"pkg/foo/extra.go": "package foo\n\ntype AddedUndocumented struct{}\n",
	})
	runGitCmd(t, repoDir, "add", ".")
	runGitCmd(t, repoDir, "commit", "-m", "add undocumented file")
	return repoDir
}

func writeSimpleDocRule(t *testing.T, rulesDir string, includeGlob string) {
	t.Helper()
	writeFiles(t, rulesDir, map[string]string{
		"doc.yaml": `
id: pkg_doc_alignment
name: Package Doc Alignment
enforcement: advisory
target:
  kind: go_logical_package
  include: ["` + includeGlob + `"]
  require_files: ["README.md"]
questions:
  - id: unit
    for_each: go_substantive_units
    type: noul
    instructions: "Is {{.Name}} documented?"
    criteria:
      false_description: "{{.Name}} is missing from README.md."
    fix_hint: "Document {{.Name}} in README.md."
    pass:
      expect: true
      min_probability: 0.50
`,
	})
}

func mapGetenv(env map[string]string) func(string) string {
	return func(key string) string {
		return env[key]
	}
}
