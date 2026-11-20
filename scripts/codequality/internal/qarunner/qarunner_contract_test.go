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

package qarunner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type recordedCommand struct {
	Dir  string
	Name string
	Args []string
}

type commandRecorder struct {
	mu        sync.Mutex
	calls     []recordedCommand
	responses map[string]CommandResult
}

type speedFilterCall struct {
	Speed  rule.Speed
	Filter rule.Filter
}

type codequalityRecorder struct {
	mu           sync.Mutex
	calls        []speedFilterCall
	emitAdvisory bool
	mustFix      []rule.QAItem
	err          error
}

type mockAdvisoryProvider struct {
	name     string
	speed    rule.Speed
	items    []rule.QAItem
	skipped  *rule.SkippedSource
	err      error
	mu       sync.Mutex
	requests []rule.AdvisoryRequest
}

type classifyErrorOutcome struct {
	ExternalSkip    bool
	ReasonCitesFile bool
	ReasonNonEmpty  bool
}

type runMatrixOutcome struct {
	BaseLabel        string
	Speed            rule.Speed
	Filter           rule.Filter
	ExternalChecks   []string
	CodequalityCalls []speedFilterCall
	ProviderNames    []string
	HasMustFix       bool
	AdvisoryCount    int
}

type argExpectation struct {
	CheckName  string
	WantArgs   []string
	ForbidArgs []string
}

type failFastOutcome struct {
	HasMustFix             bool
	CodequalityCalls       []speedFilterCall
	RanSlowExternal        bool
	RanSlowProvider        bool
	HasSlowSkippedNonEmpty bool
}

type faultIsolationOutcome struct {
	HasMustFix        bool
	MustFixSources    []string
	SkippedSources    []string
	CitesExternalDocs bool
}

type advisoryProviderOutcome struct {
	Requests       []rule.AdvisoryRequest
	SkippedSources []string
}

// TestClassifyGoCompileError verifies fast fault-isolation classification for Go compiler and test diagnostics.
func TestClassifyGoCompileError(t *testing.T) {
	targetFiles := map[string]struct{}{"pkg/common/collection.go": {}}
	testCases := []struct {
		name      string
		output    string
		speed     rule.Speed
		citedFile string
		want      classifyErrorOutcome
	}{
		// Downgrades a sibling compile error in the same Go package outside targetFiles during fast checks.
		{
			name: "fast skips external sibling file compile error",
			output: `# github.com/GoogleCloudPlatform/khi/pkg/common [github.com/GoogleCloudPlatform/khi/pkg/common.test]
pkg/common/timeseries.go:42:10: syntax error: unexpected newline, expected }
FAIL	github.com/GoogleCloudPlatform/khi/pkg/common [build failed]`,
			speed: rule.SpeedFast, citedFile: "pkg/common/timeseries.go",
			want: classifyErrorOutcome{ExternalSkip: true, ReasonCitesFile: true, ReasonNonEmpty: true},
		},
		// Treats a compile error inside targetFiles as MUSTFIX during fast checks.
		{
			name: "fast blocks on target file compile error", speed: rule.SpeedFast,
			output: `# github.com/GoogleCloudPlatform/khi/pkg/common
pkg/common/collection.go:18:5: undefined: MissingHelper`,
		},
		// Treats mixed external and target diagnostics conservatively as MUSTFIX during fast checks.
		{
			name: "fast blocks when both external and target diagnostics appear", speed: rule.SpeedFast,
			output: `pkg/common/timeseries.go:42:10: syntax error
pkg/common/collection.go:18:5: undefined: MissingHelper`,
		},
		// Never downgrades runtime test failures even when the failing assertion line sits outside targetFiles.
		{
			name: "fast blocks on runtime test failure", speed: rule.SpeedFast,
			output: `--- FAIL: TestFoo (0.01s)
    timeseries_test.go:25: got 1, want 2
FAIL
FAIL	github.com/GoogleCloudPlatform/khi/pkg/common	0.012s`,
		},
		// Treats unattributable build failures with no file:line diagnostics conservatively as MUSTFIX.
		{
			name: "fast blocks on unattributable build error", speed: rule.SpeedFast,
			output: `go: error loading module requirements
FAIL	github.com/GoogleCloudPlatform/khi/pkg/common [setup failed]`,
		},
		// Treats empty diagnostic output conservatively as MUSTFIX.
		{name: "fast blocks on empty output", output: "", speed: rule.SpeedFast},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotSkip, gotReason := ClassifyGoCompileError(tc.output, targetFiles, tc.speed)

			gotOutcome := classifyErrorOutcome{
				ExternalSkip:    gotSkip,
				ReasonCitesFile: tc.citedFile != "" && strings.Contains(gotReason, tc.citedFile),
				ReasonNonEmpty:  gotReason != "",
			}
			if diff := cmp.Diff(tc.want, gotOutcome); diff != "" {
				t.Errorf("ClassifyGoCompileError() mismatch (-want +got):\n%s\nreason=%q", diff, gotReason)
			}
		})
	}

	// Never downgrades external compile errors under slow or all speed, and handles upstream/dot-slash external & target diagnostics.
	t.Run("additional external and target diagnostic variants", func(t *testing.T) {
		assertCompileErrorVariants(t, targetFiles)
	})
}

