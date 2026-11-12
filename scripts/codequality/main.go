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
	"cmp"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/covcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/mutcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/overlay"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/report"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/tscheck"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const (
	defaultMaxMutants     = 80
	defaultOverallTimeout = 3 * time.Minute
	defaultMaxAnnotations = 20
)

type cliFlags struct {
	repoRoot           string
	lang               string
	fast               bool
	vcs                string
	base               string
	target             string
	coverprofile       string
	tsCoverage         string
	full               bool
	workers            int
	maxMutants         int
	timeout            time.Duration
	format             string
	color              string
	summaryFile        string
	maxAnnotations     int
	failOnViolations   bool
	tsFailOnViolations bool
}

type cliEnv struct {
	args   []string
	stdout io.Writer
	stderr io.Writer
	getenv func(string) string
}

type pipelineEnv struct {
	flags    cliFlags
	repoRoot string
	diffRes  gotestutil.DiffResult
	progress *report.ProgressReporter
	errOut   io.Writer
}

func main() {
	code, err := runCLI(context.Background(), cliEnv{
		args:   os.Args[1:],
		stdout: os.Stdout,
		stderr: os.Stderr,
		getenv: os.Getenv,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "codequality: %v\n", err)
		os.Exit(2)
	}
	os.Exit(code)
}

func runCLI(ctx context.Context, env cliEnv) (int, error) {
	flags, err := parseFlags(env.args, env.stderr, env.getenv)
	if err != nil {
		return 2, err
	}
	colorMode, err := gotestutil.ParseColorMode(flags.color)
	if err != nil {
		return 2, err
	}
	repoRoot, err := gotestutil.ResolveRepoRoot(flags.repoRoot)
	if err != nil {
		return 2, err
	}

	reports, baseLabel, err := executePipeline(ctx, flags, repoRoot, env)
	if err != nil {
		return 2, err
	}
	if err := emitOutputs(emitRequest{
		flags:     flags,
		baseLabel: baseLabel,
		reports:   reports,
		colorMode: colorMode,
		env:       env,
	}); err != nil {
		return 2, err
	}
	if hasAnyBlockingReport(flags, reports) {
		return 1, nil
	}
	return 0, nil
}

func hasAnyBlockingReport(flags cliFlags, reports []*rule.Report) bool {
	if !flags.failOnViolations {
		return false
	}
	for _, rep := range reports {
		if rep.HasBlockingIssues() {
			return true
		}
	}
	return false
}

