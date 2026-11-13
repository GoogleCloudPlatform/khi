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

package tscheck

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/redundancy"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const (
	defaultTSMaxMutants     = 80
	defaultTSOverallTimeout = 3 * time.Minute
)

// ProgressHooks reports high-level phase progress during TypeScript evaluation.
type ProgressHooks interface {
	Phase(step int, totalSteps int, message string)
}

// Options configures unified TypeScript differential coverage, mutation, and redundancy evaluation.
type Options struct {
	RepoRoot               string
	BaseLabel              string
	TSCoveragePath         string
	Full                   bool
	FailOnViolations       bool
	ChangedLinesByFile     map[string]map[int]struct{}
	ChangedTestLinesByFile map[string]map[int]struct{}
	MaxMutants             int
	Timeout                time.Duration
	LogWriter              io.Writer
	Progress               ProgressHooks
	RunKarmaCoverage       func(ctx context.Context, req KarmaCoverageRequest) error
	RunKarmaSession        func(ctx context.Context, req karmaSessionRequest) error
}

// Evaluate runs TypeScript differential coverage, Super-Mutant evaluation, and Jasmine spec redundancy analysis.
func Evaluate(ctx context.Context, opts Options) (*rule.Report, error) {
	rep := &rule.Report{
		Language:         "TypeScript",
		BaseLabel:        opts.BaseLabel,
		Advisory:         !opts.FailOnViolations,
		ProdFilesChanged: len(opts.ChangedLinesByFile),
		TestFilesChanged: len(opts.ChangedTestLinesByFile),
		Mutation: rule.MutationSummary{
			Enabled:    true,
			ByCategory: make(map[rule.AllowCategory]int),
		},
	}

	covByFile, hasTSCov, err := loadOrRunTSCoverage(ctx, opts)
	if err != nil {
		return nil, err
	}
	covRes, err := AnalyzeTSCoverage(TSCoverageOptions{
		CovByFile:          covByFile,
		ChangedLinesByFile: opts.ChangedLinesByFile,
		ReadSource: func(relPath string) ([]byte, error) {
			return os.ReadFile(filepath.Join(opts.RepoRoot, filepath.FromSlash(relPath)))
		},
		IncludeOverall: opts.TSCoveragePath != "" || opts.Full,
	})
	if err != nil {
		return nil, err
	}
	rep.Coverage = covRes.Summary
	rep.Findings = append(rep.Findings, covRes.Findings...)

	if err := evaluateTSMutationsAndSpecs(ctx, opts, covByFile, hasTSCov, rep); err != nil {
		return nil, err
	}
	sortTSFindings(rep)
	return rep, nil
}

type tsMutPlanBundle struct {
	allPlans       []*TSFilePlan
	stagedFiles    []tsStagedFile
	pendingMutants []*TSMutant
	filesPending   []string
	mutantFileByID map[int]string
	changedSpecs   []redundancy.ChangedTSSpec
	maxMutantID    int
	maxProbeID     int
}

func evaluateTSMutationsAndSpecs(
	ctx context.Context,
	opts Options,
	covByFile map[string]gotestutil.IstanbulFileCoverage,
	hasTSCov bool,
	rep *rule.Report,
) error {
	changedFiles := sortedTSProdFiles(opts.ChangedLinesByFile)
	if len(changedFiles) == 0 {
		return nil
	}
	timeout, hardTimeout := resolveTSMutationTimeouts(opts.Timeout)
	hardCtx, cancel := context.WithTimeout(ctx, hardTimeout)
	defer cancel()

	bundle, err := buildTSMutPlanBundle(hardCtx, opts, changedFiles, covByFile, hasTSCov)
	if err != nil {
		return err
	}
	rep.FuncsEvaluated = len(bundle.allPlans)

	pluginRes, err := runTSPendingMutantsIfNeeded(hardCtx, opts, &bundle, timeout)
	if err != nil {
		return err
	}
	recordTSBundleIntoReport(rep, &bundle, pluginRes)
	return nil
}

func resolveTSMutationTimeouts(configured time.Duration) (time.Duration, time.Duration) {
	overall := configured
	if overall <= 0 {
		overall = defaultTSOverallTimeout
	}
	return overall, overall + 35*time.Second
}

