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

package advisorycheck

import (
	"cmp"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

const (
	providerName = "advisorycheck"

	checkTestRemoved           = "test_removed"
	checkTestCasesReduced      = "test_cases_reduced"
	checkTestAssertionWeakened = "test_assertion_weakened"
	checkTestSkipAdded         = "test_skip_added"
	checkTestStressReduced     = "test_stress_reduced"
	checkTestSleepAdded        = "test_sleep_added"
	checkSuppressionAdded      = "suppression_added"
	checkTSDoubleCastAdded     = "ts_double_cast_added"
	checkOrphanTestFile        = "orphan_test_file"
)

// Provider evaluates fast, offline differential anti-Goodhart advisory signals across Go and TypeScript changes.
type Provider struct{}

type movedGoTestOrigin struct {
	file    string
	fset    *token.FileSet
	summary goTestFuncSummary
}

type goTestPkgAccumulator struct {
	addedByPkg     map[string][]string
	additionsByPkg map[string]map[string][]string
	removalsByPkg  map[string]map[string][]movedGoTestOrigin
}

// NewProvider creates a Provider registered with qa-fast-advisory.
func NewProvider() *Provider {
	return &Provider{}
}

// Name returns the unique provider source identifier.
func (p *Provider) Name() string {
	return providerName
}

// Speed returns rule.SpeedFast so advisorycheck executes offline in fast QA runs.
func (p *Provider) Speed() rule.Speed {
	return rule.SpeedFast
}

// Evaluate inspects changed and deleted Go/TypeScript files against base revision content and emits advisory items.
func (p *Provider) Evaluate(ctx context.Context, req rule.AdvisoryRequest) ([]rule.QAItem, *rule.SkippedSource, error) {
	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}

	var items []rule.QAItem
	goCtx := collectGoTestPkgContext(req)
	tsAddedTests := collectAddedTSTestsByDir(req)

	for _, relPath := range sortedUniqueStrings(req.DeletedFiles) {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		items = append(items, evaluateDeletedFile(req, relPath, goCtx, tsAddedTests)...)
	}

	for _, relPath := range sortedUniqueStrings(req.ChangedFiles) {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		items = append(items, evaluateChangedFile(req, relPath, goCtx, tsAddedTests)...)
	}

	sortAndDeduplicateItems(&items)
	return items, nil, nil
}

func (a *goTestPkgAccumulator) addAddition(pkgDir string, name string, relPath string) {
	a.addedByPkg[pkgDir] = append(a.addedByPkg[pkgDir], name)
	byName := a.additionsByPkg[pkgDir]
	if byName == nil {
		byName = make(map[string][]string)
		a.additionsByPkg[pkgDir] = byName
	}
	byName[name] = append(byName[name], relPath)
}

func (a *goTestPkgAccumulator) addRemoval(pkgDir string, name string, origin movedGoTestOrigin) {
	byName := a.removalsByPkg[pkgDir]
	if byName == nil {
		byName = make(map[string][]movedGoTestOrigin)
		a.removalsByPkg[pkgDir] = byName
	}
	byName[name] = append(byName[name], origin)
}

func evaluateDeletedFile(
	req rule.AdvisoryRequest,
	relPath string,
	goCtx goTestPkgContext,
	tsAddedTests map[string][]string,
) []rule.QAItem {
	norm := normalizePath(relPath)
	if gotestutil.IsTargetGoTestFile(norm) {
		pkgDir := filepath.ToSlash(filepath.Dir(norm))
		return analyzeDeletedGoTestFile(req, norm, goCtx.addedByPkg[pkgDir], goCtx.movedOutByFile[norm])
	}
	if gotestutil.IsTargetTSTestFile(norm) {
		return analyzeDeletedTSTestFile(req, norm, tsAddedTests)
	}
	return nil
}

