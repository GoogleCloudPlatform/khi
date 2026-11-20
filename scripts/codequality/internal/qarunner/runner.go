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
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// RunConfig configures a unified differential QA run across external checks, codequality, and advisory providers.
type RunConfig struct {
	// RepoRoot is the absolute path to the working-tree repository root.
	RepoRoot string
	// TrustedRoot is the trusted repository root used for loading rules or configs.
	TrustedRoot string
	// Speed selects which execution tier (fast, slow, or all) runs.
	Speed rule.Speed
	// Filter selects which severity tier (all, must, or advisory) is evaluated.
	Filter rule.Filter
	// Lang restricts checks to a language domain ("all", "go", or "ts").
	Lang string
	// Paths narrows deadcode and scoped diagnostics to matching repository-relative patterns.
	Paths []string
	// Diff holds the differential file and line changes to evaluate.
	Diff gotestutil.DiffResult
	// SkipSources lists external check source names to skip explicitly.
	SkipSources map[string]struct{}
	// UseContainer enables containerized execution for supported external tools.
	UseContainer bool
	// CommandRunner executes external commands; defaults to ExecCommandRunner when nil.
	CommandRunner CommandRunner
	// CodequalityCheck evaluates internal AST, coverage, mutation, and decision rules.
	CodequalityCheck func(ctx context.Context, speed rule.Speed, filter rule.Filter) (*rule.QAReport, error)
	// UnusedExportsCheck evaluates whole-repo Go unused exported declarations filtered by paths.
	UnusedExportsCheck func(repoRoot string, paths []string) ([]rule.QAItem, error)
	// AdvisoryProviders lists external advisory analyzers evaluated during matching phases.
	AdvisoryProviders []rule.AdvisoryProvider
}

type commandSpec struct {
	dir  string
	env  []string
	name string
	args []string
}

// Run executes the configured QA phases and returns a normalized QAReport.
func Run(ctx context.Context, cfg RunConfig) (*rule.QAReport, error) {
	rep := &rule.QAReport{
		BaseLabel: cfg.Diff.BaseLabel,
		Speed:     cfg.Speed,
		Filter:    cfg.Filter,
	}
	if isEmptyDiff(cfg.Diff) {
		rep.Normalize()
		return rep, nil
	}
	if cfg.CommandRunner == nil {
		cfg.CommandRunner = ExecCommandRunner
	}

	scope := buildDomainScope(cfg.RepoRoot, cfg.Diff, cfg.Lang)
	fastBuildFailed := false

	if cfg.Speed.IncludesFast() {
		var err error
		fastBuildFailed, err = runFastPhase(ctx, cfg, scope, rep)
		if err != nil {
			return nil, err
		}
	}

	if cfg.Speed == rule.SpeedAll && cfg.Filter.IncludesMust() && fastBuildFailed {
		rep.SkippedSources = append(rep.SkippedSources, rule.SkippedSource{
			Source: "slow_phase",
			Reason: "skipped slow phase after fast build or test failure",
		})
		rep.Normalize()
		return rep, nil
	}

	if cfg.Speed.IncludesSlow() {
		if err := runSlowPhase(ctx, cfg, scope, rep); err != nil {
			return nil, err
		}
	}

	rep.Normalize()
	return rep, nil
}

func runFastPhase(ctx context.Context, cfg RunConfig, scope domainScope, rep *rule.QAReport) (bool, error) {
	fastBuildFailed := false
	if cfg.Filter.IncludesMust() {
		var err error
		fastBuildFailed, err = runFastExternalChecks(ctx, cfg, scope, rep)
		if err != nil {
			return false, err
		}
	}
	if err := runCodequalityPhase(ctx, cfg, rule.SpeedFast, rep); err != nil {
		return false, err
	}
	if cfg.Filter.IncludesAdvisory() {
		if err := runAdvisoryProviders(ctx, cfg, rule.SpeedFast, rep); err != nil {
			return false, err
		}
	}
	return fastBuildFailed, nil
}

