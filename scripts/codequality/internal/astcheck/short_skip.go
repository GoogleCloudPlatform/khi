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
	"go/ast"
	"go/token"
	"maps"
	"strconv"
)

const (
	shortCondNone shortCondKind = iota
	shortCondPositive
	shortCondNegated
)

const (
	boolLitNone boolLitKind = iota
	boolLitTrue
	boolLitFalse
)

const (
	msgNoShortNegatedGuard = "do not guard test logic with !testing.Short(); declare full-strength defaults and scale down parameters inside if testing.Short()"
	msgNoShortEmptyBranch  = "do not use an empty if testing.Short() {} else { ... } branch; declare full-strength defaults and scale down parameters inside if testing.Short()"
	msgNoShortSkipCall     = "do not call Skip/Skipf/SkipNow inside testing.Short(); scale down iteration counts instead of skipping tests"
	msgNoShortEarlyReturn  = "do not return early inside testing.Short(); scale down iteration counts instead of skipping tests"
)

type shortCondKind int

type boolLitKind int

type shortWalkScope struct {
	shortVars    map[string]shortCondKind
	inShortGuard bool
}

type shortSkipChecker struct {
	fset        *token.FileSet
	relPath     string
	testingName string
	violations  []ConventionViolation
}

func checkNoShortSkip(fset *token.FileSet, relPath string, file *ast.File) []ConventionViolation {
	testingName := findTestingImportName(file)
	if testingName == "" {
		return nil
	}
	checker := &shortSkipChecker{
		fset:        fset,
		relPath:     relPath,
		testingName: testingName,
	}
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Body != nil {
			checker.walkBlock(fd.Body, shortWalkScope{})
		}
	}
	return checker.violations
}

func findTestingImportName(file *ast.File) string {
	for _, imp := range file.Imports {
		importPath, _ := strconv.Unquote(imp.Path.Value)
		if importPath != "testing" {
			continue
		}
		if imp.Name == nil {
			return "testing"
		}
		if imp.Name.Name == "_" {
			continue
		}
		return imp.Name.Name
	}
	return ""
}

func (c *shortSkipChecker) walkBlock(body *ast.BlockStmt, scope shortWalkScope) {
	blockScope := shortWalkScope{
		shortVars:    cloneShortVars(scope.shortVars),
		inShortGuard: scope.inShortGuard,
	}
	c.walkStmts(body.List, blockScope)
}

func (c *shortSkipChecker) walkStmts(stmts []ast.Stmt, scope shortWalkScope) {
	for _, stmt := range stmts {
		c.walkNode(stmt, scope)
	}
}

func (c *shortSkipChecker) walkNode(root ast.Node, scope shortWalkScope) {
	ast.Inspect(root, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.BlockStmt:
			c.walkBlock(node, scope)
			return false
		case *ast.FuncLit:
			c.walkBlock(node.Body, shortWalkScope{shortVars: cloneShortVars(scope.shortVars)})
			return false
		case *ast.IfStmt:
			c.walkIfStmt(node, scope)
			return false
		case *ast.SwitchStmt:
			c.walkSwitchStmt(node, scope)
			return false
		case *ast.AssignStmt:
			c.recordAssignBindings(node, scope.shortVars)
		case *ast.ValueSpec:
			c.recordValueSpecBindings(node, scope.shortVars)
		case *ast.CallExpr:
			if scope.inShortGuard && isSkipCall(node) {
				c.addViolation(node.Pos(), msgNoShortSkipCall)
			}
		case *ast.ReturnStmt:
			if scope.inShortGuard && len(node.Results) == 0 {
				c.addViolation(node.Pos(), msgNoShortEarlyReturn)
			}
		}
		return true
	})
}

func (c *shortSkipChecker) walkIfStmt(stmt *ast.IfStmt, scope shortWalkScope) {
	ifScope := c.scopeWithInit(stmt.Init, scope)
	bodyScope := ifScope
	switch c.classifyShortCond(stmt.Cond, ifScope.shortVars) {
	case shortCondNegated:
		c.addViolation(stmt.Cond.Pos(), msgNoShortNegatedGuard)
	case shortCondPositive:
		if stmt.Else != nil && len(stmt.Body.List) == 0 {
			c.addViolation(stmt.Pos(), msgNoShortEmptyBranch)
		}
		bodyScope.inShortGuard = true
	}
	c.walkBlock(stmt.Body, bodyScope)
	if stmt.Else != nil {
		c.walkNode(stmt.Else, ifScope)
	}
}

