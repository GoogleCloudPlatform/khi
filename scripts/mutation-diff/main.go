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
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type cliConfig struct {
	repoRoot       string
	vcs            string
	base           string
	target         string
	coverProfile   string
	includeScripts bool
	workers        int
	maxMutants     int
	timeout        time.Duration
	format         string
	summaryFile    string
	maxAnnotations int
	failUnder      float64
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

	repoRoot, err := gotestutil.ResolveRepoRoot(cfg.repoRoot)
	if err != nil {
		return err
	}
	modulePath, err := gotestutil.ReadModulePath(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return err
	}

	filterOpts := gotestutil.FilterOptions{
		IncludeScripts: cfg.includeScripts,
	}

	diffRes, err := gotestutil.CollectChangedLines(ctx, repoRoot, gotestutil.DiffOptions{
		VCS:    cfg.vcs,
		Base:   cfg.base,
		Target: cfg.target,
		Filter: filterOpts,
	})
	if err != nil {
		return err
	}

	var coverBlocks map[string][]gotestutil.CoverBlock
	hasCoverProfile := false
	if cfg.coverProfile != "" {
		if info, err := os.Stat(cfg.coverProfile); err == nil && !info.IsDir() {
			coverBlocks, err = gotestutil.LoadCoverProfileIfPresent(cfg.coverProfile, modulePath, filterOpts)
			if err != nil {
				return err
			}
			hasCoverProfile = true
		}
	}

	report, err := ExecuteMutationRun(ctx, &RunnerOptions{
		RepoRoot:           repoRoot,
		BaseLabel:          diffRes.BaseLabel,
		ChangedLinesByFile: diffRes.ChangedLines,
		CoverBlocksByFile:  coverBlocks,
		HasCoverProfile:    hasCoverProfile,
		Workers:            cfg.workers,
		MaxMutants:         cfg.maxMutants,
		Timeout:            cfg.timeout,
		LogWriter:          stderr,
	})
	if err != nil {
		return err
	}

	switch cfg.format {
	case "text":
		RenderTextReport(stdout, report)
	case "github":
		RenderGitHubAnnotations(stdout, report, cfg.maxAnnotations)
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
	fs.StringVar(&cfg.coverProfile, "coverprofile", "", "Optional Go coverprofile to skip compiling/running uncovered lines.")
	fs.BoolVar(&cfg.includeScripts, "include-scripts", false, "Include Go packages under scripts/ in differential mutation analysis.")
	fs.IntVar(&cfg.workers, "workers", 0, "Parallel mutant test worker count (0 selects min(NumCPU, 8)).")
	fs.IntVar(&cfg.maxMutants, "max-mutants", defaultMaxMutants, "Maximum reached mutants to execute per run.")
	fs.DurationVar(&cfg.timeout, "timeout", defaultOverallTimeout, "Overall timeout budget for the mutation run.")
	fs.StringVar(&cfg.format, "format", "text", "Output format: text or github.")
	fs.StringVar(&cfg.summaryFile, "summary-file", "", "File path to append GitHub Markdown summary (defaults to GITHUB_STEP_SUMMARY).")
	fs.IntVar(&cfg.maxAnnotations, "max-annotations", 5, "Maximum number of GitHub warning annotations to emit.")
	fs.Float64Var(&cfg.failUnder, "fail-under", 0, "Minimum required differential mutation score percentage (0 disables failing exit status).")
	if err := fs.Parse(args); err != nil {
		return cliConfig{}, err
	}
	return cfg, nil
}
