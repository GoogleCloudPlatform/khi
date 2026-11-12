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

package covcheck

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/overlay"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

// Options configures Stage-2 differential coverage and func_uncovered_lines evaluation.
type Options struct {
	RepoRoot               string
	ModulePath             string
	WorkDir                string
	Full                   bool
	PrecomputedProfilePath string
	Thresholds             rule.Thresholds
	ChangedProdLinesByFile map[string]map[int]struct{}
	ASTResult              *astcheck.Result
	ReadTargetFile         func(relPath string) ([]byte, error)
	EnsureBaseOverlay      func() (*overlay.Snapshot, error)
	ErrOut                 io.Writer
}

// Result holds coverage summary metrics, func_uncovered_lines findings, and parsed CoverBlocks for mutcheck.
type Result struct {
	Summary           rule.CoverageSummary
	Findings          []rule.Finding
	Suppressed        []rule.SuppressedItem
	CoverBlocksByFile map[string][]gotestutil.CoverBlock
	HasCoverProfile   bool
}

type pendingBaseCov struct {
	relFile  string
	fm       *astcheck.FileMetrics
	fn       astcheck.FuncMetrics
	baseFile string
	baseFn   astcheck.FuncMetrics
	after    int
	spans    []rule.UncoveredSpan
}

// Evaluate runs target coverage, evaluates func_uncovered_lines, and lazily runs base -overlay coverage when needed.
func Evaluate(ctx context.Context, opts Options) (*Result, error) {
	loaded, err := loadTargetBlocks(ctx, opts)
	if err != nil {
		return nil, err
	}

	res := &Result{
		Summary: rule.CoverageSummary{
			Enabled:    true,
			HasOverall: loaded.includeOverall,
		},
		CoverBlocksByFile: loaded.blocks,
		HasCoverProfile:   loaded.hasProfile,
	}
	if loaded.includeOverall {
		accumulateOverallStatements(&res.Summary, loaded.blocks)
	}

	limit := opts.Thresholds.Limit(rule.RuleFuncUncoveredLines)
	pending, err := evaluateTargetFunctions(opts, loaded.blocks, res)
	if err != nil {
		return nil, err
	}

	if len(pending) > 0 && opts.EnsureBaseOverlay != nil {
		if err := evaluateLazyBaseFunctions(ctx, opts, pending, limit, res); err != nil {
			return nil, err
		}
	}
	return res, nil
}

type targetCoverLoaded struct {
	blocks         map[string][]gotestutil.CoverBlock
	hasProfile     bool
	includeOverall bool
}

func loadTargetBlocks(ctx context.Context, opts Options) (targetCoverLoaded, error) {
	if opts.PrecomputedProfilePath != "" {
		if _, err := os.Stat(opts.PrecomputedProfilePath); err != nil {
			return targetCoverLoaded{}, fmt.Errorf("failed to open coverprofile %s: %w", opts.PrecomputedProfilePath, err)
		}
		blocks, err := gotestutil.LoadCoverProfileIfPresent(opts.PrecomputedProfilePath, opts.ModulePath)
		if err != nil {
			return targetCoverLoaded{}, err
		}
		return targetCoverLoaded{blocks: blocks, hasProfile: true, includeOverall: true}, nil
	}
	prodFiles := sortedFiles(opts.ChangedProdLinesByFile)
	pkgs := PlanPackages(PlanRequest{
		RepoRoot:  opts.RepoRoot,
		ProdFiles: prodFiles,
		Full:      opts.Full,
	})
	if len(pkgs) == 0 {
		return targetCoverLoaded{blocks: map[string][]gotestutil.CoverBlock{}, includeOverall: opts.Full}, nil
	}
	blocks, err := RunCoverProfile(ctx, RunRequest{
		RepoRoot:    opts.RepoRoot,
		ModulePath:  opts.ModulePath,
		Packages:    pkgs,
		WorkDir:     opts.WorkDir,
		ProfileName: "target.coverprofile",
		ErrOut:      opts.ErrOut,
	})
	if err != nil {
		return targetCoverLoaded{}, err
	}
	return targetCoverLoaded{blocks: blocks, hasProfile: true, includeOverall: opts.Full}, nil
}

