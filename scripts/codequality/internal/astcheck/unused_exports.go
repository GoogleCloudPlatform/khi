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
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const (
	// RuleUnusedExportedDecl enforces that exported top-level declarations in pkg/ are referenced somewhere in the repository.
	RuleUnusedExportedDecl = "unused-exported-decl"
	// RuleUnregisteredImplTask enforces that task variables defined in *_impl packages are referenced in production code.
	RuleUnregisteredImplTask = "unregistered-impl-task"
)

var skippedRepoDirs = map[string]bool{
	".git":         true,
	".jj":          true,
	"node_modules": true,
	"vendor":       true,
	"testdata":     true,
}

type unusedExportDecl struct {
	pkgImportPath string
	pkgName       string
	relFile       string
	pos           token.Pos
	name          string
	kind          string
	exported      bool
	isImplTaskVar bool
}

type parsedPkg struct {
	importPath string
	relDir     string
	pkgName    string
	prodFiles  map[string]*ast.File
	testFiles  map[string]*ast.File
}

type symbolKey struct {
	pkgImportPath string
	name          string
}

type symbolUses struct {
	prod int
	test int
}

type unusedExportAnalyzer struct {
	repoRoot     string
	modulePrefix string
	fset         *token.FileSet
	pkgsByImport map[string]*parsedPkg
	declIdentPos map[token.Pos]bool
	targets      []unusedExportDecl
	uses         map[symbolKey]symbolUses
}

type fileUseScanner struct {
	pkg           *parsedPkg
	aliasToImport map[string]string
	dotImports    []string
	sameScope     bool
	declIdentPos  map[token.Pos]bool
	record        func(pkgImp string, name string)
}

// CheckUnusedExports checks for unreferenced exported top-level declarations in pkg/ and unregistered task variables in *_impl packages.
func CheckUnusedExports(repoRoot string, pkgsFlag string) ([]ConventionViolation, error) {
	patterns := parsePkgPatterns(pkgsFlag)
	if len(patterns) == 0 {
		return nil, nil
	}

	modulePath, err := gotestutil.ReadModulePath(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return nil, err
	}

	analyzer := &unusedExportAnalyzer{
		repoRoot:     repoRoot,
		modulePrefix: modulePath + "/",
		fset:         token.NewFileSet(),
		pkgsByImport: make(map[string]*parsedPkg),
		declIdentPos: make(map[token.Pos]bool),
		uses:         make(map[symbolKey]symbolUses),
	}
	if err := analyzer.parseAllPackages(); err != nil {
		return nil, err
	}

	analyzer.indexDeclPositions()
	analyzer.collectTargets(patterns)
	analyzer.scanAllUses()

	violations := analyzer.buildViolations()
	sortConventionViolations(violations)
	return violations, nil
}

func (a *unusedExportAnalyzer) parseAllPackages() error {
	for _, rootDir := range []string{"cmd", "pkg", "scripts"} {
		if err := a.walkRootDir(rootDir); err != nil {
			return err
		}
	}
	return nil
}

func (a *unusedExportAnalyzer) walkRootDir(rootDir string) error {
	absRoot := filepath.Join(a.repoRoot, rootDir)
	if _, err := os.Stat(absRoot); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(absRoot, a.visitRepoEntry)
}

func (a *unusedExportAnalyzer) visitRepoEntry(fullPath string, d fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if d.IsDir() {
		if skippedRepoDirs[d.Name()] {
			return filepath.SkipDir
		}
		return nil
	}
	if !strings.HasSuffix(d.Name(), ".go") {
		return nil
	}
	return a.parseGoFile(fullPath)
}

func (a *unusedExportAnalyzer) parseGoFile(fullPath string) error {
	relFile, err := filepath.Rel(a.repoRoot, fullPath)
	if err != nil {
		return err
	}
	slashFile := filepath.ToSlash(relFile)
	relDir := path.Dir(slashFile)
	importPath := a.modulePrefix + relDir

	src, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", slashFile, err)
	}
	af, err := parser.ParseFile(a.fset, slashFile, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", slashFile, err)
	}

	pd := a.ensurePackage(importPath, relDir, af.Name.Name)
	if strings.HasSuffix(slashFile, "_test.go") {
		pd.testFiles[slashFile] = af
		return nil
	}
	pd.prodFiles[slashFile] = af
	pd.pkgName = af.Name.Name
	return nil
}

func (a *unusedExportAnalyzer) ensurePackage(importPath string, relDir string, pkgName string) *parsedPkg {
	if pd, ok := a.pkgsByImport[importPath]; ok {
		return pd
	}
	pd := &parsedPkg{
		importPath: importPath,
		relDir:     relDir,
		pkgName:    pkgName,
		prodFiles:  make(map[string]*ast.File),
		testFiles:  make(map[string]*ast.File),
	}
	a.pkgsByImport[importPath] = pd
	return pd
}

