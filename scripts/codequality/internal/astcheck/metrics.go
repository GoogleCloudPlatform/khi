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
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// FuncMetrics holds static AST structural measurements for a single Go function or method.
type FuncMetrics struct {
	// Key is the canonical function or method identifier.
	Key string
	// DisplayName is the human-readable function or method signature label.
	DisplayName string
	// StartLine is the 1-based starting line of the function declaration.
	StartLine int
	// EndLine is the 1-based ending line of the function declaration.
	EndLine int
	// CodeLines is the number of non-blank, non-comment lines in the function body.
	CodeLines int
	// Statements is the total AST statement count in the function body.
	Statements int
	// Cognitive is the cognitive complexity score of the function.
	Cognitive int
	// MaxNesting is the maximum control-flow nesting depth in the function.
	MaxNesting int
	// Params is the number of input parameters.
	Params int
	// Results is the number of return values.
	Results int
	// ControlFlagParams lists boolean parameters used directly in control-flow conditions.
	ControlFlagParams []string
	// Hotspots lists complexity hotspot locations inside the function.
	Hotspots []rule.Hotspot
	// IsTestEntry reports whether the function is a Go test entry point.
	IsTestEntry bool
}

// TRunMetrics holds cognitive complexity measurements for a single t.Run or b.Run closure body.
type TRunMetrics struct {
	// ParentFunc is the enclosing test function name.
	ParentFunc string
	// SubtestLabel is the extracted subtest name or expression label.
	SubtestLabel string
	// StartLine is the 1-based starting line of the subtest call.
	StartLine int
	// EndLine is the 1-based ending line of the subtest call.
	EndLine int
	// Cognitive is the cognitive complexity score of the subtest body.
	Cognitive int
	// Hotspots lists complexity hotspot locations inside the subtest body.
	Hotspots []rule.Hotspot
}

// FileMetrics holds parsed AST metrics, subtests, and CODEQUALITY_ALLOW directives for a Go file.
type FileMetrics struct {
	// File is the slash-separated path relative to the repository root.
	File string
	// PackageName is the declared Go package name.
	PackageName string
	// PkgLine is the 1-based line number of the package clause.
	PkgLine int
	// IsTest reports whether the file is a *_test.go file.
	IsTest bool
	// CodeLines is the non-blank, non-comment, non-import code line count.
	CodeLines int
	// Funcs holds metrics for all functions with bodies in the file.
	Funcs []FuncMetrics
	// TRuns holds metrics for all t.Run/b.Run subtests in the file.
	TRuns []TRunMetrics
	// Directives holds valid CODEQUALITY_ALLOW directives parsed from the file.
	Directives []rule.Directive
	// DirectiveErrors holds syntax or validation errors for directives in the file.
	DirectiveErrors []rule.DirectiveError
}

type fileAnalyzer struct {
	fset             *token.FileSet
	codeMask         []bool
	isTest           bool
	minParamsForFlag int
}

// OverlapsLines reports whether the inclusive interval [startLine, endLine] contains any line in changed.
func OverlapsLines(startLine int, endLine int, changed map[int]struct{}) bool {
	for ln := startLine; ln <= endLine; ln++ {
		if _, ok := changed[ln]; ok {
			return true
		}
	}
	return false
}

// AnalyzeFile parses src as Go source for relPath and extracts file, function, subtest, and directive metrics.
func AnalyzeFile(relPath string, src []byte, minParamsForFlag int) (*FileMetrics, error) {
	cleanPath := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, cleanPath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", cleanPath, err)
	}
	if isGeneratedAST(file) {
		return nil, nil
	}

	isTest := strings.HasSuffix(cleanPath, "_test.go")
	codeMask := rule.BuildCodeLineMask(fset, file, src)
	fileMask := excludeHeaderAndImports(fset, file, codeMask)
	directives, dirErrs := rule.ParseDirectives(cleanPath, fset, file, src)

	out := &FileMetrics{
		File:            cleanPath,
		PackageName:     file.Name.Name,
		PkgLine:         fset.Position(file.Name.Pos()).Line,
		IsTest:          isTest,
		CodeLines:       countTrueInRange(fileMask, 1, len(fileMask)-1),
		Directives:      directives,
		DirectiveErrors: dirErrs,
	}

	fa := &fileAnalyzer{
		fset:             fset,
		codeMask:         codeMask,
		isTest:           isTest,
		minParamsForFlag: minParamsForFlag,
	}
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		out.Funcs = append(out.Funcs, fa.analyzeFuncDecl(fd))
		if isTest {
			out.TRuns = append(out.TRuns, collectTRunMetrics(fset, fd)...)
		}
	}
	return out, nil
}

