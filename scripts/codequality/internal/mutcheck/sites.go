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

package mutcheck

import (
	"go/ast"
	"go/token"
	"strings"
)

func isMutatableAssignTok(tok token.Token) bool {
	return tok == token.ASSIGN || tok == token.ADD_ASSIGN || tok == token.SUB_ASSIGN
}

func collectBlockListStmts(file *ast.File) map[ast.Stmt]struct{} {
	members := make(map[ast.Stmt]struct{})
	ast.Inspect(file, func(n ast.Node) bool {
		switch b := n.(type) {
		case *ast.BlockStmt:
			addStmtSlice(members, b.List)
		case *ast.CaseClause:
			addStmtSlice(members, b.Body)
		case *ast.CommClause:
			addStmtSlice(members, b.Body)
		}
		return true
	})
	return members
}

func addStmtSlice(dst map[ast.Stmt]struct{}, stmts []ast.Stmt) {
	for _, st := range stmts {
		dst[st] = struct{}{}
	}
}

func shouldApplyBCR(cond ast.Expr) bool {
	switch e := ast.Unparen(cond).(type) {
	case *ast.BinaryExpr:
		return false
	case *ast.UnaryExpr:
		return e.Op != token.NOT
	case *ast.Ident:
		return e.Name != "true" && e.Name != "false"
	default:
		return true
	}
}

func isZeroOrEmptyLiteral(expr ast.Expr) bool {
	lit, ok := ast.Unparen(expr).(*ast.BasicLit)
	if !ok {
		return false
	}
	switch lit.Kind {
	case token.INT, token.FLOAT:
		return lit.Value == "0" || lit.Value == "0.0"
	case token.STRING:
		return lit.Value == `""` || lit.Value == "``"
	default:
		return false
	}
}

func isAllBlankLHS(lhs []ast.Expr) bool {
	for _, e := range lhs {
		ident, ok := ast.Unparen(e).(*ast.Ident)
		if !ok || ident.Name != "_" {
			return false
		}
	}
	return true
}

func isStandardBoolReturnSlot(ancestors []ast.Node, ret *ast.ReturnStmt, target *ast.Ident) bool {
	slotIdx := findReturnSlot(ret.Results, target)
	if slotIdx < 0 {
		return false
	}
	funcResults := findEnclosingFuncResults(ancestors)
	if funcResults == nil {
		return false
	}
	slotTypes := expandResultTypes(funcResults)
	if slotIdx >= len(slotTypes) {
		return false
	}
	return isBoolOrAnyTypeExpr(slotTypes[slotIdx])
}

func findReturnSlot(results []ast.Expr, target *ast.Ident) int {
	for i, res := range results {
		if res == target {
			return i
		}
	}
	return -1
}

func findEnclosingFuncResults(ancestors []ast.Node) *ast.FieldList {
	for i := len(ancestors) - 1; i >= 0; i-- {
		switch fn := ancestors[i].(type) {
		case *ast.FuncLit:
			return fn.Type.Results
		case *ast.FuncDecl:
			return fn.Type.Results
		}
	}
	return nil
}

func expandResultTypes(fl *ast.FieldList) []ast.Expr {
	var out []ast.Expr
	for _, field := range fl.List {
		count := max(len(field.Names), 1)
		for range count {
			out = append(out, field.Type)
		}
	}
	return out
}

func isBoolOrAnyTypeExpr(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name == "bool" || t.Name == "any"
	case *ast.InterfaceType:
		return t.Methods == nil || len(t.Methods.List) == 0
	default:
		return false
	}
}

func hasBuildIgnoreDirective(file *ast.File) bool {
	for _, group := range file.Comments {
		if group.Pos() > file.Package {
			break
		}
		for _, c := range group.List {
			txt := strings.TrimSpace(c.Text)
			if txt == "//go:build ignore" || txt == "// +build ignore" {
				return true
			}
		}
	}
	return false
}

func computeLineOffsets(src []byte) []int {
	starts := []int{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func (w *funcWalker) buildDiffLines(line int, site MutationSite, replacement string) (string, string) {
	origLine := firstLineTrimmed(w.src, w.lineStarts, line)
	lineStart, lineEnd := lineByteBounds(w.src, w.lineStarts, line)
	if site.StartOffset >= lineStart && site.EndOffset <= lineEnd {
		var sb strings.Builder
		sb.Write(w.src[lineStart:site.StartOffset])
		sb.WriteString(replacement)
		sb.Write(w.src[site.EndOffset:lineEnd])
		return strings.TrimSpace(string(w.src[lineStart:lineEnd])), strings.TrimSpace(sb.String())
	}
	return origLine, replacement
}

func firstLineTrimmed(src []byte, lineStarts []int, line int) string {
	lineStart, lineEnd := lineByteBounds(src, lineStarts, line)
	return strings.TrimSpace(string(src[lineStart:lineEnd]))
}

func lineByteBounds(src []byte, lineStarts []int, line int) (int, int) {
	lineStart := lineStarts[line-1]
	lineEnd := len(src)
	if line < len(lineStarts) {
		lineEnd = lineStarts[line] - 1
	}
	if lineEnd > lineStart && src[lineEnd-1] == '\r' {
		lineEnd--
	}
	return lineStart, lineEnd
}

func compactWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
