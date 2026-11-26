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
	"go/token"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type lineSubject struct {
	line    int
	subject string
}

type countedLineSignalSpec struct {
	check        string
	summary      string
	reviewPrompt string
	collectFunc  func(fset *token.FileSet, fd *ast.FuncDecl) []lineSubject
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
	walkSkippingShortIfs(fd.Body, func(call *ast.CallExpr) {
		if isGoSkipCall(call) {
			ln := 1
			if fset != nil {
				ln = fset.Position(call.Pos()).Line
			}
			out = append(out, lineSubject{line: ln, subject: fd.Name.Name})
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
	baseCount := countRemainingBaseSubjects(baseFile, in.movedOutTests, spec.collectFunc)
	targetEntries := collectFileLineSubjects(targetFset, targetFile, spec.collectFunc)
	targetFuncCounts := countLineSubjectsByName(targetEntries)
	movedBaseCounts, movedCredit := collectMovedCreditCounts(in.movedInTests, targetFuncCounts, spec.collectFunc)
	baseCount += movedCredit

	if len(targetEntries) <= baseCount {
		return nil
	}
	var items []rule.QAItem
	for _, entry := range targetEntries {
		if !isLineInScope(entry.line, in.changedLines) || isUnchangedMovedSubject(entry.subject, movedBaseCounts, targetFuncCounts) {
			continue
		}
		item := makeAdvisoryItem(spec.check, in.relPath, entry.line, entry.subject, spec.summary)
		item.Evidence = map[string]any{"before_count": baseCount, "after_count": len(targetEntries)}
		item.ReviewPrompt = spec.reviewPrompt
		items = append(items, item)
	}
	return items
}

func countRemainingBaseSubjects(
	baseFile *ast.File,
	movedOutTests map[string]movedGoTestBase,
	collectFunc func(fset *token.FileSet, fd *ast.FuncDecl) []lineSubject,
) int {
	if baseFile == nil {
		return 0
	}
	total := 0
	for _, decl := range baseFile.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		if fd.Recv == nil && isMovedTestName(movedOutTests, fd.Name.Name) {
			continue
		}
		total += len(collectFunc(nil, fd))
	}
	return total
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

func countLineSubjectsByName(entries []lineSubject) map[string]int {
	counts := make(map[string]int, len(entries))
	for _, entry := range entries {
		counts[entry.subject]++
	}
	return counts
}

func isUnchangedMovedSubject(subject string, movedBaseCounts map[string]int, targetFuncCounts map[string]int) bool {
	baseCnt, isMoved := movedBaseCounts[subject]
	return isMoved && targetFuncCounts[subject] <= baseCnt
}

func collectFuncTimeSleepCalls(fset *token.FileSet, fd *ast.FuncDecl) []lineSubject {
	var out []lineSubject
	if fd == nil || fd.Body == nil {
		return out
	}
	for node := range ast.Preorder(fd.Body) {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isTimeSleepCall(call) {
			continue
		}
		ln := 1
		if fset != nil {
			ln = fset.Position(call.Pos()).Line
		}
		out = append(out, lineSubject{line: ln, subject: fd.Name.Name})
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

func detectGoTestDiscardCalls(
	targetFset *token.FileSet,
	targetFile *ast.File,
	_ *token.FileSet,
	baseFile *ast.File,
	in goFileInput,
) []rule.QAItem {
	baseDiscards := collectRemainingBaseDiscards(baseFile, in.movedOutTests)
	movedDiscardsByFunc := collectMovedInDiscards(in.movedInTests)
	var items []rule.QAItem
	for _, decl := range targetFile.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		var movedDiscards map[string]int
		if fd.Recv == nil {
			movedDiscards = movedDiscardsByFunc[fd.Name.Name]
		}
		items = append(items, inspectFuncDiscardCalls(targetFset, fd, in, baseDiscards, movedDiscards)...)
	}
	return items
}

func inspectFuncDiscardCalls(
	fset *token.FileSet,
	fd *ast.FuncDecl,
	in goFileInput,
	baseDiscards map[string]int,
	movedDiscards map[string]int,
) []rule.QAItem {
	var items []rule.QAItem
	for node := range ast.Preorder(fd.Body) {
		pos, sig, ok := matchSingleDiscardCall(node)
		if !ok {
			continue
		}
		ln := fset.Position(pos).Line
		if !isLineInScope(ln, in.changedLines) {
			continue
		}
		if baseDiscards[sig] > 0 {
			continue
		}
		if movedDiscards[sig] > 0 {
			movedDiscards[sig]--
			continue
		}
		item := makeAdvisoryItem(checkTestAssertionWeakened, in.relPath, ln, fd.Name.Name, fmt.Sprintf("newly added full return-value discard `_ = %s(...)`", sig))
		item.Evidence = map[string]any{"discarded_call": sig}
		item.ReviewPrompt = "Verify that discarding the return value of this call is intentional and not hiding an unchecked result or error."
		items = append(items, item)
	}
	return items
}

func collectRemainingBaseDiscards(file *ast.File, movedOutTests map[string]movedGoTestBase) map[string]int {
	out := make(map[string]int)
	if file == nil {
		return out
	}
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv == nil && isMovedTestName(movedOutTests, fd.Name.Name) {
			continue
		}
		addNodeDiscardCallTexts(decl, out)
	}
	return out
}

func collectMovedInDiscards(movedInTests map[string]movedGoTestBase) map[string]map[string]int {
	if len(movedInTests) == 0 {
		return nil
	}
	out := make(map[string]map[string]int, len(movedInTests))
	for name, moved := range movedInTests {
		if moved.summary.decl == nil {
			continue
		}
		perFunc := make(map[string]int)
		addNodeDiscardCallTexts(moved.summary.decl, perFunc)
		out[name] = perFunc
	}
	return out
}

func addNodeDiscardCallTexts(node ast.Node, out map[string]int) {
	if node == nil {
		return
	}
	for cur := range ast.Preorder(node) {
		if _, sig, ok := matchSingleDiscardCall(cur); ok {
			out[sig]++
		}
	}
}

func matchSingleDiscardCall(node ast.Node) (token.Pos, string, bool) {
	assign, ok := node.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return token.NoPos, "", false
	}
	lhsID, isID := assign.Lhs[0].(*ast.Ident)
	call, isCall := assign.Rhs[0].(*ast.CallExpr)
	if !isID || lhsID.Name != "_" || !isCall {
		return token.NoPos, "", false
	}
	return assign.Pos(), formatCallName(call.Fun), true
}

func formatCallName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return formatCallName(e.X) + "." + e.Sel.Name
	default:
		return "call"
	}
}

