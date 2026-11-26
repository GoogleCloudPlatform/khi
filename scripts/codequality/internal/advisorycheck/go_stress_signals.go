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
	"fmt"
	"go/ast"
	"go/token"
	"math"
	"slices"
	"strconv"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type stressBound struct {
	key     string
	varName string
	isLoop  bool
	value   int
	line    int
}

type stressCollection struct {
	bounds   []stressBound
	loopVars map[string]bool
}

type stressAccumulator struct {
	varBounds  map[string]stressBound
	varOrder   []string
	loopBounds []stressBound
	loopVars   map[string]bool
}

type stressBoundExactKey struct {
	key   string
	value int
}

func (a *stressAccumulator) visitNode(fset *token.FileSet, node ast.Node) {
	switch s := node.(type) {
	case *ast.ValueSpec:
		a.addValueSpec(fset, s)
	case *ast.AssignStmt:
		a.addAssignStmt(fset, s)
	case *ast.ForStmt:
		a.addForStmt(fset, s)
	case *ast.RangeStmt:
		a.addRangeStmt(fset, s)
	}
}

func (a *stressAccumulator) addValueSpec(fset *token.FileSet, vs *ast.ValueSpec) {
	for i, name := range vs.Names {
		if i >= len(vs.Values) {
			break
		}
		a.addNamedVarBound(fset, name.Pos(), name.Name, vs.Values[i])
	}
}

func (a *stressAccumulator) addAssignStmt(fset *token.FileSet, s *ast.AssignStmt) {
	if (s.Tok != token.ASSIGN && s.Tok != token.DEFINE) || len(s.Lhs) != len(s.Rhs) {
		return
	}
	for i, lhs := range s.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		a.addNamedVarBound(fset, s.Pos(), id.Name, s.Rhs[i])
	}
}

func (a *stressAccumulator) addNamedVarBound(fset *token.FileSet, pos token.Pos, name string, expr ast.Expr) {
	if name == "_" {
		return
	}
	b, ok := makeStressBound(fset, pos, "var:"+name, expr)
	if !ok {
		return
	}
	b.varName = name
	if prev, exists := a.varBounds[b.key]; !exists {
		a.varOrder = append(a.varOrder, b.key)
		a.varBounds[b.key] = b
		return
	} else if b.value > prev.value {
		a.varBounds[b.key] = b
	}
}

func (a *stressAccumulator) addForStmt(fset *token.FileSet, s *ast.ForStmt) {
	bin, ok := s.Cond.(*ast.BinaryExpr)
	if !ok || !isComparisonOp(bin.Op) {
		return
	}
	recordLoopExprIdents(bin.X, a.loopVars)
	recordLoopExprIdents(bin.Y, a.loopVars)
	if b, ok := makeStressBound(fset, s.Pos(), "for:"+formatCallName(bin.X), bin.Y); ok {
		b.isLoop = true
		a.loopBounds = append(a.loopBounds, b)
		return
	}
	if b, ok := makeStressBound(fset, s.Pos(), "for:"+formatCallName(bin.Y), bin.X); ok {
		b.isLoop = true
		a.loopBounds = append(a.loopBounds, b)
	}
}

func (a *stressAccumulator) addRangeStmt(fset *token.FileSet, s *ast.RangeStmt) {
	recordLoopExprIdents(s.X, a.loopVars)
	if b, ok := makeStressBound(fset, s.Pos(), "range", s.X); ok {
		b.isLoop = true
		a.loopBounds = append(a.loopBounds, b)
	}
}

func (a *stressAccumulator) finish() stressCollection {
	bounds := make([]stressBound, 0, len(a.varOrder)+len(a.loopBounds))
	for _, key := range a.varOrder {
		bounds = append(bounds, a.varBounds[key])
	}
	bounds = append(bounds, a.loopBounds...)
	return stressCollection{
		bounds:   bounds,
		loopVars: a.loopVars,
	}
}

