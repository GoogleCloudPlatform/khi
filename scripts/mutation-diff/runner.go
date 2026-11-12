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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const (
	defaultMaxMutants       = 80
	defaultOverallTimeout   = 3 * time.Minute
	minPerMutantTimeout     = 3 * time.Second
	baselineTimeoutFactor   = 5
	fastVerboseBaselineCap  = 500 * time.Millisecond
	maxTracedSubtestsPerPkg = 40
)

// RunnerOptions configures package compilation and parallel mutant execution.
type RunnerOptions struct {
	RepoRoot               string
	BaseLabel              string
	ChangedLinesByFile     map[string]map[int]struct{}
	ChangedTestLinesByFile map[string]map[int]struct{}
	CoverBlocksByFile      map[string][]gotestutil.CoverBlock
	HasCoverProfile        bool
	CheckRedundantTests    bool
	Rules                  []ExclusionRule
	Workers                int
	MaxMutants             int
	Timeout                time.Duration
	LogWriter              io.Writer
}

type flatSiteRef struct {
	fileIdx int
	siteIdx int
}

type pkgRedundancyTracker struct {
	mu               sync.Mutex
	pkgPlans         []*FilePlan
	changedFuncs     []ChangedTestFunc
	leafOrder        []string
	polarityByLeaf   map[string]map[string]struct{}
	targetedRunRegex string
	mutantRuns       map[int]VerboseTestResult
}

func (t *pkgRedundancyTracker) recordMutantRun(mutantID int, res VerboseTestResult) {
	t.mu.Lock()
	t.mutantRuns[mutantID] = res
	t.mu.Unlock()
}

type mutantJob struct {
	mutant           *Mutant
	binPath          string
	pkgAbsDir        string
	perMutantTimeout time.Duration
	tracker          *pkgRedundancyTracker
}