// FormatFuncIdentity returns the stable lookup key ("Type.Method" or "Func") and display name ("(*Type).Method").
func FormatFuncIdentity(fd *ast.FuncDecl) (string, string) {
	name := fd.Name.Name
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return name, name
	}
	recvType, isPtr := unwrapReceiverName(fd.Recv.List[0].Type)
	key := recvType + "." + name
	if isPtr {
		return key, fmt.Sprintf("(*%s).%s", recvType, name)
	}
	return key, key
}

func isGeneratedAST(file *ast.File) bool {
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "Code generated") && strings.Contains(c.Text, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}

func excludeHeaderAndImports(fset *token.FileSet, file *ast.File, codeMask []bool) []bool {
	out := make([]bool, len(codeMask))
	copy(out, codeMask)
	out[fset.Position(file.Name.Pos()).Line] = false
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		start := fset.Position(gd.Pos()).Line
		end := fset.Position(gd.End()).Line
		for ln := start; ln <= end; ln++ {
			out[ln] = false
		}
	}
	return out
}

func countTrueInRange(mask []bool, start int, end int) int {
	n := 0
	for ln := start; ln <= end; ln++ {
		if mask[ln] {
			n++
		}
	}
	return n
}

func (a *fileAnalyzer) analyzeFuncDecl(fd *ast.FuncDecl) FuncMetrics {
	startLine := a.fset.Position(fd.Pos()).Line
	endLine := a.fset.Position(fd.End()).Line
	comp := AnalyzeBlockComplexity(a.fset, fd.Body)
	params := countFields(fd.Type.Params)
	results := countFields(fd.Type.Results)
	key, display := FormatFuncIdentity(fd)

	var flagParams []string
	if !a.isTest && params >= a.minParamsForFlag {
		flagParams = detectControlBoolParams(fd)
	}

	return FuncMetrics{
		Key:               key,
		DisplayName:       display,
		StartLine:         startLine,
		EndLine:           endLine,
		CodeLines:         countTrueInRange(a.codeMask, startLine, endLine),
		Statements:        countStatements(fd.Body),
		Cognitive:         comp.Cognitive,
		MaxNesting:        comp.MaxNesting,
		Params:            params,
		Results:           results,
		ControlFlagParams: flagParams,
		Hotspots:          comp.Hotspots,
		IsTestEntry:       a.isTest && isGoTestEntryFunc(fd),
	}
}

func unwrapReceiverName(expr ast.Expr) (string, bool) {
	isPtr := false
	if star, ok := expr.(*ast.StarExpr); ok {
		isPtr = true
		expr = star.X
	}
	switch t := expr.(type) {
	case *ast.IndexExpr:
		base, _ := unwrapReceiverName(t.X)
		return base, isPtr
	case *ast.IndexListExpr:
		base, _ := unwrapReceiverName(t.X)
		return base, isPtr
	default:
		ident := expr.(*ast.Ident)
		return ident.Name, isPtr
	}
}

func countFields(fl *ast.FieldList) int {
	if fl == nil {
		return 0
	}
	total := 0
	for _, f := range fl.List {
		if len(f.Names) == 0 {
			total++
		} else {
			total += len(f.Names)
		}
	}
	return total
}

