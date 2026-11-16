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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/covcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/mutcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/overlay"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/report"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/tscheck"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type pipelineEnv struct {
	flags      cliFlags
	repoRoot   string
	diffRes    gotestutil.DiffResult
	progress   *report.ProgressReporter
	errOut     io.Writer
	simSession *similaritySession
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

func main() {
	code, err := runCLI(context.Background(), cliEnv{
		args:         os.Args[1:],
		stdout:       os.Stdout,
		stderr:       os.Stderr,
		getenv:       os.Getenv,
		embedFactory: simcheck.TryNewDefaultEmbedder,
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
	if flags.conventions {
		return runConventionsMode(repoRoot, flags.pkgs, env.stdout)
	}

	reports, baseLabel, err := executePipeline(ctx, flags, repoRoot, env)
	if err != nil {
		return 2, err
	}
	if err := emitOutputs(ctx, emitRequest{
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

func runConventionsMode(repoRoot string, pkgs string, stdout io.Writer) (int, error) {
	violations, err := astcheck.CheckConventions(repoRoot, pkgs)
	if err != nil {
		return 2, err
	}
	for _, v := range violations {
		fmt.Fprintf(stdout, "%s:%d:%d: %s (%s)\n", v.File, v.Line, v.Column, v.Message, v.Rule)
	}
	if len(violations) > 0 {
		return 1, nil
	}
	return 0, nil
}

func executePipeline(ctx context.Context, flags cliFlags, repoRoot string, env cliEnv) ([]*rule.Report, string, error) {
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

	simSession := newSimilaritySession(ctx, flags, env.embedFactory, env.getenv)
	defer simSession.Close()

	penv := pipelineEnv{
		flags:      flags,
		repoRoot:   repoRoot,
		diffRes:    diffRes,
		progress:   progress,
		errOut:     env.stderr,
		simSession: simSession,
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
	simSum, err := penv.simSession.evaluateGo(ctx, penv.repoRoot, penv.diffRes)
	if err != nil {
		return nil, err
	}
	rep.Similarity = simSum
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
	rep, err := tscheck.Evaluate(ctx, tscheck.Options{
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
	if err != nil {
		return nil, err
	}
	simSum, err := penv.simSession.evaluateTS(ctx, penv.repoRoot, penv.diffRes)
	if err != nil {
		return nil, err
	}
	rep.Similarity = simSum
	return rep, nil
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

func runCoverageStage(ctx context.Context, in dynamicStageInput, workDir string, ensureOverlay func() (*overlay.Snapshot, error)) (*covcheck.Result, error) {
	in.progress.Phase(2, 4, "Evaluating Go differential line coverage...")
	return covcheck.Evaluate(ctx, covcheck.Options{
		RepoRoot:               in.repoRoot,
		ModulePath:             in.modulePath,
		WorkDir:                filepath.Join(workDir, "cov"),
		Full:                   in.flags.full,
		PrecomputedProfilePath: in.flags.coverprofile,
		Thresholds:             rule.DefaultThresholds(),
		ChangedProdLinesByFile: in.astRes.DynamicChangedLines,
		ASTResult:              in.astRes,
		ReadTargetFile: func(relPath string) ([]byte, error) {
			return os.ReadFile(filepath.Join(in.repoRoot, filepath.FromSlash(relPath)))
		},
		EnsureBaseOverlay: ensureOverlay,
		ErrOut:            in.errOut,
	})
}

func runMutationStage(ctx context.Context, in dynamicStageInput, workDir string, covRes *covcheck.Result, ensureOverlay func() (*overlay.Snapshot, error)) (*mutcheck.Result, error) {
	return mutcheck.Evaluate(ctx, mutcheck.Options{
		RepoRoot:               in.repoRoot,
		WorkDir:                filepath.Join(workDir, "mut"),
		Thresholds:             rule.DefaultThresholds(),
		ChangedProdLinesByFile: in.astRes.DynamicChangedLines,
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