// ExecuteMutationRun plans, compiles, baselines, and evaluates differential mutants across modified Go packages.
func ExecuteMutationRun(ctx context.Context, opts *RunnerOptions) (*MutationReport, error) {
	timeout := opts.Timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	filesByPkg, sortedPkgs := groupFilesByPackage(opts.ChangedLinesByFile)
	if len(sortedPkgs) == 0 {
		return BuildMutationReport(opts.BaseLabel, nil, nil, nil), nil
	}
	testFilesByPkg, _ := groupFilesByPackage(opts.ChangedTestLinesByFile)
	workers := resolveWorkerCount(opts.Workers)

	rootWorkDir, err := os.MkdirTemp("", "khi-mutation-diff-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary workspace: %w", err)
	}
	defer os.RemoveAll(rootWorkDir)

	var allFilePlans []*FilePlan
	var allWarnings []DirectiveWarning
	var allJobs []mutantJob
	var trackers []*pkgRedundancyTracker
	globalIdx := 1

	for pkgIdx, pkgDir := range sortedPkgs {
		if err := ctx.Err(); err != nil {
			break
		}

		pkgFiles := filesByPkg[pkgDir]
		pkgPlans := make([]*FilePlan, 0, len(pkgFiles))
		nextPkgMutantID := 1

		for _, relFile := range pkgFiles {
			absFile := filepath.Join(opts.RepoRoot, filepath.FromSlash(relFile))
			src, err := os.ReadFile(absFile)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return nil, fmt.Errorf("failed to read %s: %w", relFile, err)
			}

			var coveredLines map[int]struct{}
			if opts.HasCoverProfile {
				coveredLines = gotestutil.CoveredLineSet(opts.CoverBlocksByFile[relFile])
			}

			plan, err := PlanFileMutations(relFile, src, &MutateFileOptions{
				ChangedLines: opts.ChangedLinesByFile[relFile],
				CoveredLines: coveredLines,
				Rules:        opts.Rules,
				NextMutantID: &nextPkgMutantID,
			})
			if err != nil {
				return nil, err
			}
			for _, m := range plan.Mutants {
				m.GlobalIndex = globalIdx
				globalIdx++
			}
			pkgPlans = append(pkgPlans, plan)
			allFilePlans = append(allFilePlans, plan)
			allWarnings = append(allWarnings, plan.Warnings...)
		}

		maxMutantID := nextPkgMutantID - 1
		if maxMutantID == 0 {
			continue
		}

		pkgAbsDir := filepath.Join(opts.RepoRoot, filepath.FromSlash(pkgDir))
		hasTests, err := hasPackageTestFiles(pkgAbsDir)
		if err != nil {
			return nil, err
		}
		if !hasTests {
			markPendingMutants(pkgPlans, MutantUncovered)
			continue
		}

		pkgWorkDir := filepath.Join(rootWorkDir, fmt.Sprintf("pkg_%d", pkgIdx))
		if err := os.MkdirAll(pkgWorkDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create package workdir: %w", err)
		}

		binPath, err := compilePackageWithFallback(ctx, opts.RepoRoot, pkgDir, pkgPlans, maxMutantID, pkgWorkDir, opts.LogWriter)
		if err != nil {
			return nil, err
		}
		if binPath == "" {
			continue
		}

		var changedTestFuncs []ChangedTestFunc
		if opts.CheckRedundantTests {
			changedTestFuncs, err = collectPackageChangedTestFuncs(opts.RepoRoot, testFilesByPkg[pkgDir], opts.ChangedTestLinesByFile)
			if err != nil {
				return nil, err
			}
		}

		tracePath := filepath.Join(pkgWorkDir, "baseline.trace")
		baselineDur, baselineOut, err := runBaselineTestBinary(ctx, binPath, pkgAbsDir, tracePath, len(changedTestFuncs) > 0)
		if err != nil {
			return nil, fmt.Errorf("baseline unit tests failed in ./%s: %w", pkgDir, err)
		}

		reached, err := ParseTraceFile(tracePath)
		if err != nil {
			return nil, err
		}

		perMutTimeout := computePerMutantTimeout(baselineDur)
		var pkgTracker *pkgRedundancyTracker
		if len(changedTestFuncs) > 0 {
			pkgTracker = prepareRedundancyTracker(ctx, binPath, pkgAbsDir, pkgWorkDir, baselineDur, perMutTimeout, workers, baselineOut, changedTestFuncs, pkgPlans)
			if pkgTracker != nil {
				trackers = append(trackers, pkgTracker)
			}
		}

		for _, plan := range pkgPlans {
			for _, m := range plan.Mutants {
				if m.Status != MutantPending {
					continue
				}
				if _, ok := reached[m.ID]; !ok {
					m.Status = MutantUncovered
					continue
				}
				allJobs = append(allJobs, mutantJob{
					mutant:           m,
					binPath:          binPath,
					pkgAbsDir:        pkgAbsDir,
					perMutantTimeout: perMutTimeout,
					tracker:          pkgTracker,
				})
			}
		}
	}

	selectedJobs := applyMutantBudget(allJobs, opts.MaxMutants)
	if len(selectedJobs) > 0 {
		if opts.LogWriter != nil {
			fmt.Fprintf(opts.LogWriter, "Running %d reached Go mutants across %d workers...\n", len(selectedJobs), workers)
		}
		runMutantWorkerPool(ctx, selectedJobs, workers)
	}

	var redundantFindings []RedundantTestFinding
	for _, tr := range trackers {
		skippedIDs := collectBudgetSkippedMutantIDs(tr.pkgPlans)
		redundantFindings = append(redundantFindings, AnalyzeTestRedundancy(tr.changedFuncs, tr.leafOrder, tr.polarityByLeaf, tr.mutantRuns, skippedIDs)...)
	}
	for i := range redundantFindings {
		redundantFindings[i].GlobalIndex = i + 1
	}

	return BuildMutationReport(opts.BaseLabel, allFilePlans, allWarnings, redundantFindings), nil
}