func evaluateChangedFile(
	req rule.AdvisoryRequest,
	relPath string,
	goCtx goTestPkgContext,
	tsAddedTests map[string][]string,
) []rule.QAItem {
	norm := normalizePath(relPath)
	targetSrc, ok := readTargetBytes(req.RepoRoot, norm)
	if !ok {
		return nil
	}
	baseSrc, hasBase := readBaseBytes(req.ReadBaseFile, norm)
	changedLines := req.ChangedLines[norm]

	if gotestutil.IsTargetGoTestFile(norm) || gotestutil.IsTargetGoFile(norm) {
		return analyzeChangedGoFile(goFileInput{
			repoRoot:      req.RepoRoot,
			relPath:       norm,
			targetSrc:     targetSrc,
			baseSrc:       baseSrc,
			hasBase:       hasBase,
			changedLines:  changedLines,
			addedByPkg:    goCtx.addedByPkg,
			movedOutTests: goCtx.movedOutByFile[norm],
			movedInTests:  goCtx.movedInByFile[norm],
		})
	}
	if gotestutil.IsTargetTSTestFile(norm) || gotestutil.IsTargetProductionTSFile(norm) {
		return analyzeChangedTSFile(tsFileInput{
			relPath:      norm,
			targetSrc:    targetSrc,
			baseSrc:      baseSrc,
			hasBase:      hasBase,
			changedLines: changedLines,
			addedByDir:   tsAddedTests,
		})
	}
	return nil
}

func collectGoTestPkgContext(req rule.AdvisoryRequest) goTestPkgContext {
	acc := &goTestPkgAccumulator{
		addedByPkg:     make(map[string][]string),
		additionsByPkg: make(map[string]map[string][]string),
		removalsByPkg:  make(map[string]map[string][]movedGoTestOrigin),
	}
	for _, rel := range sortedUniqueStrings(req.ChangedFiles) {
		collectChangedGoTestPkgEntry(req, normalizePath(rel), acc)
	}
	for _, rel := range sortedUniqueStrings(req.DeletedFiles) {
		collectDeletedGoTestPkgEntry(req, normalizePath(rel), acc)
	}
	for pkgDir, names := range acc.addedByPkg {
		slices.Sort(names)
		acc.addedByPkg[pkgDir] = slices.Compact(names)
	}
	movedOutByFile, movedInByFile := resolveUnambiguousGoTestMoves(acc)
	return goTestPkgContext{
		addedByPkg:     acc.addedByPkg,
		movedOutByFile: movedOutByFile,
		movedInByFile:  movedInByFile,
	}
}

func collectChangedGoTestPkgEntry(req rule.AdvisoryRequest, norm string, acc *goTestPkgAccumulator) {
	if !gotestutil.IsTargetGoTestFile(norm) {
		return
	}
	targetSrc, ok := readTargetBytes(req.RepoRoot, norm)
	if !ok {
		return
	}
	targetFile, err := parser.ParseFile(token.NewFileSet(), norm, targetSrc, 0)
	if err != nil {
		return
	}
	targetNames := collectTestFuncNames(targetFile)
	pkgDir := filepath.ToSlash(filepath.Dir(norm))
	baseNames := recordChangedGoTestRemovals(req, norm, pkgDir, targetNames, acc)
	for name := range targetNames {
		if !baseNames[name] {
			acc.addAddition(pkgDir, name, norm)
		}
	}
}

func recordChangedGoTestRemovals(
	req rule.AdvisoryRequest,
	norm string,
	pkgDir string,
	targetNames map[string]bool,
	acc *goTestPkgAccumulator,
) map[string]bool {
	baseSrc, hasBase := readBaseBytes(req.ReadBaseFile, norm)
	if !hasBase {
		return nil
	}
	baseFset := token.NewFileSet()
	baseFile, err := parser.ParseFile(baseFset, norm, baseSrc, 0)
	if err != nil {
		return nil
	}
	baseNames := make(map[string]bool)
	for _, fn := range collectGoTestFuncSummaries(baseFset, baseFile) {
		baseNames[fn.name] = true
		if !targetNames[fn.name] {
			acc.addRemoval(pkgDir, fn.name, movedGoTestOrigin{file: norm, fset: baseFset, summary: fn})
		}
	}
	return baseNames
}

