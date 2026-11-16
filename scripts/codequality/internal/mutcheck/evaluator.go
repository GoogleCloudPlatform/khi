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

package mutcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/overlay"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/redundancy"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

// Options configures Stage-2 differential mutation testing and test_redundancy evaluation.
type Options struct {
	// RepoRoot is the repository root directory path.
	RepoRoot string
	// WorkDir is the temporary working directory for mutation builds and traces.
	WorkDir string
	// Thresholds specifies quality limits for mutation and redundancy rules.
	Thresholds rule.Thresholds
	// ChangedProdLinesByFile maps changed production file paths to modified 1-based line sets.
	ChangedProdLinesByFile map[string]map[int]struct{}
	// ChangedTestLinesByFile maps changed test file paths to modified 1-based line sets.
	ChangedTestLinesByFile map[string]map[int]struct{}
	// CoverBlocksByFile maps file paths to parsed Go coverprofile blocks.
	CoverBlocksByFile map[string][]gotestutil.CoverBlock
	// HasCoverProfile reports whether coverage data is available to pre-filter uncovered lines.
	HasCoverProfile bool
	// ASTResult holds Stage-1 AST analysis results shared with Stage-2 checks.
	ASTResult *astcheck.Result
	// EnsureBaseOverlay lazily materializes the base-revision overlay workspace.
	EnsureBaseOverlay func() (*overlay.Snapshot, error)
	// Workers specifies the parallel mutant execution worker count.
	Workers int
	// MaxMutants caps the total number of evaluated mutants across packages.
	MaxMutants int
	// Timeout bounds the overall mutation testing duration.
	Timeout time.Duration
	// LogWriter receives diagnostic compile and bisect messages.
	LogWriter io.Writer
	// Progress receives progress updates during compilation, tracing, and mutant execution.
	Progress ProgressHooks
}

// Result holds Stage-2 mutation summary stats and findings for func_surviving_mutants and test_redundancy.
type Result struct {
	// Summary holds aggregated mutant counts across evaluation statuses.
	Summary rule.MutationSummary
	// Findings contains func_surviving_mutants and test_redundancy violations.
	Findings []rule.Finding
	// Suppressed contains mutation findings suppressed by CODEQUALITY_ALLOW.
	Suppressed []rule.SuppressedItem
}

type pkgPrepSpec struct {
	pkgIdx    int
	totalPkgs int
	pkgDir    string
	prodFiles []string
	testFiles []string
	workers   int
}

type pkgExecEnv struct {
	pkgAbsDir  string
	pkgWorkDir string
	pkgLabel   string
	binPath    string
}

type pkgRunBundle struct {
	plans   []*FilePlan
	jobs    []mutantJob
	tracker *pkgRedundancyTracker
}

// Evaluate runs differential Super-Mutant evaluation, test_redundancy analysis, and lazy base ratchets.
func Evaluate(ctx context.Context, opts Options) (*Result, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	res := &Result{
		Summary: rule.MutationSummary{
			Enabled:    true,
			ByCategory: make(map[rule.AllowCategory]int),
		},
	}

	filesByPkg, sortedPkgs := groupFilesByPackage(opts.ChangedProdLinesByFile)
	testFilesByPkg, _ := groupFilesByPackage(opts.ChangedTestLinesByFile)
	workers := resolveWorkerCount(opts.Workers)

	var allPlans []*FilePlan
	var allJobs []mutantJob
	var trackers []*pkgRedundancyTracker

	for pkgIdx, pkgDir := range sortedPkgs {
		if ctx.Err() != nil {
			break
		}
		bundle, err := buildPackageBundle(ctx, opts, pkgPrepSpec{
			pkgIdx:    pkgIdx,
			totalPkgs: len(sortedPkgs),
			pkgDir:    pkgDir,
			prodFiles: filesByPkg[pkgDir],
			testFiles: testFilesByPkg[pkgDir],
			workers:   workers,
		})
		if err != nil {
			return nil, err
		}
		allPlans = append(allPlans, bundle.plans...)
		allJobs = append(allJobs, bundle.jobs...)
		if bundle.tracker != nil {
			trackers = append(trackers, bundle.tracker)
		}
	}

	selectedJobs := applyMutantBudget(allJobs, opts.MaxMutants)
	if len(selectedJobs) > 0 {
		if opts.Progress != nil {
			opts.Progress.StartMutantPool(len(selectedJobs), workers)
		}
		runMutantWorkerPool(ctx, selectedJobs, workers, opts.Progress)
	}

	accumulateSummary(&res.Summary, allPlans)
	if err := collectSurvivingMutantFindings(ctx, opts, allPlans, workers, res); err != nil {
		return nil, err
	}
	collectRedundancyFindings(opts, trackers, res)
	return res, nil
}