func compilePackageWithFallback(
	ctx context.Context,
	repoRoot string,
	pkgDir string,
	pkgPlans []*FilePlan,
	maxMutantID int,
	pkgWorkDir string,
	logWriter io.Writer,
) (string, error) {
	var allRefs []flatSiteRef
	for fIdx, p := range pkgPlans {
		for sIdx := range p.Sites {
			allRefs = append(allRefs, flatSiteRef{fileIdx: fIdx, siteIdx: sIdx})
		}
	}
	if len(allRefs) == 0 {
		return "", nil
	}
	pkgName := pkgPlans[0].PackageName

	binPath := filepath.Join(pkgWorkDir, "pkg.test")
	if err := tryCompileSubset(ctx, repoRoot, pkgDir, pkgName, pkgPlans, allRefs, maxMutantID, filepath.Join(pkgWorkDir, "full"), binPath); err == nil {
		return binPath, nil
	}

	// Verify clean package compiles before bisecting individual mutation sites.
	cleanBin := filepath.Join(pkgWorkDir, "clean.test")
	if err := runGoTestCompile(ctx, repoRoot, "", cleanBin, pkgDir); err != nil {
		return "", fmt.Errorf("package ./%s failed to compile before mutation: %w", pkgDir, err)
	}

	if logWriter != nil {
		fmt.Fprintf(logWriter, "Package ./%s had type-inference edge case in Super-Mutant build; isolating valid sites...\n", pkgDir)
	}
	attemptCounter := 0
	validRefs := filterCompilableRefs(ctx, repoRoot, pkgDir, pkgName, pkgPlans, allRefs, maxMutantID, pkgWorkDir, &attemptCounter)

	validSet := make(map[flatSiteRef]struct{}, len(validRefs))
	for _, r := range validRefs {
		validSet[r] = struct{}{}
	}
	for fIdx, p := range pkgPlans {
		for sIdx := range p.Sites {
			ref := flatSiteRef{fileIdx: fIdx, siteIdx: sIdx}
			if _, ok := validSet[ref]; !ok {
				for _, m := range p.Sites[sIdx].Mutants {
					m.Status = MutantCompileSkipped
				}
			}
		}
	}

	if len(validRefs) == 0 {
		return "", nil
	}
	if err := tryCompileSubset(ctx, repoRoot, pkgDir, pkgName, pkgPlans, validRefs, maxMutantID, filepath.Join(pkgWorkDir, "final"), binPath); err != nil {
		return "", fmt.Errorf("failed to compile isolated mutant subset for ./%s: %w", pkgDir, err)
	}
	return binPath, nil
}

func filterCompilableRefs(
	ctx context.Context,
	repoRoot string,
	pkgDir string,
	pkgName string,
	pkgPlans []*FilePlan,
	refs []flatSiteRef,
	maxMutantID int,
	pkgWorkDir string,
	attemptCounter *int,
) []flatSiteRef {
	if len(refs) == 0 || ctx.Err() != nil {
		return nil
	}
	*attemptCounter++
	attemptDir := filepath.Join(pkgWorkDir, fmt.Sprintf("bisect_%d", *attemptCounter))
	tmpBin := filepath.Join(attemptDir, "sub.test")
	if err := tryCompileSubset(ctx, repoRoot, pkgDir, pkgName, pkgPlans, refs, maxMutantID, attemptDir, tmpBin); err == nil {
		return refs
	}
	if len(refs) == 1 {
		return nil
	}
	mid := len(refs) / 2
	left := filterCompilableRefs(ctx, repoRoot, pkgDir, pkgName, pkgPlans, refs[:mid], maxMutantID, pkgWorkDir, attemptCounter)
	right := filterCompilableRefs(ctx, repoRoot, pkgDir, pkgName, pkgPlans, refs[mid:], maxMutantID, pkgWorkDir, attemptCounter)
	return append(left, right...)
}

