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
	"maps"
	"slices"
)

type goTestBodyMetrics struct {
	cmpAsserts    int
	scalarAsserts int
	totalAsserts  int
	calledHelpers map[string]bool
}

type testFileIndex struct {
	localStructs map[string]bool
	funcDecls    map[string]*ast.FuncDecl
	helperMemo   map[string]goTestBodyMetrics
}

type metricContext struct {
	idx           *testFileIndex
	testingParams map[string]bool
	localVars     map[string]bool
	visiting      map[string]bool
}

func newTestFileIndex(file *ast.File) *testFileIndex {
	idx := &testFileIndex{localStructs: make(map[string]bool), funcDecls: make(map[string]*ast.FuncDecl), helperMemo: make(map[string]goTestBodyMetrics)}
	if file == nil {
		return idx
	}
	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			recordStructSpecs(gd, idx.localStructs)
		} else if fd, ok := decl.(*ast.FuncDecl); ok && fd.Body != nil && fd.Recv == nil {
			idx.funcDecls[fd.Name.Name] = fd
		}
	}
	return idx
}

func (idx *testFileIndex) resolveTestingHelper(call *ast.CallExpr) (*ast.FuncDecl, string, bool) {
	ident, ok := unwrapCallFun(call.Fun).(*ast.Ident)
	if !ok {
		return nil, "", false
	}
	fd := idx.funcDecls[ident.Name]
	return fd, ident.Name, fd != nil && hasTestingParam(fd)
}

func (idx *testFileIndex) evalHelperMetrics(name string, fd *ast.FuncDecl, visiting map[string]bool) (goTestBodyMetrics, bool) {
	if cached, ok := idx.helperMemo[name]; ok {
		return cached, true
	}
	visiting[name] = true
	defer delete(visiting, name)
	ctx := &metricContext{
		idx:           idx,
		testingParams: collectTestingParamNames(fd),
		localVars:     collectLocalBoundIdents(fd),
		visiting:      visiting,
	}
	metrics, acyclic := collectBodyMetrics(ctx, fd.Body)
	if acyclic {
		idx.helperMemo[name] = metrics
	}
	return metrics, acyclic
}

func recordStructSpecs(gd *ast.GenDecl, out map[string]bool) {
	if out == nil || gd.Tok != token.TYPE {
		return
	}
	for _, spec := range gd.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok {
			if _, isStruct := ts.Type.(*ast.StructType); isStruct {
				out[ts.Name.Name] = true
			}
		}
	}
}

func hasTestingParam(fd *ast.FuncDecl) bool {
	return fd.Type.Params != nil && slices.ContainsFunc(fd.Type.Params.List, func(f *ast.Field) bool { return isTestingTypeExpr(f.Type) })
}

func collectTestingParamNames(fd *ast.FuncDecl) map[string]bool {
	params := make(map[string]bool)
	addTestingFieldParams(fd.Type.Params, params)
	for node := range ast.Preorder(fd.Body) {
		if lit, ok := node.(*ast.FuncLit); ok {
			addTestingFieldParams(lit.Type.Params, params)
		}
	}
	return params
}

func addTestingFieldParams(fields *ast.FieldList, out map[string]bool) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		if isTestingTypeExpr(field.Type) {
			for _, id := range field.Names {
				if id.Name != "_" {
					out[id.Name] = true
				}
			}
		}
	}
}

func collectLocalBoundIdents(fd *ast.FuncDecl) map[string]bool {
	locals := make(map[string]bool)
	addFieldListIdents(fd.Type.Params, locals)
	addFieldListIdents(fd.Type.Results, locals)
	for node := range ast.Preorder(fd.Body) {
		recordLocalBindingNode(node, locals)
	}
	return locals
}

func recordLocalBindingNode(node ast.Node, locals map[string]bool) {
	switch n := node.(type) {
	case *ast.FuncLit:
		addFieldListIdents(n.Type.Params, locals)
		addFieldListIdents(n.Type.Results, locals)
	case *ast.AssignStmt:
		if n.Tok == token.DEFINE {
			for _, expr := range n.Lhs {
				addIdentName(expr, locals)
			}
		}
	case *ast.RangeStmt:
		if n.Tok == token.DEFINE {
			addIdentName(n.Key, locals)
			addIdentName(n.Value, locals)
		}
	case *ast.ValueSpec:
		for _, id := range n.Names {
			addIdentName(id, locals)
		}
	}
}

func addFieldListIdents(fields *ast.FieldList, out map[string]bool) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		for _, id := range field.Names {
			addIdentName(id, out)
		}
	}
}

