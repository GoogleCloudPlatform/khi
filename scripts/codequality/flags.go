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
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

const (
	subcmdQA            = "qa"
	subcmdCheck         = "check"
	subcmdConventions   = "conventions"
	subcmdUnusedExports = "unused-exports"

	defaultMaxMutants     = 0
	defaultOverallTimeout = 3 * time.Minute
	defaultMaxAnnotations = 20
)

type cliFlags struct {
	subcommand                string
	repoRoot                  string
	trustedRoot               string
	lang                      string
	speed                     string
	filter                    string
	paths                     string
	changedSince              string
	skipSources               string
	useContainer              bool
	pkgs                      string
	vcs                       string
	base                      string
	target                    string
	coverprofile              string
	tsCoverage                string
	full                      bool
	workers                   int
	maxMutants                int
	timeout                   time.Duration
	format                    string
	color                     string
	summaryFile               string
	maxAnnotations            int
	similarity                bool
	similarityProject         string
	decisions                 bool
	decisionsProject          string
	decisionsRules            string
	decisionsRulesDir         string
	decisionsMaxTargets       int
	decisionsFailOnViolations bool
	failOnViolations          bool
	tsFailOnViolations        bool
	commentEnvelope           bool
	commitSHA                 string
	buildURL                  string
}

type cliEnv struct {
	args           []string
	stdout         io.Writer
	stderr         io.Writer
	getenv         func(string) string
	embedFactory   embedderFactory
	deciderFactory deciderFactory
}

func parseFlags(args []string, errOut io.Writer, getenv func(string) string) (cliFlags, error) {
	if len(args) == 0 {
		return cliFlags{}, fmt.Errorf("missing subcommand (want %s, %s, %s, or %s)", subcmdQA, subcmdCheck, subcmdConventions, subcmdUnusedExports)
	}
	subcmd := args[0]
	switch subcmd {
	case subcmdConventions, subcmdUnusedExports:
		return parseASTSubcommandFlags(subcmd, args[1:], errOut)
	case subcmdQA, subcmdCheck:
		return parseQASubcommandFlags(subcmd, args[1:], errOut, getenv)
	default:
		return cliFlags{}, fmt.Errorf("unknown subcommand %q (want %s, %s, %s, or %s)", subcmd, subcmdQA, subcmdCheck, subcmdConventions, subcmdUnusedExports)
	}
}

