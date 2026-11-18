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

package decisioncheck

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type testCaseExtractionCtx struct {
	fset     *token.FileSet
	targetID string
	relPath  string
	funcName string
	comments []*ast.CommentGroup
	lineSet  map[int]struct{}
}

// ExtractChangedGoTestFuncs extracts Go unit test functions (Test*) in *_test.go files overlapping changedLines.
func ExtractChangedGoTestFuncs(targetID string, files []BundledFile, changedLines map[string]map[int]struct{}) ([]ExtractedItem, error) {
	return extractGoFileItems(targetID, files, changedLines, parseTestGoFiles, collectTouchedTestFuncs)
}

// ExtractChangedGoTestCases extracts table-driven test case entries and standalone Test* functions overlapping changedLines.
func ExtractChangedGoTestCases(targetID string, files []BundledFile, changedLines map[string]map[int]struct{}) ([]ExtractedItem, error) {
	return extractGoFileItems(targetID, files, changedLines, parseTestGoFiles, collectFileTestCases)
}

func collectTouchedTestFuncs(fset *token.FileSet, targetID string, pf parsedGoFile, lineSet map[int]struct{}) []ExtractedItem {
	var out []ExtractedItem
	for _, decl := range pf.astFile.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || !gotestutil.IsGoUnitTestFunc(fd) {
			continue
		}
		declStart := fset.Position(fd.Pos()).Line
		spanStart := commentAwareStart(fset, fd.Doc, declStart)
		end := fset.Position(fd.End()).Line
		if !spanOverlaps(spanStart, end, lineSet) {
			continue
		}
		out = append(out, ExtractedItem{
			Target:    targetID,
			File:      pf.relPath,
			Line:      declStart,
			StartLine: spanStart,
			EndLine:   end,
			Name:      fd.Name.Name,
			Kind:      "test func",
			Doc:       cleanDocGroup(fd.Doc),
		})
	}
	return out
}

func collectFileTestCases(fset *token.FileSet, targetID string, pf parsedGoFile, lineSet map[int]struct{}) []ExtractedItem {
	pkgTables := collectPackageLevelComposites(pf.astFile)
	var out []ExtractedItem
	for _, decl := range pf.astFile.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || !gotestutil.IsGoUnitTestFunc(fd) || fd.Body == nil {
			continue
		}
		tables := findSubtestTableLiterals(fd.Body, pkgTables)
		if len(tables) == 0 {
			if item, matched := buildStandaloneTestCaseItem(fset, targetID, pf.relPath, fd, lineSet); matched {
				out = append(out, item)
			}
			continue
		}
		ctx := testCaseExtractionCtx{
			fset:     fset,
			targetID: targetID,
			relPath:  pf.relPath,
			funcName: fd.Name.Name,
			comments: pf.astFile.Comments,
			lineSet:  lineSet,
		}
		for _, tbl := range tables {
			out = append(out, extractTableElements(ctx, tbl)...)
		}
	}
	return out
}

func buildStandaloneTestCaseItem(fset *token.FileSet, targetID string, relPath string, fd *ast.FuncDecl, lineSet map[int]struct{}) (ExtractedItem, bool) {
	declStart := fset.Position(fd.Pos()).Line
	spanStart := commentAwareStart(fset, fd.Doc, declStart)
	end := fset.Position(fd.End()).Line
	if !spanOverlaps(spanStart, end, lineSet) {
		return ExtractedItem{}, false
	}
	return ExtractedItem{
		Target:    targetID,
		File:      relPath,
		Line:      declStart,
		StartLine: spanStart,
		EndLine:   end,
		Name:      fd.Name.Name,
		Kind:      "standalone test",
		Doc:       cleanDocGroup(fd.Doc),
	}, true
}

func collectPackageLevelComposites(f *ast.File) map[string]*ast.CompositeLit {
	tables := make(map[string]*ast.CompositeLit)
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		recordValueSpecComposites(gd.Specs, tables)
	}
	return tables
}

func findSubtestTableLiterals(body *ast.BlockStmt, pkgTables map[string]*ast.CompositeLit) []*ast.CompositeLit {
	localTables := collectLocalComposites(body, pkgTables)
	var out []*ast.CompositeLit
	seen := make(map[*ast.CompositeLit]struct{})
	ast.Inspect(body, func(n ast.Node) bool {
		rng, ok := n.(*ast.RangeStmt)
		if !ok || !blockContainsSubtestRun(rng.Body) {
			return true
		}
		lit := resolveRangeCompositeLit(rng.X, localTables)
		if lit == nil || len(lit.Elts) == 0 {
			return true
		}
		if _, exists := seen[lit]; !exists {
			seen[lit] = struct{}{}
			out = append(out, lit)
		}
		return true
	})
	return out
}

func collectLocalComposites(body *ast.BlockStmt, pkgTables map[string]*ast.CompositeLit) map[string]*ast.CompositeLit {
	locals := make(map[string]*ast.CompositeLit, len(pkgTables))
	for name, lit := range pkgTables {
		locals[name] = lit
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			recordAssignedComposites(stmt, locals)
		case *ast.DeclStmt:
			if gd := stmt.Decl.(*ast.GenDecl); gd.Tok == token.VAR {
				recordValueSpecComposites(gd.Specs, locals)
			}
		}
		return true
	})
	return locals
}

func recordAssignedComposites(stmt *ast.AssignStmt, dst map[string]*ast.CompositeLit) {
	if len(stmt.Lhs) != len(stmt.Rhs) {
		return
	}
	for i, lhs := range stmt.Lhs {
		if id, ok := lhs.(*ast.Ident); ok {
			recordCompositeLiteral(id.Name, stmt.Rhs[i], dst)
		}
	}
}

