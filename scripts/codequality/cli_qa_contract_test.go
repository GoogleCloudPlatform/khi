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
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/decisioncheck"
)

type contractSkippedSource struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

type contractJSONReport struct {
	BaseLabel      string                  `json:"base_label"`
	Speed          string                  `json:"speed"`
	Filter         string                  `json:"filter"`
	MustFix        []json.RawMessage       `json:"mustfix"`
	Advisory       []json.RawMessage       `json:"advisory"`
	SkippedSources []contractSkippedSource `json:"skipped_sources"`
}

type contractDeciderStub struct {
	answer decisioncheck.WireAnswer
}

type contractSubcommandCase struct {
	name           string
	args           []string
	wantCode       int
	wantSpeed      string
	wantFilter     string
	wantMinMustFix int
	wantMaxMustFix int
	wantBaseSubstr string
	forbidBaseSub  string
	expectJSON     bool
	wantStdoutSubs []string
}

type contractRejectedCase struct {
	name       string
	argLists   [][]string
	wantErrSub string
}

type contractASTSubcmdCase struct {
	name       string
	args       []string
	wantCode   int
	wantSubstr string
}

type contractJSONCase struct {
	name              string
	args              []string
	useStubDecider    bool
	useNilDecider     bool
	wantCode          int
	wantSpeed         string
	wantFilter        string
	wantMinMustFix    int
	wantMaxMustFix    int
	wantMinAdvisory   int
	wantMaxAdvisory   int
	wantMinSkipped    int
	wantMaxDecideCall int
}