func detectGoCommentSuppressions(
	targetFset *token.FileSet,
	targetFile *ast.File,
	_ *token.FileSet,
	baseFile *ast.File,
	in goFileInput,
) []rule.QAItem {
	baseCount := 0
	if baseFile != nil {
		baseCount = len(collectGoSuppressions(nil, baseFile))
	}
	targetSuppressions := collectGoSuppressions(targetFset, targetFile)
	if len(targetSuppressions) <= baseCount {
		return nil
	}
	var items []rule.QAItem
	for _, sup := range targetSuppressions {
		if !isLineInScope(sup.line, in.changedLines) {
			continue
		}
		item := makeAdvisoryItem(checkSuppressionAdded, in.relPath, sup.line, "<comment>", "newly added quality or linter suppression comment")
		item.Evidence = map[string]any{"marker": sup.subject, "before_count": baseCount, "after_count": len(targetSuppressions)}
		item.ReviewPrompt = "Verify that this CODEQUALITY_ALLOW or nolint suppression is strictly justified and not hiding an avoidable defect."
		items = append(items, item)
	}
	return items
}

func collectGoSuppressions(fset *token.FileSet, file *ast.File) []lineSubject {
	var out []lineSubject
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			marker := matchGoSuppressionMarker(c.Text)
			if marker == "" {
				continue
			}
			ln := 1
			if fset != nil {
				ln = fset.Position(c.Pos()).Line
			}
			out = append(out, lineSubject{line: ln, subject: marker})
		}
	}
	return out
}

func matchGoSuppressionMarker(commentText string) string {
	for _, rawLine := range strings.Split(commentText, "\n") {
		trimmedRaw := strings.TrimSpace(rawLine)
		body := stripCommentLineDelimiters(trimmedRaw)
		if hasAllowDirectivePrefix(body) {
			return "CODEQUALITY_ALLOW"
		}
		if hasGoNolintDirective(trimmedRaw, body) {
			return "nolint"
		}
	}
	return ""
}

func hasGoNolintDirective(trimmedRaw string, body string) bool {
	// Detect naked //nolint even when followed by non-standard text after whitespace.
	if after, ok := strings.CutPrefix(trimmedRaw, "//nolint"); ok {
		return after == "" || after[0] == ':' || after[0] == '/' || after[0] == ' ' || after[0] == '\t'
	}
	after, ok := strings.CutPrefix(body, "nolint")
	if !ok {
		return false
	}
	rest := strings.TrimLeft(after, " \t")
	return rest == "" || strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, "//")
}