func detectGoStressReductions(
	targetFset *token.FileSet,
	targetFile *ast.File,
	baseFset *token.FileSet,
	baseFile *ast.File,
	in goFileInput,
) []rule.QAItem {
	targetFuncs := collectGoTestFuncSummaries(targetFset, targetFile)
	var baseByName map[string]goTestFuncSummary
	if baseFile != nil {
		baseByName = indexGoTestFuncs(collectGoTestFuncSummaries(baseFset, baseFile))
	}
	var items []rule.QAItem

	for _, targetFn := range targetFuncs {
		if len(in.changedLines) > 0 && !overlapsChangedLines(targetFn.line, targetFn.endLine, in.changedLines) {
			continue
		}
		fnBaseFset, baseFn, ok := resolveBaseTestSummary(targetFn.name, baseFset, baseByName, in.movedInTests)
		if !ok || !isConcurrentOrStressTest(baseFn.decl) {
			continue
		}
		baseCollected := collectStressBounds(fnBaseFset, baseFn.decl.Body)
		targetCollected := collectStressBounds(targetFset, targetFn.decl.Body)
		items = append(items, compareStressBounds(in.relPath, targetFn.name, baseCollected, targetCollected)...)
	}
	return items
}

func resolveBaseTestSummary(
	name string,
	baseFset *token.FileSet,
	baseByName map[string]goTestFuncSummary,
	movedInTests map[string]movedGoTestBase,
) (*token.FileSet, goTestFuncSummary, bool) {
	if baseFn, ok := baseByName[name]; ok {
		return baseFset, baseFn, true
	}
	if moved, ok := movedInTests[name]; ok {
		return moved.fset, moved.summary, true
	}
	return nil, goTestFuncSummary{}, false
}

func isConcurrentOrStressTest(fd *ast.FuncDecl) bool {
	if fd == nil || fd.Body == nil {
		return false
	}
	found := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		if isStressSignalNode(n) {
			found = true
			return false
		}
		return true
	})
	return found
}

func isStressSignalNode(n ast.Node) bool {
	switch node := n.(type) {
	case *ast.GoStmt:
		return true
	case *ast.SelectorExpr:
		return node.Sel.Name == "WaitGroup"
	case *ast.BasicLit:
		val, ok := parsePositiveIntLit(node)
		return ok && val >= 50
	default:
		return false
	}
}

func collectStressBounds(fset *token.FileSet, body *ast.BlockStmt) stressCollection {
	acc := stressAccumulator{
		varBounds: make(map[string]stressBound),
		loopVars:  make(map[string]bool),
	}
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		ast.Inspect(n, func(cur ast.Node) bool {
			if cur == nil {
				return false
			}
			if ifStmt, ok := cur.(*ast.IfStmt); ok && isTestingShortCond(ifStmt.Cond) {
				if ifStmt.Else != nil {
					walk(ifStmt.Else)
				}
				return false
			}
			acc.visitNode(fset, cur)
			return true
		})
	}
	walk(body)
	return acc.finish()
}

func recordLoopExprIdents(expr ast.Expr, loopVars map[string]bool) {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name != "_" {
			loopVars[e.Name] = true
		}
	case *ast.ParenExpr:
		recordLoopExprIdents(e.X, loopVars)
	case *ast.BinaryExpr:
		recordLoopExprIdents(e.X, loopVars)
		recordLoopExprIdents(e.Y, loopVars)
	}
}

func isComparisonOp(op token.Token) bool {
	return op == token.LSS || op == token.LEQ || op == token.GTR || op == token.GEQ
}

func makeStressBound(fset *token.FileSet, pos token.Pos, key string, expr ast.Expr) (stressBound, bool) {
	val, ok := parsePositiveIntLit(expr)
	if !ok || val < 2 {
		return stressBound{}, false
	}
	return stressBound{key: key, value: val, line: fset.Position(pos).Line}, true
}

func parsePositiveIntLit(expr ast.Expr) (int, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}
	val, err := strconv.ParseInt(lit.Value, 0, 64)
	if err != nil || val <= 0 || val > math.MaxInt {
		return 0, false
	}
	return int(val), true
}

func compareStressBounds(relPath string, testName string, base stressCollection, target stressCollection) []rule.QAItem {
	baseMatched, targetMatched, items := matchExactStressBounds(relPath, testName, base.bounds, target.bounds)
	baseRem := collectUnmatchedLoopBounds(base.bounds, baseMatched, base.loopVars)
	targetRem := collectUnmatchedLoopBounds(target.bounds, targetMatched, target.loopVars)
	slices.SortFunc(baseRem, compareStressBoundDesc)
	slices.SortFunc(targetRem, compareStressBoundDesc)

	pairCount := min(len(baseRem), len(targetRem))
	for i := range pairCount {
		if isReducedStressBound(baseRem[i], targetRem[i]) {
			items = append(items, makeStressReducedItem(relPath, testName, baseRem[i], targetRem[i]))
		}
	}
	return items
}