func (c *shortSkipChecker) walkSwitchStmt(stmt *ast.SwitchStmt, scope shortWalkScope) {
	swScope := c.scopeWithInit(stmt.Init, scope)
	if stmt.Tag == nil {
		c.walkTaglessSwitch(stmt.Body.List, swScope)
		return
	}
	c.walkTaggedSwitch(stmt.Tag, stmt.Body.List, swScope)
}

func (c *shortSkipChecker) walkTaglessSwitch(clauses []ast.Stmt, scope shortWalkScope) {
	for _, raw := range clauses {
		cc := raw.(*ast.CaseClause)
		caseGuarded := scope.inShortGuard
		for _, expr := range cc.List {
			switch c.classifyShortCond(expr, scope.shortVars) {
			case shortCondNegated:
				c.addViolation(expr.Pos(), msgNoShortNegatedGuard)
			case shortCondPositive:
				if len(cc.Body) == 0 && len(clauses) > 1 {
					c.addViolation(cc.Pos(), msgNoShortEmptyBranch)
				}
				caseGuarded = true
			}
		}
		c.walkStmts(cc.Body, shortWalkScope{
			shortVars:    cloneShortVars(scope.shortVars),
			inShortGuard: caseGuarded,
		})
	}
}

func (c *shortSkipChecker) walkTaggedSwitch(tag ast.Expr, clauses []ast.Stmt, scope shortWalkScope) {
	tagKind := c.classifyShortCond(tag, scope.shortVars)
	if tagKind == shortCondNegated {
		c.addViolation(tag.Pos(), msgNoShortNegatedGuard)
	}
	if tagKind != shortCondPositive {
		c.walkSwitchClauses(clauses, scope)
		return
	}
	hasTrueBody := hasNonEmptyTrueCase(clauses)
	for _, raw := range clauses {
		cc := raw.(*ast.CaseClause)
		caseGuarded := scope.inShortGuard
		if caseHasBoolLit(cc, boolLitTrue) {
			if len(cc.Body) == 0 && len(clauses) > 1 {
				c.addViolation(cc.Pos(), msgNoShortEmptyBranch)
			}
			caseGuarded = true
		} else if caseHasBoolLit(cc, boolLitFalse) && !hasTrueBody {
			c.addViolation(cc.Pos(), msgNoShortNegatedGuard)
		}
		c.walkStmts(cc.Body, shortWalkScope{
			shortVars:    cloneShortVars(scope.shortVars),
			inShortGuard: caseGuarded,
		})
	}
}

func (c *shortSkipChecker) walkSwitchClauses(clauses []ast.Stmt, scope shortWalkScope) {
	for _, raw := range clauses {
		cc := raw.(*ast.CaseClause)
		c.walkStmts(cc.Body, shortWalkScope{
			shortVars:    cloneShortVars(scope.shortVars),
			inShortGuard: scope.inShortGuard,
		})
	}
}

func (c *shortSkipChecker) scopeWithInit(initStmt ast.Stmt, scope shortWalkScope) shortWalkScope {
	if initStmt == nil {
		return scope
	}
	initScope := shortWalkScope{
		shortVars:    cloneShortVars(scope.shortVars),
		inShortGuard: scope.inShortGuard,
	}
	c.walkNode(initStmt, initScope)
	return initScope
}

func (c *shortSkipChecker) recordAssignBindings(stmt *ast.AssignStmt, shortVars map[string]shortCondKind) {
	for i, lhs := range stmt.Lhs {
		if i >= len(stmt.Rhs) {
			break
		}
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		c.recordShortBinding(ident.Name, stmt.Rhs[i], shortVars)
	}
}

func (c *shortSkipChecker) recordValueSpecBindings(spec *ast.ValueSpec, shortVars map[string]shortCondKind) {
	for i, ident := range spec.Names {
		if i >= len(spec.Values) {
			break
		}
		c.recordShortBinding(ident.Name, spec.Values[i], shortVars)
	}
}

