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
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type cliConfig struct {
	repoRoot            string
	lang                string
	vcs                 string
	base                string
	target              string
	coverProfile        string
	tsCoverage          string
	full                bool
	checkRedundantTests bool
	workers             int
	maxMutants          int
	timeout             time.Duration
	format              string
	color               string
	summaryFile         string
	maxAnnotations      int
	minUncoveredLines   int
	failUnder           float64
	failUnderCoverage   float64
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "mutation-diff: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}
	if cfg.lang != "all" && cfg.lang != "go" && cfg.lang != "ts" {
		return fmt.Errorf("invalid --lang %q (want all, go, or ts)", cfg.lang)
	}
	if cfg.format != "text" && cfg.format != "github" {
		return fmt.Errorf("invalid --format %q (want text or github)", cfg.format)
	}
	colorMode, err := gotestutil.ParseColorMode(cfg.color)
	if err != nil {
		return err
	}

	repoRoot, err := gotestutil.ResolveRepoRoot(cfg.repoRoot)
	if err != nil {
		return err
	}

	errStyle := gotestutil.TermStyle{
		Enabled: gotestutil.ResolveColorEnabled(colorMode, stderr, os.Getenv),
	}
	interactiveErr := gotestutil.ResolveInteractive(stderr, os.Getenv)
	progress := NewProgressReporter(stderr, interactiveErr, errStyle)
	defer progress.Finish()

	progress.Phase(1, 4, "Collecting VCS diff...")
	diffRes, err := gotestutil.CollectChangedLines(ctx, repoRoot, gotestutil.DiffOptions{
		VCS:    cfg.vcs,
		Base:   cfg.base,
		Target: cfg.target,
	})
	if err != nil {
		return err
	}

	runGo := cfg.lang == "go" || (cfg.lang == "all" && (len(diffRes.ChangedLines) > 0 || cfg.coverProfile != "" || cfg.full))
	runTS := cfg.lang == "ts" || (cfg.lang == "all" && (len(diffRes.TSChangedLines) > 0 || cfg.tsCoverage != "" || cfg.full))

	var reports []*MutationReport
	if runGo {
		goReport, err := executeGoUnified(ctx, repoRoot, cfg, diffRes, stderr, progress)
		if err != nil {
			return err
		}
		reports = append(reports, goReport)
	}
	if runTS {
		tsReport, err := executeTSUnified(ctx, repoRoot, cfg, diffRes, stderr, progress)
		if err != nil {
			return err
		}
		reports = append(reports, tsReport)
	}
	progress.Finish()

	if err := emitReports(cfg, colorMode, diffRes.BaseLabel, reports, stdout); err != nil {
		return err
	}

	for _, rep := range reports {
		if cfg.failUnderCoverage > 0 && rep.Coverage != nil && rep.Coverage.PatchExecutableLines > 0 && rep.Coverage.PatchPercent() < cfg.failUnderCoverage {
			return fmt.Errorf("%s patch line coverage %.1f%% is below required threshold %.1f%%",
				rep.Language.DisplayName(), rep.Coverage.PatchPercent(), cfg.failUnderCoverage)
		}
		if cfg.failUnder > 0 && rep.Tested() > 0 && rep.ScorePercent() < cfg.failUnder {
			return fmt.Errorf("%s differential mutation score %.1f%% is below required threshold %.1f%%",
				rep.Language.DisplayName(), rep.ScorePercent(), cfg.failUnder)
		}
	}
	return nil
}