// TestContractCLISubcommandsAndFlags verifies that qa and check recognize all new flags, scope filters, and output formats.
func TestContractCLISubcommandsAndFlags(t *testing.T) {
	gitRepo := initThreeCommitContractGitRepo(t)
	t.Chdir(gitRepo)
	testCases := []contractSubcommandCase{
		// Recognizes qa subcommand with fast speed, must filter, paths, changed-since, skip-sources, use-container, and json format.
		{
			name: "qa fast must accepts all new flags with empty path scope",
			args: []string{
				"qa", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~2",
				"--speed=fast", "--filter=must", "--paths=nonexistent/scope.go",
				"--changed-since=HEAD~1", "--skip-sources=decisions", "--use-container", "--format=json",
			},
			wantCode:       0,
			wantSpeed:      "fast",
			wantFilter:     "must",
			wantMinMustFix: 0,
			wantMaxMustFix: 0,
			wantBaseSubstr: "HEAD~2",
			expectJSON:     true,
		},
		// Recognizes qa subcommand in slow advisory mode without running blocking checks.
		{
			name: "qa slow advisory emits valid json",
			args: []string{
				"qa", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~2",
				"--speed=slow", "--filter=advisory", "--skip-sources=decisions", "--format=json",
			},
			wantCode:       0,
			wantSpeed:      "slow",
			wantFilter:     "advisory",
			wantMinMustFix: 0,
			wantMaxMustFix: 0,
			expectJSON:     true,
		},
		// Recognizes qa subcommand in all advisory mode.
		{
			name: "qa all advisory emits valid json",
			args: []string{
				"qa", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~2",
				"--speed=all", "--filter=advisory", "--skip-sources=decisions", "--format=json",
			},
			wantCode:       0,
			wantSpeed:      "all",
			wantFilter:     "advisory",
			wantMinMustFix: 0,
			wantMaxMustFix: 0,
			expectJSON:     true,
		},
		// Restricts check fast must evaluation to clean paths and excludes violations outside --paths.
		{
			name: "check paths excludes out-of-scope violation",
			args: []string{
				"check", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~2",
				"--speed=fast", "--filter=must", "--paths=pkg/clean", "--format=json",
			},
			wantCode:       0,
			wantSpeed:      "fast",
			wantFilter:     "must",
			wantMinMustFix: 0,
			wantMaxMustFix: 0,
			expectJSON:     true,
		},
		// Retains violations inside --paths and exits with code 1.
		{
			name: "check paths includes in-scope violation and exits 1",
			args: []string{
				"check", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~2",
				"--speed=fast", "--filter=must", "--paths=pkg/bad", "--format=json",
			},
			wantCode:       1,
			wantSpeed:      "fast",
			wantFilter:     "must",
			wantMinMustFix: 1,
			wantMaxMustFix: 10,
			expectJSON:     true,
		},
		// Filters target files via --changed-since while keeping BaseLabel anchored to --base.
		{
			name: "check changed-since filters intermediate commit while preserving base label",
			args: []string{
				"check", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~2",
				"--changed-since=HEAD~1", "--speed=fast", "--filter=must", "--format=json",
			},
			wantCode:       0,
			wantSpeed:      "fast",
			wantFilter:     "must",
			wantMinMustFix: 0,
			wantMaxMustFix: 0,
			wantBaseSubstr: "HEAD~2",
			forbidBaseSub:  "HEAD~1",
			expectJSON:     true,
		},
		// Executes slow dynamic Go checks cleanly on a well-tested package.
		{
			name: "check slow must passes clean covered Go package",
			args: []string{
				"check", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~2",
				"--speed=slow", "--filter=must", "--paths=pkg/clean", "--format=json",
			},
			wantCode:       0,
			wantSpeed:      "slow",
			wantFilter:     "must",
			wantMinMustFix: 0,
			wantMaxMustFix: 0,
			expectJSON:     true,
		},
		// Executes all speed and filter stages while accepting --use-container and --skip-sources.
		{
			name: "check all all passes clean scope with skip-sources and use-container",
			args: []string{
				"check", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~2",
				"--speed=all", "--filter=all", "--paths=pkg/clean", "--skip-sources=decisions", "--use-container", "--format=json",
			},
			wantCode:       0,
			wantSpeed:      "all",
			wantFilter:     "all",
			wantMinMustFix: 0,
			wantMaxMustFix: 0,
			expectJSON:     true,
		},
		// Emits human-readable text output on violation when --format=text is requested.
		{
			name: "check format text renders violation report",
			args: []string{
				"check", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~2",
				"--speed=fast", "--filter=must", "--paths=pkg/bad", "--format=text", "--color=never",
			},
			wantCode:       1,
			wantStdoutSubs: []string{"util.go"},
		},
		// Emits GitHub workflow error annotations when --format=github is requested.
		{
			name: "check format github renders workflow error annotation",
			args: []string{
				"check", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~2",
				"--speed=fast", "--filter=must", "--paths=pkg/bad", "--format=github",
			},
			wantCode:       1,
			wantStdoutSubs: []string{"::error ", "util.go"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdoutBuf, stderrBuf bytes.Buffer

			exitCode, err := runCLI(context.Background(), cliEnv{
				args:   tc.args,
				stdout: &stdoutBuf,
				stderr: &stderrBuf,
				getenv: func(string) string { return "" },
			})

			assertSubcommandOutcome(t, tc, exitCode, err, stdoutBuf.Bytes(), stderrBuf.String())
		})
	}
}

