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

package rule

import (
	"go/ast"
	"go/token"
	"strings"
)

// ExclusionScope specifies how far an AST boilerplate exclusion applies during traversal.
type ExclusionScope int

const (
	// ExclusionNone allows normal quality analysis on the node and its children.
	ExclusionNone ExclusionScope = iota
	// ExclusionNodeOnly skips mutating the node directly while still visiting its children.
	ExclusionNodeOnly
	// ExclusionSubtree skips the node and all of its AST descendants.
	ExclusionSubtree
)

// ExclusionContext carries file, token position, and ancestor chain state for ExclusionRule evaluation.
type ExclusionContext struct {
	FilePath  string
	Fset      *token.FileSet
	File      *ast.File
	Source    []byte
	Ancestors []ast.Node
}

// Parent returns the immediate enclosing AST node, or nil at the root.
func (c *ExclusionContext) Parent() ast.Node {
	if len(c.Ancestors) == 0 {
		return nil
	}
	return c.Ancestors[len(c.Ancestors)-1]
}

// ExclusionRule determines whether an AST node represents non-domain boilerplate.
type ExclusionRule interface {
	Name() string
	Evaluate(ctx *ExclusionContext, node ast.Node) ExclusionScope
}

// DefaultExclusionRules returns the shared KHI boilerplate suppression rules for coverage and mutation checks.
func DefaultExclusionRules() []ExclusionRule {
	return []ExclusionRule{
		&constAndTypeRule{},
		&errorGuardRule{},
		&loggingAndDiagCallRule{},
		&allocationHintRule{},
		&deferAndPanicRule{},
	}
}

// EvaluateExclusionRules runs rules against node and returns the strongest ExclusionScope matched.
func EvaluateExclusionRules(rules []ExclusionRule, ctx *ExclusionContext, node ast.Node) ExclusionScope {
	scope := ExclusionNone
	for _, r := range rules {
		switch r.Evaluate(ctx, node) {
		case ExclusionSubtree:
			return ExclusionSubtree
		case ExclusionNodeOnly:
			scope = ExclusionNodeOnly
		}
	}
	return scope
}

// BoilerplateLineSet returns 1-based line numbers excluded from func_uncovered_lines defect counts.
func BoilerplateLineSet(fset *token.FileSet, file *ast.File) map[int]struct{} {
	excluded := make(map[int]struct{})
	if file == nil {
		return excluded
	}
	isMainPkg := file.Name.Name == "main"
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		if isMainPkg && fd.Recv == nil && fd.Name.Name == "main" {
			markNodeLines(fset, fd, excluded)
			continue
		}
		collectFuncBoilerplateLines(fset, fd.Body, excluded)
	}
	return excluded
}

func collectFuncBoilerplateLines(fset *token.FileSet, body *ast.BlockStmt, excluded map[int]struct{}) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.IfStmt:
			if isErrorGuardIf(s) {
				markErrorGuardLines(fset, s, excluded)
			}
		case *ast.DeferStmt:
			markNodeLines(fset, s, excluded)
		case *ast.ExprStmt:
			if isDiagOrPanicCall(s.X) {
				markNodeLines(fset, s, excluded)
			}
		}
		return true
	})
}

func markErrorGuardLines(fset *token.FileSet, s *ast.IfStmt, excluded map[int]struct{}) {
	if s.Init == nil {
		markNodeLines(fset, s, excluded)
		return
	}
	for _, stmt := range s.Body.List {
		markNodeLines(fset, stmt, excluded)
	}
}

func markNodeLines(fset *token.FileSet, n ast.Node, excluded map[int]struct{}) {
	start := fset.Position(n.Pos()).Line
	end := fset.Position(n.End()).Line
	for ln := start; ln <= end; ln++ {
		excluded[ln] = struct{}{}
	}
}

type constAndTypeRule struct{}

var _ ExclusionRule = (*constAndTypeRule)(nil)

func (r *constAndTypeRule) Name() string { return "const_and_type" }

func (r *constAndTypeRule) Evaluate(_ *ExclusionContext, node ast.Node) ExclusionScope {
	switch n := node.(type) {
	case *ast.GenDecl:
		if n.Tok == token.CONST || n.Tok == token.TYPE {
			return ExclusionSubtree
		}
	case *ast.ArrayType, *ast.StructType, *ast.InterfaceType, *ast.MapType, *ast.ChanType, *ast.FuncType:
		return ExclusionSubtree
	case *ast.BinaryExpr:
		if isPureLiteralExpr(n) {
			return ExclusionSubtree
		}
	}
	return ExclusionNone
}

type errorGuardRule struct{}

var _ ExclusionRule = (*errorGuardRule)(nil)

func (r *errorGuardRule) Name() string { return "error_guard" }

func (r *errorGuardRule) Evaluate(_ *ExclusionContext, node ast.Node) ExclusionScope {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		if isErrorNilComparison(n) {
			return ExclusionSubtree
		}
	case *ast.IfStmt:
		if isErrorGuardIf(n) {
			return ExclusionSubtree
		}
	}
	return ExclusionNone
}

