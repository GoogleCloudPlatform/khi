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

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type discardOccurrence struct {
	line int
	sig  string
}

func detectGoTestDiscardCalls(
	targetFset *token.FileSet,
	targetFile *ast.File,
	_ *token.FileSet,
	baseFile *ast.File,
	in goFileInput,
) []rule.QAItem {
	baseDiscardsByFunc := collectRemainingBaseDiscards(baseFile, in.movedOutTests)
	movedDiscardsByFunc := collectMovedInDiscards(in.movedInTests)
	var items []rule.QAItem
	for _, decl := range targetFile.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		baseDiscards := baseDiscardsByFunc[funcDeclKey(fd)]
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
	occurrences := collectFuncDiscardOccurrences(fset, fd)
	if len(occurrences) == 0 {
		return nil
	}
	remainingCredit, excessBySig := computeDiscardBudgets(occurrences, in.changedLines, baseDiscards, movedDiscards)
	emittedBySig := make(map[string]int, len(excessBySig))
	var items []rule.QAItem
	for _, occ := range occurrences {
		if !isLineInScope(occ.line, in.changedLines) {
			continue
		}
		if remainingCredit[occ.sig] > 0 {
			remainingCredit[occ.sig]--
			continue
		}
		emittedBySig[occ.sig]++
		items = append(items, makeDiscardWeakenedItem(in.relPath, occ.line, fd.Name.Name, occ.sig))
	}
	// If Git LCS diff aligned a newly transferred discard as unchanged context, emit remaining excess per signature.
	for _, occ := range occurrences {
		if isLineInScope(occ.line, in.changedLines) || emittedBySig[occ.sig] >= excessBySig[occ.sig] {
			continue
		}
		emittedBySig[occ.sig]++
		items = append(items, makeDiscardWeakenedItem(in.relPath, occ.line, fd.Name.Name, occ.sig))
	}
	return items
}

func makeDiscardWeakenedItem(relPath string, line int, subject string, sig string) rule.QAItem {
	item := makeAdvisoryItem(checkTestAssertionWeakened, relPath, line, subject, fmt.Sprintf("newly added full return-value discard `_ = %s(...)`", sig))
	item.Evidence = map[string]any{"discarded_call": sig}
	item.ReviewPrompt = "Verify that discarding the return value of this call is intentional and not hiding an unchecked result or error."
	return item
}

func collectFuncDiscardOccurrences(fset *token.FileSet, fd *ast.FuncDecl) []discardOccurrence {
	var out []discardOccurrence
	for node := range ast.Preorder(fd.Body) {
		pos, sig, ok := matchSingleDiscardCall(node)
		if !ok {
			continue
		}
		out = append(out, discardOccurrence{line: fset.Position(pos).Line, sig: sig})
	}
	return out
}

func computeDiscardBudgets(
	occurrences []discardOccurrence,
	changedLines map[int]struct{},
	baseDiscards map[string]int,
	movedDiscards map[string]int,
) (map[string]int, map[string]int) {
	credit := make(map[string]int, len(baseDiscards)+len(movedDiscards))
	for sig, n := range baseDiscards {
		credit[sig] += n
	}
	for sig, n := range movedDiscards {
		credit[sig] += n
	}
	targetCounts := make(map[string]int, len(occurrences))
	for _, occ := range occurrences {
		targetCounts[occ.sig]++
	}
	excess := make(map[string]int, len(targetCounts))
	for sig, cnt := range targetCounts {
		if cnt > credit[sig] {
			excess[sig] = cnt - credit[sig]
		}
	}
	// Consume baseline credit with untouched occurrences first so adding a duplicate discard on a changed line cannot reuse untouched credit.
	for _, occ := range occurrences {
		if !isLineInScope(occ.line, changedLines) && credit[occ.sig] > 0 {
			credit[occ.sig]--
		}
	}
	return credit, excess
}

func collectRemainingBaseDiscards(file *ast.File, movedOutTests map[string]movedGoTestBase) map[string]map[string]int {
	out := make(map[string]map[string]int)
	if file == nil {
		return out
	}
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		if fd.Recv == nil && isMovedTestName(movedOutTests, fd.Name.Name) {
			continue
		}
		key := funcDeclKey(fd)
		perFunc := out[key]
		if perFunc == nil {
			perFunc = make(map[string]int)
			out[key] = perFunc
		}
		addNodeDiscardCallTexts(fd.Body, perFunc)
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