func runSlowPhase(ctx context.Context, cfg RunConfig, scope domainScope, rep *rule.QAReport) error {
	if cfg.Filter.IncludesMust() {
		if err := runSlowExternalChecks(ctx, cfg, scope, rep); err != nil {
			return err
		}
	}
	if err := runCodequalityPhase(ctx, cfg, rule.SpeedSlow, rep); err != nil {
		return err
	}
	if cfg.Filter.IncludesAdvisory() {
		if err := runAdvisoryProviders(ctx, cfg, rule.SpeedSlow, rep); err != nil {
			return err
		}
	}
	return nil
}

func runFastExternalChecks(ctx context.Context, cfg RunConfig, scope domainScope, rep *rule.QAReport) (bool, error) {
	if err := runSimpleFileChecks(ctx, cfg, scope, rep); err != nil {
		return false, err
	}
	if err := runWebAndMiscFastChecks(ctx, cfg, scope, rep); err != nil {
		return false, err
	}
	return runGoFastPackageChecks(ctx, cfg, scope, rep)
}

func runSimpleFileChecks(ctx context.Context, cfg RunConfig, scope domainScope, rep *rule.QAReport) error {
	if len(scope.licenseFiles) > 0 && !isSourceSkipped(cfg.SkipSources, "addlicense", rep) {
		args := append([]string{"tool", "addlicense", "-c", "Google LLC", "-l", "apache", "-check"}, scope.licenseFiles...)
		res, err := invokeCommand(ctx, cfg, commandSpec{dir: cfg.RepoRoot, name: "go", args: args})
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			appendCommandFailure(rep, "addlicense", "missing or invalid license header", res)
		}
	}
	if len(scope.goFiles) > 0 && !isSourceSkipped(cfg.SkipSources, "gofmt", rep) {
		args := append([]string{"-l"}, scope.goFiles...)
		res, err := invokeCommand(ctx, cfg, commandSpec{dir: cfg.RepoRoot, name: "gofmt", args: args})
		if err != nil {
			return err
		}
		appendGofmtFindings(rep, res)
	}
	return nil
}

func runGoFastPackageChecks(ctx context.Context, cfg RunConfig, scope domainScope, rep *rule.QAReport) (bool, error) {
	if !scope.hasGo {
		return false, nil
	}
	pkgArgs := scope.goPkgArgs()

	if !isSourceSkipped(cfg.SkipSources, "golangci_lint", rep) {
		args := append([]string{"run", "--config=.golangci.yaml"}, pkgArgs...)
		res, err := invokeCommand(ctx, cfg, commandSpec{dir: cfg.RepoRoot, name: "golangci-lint", args: args})
		if err != nil {
			return false, err
		}
		handleFastGoCommandResult("golangci_lint", "golangci-lint reported issues", res, scope.targetGoFiles, rep)
	}

	if isSourceSkipped(cfg.SkipSources, "go_test_short", rep) {
		return false, nil
	}
	args := append([]string{"test", "-short"}, pkgArgs...)
	res, err := invokeCommand(ctx, cfg, commandSpec{dir: cfg.RepoRoot, env: []string{"KHI_SKIP_CLOUD_LOGGING=true"}, name: "go", args: args})
	if err != nil {
		return false, err
	}
	if handleFastGoCommandResult("go_test_short", "go test -short failed", res, scope.targetGoFiles, rep) {
		return true, nil
	}
	return false, nil
}

func handleFastGoCommandResult(source string, summary string, res CommandResult, targetFiles map[string]struct{}, rep *rule.QAReport) bool {
	if res.ExitCode == 0 {
		return false
	}
	combined := combineCommandOutput(res)
	if skip, reason := ClassifyGoCompileError(combined, targetFiles, rule.SpeedFast); skip {
		rep.SkippedSources = append(rep.SkippedSources, rule.SkippedSource{Source: source, Reason: reason})
		return false
	}
	appendCommandFailure(rep, source, summary, res)
	return true
}

