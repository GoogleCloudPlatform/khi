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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type goFileInput struct {
	repoRoot      string
	relPath       string
	targetSrc     []byte
	baseSrc       []byte
	hasBase       bool
	changedLines  map[int]struct{}
	addedByPkg    map[string][]string
	movedOutTests map[string]movedGoTestBase
	movedInTests  map[string]movedGoTestBase
}

type movedGoTestBase struct {
	fset    *token.FileSet
	summary goTestFuncSummary
}

type goTestPkgContext struct {
	addedByPkg     map[string][]string
	movedOutByFile map[string]map[string]movedGoTestBase
	movedInByFile  map[string]map[string]movedGoTestBase
}

type goTestFuncSummary struct {
	name              string
	line              int
	endLine           int
	caseCount         int
	tableElems        int
	standaloneRuns    int
	cmpAssertCount    int
	scalarAssertCount int
	totalAsserts      int
	calledHelpers     map[string]bool
	decl              *ast.FuncDecl
}

func analyzeDeletedGoTestFile(
	req rule.AdvisoryRequest,
	relPath string,
	candidates []string,
	movedOut map[string]movedGoTestBase,
) []rule.QAItem {
	if items, handled := analyzeDeletedGoTestFuncs(req, relPath, candidates, movedOut); handled {
		return items
	}
	item := makeAdvisoryItem(checkTestRemoved, relPath, 1, filepath.Base(relPath), "test file deleted")
	item.Evidence = map[string]any{"deleted_file": relPath, "renamed_candidates": candidates}
	item.ReviewPrompt = "Verify whether this deleted test file was intentionally merged or removed with user confirmation."
	return []rule.QAItem{item}
}

func analyzeDeletedGoTestFuncs(
	req rule.AdvisoryRequest,
	relPath string,
	candidates []string,
	movedOut map[string]movedGoTestBase,
) ([]rule.QAItem, bool) {
	baseSrc, hasBase := readBaseBytes(req.ReadBaseFile, relPath)
	if !hasBase {
		return nil, false
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, baseSrc, 0)
	if err != nil {
		return nil, false
	}
	funcs := collectGoTestFuncSummaries(fset, file)
	if len(funcs) == 0 {
		return nil, false
	}
	var items []rule.QAItem
	for _, fn := range funcs {
		if _, moved := movedOut[fn.name]; moved {
			continue
		}
		item := makeAdvisoryItem(checkTestRemoved, relPath, fn.line, fn.name, fmt.Sprintf("test function %s removed with deleted file", fn.name))
		item.Evidence = map[string]any{"test_name": fn.name, "renamed_candidates": candidates}
		item.ReviewPrompt = "Verify whether this deleted test was intentionally superseded by an equivalent test or accidentally dropped."
		items = append(items, item)
	}
	return items, true
}

func extractAddedGoTestNames(relPath string, baseSrc []byte, targetSrc []byte) []string {
	targetFile, err := parser.ParseFile(token.NewFileSet(), relPath, targetSrc, 0)
	if err != nil {
		return nil
	}
	targetNames := collectTestFuncNames(targetFile)
	var baseNames map[string]bool
	if len(baseSrc) > 0 {
		if baseFile, err := parser.ParseFile(token.NewFileSet(), relPath, baseSrc, 0); err == nil {
			baseNames = collectTestFuncNames(baseFile)
		}
	}
	var added []string
	for name := range targetNames {
		if !baseNames[name] {
			added = append(added, name)
		}
	}
	slices.Sort(added)
	return added
}

func collectTestFuncNames(file *ast.File) map[string]bool {
	names := make(map[string]bool)
	if file == nil {
		return names
	}
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil || fd.Recv != nil {
			continue
		}
		name := fd.Name.Name
		if strings.HasPrefix(name, "Test") && name != "TestMain" {
			names[name] = true
		}
	}
	return names
}

func analyzeChangedGoFile(in goFileInput) []rule.QAItem {
	targetFset := token.NewFileSet()
	targetFile, err := parser.ParseFile(targetFset, in.relPath, in.targetSrc, parser.ParseComments)
	if err != nil {
		return nil
	}

	var baseFile *ast.File
	var baseFset *token.FileSet
	if in.hasBase {
		baseFset = token.NewFileSet()
		if bf, parseErr := parser.ParseFile(baseFset, in.relPath, in.baseSrc, parser.ParseComments); parseErr == nil {
			baseFile = bf
		}
	}

	var items []rule.QAItem
	items = append(items, detectGoCommentSuppressions(targetFset, targetFile, baseFset, baseFile, in)...)
	if !gotestutil.IsTargetGoTestFile(in.relPath) {
		return items
	}

	if item, ok := detectOrphanTestFile(in); ok {
		items = append(items, item)
	}
	if baseFile != nil || len(in.movedInTests) > 0 {
		items = append(items, detectGoTestDiffSignals(targetFset, targetFile, baseFset, baseFile, in)...)
		items = append(items, detectGoStressReductions(targetFset, targetFile, baseFset, baseFile, in)...)
	}
	items = append(items, detectGoTestSkips(targetFset, targetFile, baseFile, in)...)
	items = append(items, detectGoTestSleeps(targetFset, targetFile, baseFset, baseFile, in)...)
	items = append(items, detectGoTestDiscardCalls(targetFset, targetFile, baseFset, baseFile, in)...)
	return items
}