func (a *unusedExportAnalyzer) indexDeclPositions() {
	for _, pd := range a.pkgsByImport {
		for _, af := range pd.prodFiles {
			for _, d := range af.Decls {
				a.recordDeclPositions(d)
			}
		}
	}
}

func (a *unusedExportAnalyzer) recordDeclPositions(d ast.Decl) {
	switch decl := d.(type) {
	case *ast.FuncDecl:
		if decl.Recv == nil && decl.Name.Name != "_" {
			a.declIdentPos[decl.Name.Pos()] = true
		}
	case *ast.GenDecl:
		for _, spec := range decl.Specs {
			a.recordSpecPositions(spec)
		}
	}
}

func (a *unusedExportAnalyzer) recordSpecPositions(spec ast.Spec) {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		if s.Name.Name != "_" {
			a.declIdentPos[s.Name.Pos()] = true
		}
	case *ast.ValueSpec:
		for _, n := range s.Names {
			if n.Name != "_" {
				a.declIdentPos[n.Pos()] = true
			}
		}
	}
}

func (a *unusedExportAnalyzer) collectTargets(patterns []string) {
	for _, pd := range a.pkgsByImport {
		if !isTargetPkgDir(pd.relDir, patterns) {
			continue
		}
		for relFile, af := range pd.prodFiles {
			if strings.HasPrefix(path.Base(relFile), "zzz_") || IsGeneratedAST(af) {
				continue
			}
			a.collectFileTargets(pd, relFile, af)
		}
	}
}

func (a *unusedExportAnalyzer) collectFileTargets(pd *parsedPkg, relFile string, af *ast.File) {
	for _, d := range af.Decls {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if decl.Recv == nil && decl.Name.IsExported() {
				a.addExportedTarget(pd, relFile, decl.Name, "func")
			}
		case *ast.GenDecl:
			a.collectGenDeclTargets(pd, relFile, decl)
		}
	}
}

func (a *unusedExportAnalyzer) collectGenDeclTargets(pd *parsedPkg, relFile string, gd *ast.GenDecl) {
	for _, spec := range gd.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if s.Name.IsExported() {
				a.addExportedTarget(pd, relFile, s.Name, "type")
			}
		case *ast.ValueSpec:
			a.collectValueSpecTargets(pd, relFile, gd.Tok, s)
		}
	}
}

func (a *unusedExportAnalyzer) addExportedTarget(pd *parsedPkg, relFile string, ident *ast.Ident, kind string) {
	a.targets = append(a.targets, unusedExportDecl{
		pkgImportPath: pd.importPath,
		pkgName:       pd.pkgName,
		relFile:       relFile,
		pos:           ident.Pos(),
		name:          ident.Name,
		kind:          kind,
		exported:      true,
	})
}

func (a *unusedExportAnalyzer) collectValueSpecTargets(pd *parsedPkg, relFile string, tok token.Token, vs *ast.ValueSpec) {
	kind := "var"
	if tok == token.CONST {
		kind = "const"
	}
	isImplTask := strings.HasSuffix(pd.pkgName, "_impl") && tok == token.VAR && isTaskDefinitionValueSpec(vs)
	for _, n := range vs.Names {
		if n.Name == "_" || (!n.IsExported() && !isImplTask) {
			continue
		}
		a.targets = append(a.targets, unusedExportDecl{
			pkgImportPath: pd.importPath,
			pkgName:       pd.pkgName,
			relFile:       relFile,
			pos:           n.Pos(),
			name:          n.Name,
			kind:          kind,
			exported:      n.IsExported(),
			isImplTaskVar: isImplTask,
		})
	}
}

func (a *unusedExportAnalyzer) scanAllUses() {
	for _, pd := range a.pkgsByImport {
		for _, af := range pd.prodFiles {
			a.newFileScanner(pd, af, a.recordProdUse).scan(af)
		}
		for _, af := range pd.testFiles {
			a.newFileScanner(pd, af, a.recordTestUse).scan(af)
		}
	}
}

func (a *unusedExportAnalyzer) recordProdUse(pkgImp string, name string) {
	key := symbolKey{pkgImportPath: pkgImp, name: name}
	u := a.uses[key]
	u.prod++
	a.uses[key] = u
}

func (a *unusedExportAnalyzer) recordTestUse(pkgImp string, name string) {
	key := symbolKey{pkgImportPath: pkgImp, name: name}
	u := a.uses[key]
	u.test++
	a.uses[key] = u
}

func (a *unusedExportAnalyzer) newFileScanner(pd *parsedPkg, af *ast.File, record func(string, string)) *fileUseScanner {
	aliasToImport, dotImports := a.resolveFileImports(af)
	return &fileUseScanner{
		pkg:           pd,
		aliasToImport: aliasToImport,
		dotImports:    dotImports,
		sameScope:     af.Name.Name == pd.pkgName,
		declIdentPos:  a.declIdentPos,
		record:        record,
	}
}