func matchExactStressBounds(
	relPath string,
	testName string,
	baseBounds []stressBound,
	targetBounds []stressBound,
) ([]bool, []bool, []rule.QAItem) {
	baseOrder := sortedBoundIndicesDesc(baseBounds)
	targetOrder := sortedBoundIndicesDesc(targetBounds)
	// Pre-match identical (key, value) bounds before pairing remaining same-key bounds by value,
	// so modernizing one of multiple same-key loops does not pair it with a smaller unchanged sibling.
	baseMatched, targetMatched := markExactKeyValueMatches(baseBounds, baseOrder, targetBounds, targetOrder)

	targetByKey := make(map[string][]int, len(targetBounds))
	for _, ti := range targetOrder {
		if !targetMatched[ti] {
			key := targetBounds[ti].key
			targetByKey[key] = append(targetByKey[key], ti)
		}
	}

	var items []rule.QAItem
	for _, bi := range baseOrder {
		if baseMatched[bi] {
			continue
		}
		bb := baseBounds[bi]
		candidates := targetByKey[bb.key]
		if len(candidates) == 0 {
			continue
		}
		ti := candidates[0]
		targetByKey[bb.key] = candidates[1:]
		baseMatched[bi] = true
		targetMatched[ti] = true
		if isReducedStressBound(bb, targetBounds[ti]) {
			items = append(items, makeStressReducedItem(relPath, testName, bb, targetBounds[ti]))
		}
	}
	return baseMatched, targetMatched, items
}

func markExactKeyValueMatches(
	baseBounds []stressBound,
	baseOrder []int,
	targetBounds []stressBound,
	targetOrder []int,
) ([]bool, []bool) {
	baseMatched := make([]bool, len(baseBounds))
	targetMatched := make([]bool, len(targetBounds))
	targetByExact := make(map[stressBoundExactKey][]int, len(targetBounds))
	for _, ti := range targetOrder {
		ek := stressBoundExactKey{key: targetBounds[ti].key, value: targetBounds[ti].value}
		targetByExact[ek] = append(targetByExact[ek], ti)
	}
	for _, bi := range baseOrder {
		bb := baseBounds[bi]
		ek := stressBoundExactKey{key: bb.key, value: bb.value}
		candidates := targetByExact[ek]
		if len(candidates) == 0 {
			continue
		}
		targetByExact[ek] = candidates[1:]
		baseMatched[bi] = true
		targetMatched[candidates[0]] = true
	}
	return baseMatched, targetMatched
}

func sortedBoundIndicesDesc(bounds []stressBound) []int {
	indices := make([]int, len(bounds))
	for i := range indices {
		indices[i] = i
	}
	slices.SortFunc(indices, func(i, j int) int {
		return compareStressBoundDesc(bounds[i], bounds[j])
	})
	return indices
}

func collectUnmatchedLoopBounds(bounds []stressBound, matched []bool, loopVars map[string]bool) []stressBound {
	var rem []stressBound
	for i, b := range bounds {
		if matched[i] {
			continue
		}
		if b.isLoop || loopVars[b.varName] {
			rem = append(rem, b)
		}
	}
	return rem
}

func compareStressBoundDesc(a stressBound, b stressBound) int {
	return cmp.Or(
		cmp.Compare(b.value, a.value),
		cmp.Compare(a.line, b.line),
	)
}

func isReducedStressBound(bb stressBound, tb stressBound) bool {
	if bb.value < 10 || tb.value >= bb.value {
		return false
	}
	return bb.value-tb.value > bb.value/4
}

func makeStressReducedItem(relPath string, testName string, bb stressBound, tb stressBound) rule.QAItem {
	summary := fmt.Sprintf("stress bound %s reduced from %d to %d outside testing.Short()", bb.key, bb.value, tb.value)
	evidence := map[string]any{"bound_key": bb.key, "before": bb.value, "after": tb.value}
	if bb.key != tb.key {
		summary = fmt.Sprintf("stress bound %s -> %s reduced from %d to %d outside testing.Short()", bb.key, tb.key, bb.value, tb.value)
		evidence["target_bound_key"] = tb.key
	}
	item := makeAdvisoryItem(checkTestStressReduced, relPath, tb.line, testName, summary)
	item.Evidence = evidence
	item.ReviewPrompt = "Verify whether this >25% reduction in stress/concurrency iterations preserves race and contention coverage (scale inside `if testing.Short()` instead)."
	return item
}