func parseASTSubcommandFlags(subcmd string, args []string, errOut io.Writer) (cliFlags, error) {
	f := cliFlags{
		subcommand: subcmd,
		lang:       "go",
		speed:      string(rule.SpeedAll),
		filter:     string(rule.FilterMust),
		format:     "text",
		color:      "auto",
	}
	fs := flag.NewFlagSet("codequality "+subcmd, flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.StringVar(&f.repoRoot, "repo-root", "", "repository root path (auto-detected by default)")
	fs.StringVar(&f.pkgs, "pkgs", "./...", "comma-separated Go package patterns, e.g. ./pkg/api/...,./pkg/common")
	if err := fs.Parse(args); err != nil {
		return cliFlags{}, err
	}
	return f, nil
}

func parseQASubcommandFlags(subcmd string, args []string, errOut io.Writer, getenv func(string) string) (cliFlags, error) {
	defaultSpeed := string(rule.SpeedAll)
	if subcmd == subcmdQA {
		defaultSpeed = string(rule.SpeedFast)
	}

	f := cliFlags{subcommand: subcmd}
	fs := flag.NewFlagSet("codequality "+subcmd, flag.ContinueOnError)
	fs.SetOutput(errOut)
	registerScopeAndExecutionFlags(fs, &f, defaultSpeed, getenv)
	registerOutputFlags(fs, &f, getenv)

	if err := fs.Parse(args); err != nil {
		return cliFlags{}, err
	}
	if err := validateEnumFlags(f); err != nil {
		return cliFlags{}, err
	}
	return f, nil
}

func registerScopeAndExecutionFlags(fs *flag.FlagSet, f *cliFlags, defaultSpeed string, getenv func(string) string) {
	fs.StringVar(&f.repoRoot, "repo-root", "", "repository root path (auto-detected by default)")
	fs.StringVar(&f.trustedRoot, "trusted-root", "", "trusted repository root for loading decision rules and reference_files (defaults to --repo-root)")
	fs.StringVar(&f.lang, "lang", "all", "target language(s): all, go, or ts")
	fs.StringVar(&f.speed, "speed", defaultSpeed, "execution speed tier: fast, slow, or all")
	fs.StringVar(&f.filter, "filter", string(rule.FilterAll), "severity filter: all, must, or advisory")
	fs.StringVar(&f.paths, "paths", "", "comma-separated repository-relative path prefixes or Go package patterns")
	fs.StringVar(&f.changedSince, "changed-since", "", "narrow target files to changes since revision while preserving --base")
	fs.StringVar(&f.skipSources, "skip-sources", "", "comma-separated check or advisory sources to skip")
	fs.BoolVar(&f.useContainer, "use-container", parseBoolEnv(getenv, "USE_CONTAINER"), "run supported external tools in container mode")
	fs.StringVar(&f.vcs, "vcs", "auto", "version control system: auto, jj, or git")
	fs.StringVar(&f.base, "base", "", "base revision for differential checks")
	fs.StringVar(&f.target, "target", "", "target revision for differential checks")
	fs.StringVar(&f.coverprofile, "coverprofile", "", "optional precomputed Go test coverprofile path")
	fs.StringVar(&f.tsCoverage, "ts-coverage", "", "optional precomputed Istanbul coverage-final.json path")
	fs.BoolVar(&f.full, "full", false, "run full-repository coverage in addition to patch checks")
	fs.IntVar(&f.workers, "workers", 0, "parallel Go mutant worker count (0 = logical CPU core count)")
	fs.IntVar(&f.maxMutants, "max-mutants", defaultMaxMutants, "maximum reached mutants to evaluate per language (0 = unlimited)")
	fs.DurationVar(&f.timeout, "timeout", defaultOverallTimeout, "overall timeout for mutation evaluation per language")
}

func registerOutputFlags(fs *flag.FlagSet, f *cliFlags, getenv func(string) string) {
	fs.StringVar(&f.format, "format", "text", "report format: text, github, or json")
	fs.StringVar(&f.color, "color", "auto", "ANSI color mode: auto, always, or never")
	fs.StringVar(&f.summaryFile, "summary-file", getenv("GITHUB_STEP_SUMMARY"), "GitHub step summary file path")
	fs.IntVar(&f.maxAnnotations, "max-annotations", defaultMaxAnnotations, "maximum GitHub workflow annotations per language")
	fs.BoolVar(&f.similarity, "similarity", parseBoolEnv(getenv, "CODEQUALITY_SIMILARITY"), "enable AST-normalized function similarity advisories via Vertex AI (opt-in; also enabled via CODEQUALITY_SIMILARITY=true)")
	fs.StringVar(&f.similarityProject, "similarity-project", parseSimilarityProjectEnv(getenv), "GCP project ID for Vertex AI similarity embeddings (also set via CODEQUALITY_SIMILARITY_PROJECT)")
	fs.BoolVar(&f.decisions, "decisions", parseBoolEnv(getenv, "CODEQUALITY_DECISIONS"), "enable declarative YAML semantic rules via the Gemini Decisions API (opt-in; also enabled via CODEQUALITY_DECISIONS=true)")
	fs.StringVar(&f.decisionsProject, "decisions-project", parseDecisionsProjectEnv(getenv), "optional GCP project ID for scoped OAuth Gemini Decisions requests (also set via CODEQUALITY_DECISIONS_PROJECT)")
	fs.StringVar(&f.decisionsRules, "decisions-rules", "", "optional comma-separated decision rule IDs to evaluate (default: all enabled YAML rules)")
	fs.StringVar(&f.decisionsRulesDir, "decisions-rules-dir", "", "optional directory containing decision rule YAML files (defaults to <trusted-root>/scripts/codequality/rules)")
	fs.IntVar(&f.decisionsMaxTargets, "decisions-max-targets", 0, "maximum target bundles evaluated per decision rule (0 = unlimited)")
	fs.BoolVar(&f.decisionsFailOnViolations, "decisions-fail-on-violations", parseDecisionsFailEnv(getenv), "promote advisory decision rule findings to blocking violations (also enabled via CODEQUALITY_DECISIONS_FAIL_ON_VIOLATIONS=true)")
	fs.BoolVar(&f.failOnViolations, "fail-on-violations", true, "exit with code 1 when blocking violations are found")
	fs.BoolVar(&f.tsFailOnViolations, "ts-fail-on-violations", false, "treat TypeScript findings as blocking errors instead of advisory warnings")
	fs.BoolVar(&f.commentEnvelope, "comment-envelope", false, "emit Base64 sticky PR comment envelope to stdout for Check Run consumers")
	fs.StringVar(&f.commitSHA, "commit-sha", getenv("COMMIT_SHA"), "commit SHA rendered in sticky PR comment footer")
	fs.StringVar(&f.buildURL, "build-url", "", "CI build log URL rendered in sticky PR comment footer")
}

func validateEnumFlags(f cliFlags) error {
	if f.lang != "all" && f.lang != "go" && f.lang != "ts" {
		return fmt.Errorf("invalid --lang %q (want all, go, or ts)", f.lang)
	}
	if f.format != "text" && f.format != "github" && f.format != "json" {
		return fmt.Errorf("invalid --format %q (want text, github, or json)", f.format)
	}
	if _, err := rule.ParseSpeed(f.speed); err != nil {
		return err
	}
	if _, err := rule.ParseFilter(f.filter); err != nil {
		return err
	}
	return nil
}

func parseCSVList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for part := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
