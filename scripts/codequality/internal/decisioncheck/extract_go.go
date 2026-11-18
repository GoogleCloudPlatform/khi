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
	"go/parser"
	"go/token"
	"go/types"
	"path"
	"strings"
)

const maxStructFieldsInSummary = 16

type parsedGoFile struct {
	relPath string
	astFile *ast.File
}

type fileSetParser func(fset *token.FileSet, files []BundledFile) ([]parsedGoFile, error)

type fileItemCollector func(fset *token.FileSet, targetID string, parsedFile parsedGoFile, lineSet map[int]struct{}) []ExtractedItem

// ExtractGoSubstantiveUnits extracts core architectural Go units and a compact AST structural index from files.
func ExtractGoSubstantiveUnits(targetID string, files []BundledFile) ([]ExtractedItem, string, error) {
	fset := token.NewFileSet()
	parsed, err := parseProdGoFiles(fset, files)
	if err != nil {
		return nil, "", err
	}

	var tasks []ExtractedItem
	var fileUnits []ExtractedItem
	var summary strings.Builder

	for _, pf := range parsed {
		fmt.Fprintf(&summary, "- %s (package %s)\n", pf.relPath, pf.astFile.Name.Name)
		if path.Base(pf.relPath) == "module.go" {
			tasks = append(tasks, extractModuleTaskUnits(fset, targetID, pf)...)
		}
		if unit, ok := collectFileSubsystemUnit(fset, targetID, pf, &summary); ok {
			fileUnits = append(fileUnits, unit)
		}
	}

	if len(tasks) > 0 {
		return tasks, summary.String(), nil
	}
	return fileUnits, summary.String(), nil
}

// ExtractChangedGoDecls extracts top-level production Go declarations overlapping changedLines.
func ExtractChangedGoDecls(targetID string, files []BundledFile, changedLines map[string]map[int]struct{}) ([]ExtractedItem, error) {
	return extractGoFileItems(targetID, files, changedLines, parseProdGoFiles, collectTouchedFileDecls)
}

// ExtractChangedGoFuncs extracts production Go functions and methods overlapping changedLines.
func ExtractChangedGoFuncs(targetID string, files []BundledFile, changedLines map[string]map[int]struct{}) ([]ExtractedItem, error) {
	return extractGoFileItems(targetID, files, changedLines, parseProdGoFiles, collectTouchedFuncs)
}

func extractGoFileItems(targetID string, files []BundledFile, changedLines map[string]map[int]struct{}, parse fileSetParser, collect fileItemCollector) ([]ExtractedItem, error) {
	fset := token.NewFileSet()
	parsedFiles, err := parse(fset, files)
	if err != nil {
		return nil, err
	}
	return runCollectorsOnFiles(fset, targetID, parsedFiles, changedLines, collect), nil
}

func runCollectorsOnFiles(fset *token.FileSet, targetID string, parsedFiles []parsedGoFile, changedLines map[string]map[int]struct{}, collect fileItemCollector) []ExtractedItem {
	var items []ExtractedItem
	for _, parsedFile := range parsedFiles {
		var lineSet map[int]struct{}
		if changedLines != nil {
			lineSet = changedLines[parsedFile.relPath]
			if len(lineSet) == 0 {
				continue
			}
		}
		items = append(items, collect(fset, targetID, parsedFile, lineSet)...)
	}
	return items
}

func parseProdGoFiles(fset *token.FileSet, files []BundledFile) ([]parsedGoFile, error) {
	return parseGoFilesByFilter(fset, files, func(relPath string) bool {
		return strings.HasSuffix(relPath, ".go") && !strings.HasSuffix(relPath, "_test.go")
	})
}

func parseTestGoFiles(fset *token.FileSet, files []BundledFile) ([]parsedGoFile, error) {
	return parseGoFilesByFilter(fset, files, func(relPath string) bool {
		return strings.HasSuffix(relPath, "_test.go")
	})
}

func parseGoFilesByFilter(fset *token.FileSet, files []BundledFile, include func(string) bool) ([]parsedGoFile, error) {
	var parsedFiles []parsedGoFile
	for _, bf := range files {
		if !include(bf.RelPath) {
			continue
		}
		astFile, err := parser.ParseFile(fset, bf.RelPath, bf.Content, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("failed parsing Go AST for %s: %w", bf.RelPath, err)
		}
		parsedFiles = append(parsedFiles, parsedGoFile{relPath: bf.RelPath, astFile: astFile})
	}
	return parsedFiles, nil
}

