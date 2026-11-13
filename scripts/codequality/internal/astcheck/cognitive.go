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

package astcheck

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// ComplexitySummary holds cognitive complexity, maximum control-flow nesting depth, and per-line hotspots.
type ComplexitySummary struct {
	Cognitive  int
	MaxNesting int
	Hotspots   []rule.Hotspot
}

// AnalyzeBlockComplexity computes cognitive complexity and maximum nesting depth for a function body.
func AnalyzeBlockComplexity(fset *token.FileSet, body *ast.BlockStmt) ComplexitySummary {
	w := &complexityWalker{fset: fset}
	w.walkBlock(body, 0)
	return ComplexitySummary{
		Cognitive:  w.total,
		MaxNesting: w.maxNest,
		Hotspots:   w.hotspots,
	}
}

// AnalyzeSubtestBlockComplexity computes cognitive complexity for a t.Run/b.Run body while skipping nested .Run subtests.
func AnalyzeSubtestBlockComplexity(fset *token.FileSet, body *ast.BlockStmt) ComplexitySummary {
	w := &complexityWalker{fset: fset, skipNestedRun: true}
	w.walkBlock(body, 0)
	return ComplexitySummary{
		Cognitive:  w.total,
		MaxNesting: w.maxNest,
		Hotspots:   w.hotspots,
	}
}

type complexityWalker struct {
	fset          *token.FileSet
	skipNestedRun bool
	total         int
	maxNest       int
	hotspots      []rule.Hotspot
}

func (w *complexityWalker) recordNest(depth int) {
	w.maxNest = max(w.maxNest, depth)
}

func (w *complexityWalker) add(pos token.Pos, delta int, reason string) {
	w.total += delta
	w.hotspots = append(w.hotspots, rule.Hotspot{
		Line:    w.fset.Position(pos).Line,
		Delta:   delta,
		Summary: fmt.Sprintf("+%d (%s)", delta, reason),
	})
}

func (w *complexityWalker) addControl(pos token.Pos, nest int, keyword string) {
	if nest == 0 {
		w.add(pos, 1, keyword)
		return
	}
	w.add(pos, 1+nest, fmt.Sprintf("nested %s, depth=%d", keyword, nest))
}

func (w *complexityWalker) walkBlock(b *ast.BlockStmt, nest int) {
	if b == nil {
		return
	}
	for _, stmt := range b.List {
		w.walkStmt(stmt, nest)
	}
}

func (w *complexityWalker) walkStmt(stmt ast.Stmt, nest int) {
	if stmt == nil {
		return
	}
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		w.walkBlock(s, nest)
	case *ast.LabeledStmt:
		w.walkStmt(s.Stmt, nest)
	case *ast.IfStmt:
		w.walkIf(s, nest)
	case *ast.SwitchStmt:
		w.walkSwitch(s, nest)
	case *ast.TypeSwitchStmt:
		w.walkTypeSwitch(s, nest)
	case *ast.SelectStmt:
		w.walkSelect(s, nest)
	case *ast.ForStmt:
		w.walkFor(s, nest)
	case *ast.RangeStmt:
		w.walkRange(s, nest)
	case *ast.BranchStmt:
		w.walkBranch(s)
	default:
		w.walkNestedNodes(s, nest)
	}
}

func (w *complexityWalker) walkIf(s *ast.IfStmt, nest int) {
	w.walkStmt(s.Init, nest)
	w.evalBoolExpr(s.Cond, nest)
	w.addControl(s.If, nest, "if")
	w.recordNest(nest + 1)
	w.walkBlock(s.Body, nest+1)
	w.walkElse(s.Else, nest)
}

func (w *complexityWalker) walkElseIf(s *ast.IfStmt, nest int) {
	w.walkStmt(s.Init, nest)
	w.evalBoolExpr(s.Cond, nest)
	w.add(s.If, 1, "else if")
	w.walkBlock(s.Body, nest+1)
	w.walkElse(s.Else, nest)
}

func (w *complexityWalker) walkElse(elseStmt ast.Stmt, nest int) {
	if elseStmt == nil {
		return
	}
	if elif, ok := elseStmt.(*ast.IfStmt); ok {
		w.walkElseIf(elif, nest)
		return
	}
	w.add(elseStmt.Pos(), 1, "else")
	w.walkStmt(elseStmt, nest+1)
}

func (w *complexityWalker) walkSwitch(s *ast.SwitchStmt, nest int) {
	w.walkStmt(s.Init, nest)
	w.evalBoolExpr(s.Tag, nest)
	w.addControl(s.Switch, nest, "switch")
	w.recordNest(nest + 1)
	w.walkClauseBlock(s.Body, nest+1)
}

