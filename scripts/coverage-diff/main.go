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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const defaultDiffCoverProfile = "go-cover-diff.output"

type cliConfig struct {
	repoRoot           string
	vcs                string
	base               string
	target             string
	coverProfile       string
	full               bool
	format             string
	summaryFile        string
	maxAnnotations     int
	minAnnotationLines int
	failUnder          float64
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "coverage-diff: %v\n", err)
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

	diffRes, err := gotestutil.CollectChangedLines(ctx, repoRoot, gotestutil.DiffOptions{
		VCS:    cfg.vcs,
		Base:   cfg.base,
		Target: cfg.target,
	})
	if err != nil {
		return err
	}

	changedFiles := sortedFileKeys(diffRes.ChangedLines)
	profilePath := cfg.coverProfile
	includeOverall := cfg.coverProfile != "" || cfg.full

	blocksByFile := make(map[string][]gotestutil.CoverBlock)
	if profilePath == "" {
		plan := PlanTestExecution(repoRoot, changedFiles, cfg.full)
		if !plan.Skip {
			profilePath = filepath.Join(repoRoot, defaultDiffCoverProfile)
			if err := os.Remove(profilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to remove stale coverprofile %s: %w", profilePath, err)
			}
			if err := RunGoTestCoverage(ctx, repoRoot, plan, profilePath, stderr); err != nil {
				return err
			}
		}
	}

	if profilePath != "" {
		blocksByFile, err = gotestutil.LoadCoverProfileIfPresent(profilePath, modulePath, gotestutil.FilterOptions{})
		if err != nil {
			return err
		}
	}

	readSource := func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
	}

	report, err := AnalyzeCoverage(diffRes.BaseLabel, blocksByFile, diffRes.ChangedLines, readSource, includeOverall)
	if err != nil {
		return err
	}

	switch cfg.format {
	case "text":
		RenderTextReport(stdout, report)
	case "github":
		RenderGitHubAnnotations(stdout, report, cfg.maxAnnotations, cfg.minAnnotationLines)
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

	if cfg.failUnder > 0 && report.PatchExecutableLines > 0 && report.PatchPercent() < cfg.failUnder {
		return fmt.Errorf("patch coverage %.1f%% is below required threshold %.1f%%", report.PatchPercent(), cfg.failUnder)
	}
	return nil
}

func parseFlags(args []string) (cliConfig, error) {
	var cfg cliConfig
	fs := flag.NewFlagSet("coverage-diff", flag.ContinueOnError)
	fs.StringVar(&cfg.repoRoot, "repo-root", "", "Path to the KHI repository root (defaults to auto-detecting go.mod).")
	fs.StringVar(&cfg.vcs, "vcs", "auto", "Version control system to query: auto, jj, or git.")
	fs.StringVar(&cfg.base, "base", "", "Base revision to diff against (defaults to trunk() in jj or origin/main in git).")
	fs.StringVar(&cfg.target, "target", "", "Optional target revision to diff against.")
	fs.StringVar(&cfg.coverProfile, "coverprofile", "", "Existing Go coverprofile to analyze instead of running go test.")
	fs.BoolVar(&cfg.full, "full", false, "Run go test across the entire repository (./...) when generating coverage locally.")
	fs.StringVar(&cfg.format, "format", "text", "Output format: text or github.")
	fs.StringVar(&cfg.summaryFile, "summary-file", "", "File path to append GitHub Markdown summary (defaults to GITHUB_STEP_SUMMARY).")
	fs.IntVar(&cfg.maxAnnotations, "max-annotations", 10, "Maximum number of GitHub warning annotations to emit.")
	fs.IntVar(&cfg.minAnnotationLines, "min-annotation-lines", 4, "Minimum uncovered executable lines in a span required to emit a GitHub warning annotation.")
	fs.Float64Var(&cfg.failUnder, "fail-under", 0, "Minimum required patch coverage percentage (0 disables failing exit status).")
	if err := fs.Parse(args); err != nil {
		return cliConfig{}, err
	}
	return cfg, nil
}

func sortedFileKeys(m map[string]map[int]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