func accumulateOverallStatements(sum *rule.CoverageSummary, blocksByFile map[string][]gotestutil.CoverBlock) {
	for _, blocks := range blocksByFile {
		for _, b := range blocks {
			sum.OverallStatements += b.NumStatements
			if b.Count > 0 {
				sum.OverallCovered += b.NumStatements
			}
		}
	}
}

func evaluateTargetFunctions(
	opts Options,
	blocksByFile map[string][]gotestutil.CoverBlock,
	res *Result,
) ([]pendingBaseCov, error) {
	var pending []pendingBaseCov
	for _, relFile := range sortedFiles(opts.ChangedProdLinesByFile) {
		filePending, err := evaluateTargetFile(opts, relFile, blocksByFile[relFile], res)
		if err != nil {
			return nil, err
		}
		pending = append(pending, filePending...)
	}
	return pending, nil
}

func evaluateTargetFile(
	opts Options,
	relFile string,
	blocks []gotestutil.CoverBlock,
	res *Result,
) ([]pendingBaseCov, error) {
	src, err := opts.ReadTargetFile(relFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", relFile, err)
	}
	fa := AnalyzeFileLines(relFile, src, blocks)
	cov, exec := MeasurePatchTotals(fa, opts.ChangedProdLinesByFile[relFile])
	res.Summary.PatchCovered += cov
	res.Summary.PatchExecutable += exec

	fm := opts.ASTResult.Files[relFile]
	if fm == nil {
		return nil, nil
	}
	var pending []pendingBaseCov
	for _, fn := range opts.ASTResult.TouchedFuncs[relFile] {
		if p, ok := evaluateSingleTargetFunc(opts, fm, fa, fn, res); ok {
			pending = append(pending, p)
		}
	}
	return pending, nil
}

func evaluateSingleTargetFunc(
	opts Options,
	fm *astcheck.FileMetrics,
	fa FileAnalysis,
	fn astcheck.FuncMetrics,
	res *Result,
) (pendingBaseCov, bool) {
	limit := opts.Thresholds.Limit(rule.RuleFuncUncoveredLines)
	after, spans := MeasureFuncUncovered(fm.File, fa, fn, fm.Directives)
	if after <= limit {
		return pendingBaseCov{}, false
	}
	baseFile, baseFn, hasBase := opts.ASTResult.Base.LookupFuncLocation(fm.File, fn.Key)
	if !hasBase || opts.EnsureBaseOverlay == nil {
		recordCovOutcome(res, fm, fn, rule.Sample{After: after, Limit: limit}, spans)
		return pendingBaseCov{}, false
	}
	return pendingBaseCov{
		relFile:  fm.File,
		fm:       fm,
		fn:       fn,
		baseFile: baseFile,
		baseFn:   baseFn,
		after:    after,
		spans:    spans,
	}, true
}

func evaluateLazyBaseFunctions(
	ctx context.Context,
	opts Options,
	pending []pendingBaseCov,
	limit int,
	res *Result,
) error {
	snap, err := opts.EnsureBaseOverlay()
	if err != nil {
		return err
	}
	basePkgs := collectPendingBasePackages(snap.RepoRoot(), pending)
	baseBlocks := runBaseCoverBlocks(ctx, opts, snap.RepoRoot(), basePkgs)
	res.Summary.BaseOverlayRuns++

	for _, item := range pending {
		before := measureBaseFuncUncovered(opts, snap, baseBlocks, item)
		sample := rule.Sample{
			After:     item.after,
			Limit:     limit,
			Before:    before,
			HasBefore: true,
		}
		recordCovOutcome(res, item.fm, item.fn, sample, item.spans)
	}
	return nil
}