func parseFlags(args []string, errOut io.Writer, getenv func(string) string) (cliFlags, error) {
	var f cliFlags
	fs := flag.NewFlagSet("codequality", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.StringVar(&f.repoRoot, "repo-root", "", "repository root path (auto-detected by default)")
	fs.StringVar(&f.lang, "lang", "all", "target language(s): all, go, or ts")
	fs.BoolVar(&f.fast, "fast", false, "run fast Stage-1 Go AST structural and naming rules only")
	fs.StringVar(&f.vcs, "vcs", "auto", "version control system: auto, jj, or git")
	fs.StringVar(&f.base, "base", "", "base revision for differential checks")
	fs.StringVar(&f.target, "target", "", "target revision for differential checks")
	fs.StringVar(&f.coverprofile, "coverprofile", "", "optional precomputed Go test coverprofile path")
	fs.StringVar(&f.tsCoverage, "ts-coverage", "", "optional precomputed Istanbul coverage-final.json path")
	fs.BoolVar(&f.full, "full", false, "run full-repository coverage in addition to patch checks")
	fs.IntVar(&f.workers, "workers", 0, "parallel Go mutant worker count (0 = logical CPU core count)")
	fs.IntVar(&f.maxMutants, "max-mutants", defaultMaxMutants, "maximum reached mutants to evaluate per language (0 = unlimited)")
	fs.DurationVar(&f.timeout, "timeout", defaultOverallTimeout, "overall timeout for mutation evaluation per language")
	fs.StringVar(&f.format, "format", "text", "report format: text or github")
	fs.StringVar(&f.color, "color", "auto", "ANSI color mode: auto, always, or never")
	fs.StringVar(&f.summaryFile, "summary-file", getenv("GITHUB_STEP_SUMMARY"), "GitHub step summary file path")
	fs.IntVar(&f.maxAnnotations, "max-annotations", defaultMaxAnnotations, "maximum GitHub workflow annotations per language")
	fs.BoolVar(&f.failOnViolations, "fail-on-violations", true, "exit with code 1 when blocking violations are found")
	fs.BoolVar(&f.tsFailOnViolations, "ts-fail-on-violations", false, "treat TypeScript findings as blocking errors instead of advisory warnings")

	if err := fs.Parse(args); err != nil {
		return cliFlags{}, err
	}
	if err := validateEnumFlags(f); err != nil {
		return cliFlags{}, err
	}
	return f, nil
}

func validateEnumFlags(f cliFlags) error {
	if f.lang != "all" && f.lang != "go" && f.lang != "ts" {
		return fmt.Errorf("invalid --lang %q (want all, go, or ts)", f.lang)
	}
	if f.format != "text" && f.format != "github" {
		return fmt.Errorf("invalid --format %q (want text or github)", f.format)
	}
	return nil
}

func executePipeline(
	ctx context.Context,
	flags cliFlags,
	repoRoot string,
	env cliEnv,
) ([]*rule.Report, string, error) {
	errStyle := gotestutil.TermStyle{Enabled: gotestutil.ResolveColorEnabled(gotestutil.ColorAuto, env.stderr, env.getenv)}
	progress := report.NewProgressReporter(report.ProgressConfig{
		Writer:      env.stderr,
		Interactive: gotestutil.ResolveInteractive(env.stderr, env.getenv),
		Style:       errStyle,
	})
	defer progress.Finish()

	totalSteps := 4
	if flags.fast {
		totalSteps = 2
	}
	progress.Phase(1, totalSteps, "Collecting VCS diff...")
	diffRes, err := gotestutil.CollectChangedLines(ctx, repoRoot, gotestutil.DiffOptions{
		VCS:    flags.vcs,
		Base:   flags.base,
		Target: flags.target,
	})
	if err != nil {
		return nil, "", err
	}

	penv := pipelineEnv{
		flags:    flags,
		repoRoot: repoRoot,
		diffRes:  diffRes,
		progress: progress,
		errOut:   env.stderr,
	}
	var reports []*rule.Report
	if shouldRunGo(flags, diffRes) {
		goRep, err := executeGoPipeline(ctx, penv)
		if err != nil {
			return nil, "", err
		}
		reports = append(reports, goRep)
	}
	if shouldRunTS(flags, diffRes) {
		tsRep, err := executeTSPipeline(ctx, penv)
		if err != nil {
			return nil, "", err
		}
		reports = append(reports, tsRep)
	}
	return reports, diffRes.BaseLabel, nil
}

func shouldRunGo(f cliFlags, d gotestutil.DiffResult) bool {
	if f.lang == "go" {
		return true
	}
	if f.lang != "all" {
		return false
	}
	return f.fast || len(d.ChangedLines) > 0 || len(d.ChangedTestLines) > 0 || f.coverprofile != "" || f.full
}

func shouldRunTS(f cliFlags, d gotestutil.DiffResult) bool {
	if f.fast {
		return false
	}
	if f.lang == "ts" {
		return true
	}
	if f.lang != "all" {
		return false
	}
	return len(d.TSChangedLines) > 0 || len(d.TSChangedTestLines) > 0 || f.tsCoverage != "" || f.full
}

func executeGoPipeline(ctx context.Context, penv pipelineEnv) (*rule.Report, error) {
	modulePath, err := gotestutil.ReadModulePath(filepath.Join(penv.repoRoot, "go.mod"))
	if err != nil {
		return nil, err
	}
	totalSteps := 4
	if penv.flags.fast {
		totalSteps = 2
	}
	penv.progress.Phase(2, totalSteps, "Evaluating Go AST structural and naming rules...")
	astRes, err := runASTStage(ctx, penv.repoRoot, penv.diffRes)
	if err != nil {
		return nil, err
	}

	rep := newInitialGoReport(penv.diffRes, penv.flags.fast, astRes)
	if !penv.flags.fast {
		if err := runDynamicStages(ctx, dynamicStageInput{
			flags:      penv.flags,
			repoRoot:   penv.repoRoot,
			modulePath: modulePath,
			diffRes:    penv.diffRes,
			astRes:     astRes,
			progress:   penv.progress,
			errOut:     penv.errOut,
		}, rep); err != nil {
			return nil, err
		}
	}
	appendUnusedDirectiveErrors(astRes, rep)
	finalizeReportOrdering(rep)
	return rep, nil
}

func executeTSPipeline(ctx context.Context, penv pipelineEnv) (*rule.Report, error) {
	return tscheck.Evaluate(ctx, tscheck.Options{
		RepoRoot:               penv.repoRoot,
		BaseLabel:              penv.diffRes.BaseLabel,
		TSCoveragePath:         penv.flags.tsCoverage,
		Full:                   penv.flags.full,
		FailOnViolations:       penv.flags.tsFailOnViolations,
		ChangedLinesByFile:     penv.diffRes.TSChangedLines,
		ChangedTestLinesByFile: penv.diffRes.TSChangedTestLines,
		MaxMutants:             penv.flags.maxMutants,
		Timeout:                penv.flags.timeout,
		LogWriter:              penv.errOut,
		Progress:               penv.progress,
	})
}

func newInitialGoReport(diffRes gotestutil.DiffResult, fast bool, astRes *astcheck.Result) *rule.Report {
	return &rule.Report{
		Language:         "Go",
		BaseLabel:        diffRes.BaseLabel,
		FastMode:         fast,
		ProdFilesChanged: len(diffRes.ChangedLines),
		TestFilesChanged: len(diffRes.ChangedTestLines),
		FuncsEvaluated:   astRes.FuncsEvaluated,
		Findings:         append([]rule.Finding(nil), astRes.Findings...),
		Suppressed:       append([]rule.SuppressedItem(nil), astRes.Suppressed...),
		DirectiveErrors:  append([]rule.DirectiveError(nil), astRes.DirectiveErrors...),
	}
}

func runASTStage(ctx context.Context, repoRoot string, diffRes gotestutil.DiffResult) (*astcheck.Result, error) {
	var corpus []string
	if len(diffRes.ChangedLines) > 0 {
		var err error
		corpus, err = astcheck.DiscoverProdGoFiles(repoRoot)
		if err != nil {
			return nil, err
		}
	}
	return astcheck.Evaluate(astcheck.Options{
		Thresholds:             rule.DefaultThresholds(),
		ChangedProdLinesByFile: diffRes.ChangedLines,
		ChangedTestLinesByFile: diffRes.ChangedTestLines,
		DeletedGoFiles:         diffRes.DeletedGoFiles,
		CloneCorpusFiles:       corpus,
		ReadTargetFile: func(relPath string) ([]byte, error) {
			return os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
		},
		ReadBaseFile: func(relPath string) ([]byte, bool, error) {
			return gotestutil.ReadBaseFile(ctx, repoRoot, diffRes, relPath)
		},
	})
}

type dynamicStageInput struct {
	flags      cliFlags
	repoRoot   string
	modulePath string
	diffRes    gotestutil.DiffResult
	astRes     *astcheck.Result
	progress   *report.ProgressReporter
	errOut     io.Writer
}

func runDynamicStages(ctx context.Context, in dynamicStageInput, rep *rule.Report) error {
	workDir, err := os.MkdirTemp("", "khi-codequality-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary workspace: %w", err)
	}
	defer os.RemoveAll(workDir)

	ensureOverlay := newLazyOverlayBuilder(ctx, in, workDir)
	covRes, err := runCoverageStage(ctx, in, workDir, ensureOverlay)
	if err != nil {
		return err
	}
	rep.Coverage = covRes.Summary
	rep.Findings = append(rep.Findings, covRes.Findings...)
	rep.Suppressed = append(rep.Suppressed, covRes.Suppressed...)

	mutRes, err := runMutationStage(ctx, in, workDir, covRes, ensureOverlay)
	if err != nil {
		return err
	}
	rep.Mutation = mutRes.Summary
	rep.Findings = append(rep.Findings, mutRes.Findings...)
	rep.Suppressed = append(rep.Suppressed, mutRes.Suppressed...)
	return nil
}