func tryCompileSubset(
	ctx context.Context,
	repoRoot string,
	pkgDir string,
	pkgName string,
	pkgPlans []*FilePlan,
	refs []flatSiteRef,
	maxMutantID int,
	workDir string,
	outBin string,
) error {
	sitesPerFile := make([][]MutationSite, len(pkgPlans))
	for _, r := range refs {
		sitesPerFile[r.fileIdx] = append(sitesPerFile[r.fileIdx], pkgPlans[r.fileIdx].Sites[r.siteIdx])
	}

	specs := make([]OverlayFileSpec, 0, len(pkgPlans))
	for i, p := range pkgPlans {
		if len(sitesPerFile[i]) == 0 {
			continue
		}
		specs = append(specs, OverlayFileSpec{
			RelFile: p.File,
			Source:  p.Source,
			Sites:   sitesPerFile[i],
		})
	}

	overlayPath, err := WritePackageOverlay(repoRoot, pkgDir, pkgName, specs, maxMutantID, workDir)
	if err != nil {
		return err
	}
	return runGoTestCompile(ctx, repoRoot, overlayPath, outBin, pkgDir)
}

func runGoTestCompile(ctx context.Context, repoRoot string, overlayPath string, outBin string, pkgDir string) error {
	pkgTarget := "./" + strings.TrimPrefix(pkgDir, "./")
	if pkgDir == "." || pkgDir == "" {
		pkgTarget = "."
	}
	args := []string{"test"}
	if overlayPath != "" {
		args = append(args, "-overlay="+overlayPath)
	}
	args = append(args, "-c", "-o", outBin, pkgTarget)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = buildMutantEnv(os.Environ(), 0, "")
	var stderr bytes.Buffer
	cmd.Stdout = &stderr
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func runBaselineTestBinary(ctx context.Context, binPath string, pkgAbsDir string, tracePath string, verbose bool) (time.Duration, []byte, error) {
	start := time.Now()
	var args []string
	if verbose {
		args = append(args, "-test.v")
	}
	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Dir = pkgAbsDir
	cmd.Env = buildMutantEnv(os.Environ(), 0, tracePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return 0, nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(out.String()))
	}
	return time.Since(start), out.Bytes(), nil
}

func collectPackageChangedTestFuncs(
	repoRoot string,
	relTestFiles []string,
	changedTestLines map[string]map[int]struct{},
) ([]ChangedTestFunc, error) {
	var out []ChangedTestFunc
	for _, relFile := range relTestFiles {
		absPath := filepath.Join(repoRoot, filepath.FromSlash(relFile))
		src, err := os.ReadFile(absPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("failed to read test file %s: %w", relFile, err)
		}
		funcs, err := DiscoverChangedTestFuncs(relFile, src, changedTestLines[relFile])
		if err != nil {
			return nil, err
		}
		out = append(out, funcs...)
	}
	return out, nil
}

func prepareRedundancyTracker(
	ctx context.Context,
	binPath string,
	pkgAbsDir string,
	pkgWorkDir string,
	baselineDur time.Duration,
	perMutTimeout time.Duration,
	workers int,
	baselineOut []byte,
	changedFuncs []ChangedTestFunc,
	pkgPlans []*FilePlan,
) *pkgRedundancyTracker {
	parsed := ParseVerboseTestOutput(baselineOut, false)
	fnSet := make(map[string]struct{}, len(changedFuncs))
	for _, fn := range changedFuncs {
		fnSet[fn.FuncName] = struct{}{}
	}

	var relevantLeafs []string
	for _, leaf := range parsed.LeafOrder {
		topFn, _ := splitTopAndSubtest(leaf)
		if _, ok := fnSet[topFn]; ok {
			relevantLeafs = append(relevantLeafs, leaf)
			if len(relevantLeafs) >= maxTracedSubtestsPerPkg {
				break
			}
		}
	}
	if len(relevantLeafs) < 2 {
		return nil
	}

	polarityByLeaf := traceLeafPolarities(ctx, binPath, pkgAbsDir, pkgWorkDir, perMutTimeout, workers, relevantLeafs)
	targetedRegex := ""
	if baselineDur >= fastVerboseBaselineCap {
		targetedRegex = BuildTestFuncsRunRegex(changedFuncs)
	}
	return &pkgRedundancyTracker{
		pkgPlans:         pkgPlans,
		changedFuncs:     changedFuncs,
		leafOrder:        relevantLeafs,
		polarityByLeaf:   polarityByLeaf,
		targetedRunRegex: targetedRegex,
		mutantRuns:       make(map[int]VerboseTestResult),
	}
}