type loggingAndDiagCallRule struct{}

var _ ExclusionRule = (*loggingAndDiagCallRule)(nil)

func (r *loggingAndDiagCallRule) Name() string { return "logging_and_diag_call" }

func (r *loggingAndDiagCallRule) Evaluate(_ *ExclusionContext, node ast.Node) ExclusionScope {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return ExclusionNone
	}
	if isDiagSelectorCall(call, true) {
		return ExclusionSubtree
	}
	return ExclusionNone
}

type allocationHintRule struct{}

var _ ExclusionRule = (*allocationHintRule)(nil)

func (r *allocationHintRule) Name() string { return "allocation_hint" }

func (r *allocationHintRule) Evaluate(ctx *ExclusionContext, node ast.Node) ExclusionScope {
	if call, ok := node.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Grow" {
			return ExclusionSubtree
		}
	}
	if isMakeCapacityArg(ctx.Parent(), node) {
		return ExclusionSubtree
	}
	return ExclusionNone
}

func isMakeCapacityArg(parent ast.Node, node ast.Node) bool {
	call, ok := parent.(*ast.CallExpr)
	if !ok {
		return false
	}
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "make" {
		return false
	}
	if len(call.Args) == 3 && call.Args[2] == node {
		arr, ok := call.Args[0].(*ast.ArrayType)
		return ok && arr.Len == nil
	}
	if len(call.Args) == 2 && call.Args[1] == node {
		_, ok := call.Args[0].(*ast.MapType)
		return ok
	}
	return false
}

type deferAndPanicRule struct{}

var _ ExclusionRule = (*deferAndPanicRule)(nil)

func (r *deferAndPanicRule) Name() string { return "defer_and_panic" }

func (r *deferAndPanicRule) Evaluate(_ *ExclusionContext, node ast.Node) ExclusionScope {
	switch n := node.(type) {
	case *ast.DeferStmt:
		return ExclusionSubtree
	case *ast.CallExpr:
		if isPanicCall(n) {
			return ExclusionSubtree
		}
	}
	return ExclusionNone
}

func isDiagOrPanicCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	return isPanicCall(call) || isDiagSelectorCall(call, false)
}

func isPanicCall(call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	return ok && ident.Name == "panic"
}

func isDiagSelectorCall(call *ast.CallExpr, includeErrorConstructors bool) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	recv, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	method := sel.Sel.Name
	switch recv.Name {
	case "slog", "log", "klog", "otel", "span", "tracer":
		return true
	case "fmt":
		return strings.HasPrefix(method, "Print") || strings.HasPrefix(method, "Fprint") || (includeErrorConstructors && method == "Errorf")
	case "errors":
		return includeErrorConstructors && (method == "New" || method == "Join")
	default:
		return false
	}
}

func isErrorGuardIf(s *ast.IfStmt) bool {
	return s.Else == nil && IsErrorNotNilGuard(s.Cond) && endsWithReturnStmt(s.Body)
}

// IsErrorNotNilGuard reports whether expr compares an error-like expression against nil with !=.
func IsErrorNotNilGuard(expr ast.Expr) bool {
	bin, ok := ast.Unparen(expr).(*ast.BinaryExpr)
	return ok && bin.Op == token.NEQ && isErrorNilComparison(bin)
}

func isErrorNilComparison(bin *ast.BinaryExpr) bool {
	return (isNilIdent(bin.Y) && isErrorLikeExpr(bin.X)) || (isNilIdent(bin.X) && isErrorLikeExpr(bin.Y))
}

func isNilIdent(expr ast.Expr) bool {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	return ok && ident.Name == "nil"
}

func isErrorLikeExpr(expr ast.Expr) bool {
	switch e := ast.Unparen(expr).(type) {
	case *ast.Ident:
		return isErrorName(e.Name)
	case *ast.SelectorExpr:
		return isErrorName(e.Sel.Name)
	case *ast.CallExpr:
		sel, ok := e.Fun.(*ast.SelectorExpr)
		return ok && len(e.Args) == 0 && sel.Sel.Name == "Err"
	default:
		return false
	}
}

func isErrorName(name string) bool {
	return strings.EqualFold(name, "err") || strings.HasSuffix(name, "Err") || strings.HasSuffix(name, "Error")
}

func endsWithReturnStmt(block *ast.BlockStmt) bool {
	if len(block.List) == 0 {
		return false
	}
	_, ok := block.List[len(block.List)-1].(*ast.ReturnStmt)
	return ok
}

func isPureLiteralExpr(expr ast.Expr) bool {
	switch e := ast.Unparen(expr).(type) {
	case *ast.BasicLit:
		return true
	case *ast.UnaryExpr:
		return (e.Op == token.SUB || e.Op == token.ADD) && isPureLiteralExpr(e.X)
	case *ast.BinaryExpr:
		return isPureLiteralExpr(e.X) && isPureLiteralExpr(e.Y)
	default:
		return false
	}
}