func buildPackageBundle(ctx context.Context, opts Options, spec pkgPrepSpec) (pkgRunBundle, error) {
	pkgPlans, maxMutantID, err := planPackageFiles(opts, spec.prodFiles)
	if err != nil || maxMutantID == 0 {
		return pkgRunBundle{plans: pkgPlans}, err
	}

	pkgAbsDir := filepath.Join(opts.RepoRoot, filepath.FromSlash(spec.pkgDir))
	hasTests, err := hasPackageTestFiles(pkgAbsDir)
	if err != nil || !hasTests {
		markPendingMutants(pkgPlans, rule.MutantUncovered)
		return pkgRunBundle{plans: pkgPlans}, err
	}

	pkgWorkDir := filepath.Join(opts.WorkDir, fmt.Sprintf("mut_pkg_%d", spec.pkgIdx))
	if err := os.MkdirAll(pkgWorkDir, 0o755); err != nil {
		return pkgRunBundle{}, fmt.Errorf("failed to create package workdir: %w", err)
	}
	pkgLabel := formatPkgLabel(spec.pkgDir)
	if opts.Progress != nil {
		opts.Progress.Phase(3, 4, fmt.Sprintf("Compiling package %d/%d: %s (%d candidate mutants)...", spec.pkgIdx+1, spec.totalPkgs, pkgLabel, maxMutantID))
	}

	binPath, err := compilePackageWithFallback(ctx, compileRequest{
		repoRoot:    opts.RepoRoot,
		pkgDir:      spec.pkgDir,
		pkgPlans:    pkgPlans,
		maxMutantID: maxMutantID,
		pkgWorkDir:  pkgWorkDir,
		logWriter:   opts.LogWriter,
	})
	if err != nil {
		return pkgRunBundle{plans: pkgPlans}, err
	}
	if binPath == "" {
		return pkgRunBundle{plans: pkgPlans}, nil
	}
	return baselinePackageBundle(ctx, opts, spec, pkgPlans, pkgExecEnv{
		pkgAbsDir:  pkgAbsDir,
		pkgWorkDir: pkgWorkDir,
		pkgLabel:   pkgLabel,
		binPath:    binPath,
	})
}

func baselinePackageBundle(
	ctx context.Context,
	opts Options,
	spec pkgPrepSpec,
	pkgPlans []*FilePlan,
	env pkgExecEnv,
) (pkgRunBundle, error) {
	changedTestFuncs, err := collectPackageChangedTestFuncs(opts.RepoRoot, spec.testFiles, opts.ChangedTestLinesByFile)
	if err != nil {
		return pkgRunBundle{}, err
	}
	if opts.Progress != nil {
		opts.Progress.Phase(3, 4, fmt.Sprintf("Running baseline reachability in %s...", env.pkgLabel))
	}
	tracePath := filepath.Join(env.pkgWorkDir, "baseline.trace")
	dur, out, err := runBaselineTestBinary(ctx, baselineRequest{
		binPath:   env.binPath,
		pkgAbsDir: env.pkgAbsDir,
		tracePath: tracePath,
		verbose:   len(changedTestFuncs) > 0,
	})
	if err != nil {
		return pkgRunBundle{}, fmt.Errorf("baseline unit tests failed in %s: %w", env.pkgLabel, err)
	}
	_, reached, err := ParsePolarityTraceFile(tracePath)
	if err != nil {
		return pkgRunBundle{}, err
	}

	perMutTimeout := computePerMutantTimeout(dur)
	var tracker *pkgRedundancyTracker
	if len(changedTestFuncs) > 0 {
		tracker = prepareRedundancyTracker(ctx, trackerSetup{
			binPath:       env.binPath,
			pkgLabel:      env.pkgLabel,
			pkgAbsDir:     env.pkgAbsDir,
			pkgWorkDir:    env.pkgWorkDir,
			baselineDur:   dur,
			perMutTimeout: perMutTimeout,
			workers:       spec.workers,
			baselineOut:   out,
			changedFuncs:  changedTestFuncs,
			pkgPlans:      pkgPlans,
			progress:      opts.Progress,
		})
	}
	jobs := buildReachedJobs(pkgPlans, reached, env, perMutTimeout, tracker)
	return pkgRunBundle{plans: pkgPlans, jobs: jobs, tracker: tracker}, nil
}