func collectBudgetSkippedMutantIDs(plans []*FilePlan) map[int]struct{} {
	skipped := make(map[int]struct{})
	for _, p := range plans {
		for _, m := range p.Mutants {
			if m.Status == MutantSkippedBudget && m.ID > 0 {
				skipped[m.ID] = struct{}{}
			}
		}
	}
	return skipped
}

func traceLeafPolarities(
	ctx context.Context,
	binPath string,
	pkgAbsDir string,
	pkgWorkDir string,
	timeout time.Duration,
	workers int,
	leaves []string,
) map[string]map[string]struct{} {
	type leafTask struct {
		idx  int
		leaf string
	}
	taskCh := make(chan leafTask, len(leaves))
	for i, leaf := range leaves {
		taskCh <- leafTask{idx: i, leaf: leaf}
	}
	close(taskCh)

	var mu sync.Mutex
	result := make(map[string]map[string]struct{}, len(leaves))
	baseEnv := os.Environ()

	var wg sync.WaitGroup
	for range min(workers, len(leaves)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				if ctx.Err() != nil {
					return
				}
				traceFile := filepath.Join(pkgWorkDir, fmt.Sprintf("leaf_%d.trace", task.idx))
				runCtx, cancel := context.WithTimeout(ctx, timeout)
				cmd := exec.CommandContext(runCtx, binPath, "-test.run="+BuildSingleLeafRunRegex(task.leaf))
				cmd.Dir = pkgAbsDir
				cmd.Env = buildMutantEnv(baseEnv, 0, traceFile)
				_ = cmd.Run()
				cancel()

				tokens, _, err := ParsePolarityTraceFile(traceFile)
				if err == nil && len(tokens) > 0 {
					mu.Lock()
					result[task.leaf] = tokens
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	return result
}

func runMutantWorkerPool(ctx context.Context, jobs []mutantJob, workers int) {
	jobCh := make(chan mutantJob, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			baseEnv := os.Environ()
			for job := range jobCh {
				if ctx.Err() != nil {
					job.mutant.Status = MutantSkippedBudget
					continue
				}
				job.mutant.Status = evaluateSingleMutant(ctx, baseEnv, job)
			}
		}()
	}
	wg.Wait()
}

func evaluateSingleMutant(parentCtx context.Context, baseEnv []string, job mutantJob) MutantStatus {
	mutCtx, cancel := context.WithTimeout(parentCtx, job.perMutantTimeout)
	defer cancel()

	if job.tracker != nil {
		args := []string{"-test.v"}
		if job.tracker.targetedRunRegex != "" {
			args = append(args, "-test.run="+job.tracker.targetedRunRegex)
		}
		cmd := exec.CommandContext(mutCtx, job.binPath, args...)
		cmd.Dir = job.pkgAbsDir
		cmd.Env = buildMutantEnv(baseEnv, job.mutant.ID, "")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		if parentCtx.Err() != nil {
			return MutantSkippedBudget
		}
		if errors.Is(mutCtx.Err(), context.DeadlineExceeded) {
			return MutantTimedOut
		}
		job.tracker.recordMutantRun(job.mutant.ID, ParseVerboseTestOutput(out.Bytes(), err != nil))
		if err != nil {
			return MutantKilled
		}
		if job.tracker.targetedRunRegex == "" {
			return MutantSurvived
		}
	}

	cmd := exec.CommandContext(mutCtx, job.binPath, "-test.failfast")
	cmd.Dir = job.pkgAbsDir
	cmd.Env = buildMutantEnv(baseEnv, job.mutant.ID, "")

	err := cmd.Run()
	if parentCtx.Err() != nil {
		return MutantSkippedBudget
	}
	if errors.Is(mutCtx.Err(), context.DeadlineExceeded) {
		return MutantTimedOut
	}
	if err != nil {
		return MutantKilled
	}
	return MutantSurvived
}

func applyMutantBudget(jobs []mutantJob, maxMutants int) []mutantJob {
	if maxMutants <= 0 || len(jobs) <= maxMutants {
		return jobs
	}

	// Round-robin across files so a single large modified file cannot starve other files in the diff.
	var fileOrder []string
	byFile := make(map[string][]mutantJob)
	for _, j := range jobs {
		f := j.mutant.File
		if _, seen := byFile[f]; !seen {
			fileOrder = append(fileOrder, f)
		}
		byFile[f] = append(byFile[f], j)
	}

	selected := make([]mutantJob, 0, maxMutants)
	cursors := make(map[string]int, len(fileOrder))
	for len(selected) < maxMutants {
		progressed := false
		for _, f := range fileOrder {
			if len(selected) >= maxMutants {
				break
			}
			idx := cursors[f]
			if idx < len(byFile[f]) {
				selected = append(selected, byFile[f][idx])
				cursors[f] = idx + 1
				progressed = true
			}
		}
		if !progressed {
			break
		}
	}

	for _, f := range fileOrder {
		for i := cursors[f]; i < len(byFile[f]); i++ {
			byFile[f][i].mutant.Status = MutantSkippedBudget
		}
	}
	return selected
}

func groupFilesByPackage(changed map[string]map[int]struct{}) (map[string][]string, []string) {
	byPkg := make(map[string][]string)
	var pkgs []string
	for relFile := range changed {
		clean := strings.TrimPrefix(filepath.ToSlash(relFile), "./")
		pkgDir := path.Dir(clean)
		if _, ok := byPkg[pkgDir]; !ok {
			pkgs = append(pkgs, pkgDir)
		}
		byPkg[pkgDir] = append(byPkg[pkgDir], clean)
	}
	slices.Sort(pkgs)
	for _, pkg := range pkgs {
		slices.Sort(byPkg[pkg])
	}
	return byPkg, pkgs
}

func hasPackageTestFiles(pkgAbsDir string) (bool, error) {
	entries, err := os.ReadDir(pkgAbsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read package directory %s: %w", pkgAbsDir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
			return true, nil
		}
	}
	return false, nil
}