func buildTSMutPlanBundle(
	ctx context.Context,
	opts Options,
	changedFiles []string,
	covByFile map[string]gotestutil.IstanbulFileCoverage,
	hasTSCov bool,
) (tsMutPlanBundle, error) {
	changedSpecFiles := sortedTSSpecFiles(opts.ChangedTestLinesByFile)
	targets := slices.Concat(changedFiles, changedSpecFiles)
	phaseIfPresent(opts.Progress, 3, 4, fmt.Sprintf("Extracting AST mutation sites from %d TypeScript file(s)...", len(changedFiles)))
	extractions, err := InvokeTSMutatorScript(ctx, opts.RepoRoot, targets)
	if err != nil {
		return tsMutPlanBundle{}, err
	}

	var changedSpecs []redundancy.ChangedTSSpec
	for _, sf := range changedSpecFiles {
		changedSpecs = append(changedSpecs, redundancy.DiscoverChangedTSSpecs(sf, extractions[sf].SpecCases, opts.ChangedTestLinesByFile[sf])...)
	}

	out := tsMutPlanBundle{
		mutantFileByID: make(map[int]string),
		changedSpecs:   changedSpecs,
	}
	nextMutID := 1
	nextProbeID := 1

	for _, relFile := range changedFiles {
		if err := planSingleTSChangedFile(opts, tsSingleFileCtx{
			relFile:          relFile,
			extractions:      extractions,
			covByFile:        covByFile,
			hasTSCov:         hasTSCov,
			changedSpecFiles: changedSpecFiles,
			nextMutID:        &nextMutID,
			nextProbeID:      &nextProbeID,
		}, &out); err != nil {
			return tsMutPlanBundle{}, err
		}
	}
	out.maxMutantID = nextMutID - 1
	out.maxProbeID = nextProbeID - 1
	return out, nil
}

type tsSingleFileCtx struct {
	relFile          string
	extractions      map[string]RawTSFileExtraction
	covByFile        map[string]gotestutil.IstanbulFileCoverage
	hasTSCov         bool
	changedSpecFiles []string
	nextMutID        *int
	nextProbeID      *int
}