// TestContractCLIRejectedInputs verifies that unknown subcommands, invalid enums, and removed legacy flags return exit code 2.
func TestContractCLIRejectedInputs(t *testing.T) {
	gitRepo := initThreeCommitContractGitRepo(t)
	t.Chdir(gitRepo)

	testCases := []contractRejectedCase{
		// Rejects empty argument list with exit code 2 and a missing subcommand error.
		{
			name:       "missing subcommand returns exit code 2",
			argLists:   [][]string{nil},
			wantErrSub: "missing subcommand",
		},
		// Rejects an unrecognized subcommand with exit code 2 and an unknown subcommand error.
		{
			name:       "unknown subcommand returns exit code 2",
			argLists:   [][]string{{"unknown-subcommand"}},
			wantErrSub: "unknown subcommand",
		},
		// Rejects unsupported --speed enum values on both qa and check subcommands.
		{
			name: "invalid speed enum returns exit code 2",
			argLists: [][]string{
				{"qa", "--speed=turbo"},
				{"check", "--speed=turbo"},
			},
			wantErrSub: "invalid speed",
		},
		// Rejects unsupported --filter enum values on both qa and check subcommands.
		{
			name: "invalid filter enum returns exit code 2",
			argLists: [][]string{
				{"qa", "--filter=blocking"},
				{"check", "--filter=blocking"},
			},
			wantErrSub: "invalid filter",
		},
		// Rejects unsupported --format enum values on both qa and check subcommands.
		{
			name: "invalid format enum returns exit code 2",
			argLists: [][]string{
				{"qa", "--format=xml"},
				{"check", "--format=xml"},
			},
			wantErrSub: "invalid --format",
		},
		// Rejects all removed legacy boolean flags across qa, check, and root invocations.
		{
			name: "removed legacy flags return exit code 2",
			argLists: [][]string{
				{"--fast", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~1"},
				{"--conventions", "--repo-root=" + gitRepo},
				{"--unused-exports", "--repo-root=" + gitRepo},
				{"--decisions", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~1"},
				{"--decisions-only", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~1"},
				{"--similarity", "--repo-root=" + gitRepo, "--vcs=git", "--base=HEAD~1"},
				{"check", "--fast"},
				{"check", "--conventions"},
				{"check", "--unused-exports"},
				{"check", "--decisions-only"},
				{"qa", "--fast"},
				{"qa", "--decisions-only"},
			},
		},
	}

	for _, tc := range testCases {
		assertAllRejected(t, tc.argLists, tc.wantErrSub)
	}
}

// TestContractCLIConventionsAndUnusedExports verifies that conventions and unused-exports run as standalone subcommands with --pkgs and --repo-root.
func TestContractCLIConventionsAndUnusedExports(t *testing.T) {
	modDir := t.TempDir()
	writeContractFiles(t, modDir, map[string]string{
		"go.mod": "module example.com/subcmdmod\n\ngo 1.24\n",
		"pkg/clean/clean.go": `package clean

// PublicRecord holds a documented field.
type PublicRecord struct {
	// Label identifies the record.
	Label string
}

// ExportedHelper returns a constant label.
func ExportedHelper() string {
	return "ok"
}
`,
		"pkg/consumer/consumer.go": `package consumer

import "example.com/subcmdmod/pkg/clean"

// UseClean references clean package exports so unused-exports passes.
func UseClean() string {
	record := clean.PublicRecord{Label: clean.ExportedHelper()}
	return record.Label
}
`,
		"pkg/badconv/bad.go": `package badconv

// BrokenDoc lacks a comment on its exported field.
type BrokenDoc struct {
	MissingDoc int
}
`,
		"pkg/unused/unused.go": `package unused

// DeadExport is never referenced anywhere in the module.
const DeadExport = 42
`,
	})
	t.Chdir(modDir)

	testCases := []contractASTSubcmdCase{
		// Passes with exit code 0 when conventions checks a documented package without VCS metadata.
		{
			name:     "conventions passes clean package",
			args:     []string{"conventions", "--repo-root=" + modDir, "--pkgs=./pkg/clean"},
			wantCode: 0,
		},
		// Exits with code 1 and reports exported-member-doc when a struct field lacks GoDoc.
		{
			name:       "conventions reports violation with exit code 1",
			args:       []string{"conventions", "--repo-root=" + modDir, "--pkgs=./pkg/badconv"},
			wantCode:   1,
			wantSubstr: "exported-member-doc",
		},
		// Passes with exit code 0 when unused-exports inspects referenced packages.
		{
			name:     "unused-exports passes referenced packages",
			args:     []string{"unused-exports", "--repo-root=" + modDir, "--pkgs=./pkg/clean"},
			wantCode: 0,
		},
		// Exits with code 1 and reports unused-exported-decl when an export has zero callers.
		{
			name:       "unused-exports reports unreferenced export with exit code 1",
			args:       []string{"unused-exports", "--repo-root=" + modDir, "--pkgs=./pkg/unused"},
			wantCode:   1,
			wantSubstr: "unused-exported-decl",
		},
	}

	for _, tc := range testCases {
		var stdoutBuf, stderrBuf bytes.Buffer

		exitCode, err := runCLI(context.Background(), cliEnv{
			args:   tc.args,
			stdout: &stdoutBuf,
			stderr: &stderrBuf,
			getenv: func(string) string { return "" },
		})

		if err != nil {
			t.Fatalf("%s: runCLI(%v) unexpected error: %v (stderr=%s)", tc.name, tc.args, err, stderrBuf.String())
		}
		if exitCode != tc.wantCode {
			t.Errorf("%s: runCLI(%v) exitCode = %d, want %d\nstdout:\n%s", tc.name, tc.args, exitCode, tc.wantCode, stdoutBuf.String())
		}
		if tc.wantCode == 0 && stdoutBuf.Len() != 0 {
			t.Errorf("%s: runCLI(%v) stdout = %q, want empty on clean pass", tc.name, tc.args, stdoutBuf.String())
		}
		if tc.wantSubstr != "" && !strings.Contains(stdoutBuf.String(), tc.wantSubstr) {
			t.Errorf("%s: runCLI(%v) stdout missing %q:\n%s", tc.name, tc.args, tc.wantSubstr, stdoutBuf.String())
		}
	}
}

// TestContractCLIUnifiedExitCodeAndJSON verifies exit codes across MUSTFIX, ADVISORY, skipped_sources, and strict JSON stdout schema.
func TestContractCLIUnifiedExitCodeAndJSON(t *testing.T) {
	violRepo, rulesDir := initAdvisoryAndViolationRepo(t)
	t.Chdir(violRepo)
	testCases := []contractJSONCase{
		// Exits 1 when at least one MUSTFIX violation exists under filter=must and suppresses decisions even when --decisions is set.
		{
			name:              "mustfix present exits 1",
			args:              []string{"check", "--repo-root=" + violRepo, "--vcs=git", "--base=HEAD~1", "--speed=fast", "--filter=must", "--decisions", "--decisions-rules-dir=" + rulesDir, "--format=json"},
			useStubDecider:    true,
			wantCode:          1,
			wantSpeed:         "fast",
			wantFilter:        "must",
			wantMinMustFix:    1,
			wantMaxMustFix:    10,
			wantMaxDecideCall: 0,
		},
		// Exits 0 without invoking decider when --decisions opt-in is not set.
		{
			name:              "advisory without decisions opt-in skips decider cleanly",
			args:              []string{"check", "--repo-root=" + violRepo, "--vcs=git", "--base=HEAD~1", "--speed=fast", "--filter=advisory", "--decisions-rules-dir=" + rulesDir, "--format=json"},
			useStubDecider:    true,
			wantCode:          0,
			wantSpeed:         "fast",
			wantFilter:        "advisory",
			wantMaxDecideCall: 0,
		},
		// Exits 0 when 0 MUSTFIX exist even though ADVISORY contains findings.
		{
			name:            "advisory only exits 0",
			args:            []string{"check", "--repo-root=" + violRepo, "--vcs=git", "--base=HEAD~1", "--speed=fast", "--filter=advisory", "--decisions", "--decisions-rules-dir=" + rulesDir, "--format=json"},
			useStubDecider:  true,
			wantCode:        0,
			wantSpeed:       "fast",
			wantFilter:      "advisory",
			wantMinAdvisory: 1,
			wantMaxAdvisory: 10,
		},
		// Exits 0 when 0 MUSTFIX exist and records unconfigured decision client in skipped_sources.
		{
			name:           "skipped_sources only exits 0",
			args:           []string{"check", "--repo-root=" + violRepo, "--vcs=git", "--base=HEAD~1", "--speed=fast", "--filter=advisory", "--decisions", "--decisions-rules-dir=" + rulesDir, "--format=json"},
			useNilDecider:  true,
			wantCode:       0,
			wantSpeed:      "fast",
			wantFilter:     "advisory",
			wantMinSkipped: 1,
		},
		// Suppresses decider invocation and exits 0 when --skip-sources=decisions is passed.
		{
			name:              "skip-sources decisions disables decider invocation",
			args:              []string{"check", "--repo-root=" + violRepo, "--vcs=git", "--base=HEAD~1", "--speed=fast", "--filter=advisory", "--decisions", "--decisions-rules-dir=" + rulesDir, "--skip-sources=decisions", "--format=json"},
			useStubDecider:    true,
			wantCode:          0,
			wantSpeed:         "fast",
			wantFilter:        "advisory",
			wantMinSkipped:    1,
			wantMaxDecideCall: 0,
		},
		// Emits both mustfix and advisory items in JSON and exits 1 under filter=all when MUSTFIX exists.
		{
			name:            "filter all combines mustfix and advisory and exits 1",
			args:            []string{"check", "--repo-root=" + violRepo, "--vcs=git", "--base=HEAD~1", "--speed=fast", "--filter=all", "--decisions", "--decisions-rules-dir=" + rulesDir, "--format=json"},
			useStubDecider:  true,
			wantCode:        1,
			wantSpeed:       "fast",
			wantFilter:      "all",
			wantMinMustFix:  1,
			wantMaxMustFix:  10,
			wantMinAdvisory: 1,
			wantMaxAdvisory: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cacheDir := t.TempDir()
			decideCalls := 0
			factory := buildContractDeciderFactory(tc, &decideCalls)
			var stdoutBuf, stderrBuf bytes.Buffer

			exitCode, err := runCLI(context.Background(), cliEnv{
				args:           tc.args,
				stdout:         &stdoutBuf,
				stderr:         &stderrBuf,
				deciderFactory: factory,
				getenv: func(key string) string {
					if key == "CODEQUALITY_DECISIONS_CACHE_DIR" {
						return cacheDir
					}
					return ""
				},
			})

			assertJSONContractOutcome(t, tc, exitCode, err, stdoutBuf.Bytes(), stderrBuf.String(), decideCalls)
		})
	}
}

// TestContractSlowAdvisoryGoDynamicAndFastAdvisoryDiff verifies that slow-must passes with 0 blocking when surviving mutants/uncovered lines exist, slow-advisory emits them in advisory[] with Evidence/ReviewPrompt, and fast-advisory emits offline diff signals with Exit 0.
func TestContractSlowAdvisoryGoDynamicAndFastAdvisoryDiff(t *testing.T) {
	repoDir := t.TempDir()
	runContractGit(t, repoDir, "init", "-b", "main")
	writeContractFiles(t, repoDir, map[string]string{
		"go.mod": `module example.com/goodhartcontract

go 1.24
`,
		"pkg/svc/svc.go": `package svc

// Check returns whether x is positive.
func Check(x int) bool {
	return x > 0
}
`,
		"pkg/svc/svc_test.go": `package svc

import "testing"

// TestCheck verifies Check behavior.
func TestCheck(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		if !Check(5) {
			t.Fatal("expected true")
		}
	})
	t.Run("removed later", func(t *testing.T) {
		if Check(-1) {
			t.Fatal("expected false")
		}
	})
}
`,
	})
	runContractGit(t, repoDir, "add", ".")
	runContractGit(t, repoDir, "commit", "-m", "base")

	writeContractFiles(t, repoDir, map[string]string{
		"pkg/svc/svc.go": `package svc

// Check returns whether x is at least 10 or negative.
func Check(x int) bool {
	if x < 0 {
		return false
	}
	return x >= 10
}
`,
		"pkg/svc/svc_test.go": `package svc

import "testing"

// TestCheck verifies Check behavior.
func TestCheck(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		if !Check(20) {
			t.Fatal("expected true")
		}
	})
}
`,
	})
	runContractGit(t, repoDir, "add", ".")
	runContractGit(t, repoDir, "commit", "-m", "introduce uncovered branch, surviving mutant, and reduced subtests")
	t.Chdir(repoDir)

	testCases := []struct {
		name            string
		args            []string
		wantMinAdvisory int
		wantChecks      []string
	}{
		// qa --speed=slow --filter=must exits 0 with 0 mustfix even though Check has uncovered lines and surviving mutants.
		{
			name: "qa slow must exits 0 despite uncovered lines and surviving mutants",
			args: []string{
				"qa", "--repo-root=" + repoDir, "--vcs=git", "--base=HEAD~1", "--speed=slow", "--filter=must",
				"--lang=go", "--skip-sources=unused_exports,deadcode,go_test_race,build_go", "--format=json",
			},
			wantMinAdvisory: 0,
		},
		// qa --speed=slow --filter=advisory exits 0 and emits Go dynamic findings with Evidence, EndLine, and ReviewPrompt in advisory[].
		{
			name:            "qa slow advisory emits Go dynamic findings with Evidence and ReviewPrompt",
			args:            []string{"qa", "--repo-root=" + repoDir, "--vcs=git", "--base=HEAD~1", "--speed=slow", "--filter=advisory", "--lang=go", "--format=json"},
			wantMinAdvisory: 2,
			wantChecks:      []string{"func_uncovered_lines", "func_surviving_mutants", "end_line"},
		},
		// qa --speed=fast --filter=advisory exits 0 and emits offline advisorycheck diff signals.
		{
			name:            "qa fast advisory emits offline advisorycheck diff signals",
			args:            []string{"qa", "--repo-root=" + repoDir, "--vcs=git", "--base=HEAD~1", "--speed=fast", "--filter=advisory", "--lang=go", "--format=json"},
			wantMinAdvisory: 1,
			wantChecks:      []string{"test_cases_reduced"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdoutBuf, stderrBuf bytes.Buffer
			exitCode, err := runCLI(context.Background(), cliEnv{
				args:   tc.args,
				stdout: &stdoutBuf,
				stderr: &stderrBuf,
				getenv: func(string) string { return "" },
			})
			if err != nil {
				t.Fatalf("runCLI() unexpected error: %v (stderr=%s)", err, stderrBuf.String())
			}
			if exitCode != 0 {
				t.Fatalf("runCLI() exitCode = %d, want 0\nstdout:\n%s", exitCode, stdoutBuf.String())
			}
			parsed := parseAndValidateJSONContract(t, stdoutBuf.Bytes())
			if len(parsed.MustFix) != 0 {
				t.Errorf("len(mustfix) = %d, want 0", len(parsed.MustFix))
			}
			if len(parsed.Advisory) < tc.wantMinAdvisory {
				t.Errorf("len(advisory) = %d, want >= %d\nstdout:\n%s", len(parsed.Advisory), tc.wantMinAdvisory, stdoutBuf.String())
			}
			var missingChecks []string
			outStr := stdoutBuf.String()
			for _, wantCheck := range tc.wantChecks {
				if !strings.Contains(outStr, `"`+wantCheck+`"`) {
					missingChecks = append(missingChecks, wantCheck)
				}
			}
			if len(missingChecks) > 0 {
				t.Errorf("JSON output missing expected checks %v:\n%s", missingChecks, outStr)
			}
		})
	}
}

func (s contractDeciderStub) Decide(_ context.Context, req decisioncheck.DecisionRequest) (decisioncheck.DecisionResponse, error) {
	answers := make(map[string]decisioncheck.WireAnswer, len(req.Questions))
	for qID := range req.Questions {
		answers[qID] = s.answer
	}
	return decisioncheck.DecisionResponse{Answers: answers}, nil
}

func assertSubcommandOutcome(t *testing.T, tc contractSubcommandCase, exitCode int, runErr error, stdoutBytes []byte, stderrText string) {
	t.Helper()
	if runErr != nil {
		t.Fatalf("runCLI(%v) unexpected error: %v (stderr=%s)", tc.args, runErr, stderrText)
	}
	if exitCode != tc.wantCode {
		t.Errorf("runCLI(%v) exitCode = %d, want %d\nstdout:\n%s\nstderr:\n%s", tc.args, exitCode, tc.wantCode, string(stdoutBytes), stderrText)
	}
	for _, sub := range tc.wantStdoutSubs {
		if !strings.Contains(string(stdoutBytes), sub) {
			t.Errorf("stdout missing %q:\n%s", sub, string(stdoutBytes))
		}
	}
	if tc.expectJSON {
		assertSubcommandJSON(t, tc, stdoutBytes)
	}
}

func assertSubcommandJSON(t *testing.T, tc contractSubcommandCase, stdoutBytes []byte) {
	t.Helper()
	parsed := parseAndValidateJSONContract(t, stdoutBytes)
	if parsed.Speed != tc.wantSpeed {
		t.Errorf("json speed = %q, want %q", parsed.Speed, tc.wantSpeed)
	}
	if parsed.Filter != tc.wantFilter {
		t.Errorf("json filter = %q, want %q", parsed.Filter, tc.wantFilter)
	}
	if len(parsed.MustFix) < tc.wantMinMustFix || len(parsed.MustFix) > tc.wantMaxMustFix {
		t.Errorf("json len(mustfix) = %d, want [%d, %d]", len(parsed.MustFix), tc.wantMinMustFix, tc.wantMaxMustFix)
	}
	if tc.wantBaseSubstr != "" && !strings.Contains(parsed.BaseLabel, tc.wantBaseSubstr) {
		t.Errorf("json base_label = %q, want substring %q", parsed.BaseLabel, tc.wantBaseSubstr)
	}
	if tc.forbidBaseSub != "" && strings.Contains(parsed.BaseLabel, tc.forbidBaseSub) {
		t.Errorf("json base_label = %q, must not contain %q", parsed.BaseLabel, tc.forbidBaseSub)
	}
}

func assertAllRejected(t *testing.T, argLists [][]string, wantErrSub string) {
	t.Helper()
	for _, args := range argLists {
		var stdoutBuf, stderrBuf bytes.Buffer
		exitCode, err := runCLI(context.Background(), cliEnv{
			args:   args,
			stdout: &stdoutBuf,
			stderr: &stderrBuf,
			getenv: func(string) string { return "" },
		})
		if exitCode != 2 {
			t.Errorf("runCLI(%v) exitCode = %d, want 2 (err=%v, stdout=%s)", args, exitCode, err, stdoutBuf.String())
		}
		if err == nil {
			t.Errorf("runCLI(%v) err = nil, want non-nil validation error", args)
		} else if wantErrSub != "" && !strings.Contains(err.Error(), wantErrSub) {
			t.Errorf("runCLI(%v) err = %v, want substring %q", args, err, wantErrSub)
		}
	}
}

func buildContractDeciderFactory(tc contractJSONCase, decideCalls *int) deciderFactory {
	if tc.useNilDecider || !tc.useStubDecider {
		return nil
	}
	return func(context.Context, decisioncheck.DeciderOptions) (decisioncheck.Decider, string, error) {
		*decideCalls++
		pFail := 0.08
		return contractDeciderStub{
			answer: decisioncheck.WireAnswer{Noul: &pFail, Confidence: 0.92},
		}, "", nil
	}
}

func assertJSONContractOutcome(t *testing.T, tc contractJSONCase, exitCode int, runErr error, stdoutBytes []byte, stderrText string, decideCalls int) {
	t.Helper()
	if runErr != nil {
		t.Fatalf("runCLI(%v) unexpected error: %v (stderr=%s)", tc.args, runErr, stderrText)
	}
	if exitCode != tc.wantCode {
		t.Errorf("runCLI(%v) exitCode = %d, want %d\nstdout:\n%s", tc.args, exitCode, tc.wantCode, string(stdoutBytes))
	}
	parsed := parseAndValidateJSONContract(t, stdoutBytes)
	if parsed.BaseLabel == "" {
		t.Errorf("json base_label is empty")
	}
	if parsed.Speed != tc.wantSpeed {
		t.Errorf("json speed = %q, want %q", parsed.Speed, tc.wantSpeed)
	}
	if parsed.Filter != tc.wantFilter {
		t.Errorf("json filter = %q, want %q", parsed.Filter, tc.wantFilter)
	}
	if len(parsed.MustFix) < tc.wantMinMustFix || len(parsed.MustFix) > tc.wantMaxMustFix {
		t.Errorf("json len(mustfix) = %d, want [%d, %d]", len(parsed.MustFix), tc.wantMinMustFix, tc.wantMaxMustFix)
	}
	if len(parsed.Advisory) < tc.wantMinAdvisory || len(parsed.Advisory) > tc.wantMaxAdvisory {
		t.Errorf("json len(advisory) = %d, want [%d, %d]", len(parsed.Advisory), tc.wantMinAdvisory, tc.wantMaxAdvisory)
	}
	if len(parsed.SkippedSources) < tc.wantMinSkipped {
		t.Errorf("json len(skipped_sources) = %d, want >= %d", len(parsed.SkippedSources), tc.wantMinSkipped)
	}
	if tc.wantMinSkipped > 0 && (parsed.SkippedSources[0].Source == "" || parsed.SkippedSources[0].Reason == "") {
		t.Errorf("json skipped_sources[0] = %+v, want non-empty source and reason", parsed.SkippedSources[0])
	}
	if decideCalls > tc.wantMaxDecideCall && tc.wantMinAdvisory == 0 {
		t.Errorf("deciderFactory calls = %d, want <= %d", decideCalls, tc.wantMaxDecideCall)
	}
}

func parseAndValidateJSONContract(t *testing.T, stdoutBytes []byte) contractJSONReport {
	t.Helper()
	trimmed := bytes.TrimSpace(stdoutBytes)
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not strictly valid JSON:\n%s", string(stdoutBytes))
	}
	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &rawFields); err != nil {
		t.Fatalf("failed to unmarshal JSON keys: %v", err)
	}
	gotKeys := make([]string, 0, len(rawFields))
	for key := range rawFields {
		gotKeys = append(gotKeys, key)
	}
	slices.Sort(gotKeys)
	wantKeys := []string{"advisory", "base_label", "filter", "mustfix", "skipped_sources", "speed"}
	if diff := cmp.Diff(wantKeys, gotKeys); diff != "" {
		t.Errorf("JSON top-level keys mismatch (-want +got):\n%s", diff)
	}
	for _, arrayKey := range []string{"mustfix", "advisory", "skipped_sources"} {
		valBytes := bytes.TrimSpace(rawFields[arrayKey])
		if len(valBytes) == 0 || valBytes[0] != '[' {
			t.Errorf("JSON field %q = %s, want non-null JSON array", arrayKey, string(valBytes))
		}
	}
	var parsed contractJSONReport
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		t.Fatalf("failed to decode contractJSONReport: %v", err)
	}
	return parsed
}