func markPendingMutants(plans []*FilePlan, status MutantStatus) {
	for _, p := range plans {
		for _, m := range p.Mutants {
			if m.Status == MutantPending {
				m.Status = status
			}
		}
	}
}

func computePerMutantTimeout(baseline time.Duration) time.Duration {
	return max(baseline*baselineTimeoutFactor, minPerMutantTimeout)
}

func resolveWorkerCount(requested int) int {
	if requested > 0 {
		return requested
	}
	return min(max(runtime.NumCPU(), 1), 8)
}

func buildMutantEnv(base []string, mutantID int, tracePath string) []string {
	env := make([]string, 0, len(base)+3)
	hasSkipCloudLogging := false
	for _, kv := range base {
		switch {
		case strings.HasPrefix(kv, "KHI_MUTANT_ID="), strings.HasPrefix(kv, "KHI_MUTANT_TRACE_FILE="):
			continue
		case strings.HasPrefix(kv, "KHI_SKIP_CLOUD_LOGGING="):
			hasSkipCloudLogging = true
		}
		env = append(env, kv)
	}
	if !hasSkipCloudLogging {
		env = append(env, "KHI_SKIP_CLOUD_LOGGING=true")
	}
	env = append(env, "KHI_MUTANT_ID="+strconv.Itoa(mutantID))
	if tracePath != "" {
		env = append(env, "KHI_MUTANT_TRACE_FILE="+tracePath)
	}
	return env
}