func runWebAndMiscFastChecks(ctx context.Context, cfg RunConfig, scope domainScope, rep *rule.QAReport) error {
	webDir := filepath.Join(cfg.RepoRoot, "web")
	type checkCmd struct {
		enabled bool
		source  string
		dir     string
		name    string
		args    []string
		summary string
	}
	checks := []checkCmd{
		{len(scope.prettierFiles) > 0, "prettier", webDir, "npx", append([]string{"prettier", "--check"}, scope.prettierFiles...), "prettier formatting check failed"},
		{len(scope.tsFiles) > 0, "eslint", webDir, "npx", append([]string{"eslint"}, scope.tsFiles...), "eslint reported issues"},
		{len(scope.scssFiles) > 0, "stylelint", webDir, "npx", append([]string{"stylelint"}, scope.scssFiles...), "stylelint reported issues"},
		{scope.hasWebScripts, "test_scripts", webDir, "npm", []string{"run", "test:scripts"}, "web script tests failed"},
		{scope.hasProto, "buf_lint", cfg.RepoRoot, "npx", []string{"@bufbuild/buf", "lint"}, "buf lint reported issues"},
		{len(scope.miscPrettierFiles) > 0, "prettier_misc", cfg.RepoRoot, "npx", append([]string{"prettier", "--check"}, scope.miscPrettierFiles...), "misc config prettier formatting check failed"},
		{len(scope.mdFiles) > 0, "markdownlint", cfg.RepoRoot, "npx", append([]string{"markdownlint-cli2"}, scope.mdFiles...), "markdownlint reported issues"},
	}
	for _, check := range checks {
		if !check.enabled || isSourceSkipped(cfg.SkipSources, check.source, rep) {
			continue
		}
		res, err := invokeCommand(ctx, cfg, commandSpec{dir: check.dir, name: check.name, args: check.args})
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			appendCommandFailure(rep, check.source, check.summary, res)
		}
	}
	return nil
}

func invokeCommand(ctx context.Context, cfg RunConfig, spec commandSpec) (CommandResult, error) {
	res := cfg.CommandRunner(ctx, spec.dir, spec.env, spec.name, spec.args...)
	if res.Err != nil {
		return CommandResult{}, fmt.Errorf("command %s failed: %w", spec.name, res.Err)
	}
	return res, nil
}

func runCodequalityPhase(ctx context.Context, cfg RunConfig, speed rule.Speed, rep *rule.QAReport) error {
	if cfg.CodequalityCheck == nil {
		return nil
	}
	subRep, err := cfg.CodequalityCheck(ctx, speed, cfg.Filter)
	if err != nil {
		return err
	}
	if subRep == nil {
		return nil
	}
	if cfg.Filter.IncludesMust() {
		rep.MustFix = append(rep.MustFix, subRep.MustFix...)
	}
	if cfg.Filter.IncludesAdvisory() {
		rep.Advisory = append(rep.Advisory, subRep.Advisory...)
	}
	rep.SkippedSources = append(rep.SkippedSources, subRep.SkippedSources...)
	return nil
}

func runAdvisoryProviders(ctx context.Context, cfg RunConfig, phase rule.Speed, rep *rule.QAReport) error {
	if len(cfg.AdvisoryProviders) == 0 {
		return nil
	}
	req := buildAdvisoryRequest(cfg)
	for _, provider := range cfg.AdvisoryProviders {
		if !providerMatchesPhase(provider.Speed(), phase) {
			continue
		}
		items, skipped, err := provider.Evaluate(ctx, req)
		if err != nil {
			return fmt.Errorf("advisory provider %s failed: %w", provider.Name(), err)
		}
		rep.Advisory = append(rep.Advisory, items...)
		if skipped != nil {
			rep.SkippedSources = append(rep.SkippedSources, *skipped)
		}
	}
	return nil
}

func providerMatchesPhase(providerSpeed rule.Speed, phase rule.Speed) bool {
	if phase == rule.SpeedFast {
		return providerSpeed.IncludesFast()
	}
	return providerSpeed == rule.SpeedSlow
}

