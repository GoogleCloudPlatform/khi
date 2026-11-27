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
	"maps"
	"slices"
)

type tableWalkContext struct {
	idx           *testFileIndex
	localStructs  map[string]bool
	testingParams map[string]bool
	visited       map[string]bool
}

func countTableElements(idx *testFileIndex, fd *ast.FuncDecl) int {
	ctx := &tableWalkContext{
		idx:           idx,
		localStructs:  maps.Clone(idx.localStructs),
		testingParams: collectTestingParamNames(fd),
		visited:       make(map[string]bool),
	}
	used, eligible, assigned := collectTableUses(ctx, fd.Body)
	markAssignedEligible(assigned, used, eligible)
	total := 0
	ast.Inspect(fd.Body, func(node ast.Node) bool {
		if comp, ok := node.(*ast.CompositeLit); ok && isCountedTableComp(comp, ctx.localStructs, eligible) {
			total += len(comp.Elts)
			// Skip nested slices inside a counted table row so they are not double-counted.
			return false
		}
		return true
	})
	return total
}

func markAssignedEligible(assigned map[string][]*ast.CompositeLit, used map[string]bool, eligible map[*ast.CompositeLit]bool) {
	for name, comps := range assigned {
		if !used[name] {
			continue
		}
		for _, comp := range comps {
			eligible[comp] = true
		}
	}
}

func collectTableUses(ctx *tableWalkContext, body *ast.BlockStmt) (map[string]bool, map[*ast.CompositeLit]bool, map[string][]*ast.CompositeLit) {
	used := make(map[string]bool)
	eligible := make(map[*ast.CompositeLit]bool)
	assigned := make(map[string][]*ast.CompositeLit)
	for node := range ast.Preorder(body) {
		switch n := node.(type) {
		case *ast.GenDecl:
			recordStructSpecs(n, ctx.localStructs)
		case *ast.AssignStmt, *ast.ValueSpec:
			recordAssignedTableNode(n, assigned)
		case *ast.RangeStmt:
			if blockHasTRun(ctx.idx, n.Body, ctx.testingParams, ctx.visited) {
				markTableExpr(n.X, eligible, used)
			}
		case *ast.CallExpr:
			recordCallTableArgUses(ctx, n, eligible, used)
		}
	}
	return used, eligible, assigned
}

func recordCallTableArgUses(ctx *tableWalkContext, call *ast.CallExpr, eligible map[*ast.CompositeLit]bool, used map[string]bool) {
	for argIdx, arg := range call.Args {
		if helperRunsTRunOnArg(ctx.idx, call, argIdx, ctx.visited) {
			markTableExpr(arg, eligible, used)
		}
	}
}

func recordAssignedTableNode(node ast.Node, assigned map[string][]*ast.CompositeLit) {
	if as, ok := node.(*ast.AssignStmt); ok && len(as.Lhs) == len(as.Rhs) {
		for i, rhs := range as.Rhs {
			recordAssignedIdentComp(as.Lhs[i], rhs, assigned)
		}
		return
	}
	if vs, ok := node.(*ast.ValueSpec); ok {
		for i, val := range vs.Values {
			if i < len(vs.Names) {
				recordAssignedIdentComp(vs.Names[i], val, assigned)
			}
		}
	}
}

func recordAssignedIdentComp(lhs ast.Expr, rhs ast.Expr, assigned map[string][]*ast.CompositeLit) {
	id, ok := lhs.(*ast.Ident)
	if !ok || id.Name == "_" {
		return
	}
	if comp, ok := ast.Unparen(rhs).(*ast.CompositeLit); ok {
		assigned[id.Name] = append(assigned[id.Name], comp)
	}
}

func markTableExpr(expr ast.Expr, eligible map[*ast.CompositeLit]bool, used map[string]bool) {
	switch e := ast.Unparen(expr).(type) {
	case *ast.CompositeLit:
		eligible[e] = true
	case *ast.Ident:
		used[e.Name] = true
	}
}

func helperRunsTRunOnArg(idx *testFileIndex, call *ast.CallExpr, argIdx int, visited map[string]bool) bool {
	fd, name, ok := idx.resolveTestingHelper(call)
	if !ok || visited[name] {
		return false
	}
	paramName := paramNameAt(fd, argIdx)
	if paramName == "" {
		return false
	}
	visited[name] = true
	defer delete(visited, name)
	// Pass nil for localStructs so helper-local type declarations never leak into idx.localStructs.
	helperCtx := &tableWalkContext{
		idx:           idx,
		localStructs:  nil,
		testingParams: collectTestingParamNames(fd),
		visited:       visited,
	}
	used, _, _ := collectTableUses(helperCtx, fd.Body)
	return used[paramName]
}

func paramNameAt(fd *ast.FuncDecl, argIdx int) string {
	if fd.Type.Params == nil || argIdx < 0 {
		return ""
	}
	pos := 0
	for _, f := range fd.Type.Params.List {
		for _, id := range f.Names {
			if pos == argIdx && id.Name != "_" {
				return id.Name
			}
			pos++
		}
		pos += max(1-len(f.Names), 0)
	}
	return ""
}

func blockHasTRun(idx *testFileIndex, body *ast.BlockStmt, testingParams map[string]bool, visited map[string]bool) bool {
	if body == nil {
		return false
	}
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || found {
			return n != nil && !found
		}
		if isTestingRunCall(call, testingParams) {
			found = true
		} else if fd, name, ok := idx.resolveTestingHelper(call); ok && !visited[name] {
			visited[name] = true
			found = blockHasTRun(idx, fd.Body, collectTestingParamNames(fd), visited)
			delete(visited, name)
		}
		return !found
	})
	return found
}

func isTestingRunCall(call *ast.CallExpr, testingParams map[string]bool) bool {
	sel, ok := unwrapCallFun(call.Fun).(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Run" || len(call.Args) < 2 {
		return false
	}
	id, ok := ast.Unparen(sel.X).(*ast.Ident)
	return ok && testingParams[id.Name]
}

func isCountedTableComp(comp *ast.CompositeLit, localStructs map[string]bool, eligible map[*ast.CompositeLit]bool) bool {
	arr, ok := comp.Type.(*ast.ArrayType)
	if !ok {
		return false
	}
	if _, isStruct := arr.Elt.(*ast.StructType); isStruct {
		return true
	}
	id, ok := unwrapCallFun(arr.Elt).(*ast.Ident)
	if !ok || !eligible[comp] || len(comp.Elts) == 0 || !localStructs[id.Name] {
		return false
	}
	return !slices.ContainsFunc(comp.Elts, func(elt ast.Expr) bool { _, isLit := elt.(*ast.CompositeLit); return !isLit })
}

func countStandaloneTRuns(body *ast.BlockStmt) int {
	count := 0
	walkStandaloneTRuns(body, 0, &count)
	return count
}

func walkStandaloneTRuns(root ast.Node, loopDepth int, count *int) {
	ast.Inspect(root, func(cur ast.Node) bool {
		if cur == nil {
			return false
		}
		_, isFor := cur.(*ast.ForStmt)
		_, isRange := cur.(*ast.RangeStmt)
		if cur != root && (isFor || isRange) {
			walkStandaloneTRuns(cur, loopDepth+1, count)
			return false
		}
		if call, ok := cur.(*ast.CallExpr); ok && loopDepth == 0 && isRunSubtestCall(call) {
			*count++
		}
		return true
	})
}

func isRunSubtestCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Run" || len(call.Args) < 2 {
		return false
	}
	_, isLit := call.Args[1].(*ast.FuncLit)
	return isLit
}
