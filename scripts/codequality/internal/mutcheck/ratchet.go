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
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/overlay"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type pendingBaseMut struct {
	relFile  string
	fm       *astcheck.FileMetrics
	fn       astcheck.FuncMetrics
	baseFile string
	baseFn   astcheck.FuncMetrics
	survived []rule.MutantDetail
}

type lazyBaseEnv struct {
	snap    *overlay.Snapshot
	workers int
}

type fullFuncRequest struct {
	repoRoot   string
	workDir    string
	relFile    string
	fn         astcheck.FuncMetrics
	directives []rule.Directive
	isBase     bool
	workers    int
}

func collectSurvivingMutantFindings(
	ctx context.Context,
	opts Options,
	allPlans []*FilePlan,
	workers int,
	res *Result,
) error {
	survivedByKey := groupSurvivedByFunc(allPlans)
	var pending []pendingBaseMut

	for _, relFile := range sortedMapKeys(opts.ASTResult.TouchedFuncs) {
		fm := opts.ASTResult.Files[relFile]
		if fm == nil {
			continue
		}
		for _, fn := range opts.ASTResult.TouchedFuncs[relFile] {
			if p, ok := classifyFuncSurvival(opts, fm, fn, survivedByKey[relFile+"|"+fn.Key], res); ok {
				pending = append(pending, p)
			}
		}
	}
	if len(pending) == 0 {
		return nil
	}
	return evaluateLazyBaseMutants(ctx, opts, pending, workers, res)
}

func classifyFuncSurvival(
	opts Options,
	fm *astcheck.FileMetrics,
	fn astcheck.FuncMetrics,
	survived []rule.MutantDetail,
	res *Result,
) (pendingBaseMut, bool) {
	limit := opts.Thresholds.Limit(rule.RuleFuncSurvivingMutants)
	if len(survived) <= limit {
		return pendingBaseMut{}, false
	}
	baseFile, baseFn, hasBase := opts.ASTResult.Base.LookupFuncLocation(fm.File, fn.Key)
	if !hasBase || opts.EnsureBaseOverlay == nil {
		recordMutOutcome(res, fm, fn, rule.Sample{After: len(survived), Limit: limit}, survived)
		return pendingBaseMut{}, false
	}
	return pendingBaseMut{
		relFile:  fm.File,
		fm:       fm,
		fn:       fn,
		baseFile: baseFile,
		baseFn:   baseFn,
		survived: survived,
	}, true
}

func evaluateLazyBaseMutants(
	ctx context.Context,
	opts Options,
	pending []pendingBaseMut,
	workers int,
	res *Result,
) error {
	snap, err := opts.EnsureBaseOverlay()
	if err != nil {
		return err
	}
	limit := opts.Thresholds.Limit(rule.RuleFuncSurvivingMutants)
	env := lazyBaseEnv{snap: snap, workers: workers}

	for idx, item := range pending {
		if ctx.Err() != nil {
			break
		}
		fullAfter, fullSurvived, err := countFullFuncSurvived(ctx, fullFuncRequest{
			repoRoot:   opts.RepoRoot,
			workDir:    filepath.Join(opts.WorkDir, fmt.Sprintf("lazy_target_%d", idx)),
			relFile:    item.relFile,
			fn:         item.fn,
			directives: cloneDirectives(item.fm.Directives),
			workers:    workers,
		})
		if err != nil {
			return err
		}
		fullBefore := countBaseFuncSurvived(ctx, opts, env, idx, item)
		res.Summary.BaseOverlayRuns++

		details := fullSurvived
		if len(details) == 0 {
			details = item.survived
		}
		recordMutOutcome(res, item.fm, item.fn, rule.Sample{
			After:     fullAfter,
			Limit:     limit,
			Before:    fullBefore,
			HasBefore: true,
		}, details)
	}
	return nil
}

func countBaseFuncSurvived(
	ctx context.Context,
	opts Options,
	env lazyBaseEnv,
	idx int,
	item pendingBaseMut,
) int {
	if _, ok := env.snap.BaseSource(item.baseFile); !ok {
		return 0
	}
	var baseDirectives []rule.Directive
	if baseFM, hasFM := opts.ASTResult.Base.FileMetrics(item.baseFile); hasFM {
		baseDirectives = cloneDirectives(baseFM.Directives)
	}
	before, _, err := countFullFuncSurvived(ctx, fullFuncRequest{
		repoRoot:   env.snap.RepoRoot(),
		workDir:    filepath.Join(opts.WorkDir, fmt.Sprintf("lazy_base_%d", idx)),
		relFile:    item.baseFile,
		fn:         item.baseFn,
		directives: baseDirectives,
		isBase:     true,
		workers:    env.workers,
	})
	if err != nil {
		return 0
	}
	return before
}