func addIdentName(expr ast.Expr, out map[string]bool) {
	if id, ok := expr.(*ast.Ident); ok && id.Name != "_" {
		out[id.Name] = true
	}
}

func isTestingTypeExpr(expr ast.Expr) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		sel, ok := star.X.(*ast.SelectorExpr)
		return ok && isIdentNamed(sel.X, "testing") && (sel.Sel.Name == "T" || sel.Sel.Name == "B")
	}
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && isIdentNamed(sel.X, "testing") && sel.Sel.Name == "TB"
}

func isIdentNamed(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

func inspectGoTestBodyMetrics(idx *testFileIndex, fd *ast.FuncDecl) goTestBodyMetrics {
	ctx := &metricContext{
		idx:           idx,
		testingParams: collectTestingParamNames(fd),
		localVars:     collectLocalBoundIdents(fd),
		visiting:      make(map[string]bool),
	}
	metrics, _ := collectBodyMetrics(ctx, fd.Body)
	return metrics
}

func collectBodyMetrics(ctx *metricContext, body *ast.BlockStmt) (goTestBodyMetrics, bool) {
	metrics := goTestBodyMetrics{calledHelpers: make(map[string]bool)}
	acyclic := true
	cmpVars := collectCmpResultVars(body)
	ast.Inspect(body, func(node ast.Node) bool {
		descend, nodeAcyclic := visitBodyMetricNode(ctx, node, cmpVars, &metrics)
		if !nodeAcyclic {
			acyclic = false
		}
		return descend
	})
	return metrics, acyclic
}

func visitBodyMetricNode(
	ctx *metricContext,
	node ast.Node,
	cmpVars map[string]bool,
	metrics *goTestBodyMetrics,
) (bool, bool) {
	if node == nil {
		return false, true
	}
	if ifStmt, ok := node.(*ast.IfStmt); ok {
		if blockInvokesTestingAssert(ifStmt.Body, ctx.testingParams) && exprHasScalarEquality(ifStmt.Cond, cmpVars) {
			metrics.scalarAsserts++
		}
		return true, true
	}
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return true, true
	}
	// Skip descending into testing assert args so formatting-only cmp.Diff(...) inside t.Errorf does not double-count.
	if isTestingMethodAssert(call, ctx.testingParams) {
		metrics.totalAsserts++
		return false, true
	}
	return true, recordCallMetric(ctx, call, metrics)
}

func recordCallMetric(ctx *metricContext, call *ast.CallExpr, metrics *goTestBodyMetrics) bool {
	if isCmpCallExpr(call) {
		metrics.cmpAsserts++
		metrics.totalAsserts++
		return true
	}
	switch fun := unwrapCallFun(call.Fun).(type) {
	case *ast.Ident:
		return recordIdentCallMetric(ctx, call, fun.Name, metrics)
	case *ast.SelectorExpr:
		if pkg, ok := fun.X.(*ast.Ident); ok && !ctx.testingParams[pkg.Name] && !isLogCall(pkg.Name, fun.Sel.Name) && callPassesTestingArg(call, ctx.testingParams) {
			metrics.calledHelpers[pkg.Name+"."+fun.Sel.Name] = true
		}
	}
	return true
}

func recordIdentCallMetric(ctx *metricContext, call *ast.CallExpr, name string, metrics *goTestBodyMetrics) bool {
	fd := ctx.idx.funcDecls[name]
	if fd == nil {
		// Locally bound variables/closures have their bodies already counted inside body traversal.
		if !ctx.localVars[name] && !isLogCall("", name) && callPassesTestingArg(call, ctx.testingParams) {
			metrics.calledHelpers[name] = true
		}
		return true
	}
	if !hasTestingParam(fd) {
		return true
	}
	if ctx.visiting[name] {
		return false
	}
	hm, acyclic := ctx.idx.evalHelperMetrics(name, fd, ctx.visiting)
	metrics.cmpAsserts += hm.cmpAsserts
	metrics.scalarAsserts += hm.scalarAsserts
	metrics.totalAsserts += hm.totalAsserts
	if hm.totalAsserts > 0 || len(hm.calledHelpers) > 0 {
		metrics.calledHelpers[name] = true
		maps.Copy(metrics.calledHelpers, hm.calledHelpers)
	}
	return acyclic
}

func unwrapCallFun(expr ast.Expr) ast.Expr {
	for {
		switch e := expr.(type) {
		case *ast.ParenExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		default:
			return expr
		}
	}
}