func collectDeletedGoTestPkgEntry(req rule.AdvisoryRequest, norm string, acc *goTestPkgAccumulator) {
	if !gotestutil.IsTargetGoTestFile(norm) {
		return
	}
	baseSrc, hasBase := readBaseBytes(req.ReadBaseFile, norm)
	if !hasBase {
		return
	}
	baseFset := token.NewFileSet()
	baseFile, err := parser.ParseFile(baseFset, norm, baseSrc, 0)
	if err != nil {
		return
	}
	pkgDir := filepath.ToSlash(filepath.Dir(norm))
	for _, fn := range collectGoTestFuncSummaries(baseFset, baseFile) {
		acc.addRemoval(pkgDir, fn.name, movedGoTestOrigin{file: norm, fset: baseFset, summary: fn})
	}
}

func resolveUnambiguousGoTestMoves(acc *goTestPkgAccumulator) (map[string]map[string]movedGoTestBase, map[string]map[string]movedGoTestBase) {
	movedOutByFile := make(map[string]map[string]movedGoTestBase)
	movedInByFile := make(map[string]map[string]movedGoTestBase)
	for pkgDir, remByName := range acc.removalsByPkg {
		addByName := acc.additionsByPkg[pkgDir]
		for name, origins := range remByName {
			targets := addByName[name]
			if len(origins) != 1 || len(targets) != 1 || origins[0].file == targets[0] {
				continue
			}
			moved := movedGoTestBase{fset: origins[0].fset, summary: origins[0].summary}
			setFileMovedTest(movedOutByFile, origins[0].file, name, moved)
			setFileMovedTest(movedInByFile, targets[0], name, moved)
		}
	}
	return movedOutByFile, movedInByFile
}

func setFileMovedTest(byFile map[string]map[string]movedGoTestBase, relPath string, name string, moved movedGoTestBase) {
	perFile := byFile[relPath]
	if perFile == nil {
		perFile = make(map[string]movedGoTestBase)
		byFile[relPath] = perFile
	}
	perFile[name] = moved
}

func collectAddedTSTestsByDir(req rule.AdvisoryRequest) map[string][]string {
	out := make(map[string][]string)
	for _, rel := range sortedUniqueStrings(req.ChangedFiles) {
		norm := normalizePath(rel)
		if !gotestutil.IsTargetTSTestFile(norm) {
			continue
		}
		targetSrc, ok := readTargetBytes(req.RepoRoot, norm)
		if !ok {
			continue
		}
		baseSrc, _ := readBaseBytes(req.ReadBaseFile, norm)
		added := extractAddedTSTestNames(baseSrc, targetSrc)
		if len(added) > 0 {
			dir := filepath.ToSlash(filepath.Dir(norm))
			out[dir] = append(out[dir], added...)
		}
	}
	for k := range out {
		slices.Sort(out[k])
		out[k] = slices.Compact(out[k])
	}
	return out
}

func readTargetBytes(repoRoot string, relPath string) ([]byte, bool) {
	if repoRoot == "" {
		return nil, false
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
	if err != nil {
		return nil, false
	}
	return data, true
}

func readBaseBytes(fn func(string) ([]byte, bool, error), relPath string) ([]byte, bool) {
	if fn == nil {
		return nil, false
	}
	data, ok, err := fn(relPath)
	if err != nil || !ok {
		return nil, false
	}
	return data, true
}

func normalizePath(relPath string) string {
	return strings.TrimPrefix(filepath.ToSlash(relPath), "./")
}

func sortedUniqueStrings(in []string) []string {
	out := slices.Clone(in)
	slices.Sort(out)
	return slices.Compact(out)
}

func makeAdvisoryItem(check string, relPath string, line int, symbol string, summary string) rule.QAItem {
	return rule.QAItem{
		ID:      fmt.Sprintf("%s:%s:%s:%d:%s", providerName, check, relPath, max(line, 1), symbol),
		Source:  providerName,
		Check:   check,
		File:    relPath,
		Line:    max(line, 1),
		Symbol:  symbol,
		Summary: summary,
		Detail:  summary,
	}
}

func sortAndDeduplicateItems(items *[]rule.QAItem) {
	slices.SortFunc(*items, func(a, b rule.QAItem) int {
		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.Check, b.Check),
			cmp.Compare(a.Symbol, b.Symbol),
			cmp.Compare(a.ID, b.ID),
		)
	})
	*items = slices.CompactFunc(*items, func(a, b rule.QAItem) bool {
		return a.ID == b.ID && a.Summary == b.Summary
	})
}
