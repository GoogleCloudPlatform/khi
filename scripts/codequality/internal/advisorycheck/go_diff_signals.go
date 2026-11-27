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
	"go/ast"
	"go/token"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type lineSubject struct {
	line    int
	subject string
	funcKey string
}

type countedLineSignalSpec struct {
	check        string
	summary      string
	reviewPrompt string
	collectFunc  func(fset *token.FileSet, fd *ast.FuncDecl) []lineSubject
}

type countedLineBaselines struct {
	baseCount        int
	baseFuncCounts   map[string]int
	movedBaseCounts  map[string]int
	targetFuncCounts map[string]int
}

func detectGoTestSkips(
	targetFset *token.FileSet,
	targetFile *ast.File,
	baseFile *ast.File,
	in goFileInput,
) []rule.QAItem {
	return detectCountedGoLineSignals(targetFset, targetFile, baseFile, in, countedLineSignalSpec{
		check:        checkTestSkipAdded,
		summary:      "newly added test skip outside testing.Short()",
		reviewPrompt: "Verify whether skipping this test is justified or masks a failing test case.",
		collectFunc:  collectFuncSkipsOutsideShort,
	})
}

func collectFuncSkipsOutsideShort(fset *token.FileSet, fd *ast.FuncDecl) []lineSubject {
	var out []lineSubject
	if fd == nil || fd.Body == nil {
		return out
	}
	key := funcDeclKey(fd)
	walkSkippingShortIfs(fd.Body, func(call *ast.CallExpr) {
		if isGoSkipCall(call) {
			ln := 1
			if fset != nil {
				ln = fset.Position(call.Pos()).Line
			}
			out = append(out, lineSubject{line: ln, subject: fd.Name.Name, funcKey: key})
		}
	})
	return out
}

func walkSkippingShortIfs(node ast.Node, visitCall func(*ast.CallExpr)) {
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if ifStmt, ok := n.(*ast.IfStmt); ok && isTestingShortCond(ifStmt.Cond) {
			if ifStmt.Else != nil {
				walkSkippingShortIfs(ifStmt.Else, visitCall)
			}
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok {
			visitCall(call)
		}
		return true
	})
}

func isTestingShortCond(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Short" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "testing"
}

func isGoSkipCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "Skip", "Skipf", "SkipNow":
		return true
	default:
		return false
	}
}

func detectGoTestSleeps(
	targetFset *token.FileSet,
	targetFile *ast.File,
	_ *token.FileSet,
	baseFile *ast.File,
	in goFileInput,
) []rule.QAItem {
	return detectCountedGoLineSignals(targetFset, targetFile, baseFile, in, countedLineSignalSpec{
		check:        checkTestSleepAdded,
		summary:      "newly added time.Sleep in test file",
		reviewPrompt: "Verify whether time.Sleep is genuinely required (e.g. TTL test) or should use deterministic synchronization.",
		collectFunc:  collectFuncTimeSleepCalls,
	})
}

func detectCountedGoLineSignals(
	targetFset *token.FileSet,
	targetFile *ast.File,
	baseFile *ast.File,
	in goFileInput,
	spec countedLineSignalSpec,
) []rule.QAItem {
	targetEntries := collectFileLineSubjects(targetFset, targetFile, spec.collectFunc)
	targetFuncCounts := countLineSubjectsByKey(targetEntries)
	baseFuncCounts, baseCount := collectRemainingBaseCreditCounts(baseFile, in.movedOutTests, targetFuncCounts, spec.collectFunc)
	movedBaseCounts, movedCredit := collectMovedCreditCounts(in.movedInTests, targetFuncCounts, spec.collectFunc)
	baseCount += movedCredit

	if len(targetEntries) <= baseCount {
		return nil
	}
	baselines := countedLineBaselines{
		baseCount:        baseCount,
		baseFuncCounts:   baseFuncCounts,
		movedBaseCounts:  movedBaseCounts,
		targetFuncCounts: targetFuncCounts,
	}
	return buildCountedGoLineItems(targetEntries, in, spec, baselines)
}

func buildCountedGoLineItems(
	targetEntries []lineSubject,
	in goFileInput,
	spec countedLineSignalSpec,
	baselines countedLineBaselines,
) []rule.QAItem {
	inScopeByKey := make(map[string][]lineSubject)
	allByKey := make(map[string][]lineSubject)
	var keyOrder []string
	for _, entry := range targetEntries {
		if _, seen := allByKey[entry.funcKey]; !seen {
			keyOrder = append(keyOrder, entry.funcKey)
		}
		allByKey[entry.funcKey] = append(allByKey[entry.funcKey], entry)
		if isLineInScope(entry.line, in.changedLines) {
			inScopeByKey[entry.funcKey] = append(inScopeByKey[entry.funcKey], entry)
		}
	}
	var items []rule.QAItem
	for _, key := range keyOrder {
		if isUnchangedSubject(key, baselines) {
			continue
		}
		excess := baselines.targetFuncCounts[key] - (baselines.baseFuncCounts[key] + baselines.movedBaseCounts[key])
		selected := selectExcessLineSubjects(allByKey[key], inScopeByKey[key], excess)
		for _, entry := range selected {
			item := makeAdvisoryItem(spec.check, in.relPath, entry.line, entry.subject, spec.summary)
			item.Evidence = map[string]any{"before_count": baselines.baseCount, "after_count": len(targetEntries)}
			item.ReviewPrompt = spec.reviewPrompt
			items = append(items, item)
		}
	}
	return items
}

