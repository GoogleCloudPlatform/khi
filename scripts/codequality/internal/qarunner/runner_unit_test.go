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
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestFilterDeadcodeOutputEdgeCases verifies catch-all patterns, trailing slashes, empty entries, blank lines, and malformed lines.
func TestFilterDeadcodeOutputEdgeCases(t *testing.T) {
	raw := `pkg/common/collection.go:12:6: unreachable func: deadHelper

not-a-diagnostic-line
pkg/other/legacy.go:30:6: unreachable func: deadLegacy`

	testCases := []struct {
		name  string
		paths []string
		want  string
	}{
		// Preserves full output untouched when a catch-all dot or wildcard pattern is provided.
		{name: "catch-all dot pattern preserves output", paths: []string{"."}, want: raw},
		// Trims trailing slash, skips empty entries, and drops blank and malformed lines.
		{name: "trailing slash and empty entry scope correctly", paths: []string{"", "pkg/common/"}, want: `pkg/common/collection.go:12:6: unreachable func: deadHelper`},
		// Strips recursive /... suffix when matching directory prefixes.
		{name: "recursive suffix without non-wildcard sibling", paths: []string{"./pkg/common/..."}, want: `pkg/common/collection.go:12:6: unreachable func: deadHelper`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterDeadcodeOutput(raw, tc.paths)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FilterDeadcodeOutput() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestExecCommandRunner verifies real process execution for working directory, env propagation, non-zero exit, and missing binary errors.
func TestExecCommandRunner(t *testing.T) {
	tmpDir := t.TempDir()

	// Verifies exit 0 runs in dir, captures stdout, and forwards custom environment variables.
	t.Run("success with dir and env", func(t *testing.T) {
		res := ExecCommandRunner(context.Background(), tmpDir, []string{"KHI_TEST_ENV=hello"}, "/bin/sh", "-c", "printf '%s:%s' \"$PWD\" \"$KHI_TEST_ENV\"")
		if res.Err != nil || res.ExitCode != 0 || res.Stdout != tmpDir+":hello" {
			t.Errorf("ExecCommandRunner() = %+v, want ExitCode=0, Stdout=%q, Err=nil", res, tmpDir+":hello")
		}
	})

	// Verifies non-zero exit code captures both stdout and stderr with nil launch error.
	t.Run("non-zero exit captures stdout and stderr", func(t *testing.T) {
		res := ExecCommandRunner(context.Background(), tmpDir, nil, "/bin/sh", "-c", "printf 'out'; printf 'err' >&2; exit 3")
		if res.Err != nil || res.ExitCode != 3 || res.Stdout != "out" || res.Stderr != "err" {
			t.Errorf("ExecCommandRunner() = %+v, want ExitCode=3, Stdout=out, Stderr=err, Err=nil", res)
		}
	})

	// Verifies missing executable returns ExitCode -1 and non-nil Err.
	t.Run("missing binary returns launch error", func(t *testing.T) {
		res := ExecCommandRunner(context.Background(), tmpDir, nil, filepath.Join(tmpDir, "nonexistent-binary"))
		if res.Err == nil || res.ExitCode != -1 {
			t.Errorf("ExecCommandRunner() = %+v, want ExitCode=-1 and non-nil Err", res)
		}
	})
}

// TestDomainScopeAndHelpers verifies domain scoping, global triggers, license filtering, directory checks, and output combining.
func TestDomainScopeAndHelpers(t *testing.T) {
	// Verifies individual global Go triggers set hasGo and goAllPkgs even when goDirs is non-empty.
	t.Run("global go triggers", func(t *testing.T) {
		repoRoot := materializeRepoFiles(t, []string{"pkg/common/collection.go"})
		for _, trigger := range []string{"go.mod", "go.sum", ".golangci.yaml", "scripts/backend-codegen/gen.go"} {
			scope := buildDomainScope(repoRoot, gotestutil.DiffResult{ChangedFiles: []string{trigger, "pkg/common/collection.go"}}, "go")
			if !scope.hasGo || !scope.goAllPkgs || len(scope.goDirs) != 1 {
				t.Errorf("buildDomainScope(%s) = %+v, want hasGo=true, goAllPkgs=true, len(goDirs)=1", trigger, scope)
			}
			if diff := cmp.Diff([]string{"./..."}, scope.goPkgArgs()); diff != "" {
				t.Errorf("goPkgArgs(%s) mismatch (-want +got):\n%s", trigger, diff)
			}
		}
		emptyScope := buildDomainScope(t.TempDir(), gotestutil.DiffResult{ChangedFiles: []string{"go.mod"}}, "go")
		if !emptyScope.hasGo || emptyScope.goDirs == nil || len(emptyScope.goDirs) != 0 {
			t.Errorf("empty go.mod scope = %+v, want hasGo=true and non-nil empty goDirs", emptyScope)
		}
	})

	// Verifies individual global Web triggers set hasWeb.
	t.Run("global web triggers", func(t *testing.T) {
		for _, trigger := range []string{
			".node-version",
			"scripts/frontend-codegen/gen.go",
			"scripts/fontlist-gen/main.go",
			"scripts/msdf-generator/index.js",
			"scripts/generate-angular-json.sh",
			"scripts/generate-version.sh",
		} {
			scope := buildDomainScope(t.TempDir(), gotestutil.DiffResult{ChangedFiles: []string{trigger}}, "ts")
			if !scope.hasWeb {
				t.Errorf("buildDomainScope(%s).hasWeb = false, want true", trigger)
			}
		}
	})

	// Verifies misc YAML/JSON files under managed config directories populate miscPrettierFiles.
	t.Run("misc prettier config files", func(t *testing.T) {
		repoRoot := materializeRepoFiles(t, []string{
			".cloudbuild/ci.yaml",
			".github/workflows/pr.yml",
			".vscode/settings.json",
			"scripts/codequality/rules/go.yaml",
			".github/README.md",
			"pkg/other/data.json",
		})
		scope := buildDomainScope(repoRoot, gotestutil.DiffResult{
			ChangedFiles: []string{
				".cloudbuild/ci.yaml",
				".github/workflows/pr.yml",
				".vscode/settings.json",
				"scripts/codequality/rules/go.yaml",
				".github/README.md",
				"pkg/other/data.json",
			},
		}, "all")
		wantMisc := []string{
			".cloudbuild/ci.yaml",
			".github/workflows/pr.yml",
			".vscode/settings.json",
			"scripts/codequality/rules/go.yaml",
		}
		if diff := cmp.Diff(wantMisc, scope.miscPrettierFiles); diff != "" {
			t.Errorf("miscPrettierFiles mismatch (-want +got):\n%s", diff)
		}
	})

	// Verifies lang filtering, deleted/nonexistent files, root Go files, generated/golden/LICENSE exclusions, and web extensions.
	t.Run("mixed files and lang exclusions", func(t *testing.T) {
		assertMixedDomainScope(t)
	})
}

func assertMixedDomainScope(t *testing.T) {
	t.Helper()
	repoRoot := materializeRepoFiles(t, []string{
		"main.go",
		"pkg/generated/types.go",
		"web/a.ts",
		"web/src/app/panel.html",
		"web/src/app/panel.json",
		"web/src/app/panel.mjs",
		"pkg/common/test.golden",
		"LICENSE",
		"docs/guide.md",
		"proto/a.proto",
	})
	if err := os.MkdirAll(filepath.Join(repoRoot, "pkg/empty/sub"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "pkg/empty/README.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	diff := gotestutil.DiffResult{
		ChangedFiles: []string{
			"", "main.go", "pkg/deleted/gone.go", "pkg/missing/x.go", "pkg/generated/types.go",
			"web/a.ts", "web/src/app/missing.ts", "web/src/app/panel.html", "web/src/app/panel.json",
			"web/src/app/panel.mjs", "pkg/common/test.golden", "LICENSE", "docs/guide.md", "proto/a.proto",
		},
	}
	scopeAll := buildDomainScope(repoRoot, diff, "all")
	scopeGo := buildDomainScope(repoRoot, diff, "go")

	if diff := cmp.Diff([]string{"main.go"}, scopeAll.goFiles); diff != "" {
		t.Errorf("scopeAll.goFiles mismatch (-want +got):\n%s", diff)
	}
	if !slices.Contains(scopeAll.goDirs, ".") || !scopeAll.goAllPkgs {
		t.Errorf("scopeAll Go dirs/allPkgs unexpected: %+v", scopeAll)
	}
	if diff := cmp.Diff([]string{"a.ts", "src/app/panel.html", "src/app/panel.json", "src/app/panel.mjs"}, scopeAll.prettierFiles); diff != "" {
		t.Errorf("prettierFiles mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"main.go", "pkg/generated/types.go"}, scopeGo.licenseFiles); diff != "" {
		t.Errorf("scopeGo.licenseFiles mismatch (-want +got):\n%s", diff)
	}
	if len(scopeGo.mdFiles) != 0 || scopeGo.hasProto {
		t.Errorf("scopeGo leaked misc files: %+v", scopeGo)
	}
	if dirHasGoFiles(repoRoot, "pkg/empty") || dirHasGoFiles(repoRoot, "pkg/nonexistent") {
		t.Errorf("dirHasGoFiles() unexpectedly returned true")
	}
	if got := combineCommandOutput(CommandResult{Stdout: "out", Stderr: "err"}); got != "out\nerr" {
		t.Errorf("combineCommandOutput(both) = %q, want %q", got, "out\nerr")
	}
	if got := combineCommandOutput(CommandResult{Stdout: "out"}); got != "out" {
		t.Errorf("combineCommandOutput(stdout) = %q, want %q", got, "out")
	}
}

// TestRunEdgeCasesAndFailures verifies nil runner/callbacks, filter isolation, failure reporting, deadcode parsing, and read-only build/test flags.
func TestRunEdgeCasesAndFailures(t *testing.T) {
	// Verifies CommandRunner == nil defaults to ExecCommandRunner and CodequalityCheck filter isolation + SkippedSources merging.
	t.Run("default runner and codequality filter isolation", func(t *testing.T) {
		repoRoot := materializeRepoFiles(t, []string{"docs/guide.md"})
		cfg := RunConfig{
			RepoRoot:    repoRoot,
			Speed:       rule.SpeedFast,
			Filter:      rule.FilterMust,
			Lang:        "all",
			SkipSources: map[string]struct{}{"markdownlint": {}},
			Diff:        buildTestDiff("docs/guide.md"),
			CodequalityCheck: func(_ context.Context, _ rule.Speed, _ rule.Filter) (*rule.QAReport, error) {
				return &rule.QAReport{
					MustFix:        []rule.QAItem{{ID: "m1"}},
					Advisory:       []rule.QAItem{{ID: "a1"}},
					SkippedSources: []rule.SkippedSource{{Source: "cq-skip", Reason: "disabled"}},
				}, nil
			},
		}
		repMust, err := Run(context.Background(), cfg)
		if err != nil || len(repMust.MustFix) != 1 || len(repMust.Advisory) != 0 || len(repMust.SkippedSources) != 2 {
			t.Fatalf("Run(FilterMust) = (%+v, %v), want 1 mustfix, 0 advisory, 2 skipped", repMust, err)
		}

		cfg.Filter = rule.FilterAdvisory
		repAdv, err := Run(context.Background(), cfg)
		if err != nil || len(repAdv.MustFix) != 0 || len(repAdv.Advisory) != 1 {
			t.Fatalf("Run(FilterAdvisory) = (%+v, %v), want 0 mustfix, 1 advisory", repAdv, err)
		}
	})

	// Verifies nil runner actually executes ExecCommandRunner on a real command and CodequalityCheck returning nil is tolerated.
	t.Run("nil runner executes real command", func(t *testing.T) {
		repoRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("WriteFile() error: %v", err)
		}
		cfg := RunConfig{
			RepoRoot:         repoRoot,
			Speed:            rule.SpeedFast,
			Filter:           rule.FilterMust,
			Lang:             "go",
			SkipSources:      map[string]struct{}{"addlicense": {}, "golangci_lint": {}, "go_test_short": {}},
			Diff:             buildTestDiff("main.go"),
			CodequalityCheck: func(_ context.Context, _ rule.Speed, _ rule.Filter) (*rule.QAReport, error) { return nil, nil },
		}
		rep, err := Run(context.Background(), cfg)
		if err != nil || rep.HasMustFix() {
			t.Fatalf("Run(nil CommandRunner) = (%+v, %v), want no error and no MustFix", rep, err)
		}
	})

	// Verifies SpeedAll does not fail-fast when Go is untouched or go_test_short is skipped.
	t.Run("speed all without go_test_short execution proceeds to slow", func(t *testing.T) {
		cfgWeb, cmdWeb, _ := newTestSetup(t, []string{"web/src/app/panel.ts"}, nil)
		cfgWeb.Speed, cfgWeb.Lang = rule.SpeedAll, "all"
		if _, err := Run(context.Background(), cfgWeb); err != nil || !slices.Contains(cmdWeb.logicalChecks(), "test_web") {
			t.Fatalf("Run(web only all) failed to reach slow phase: checks=%v err=%v", cmdWeb.logicalChecks(), err)
		}

		cfgSkip, cmdSkip, _ := newTestSetup(t, []string{"pkg/common/collection.go"}, nil)
		cfgSkip.Speed, cfgSkip.SkipSources = rule.SpeedAll, map[string]struct{}{"go_test_short": {}}
		if _, err := Run(context.Background(), cfgSkip); err != nil || !slices.Contains(cmdSkip.logicalChecks(), "go_test_race") {
			t.Fatalf("Run(skip go_test_short all) failed to reach slow phase: checks=%v err=%v", cmdSkip.logicalChecks(), err)
		}
	})

	// Verifies fast gofmt ID formatting, non-zero exit with stdout, empty-stdout gofmt failure, and empty-output fallback detail.
	t.Run("fast check failures and fallback details", func(t *testing.T) {
		cfg, _, _ := newTestSetup(t, []string{"pkg/common/collection.go", "docs/guide.md"}, map[string]CommandResult{
			"gofmt":        {Stdout: "pkg/common/collection.go\n", ExitCode: 1},
			"markdownlint": {ExitCode: 2},
		})
		cfg.Lang, cfg.CodequalityCheck = "all", nil

		rep, err := Run(context.Background(), cfg)
		if err != nil {
			t.Fatalf("Run() unexpected error: %v", err)
		}
		if len(rep.MustFix) != 2 || rep.MustFix[0].Detail != "markdownlint exited with code 2" || rep.MustFix[1].ID != "gofmt:pkg/common/collection.go" {
			t.Errorf("MustFix mismatch: %+v", rep.MustFix)
		}

		cfgEmpty, _, _ := newTestSetup(t, []string{"pkg/common/collection.go"}, map[string]CommandResult{
			"gofmt": {ExitCode: 2},
		})
		cfgEmpty.CodequalityCheck = nil
		repEmpty, err := Run(context.Background(), cfgEmpty)
		if err != nil || len(repEmpty.MustFix) != 1 || repEmpty.MustFix[0].ID != "gofmt" {
			t.Errorf("Run(empty gofmt failure) = (%+v, %v), want 1 gofmt failure", repEmpty, err)
		}
	})

	// Verifies slow Go and Web failures, deadcode parsing, unused_exports, and read-only build/test flags.
	t.Run("slow failures deadcode parsing and read-only args", func(t *testing.T) {
		var recordedEnv [][]string
		var recordedArgs [][]string
		cfg, _, _ := newTestSetup(t, []string{"pkg/common/collection.go", "web/src/app/panel.ts"}, nil)
		cfg.Speed, cfg.Lang = rule.SpeedSlow, "all"
		cfg.UnusedExportsCheck = func(_ string, _ []string) ([]rule.QAItem, error) {
			return []rule.QAItem{{ID: "unused_exports:1", Source: "unused_exports", Check: "unused-exported-decl", File: "pkg/common/collection.go", Line: 5, Summary: "unused"}}, nil
		}
		cfg.CommandRunner = func(_ context.Context, _ string, env []string, name string, args ...string) CommandResult {
			recordedEnv = append(recordedEnv, slices.Clone(env))
			recordedArgs = append(recordedArgs, append([]string{name}, args...))
			switch classifyCommand(name, args) {
			case "deadcode":
				return CommandResult{Stdout: "\nmalformed-line\npkg/common/collection.go:99999999999999999999999:1: unreachable\n"}
			case "build_go", "lint_unused", "build_web", "build_web_dev", "build_storybook", "test_web":
				return CommandResult{ExitCode: 1, Stderr: "failed"}
			default:
				return CommandResult{}
			}
		}

		rep, err := Run(context.Background(), cfg)
		if err != nil {
			t.Fatalf("Run() unexpected error: %v", err)
		}
		if len(rep.MustFix) != 9 {
			t.Errorf("MustFix len = %d, want 9: %+v", len(rep.MustFix), rep.MustFix)
		}
		assertSlowFlagsAndEnv(t, recordedEnv, recordedArgs)
	})

	// Verifies cross-platform build_go targets, UnusedExportsCheck skip/error, proto slow scoping, and fast prettier_misc execution.
	t.Run("cross-platform build unused exports proto and misc prettier", func(t *testing.T) {
		assertSlowParityAndMiscChecks(t)
	})

	// Verifies deadcode non-zero exit with empty stdout records a MustFix failure.
	t.Run("deadcode empty stdout non-zero exit", func(t *testing.T) {
		cfg, _, _ := newTestSetup(t, []string{"pkg/common/collection.go"}, map[string]CommandResult{
			"deadcode": {ExitCode: 1, Stderr: "deadcode build failed"},
		})
		cfg.Speed = rule.SpeedSlow

		rep, err := Run(context.Background(), cfg)
		if err != nil || !rep.HasMustFix() {
			t.Fatalf("Run() = (%+v, %v), want HasMustFix=true", rep, err)
		}
	})

	// Verifies UseContainer routes test_web through container runtime without running Go slow checks.
	t.Run("use container routes test_web", func(t *testing.T) {
		t.Setenv("CONTAINER_CMD", "docker")
		t.Setenv("BUILDER_IMAGE", "khi-builder:test")
		cfg, cmdRec, _ := newTestSetup(t, []string{"web/src/app/panel.ts"}, nil)
		cfg.Speed, cfg.Lang, cfg.UseContainer = rule.SpeedSlow, "ts", true
		cfg.SkipSources = map[string]struct{}{"lint_unused": {}, "build_web": {}, "build_web_dev": {}, "build_storybook": {}}

		if _, err := Run(context.Background(), cfg); err != nil {
			t.Fatalf("Run() unexpected error: %v", err)
		}
		calls := cmdRec.snapshot()
		if diff := cmp.Diff([]string{"test_web"}, cmdRec.logicalChecks()); diff != "" {
			t.Errorf("logicalChecks mismatch (-want +got):\n%s", diff)
		}
		if len(calls) != 1 || calls[0].Name != "docker" || !slices.Contains(calls[0].Args, cfg.RepoRoot+":/workspace") {
			t.Errorf("expected docker invocation with repo mount, got: %+v", calls)
		}
	})
}

func assertSlowParityAndMiscChecks(t *testing.T) {
	t.Helper()

	var goBuildEnvs [][]string
	cfgGo, _, _ := newTestSetup(t, []string{"pkg/common/collection.go"}, nil)
	cfgGo.Speed = rule.SpeedSlow
	cfgGo.SkipSources = map[string]struct{}{"deadcode": {}, "go_test_race": {}, "unused_exports": {}}
	unusedCalled := false
	cfgGo.UnusedExportsCheck = func(string, []string) ([]rule.QAItem, error) {
		unusedCalled = true
		return nil, nil
	}
	cfgGo.CommandRunner = func(_ context.Context, _ string, env []string, _ string, _ ...string) CommandResult {
		goBuildEnvs = append(goBuildEnvs, slices.Clone(env))
		return CommandResult{}
	}
	repGo, err := Run(context.Background(), cfgGo)
	if err != nil || unusedCalled || len(goBuildEnvs) != len(goReleaseTargets) || len(repGo.SkippedSources) != 3 {
		t.Fatalf("build_go targets/skip mismatch: err=%v unusedCalled=%v buildCalls=%d skipped=%+v", err, unusedCalled, len(goBuildEnvs), repGo.SkippedSources)
	}
	for i, target := range goReleaseTargets {
		wantEnv := []string{"CGO_ENABLED=0", "GOOS=" + target.goos, "GOARCH=" + target.goarch}
		if diff := cmp.Diff(wantEnv, goBuildEnvs[i]); diff != "" {
			t.Errorf("build_go target[%d] env mismatch (-want +got):\n%s", i, diff)
		}
	}

	cfgErr, _, _ := newTestSetup(t, []string{"pkg/common/collection.go"}, nil)
	cfgErr.Speed = rule.SpeedSlow
	cfgErr.UnusedExportsCheck = func(string, []string) ([]rule.QAItem, error) {
		return nil, os.ErrPermission
	}
	if _, runErr := Run(context.Background(), cfgErr); runErr == nil {
		t.Errorf("expected error from UnusedExportsCheck to propagate")
	}

	cfgProto, cmdProto, _ := newTestSetup(t, []string{"proto/khifile/v6/khifile.proto"}, nil)
	cfgProto.Speed, cfgProto.Lang = rule.SpeedSlow, "all"
	if _, protoErr := Run(context.Background(), cfgProto); protoErr != nil {
		t.Fatalf("Run(proto slow all) error: %v", protoErr)
	}
	wantProtoSlow := []string{"build_go", "build_storybook", "build_web", "build_web_dev", "deadcode", "go_test_race", "lint_unused", "test_web"}
	if diff := cmp.Diff(wantProtoSlow, cmdProto.logicalChecks()); diff != "" {
		t.Errorf("proto slow all checks mismatch (-want +got):\n%s", diff)
	}

	cfgMisc, cmdMisc, _ := newTestSetup(t, []string{".github/workflows/pullrequest.yaml"}, map[string]CommandResult{
		"prettier_misc": {ExitCode: 1, Stderr: "unformatted yaml"},
	})
	cfgMisc.Speed, cfgMisc.Lang = rule.SpeedFast, "all"
	repMisc, miscErr := Run(context.Background(), cfgMisc)
	if miscErr != nil || !slices.Contains(cmdMisc.logicalChecks(), "prettier_misc") || !repMisc.HasMustFix() {
		t.Fatalf("Run(prettier_misc) = (checks=%v, mustfix=%+v, err=%v), want prettier_misc MustFix", cmdMisc.logicalChecks(), repMisc.MustFix, miscErr)
	}
}

func assertSlowFlagsAndEnv(t *testing.T, envs [][]string, cmds [][]string) {
	t.Helper()
	if !slices.ContainsFunc(envs, func(e []string) bool { return slices.Contains(e, "KHI_SKIP_CLOUD_LOGGING=true") }) {
		t.Errorf("expected KHI_SKIP_CLOUD_LOGGING=true in go_test_race env, got %v", envs)
	}
	for _, cmd := range cmds {
		joined := strings.Join(cmd, " ")
		if strings.Contains(joined, "pkg/server/dist") {
			t.Errorf("build_web unexpectedly targeted repo pkg/server/dist: %v", cmd)
		}
		if len(cmd) >= 4 && cmd[0] == "go" && cmd[1] == "build" && cmd[2] == "-o" && !strings.HasSuffix(cmd[3], string(os.PathSeparator)) {
			t.Errorf("build_go -o dir %q missing trailing separator", cmd[3])
		}
		if strings.Contains(joined, "ng test") && !slices.Contains(cmd, "--progress=false") {
			t.Errorf("test_web missing --progress=false: %v", cmd)
		}
	}
}