func executeGoUnified(
	ctx context.Context,
	repoRoot string,
	cfg cliConfig,
	diffRes gotestutil.DiffResult,
	stderr io.Writer,
	progress *ProgressReporter,
) (*MutationReport, error) {
	modulePath, err := gotestutil.ReadModulePath(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return nil, err
	}

	changedFiles := sortedChangedFiles(diffRes.ChangedLines)
	includeOverall := cfg.coverProfile != "" || cfg.full
	coverBlocks := make(map[string][]gotestutil.CoverBlock)
	hasCoverProfile := false

	if cfg.coverProfile != "" {
		if info, err := os.Stat(cfg.coverProfile); err == nil && !info.IsDir() {
			progress.Phase(2, 4, fmt.Sprintf("Loading Go coverprofile %s...", filepath.Base(cfg.coverProfile)))
			coverBlocks, err = gotestutil.LoadCoverProfileIfPresent(cfg.coverProfile, modulePath)
			if err != nil {
				return nil, err
			}
			hasCoverProfile = true
		}
	} else {
		covPlan := PlanCoverageExecution(repoRoot, changedFiles, cfg.full)
		if !covPlan.Skip {
			progress.Phase(2, 4, fmt.Sprintf("Measuring Go line coverage across %d package(s)...", len(covPlan.Packages)))
			tmpCovDir, err := os.MkdirTemp("", "khi-mutation-cov-*")
			if err != nil {
				return nil, fmt.Errorf("failed to create temporary coverage directory: %w", err)
			}
			defer os.RemoveAll(tmpCovDir)

			tmpProfile := filepath.Join(tmpCovDir, "cover.out")
			if err := RunGoTestCoverage(ctx, repoRoot, covPlan, tmpProfile, stderr); err != nil {
				return nil, err
			}
			coverBlocks, err = gotestutil.LoadCoverProfileIfPresent(tmpProfile, modulePath)
			if err != nil {
				return nil, err
			}
			hasCoverProfile = true // NOMUTANTS(perf): baseline reachability marks uncovered lines identically; pre-filtering avoids compiling unexecuted packages.
		}
	}

	readSource := func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
	}
	covReport, err := AnalyzeCoverage(diffRes.BaseLabel, coverBlocks, diffRes.ChangedLines, readSource, includeOverall)
	if err != nil {
		return nil, err
	}

	report, err := ExecuteMutationRun(ctx, &RunnerOptions{
		RepoRoot:               repoRoot,
		BaseLabel:              diffRes.BaseLabel,
		ChangedLinesByFile:     diffRes.ChangedLines,
		ChangedTestLinesByFile: diffRes.ChangedTestLines,
		CoverBlocksByFile:      coverBlocks,
		HasCoverProfile:        hasCoverProfile,
		CheckRedundantTests:    cfg.checkRedundantTests,
		Workers:                cfg.workers,
		MaxMutants:             cfg.maxMutants,
		Timeout:                cfg.timeout,
		LogWriter:              stderr,
		Progress:               progress,
	})
	if err != nil {
		return nil, err
	}
	AttachLineCoverage(report, covReport)
	return report, nil
}

func executeTSUnified(
	ctx context.Context,
	repoRoot string,
	cfg cliConfig,
	diffRes gotestutil.DiffResult,
	stderr io.Writer,
	progress *ProgressReporter,
) (*MutationReport, error) {
	changedTSFiles := sortedTSChangedFiles(diffRes.TSChangedLines)
	includeOverall := cfg.tsCoverage != "" || cfg.full
	covByFile := make(map[string]gotestutil.IstanbulFileCoverage)
	hasTSCoverage := false

	if cfg.tsCoverage != "" {
		if info, err := os.Stat(cfg.tsCoverage); err == nil && !info.IsDir() {
			progress.Phase(2, 4, fmt.Sprintf("Loading TypeScript coverage %s...", filepath.Base(cfg.tsCoverage)))
			var err error
			covByFile, err = gotestutil.LoadTSCoverageIfPresent(cfg.tsCoverage, repoRoot)
			if err != nil {
				return nil, err
			}
			hasTSCoverage = true
		}
	} else {
		tsCovPlan, err := gotestutil.PlanTSTestExecution(repoRoot, changedTSFiles, cfg.full)
		if err != nil {
			return nil, err
		}
		if !tsCovPlan.SkipKarma {
			if cfg.full {
				progress.Phase(2, 4, "Measuring full TypeScript line coverage in Karma...")
			} else {
				progress.Phase(2, 4, fmt.Sprintf("Measuring TypeScript line coverage across %d spec file(s)...", len(tsCovPlan.IncludeSpecs)))
			}
			tmpCovDir, err := os.MkdirTemp("", "khi-ts-cov-*")
			if err != nil {
				return nil, fmt.Errorf("failed to create temporary TypeScript coverage directory: %w", err)
			}
			defer os.RemoveAll(tmpCovDir)

			if err := RunTSTestCoverage(ctx, repoRoot, tsCovPlan, tmpCovDir, stderr); err != nil {
				return nil, err
			}
			covByFile, err = gotestutil.LoadTSCoverageIfPresent(filepath.Join(tmpCovDir, "coverage-final.json"), repoRoot)
			if err != nil {
				return nil, err
			}
			hasTSCoverage = true
		}
	}

	if len(changedTSFiles) > 0 {
		if _, err := os.Stat(filepath.Join(repoRoot, "web", "package.json")); err == nil {
			if err := SupplementMissingTSCoverage(ctx, repoRoot, changedTSFiles, covByFile); err != nil {
				return nil, err
			}
			if len(covByFile) > 0 {
				hasTSCoverage = true
			}
		}
	}

	readSource := func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
	}
	covReport, err := AnalyzeTSCoverage(diffRes.BaseLabel, covByFile, diffRes.TSChangedLines, readSource, includeOverall)
	if err != nil {
		return nil, err
	}

	report, err := ExecuteTSMutationRun(ctx, &TSRunnerOptions{
		RepoRoot:               repoRoot,
		BaseLabel:              diffRes.BaseLabel,
		ChangedLinesByFile:     diffRes.TSChangedLines,
		ChangedTestLinesByFile: diffRes.TSChangedTestLines,
		TSCoverageByFile:       covByFile,
		HasTSCoverage:          hasTSCoverage,
		CheckRedundantTests:    cfg.checkRedundantTests,
		MaxMutants:             cfg.maxMutants,
		Timeout:                cfg.timeout,
		LogWriter:              stderr,
		Progress:               progress,
	})
	if err != nil {
		return nil, err
	}
	AttachLineCoverage(report, covReport)
	return report, nil
}