func newLazyOverlayBuilder(ctx context.Context, in dynamicStageInput, workDir string) func() (*overlay.Snapshot, error) {
	return sync.OnceValues(func() (*overlay.Snapshot, error) {
		return overlay.Build(overlay.Config{
			RepoRoot:            in.repoRoot,
			WorkDir:             filepath.Join(workDir, "base_overlay"),
			ProdFiles:           mapKeys(in.diffRes.ChangedLines),
			TestFiles:           mapKeys(in.diffRes.ChangedTestLines),
			DeletedGoFiles:      in.diffRes.DeletedGoFiles,
			OtherChangedGoFiles: in.diffRes.OtherChangedGoFiles,
			ReadBaseFile: func(relPath string) ([]byte, bool, error) {
				return gotestutil.ReadBaseFile(ctx, in.repoRoot, in.diffRes, relPath)
			},
		})
	})
}

func runCoverageStage(
	ctx context.Context,
	in dynamicStageInput,
	workDir string,
	ensureOverlay func() (*overlay.Snapshot, error),
) (*covcheck.Result, error) {
	in.progress.Phase(2, 4, "Evaluating Go differential line coverage...")
	return covcheck.Evaluate(ctx, covcheck.Options{
		RepoRoot:               in.repoRoot,
		ModulePath:             in.modulePath,
		WorkDir:                filepath.Join(workDir, "cov"),
		Full:                   in.flags.full,
		PrecomputedProfilePath: in.flags.coverprofile,
		Thresholds:             rule.DefaultThresholds(),
		ChangedProdLinesByFile: in.diffRes.ChangedLines,
		ASTResult:              in.astRes,
		ReadTargetFile: func(relPath string) ([]byte, error) {
			return os.ReadFile(filepath.Join(in.repoRoot, filepath.FromSlash(relPath)))
		},
		EnsureBaseOverlay: ensureOverlay,
		ErrOut:            in.errOut,
	})
}

