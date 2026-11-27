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
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// Options configures Stage-2 differential coverage and func_uncovered_lines evaluation.
type Options struct {
	// RepoRoot is the repository root directory path.
	RepoRoot string
	// ModulePath is the Go module path declared in go.mod.
	ModulePath string
	// WorkDir is the temporary working directory for coverage profiles.
	WorkDir string
	// Full enables full-repository coverage in addition to touched-package coverage.
	Full bool
	// PrecomputedProfilePath points to an optional existing Go coverprofile file.
	PrecomputedProfilePath string
	// Thresholds specifies quality limits for coverage rules.
	Thresholds rule.Thresholds
	// ChangedProdLinesByFile maps changed production file paths to modified 1-based line sets.
	ChangedProdLinesByFile map[string]map[int]struct{}
	// ASTResult holds Stage-1 AST analysis results shared with Stage-2 coverage checks.
	ASTResult *astcheck.Result
	// ReadTargetFile reads raw bytes for a target working-tree file.
	ReadTargetFile func(relPath string) ([]byte, error)
	// ErrOut receives diagnostic progress and command error output.
	ErrOut io.Writer
}

// Result holds coverage summary metrics, func_uncovered_lines findings, and parsed CoverBlocks for mutcheck.
type Result struct {
	// Summary holds aggregated patch and overall statement coverage counts.
	Summary rule.CoverageSummary
	// Findings contains func_uncovered_lines advisory violations.
	Findings []rule.Finding
	// CoverBlocksByFile maps file paths to parsed Go coverprofile blocks.
	CoverBlocksByFile map[string][]gotestutil.CoverBlock
	// HasCoverProfile reports whether coverage profile data was loaded.
	HasCoverProfile bool
}

type targetCoverLoaded struct {
	blocks         map[string][]gotestutil.CoverBlock
	hasProfile     bool
	includeOverall bool
}

// Evaluate runs target coverage and emits func_uncovered_lines advisories for changed uncovered lines.
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

	if err := evaluateTargetFunctions(opts, loaded.blocks, res); err != nil {
		return nil, err
	}
	return res, nil
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
) error {
	limit := opts.Thresholds.Limit(rule.RuleFuncUncoveredLines)
	for _, relFile := range sortedFiles(opts.ChangedProdLinesByFile) {
		if err := evaluateTargetFile(opts, relFile, blocksByFile[relFile], limit, res); err != nil {
			return err
		}
	}
	return nil
}

func evaluateTargetFile(
	opts Options,
	relFile string,
	blocks []gotestutil.CoverBlock,
	limit int,
	res *Result,
) error {
	src, err := opts.ReadTargetFile(relFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read %s: %w", relFile, err)
	}
	changedSet := opts.ChangedProdLinesByFile[relFile]
	fa := AnalyzeFileLines(relFile, src, blocks)
	cov, exec := MeasurePatchTotals(fa, changedSet)
	res.Summary.PatchCovered += cov
	res.Summary.PatchExecutable += exec

	fm := opts.ASTResult.Files[relFile]
	if fm == nil {
		return nil
	}
	for _, fn := range opts.ASTResult.TouchedFuncs[relFile] {
		if finding, ok := evaluateSingleTargetFunc(fm.File, fa, fn, changedSet, limit); ok {
			res.Findings = append(res.Findings, finding)
		}
	}
	return nil
}

func evaluateSingleTargetFunc(
	relFile string,
	fa FileAnalysis,
	fn astcheck.FuncMetrics,
	changedSet map[int]struct{},
	limit int,
) (rule.Finding, bool) {
	after, spans := MeasureFuncUncovered(relFile, fa, fn, changedSet)
	if after <= limit {
		return rule.Finding{}, false
	}
	reportLine := fn.StartLine
	if len(spans) > 0 {
		reportLine = spans[0].StartLine
	}
	return rule.Finding{
		Rule:           rule.RuleFuncUncoveredLines,
		Kind:           rule.ViolationAdvisory,
		File:           relFile,
		Line:           reportLine,
		EndLine:        fn.EndLine,
		Subject:        fn.DisplayName,
		After:          after,
		Limit:          limit,
		Detail:         fmt.Sprintf("%d substantive uncovered line(s) across %d span(s)", after, len(spans)),
		UncoveredSpans: spans,
	}, true
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
