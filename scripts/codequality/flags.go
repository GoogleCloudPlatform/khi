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
	"time"
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
	conventions        bool
	pkgs               string
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
	similarity         bool
	similarityProject  string
	failOnViolations   bool
	tsFailOnViolations bool
	commentEnvelope    bool
	commitSHA          string
	buildURL           string
}

type cliEnv struct {
	args         []string
	stdout       io.Writer
	stderr       io.Writer
	getenv       func(string) string
	embedFactory embedderFactory
}

func parseFlags(args []string, errOut io.Writer, getenv func(string) string) (cliFlags, error) {
	var f cliFlags
	fs := flag.NewFlagSet("codequality", flag.ContinueOnError)
	fs.SetOutput(errOut)
	registerScopeAndExecutionFlags(fs, &f)
	registerOutputFlags(fs, &f, getenv)

	if err := fs.Parse(args); err != nil {
		return cliFlags{}, err
	}
	if err := validateEnumFlags(f); err != nil {
		return cliFlags{}, err
	}
	return f, nil
}

func registerScopeAndExecutionFlags(fs *flag.FlagSet, f *cliFlags) {
	fs.StringVar(&f.repoRoot, "repo-root", "", "repository root path (auto-detected by default)")
	fs.StringVar(&f.lang, "lang", "all", "target language(s): all, go, or ts")
	fs.BoolVar(&f.fast, "fast", false, "run fast Stage-1 Go AST structural and naming rules only")
	fs.BoolVar(&f.conventions, "conventions", false, "check Go declaration visibility ordering and exported struct/interface member GoDoc conventions")
	fs.StringVar(&f.pkgs, "pkgs", "./...", "comma-separated Go package patterns for --conventions, e.g. ./pkg/api/...,./pkg/common")
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
	fs.StringVar(&f.format, "format", "text", "report format: text or github")
	fs.StringVar(&f.color, "color", "auto", "ANSI color mode: auto, always, or never")
	fs.StringVar(&f.summaryFile, "summary-file", getenv("GITHUB_STEP_SUMMARY"), "GitHub step summary file path")
	fs.IntVar(&f.maxAnnotations, "max-annotations", defaultMaxAnnotations, "maximum GitHub workflow annotations per language")
	fs.BoolVar(&f.similarity, "similarity", parseSimilarityEnv(getenv), "evaluate AST-normalized function similarity advisories via Vertex AI (opt-in; also enabled via CODEQUALITY_SIMILARITY=true)")
	fs.StringVar(&f.similarityProject, "similarity-project", parseSimilarityProjectEnv(getenv), "GCP project ID for Vertex AI similarity embeddings (also set via CODEQUALITY_SIMILARITY_PROJECT)")
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
	if f.format != "text" && f.format != "github" {
		return fmt.Errorf("invalid --format %q (want text or github)", f.format)
	}
	return nil
}