func detectGoTestDiffSignals(
	targetFset *token.FileSet,
	targetFile *ast.File,
	baseFset *token.FileSet,
	baseFile *ast.File,
	in goFileInput,
) []rule.QAItem {
	targetSummaries := collectGoTestFuncSummaries(targetFset, targetFile)
	targetByName := indexGoTestFuncs(targetSummaries)

	var baseSummaries []goTestFuncSummary
	if baseFile != nil {
		baseSummaries = collectGoTestFuncSummaries(baseFset, baseFile)
	}
	baseByName := indexGoTestFuncs(baseSummaries)

	addedInFile := extractAddedGoTestNames(in.relPath, in.baseSrc, in.targetSrc)
	pkgDir := filepath.ToSlash(filepath.Dir(in.relPath))
	candidates := mergeRenamedCandidates(addedInFile, in.addedByPkg[pkgDir])

	items := detectRemovedGoTests(in.relPath, baseSummaries, targetByName, in.movedOutTests, candidates)
	for _, targetFn := range targetSummaries {
		baseFn, exists := baseByName[targetFn.name]
		if !exists {
			moved, movedExists := in.movedInTests[targetFn.name]
			if !movedExists {
				continue
			}
			baseFn = moved.summary
		}
		items = append(items, compareGoTestPair(in.relPath, baseFn, targetFn)...)
	}
	return items
}

func detectRemovedGoTests(
	relPath string,
	baseSummaries []goTestFuncSummary,
	targetByName map[string]goTestFuncSummary,
	movedOut map[string]movedGoTestBase,
	candidates []string,
) []rule.QAItem {
	var items []rule.QAItem
	for _, baseFn := range baseSummaries {
		if _, exists := targetByName[baseFn.name]; exists {
			continue
		}
		if _, moved := movedOut[baseFn.name]; moved {
			continue
		}
		item := makeAdvisoryItem(checkTestRemoved, relPath, baseFn.line, baseFn.name, fmt.Sprintf("test function %s removed", baseFn.name))
		item.Evidence = map[string]any{"test_name": baseFn.name, "renamed_candidates": candidates}
		item.ReviewPrompt = "Verify whether this removed test was renamed/merged intentionally or dropped to silence failures."
		items = append(items, item)
	}
	return items
}

func compareGoTestPair(
	relPath string,
	baseFn goTestFuncSummary,
	targetFn goTestFuncSummary,
) []rule.QAItem {
	// Compare base and target AST summaries unconditionally so pure-deletion hunks and helper weakenings cannot evade detection.
	var items []rule.QAItem
	if targetFn.caseCount < baseFn.caseCount {
		item := makeAdvisoryItem(checkTestCasesReduced, relPath, targetFn.line, targetFn.name, fmt.Sprintf("test cases reduced from %d to %d", baseFn.caseCount, targetFn.caseCount))
		item.Evidence = map[string]any{"before_count": baseFn.caseCount, "after_count": targetFn.caseCount}
		item.ReviewPrompt = "Confirm whether test cases were consolidated equivalently or dropped to bypass redundancy/coverage pressure."
		items = append(items, item)
	}
	if item, weakened := compareGoTestAssertions(relPath, baseFn, targetFn); weakened {
		items = append(items, item)
	}
	return items
}