func (w *complexityWalker) walkTypeSwitch(s *ast.TypeSwitchStmt, nest int) {
	w.walkStmt(s.Init, nest)
	w.addControl(s.Switch, nest, "type switch")
	w.recordNest(nest + 1)
	w.walkClauseBlock(s.Body, nest+1)
}

func (w *complexityWalker) walkSelect(s *ast.SelectStmt, nest int) {
	w.addControl(s.Select, nest, "select")
	w.recordNest(nest + 1)
	w.walkClauseBlock(s.Body, nest+1)
}

func (w *complexityWalker) walkClauseBlock(b *ast.BlockStmt, nest int) {
	for _, stmt := range b.List {
		switch c := stmt.(type) {
		case *ast.CaseClause:
			w.walkCaseClause(c, nest)
		case *ast.CommClause:
			w.walkStmtList(c.Body, nest)
		}
	}
}

func (w *complexityWalker) walkCaseClause(c *ast.CaseClause, nest int) {
	for _, expr := range c.List {
		w.evalBoolExpr(expr, nest)
	}
	w.walkStmtList(c.Body, nest)
}

func (w *complexityWalker) walkStmtList(stmts []ast.Stmt, nest int) {
	for _, st := range stmts {
		w.walkStmt(st, nest)
	}
}

func (w *complexityWalker) walkFor(s *ast.ForStmt, nest int) {
	w.walkStmt(s.Init, nest)
	w.evalBoolExpr(s.Cond, nest)
	w.walkStmt(s.Post, nest)
	w.addControl(s.For, nest, "for")
	w.recordNest(nest + 1)
	w.walkBlock(s.Body, nest+1)
}

func (w *complexityWalker) walkRange(s *ast.RangeStmt, nest int) {
	w.walkNestedNodes(s.X, nest)
	w.addControl(s.For, nest, "range")
	w.recordNest(nest + 1)
	w.walkBlock(s.Body, nest+1)
}

func (w *complexityWalker) walkBranch(s *ast.BranchStmt) {
	if s.Label != nil {
		w.add(s.TokPos, 1, "labeled "+s.Tok.String())
	}
}

func (w *complexityWalker) evalBoolExpr(expr ast.Expr, nest int) {
	if expr == nil {
		return
	}
	w.addBoolComplexity(expr)
	for _, leaf := range collectNonLogicalLeaves(expr) {
		w.walkNestedNodes(leaf, nest)
	}
}

func (w *complexityWalker) walkNestedNodes(root ast.Node, nest int) {
	ast.Inspect(root, func(n ast.Node) bool {
		return w.visitNestedNode(n, nest)
	})
}

func (w *complexityWalker) visitNestedNode(n ast.Node, nest int) bool {
	switch x := n.(type) {
	case *ast.CallExpr:
		if w.skipNestedRun {
			if _, _, isRun := extractSubtestRunCall(x); isRun {
				return false
			}
		}
		return true
	case *ast.FuncLit:
		w.recordNest(nest + 1)
		w.walkBlock(x.Body, nest+1)
		return false
	case *ast.BinaryExpr:
		if isLogicalOp(x.Op) {
			w.evalBoolExpr(x, nest)
			return false
		}
		return true
	default:
		return true
	}
}

func collectNonLogicalLeaves(expr ast.Expr) []ast.Expr {
	var leaves []ast.Expr
	var walk func(e ast.Expr)
	walk = func(e ast.Expr) {
		unp := ast.Unparen(e)
		if b, ok := unp.(*ast.BinaryExpr); ok && isLogicalOp(b.Op) {
			walk(b.X)
			walk(b.Y)
			return
		}
		leaves = append(leaves, unp)
	}
	walk(expr)
	return leaves
}

func (w *complexityWalker) addBoolComplexity(expr ast.Expr) {
	ops := flattenLogicalOps(expr)
	if len(ops) == 0 {
		return
	}
	runs := 1
	for i := 1; i < len(ops); i++ {
		if ops[i] != ops[i-1] {
			runs++
		}
	}
	reason := "boolean operator sequence"
	if runs > 1 {
		reason = "mixed && / || sequence"
	}
	w.add(expr.Pos(), runs, reason)
}

func flattenLogicalOps(expr ast.Expr) []token.Token {
	var ops []token.Token
	var walk func(e ast.Expr)
	walk = func(e ast.Expr) {
		b, ok := ast.Unparen(e).(*ast.BinaryExpr)
		if !ok || !isLogicalOp(b.Op) {
			return
		}
		walk(b.X)
		ops = append(ops, b.Op)
		walk(b.Y)
	}
	walk(expr)
	return ops
}

func isLogicalOp(op token.Token) bool {
	return op == token.LAND || op == token.LOR
}