func (a *unusedExportAnalyzer) resolveFileImports(af *ast.File) (map[string]string, []string) {
	aliasToImport := make(map[string]string)
	var dotImports []string
	for _, impSpec := range af.Imports {
		impPath := strings.Trim(impSpec.Path.Value, `"`)
		if !strings.HasPrefix(impPath, a.modulePrefix) {
			continue
		}
		if impSpec.Name != nil {
			switch impSpec.Name.Name {
			case "_":
				continue
			case ".":
				dotImports = append(dotImports, impPath)
			default:
				aliasToImport[impSpec.Name.Name] = impPath
			}
			continue
		}
		aliasToImport[a.defaultImportAlias(impPath)] = impPath
	}
	return aliasToImport, dotImports
}

func (a *unusedExportAnalyzer) defaultImportAlias(impPath string) string {
	if targetPkg, ok := a.pkgsByImport[impPath]; ok && targetPkg.pkgName != "" {
		return targetPkg.pkgName
	}
	return path.Base(impPath)
}

func (a *unusedExportAnalyzer) buildViolations() []ConventionViolation {
	var violations []ConventionViolation
	for _, d := range a.targets {
		if v, ok := a.violationForDecl(d); ok {
			violations = append(violations, v)
		}
	}
	return violations
}

func (a *unusedExportAnalyzer) violationForDecl(d unusedExportDecl) (ConventionViolation, bool) {
	u := a.uses[symbolKey{pkgImportPath: d.pkgImportPath, name: d.name}]
	p := a.fset.Position(d.pos)
	if d.isImplTaskVar && u.prod == 0 {
		return ConventionViolation{
			File:    d.relFile,
			Line:    p.Line,
			Column:  p.Column,
			Rule:    RuleUnregisteredImplTask,
			Message: fmt.Sprintf("task variable %s in %s is never referenced in production code (missing registration?)", d.name, d.pkgName),
		}, true
	}
	if d.exported && u.prod == 0 && u.test == 0 {
		return ConventionViolation{
			File:    d.relFile,
			Line:    p.Line,
			Column:  p.Column,
			Rule:    RuleUnusedExportedDecl,
			Message: fmt.Sprintf("exported %s %s has no references in the repository", d.kind, d.name),
		}, true
	}
	return ConventionViolation{}, false
}

func (s *fileUseScanner) scan(af *ast.File) {
	for _, decl := range af.Decls {
		ast.Inspect(decl, s.visitNode)
	}
}

func (s *fileUseScanner) visitNode(n ast.Node) bool {
	if n == nil {
		return false
	}
	switch node := n.(type) {
	case *ast.SelectorExpr:
		s.visitSelector(node)
		return false
	case *ast.Field:
		if node.Type != nil {
			ast.Inspect(node.Type, s.visitNode)
		}
		return false
	case *ast.Ident:
		s.visitIdent(node)
		return false
	}
	return true
}

func (s *fileUseScanner) visitSelector(sel *ast.SelectorExpr) {
	if xIdent, ok := sel.X.(*ast.Ident); ok {
		if targetImp, found := s.aliasToImport[xIdent.Name]; found {
			s.record(targetImp, sel.Sel.Name)
			return
		}
	}
	ast.Inspect(sel.X, s.visitNode)
}

func (s *fileUseScanner) visitIdent(ident *ast.Ident) {
	if s.declIdentPos[ident.Pos()] || ident.Name == "_" {
		return
	}
	if s.sameScope {
		s.record(s.pkg.importPath, ident.Name)
	}
	if ident.IsExported() {
		for _, dotImp := range s.dotImports {
			s.record(dotImp, ident.Name)
		}
	}
}

func isTargetPkgDir(relDir string, patterns []string) bool {
	if !strings.HasPrefix(relDir, "pkg/") {
		return false
	}
	if relDir == "pkg/generated" || strings.HasPrefix(relDir, "pkg/generated/") {
		return false
	}
	if relDir == "pkg/testutil" || strings.HasPrefix(relDir, "pkg/testutil/") {
		return false
	}
	return matchesAnyPkgPattern(relDir, patterns)
}

func isTaskDefinitionValueSpec(vs *ast.ValueSpec) bool {
	for _, val := range vs.Values {
		call, ok := val.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel := extractCallSelector(call.Fun)
		if sel == nil {
			continue
		}
		if isTaskConstructorName(sel.Sel.Name) {
			return true
		}
	}
	return false
}

func isTaskConstructorName(name string) bool {
	if name == "Define" || strings.HasPrefix(name, "Define") {
		return true
	}
	return strings.HasPrefix(name, "New") && strings.Contains(name, "Task") &&
		!strings.Contains(name, "ID") && !strings.Contains(name, "Reference")
}

func extractCallSelector(fun ast.Expr) *ast.SelectorExpr {
	switch e := fun.(type) {
	case *ast.SelectorExpr:
		return e
	case *ast.IndexExpr:
		sel, _ := e.X.(*ast.SelectorExpr)
		return sel
	case *ast.IndexListExpr:
		sel, _ := e.X.(*ast.SelectorExpr)
		return sel
	}
	return nil
}