func (c *shortSkipChecker) recordShortBinding(name string, rhs ast.Expr, shortVars map[string]shortCondKind) {
	if name == "_" {
		return
	}
	if kind := c.classifyShortCond(rhs, shortVars); kind != shortCondNone {
		shortVars[name] = kind
		return
	}
	delete(shortVars, name)
}

func (c *shortSkipChecker) classifyShortCond(expr ast.Expr, shortVars map[string]shortCondKind) shortCondKind {
	expr = unwrapParens(expr)
	if isTestingShortCall(expr, c.testingName) {
		return shortCondPositive
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return shortVars[e.Name]
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return negateShortCond(c.classifyShortCond(e.X, shortVars))
		}
	case *ast.BinaryExpr:
		return c.classifyShortBinary(e, shortVars)
	}
	return shortCondNone
}

func (c *shortSkipChecker) classifyShortBinary(expr *ast.BinaryExpr, shortVars map[string]shortCondKind) shortCondKind {
	switch expr.Op {
	case token.EQL, token.NEQ:
		return c.classifyShortEquality(expr.Op, expr.X, expr.Y, shortVars)
	case token.LAND, token.LOR:
		return combineShortLogical(
			c.classifyShortCond(expr.X, shortVars),
			c.classifyShortCond(expr.Y, shortVars),
		)
	default:
		return shortCondNone
	}
}

func (c *shortSkipChecker) classifyShortEquality(
	op token.Token,
	left ast.Expr,
	right ast.Expr,
	shortVars map[string]shortCondKind,
) shortCondKind {
	if lit := boolLiteralKind(unwrapParens(right)); lit != boolLitNone {
		return applyBoolComparison(c.classifyShortCond(left, shortVars), (op == token.NEQ) == (lit == boolLitTrue))
	}
	if lit := boolLiteralKind(unwrapParens(left)); lit != boolLitNone {
		return applyBoolComparison(c.classifyShortCond(right, shortVars), (op == token.NEQ) == (lit == boolLitTrue))
	}
	return shortCondNone
}

func (c *shortSkipChecker) addViolation(pos token.Pos, msg string) {
	p := c.fset.Position(pos)
	c.violations = append(c.violations, ConventionViolation{
		File:    c.relPath,
		Line:    p.Line,
		Column:  p.Column,
		Rule:    RuleNoShortSkip,
		Message: msg,
	})
}

func cloneShortVars(parent map[string]shortCondKind) map[string]shortCondKind {
	cloned := make(map[string]shortCondKind, len(parent))
	maps.Copy(cloned, parent)
	return cloned
}

func hasNonEmptyTrueCase(clauses []ast.Stmt) bool {
	for _, raw := range clauses {
		cc := raw.(*ast.CaseClause)
		if len(cc.Body) > 0 && caseHasBoolLit(cc, boolLitTrue) {
			return true
		}
	}
	return false
}

func caseHasBoolLit(cc *ast.CaseClause, want boolLitKind) bool {
	for _, expr := range cc.List {
		if boolLiteralKind(unwrapParens(expr)) == want {
			return true
		}
	}
	return false
}

func isTestingShortCall(expr ast.Expr, testingName string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	if testingName == "." {
		ident, ok := call.Fun.(*ast.Ident)
		return ok && ident.Name == "Short"
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Short" {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	return ok && pkgIdent.Name == testingName
}

func isSkipCall(call *ast.CallExpr) bool {
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

func applyBoolComparison(kind shortCondKind, invert bool) shortCondKind {
	if invert {
		return negateShortCond(kind)
	}
	return kind
}

func negateShortCond(kind shortCondKind) shortCondKind {
	switch kind {
	case shortCondPositive:
		return shortCondNegated
	case shortCondNegated:
		return shortCondPositive
	default:
		return shortCondNone
	}
}

func combineShortLogical(left shortCondKind, right shortCondKind) shortCondKind {
	if left == shortCondNegated || right == shortCondNegated {
		return shortCondNegated
	}
	if left == shortCondPositive || right == shortCondPositive {
		return shortCondPositive
	}
	return shortCondNone
}

func boolLiteralKind(expr ast.Expr) boolLitKind {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return boolLitNone
	}
	switch ident.Name {
	case "true":
		return boolLitTrue
	case "false":
		return boolLitFalse
	default:
		return boolLitNone
	}
}

func unwrapParens(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
}