func initThreeCommitContractGitRepo(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()
	runContractGit(t, repoDir, "init", "-b", "main")
	writeContractFiles(t, repoDir, map[string]string{
		"go.mod": "module example.com/contract\n\ngo 1.24\n",
		"pkg/clean/clean.go": `package clean

// Base returns a fixed integer for the baseline package.
func Base() int { return 1 }
`,
	})
	runContractGit(t, repoDir, "add", ".")
	runContractGit(t, repoDir, "commit", "-m", "base")

	writeContractFiles(t, repoDir, map[string]string{
		"pkg/bad/util.go": `package bad

// BadCatchall triggers catchall_filename in Stage-1 AST checks.
func BadCatchall() int { return 7 }
`,
	})
	runContractGit(t, repoDir, "add", ".")
	runContractGit(t, repoDir, "commit", "-m", "add catchall violation")

	writeContractFiles(t, repoDir, map[string]string{
		"pkg/clean/extra.go": `package clean

// Answer returns a constant value with full test coverage.
func Answer() int { return 42 }
`,
		"pkg/clean/extra_test.go": `package clean

import "testing"

// TestAnswer verifies that Answer returns 42.
func TestAnswer(t *testing.T) {
	if got := Answer(); got != 42 {
		t.Fatalf("Answer() = %d, want 42", got)
	}
}
`,
	})
	runContractGit(t, repoDir, "add", ".")
	runContractGit(t, repoDir, "commit", "-m", "add clean tested file")
	return repoDir
}