func assertCompileErrorVariants(t *testing.T, targetFiles map[string]struct{}) {
	t.Helper()
	for _, speed := range []rule.Speed{rule.SpeedSlow, rule.SpeedAll} {
		gotSkip, gotReason := ClassifyGoCompileError(`pkg/common/timeseries.go:42:10: syntax error`, targetFiles, speed)
		if gotSkip || gotReason != "" {
			t.Errorf("ClassifyGoCompileError(speed=%q) = (%v, %q), want (false, \"\")", speed, gotSkip, gotReason)
		}
	}
	for _, item := range []struct {
		output    string
		citedFile string
		wantSkip  bool
	}{
		{"# github.com/GoogleCloudPlatform/khi/pkg/upstream\npkg/upstream/foo.go:10:2: undefined: MissingSymbol", "pkg/upstream/foo.go", true},
		{"./pkg/upstream/bar.go:10:2: undefined: MissingSymbol (typecheck)", "pkg/upstream/bar.go", true},
		{"./pkg/common/collection.go:18:5: undefined: MissingHelper", "", false},
	} {
		gotSkip, gotReason := ClassifyGoCompileError(item.output, targetFiles, rule.SpeedFast)
		if gotSkip != item.wantSkip || (item.citedFile != "" && !strings.Contains(gotReason, item.citedFile)) {
			t.Errorf("ClassifyGoCompileError(%q) = (%v, %q), want skip=%v cited=%q", item.output, gotSkip, gotReason, item.wantSkip, item.citedFile)
		}
	}
}

// TestFilterDeadcodeOutput verifies path-based filtering of go tool deadcode output lines.
func TestFilterDeadcodeOutput(t *testing.T) {
	sampleOutput := `pkg/common/collection.go:12:6: unreachable func: deadHelper
pkg/common/sub/worker.go:24:6: unreachable func: deadWorker
pkg/common2/other.go:19:6: unreachable func: deadSiblingPrefix
pkg/other/legacy.go:30:6: unreachable func: deadLegacy`
	testCases := []struct {
		name  string
		paths []string
		want  string
	}{
		// Returns the full deadcode output untouched when paths is empty.
		{name: "empty paths preserves output", paths: nil, want: sampleOutput},
		// Retains only matching exact file lines when a file path is provided.
		{name: "exact file path filter", paths: []string{"pkg/common/collection.go"}, want: `pkg/common/collection.go:12:6: unreachable func: deadHelper`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterDeadcodeOutput(sampleOutput, tc.paths)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FilterDeadcodeOutput() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	assertDeadcodePathCases(t, sampleOutput)
}

func assertDeadcodePathCases(t *testing.T, sampleOutput string) {
	t.Helper()
	want := `pkg/common/collection.go:12:6: unreachable func: deadHelper
pkg/common/sub/worker.go:24:6: unreachable func: deadWorker`
	for _, paths := range [][]string{{"pkg/common"}, {"./pkg/common/..."}} {
		got := FilterDeadcodeOutput(sampleOutput, paths)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("FilterDeadcodeOutput(%v) mismatch (-want +got):\n%s", paths, diff)
		}
	}
	if got := FilterDeadcodeOutput(sampleOutput, []string{"cmd/nonexistent/..."}); got != "" {
		t.Errorf("FilterDeadcodeOutput(nonexistent) = %q, want empty string", got)
	}
}

// TestRunSpeedFilterMatrix verifies external step selection, codequality delegation, and advisory routing across all 9 (speed, filter) pairs.
func TestRunSpeedFilterMatrix(t *testing.T) {
	fastChecks := []string{"addlicense", "eslint", "go_test_short", "gofmt", "golangci_lint", "prettier"}
	slowChecks := []string{"build_go", "build_storybook", "build_web", "build_web_dev", "deadcode", "go_test_race", "lint_unused", "test_web"}
	allChecks := []string{"addlicense", "build_go", "build_storybook", "build_web", "build_web_dev", "deadcode", "eslint", "go_test_race", "go_test_short", "gofmt", "golangci_lint", "lint_unused", "prettier", "test_web"}
	testCases := []struct {
		name   string
		speed  rule.Speed
		filter rule.Filter
		want   runMatrixOutcome
	}{
		// Runs only scoped fast MUSTFIX checks and delegates to CodequalityCheck(fast, must) without slow or advisory steps.
		{name: "fast must", speed: rule.SpeedFast, filter: rule.FilterMust, want: runMatrixOutcome{BaseLabel: "@-", Speed: rule.SpeedFast, Filter: rule.FilterMust, ExternalChecks: fastChecks, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedFast, Filter: rule.FilterMust}}, ProviderNames: []string{}}},
		// Skips all external commands and aggregates fast advisory items without setting HasMustFix.
		{name: "fast advisory", speed: rule.SpeedFast, filter: rule.FilterAdvisory, want: runMatrixOutcome{BaseLabel: "@-", Speed: rule.SpeedFast, Filter: rule.FilterAdvisory, ExternalChecks: []string{}, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedFast, Filter: rule.FilterAdvisory}}, ProviderNames: []string{"fast-provider"}, AdvisoryCount: 2}},
		// Combines scoped fast MUSTFIX checks with fast advisory providers and CodequalityCheck(fast, all).
		{name: "fast all", speed: rule.SpeedFast, filter: rule.FilterAll, want: runMatrixOutcome{BaseLabel: "@-", Speed: rule.SpeedFast, Filter: rule.FilterAll, ExternalChecks: fastChecks, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedFast, Filter: rule.FilterAll}}, ProviderNames: []string{"fast-provider"}, AdvisoryCount: 2}},
		// Runs cross-cutting slow MUSTFIX checks and delegates to CodequalityCheck(slow, must) without advisory providers.
		{name: "slow must", speed: rule.SpeedSlow, filter: rule.FilterMust, want: runMatrixOutcome{BaseLabel: "@-", Speed: rule.SpeedSlow, Filter: rule.FilterMust, ExternalChecks: slowChecks, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedSlow, Filter: rule.FilterMust}}, ProviderNames: []string{}}},
		// Skips all external commands and delegates only to slow advisory evaluation.
		{name: "slow advisory", speed: rule.SpeedSlow, filter: rule.FilterAdvisory, want: runMatrixOutcome{BaseLabel: "@-", Speed: rule.SpeedSlow, Filter: rule.FilterAdvisory, ExternalChecks: []string{}, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedSlow, Filter: rule.FilterAdvisory}}, ProviderNames: []string{"slow-provider"}, AdvisoryCount: 2}},
		// Combines slow MUSTFIX checks with slow advisory providers and CodequalityCheck(slow, all).
		{name: "slow all", speed: rule.SpeedSlow, filter: rule.FilterAll, want: runMatrixOutcome{BaseLabel: "@-", Speed: rule.SpeedSlow, Filter: rule.FilterAll, ExternalChecks: slowChecks, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedSlow, Filter: rule.FilterAll}}, ProviderNames: []string{"slow-provider"}, AdvisoryCount: 2}},
		// Executes fast then slow MUSTFIX phases when fast passes cleanly without running advisory checks.
		{name: "all must", speed: rule.SpeedAll, filter: rule.FilterMust, want: runMatrixOutcome{BaseLabel: "@-", Speed: rule.SpeedAll, Filter: rule.FilterMust, ExternalChecks: allChecks, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedFast, Filter: rule.FilterMust}, {Speed: rule.SpeedSlow, Filter: rule.FilterMust}}, ProviderNames: []string{}}},
		// Executes fast and slow advisory phases with zero external commands.
		{name: "all advisory", speed: rule.SpeedAll, filter: rule.FilterAdvisory, want: runMatrixOutcome{BaseLabel: "@-", Speed: rule.SpeedAll, Filter: rule.FilterAdvisory, ExternalChecks: []string{}, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedFast, Filter: rule.FilterAdvisory}, {Speed: rule.SpeedSlow, Filter: rule.FilterAdvisory}}, ProviderNames: []string{"fast-provider", "slow-provider"}, AdvisoryCount: 4}},
		// Executes both fast and slow MUSTFIX and advisory checks when fast passes cleanly.
		{name: "all all", speed: rule.SpeedAll, filter: rule.FilterAll, want: runMatrixOutcome{BaseLabel: "@-", Speed: rule.SpeedAll, Filter: rule.FilterAll, ExternalChecks: allChecks, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedFast, Filter: rule.FilterAll}, {Speed: rule.SpeedSlow, Filter: rule.FilterAll}}, ProviderNames: []string{"fast-provider", "slow-provider"}, AdvisoryCount: 4}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, cmdRec, cqRec := newTestSetup(t, []string{"pkg/common/collection.go", "web/src/app/panel.ts"}, nil)
			cqRec.emitAdvisory = true
			fastProvider := &mockAdvisoryProvider{name: "fast-provider", speed: rule.SpeedFast, items: []rule.QAItem{{ID: "fast-1", Source: "fast-provider"}}}
			slowProvider := &mockAdvisoryProvider{name: "slow-provider", speed: rule.SpeedSlow, items: []rule.QAItem{{ID: "slow-1", Source: "slow-provider"}}}
			cfg.Speed, cfg.Filter, cfg.Lang = tc.speed, tc.filter, "all"
			cfg.AdvisoryProviders = []rule.AdvisoryProvider{fastProvider, slowProvider}

			gotReport, err := Run(context.Background(), cfg)

			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}
			gotOutcome := runMatrixOutcome{
				BaseLabel: gotReport.BaseLabel, Speed: gotReport.Speed, Filter: gotReport.Filter,
				ExternalChecks: cmdRec.logicalChecks(), CodequalityCalls: cqRec.snapshot(),
				ProviderNames: invokedProviderNames(fastProvider, slowProvider),
				HasMustFix:    gotReport.HasMustFix(), AdvisoryCount: len(gotReport.Advisory),
			}
			if diff := cmp.Diff(tc.want, gotOutcome); diff != "" {
				t.Errorf("Run() matrix outcome mismatch (-want +got):\n%s", diff)
			}
			assertReadOnlyCommands(t, cmdRec.snapshot())
		})
	}
}

