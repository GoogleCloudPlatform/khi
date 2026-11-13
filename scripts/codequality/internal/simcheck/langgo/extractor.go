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

package langgo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck"
)

type extractFileContext struct {
	relPath      string
	modulePath   string
	fset         *token.FileSet
	src          []byte
	codeMask     []bool
	minStmts     int
	minCodeLines int
}

// ExtractUnitsFromSource parses src and returns qualifying Go functions, methods, and top-level task closures.
func ExtractUnitsFromSource(relPath string, src []byte, minStmts int, minCodeLines int) ([]simcheck.Unit, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", relPath, err)
	}
	if astcheck.IsGeneratedAST(file) {
		return nil, nil
	}

	ctx := extractFileContext{
		relPath:      filepath.ToSlash(relPath),
		modulePath:   filepath.ToSlash(filepath.Dir(relPath)),
		fset:         fset,
		src:          src,
		codeMask:     rule.BuildCodeLineMask(fset, file, src),
		minStmts:     minStmts,
		minCodeLines: minCodeLines,
	}

	var units []simcheck.Unit
	for _, decl := range file.Decls {
		units = appendDeclUnits(ctx, decl, units)
	}
	return units, nil
}

func appendDeclUnits(ctx extractFileContext, decl ast.Decl, out []simcheck.Unit) []simcheck.Unit {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if u, ok := buildFuncDeclUnit(ctx, d); ok {
			return append(out, u)
		}
	case *ast.GenDecl:
		if d.Tok == token.VAR {
			return appendVarClosureUnits(ctx, d, out)
		}
	}
	return out
}

func buildFuncDeclUnit(ctx extractFileContext, d *ast.FuncDecl) (simcheck.Unit, bool) {
	if d.Body == nil {
		return simcheck.Unit{}, false
	}
	startLine := ctx.fset.Position(d.Pos()).Line
	endLine := ctx.fset.Position(d.End()).Line
	stmts := astcheck.CountBlockStatements(d.Body)
	codeLines := astcheck.CountTrueInRange(ctx.codeMask, startLine, endLine)
	if stmts < ctx.minStmts || codeLines < ctx.minCodeLines {
		return simcheck.Unit{}, false
	}

	recv := receiverTypeName(d.Recv)
	kind := simcheck.KindFunc
	qualName := d.Name.Name
	if recv != "" {
		kind = simcheck.KindMethod
		qualName = recv + "." + d.Name.Name
	}

	declBytes := sliceNodeBytes(ctx.fset, ctx.src, d.Pos(), d.End())
	return simcheck.Unit{
		ID:         fmt.Sprintf("%s:%s", ctx.relPath, qualName),
		Key:        fmt.Sprintf("%s:%s", kind, qualName),
		Language:   simcheck.LangGo,
		ModulePath: ctx.modulePath,
		FilePath:   ctx.relPath,
		StartLine:  startLine,
		EndLine:    endLine,
		Name:       qualName,
		Kind:       kind,
		StmtCount:  stmts,
		CodeLines:  codeLines,
		NormText:   NormalizeDeclBytes(declBytes),
		Preview:    simcheck.BuildPreviewLines(string(declBytes), startLine),
	}, true
}

func appendVarClosureUnits(ctx extractFileContext, d *ast.GenDecl, out []simcheck.Unit) []simcheck.Unit {
	for _, spec := range d.Specs {
		vs := spec.(*ast.ValueSpec)
		if len(vs.Values) == 0 {
			continue
		}
		if u, ok := buildVarClosureUnit(ctx, d, vs); ok {
			out = append(out, u)
		}
	}
	return out
}

func buildVarClosureUnit(ctx extractFileContext, d *ast.GenDecl, vs *ast.ValueSpec) (simcheck.Unit, bool) {
	stmts, hasClosure := countExprClosureStmts(vs.Values)
	if !hasClosure {
		return simcheck.Unit{}, false
	}

	specStart, specEnd := resolveVarSpecSpan(d, vs)
	startLine := ctx.fset.Position(specStart).Line
	endLine := ctx.fset.Position(specEnd).Line
	codeLines := astcheck.CountTrueInRange(ctx.codeMask, startLine, endLine)
	if stmts < ctx.minStmts || codeLines < ctx.minCodeLines {
		return simcheck.Unit{}, false
	}

	rawSpan := sliceNodeBytes(ctx.fset, ctx.src, specStart, specEnd)
	declBytes := rawSpan
	if len(d.Specs) > 1 {
		declBytes = append([]byte("var "), rawSpan...)
	}
	varName := vs.Names[0].Name

	return simcheck.Unit{
		ID:         fmt.Sprintf("%s:%s", ctx.relPath, varName),
		Key:        fmt.Sprintf("%s:%s", simcheck.KindClosure, varName),
		Language:   simcheck.LangGo,
		ModulePath: ctx.modulePath,
		FilePath:   ctx.relPath,
		StartLine:  startLine,
		EndLine:    endLine,
		Name:       varName,
		Kind:       simcheck.KindClosure,
		StmtCount:  stmts,
		CodeLines:  codeLines,
		NormText:   NormalizeDeclBytes(declBytes),
		Preview:    simcheck.BuildPreviewLines(string(rawSpan), startLine),
	}, true
}

func resolveVarSpecSpan(d *ast.GenDecl, vs *ast.ValueSpec) (token.Pos, token.Pos) {
	if len(d.Specs) == 1 {
		return d.Pos(), d.End()
	}
	return vs.Pos(), vs.End()
}

func countExprClosureStmts(exprs []ast.Expr) (int, bool) {
	total := 0
	found := false
	for _, expr := range exprs {
		ast.Inspect(expr, func(n ast.Node) bool {
			if fl, ok := n.(*ast.FuncLit); ok && fl.Body != nil {
				found = true
				total += astcheck.CountBlockStatements(fl.Body)
				return false
			}
			return true
		})
	}
	return total, found
}

func sliceNodeBytes(fset *token.FileSet, src []byte, start token.Pos, end token.Pos) []byte {
	s := fset.Position(start).Offset
	e := fset.Position(end).Offset
	return src[s:e]
}

func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	return baseTypeIdent(recv.List[0].Type)
}

func baseTypeIdent(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return baseTypeIdent(t.X)
	case *ast.IndexExpr:
		return baseTypeIdent(t.X)
	case *ast.IndexListExpr:
		return baseTypeIdent(t.X)
	default:
		return expr.(*ast.Ident).Name
	}
}