func extractModuleTaskUnits(fset *token.FileSet, targetID string, pf parsedGoFile) []ExtractedItem {
	var out []ExtractedItem
	ast.Inspect(pf.astFile, func(node ast.Node) bool {
		if kvExpr, ok := node.(*ast.KeyValueExpr); ok {
			keyIdent, isIdent := kvExpr.Key.(*ast.Ident)
			compLit, isComp := kvExpr.Value.(*ast.CompositeLit)
			if isIdent && keyIdent.Name == "Tasks" && isComp {
				out = appendCompositeTaskUnits(fset, targetID, pf.relPath, compLit, out)
			}
		}
		return true
	})
	return out
}

func appendCompositeTaskUnits(fset *token.FileSet, targetID string, relPath string, compLit *ast.CompositeLit, dst []ExtractedItem) []ExtractedItem {
	for _, elem := range compLit.Elts {
		if taskIdent, ok := elem.(*ast.Ident); ok {
			pos := fset.Position(taskIdent.Pos())
			dst = append(dst, ExtractedItem{
				Target:    targetID,
				File:      relPath,
				Line:      pos.Line,
				StartLine: pos.Line,
				EndLine:   pos.Line,
				Name:      taskIdent.Name,
				Kind:      "registered DAG task",
			})
		}
	}
	return dst
}

func collectFileSubsystemUnit(fset *token.FileSet, targetID string, pf parsedGoFile, summary *strings.Builder) (ExtractedItem, bool) {
	var exportedNames []string
	for _, decl := range pf.astFile.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			exportedNames = append(exportedNames, summarizeTypeSpecs(fset, pf.relPath, d, summary)...)
		case *ast.FuncDecl:
			if name, ok := summarizeFuncDecl(fset, pf.relPath, d, summary); ok {
				exportedNames = append(exportedNames, name)
			}
		}
	}
	if len(exportedNames) == 0 {
		return ExtractedItem{}, false
	}
	endLine := max(fset.Position(pf.astFile.End()).Line, 1)
	label := fmt.Sprintf("%s (%s)", path.Base(pf.relPath), strings.Join(capNames(exportedNames, 6), ", "))
	return ExtractedItem{
		Target:    targetID,
		File:      pf.relPath,
		Line:      1,
		StartLine: 1,
		EndLine:   endLine,
		Name:      label,
		Kind:      "Go source file",
	}, true
}

func summarizeTypeSpecs(fset *token.FileSet, relPath string, gd *ast.GenDecl, summary *strings.Builder) []string {
	if gd.Tok != token.TYPE {
		return nil
	}
	var names []string
	for _, spec := range gd.Specs {
		ts := spec.(*ast.TypeSpec)
		start := fset.Position(ts.Pos()).Line
		fmt.Fprintf(summary, "  type %s (%s:%d)\n", ts.Name.Name, relPath, start)
		appendStructFieldsSummary(summary, ts.Type)
		if ts.Name.IsExported() {
			names = append(names, ts.Name.Name)
		}
	}
	return names
}

func appendStructFieldsSummary(summary *strings.Builder, expr ast.Expr) {
	st, ok := expr.(*ast.StructType)
	if !ok || st.Fields == nil {
		return
	}
	written := 0
	for _, field := range st.Fields.List {
		if written >= maxStructFieldsInSummary {
			fmt.Fprintln(summary, "    ...")
			return
		}
		written++
		typeStr := types.ExprString(field.Type)
		if len(field.Names) == 0 {
			fmt.Fprintf(summary, "    embedded %s\n", typeStr)
			continue
		}
		names := make([]string, len(field.Names))
		for i, id := range field.Names {
			names[i] = id.Name
		}
		fmt.Fprintf(summary, "    field %s %s\n", strings.Join(names, ", "), typeStr)
	}
}

func summarizeFuncDecl(fset *token.FileSet, relPath string, fd *ast.FuncDecl, summary *strings.Builder) (string, bool) {
	if !fd.Name.IsExported() {
		return "", false
	}
	start := fset.Position(fd.Pos()).Line
	formatted := formatFuncDeclName(fd)
	fmt.Fprintf(summary, "  func %s (%s:%d)\n", formatted, relPath, start)
	if fd.Recv != nil {
		return "", false
	}
	return fd.Name.Name, true
}

func capNames(names []string, limit int) []string {
	if len(names) <= limit {
		return names
	}
	clipped := make([]string, limit+1)
	copy(clipped, names[:limit])
	clipped[limit] = "..."
	return clipped
}

func collectTouchedFileDecls(fset *token.FileSet, targetID string, parsedFile parsedGoFile, lineSet map[int]struct{}) []ExtractedItem {
	var touchedItems []ExtractedItem
	for _, decl := range parsedFile.astFile.Decls {
		switch typedDecl := decl.(type) {
		case *ast.FuncDecl:
			if item, ok := buildTouchedFuncItem(fset, targetID, parsedFile.relPath, typedDecl, lineSet); ok {
				touchedItems = append(touchedItems, item)
			}
		case *ast.GenDecl:
			touchedItems = append(touchedItems, collectTouchedGenSpecs(fset, targetID, parsedFile.relPath, typedDecl, lineSet)...)
		}
	}
	return touchedItems
}