func emitReports(
	cfg cliConfig,
	colorMode gotestutil.ColorMode,
	baseLabel string,
	reports []*MutationReport,
	stdout io.Writer,
) error {
	switch cfg.format {
	case "text":
		style := gotestutil.TermStyle{
			Enabled: gotestutil.ResolveColorEnabled(colorMode, stdout, os.Getenv),
		}
		if len(reports) == 0 {
			fmt.Fprintf(stdout, "%s\n", style.Dim(fmt.Sprintf("Differential Test Report (base: %s): no executable Go or TypeScript production lines changed.", baseLabel)))
			return nil
		}
		for i, rep := range reports {
			if i > 0 {
				fmt.Fprintln(stdout)
			}
			RenderTextReport(stdout, rep, style)
		}
	case "github":
		summaryTarget := cfg.summaryFile
		if summaryTarget == "" {
			summaryTarget = os.Getenv("GITHUB_STEP_SUMMARY")
		}
		var summaryBuf bytes.Buffer
		if len(reports) == 0 {
			fmt.Fprintln(&summaryBuf, "## Differential Test & Mutation Report")
			fmt.Fprintln(&summaryBuf)
			fmt.Fprintln(&summaryBuf, "No executable Go or TypeScript production lines were added or modified in this change.")
		} else {
			for i, rep := range reports {
				RenderGitHubAnnotations(stdout, rep, cfg.maxAnnotations, cfg.minUncoveredLines)
				if i > 0 {
					fmt.Fprintln(&summaryBuf)
				}
				RenderGitHubSummary(&summaryBuf, rep)
			}
		}
		if summaryTarget != "" {
			if err := gotestutil.AppendSummaryFile(summaryTarget, summaryBuf.Bytes()); err != nil {
				return fmt.Errorf("failed to write GitHub step summary to %s: %w", summaryTarget, err)
			}
		} else {
			_, _ = stdout.Write(summaryBuf.Bytes())
		}
	}
	return nil
}

func parseFlags(args []string) (cliConfig, error) {
	var cfg cliConfig
	fs := flag.NewFlagSet("mutation-diff", flag.ContinueOnError)
	fs.StringVar(&cfg.repoRoot, "repo-root", "", "Path to the KHI repository root (defaults to auto-detecting go.mod).")
	fs.StringVar(&cfg.lang, "lang", "all", "Target language(s) to analyze: all, go, or ts.")
	fs.StringVar(&cfg.vcs, "vcs", "auto", "Version control system to query: auto, jj, or git.")
	fs.StringVar(&cfg.base, "base", "", "Base revision to diff against (defaults to trunk() in jj or origin/main in git).")
	fs.StringVar(&cfg.target, "target", "", "Optional target revision to diff against.")
	fs.StringVar(&cfg.coverProfile, "coverprofile", "", "Existing Go coverprofile to reuse instead of running a coverage pass.")
	fs.StringVar(&cfg.tsCoverage, "ts-coverage", "", "Existing Istanbul coverage-final.json to reuse instead of running a Karma coverage pass.")
	fs.BoolVar(&cfg.full, "full", false, "Run tests across the entire repository when collecting coverage locally.")
	fs.BoolVar(&cfg.checkRedundantTests, "check-redundant-tests", true, "Detect changed unit tests/subtests already covered by another test.")
	fs.IntVar(&cfg.workers, "workers", 0, "Parallel Go mutant test worker count (0 selects min(NumCPU, 8)).")
	fs.IntVar(&cfg.maxMutants, "max-mutants", defaultMaxMutants, "Maximum reached mutants to execute per language.")
	fs.DurationVar(&cfg.timeout, "timeout", defaultOverallTimeout, "Overall timeout budget per language.")
	fs.StringVar(&cfg.format, "format", "text", "Output format: text or github.")
	fs.StringVar(&cfg.color, "color", "auto", "Colorize text output: auto, always, or never.")
	fs.StringVar(&cfg.summaryFile, "summary-file", "", "File path to append GitHub Markdown summary (defaults to GITHUB_STEP_SUMMARY).")
	fs.IntVar(&cfg.maxAnnotations, "max-annotations", 10, "Maximum number of GitHub warning annotations to emit per language.")
	fs.IntVar(&cfg.minUncoveredLines, "min-annotation-lines", 4, "Minimum uncovered executable lines in a span required to emit a GitHub warning annotation.")
	fs.Float64Var(&cfg.failUnder, "fail-under", 0, "Minimum required differential mutation score percentage (0 disables failing exit status).")
	fs.Float64Var(&cfg.failUnderCoverage, "fail-under-coverage", 0, "Minimum required differential patch coverage percentage (0 disables failing exit status).")
	if err := fs.Parse(args); err != nil {
		return cliConfig{}, err
	}
	return cfg, nil
}

func sortedChangedFiles(m map[string]map[int]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
