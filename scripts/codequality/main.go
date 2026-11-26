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

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/decisioncheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/report"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/tscheck"
)

type subcmdRunRequest struct {
	flags     cliFlags
	speed     rule.Speed
	filter    rule.Filter
	repoRoot  string
	colorMode gotestutil.ColorMode
	env       cliEnv
}

type checkStagesRequest struct {
	flags    cliFlags
	speed    rule.Speed
	filter   rule.Filter
	repoRoot string
	diffRes  gotestutil.DiffResult
	progress *report.ProgressReporter
	env      cliEnv
}

type subcmdEvaluator func(ctx context.Context, req checkStagesRequest) ([]*rule.Report, *rule.QAReport, error)

type pipelineEnv struct {
	flags      cliFlags
	repoRoot   string
	diffRes    gotestutil.DiffResult
	progress   *report.ProgressReporter
	errOut     io.Writer
	simSession *similaritySession
}

func main() {
	code, err := runCLI(context.Background(), cliEnv{
		args:           os.Args[1:],
		stdout:         os.Stdout,
		stderr:         os.Stderr,
		getenv:         os.Getenv,
		embedFactory:   simcheck.TryNewDefaultEmbedder,
		deciderFactory: decisioncheck.TryNewDefaultDecider,
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
	repoRoot, err := gotestutil.ResolveRepoRoot(flags.repoRoot)
	if err != nil {
		return 2, err
	}
	if flags.subcommand == subcmdConventions {
		return runASTCheckMode(astcheck.CheckConventions, repoRoot, flags.pkgs, env.stdout)
	}
	if flags.subcommand == subcmdUnusedExports {
		return runASTCheckMode(astcheck.CheckUnusedExports, repoRoot, flags.pkgs, env.stdout)
	}

	colorMode, err := gotestutil.ParseColorMode(flags.color)
	if err != nil {
		return 2, err
	}
	speed, err := rule.ParseSpeed(flags.speed)
	if err != nil {
		return 2, err
	}
	filter, err := rule.ParseFilter(flags.filter)
	if err != nil {
		return 2, err
	}

	req := subcmdRunRequest{
		flags:     flags,
		speed:     speed,
		filter:    filter,
		repoRoot:  repoRoot,
		colorMode: colorMode,
		env:       env,
	}
	evaluate := evaluateCheckPhase
	if flags.subcommand == subcmdQA {
		evaluate = evaluateQASubcommand
	}
	return executeSubcommand(ctx, req, evaluate)
}

func runASTCheckMode(
	check func(repoRoot string, pkgs string) ([]astcheck.ConventionViolation, error),
	repoRoot string,
	pkgs string,
	stdout io.Writer,
) (int, error) {
	violations, err := check(repoRoot, pkgs)
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

func executeSubcommand(ctx context.Context, req subcmdRunRequest, evaluate subcmdEvaluator) (int, error) {
	progress := newProgressReporter(req.env)
	defer progress.Finish()

	totalSteps := 4
	if !req.speed.IncludesSlow() {
		totalSteps = 2
	}
	progress.Phase(1, totalSteps, "Collecting VCS diff...")
	diffRes, err := collectScopedDiff(ctx, req.flags, req.repoRoot)
	if err != nil {
		return 2, err
	}

	reports, qaRep, err := evaluate(ctx, checkStagesRequest{
		flags:    req.flags,
		speed:    req.speed,
		filter:   req.filter,
		repoRoot: req.repoRoot,
		diffRes:  diffRes,
		progress: progress,
		env:      req.env,
	})
	if err != nil {
		return 2, err
	}

	if err := emitOutputs(ctx, emitRequest{
		flags:     req.flags,
		baseLabel: diffRes.BaseLabel,
		reports:   reports,
		qaReport:  qaRep,
		colorMode: req.colorMode,
		env:       req.env,
	}); err != nil {
		return 2, err
	}
	if req.flags.failOnViolations && qaRep.HasMustFix() {
		return 1, nil
	}
	return 0, nil
}

func evaluateCheckPhase(ctx context.Context, req checkStagesRequest) ([]*rule.Report, *rule.QAReport, error) {
	reports, err := executeCheckStages(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return reports, report.ConvertReportsToQAReport(req.diffRes.BaseLabel, req.speed, req.filter, reports), nil
}

func newProgressReporter(env cliEnv) *report.ProgressReporter {
	errStyle := gotestutil.TermStyle{Enabled: gotestutil.ResolveColorEnabled(gotestutil.ColorAuto, env.stderr, env.getenv)}
	return report.NewProgressReporter(report.ProgressConfig{
		Writer:      env.stderr,
		Interactive: gotestutil.ResolveInteractive(env.stderr, env.getenv),
		Style:       errStyle,
	})
}

func collectScopedDiff(ctx context.Context, flags cliFlags, repoRoot string) (gotestutil.DiffResult, error) {
	diffRes, err := gotestutil.CollectChangedLines(ctx, repoRoot, gotestutil.DiffOptions{
		VCS:         flags.vcs,
		Base:        flags.base,
		Target:      flags.target,
		CurrentOnly: flags.subcommand == subcmdQA,
	})
	if err != nil {
		return gotestutil.DiffResult{}, err
	}
	diffRes = gotestutil.FilterDiffByPaths(diffRes, parseCSVList(flags.paths))
	if flags.changedSince != "" {
		diffRes, err = gotestutil.IntersectDiffChangedSince(ctx, repoRoot, diffRes, flags.changedSince, flags.target)
		if err != nil {
			return gotestutil.DiffResult{}, err
		}
	}
	return diffRes, nil
}

func executeCheckStages(ctx context.Context, req checkStagesRequest) ([]*rule.Report, error) {
	simSession := newSimilaritySession(ctx, req.flags, req.env.embedFactory, req.env.getenv)
	defer simSession.Close()
	decSession := newDecisionsSession(ctx, req.flags, req.env.deciderFactory, req.env.getenv)
	defer decSession.Close()

	penv := pipelineEnv{
		flags:      req.flags,
		repoRoot:   req.repoRoot,
		diffRes:    req.diffRes,
		progress:   req.progress,
		errOut:     req.env.stderr,
		simSession: simSession,
	}
	var reports []*rule.Report
	if shouldRunGo(req.flags, req.speed, req.filter, req.diffRes) {
		goRep, err := executeGoPipeline(ctx, penv, req.speed, req.filter)
		if err != nil {
			return nil, err
		}
		reports = append(reports, goRep)
	}
	if shouldRunTS(req.flags, req.speed, req.filter, req.diffRes) {
		tsRep, err := executeTSPipeline(ctx, penv, req.filter)
		if err != nil {
			return nil, err
		}
		reports = append(reports, tsRep)
	}
	if req.speed.IncludesFast() && req.filter.IncludesAdvisory() {
		decRep, err := decSession.evaluate(ctx, req.repoRoot, req.diffRes)
		if err != nil {
			return nil, err
		}
		if decRep != nil {
			reports = append(reports, decRep)
		}
	}
	return reports, nil
}

func shouldRunGo(flags cliFlags, speed rule.Speed, filter rule.Filter, diffRes gotestutil.DiffResult) bool {
	if flags.lang != "go" && flags.lang != "all" {
		return false
	}
	runMust := filter.IncludesMust()
	runSlowAdv := speed.IncludesSlow() && filter.IncludesAdvisory()
	if !runMust && !runSlowAdv {
		return false
	}
	if flags.lang == "go" {
		return true
	}
	return len(diffRes.ChangedLines) > 0 || len(diffRes.ChangedTestLines) > 0 || (speed.IncludesSlow() && (flags.coverprofile != "" || flags.full))
}

func shouldRunTS(flags cliFlags, speed rule.Speed, filter rule.Filter, diffRes gotestutil.DiffResult) bool {
	if (flags.lang != "ts" && flags.lang != "all") || !speed.IncludesSlow() {
		return false
	}
	if !filter.IncludesAdvisory() && !(filter.IncludesMust() && flags.tsFailOnViolations) {
		return false
	}
	if flags.lang == "ts" {
		return true
	}
	return len(diffRes.TSChangedLines) > 0 || len(diffRes.TSChangedTestLines) > 0 || flags.tsCoverage != "" || flags.full
}

func executeGoPipeline(ctx context.Context, penv pipelineEnv, speed rule.Speed, filter rule.Filter) (*rule.Report, error) {
	rep := &rule.Report{
		Language:         "Go",
		BaseLabel:        penv.diffRes.BaseLabel,
		FastMode:         !speed.IncludesSlow(),
		ProdFilesChanged: len(penv.diffRes.ChangedLines),
		TestFilesChanged: len(penv.diffRes.ChangedTestLines),
	}
	runMust := filter.IncludesMust()
	runDynamic := speed.IncludesSlow() && filter.IncludesAdvisory()
	if !runMust && !runDynamic {
		return rep, nil
	}

	var astRes *astcheck.Result
	var err error
	if runMust {
		astRes, err = runGoASTAndConventions(ctx, penv, speed, rep)
	} else {
		astRes, err = loadGoASTForDynamic(ctx, penv, rep)
	}
	if err != nil {
		return nil, err
	}
	if runDynamic {
		if err := runGoSlowAdvisory(ctx, penv, astRes, rep); err != nil {
			return nil, err
		}
	}
	if runMust {
		appendUnusedDirectiveErrors(astRes, rep)
	}
	finalizeReportOrdering(rep)
	return rep, nil
}

func loadGoASTForDynamic(ctx context.Context, penv pipelineEnv, rep *rule.Report) (*astcheck.Result, error) {
	penv.progress.Phase(2, 4, "Evaluating Go AST structural and naming rules...")
	astRes, err := runASTStage(ctx, penv.repoRoot, penv.diffRes, rule.SpeedFast)
	if err != nil {
		return nil, err
	}
	rep.FuncsEvaluated = astRes.FuncsEvaluated
	return astRes, nil
}

func runGoASTAndConventions(ctx context.Context, penv pipelineEnv, speed rule.Speed, rep *rule.Report) (*astcheck.Result, error) {
	totalSteps := 4
	if !speed.IncludesSlow() {
		totalSteps = 2
	}
	penv.progress.Phase(2, totalSteps, "Evaluating Go AST structural and naming rules...")
	astRes, err := runASTStage(ctx, penv.repoRoot, penv.diffRes, speed)
	if err != nil {
		return nil, err
	}
	rep.FuncsEvaluated = astRes.FuncsEvaluated
	rep.Findings = append(rep.Findings, astRes.Findings...)
	rep.Suppressed = append(rep.Suppressed, astRes.Suppressed...)
	rep.DirectiveErrors = append(rep.DirectiveErrors, astRes.DirectiveErrors...)
	if speed.IncludesFast() {
		return astRes, appendGoFileConventions(penv.repoRoot, penv.diffRes, rep)
	}
	return astRes, nil
}

func appendGoFileConventions(repoRoot string, diffRes gotestutil.DiffResult, rep *rule.Report) error {
	convViolations, err := astcheck.CheckFileConventions(repoRoot, changedGoFiles(diffRes))
	if err != nil {
		return err
	}
	for _, v := range convViolations {
		rep.Findings = append(rep.Findings, rule.Finding{
			Rule:   rule.RuleID(v.Rule),
			Kind:   rule.ViolationHardLimit,
			File:   v.File,
			Line:   v.Line,
			Detail: v.Message,
		})
	}
	return nil
}

func runGoSlowAdvisory(ctx context.Context, penv pipelineEnv, astRes *astcheck.Result, rep *rule.Report) error {
	modulePath, err := gotestutil.ReadModulePath(filepath.Join(penv.repoRoot, "go.mod"))
	if err != nil {
		return err
	}
	if err := runDynamicStages(ctx, dynamicStageInput{
		flags:      penv.flags,
		repoRoot:   penv.repoRoot,
		modulePath: modulePath,
		diffRes:    penv.diffRes,
		astRes:     astRes,
		progress:   penv.progress,
		errOut:     penv.errOut,
	}, rep); err != nil {
		return err
	}
	simSum, err := penv.simSession.evaluateGo(ctx, penv.repoRoot, penv.diffRes)
	if err != nil {
		return err
	}
	rep.Similarity = simSum
	return nil
}

func changedGoFiles(diffRes gotestutil.DiffResult) []string {
	files := make([]string, 0, len(diffRes.ChangedLines)+len(diffRes.ChangedTestLines))
	files = append(files, mapKeys(diffRes.ChangedLines)...)
	files = append(files, mapKeys(diffRes.ChangedTestLines)...)
	slices.Sort(files)
	return files
}

func executeTSPipeline(ctx context.Context, penv pipelineEnv, filter rule.Filter) (*rule.Report, error) {
	var rep *rule.Report
	if filter.IncludesAdvisory() || penv.flags.tsFailOnViolations {
		var err error
		rep, err = tscheck.Evaluate(ctx, tscheck.Options{
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
	} else {
		rep = &rule.Report{
			Language:         "TypeScript",
			BaseLabel:        penv.diffRes.BaseLabel,
			Advisory:         true,
			ProdFilesChanged: len(penv.diffRes.TSChangedLines),
			TestFilesChanged: len(penv.diffRes.TSChangedTestLines),
		}
	}
	if filter.IncludesAdvisory() {
		simSum, err := penv.simSession.evaluateTS(ctx, penv.repoRoot, penv.diffRes)
		if err != nil {
			return nil, err
		}
		rep.Similarity = simSum
	}
	return rep, nil
}

func runASTStage(ctx context.Context, repoRoot string, diffRes gotestutil.DiffResult, speed rule.Speed) (*astcheck.Result, error) {
	skipCloneScan := !speed.IncludesSlow()
	var corpus []string
	if !skipCloneScan && len(diffRes.ChangedLines) > 0 {
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
		SkipCloneScan:          skipCloneScan,
		ReadTargetFile: func(relPath string) ([]byte, error) {
			return os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
		},
		ReadBaseFile: func(relPath string) ([]byte, bool, error) {
			return gotestutil.ReadBaseFile(ctx, repoRoot, diffRes, relPath)
		},
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
			IncludeDynamic: false,
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