func recordValueSpecComposites(specs []ast.Spec, dst map[string]*ast.CompositeLit) {
	for _, sp := range specs {
		vs := sp.(*ast.ValueSpec)
		if len(vs.Names) != len(vs.Values) {
			continue
		}
		for i, id := range vs.Names {
			recordCompositeLiteral(id.Name, vs.Values[i], dst)
		}
	}
}

func recordCompositeLiteral(name string, expr ast.Expr, dst map[string]*ast.CompositeLit) {
	if lit, isLit := expr.(*ast.CompositeLit); isLit && isCollectionComposite(lit) {
		dst[name] = lit
	}
}

func resolveRangeCompositeLit(expr ast.Expr, tables map[string]*ast.CompositeLit) *ast.CompositeLit {
	switch typedExpr := expr.(type) {
	case *ast.CompositeLit:
		return typedExpr
	case *ast.Ident:
		return tables[typedExpr.Name]
	default:
		return nil
	}
}

func isCollectionComposite(lit *ast.CompositeLit) bool {
	switch lit.Type.(type) {
	case *ast.ArrayType, *ast.MapType:
		return true
	default:
		return false
	}
}

func blockContainsSubtestRun(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, isSel := call.Fun.(*ast.SelectorExpr); isSel && sel.Sel.Name == "Run" {
				found = true
			}
		}
		return !found
	})
	return found
}

func extractTableElements(ctx testCaseExtractionCtx, tbl *ast.CompositeLit) []ExtractedItem {
	prevBound := ctx.fset.Position(tbl.Lbrace).Line
	var out []ExtractedItem
	for idx, elt := range tbl.Elts {
		eltStart := ctx.fset.Position(elt.Pos()).Line
		eltEnd := ctx.fset.Position(elt.End()).Line
		docText, spanStart := findTableElementComment(ctx.fset, ctx.comments, prevBound, elt, eltStart)
		prevBound = eltEnd
		if !spanOverlaps(spanStart, eltEnd, ctx.lineSet) {
			continue
		}
		caseLabel := extractTableCaseLabel(elt, idx)
		out = append(out, ExtractedItem{
			Target:    ctx.targetID,
			File:      ctx.relPath,
			Line:      eltStart,
			StartLine: spanStart,
			EndLine:   eltEnd,
			Name:      fmt.Sprintf("%s / %q", ctx.funcName, caseLabel),
			Kind:      "test case",
			Doc:       docText,
		})
	}
	return out
}

func findTableElementComment(fset *token.FileSet, groups []*ast.CommentGroup, prevBound int, elt ast.Expr, eltStart int) (string, int) {
	if docText, startLine, ok := findCommentAboveElement(fset, groups, prevBound, eltStart); ok {
		return docText, startLine
	}
	firstInner := firstElementChildLine(fset, elt)
	return findCommentInsideBrace(fset, groups, eltStart, firstInner), eltStart
}

func findCommentAboveElement(fset *token.FileSet, groups []*ast.CommentGroup, prevBound int, eltStart int) (string, int, bool) {
	for _, cg := range groups {
		cgStart := fset.Position(cg.Pos()).Line
		cgEnd := fset.Position(cg.End()).Line
		if cgStart <= prevBound || cgEnd != eltStart-1 {
			continue
		}
		if txt := cleanDocGroup(cg); txt != "" {
			return txt, cgStart, true
		}
	}
	return "", 0, false
}

func findCommentInsideBrace(fset *token.FileSet, groups []*ast.CommentGroup, eltStart int, firstInner int) string {
	for _, cg := range groups {
		cgStart := fset.Position(cg.Pos()).Line
		cgEnd := fset.Position(cg.End()).Line
		if cgStart < eltStart || cgEnd >= firstInner {
			continue
		}
		if txt := cleanDocGroup(cg); txt != "" {
			return txt
		}
	}
	return ""
}

func firstElementChildLine(fset *token.FileSet, elt ast.Expr) int {
	lit := unwrapCaseComposite(elt)
	if lit == nil || len(lit.Elts) == 0 {
		return 0
	}
	return fset.Position(lit.Elts[0].Pos()).Line
}

func extractTableCaseLabel(elt ast.Expr, idx int) string {
	if kv, ok := elt.(*ast.KeyValueExpr); ok {
		if keyStr, _ := gotestutil.StringLiteralValue(kv.Key); keyStr != "" {
			return keyStr
		}
	}
	if lit := unwrapCaseComposite(elt); lit != nil {
		if name := findStructCaseNameField(lit); name != "" {
			return name
		}
	}
	return fmt.Sprintf("case #%d", idx+1)
}

func unwrapCaseComposite(elt ast.Expr) *ast.CompositeLit {
	if kv, ok := elt.(*ast.KeyValueExpr); ok {
		elt = kv.Value
	}
	lit, _ := elt.(*ast.CompositeLit)
	return lit
}

func findStructCaseNameField(lit *ast.CompositeLit) string {
	for _, field := range lit.Elts {
		kv, ok := field.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		id, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch id.Name {
		case "name", "Name", "desc", "Desc", "testName":
			if label, _ := gotestutil.StringLiteralValue(kv.Value); label != "" {
				return label
			}
		}
	}
	if len(lit.Elts) > 0 {
		label, _ := gotestutil.StringLiteralValue(lit.Elts[0])
		return label
	}
	return ""
}