func selectExcessLineSubjects(all []lineSubject, inScope []lineSubject, excess int) []lineSubject {
	if len(inScope) >= excess {
		return inScope
	}
	// Git LCS diff can align lines moved from an adjacent gutted test as unchanged context; top up to excess.
	selected := append([]lineSubject(nil), inScope...)
	inScopeLines := make(map[int]bool, len(inScope))
	for _, e := range inScope {
		inScopeLines[e.line] = true
	}
	for _, e := range all {
		if len(selected) >= excess {
			break
		}
		if !inScopeLines[e.line] {
			selected = append(selected, e)
		}
	}
	return selected
}

func collectRemainingBaseCreditCounts(
	baseFile *ast.File,
	movedOutTests map[string]movedGoTestBase,
	targetFuncCounts map[string]int,
	collectFunc func(fset *token.FileSet, fd *ast.FuncDecl) []lineSubject,
) (map[string]int, int) {
	if baseFile == nil {
		return nil, 0
	}
	counts := make(map[string]int)
	for _, decl := range baseFile.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		if fd.Recv == nil && isMovedTestName(movedOutTests, fd.Name.Name) {
			continue
		}
		counts[funcDeclKey(fd)] += len(collectFunc(nil, fd))
	}
	total := 0
	for key, baseCnt := range counts {
		// Credit only the retained per-function count so stripping from one test cannot launder additions in another test of the same file.
		total += min(baseCnt, targetFuncCounts[key])
	}
	return counts, total
}

func collectFileLineSubjects(
	fset *token.FileSet,
	file *ast.File,
	collectFunc func(fset *token.FileSet, fd *ast.FuncDecl) []lineSubject,
) []lineSubject {
	var out []lineSubject
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		out = append(out, collectFunc(fset, fd)...)
	}
	return out
}

func collectMovedCreditCounts(
	movedInTests map[string]movedGoTestBase,
	targetFuncCounts map[string]int,
	collectFunc func(fset *token.FileSet, fd *ast.FuncDecl) []lineSubject,
) (map[string]int, int) {
	if len(movedInTests) == 0 {
		return nil, 0
	}
	counts := make(map[string]int, len(movedInTests))
	total := 0
	for name, moved := range movedInTests {
		baseCnt := len(collectFunc(nil, moved.summary.decl))
		counts[name] = baseCnt
		// Credit only the retained per-function count so stripping from a moved test cannot launder new additions elsewhere.
		total += min(baseCnt, targetFuncCounts[name])
	}
	return counts, total
}

func isMovedTestName(movedTests map[string]movedGoTestBase, name string) bool {
	_, ok := movedTests[name]
	return ok
}

func countLineSubjectsByKey(entries []lineSubject) map[string]int {
	counts := make(map[string]int, len(entries))
	for _, entry := range entries {
		counts[entry.funcKey]++
	}
	return counts
}

func isUnchangedSubject(key string, baselines countedLineBaselines) bool {
	allowed := baselines.baseFuncCounts[key] + baselines.movedBaseCounts[key]
	return allowed > 0 && baselines.targetFuncCounts[key] <= allowed
}

func funcDeclKey(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	return formatReceiverType(fd.Recv.List[0].Type) + "." + fd.Name.Name
}

func formatReceiverType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return formatReceiverType(e.X)
	case *ast.Ident:
		return e.Name
	case *ast.IndexExpr:
		return formatReceiverType(e.X)
	case *ast.IndexListExpr:
		return formatReceiverType(e.X)
	default:
		return "recv"
	}
}

func collectFuncTimeSleepCalls(fset *token.FileSet, fd *ast.FuncDecl) []lineSubject {
	var out []lineSubject
	if fd == nil || fd.Body == nil {
		return out
	}
	key := funcDeclKey(fd)
	for node := range ast.Preorder(fd.Body) {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isTimeSleepCall(call) {
			continue
		}
		ln := 1
		if fset != nil {
			ln = fset.Position(call.Pos()).Line
		}
		out = append(out, lineSubject{line: ln, subject: fd.Name.Name, funcKey: key})
	}
	return out
}

func isTimeSleepCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Sleep" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "time"
}