// TestRunFastDomainScoping verifies that fast mode runs only checks for touched domains with file/dir scoping.
func TestRunFastDomainScoping(t *testing.T) {
	testCases := []struct {
		name            string
		lang            string
		files           []string
		wantChecks      []string
		argExpectations []argExpectation
	}{
		// Scopes Go fast checks to touched files and target package directories without running web, proto, md, or slow checks.
		{
			name: "go only multi-package diff", lang: "all", files: []string{"pkg/common/collection.go", "pkg/api/handler_test.go"},
			wantChecks: []string{"addlicense", "go_test_short", "gofmt", "golangci_lint"},
			argExpectations: []argExpectation{
				{CheckName: "addlicense", WantArgs: []string{"-check", "pkg/api/handler_test.go", "pkg/common/collection.go"}, ForbidArgs: []string{"."}},
				{CheckName: "gofmt", WantArgs: []string{"-l", "pkg/api/handler_test.go", "pkg/common/collection.go"}, ForbidArgs: []string{"-w", "."}},
				{CheckName: "golangci_lint", WantArgs: []string{"./pkg/api", "./pkg/common"}, ForbidArgs: []string{"./...", "--fix"}},
				{CheckName: "go_test_short", WantArgs: []string{"-short", "./pkg/api", "./pkg/common"}, ForbidArgs: []string{"./...", "-race"}},
			},
		},
		// Scopes TS, SCSS, and web/scripts checks to modified frontend files without running Go, proto, md, or Karma tests.
		{
			name: "ts scss and web scripts diff", lang: "all", files: []string{"web/src/app/panel.ts", "web/src/app/panel.scss", "web/scripts/check-unused-exports.js"},
			wantChecks: []string{"addlicense", "eslint", "prettier", "stylelint", "test_scripts"},
			argExpectations: []argExpectation{
				{CheckName: "prettier", WantArgs: []string{"panel.scss", "panel.ts"}},
				{CheckName: "eslint", WantArgs: []string{"panel.ts"}, ForbidArgs: []string{"ng"}},
				{CheckName: "stylelint", WantArgs: []string{"panel.scss"}},
			},
		},
		// Runs buf lint and license check only when proto files are touched.
		{
			name: "proto only diff", lang: "all", files: []string{"proto/khifile/v6/khifile.proto"},
			wantChecks:      []string{"addlicense", "buf_lint"},
			argExpectations: []argExpectation{{CheckName: "buf_lint", WantArgs: []string{"lint"}, ForbidArgs: []string{"format"}}},
		},
		// Runs markdownlint-cli2 only when markdown files are touched.
		{
			name: "markdown only diff", lang: "all", files: []string{"docs/guide.md"},
			wantChecks:      []string{"markdownlint"},
			argExpectations: []argExpectation{{CheckName: "markdownlint", WantArgs: []string{"docs/guide.md"}}},
		},
		// Restricts checks to Go domain when Lang is go despite TypeScript edits in the diff.
		{name: "lang go filters out ts checks", lang: "go", files: []string{"pkg/common/collection.go", "web/src/app/panel.ts"}, wantChecks: []string{"addlicense", "go_test_short", "gofmt", "golangci_lint"}},
		// Restricts checks to TypeScript domain when Lang is ts despite Go edits in the diff.
		{name: "lang ts filters out go checks", lang: "ts", files: []string{"pkg/common/collection.go", "web/src/app/panel.ts"}, wantChecks: []string{"addlicense", "eslint", "prettier"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, cmdRec, _ := newTestSetup(t, tc.files, nil)
			cfg.Lang = tc.lang

			gotReport, err := Run(context.Background(), cfg)

			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}
			if gotReport.HasMustFix() {
				t.Errorf("Run() HasMustFix = true, want false (mustfix: %+v)", gotReport.MustFix)
			}
			if diff := cmp.Diff(tc.wantChecks, cmdRec.logicalChecks()); diff != "" {
				t.Errorf("Run() external checks mismatch (-want +got):\n%s", diff)
			}
			assertCommandArgExpectations(t, cmdRec.snapshot(), tc.argExpectations)
			assertReadOnlyCommands(t, cmdRec.snapshot())
		})
	}
}