func buildReachedJobs(
	pkgPlans []*FilePlan,
	reached map[int]struct{},
	env pkgExecEnv,
	perMutTimeout time.Duration,
	tracker *pkgRedundancyTracker,
) []mutantJob {
	var jobs []mutantJob
	for _, p := range pkgPlans {
		for _, m := range p.Mutants {
			if m.Status != rule.MutantPending {
				continue
			}
			if _, ok := reached[m.ID]; !ok {
				m.Status = rule.MutantUncovered
				continue
			}
			jobs = append(jobs, mutantJob{
				mutant:           m,
				binPath:          env.binPath,
				pkgAbsDir:        env.pkgAbsDir,
				perMutantTimeout: perMutTimeout,
				tracker:          tracker,
			})
		}
	}
	return jobs
}

func planPackageFiles(opts Options, prodFiles []string) ([]*FilePlan, int, error) {
	pkgPlans := make([]*FilePlan, 0, len(prodFiles))
	nextID := 1
	for _, relFile := range prodFiles {
		plan, err := planSingleProdFile(opts, relFile, &nextID)
		if err != nil {
			return nil, 0, err
		}
		if plan != nil {
			pkgPlans = append(pkgPlans, plan)
		}
	}
	return pkgPlans, nextID - 1, nil
}

func planSingleProdFile(opts Options, relFile string, nextID *int) (*FilePlan, error) {
	absFile := filepath.Join(opts.RepoRoot, filepath.FromSlash(relFile))
	src, err := os.ReadFile(absFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", relFile, err)
	}
	var coveredLines map[int]struct{}
	if opts.HasCoverProfile {
		coveredLines = gotestutil.CoveredLineSet(opts.CoverBlocksByFile[relFile])
	}
	var directives []rule.Directive
	if fm := opts.ASTResult.Files[relFile]; fm != nil {
		directives = fm.Directives
	}
	return PlanFileMutations(relFile, src, PlanOptions{
		TargetLines:  opts.ChangedProdLinesByFile[relFile],
		CoveredLines: coveredLines,
		Directives:   directives,
		NextMutantID: nextID,
	})
}

func accumulateSummary(sum *rule.MutationSummary, plans []*FilePlan) {
	for _, p := range plans {
		for _, m := range p.Mutants {
			recordMutantSummaryStatus(sum, m)
		}
	}
}

func recordMutantSummaryStatus(sum *rule.MutationSummary, m *Mutant) {
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
	case rule.MutantCompileSkipped:
		sum.CompileSkipped++
	}
}

func collectRedundancyFindings(opts Options, trackers []*pkgRedundancyTracker, res *Result) {
	limit := opts.Thresholds.Limit(rule.RuleTestRedundancy)
	for _, tr := range trackers {
		groups := redundancy.Analyze(redundancy.AnalyzeInput{
			ChangedFuncs:         tr.changedFuncs,
			LeafOrder:            tr.leafOrder,
			PolarityByLeaf:       tr.polarityByLeaf,
			MutantRuns:           tr.mutantRuns,
			BudgetSkippedMutants: collectBudgetSkippedMutantIDs(tr.pkgPlans),
		})
		res.Findings = append(res.Findings, redundancy.BuildFindings(groups, limit, nil)...)
	}
}