func buildAdvisoryRequest(cfg RunConfig) rule.AdvisoryRequest {
	return rule.AdvisoryRequest{
		RepoRoot:     cfg.RepoRoot,
		TrustedRoot:  cfg.TrustedRoot,
		ChangedFiles: slices.Clone(cfg.Diff.ChangedFiles),
		DeletedFiles: slices.Clone(cfg.Diff.DeletedFiles),
		ChangedLines: mergeChangedLines(cfg.Diff),
	}
}

func mergeChangedLines(diff gotestutil.DiffResult) map[string]map[int]struct{} {
	merged := make(map[string]map[int]struct{})
	for _, src := range []map[string]map[int]struct{}{
		diff.ChangedLines,
		diff.ChangedTestLines,
		diff.TSChangedLines,
		diff.TSChangedTestLines,
	} {
		for filePath, lines := range src {
			dest, ok := merged[filePath]
			if !ok {
				dest = make(map[int]struct{}, len(lines))
				merged[filePath] = dest
			}
			for line := range lines {
				dest[line] = struct{}{}
			}
		}
	}
	return merged
}

func appendGofmtFindings(rep *rule.QAReport, res CommandResult) {
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	added := false
	for _, line := range lines {
		file := normalizeSlashPath(strings.TrimSpace(line))
		if file == "" {
			continue
		}
		added = true
		rep.MustFix = append(rep.MustFix, rule.QAItem{
			ID:      "gofmt:" + file,
			Source:  "gofmt",
			Check:   "gofmt",
			File:    file,
			Summary: "file is not gofmt-formatted",
			Detail:  file,
		})
	}
	if !added && res.ExitCode != 0 {
		appendCommandFailure(rep, "gofmt", "gofmt check failed", res)
	}
}

func appendDeadcodeFindings(rep *rule.QAReport, res CommandResult, paths []string) {
	filtered := FilterDeadcodeOutput(res.Stdout, paths)
	if strings.TrimSpace(filtered) == "" {
		if res.ExitCode != 0 && strings.TrimSpace(res.Stdout) == "" {
			appendCommandFailure(rep, "deadcode", "deadcode check failed", res)
		}
		return
	}
	for i, line := range strings.Split(filtered, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		file, lineNum := parseDeadcodeLocation(trimmed)
		rep.MustFix = append(rep.MustFix, rule.QAItem{
			ID:      fmt.Sprintf("deadcode:%s:%d:%d", file, lineNum, i),
			Source:  "deadcode",
			Check:   "deadcode",
			File:    file,
			Line:    lineNum,
			Summary: trimmed,
			Detail:  trimmed,
		})
	}
}

func parseDeadcodeLocation(line string) (string, int) {
	match := deadcodeDiagFilePattern.FindStringSubmatch(line)
	if len(match) == 0 {
		return "", 0
	}
	lineNum, _ := strconv.Atoi(match[2])
	return normalizeSlashPath(match[1]), lineNum
}

func appendCommandFailure(rep *rule.QAReport, source string, summary string, res CommandResult) {
	detail := combineCommandOutput(res)
	if detail == "" {
		detail = fmt.Sprintf("%s exited with code %d", source, res.ExitCode)
	}
	rep.MustFix = append(rep.MustFix, rule.QAItem{
		ID:      source,
		Source:  source,
		Check:   source,
		Summary: summary,
		Detail:  detail,
	})
}

func isEmptyDiff(diff gotestutil.DiffResult) bool {
	return len(diff.ChangedFiles) == 0 &&
		len(diff.DeletedFiles) == 0 &&
		len(diff.ChangedLines) == 0 &&
		len(diff.ChangedTestLines) == 0 &&
		len(diff.TSChangedLines) == 0 &&
		len(diff.TSChangedTestLines) == 0 &&
		len(diff.DeletedGoFiles) == 0 &&
		len(diff.OtherChangedGoFiles) == 0
}