func countStatements(body *ast.BlockStmt) int {
	count := 0
	ast.Inspect(body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.BlockStmt, *ast.LabeledStmt:
		case ast.Stmt:
			count++
		}
		return true
	})
	return count
}

func isGoTestEntryFunc(fd *ast.FuncDecl) bool {
	if fd.Recv != nil {
		return false
	}
	name := fd.Name.Name
	for _, prefix := range []string{"Test", "Benchmark", "Fuzz", "Example"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func detectControlBoolParams(fd *ast.FuncDecl) []string {
	boolNames := collectBoolParamNames(fd.Type.Params)
	if len(boolNames) == 0 {
		return nil
	}
	usedInBranch := make(map[string]bool)
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		inspectBranchCondition(n, usedInBranch)
		return true
	})
	var flagged []string
	for _, name := range boolNames {
		if usedInBranch[name] {
			flagged = append(flagged, name)
		}
	}
	return flagged
}

func inspectBranchCondition(n ast.Node, used map[string]bool) {
	switch s := n.(type) {
	case *ast.IfStmt:
		markDirectBoolIdents(s.Cond, used)
	case *ast.SwitchStmt:
		if s.Tag != nil {
			markDirectBoolIdents(s.Tag, used)
			return
		}
		markCaseBoolIdents(s.Body, used)
	}
}

func collectBoolParamNames(params *ast.FieldList) []string {
	var names []string
	for _, field := range params.List {
		ident, ok := field.Type.(*ast.Ident)
		if !ok || ident.Name != "bool" {
			continue
		}
		for _, n := range field.Names {
			if n.Name != "_" {
				names = append(names, n.Name)
			}
		}
	}
	return names
}

func markCaseBoolIdents(body *ast.BlockStmt, out map[string]bool) {
	for _, stmt := range body.List {
		cc := stmt.(*ast.CaseClause)
		for _, expr := range cc.List {
			markDirectBoolIdents(expr, out)
		}
	}
}

func markDirectBoolIdents(cond ast.Expr, out map[string]bool) {
	ast.Inspect(cond, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr, *ast.SelectorExpr:
			return false
		case *ast.Ident:
			out[x.Name] = true
		}
		return true
	})
}

func collectTRunMetrics(fset *token.FileSet, fd *ast.FuncDecl) []TRunMetrics {
	var runs []TRunMetrics
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		label, body, matched := extractSubtestRunCall(call)
		if !matched {
			return true
		}
		comp := AnalyzeSubtestBlockComplexity(fset, body)
		runs = append(runs, TRunMetrics{
			ParentFunc:   fd.Name.Name,
			SubtestLabel: label,
			StartLine:    fset.Position(call.Pos()).Line,
			EndLine:      fset.Position(call.End()).Line,
			Cognitive:    comp.Cognitive,
			Hotspots:     comp.Hotspots,
		})
		return true
	})
	return runs
}

func extractSubtestRunCall(call *ast.CallExpr) (string, *ast.BlockStmt, bool) {
	if len(call.Args) != 2 {
		return "", nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Run" {
		return "", nil, false
	}
	fnLit, ok := call.Args[1].(*ast.FuncLit)
	if !ok || !hasTestingParam(fnLit.Type.Params) {
		return "", nil, false
	}
	return describeSubtestArg(call.Args[0]), fnLit.Body, true
}

func hasTestingParam(params *ast.FieldList) bool {
	if params == nil || len(params.List) != 1 {
		return false
	}
	star, ok := params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	return ok && pkgIdent.Name == "testing" && (sel.Sel.Name == "T" || sel.Sel.Name == "B")
}

func describeSubtestArg(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if unq, err := strconv.Unquote(e.Value); err == nil {
			return unq
		}
		return e.Value
	case *ast.SelectorExpr:
		if id, ok := e.X.(*ast.Ident); ok {
			return id.Name + "." + e.Sel.Name
		}
		return e.Sel.Name
	case *ast.Ident:
		return e.Name
	default:
		return "subtest"
	}
}