// TestRunAllFailFast verifies fail-fast skipping of the slow phase on fast build/test failures and full slow verification when fast isolates external compile errors.
func TestRunAllFailFast(t *testing.T) {
	testCases := []struct {
		name      string
		filter    rule.Filter
		responses map[string]CommandResult
		want      failFastOutcome
	}{
		// Skips slow dynamic execution and CodequalityCheck(slow, must) when fast go test -short reports a runtime failure under (all, must).
		{
			name: "all must skips slow on fast runtime test failure", filter: rule.FilterMust,
			responses: map[string]CommandResult{"go_test_short": {Stdout: `--- FAIL: TestCollection (0.01s)
    collection_test.go:18: got 1, want 2
FAIL`, ExitCode: 1}},
			want: failFastOutcome{HasMustFix: true, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedFast, Filter: rule.FilterMust}}, HasSlowSkippedNonEmpty: true},
		},
		// Skips slow dynamic execution, CodequalityCheck(slow, all), and slow advisory providers on an in-scope fast compile error under (all, all).
		{
			name: "all all skips slow on fast target compile failure", filter: rule.FilterAll,
			responses: map[string]CommandResult{"go_test_short": {Stderr: `pkg/common/collection.go:10:2: undefined: BrokenSymbol`, ExitCode: 1}},
			want:      failFastOutcome{HasMustFix: true, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedFast, Filter: rule.FilterAll}}, HasSlowSkippedNonEmpty: true},
		},
		// Continues to slow phase when fast isolates an external compile error, allowing slow go test -race to report MUSTFIX.
		{
			name: "all must proceeds to slow when fast isolates external compile error", filter: rule.FilterMust,
			responses: map[string]CommandResult{
				"go_test_short": {Stderr: `pkg/upstream/broken.go:5:1: syntax error: unexpected EOF`, ExitCode: 1},
				"go_test_race":  {Stderr: `pkg/upstream/broken.go:5:1: syntax error: unexpected EOF`, ExitCode: 1},
			},
			want: failFastOutcome{HasMustFix: true, CodequalityCalls: []speedFilterCall{{Speed: rule.SpeedFast, Filter: rule.FilterMust}, {Speed: rule.SpeedSlow, Filter: rule.FilterMust}}, RanSlowExternal: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, cmdRec, cqRec := newTestSetup(t, []string{"pkg/common/collection.go"}, tc.responses)
			slowProvider := &mockAdvisoryProvider{name: "slow-provider", speed: rule.SpeedSlow, items: []rule.QAItem{{ID: "slow-1", Source: "slow-provider"}}}
			cfg.Speed, cfg.Filter = rule.SpeedAll, tc.filter
			cfg.AdvisoryProviders = []rule.AdvisoryProvider{slowProvider}

			gotReport, err := Run(context.Background(), cfg)

			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}
			gotOutcome := buildFailFastOutcome(gotReport, cmdRec, cqRec, slowProvider)
			if diff := cmp.Diff(tc.want, gotOutcome); diff != "" {
				t.Errorf("Run() fail-fast mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestRunFastFaultIsolation verifies external compile skips vs in-scope MUSTFIX retention across fast and slow runs.
func TestRunFastFaultIsolation(t *testing.T) {
	testCases := []struct {
		name       string
		speed      rule.Speed
		responses  map[string]CommandResult
		cqMustFix  []rule.QAItem
		citedFiles []string
		want       faultIsolationOutcome
	}{
		// Records external compile skips for go_test_short and golangci_lint while still reporting gofmt, addlicense, and codequality findings.
		{
			name: "fast isolates external compile errors while running single-file checks", speed: rule.SpeedFast,
			responses: map[string]CommandResult{
				"go_test_short": {Stderr: `pkg/common/timeseries.go:42:10: syntax error: unexpected newline`, ExitCode: 1},
				"golangci_lint": {Stderr: `pkg/upstream/foo.go:10:2: undefined: ExternalType (typecheck)`, ExitCode: 2},
				"gofmt":         {Stdout: "pkg/common/collection.go\n"},
				"addlicense":    {Stdout: "pkg/common/collection.go\n", ExitCode: 1},
			},
			cqMustFix:  []rule.QAItem{{ID: "ast-1", Source: "astcheck", File: "pkg/common/collection.go", Line: 8, Summary: "cognitive too high"}},
			citedFiles: []string{"pkg/common/timeseries.go", "pkg/upstream/foo.go"},
			want:       faultIsolationOutcome{HasMustFix: true, MustFixSources: []string{"addlicense", "astcheck", "gofmt"}, SkippedSources: []string{"go_test_short", "golangci_lint"}, CitesExternalDocs: true},
		},
		// Passes with HasMustFix false when external compile errors are isolated and all single-file checks pass cleanly.
		{
			name: "fast passes when only external compile errors fail", speed: rule.SpeedFast,
			responses: map[string]CommandResult{
				"go_test_short": {Stderr: `pkg/common/timeseries.go:42:10: syntax error: unexpected newline`, ExitCode: 1},
				"golangci_lint": {Stderr: `pkg/upstream/foo.go:10:2: undefined: ExternalType (typecheck)`, ExitCode: 2},
			},
			citedFiles: []string{"pkg/common/timeseries.go", "pkg/upstream/foo.go"},
			want:       faultIsolationOutcome{MustFixSources: []string{}, SkippedSources: []string{"go_test_short", "golangci_lint"}, CitesExternalDocs: true},
		},
		// Reports MUSTFIX without skipping when go test -short fails with a compile error inside targetFiles.
		{
			name: "fast blocks on target file compile error", speed: rule.SpeedFast,
			responses: map[string]CommandResult{"go_test_short": {Stderr: `pkg/common/collection.go:12:3: undefined: MissingSymbol`, ExitCode: 1}},
			want:      faultIsolationOutcome{HasMustFix: true, MustFixSources: []string{"go_test_short"}, SkippedSources: []string{}},
		},
		// Reports MUSTFIX without skipping when go test -short encounters a runtime test failure.
		{
			name: "fast blocks on runtime test failure", speed: rule.SpeedFast,
			responses: map[string]CommandResult{"go_test_short": {Stdout: `--- FAIL: TestCollection (0.00s)
    collection_test.go:20: mismatch
FAIL`, ExitCode: 1}},
			want: faultIsolationOutcome{HasMustFix: true, MustFixSources: []string{"go_test_short"}, SkippedSources: []string{}},
		},
		// Never downgrades an external compile error during slow mode.
		{
			name: "slow never skips external compile error", speed: rule.SpeedSlow,
			responses: map[string]CommandResult{"go_test_race": {Stderr: `pkg/common/timeseries.go:42:10: syntax error: unexpected newline`, ExitCode: 1}},
			want:      faultIsolationOutcome{HasMustFix: true, MustFixSources: []string{"go_test_race"}, SkippedSources: []string{}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, _, cqRec := newTestSetup(t, []string{"pkg/common/collection.go"}, tc.responses)
			cfg.Speed, cqRec.mustFix = tc.speed, tc.cqMustFix

			gotReport, err := Run(context.Background(), cfg)

			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}
			gotOutcome := buildFaultIsolationOutcome(gotReport, tc.citedFiles)
			if diff := cmp.Diff(tc.want, gotOutcome); diff != "" {
				t.Errorf("Run() fault-isolation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestRunSkipSources verifies that explicitly configured SkipSources are not executed and are recorded in SkippedSources.
func TestRunSkipSources(t *testing.T) {
	testCases := []struct {
		name        string
		speed       rule.Speed
		skipSources map[string]struct{}
		wantChecks  []string
		wantSkipped []string
	}{
		// Skips go_test_short and golangci_lint while continuing gofmt and addlicense in fast mode.
		{
			name: "skip fast go_test_short and golangci_lint", speed: rule.SpeedFast,
			skipSources: map[string]struct{}{"go_test_short": {}, "golangci_lint": {}},
			wantChecks:  []string{"addlicense", "gofmt"}, wantSkipped: []string{"go_test_short", "golangci_lint"},
		},
		// Skips deadcode while continuing go_test_race and build_go in slow mode.
		{
			name: "skip slow deadcode", speed: rule.SpeedSlow,
			skipSources: map[string]struct{}{"deadcode": {}},
			wantChecks:  []string{"build_go", "go_test_race"}, wantSkipped: []string{"deadcode"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, cmdRec, _ := newTestSetup(t, []string{"pkg/common/collection.go"}, nil)
			cfg.Speed, cfg.SkipSources = tc.speed, tc.skipSources

			gotReport, err := Run(context.Background(), cfg)

			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}
			got := [2][]string{cmdRec.logicalChecks(), sortedUniqueKeys(gotReport.SkippedSources, func(entry rule.SkippedSource) string { return entry.Source })}
			if diff := cmp.Diff([2][]string{tc.wantChecks, tc.wantSkipped}, got); diff != "" {
				t.Errorf("Run() SkipSources mismatch (-want +got):\n%s", diff)
			}
			if len(gotReport.SkippedSources) == 0 || slices.ContainsFunc(gotReport.SkippedSources, func(e rule.SkippedSource) bool { return strings.TrimSpace(e.Reason) == "" }) {
				t.Errorf("Run() SkipSources had empty reason: %+v", gotReport.SkippedSources)
			}
		})
	}
}

// TestRunDeadcodePathFiltering verifies that deadcode diagnostics in slow mode are filtered against RunConfig.Paths.
func TestRunDeadcodePathFiltering(t *testing.T) {
	testCases := []struct {
		name           string
		paths          []string
		deadcodeStdout string
		want           [3]bool
	}{
		// Retains only deadcode findings matching Paths and drops unrelated package diagnostics.
		{
			name: "filters deadcode output to scoped path", paths: []string{"pkg/common"},
			deadcodeStdout: `pkg/common/collection.go:15:6: unreachable func: deadInCommon
pkg/other/legacy.go:40:6: unreachable func: deadInOther`,
			want: [3]bool{true, true, false},
		},
		// Passes cleanly when all deadcode diagnostics belong to packages outside Paths.
		{
			name: "passes when deadcode exists only outside scoped path", paths: []string{"pkg/common"},
			deadcodeStdout: `pkg/other/legacy.go:40:6: unreachable func: deadInOther`, want: [3]bool{false, false, false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, _, _ := newTestSetup(t, []string{"pkg/common/collection.go"}, map[string]CommandResult{"deadcode": {Stdout: tc.deadcodeStdout}})
			cfg.Speed, cfg.Paths = rule.SpeedSlow, tc.paths

			gotReport, err := Run(context.Background(), cfg)

			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}
			got := [3]bool{gotReport.HasMustFix(), mustFixMentionsPath(gotReport.MustFix, "pkg/common/collection.go"), mustFixMentionsPath(gotReport.MustFix, "pkg/other/legacy.go")}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Run() deadcode path filtering mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestRunAdvisoryProviderAndEmptyDiff verifies AdvisoryRequest forwarding, SkippedSource merging, and empty diff normalization.
func TestRunAdvisoryProviderAndEmptyDiff(t *testing.T) {
	// Verifies AdvisoryRequest forwarding and SkippedSource propagation from an AdvisoryProvider.
	t.Run("advisory provider receives exact request and skipped source", func(t *testing.T) {
		cfg, _, _ := newTestSetup(t, []string{"pkg/common/collection.go"}, nil)
		trustedRoot := t.TempDir()
		provider := &mockAdvisoryProvider{
			name: "custom-advisory", speed: rule.SpeedFast,
			items:   []rule.QAItem{{ID: "adv-1", Source: "custom-advisory"}},
			skipped: &rule.SkippedSource{Source: "secondary-check", Reason: "auth missing"},
		}
		cfg.Filter, cfg.TrustedRoot = rule.FilterAdvisory, trustedRoot
		cfg.Diff.DeletedFiles = []string{"pkg/common/removed.go"}
		cfg.AdvisoryProviders = []rule.AdvisoryProvider{provider}

		gotReport, err := Run(context.Background(), cfg)

		if err != nil {
			t.Fatalf("Run() unexpected error: %v", err)
		}
		got := advisoryProviderOutcome{Requests: provider.snapshotRequests(), SkippedSources: sortedUniqueKeys(gotReport.SkippedSources, func(entry rule.SkippedSource) string { return entry.Source })}
		want := advisoryProviderOutcome{
			Requests: []rule.AdvisoryRequest{{
				RepoRoot: cfg.RepoRoot, TrustedRoot: trustedRoot,
				ChangedFiles: []string{"pkg/common/collection.go"}, DeletedFiles: []string{"pkg/common/removed.go"},
				ChangedLines: map[string]map[int]struct{}{"pkg/common/collection.go": {10: {}}},
			}},
			SkippedSources: []string{"secondary-check"},
		}
		if diffOutcome := cmp.Diff(want, got); diffOutcome != "" {
			t.Errorf("AdvisoryProvider outcome mismatch (-want +got):\n%s", diffOutcome)
		}
	})

	// Verifies that an empty diff executes zero external commands and returns a normalized empty QAReport.
	t.Run("empty diff runs zero external commands and normalizes slices", func(t *testing.T) {
		cfg, cmdRec, cqRec := newTestSetup(t, nil, nil)
		cfg.Lang = "all"

		gotReport, err := Run(context.Background(), cfg)

		if err != nil {
			t.Fatalf("Run() unexpected error: %v", err)
		}
		if len(cmdRec.snapshot()) != 0 {
			t.Errorf("Run() executed %d external commands on empty diff, want 0", len(cmdRec.snapshot()))
		}
		if len(cqRec.snapshot()) != 0 {
			t.Errorf("Run() executed %d codequality calls on empty diff, want 0", len(cqRec.snapshot()))
		}
		if gotReport.HasMustFix() || gotReport.MustFix == nil || gotReport.Advisory == nil || gotReport.SkippedSources == nil {
			t.Errorf("Run() empty report not normalized cleanly: %+v", gotReport)
		}
	})
}

// TestRunExecutionErrors verifies that subprocess launch errors, CodequalityCheck errors, and AdvisoryProvider errors propagate from Run.
func TestRunExecutionErrors(t *testing.T) {
	testCases := []struct {
		name        string
		filter      rule.Filter
		responses   map[string]CommandResult
		cqErr       error
		providerErr error
	}{
		// Propagates an OS/exec command launch error when CommandRunner sets CommandResult.Err.
		{name: "command runner launch error", filter: rule.FilterMust, responses: map[string]CommandResult{"gofmt": {Err: errors.New("exec: gofmt not found")}}},
		// Propagates internal CodequalityCheck execution failures directly to the caller.
		{name: "codequality callback error", filter: rule.FilterAdvisory, cqErr: errors.New("codequality internal failure")},
		// Propagates AdvisoryProvider evaluation failures directly to the caller.
		{name: "advisory provider error", filter: rule.FilterAdvisory, providerErr: errors.New("provider failure")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, _, cqRec := newTestSetup(t, []string{"pkg/common/collection.go"}, tc.responses)
			provider := &mockAdvisoryProvider{name: "broken-provider", speed: rule.SpeedFast, err: tc.providerErr}
			cfg.Filter, cqRec.err = tc.filter, tc.cqErr
			cfg.AdvisoryProviders = []rule.AdvisoryProvider{provider}

			_, err := Run(context.Background(), cfg)

			if err == nil {
				t.Errorf("Run() error = nil, want non-nil execution error")
			}
		})
	}
}

func newTestSetup(t *testing.T, relPaths []string, responses map[string]CommandResult) (RunConfig, *commandRecorder, *codequalityRecorder) {
	t.Helper()
	repoRoot := materializeRepoFiles(t, relPaths)
	cmdRec, cqRec := &commandRecorder{responses: responses}, &codequalityRecorder{}
	cfg := RunConfig{
		RepoRoot: repoRoot, Speed: rule.SpeedFast, Filter: rule.FilterMust, Lang: "go",
		Diff: buildTestDiff(relPaths...), CommandRunner: cmdRec.run, CodequalityCheck: cqRec.run,
	}
	return cfg, cmdRec, cqRec
}

func (r *commandRecorder) run(_ context.Context, dir string, _ []string, name string, args ...string) CommandResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	copiedArgs := slices.Clone(args)
	r.calls = append(r.calls, recordedCommand{Dir: dir, Name: name, Args: copiedArgs})
	if result, ok := r.responses[classifyCommand(name, copiedArgs)]; ok {
		return result
	}
	return CommandResult{}
}

func (r *commandRecorder) snapshot() []recordedCommand {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.calls)
}

func (r *commandRecorder) logicalChecks() []string {
	return sortedUniqueKeys(r.snapshot(), func(call recordedCommand) string {
		return classifyCommand(call.Name, call.Args)
	})
}

func (r *codequalityRecorder) run(_ context.Context, speed rule.Speed, filter rule.Filter) (*rule.QAReport, error) {
	r.mu.Lock()
	r.calls = append(r.calls, speedFilterCall{Speed: speed, Filter: filter})
	r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	rep := &rule.QAReport{BaseLabel: "@-", Speed: speed, Filter: filter, MustFix: slices.Clone(r.mustFix)}
	if r.emitAdvisory && filter.IncludesAdvisory() {
		rep.Advisory = []rule.QAItem{{ID: "cq-adv-" + string(speed), Source: "decisions"}}
	}
	return rep, nil
}

func (r *codequalityRecorder) snapshot() []speedFilterCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.calls)
}

func (p *mockAdvisoryProvider) Name() string { return p.name }

func (p *mockAdvisoryProvider) Speed() rule.Speed { return p.speed }

func (p *mockAdvisoryProvider) Evaluate(_ context.Context, req rule.AdvisoryRequest) ([]rule.QAItem, *rule.SkippedSource, error) {
	p.mu.Lock()
	p.requests = append(p.requests, req)
	p.mu.Unlock()
	if p.err != nil {
		return nil, nil, p.err
	}
	return slices.Clone(p.items), p.skipped, nil
}

func (p *mockAdvisoryProvider) snapshotRequests() []rule.AdvisoryRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.requests)
}

func classifyCommand(name string, args []string) string {
	joined := name + " " + strings.Join(args, " ")
	for _, pair := range [][2]string{
		{"addlicense", "addlicense"}, {"golangci_lint", "golangci-lint"}, {"deadcode", "deadcode"},
		{"eslint", "eslint"}, {"stylelint", "stylelint"},
		{"test_scripts", "test:scripts"}, {"lint_unused", "lint:unused"}, {"markdownlint", "markdownlint-cli2"},
	} {
		if strings.Contains(joined, pair[1]) {
			return pair[0]
		}
	}
	return classifyCompoundCommand(name, args, joined)
}

func classifyCompoundCommand(name string, args []string, joined string) string {
	switch {
	case strings.Contains(joined, "prettier"):
		if slices.ContainsFunc(args, isMiscPrettierFile) {
			return "prettier_misc"
		}
		return "prettier"
	case name == "gofmt":
		return "gofmt"
	case name == "go" && len(args) > 0 && args[0] == "test" && slices.Contains(args, "-short"):
		return "go_test_short"
	case name == "go" && len(args) > 0 && args[0] == "test" && slices.Contains(args, "-race"):
		return "go_test_race"
	case (name == "go" && len(args) > 0 && args[0] == "build") || strings.Contains(joined, "build-go"):
		return "build_go"
	case (slices.Contains(args, "ng") && slices.Contains(args, "test")) || slices.Contains(args, "test-web"):
		return "test_web"
	case strings.Contains(joined, "build-storybook"):
		return "build_storybook"
	case slices.Contains(args, "ng") && slices.Contains(args, "build") && slices.Contains(args, "dev"):
		return "build_web_dev"
	case (slices.Contains(args, "ng") && slices.Contains(args, "build")) || strings.Contains(joined, "build-web"):
		return "build_web"
	case (name == "buf" || strings.Contains(joined, "@bufbuild/buf")) && slices.Contains(args, "lint"):
		return "buf_lint"
	default:
		return name
	}
}

func buildTestDiff(relPaths ...string) gotestutil.DiffResult {
	res := gotestutil.DiffResult{
		BaseLabel: "@-", ChangedLines: make(map[string]map[int]struct{}),
		ChangedTestLines: make(map[string]map[int]struct{}), TSChangedLines: make(map[string]map[int]struct{}),
		ChangedFiles: slices.Clone(relPaths),
	}
	slices.Sort(res.ChangedFiles)
	for _, relPath := range res.ChangedFiles {
		switch {
		case strings.HasSuffix(relPath, "_test.go"):
			res.ChangedTestLines[relPath] = map[int]struct{}{15: {}}
		case strings.HasSuffix(relPath, ".go"):
			res.ChangedLines[relPath] = map[int]struct{}{10: {}}
		case strings.HasPrefix(relPath, "web/src/") && strings.HasSuffix(relPath, ".ts"):
			res.TSChangedLines[relPath] = map[int]struct{}{12: {}}
		}
	}
	return res
}

func materializeRepoFiles(t *testing.T, relPaths []string) string {
	t.Helper()
	repoRoot := t.TempDir()
	for _, relPath := range relPaths {
		fullPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error: %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte("// stub\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error: %v", fullPath, err)
		}
	}
	return repoRoot
}

func invokedProviderNames(providers ...*mockAdvisoryProvider) []string {
	names := make([]string, 0, len(providers))
	for _, provider := range providers {
		if len(provider.snapshotRequests()) > 0 {
			names = append(names, provider.Name())
		}
	}
	slices.Sort(names)
	return names
}

func sortedUniqueKeys[T any](items []T, keyFn func(T) string) []string {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		seen[keyFn(item)] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for key := range seen {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

func mustFixMentionsPath(items []rule.QAItem, targetPath string) bool {
	return slices.ContainsFunc(items, func(item rule.QAItem) bool {
		return item.File == targetPath || strings.Contains(item.Detail, targetPath) || strings.Contains(item.Summary, targetPath)
	})
}

func buildFailFastOutcome(rep *rule.QAReport, cmdRec *commandRecorder, cqRec *codequalityRecorder, slowProvider *mockAdvisoryProvider) failFastOutcome {
	ranSlow := slices.ContainsFunc(cmdRec.logicalChecks(), func(checkName string) bool {
		return checkName == "go_test_race" || checkName == "deadcode" || checkName == "build_go"
	})
	hasSlowSkipped := slices.ContainsFunc(rep.SkippedSources, func(entry rule.SkippedSource) bool {
		return strings.Contains(strings.ToLower(entry.Source), "slow") && strings.TrimSpace(entry.Reason) != ""
	})
	return failFastOutcome{
		HasMustFix: rep.HasMustFix(), CodequalityCalls: cqRec.snapshot(),
		RanSlowExternal: ranSlow, RanSlowProvider: len(slowProvider.snapshotRequests()) > 0,
		HasSlowSkippedNonEmpty: hasSlowSkipped,
	}
}

func buildFaultIsolationOutcome(rep *rule.QAReport, citedFiles []string) faultIsolationOutcome {
	citesAll := len(citedFiles) > 0
	for _, targetFile := range citedFiles {
		if !slices.ContainsFunc(rep.SkippedSources, func(entry rule.SkippedSource) bool {
			return strings.Contains(entry.Reason, targetFile)
		}) {
			citesAll = false
		}
	}
	return faultIsolationOutcome{
		HasMustFix:        rep.HasMustFix(),
		MustFixSources:    sortedUniqueKeys(rep.MustFix, func(item rule.QAItem) string { return item.Source }),
		SkippedSources:    sortedUniqueKeys(rep.SkippedSources, func(entry rule.SkippedSource) string { return entry.Source }),
		CitesExternalDocs: citesAll,
	}
}

func assertCommandArgExpectations(t *testing.T, calls []recordedCommand, expectations []argExpectation) {
	t.Helper()
	for _, exp := range expectations {
		idx := slices.IndexFunc(calls, func(call recordedCommand) bool {
			return classifyCommand(call.Name, call.Args) == exp.CheckName
		})
		if idx < 0 {
			t.Errorf("expected command %q was not invoked", exp.CheckName)
			continue
		}
		checkCommandTokens(t, calls[idx], exp)
	}
}

func checkCommandTokens(t *testing.T, call recordedCommand, exp argExpectation) {
	t.Helper()
	joined := call.Name + " " + strings.Join(call.Args, " ")
	for _, wantArg := range exp.WantArgs {
		if !strings.Contains(joined, wantArg) {
			t.Errorf("command %q args %q missing expected token %q", exp.CheckName, joined, wantArg)
		}
	}
	for _, forbidArg := range exp.ForbidArgs {
		if call.Name == forbidArg || slices.Contains(call.Args, forbidArg) {
			t.Errorf("command %q args %q unexpectedly contained forbidden token %q", exp.CheckName, joined, forbidArg)
		}
	}
}

func assertReadOnlyCommands(t *testing.T, calls []recordedCommand) {
	t.Helper()
	for _, call := range calls {
		assertCommandReadOnly(t, call)
	}
}

func assertCommandReadOnly(t *testing.T, call recordedCommand) {
	t.Helper()
	for _, flag := range []string{"--fix", "-w", "--write", "-i", "--in-place"} {
		if slices.Contains(call.Args, flag) {
			t.Errorf("command %s %v passed mutating flag %q; all qa commands must be read-only", call.Name, call.Args, flag)
		}
	}
	switch classifyCommand(call.Name, call.Args) {
	case "addlicense":
		if !slices.Contains(call.Args, "-check") || slices.Contains(call.Args, ".") {
			t.Errorf("addlicense command %v must pass -check on scoped files and never '.'", call.Args)
		}
	case "gofmt":
		if !slices.Contains(call.Args, "-l") {
			t.Errorf("gofmt command %v must pass -l in read-only mode", call.Args)
		}
	case "prettier":
		if !slices.Contains(call.Args, "--check") {
			t.Errorf("prettier command %v must pass --check in read-only mode", call.Args)
		}
	}
}
