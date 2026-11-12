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
	vcs                 string
	base                string
	target              string
	coverProfile        string
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
	modulePath, err := gotestutil.ReadModulePath(filepath.Join(repoRoot, "go.mod"))
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

	changedFiles := sortedChangedFiles(diffRes.ChangedLines)
	includeOverall := cfg.coverProfile != "" || cfg.full
	coverBlocks := make(map[string][]gotestutil.CoverBlock)
	hasCoverProfile := false

	if cfg.coverProfile != "" {
		if info, err := os.Stat(cfg.coverProfile); err == nil && !info.IsDir() {
			progress.Phase(2, 4, fmt.Sprintf("Loading coverprofile %s...", filepath.Base(cfg.coverProfile)))
			coverBlocks, err = gotestutil.LoadCoverProfileIfPresent(cfg.coverProfile, modulePath)
			if err != nil {
				return err
			}
			hasCoverProfile = true
		}
	} else {
		covPlan := PlanCoverageExecution(repoRoot, changedFiles, cfg.full)
		if !covPlan.Skip {
			progress.Phase(2, 4, fmt.Sprintf("Measuring line coverage across %d package(s)...", len(covPlan.Packages)))
			tmpCovDir, err := os.MkdirTemp("", "khi-mutation-cov-*")
			if err != nil {
				return fmt.Errorf("failed to create temporary coverage directory: %w", err)
			}
			defer os.RemoveAll(tmpCovDir)

			tmpProfile := filepath.Join(tmpCovDir, "cover.out")
			if err := RunGoTestCoverage(ctx, repoRoot, covPlan, tmpProfile, stderr); err != nil {
				return err
			}
			coverBlocks, err = gotestutil.LoadCoverProfileIfPresent(tmpProfile, modulePath)
			if err != nil {
				return err
			}
			hasCoverProfile = true // NOMUTANTS(perf): baseline reachability marks uncovered lines identically; pre-filtering avoids compiling unexecuted packages.
		}
	}

	readSource := func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
	}
	covReport, err := AnalyzeCoverage(diffRes.BaseLabel, coverBlocks, diffRes.ChangedLines, readSource, includeOverall)
	if err != nil {
		return err
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
		return err
	}
	AttachLineCoverage(report, covReport)
	progress.Finish()

	switch cfg.format {
	case "text":
		style := gotestutil.TermStyle{
			Enabled: gotestutil.ResolveColorEnabled(colorMode, stdout, os.Getenv),
		}
		RenderTextReport(stdout, report, style)
	case "github":
		RenderGitHubAnnotations(stdout, report, cfg.maxAnnotations, cfg.minUncoveredLines)
		summaryTarget := cfg.summaryFile
		if summaryTarget == "" {
			summaryTarget = os.Getenv("GITHUB_STEP_SUMMARY")
		}
		if summaryTarget != "" {
			var buf bytes.Buffer
			RenderGitHubSummary(&buf, report)
			if err := gotestutil.AppendSummaryFile(summaryTarget, buf.Bytes()); err != nil {
				return fmt.Errorf("failed to write GitHub step summary to %s: %w", summaryTarget, err)
			}
		} else {
			RenderGitHubSummary(stdout, report)
		}
	}

	if cfg.failUnderCoverage > 0 && covReport.PatchExecutableLines > 0 && covReport.PatchPercent() < cfg.failUnderCoverage {
		return fmt.Errorf("patch line coverage %.1f%% is below required threshold %.1f%%", covReport.PatchPercent(), cfg.failUnderCoverage)
	}
	if cfg.failUnder > 0 && report.Tested() > 0 && report.ScorePercent() < cfg.failUnder {
		return fmt.Errorf("differential mutation score %.1f%% is below required threshold %.1f%%", report.ScorePercent(), cfg.failUnder)
	}
	return nil
}

func parseFlags(args []string) (cliConfig, error) {
	var cfg cliConfig
	fs := flag.NewFlagSet("mutation-diff", flag.ContinueOnError)
	fs.StringVar(&cfg.repoRoot, "repo-root", "", "Path to the KHI repository root (defaults to auto-detecting go.mod).")
	fs.StringVar(&cfg.vcs, "vcs", "auto", "Version control system to query: auto, jj, or git.")
	fs.StringVar(&cfg.base, "base", "", "Base revision to diff against (defaults to trunk() in jj or origin/main in git).")
	fs.StringVar(&cfg.target, "target", "", "Optional target revision to diff against.")
	fs.StringVar(&cfg.coverProfile, "coverprofile", "", "Existing Go coverprofile to reuse instead of running a coverage pass.")
	fs.BoolVar(&cfg.full, "full", false, "Run go test across the entire repository (./...) when collecting coverage locally.")
	fs.BoolVar(&cfg.checkRedundantTests, "check-redundant-tests", true, "Detect changed unit tests/subtests already covered by another test.")
	fs.IntVar(&cfg.workers, "workers", 0, "Parallel mutant test worker count (0 selects min(NumCPU, 8)).")
	fs.IntVar(&cfg.maxMutants, "max-mutants", defaultMaxMutants, "Maximum reached mutants to execute per run.")
	fs.DurationVar(&cfg.timeout, "timeout", defaultOverallTimeout, "Overall timeout budget for the mutation run.")
	fs.StringVar(&cfg.format, "format", "text", "Output format: text or github.")
	fs.StringVar(&cfg.color, "color", "auto", "Colorize text output: auto, always, or never.")
	fs.StringVar(&cfg.summaryFile, "summary-file", "", "File path to append GitHub Markdown summary (defaults to GITHUB_STEP_SUMMARY).")
	fs.IntVar(&cfg.maxAnnotations, "max-annotations", 10, "Maximum number of GitHub warning annotations to emit.")
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