func countFullFuncSurvived(ctx context.Context, req fullFuncRequest) (int, []rule.MutantDetail, error) {
	plan, maxID, err := planFullFunc(req)
	if err != nil || maxID == 0 {
		return 0, nil, err
	}
	return compileAndRunFullFunc(ctx, req, plan, maxID)
}

func planFullFunc(req fullFuncRequest) (*FilePlan, int, error) {
	src, err := os.ReadFile(filepath.Join(req.repoRoot, filepath.FromSlash(req.relFile)))
	if err != nil {
		return nil, 0, err
	}
	nextID := 1
	plan, err := PlanFileMutations(req.relFile, src, PlanOptions{
		TargetLines:  lineRangeSet(req.fn.StartLine, req.fn.EndLine),
		Directives:   req.directives,
		NextMutantID: &nextID,
	})
	return plan, nextID - 1, err
}

func compileAndRunFullFunc(
	ctx context.Context,
	req fullFuncRequest,
	plan *FilePlan,
	maxID int,
) (int, []rule.MutantDetail, error) {
	pkgDir := path.Dir(req.relFile)
	pkgAbsDir := filepath.Join(req.repoRoot, filepath.FromSlash(pkgDir))
	if hasTests, _ := hasPackageTestFiles(pkgAbsDir); !hasTests {
		if req.isBase {
			return len(plan.Mutants), nil, nil
		}
		return 0, nil, nil
	}
	if err := os.MkdirAll(req.workDir, 0o755); err != nil {
		return 0, nil, err
	}
	binPath, err := compilePackageWithFallback(ctx, compileRequest{
		repoRoot:    req.repoRoot,
		pkgDir:      pkgDir,
		pkgPlans:    []*FilePlan{plan},
		maxMutantID: maxID,
		pkgWorkDir:  req.workDir,
	})
	if err != nil {
		return 0, nil, err
	}
	if binPath == "" {
		return 0, nil, nil
	}
	return executeFullFuncMutants(ctx, req, plan, pkgAbsDir, binPath)
}

func executeFullFuncMutants(
	ctx context.Context,
	req fullFuncRequest,
	plan *FilePlan,
	pkgAbsDir string,
	binPath string,
) (int, []rule.MutantDetail, error) {
	tracePath := filepath.Join(req.workDir, "baseline.trace")
	dur, _, err := runBaselineTestBinary(ctx, baselineRequest{
		binPath:   binPath,
		pkgAbsDir: pkgAbsDir,
		tracePath: tracePath,
	})
	if err != nil {
		return 0, nil, err
	}
	_, reached, err := ParsePolarityTraceFile(tracePath)
	if err != nil {
		return 0, nil, err
	}
	env := pkgExecEnv{pkgAbsDir: pkgAbsDir, binPath: binPath}
	jobs := buildReachedJobs([]*FilePlan{plan}, reached, env, computePerMutantTimeout(dur), nil)
	if len(jobs) > 0 {
		runMutantWorkerPool(ctx, jobs, req.workers, nil)
	}
	var details []rule.MutantDetail
	for _, m := range plan.Mutants {
		if m.Status == rule.MutantSurvived {
			details = append(details, m.ToDetail())
		}
	}
	return len(details), details, nil
}

func recordMutOutcome(
	res *Result,
	fm *astcheck.FileMetrics,
	fn astcheck.FuncMetrics,
	sample rule.Sample,
	survived []rule.MutantDetail,
) {
	kind := rule.ClassifyRatchet(sample)
	if kind == rule.ViolationNone {
		return
	}
	reportLine := fn.StartLine
	if len(survived) > 0 {
		reportLine = survived[0].Line
	}
	res.Findings = append(res.Findings, rule.Finding{
		Rule:             rule.RuleFuncSurvivingMutants,
		Kind:             kind,
		File:             fm.File,
		Line:             reportLine,
		EndLine:          fn.EndLine,
		Subject:          fn.DisplayName,
		Before:           sample.Before,
		HasBefore:        sample.HasBefore,
		After:            sample.After,
		Limit:            sample.Limit,
		Detail:           fmt.Sprintf("%d mutant(s) survived unit tests", sample.After),
		SurvivingMutants: survived,
	})
}