func runBaseCoverBlocks(
	ctx context.Context,
	opts Options,
	baseRoot string,
	basePkgs []string,
) map[string][]gotestutil.CoverBlock {
	blocks, err := RunCoverProfile(ctx, RunRequest{
		RepoRoot:    baseRoot,
		ModulePath:  opts.ModulePath,
		Packages:    basePkgs,
		WorkDir:     opts.WorkDir,
		ProfileName: "base.coverprofile",
	})
	if err == nil {
		return blocks
	}
	merged := make(map[string][]gotestutil.CoverBlock)
	for i, pkg := range basePkgs {
		sub, subErr := RunCoverProfile(ctx, RunRequest{
			RepoRoot:    baseRoot,
			ModulePath:  opts.ModulePath,
			Packages:    []string{pkg},
			WorkDir:     opts.WorkDir,
			ProfileName: fmt.Sprintf("base_%d.coverprofile", i),
		})
		if subErr == nil {
			for k, v := range sub {
				merged[k] = v
			}
		}
	}
	return merged
}

func measureBaseFuncUncovered(
	opts Options,
	snap *overlay.Snapshot,
	baseBlocks map[string][]gotestutil.CoverBlock,
	item pendingBaseCov,
) int {
	baseSrc, ok := snap.BaseSource(item.baseFile)
	if !ok {
		return 0
	}
	fa := AnalyzeFileLines(item.baseFile, baseSrc, baseBlocks[item.baseFile])
	var baseDirectives []rule.Directive
	if baseFM, hasFM := opts.ASTResult.Base.FileMetrics(item.baseFile); hasFM {
		baseDirectives = baseFM.Directives
	}
	uncov, _ := MeasureFuncUncovered(item.baseFile, fa, item.baseFn, baseDirectives)
	return uncov
}

func recordCovOutcome(
	res *Result,
	fm *astcheck.FileMetrics,
	fn astcheck.FuncMetrics,
	sample rule.Sample,
	spans []rule.UncoveredSpan,
) {
	kind := rule.ClassifyRatchet(sample)
	if kind == rule.ViolationNone {
		return
	}
	if kind == rule.ViolationHardLimit {
		if d := rule.MatchDeclDirective(fm.Directives, rule.RuleFuncUncoveredLines, fn.StartLine); d != nil {
			res.Suppressed = append(res.Suppressed, rule.SuppressedItem{
				Rule:     rule.RuleFuncUncoveredLines,
				File:     fm.File,
				Line:     fn.StartLine,
				Subject:  fn.DisplayName,
				Category: d.Category,
				Reason:   d.Reason,
				After:    sample.After,
				Limit:    sample.Limit,
			})
			return
		}
	}
	reportLine := fn.StartLine
	if len(spans) > 0 {
		reportLine = spans[0].StartLine
	}
	res.Findings = append(res.Findings, rule.Finding{
		Rule:           rule.RuleFuncUncoveredLines,
		Kind:           kind,
		File:           fm.File,
		Line:           reportLine,
		EndLine:        fn.EndLine,
		Subject:        fn.DisplayName,
		Before:         sample.Before,
		HasBefore:      sample.HasBefore,
		After:          sample.After,
		Limit:          sample.Limit,
		Detail:         fmt.Sprintf("%d substantive uncovered line(s) across %d span(s)", sample.After, len(spans)),
		UncoveredSpans: spans,
	})
}

func collectPendingBasePackages(baseRoot string, pending []pendingBaseCov) []string {
	var files []string
	for _, p := range pending {
		files = append(files, p.baseFile)
	}
	return PlanPackages(PlanRequest{RepoRoot: baseRoot, ProdFiles: files})
}

func sortedFiles(m map[string]map[int]struct{}) []string {
	var out []string
	for f, lines := range m {
		if len(lines) > 0 {
			out = append(out, f)
		}
	}
	slices.Sort(out)
	return out
}