func initAdvisoryAndViolationRepo(t *testing.T) (string, string) {
	t.Helper()
	repoDir := t.TempDir()
	runContractGit(t, repoDir, "init", "-b", "main")
	writeContractFiles(t, repoDir, map[string]string{
		"go.mod":            "module example.com/advisorymod\n\ngo 1.24\n",
		"pkg/foo/README.md": "# Foo\n\nInitial overview.\n",
		"pkg/foo/store.go":  "package foo\n\n// Store holds package state.\ntype Store struct{}\n",
	})
	runContractGit(t, repoDir, "add", ".")
	runContractGit(t, repoDir, "commit", "-m", "base")

	writeContractFiles(t, repoDir, map[string]string{
		"pkg/foo/extra.go": "package foo\n\n// AddedUndocumented represents a newly added type.\ntype AddedUndocumented struct{}\n",
		"pkg/bad/util.go":  "package bad\n\n// ViolateCatchall triggers catchall_filename.\nfunc ViolateCatchall() int { return 1 }\n",
	})
	runContractGit(t, repoDir, "add", ".")
	runContractGit(t, repoDir, "commit", "-m", "add advisory and mustfix changes")

	rulesDir := filepath.Join(t.TempDir(), "rules")
	writeContractFiles(t, rulesDir, map[string]string{
		"doc.yaml": `
id: pkg_doc_alignment
name: Package Doc Alignment
enforcement: advisory
target:
  kind: go_logical_package
  include: ["pkg/**"]
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
	return repoDir, rulesDir
}

func writeContractFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", filepath.Dir(fullPath), err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", fullPath, err)
		}
	}
}

func runContractGit(t *testing.T, dir string, args ...string) {
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