func planSingleTSChangedFile(opts Options, fc tsSingleFileCtx, out *tsMutPlanBundle) error {
	ext, ok := fc.extractions[fc.relFile]
	if !ok {
		return nil
	}
	src, err := os.ReadFile(filepath.Join(opts.RepoRoot, filepath.FromSlash(fc.relFile)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read %s: %w", fc.relFile, err)
	}
	coveredLines := gotestutil.CoveredTSLineSet(fc.covByFile[fc.relFile])
	var probeCounter *int
	if len(out.changedSpecs) > 0 && hasMatchingTSSpec(fc.relFile, fc.changedSpecFiles) {
		probeCounter = fc.nextProbeID
	}
	fp, sites := PlanTSFileMutations(PlanTSFileInput{
		FilePath:      fc.relFile,
		Source:        src,
		Extraction:    ext,
		ChangedLines:  opts.ChangedLinesByFile[fc.relFile],
		CoveredLines:  coveredLines,
		HasTSCoverage: fc.hasTSCov,
		NextID:        fc.nextMutID,
		NextProbeID:   probeCounter,
	})
	registerPlannedTSFile(fc.relFile, fp, sites, out)
	return nil
}

func registerPlannedTSFile(relFile string, fp *TSFilePlan, sites []TSMutationSite, out *tsMutPlanBundle) {
	hasPending := false
	for _, m := range fp.Mutants {
		if m.Status == rule.MutantPending {
			out.pendingMutants = append(out.pendingMutants, m)
			out.mutantFileByID[m.ID] = relFile
			hasPending = true
		}
	}
	for _, s := range sites {
		for _, sm := range s.Mutants {
			if sm.ID < 0 {
				out.mutantFileByID[sm.ID] = relFile
			}
		}
	}
	if hasPending {
		out.filesPending = append(out.filesPending, relFile)
		out.stagedFiles = append(out.stagedFiles, tsStagedFile{plan: fp, tsSites: sites})
	}
	out.allPlans = append(out.allPlans, fp)
}

func runTSPendingMutantsIfNeeded(
	ctx context.Context,
	opts Options,
	bundle *tsMutPlanBundle,
	timeout time.Duration,
) (*karmaPluginResults, error) {
	if len(bundle.pendingMutants) == 0 {
		return nil, nil
	}
	specPlan, err := gotestutil.PlanTSTestExecution(opts.RepoRoot, bundle.filesPending, false)
	if err != nil {
		return nil, err
	}
	if specPlan.SkipKarma {
		for _, m := range bundle.pendingMutants {
			m.Status = rule.MutantUncovered
		}
		return nil, nil
	}

	maxBudget := opts.MaxMutants
	if maxBudget <= 0 {
		maxBudget = defaultTSMaxMutants
	}
	phaseIfPresent(opts.Progress, 4, 4, fmt.Sprintf("Evaluating up to %d TypeScript mutant(s) across %d spec file(s)...",
		min(len(bundle.pendingMutants), maxBudget), len(specPlan.IncludeSpecs)))

	workDir, err := os.MkdirTemp("", "khi-ts-mutation-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workDir)

	resultsPath, cfgPath, err := prepareTSMutationWorkspace(karmaWorkspaceRequest{
		repoRoot:       opts.RepoRoot,
		workDir:        workDir,
		stagedFiles:    bundle.stagedFiles,
		pendingMutants: bundle.pendingMutants,
		maxMutantID:    bundle.maxMutantID,
		maxProbeID:     bundle.maxProbeID,
		maxMutants:     maxBudget,
		overallTimeout: timeout,
	})
	if err != nil {
		return nil, err
	}

	runSession := opts.RunKarmaSession
	if runSession == nil {
		runSession = runKarmaMutationSession
	}
	if err := runSession(ctx, karmaSessionRequest{
		repoRoot:     opts.RepoRoot,
		workDir:      workDir,
		cfgPath:      cfgPath,
		includeSpecs: specPlan.IncludeSpecs,
		errOut:       opts.LogWriter,
	}); err != nil {
		return nil, err
	}
	return readAndValidateKarmaResults(resultsPath, bundle.pendingMutants)
}

func readAndValidateKarmaResults(resultsPath string, pending []*TSMutant) (*karmaPluginResults, error) {
	resBytes, err := os.ReadFile(resultsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read TypeScript mutation results: %w", err)
	}
	var pluginRes karmaPluginResults
	if err := json.Unmarshal(resBytes, &pluginRes); err != nil {
		return nil, fmt.Errorf("failed to decode TypeScript mutation results: %w", err)
	}
	if pluginRes.BaselineFailed {
		return nil, fmt.Errorf("baseline TypeScript Karma tests failed: %s", pluginRes.ErrorMessage)
	}
	applyKarmaOutcomes(pending, pluginRes.Outcomes)
	return &pluginRes, nil
}

func recordTSBundleIntoReport(
	rep *rule.Report,
	bundle *tsMutPlanBundle,
	pluginRes *karmaPluginResults,
) {
	skippedIDs := make(map[int]struct{})
	for _, fp := range bundle.allPlans {
		rep.Suppressed = append(rep.Suppressed, fp.Suppressed...)
		rep.DirectiveErrors = append(rep.DirectiveErrors, fp.DirectiveErrors...)
		var survived []rule.MutantDetail
		for _, m := range fp.Mutants {
			accumulateTSMutantStatus(&rep.Mutation, m, skippedIDs)
			if m.Status == rule.MutantSurvived {
				survived = append(survived, m.ToDetail())
			}
		}
		if len(survived) > 0 {
			rep.Findings = append(rep.Findings, rule.Finding{
				Rule:             rule.RuleFuncSurvivingMutants,
				Kind:             rule.ViolationHardLimit,
				File:             fp.File,
				Line:             survived[0].Line,
				EndLine:          survived[len(survived)-1].Line,
				Subject:          "<file>",
				After:            len(survived),
				Limit:            0,
				Detail:           fmt.Sprintf("%d mutant(s) survived Karma unit tests", len(survived)),
				SurvivingMutants: survived,
			})
		}
	}

	if pluginRes != nil {
		groups := redundancy.AnalyzeTS(redundancy.TSAnalyzeInput{
			ChangedSpecs:         bundle.changedSpecs,
			SpecOrder:            pluginRes.SpecOrder,
			SpecPolarities:       pluginRes.SpecPolarities,
			MutantRuns:           pluginRes.MutantRuns,
			MutantFileByID:       bundle.mutantFileByID,
			BudgetSkippedMutants: skippedIDs,
		})
		rep.Findings = append(rep.Findings, redundancy.BuildFindings(groups, 0, nil)...)
	}
}

func accumulateTSMutantStatus(sum *rule.MutationSummary, m *TSMutant, skippedIDs map[int]struct{}) {
	switch m.Status {
	case rule.MutantKilled:
		sum.Killed++
	case rule.MutantTimedOut:
		sum.TimedOut++
	case rule.MutantSurvived:
		sum.Survived++
	case rule.MutantUncovered:
		sum.Uncovered++
	case rule.MutantSuppressed:
		sum.Suppressed++
		sum.ByCategory[m.SuppressionCategory]++
	case rule.MutantSkippedBudget:
		sum.SkippedBudget++
		skippedIDs[m.ID] = struct{}{}
	}
}

func sortedTSProdFiles(m map[string]map[int]struct{}) []string {
	var out []string
	for f, lines := range m {
		if len(lines) > 0 && gotestutil.IsTargetProductionTSFile(f) {
			out = append(out, f)
		}
	}
	slices.Sort(out)
	return out
}

func sortedTSSpecFiles(m map[string]map[int]struct{}) []string {
	var out []string
	for f, lines := range m {
		if len(lines) > 0 && gotestutil.IsTargetTSTestFile(f) {
			out = append(out, f)
		}
	}
	slices.Sort(out)
	return out
}

func hasMatchingTSSpec(prodFile string, changedSpecFiles []string) bool {
	for _, specFile := range changedSpecFiles {
		if redundancy.IsSubjectMutantForTSSpec(specFile, prodFile) {
			return true
		}
	}
	return false
}

func phaseIfPresent(p ProgressHooks, step int, total int, msg string) {
	if p != nil {
		p.Phase(step, total, msg)
	}
}

func sortTSFindings(rep *rule.Report) {
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