func collectTouchedFuncs(fset *token.FileSet, targetID string, pf parsedGoFile, lineSet map[int]struct{}) []ExtractedItem {
	var out []ExtractedItem
	for _, decl := range pf.astFile.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if item, matched := buildTouchedFuncItem(fset, targetID, pf.relPath, fd, lineSet); matched {
			out = append(out, item)
		}
	}
	return out
}

func buildTouchedFuncItem(fset *token.FileSet, targetID string, relPath string, fd *ast.FuncDecl, lineSet map[int]struct{}) (ExtractedItem, bool) {
	declStart := fset.Position(fd.Pos()).Line
	spanStart := commentAwareStart(fset, fd.Doc, declStart)
	end := fset.Position(fd.End()).Line
	if !spanOverlaps(spanStart, end, lineSet) {
		return ExtractedItem{}, false
	}
	kind := "func"
	if fd.Recv != nil {
		kind = "method"
	}
	return ExtractedItem{
		Target:    targetID,
		File:      relPath,
		Line:      declStart,
		StartLine: spanStart,
		EndLine:   end,
		Name:      formatFuncDeclName(fd),
		Kind:      kind,
		Doc:       cleanDocGroup(fd.Doc),
	}, true
}

func collectTouchedGenSpecs(fset *token.FileSet, targetID string, relPath string, gd *ast.GenDecl, lineSet map[int]struct{}) []ExtractedItem {
	if gd.Tok == token.IMPORT {
		return nil
	}
	var out []ExtractedItem
	for _, spec := range gd.Specs {
		declStart := fset.Position(spec.Pos()).Line
		doc := specDocGroup(gd, spec)
		spanStart := commentAwareStart(fset, doc, declStart)
		end := fset.Position(spec.End()).Line
		if !spanOverlaps(spanStart, end, lineSet) {
			continue
		}
		for _, name := range specDeclaredNames(spec) {
			if name == "_" {
				continue
			}
			out = append(out, ExtractedItem{
				Target:    targetID,
				File:      relPath,
				Line:      declStart,
				StartLine: spanStart,
				EndLine:   end,
				Name:      name,
				Kind:      strings.ToLower(gd.Tok.String()),
				Doc:       cleanDocGroup(doc),
			})
		}
	}
	return out
}

func specDocGroup(gd *ast.GenDecl, spec ast.Spec) *ast.CommentGroup {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		if s.Doc != nil {
			return s.Doc
		}
	case *ast.ValueSpec:
		if s.Doc != nil {
			return s.Doc
		}
	}
	if len(gd.Specs) == 1 {
		return gd.Doc
	}
	return nil
}

func specDeclaredNames(spec ast.Spec) []string {
	if ts, ok := spec.(*ast.TypeSpec); ok {
		return []string{ts.Name.Name}
	}
	vs := spec.(*ast.ValueSpec)
	names := make([]string, 0, len(vs.Names))
	for _, id := range vs.Names {
		names = append(names, id.Name)
	}
	return names
}

func formatFuncDeclName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	recv := fd.Recv.List[0].Type
	if star, ok := recv.(*ast.StarExpr); ok {
		recv = star.X
	}
	if id, ok := recv.(*ast.Ident); ok {
		return id.Name + "." + fd.Name.Name
	}
	return fd.Name.Name
}

func commentAwareStart(fset *token.FileSet, doc *ast.CommentGroup, declStart int) int {
	if doc == nil {
		return declStart
	}
	return min(fset.Position(doc.Pos()).Line, declStart)
}

func cleanDocGroup(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	var kept []*ast.Comment
	for _, comment := range cg.List {
		trimmed := strings.TrimSpace(comment.Text)
		if isDirectiveComment(trimmed) {
			continue
		}
		kept = append(kept, comment)
	}
	return strings.TrimSpace((&ast.CommentGroup{List: kept}).Text())
}

func isDirectiveComment(trimmed string) bool {
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "//nolint") ||
		strings.HasPrefix(lower, "//go:") ||
		strings.Contains(trimmed, "CODEQUALITY_ALLOW")
}

func spanOverlaps(start int, end int, lineSet map[int]struct{}) bool {
	if lineSet == nil {
		return true
	}
	for ln := start; ln <= end; ln++ {
		if _, ok := lineSet[ln]; ok {
			return true
		}
	}
	return false
}