func compareGoTestAssertions(relPath string, baseFn goTestFuncSummary, targetFn goTestFuncSummary) (rule.QAItem, bool) {
	replacedCmp := min(
		max(baseFn.cmpAssertCount-targetFn.cmpAssertCount, 0),
		max(targetFn.scalarAssertCount-baseFn.scalarAssertCount, 0),
	)
	consolidated := isTableConsolidation(baseFn, targetFn)
	if targetFn.cmpAssertCount+replacedCmp < baseFn.cmpAssertCount && !(consolidated && (targetFn.cmpAssertCount > 0 || replacedCmp > 0)) {
		item := makeAdvisoryItem(checkTestAssertionWeakened, relPath, targetFn.line, targetFn.name, fmt.Sprintf("cmp.Diff/cmp.Equal assertions reduced from %d to %d", baseFn.cmpAssertCount, targetFn.cmpAssertCount))
		item.Evidence = map[string]any{"before_cmp_asserts": baseFn.cmpAssertCount, "after_cmp_asserts": targetFn.cmpAssertCount}
		item.ReviewPrompt = "Verify that removing cmp.Diff/cmp.Equal did not weaken output verification."
		return item, true
	}
	if targetFn.totalAsserts+replacedCmp < baseFn.totalAsserts && !hasNewHelperCall(baseFn.calledHelpers, targetFn.calledHelpers) && !consolidated {
		item := makeAdvisoryItem(checkTestAssertionWeakened, relPath, targetFn.line, targetFn.name, fmt.Sprintf("assertions reduced from %d to %d without helper extraction", baseFn.totalAsserts, targetFn.totalAsserts))
		item.Evidence = map[string]any{"before_asserts": baseFn.totalAsserts, "after_asserts": targetFn.totalAsserts}
		item.ReviewPrompt = "Verify that the reduced assertion count still checks every relevant output field and invariant."
		return item, true
	}
	return rule.QAItem{}, false
}

func isTableConsolidation(baseFn goTestFuncSummary, targetFn goTestFuncSummary) bool {
	if targetFn.tableElems <= baseFn.tableElems || targetFn.totalAsserts == 0 {
		return false
	}
	if baseFn.tableElems == 0 {
		return targetFn.tableElems >= 2
	}
	return targetFn.standaloneRuns < baseFn.standaloneRuns
}

func hasNewHelperCall(baseHelpers map[string]bool, targetHelpers map[string]bool) bool {
	for name := range targetHelpers {
		if !baseHelpers[name] {
			return true
		}
	}
	return false
}

func collectGoTestFuncSummaries(fset *token.FileSet, file *ast.File) []goTestFuncSummary {
	idx := newTestFileIndex(file)
	var out []goTestFuncSummary
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil || fd.Recv != nil {
			continue
		}
		name := fd.Name.Name
		if !strings.HasPrefix(name, "Test") || name == "TestMain" {
			continue
		}
		tableElems := countTableElements(idx, fd)
		standaloneRuns := countStandaloneTRuns(fd.Body)
		metrics := inspectGoTestBodyMetrics(idx, fd)
		out = append(out, goTestFuncSummary{
			name:              name,
			line:              fset.Position(fd.Pos()).Line,
			endLine:           fset.Position(fd.End()).Line,
			caseCount:         tableElems + standaloneRuns,
			tableElems:        tableElems,
			standaloneRuns:    standaloneRuns,
			cmpAssertCount:    metrics.cmpAsserts,
			scalarAssertCount: metrics.scalarAsserts,
			totalAsserts:      metrics.totalAsserts,
			calledHelpers:     metrics.calledHelpers,
			decl:              fd,
		})
	}
	return out
}

func indexGoTestFuncs(summaries []goTestFuncSummary) map[string]goTestFuncSummary {
	out := make(map[string]goTestFuncSummary, len(summaries))
	for _, s := range summaries {
		out[s.name] = s
	}
	return out
}

func detectOrphanTestFile(in goFileInput) (rule.QAItem, bool) {
	if in.hasBase || in.repoRoot == "" {
		return rule.QAItem{}, false
	}
	baseName := path.Base(in.relPath)
	if isExcludedOrphanTestName(baseName) {
		return rule.QAItem{}, false
	}
	siblingProd := strings.TrimSuffix(in.relPath, "_test.go") + ".go"
	if _, err := os.Stat(filepath.Join(in.repoRoot, filepath.FromSlash(siblingProd))); err == nil {
		return rule.QAItem{}, false
	}
	item := makeAdvisoryItem(checkOrphanTestFile, in.relPath, 1, baseName, fmt.Sprintf("new test file %s has no matching %s", baseName, path.Base(siblingProd)))
	item.Evidence = map[string]any{"expected_sibling": siblingProd}
	item.ReviewPrompt = "Confirm whether this standalone *_test.go file is intentional (integration/contract suite) rather than splitting tests away from <A>.go to game test_file_code_lines."
	return item, true
}

func isExcludedOrphanTestName(baseName string) bool {
	return baseName == "interface_test.go" || slices.ContainsFunc([]string{"_integration_test.go", "_script_test.go", "_contract_test.go"}, func(suffix string) bool {
		return strings.HasSuffix(baseName, suffix)
	})
}