func runMutationStage(
	ctx context.Context,
	in dynamicStageInput,
	workDir string,
	covRes *covcheck.Result,
	ensureOverlay func() (*overlay.Snapshot, error),
) (*mutcheck.Result, error) {
	return mutcheck.Evaluate(ctx, mutcheck.Options{
		RepoRoot:               in.repoRoot,
		WorkDir:                filepath.Join(workDir, "mut"),
		Thresholds:             rule.DefaultThresholds(),
		ChangedProdLinesByFile: in.diffRes.ChangedLines,
		ChangedTestLinesByFile: in.diffRes.ChangedTestLines,
		CoverBlocksByFile:      covRes.CoverBlocksByFile,
		HasCoverProfile:        covRes.HasCoverProfile,
		ASTResult:              in.astRes,
		EnsureBaseOverlay:      ensureOverlay,
		Workers:                in.flags.workers,
		MaxMutants:             in.flags.maxMutants,
		Timeout:                in.flags.timeout,
		LogWriter:              in.errOut,
		Progress:               in.progress,
	})
}

func appendUnusedDirectiveErrors(astRes *astcheck.Result, rep *rule.Report) {
	files := make([]string, 0, len(astRes.Files))
	for f := range astRes.Files {
		files = append(files, f)
	}
	slices.Sort(files)
	for _, relPath := range files {
		fm := astRes.Files[relPath]
		unused := rule.CollectUnusedDirectives(fm.Directives, rule.UnusedScope{
			File:           relPath,
			IncludeDynamic: !rep.FastMode,
			EvaluatedLines: astRes.EvaluatedLines[relPath],
		})
		rep.DirectiveErrors = append(rep.DirectiveErrors, unused...)
	}
}

func finalizeReportOrdering(rep *rule.Report) {
	slices.SortFunc(rep.Findings, func(a, b rule.Finding) int {
		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(string(a.Rule), string(b.Rule)),
		)
	})
	for i := range rep.Findings {
		rep.Findings[i].GlobalIndex = i + 1
	}
}

func mapKeys(m map[string]map[int]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