func callPassesTestingArg(call *ast.CallExpr, testingParams map[string]bool) bool {
	return slices.ContainsFunc(call.Args, func(arg ast.Expr) bool {
		id, ok := ast.Unparen(arg).(*ast.Ident)
		return ok && testingParams[id.Name]
	})
}

func isLogCall(pkgName string, fnName string) bool {
	return pkgName == "log" || pkgName == "slog" || slices.Contains([]string{"Log", "Logf", "Print", "Printf", "Println"}, fnName)
}

func collectCmpResultVars(body *ast.BlockStmt) map[string]bool {
	vars := make(map[string]bool)
	for node := range ast.Preorder(body) {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || !slices.ContainsFunc(assign.Rhs, exprHasCmpCall) {
			continue
		}
		for _, lhs := range assign.Lhs {
			if id, isID := lhs.(*ast.Ident); isID && id.Name != "_" {
				vars[id.Name] = true
			}
		}
	}
	return vars
}

func exprHasCmpCall(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && isCmpCallExpr(call) {
			found = true
		}
		return !found
	})
	return found
}

func blockInvokesTestingAssert(body *ast.BlockStmt, testingParams map[string]bool) bool {
	if body == nil {
		return false
	}
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if n == nil || found {
			return false
		}
		switch node := n.(type) {
		case *ast.FuncLit, *ast.IfStmt:
			return false
		case *ast.CallExpr:
			found = isTestingMethodAssert(node, testingParams)
			return !found
		}
		return true
	})
	return found
}

func exprHasScalarEquality(expr ast.Expr, cmpVars map[string]bool) bool {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return exprHasScalarEquality(e.X, cmpVars)
	case *ast.BinaryExpr:
		if e.Op == token.EQL || e.Op == token.NEQ {
			return !exprReferencesErrOrCmp(e.X, cmpVars) && !exprReferencesErrOrCmp(e.Y, cmpVars)
		}
		if e.Op == token.LOR || e.Op == token.LAND {
			return exprHasScalarEquality(e.X, cmpVars) || exprHasScalarEquality(e.Y, cmpVars)
		}
	}
	return false
}

func exprReferencesErrOrCmp(expr ast.Expr, cmpVars map[string]bool) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name == "err" || e.Name == "nil" || cmpVars[e.Name]
	case *ast.ParenExpr:
		return exprReferencesErrOrCmp(e.X, cmpVars)
	case *ast.SelectorExpr:
		// Inspect only the receiver so field names like tc.err do not suppress scalar checks.
		return exprReferencesErrOrCmp(e.X, cmpVars)
	case *ast.UnaryExpr:
		return exprReferencesErrOrCmp(e.X, cmpVars)
	case *ast.CallExpr:
		return isCmpCallExpr(e) || slices.ContainsFunc(e.Args, func(arg ast.Expr) bool { return exprReferencesErrOrCmp(arg, cmpVars) })
	case *ast.BinaryExpr:
		return exprReferencesErrOrCmp(e.X, cmpVars) || exprReferencesErrOrCmp(e.Y, cmpVars)
	}
	return false
}

func isCmpCallExpr(call *ast.CallExpr) bool {
	sel, ok := unwrapCallFun(call.Fun).(*ast.SelectorExpr)
	return ok && (sel.Sel.Name == "Diff" || sel.Sel.Name == "Equal") && isIdentNamed(sel.X, "cmp")
}

func isTestingMethodAssert(call *ast.CallExpr, testingParams map[string]bool) bool {
	sel, ok := unwrapCallFun(call.Fun).(*ast.SelectorExpr)
	if !ok || !slices.Contains([]string{"Error", "Errorf", "Fatal", "Fatalf", "Fail", "FailNow"}, sel.Sel.Name) {
		return false
	}
	return isTestingReceiverExpr(sel.X, testingParams)
}

func isTestingReceiverExpr(expr ast.Expr, testingParams map[string]bool) bool {
	switch r := ast.Unparen(expr).(type) {
	case *ast.Ident:
		if len(testingParams) > 0 {
			return testingParams[r.Name]
		}
		return r.Name == "t" || r.Name == "b" || r.Name == "tb"
	case *ast.SelectorExpr:
		return r.Sel.Name == "t" || r.Sel.Name == "b" || r.Sel.Name == "tb" ||
			r.Sel.Name == "T" || r.Sel.Name == "B" || r.Sel.Name == "TB" ||
			testingParams[r.Sel.Name]
	default:
		return false
	}
}
