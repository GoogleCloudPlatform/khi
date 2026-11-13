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
	"cmp"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const (
	// RuleDeclVisibilityOrder enforces exported-before-unexported declaration ordering within each kind.
	RuleDeclVisibilityOrder = "decl-visibility-order"
	// RuleExportedMemberDoc enforces GoDoc comments ending with a period on exported struct fields and interface methods.
	RuleExportedMemberDoc = "exported-member-doc"
)

// ConventionViolation describes a single Go declaration ordering or member GoDoc violation.
type ConventionViolation struct {
	// File is the slash-separated path relative to the repository root.
	File string
	// Line is the 1-based source line number of the violation.
	Line int
	// Column is the 1-based source column number of the violation.
	Column int
	// Rule is the violated convention rule identifier.
	Rule string
	// Message describes the convention violation.
	Message string
}

type parsedConvFile struct {
	relPath string
	astFile *ast.File
}

type convChecker struct {
	fset           *token.FileSet
	relPath        string
	exportedTypes  map[string]bool
	seenUnexpConst bool
	seenUnexpVar   bool
	seenUnexpType  bool
	maxFuncTier    int
	violations     []ConventionViolation
}

// CheckConventions checks Go declaration visibility ordering and exported member GoDoc conventions under repoRoot.
func CheckConventions(repoRoot string, pkgsFlag string) ([]ConventionViolation, error) {
	patterns := parsePkgPatterns(pkgsFlag)
	if len(patterns) == 0 {
		return nil, nil
	}
	filesByDir, err := discoverConventionFiles(repoRoot, patterns)
	if err != nil {
		return nil, err
	}
	var violations []ConventionViolation
	dirs := sortedDirKeys(filesByDir)
	for _, dir := range dirs {
		dirViolations, err := checkDirConventions(repoRoot, filesByDir[dir])
		if err != nil {
			return nil, err
		}
		violations = append(violations, dirViolations...)
	}
	sortConventionViolations(violations)
	return violations, nil
}

func parsePkgPatterns(pkgsFlag string) []string {
	var patterns []string
	for part := range strings.SplitSeq(pkgsFlag, ",") {
		raw := strings.TrimSpace(part)
		if raw == "" {
			continue
		}
		if raw == "." || raw == "./" {
			patterns = append(patterns, ".")
			continue
		}
		norm := strings.TrimPrefix(raw, "./")
		norm = strings.TrimSuffix(norm, "/")
		patterns = append(patterns, norm)
	}
	return patterns
}

func matchPkgPattern(relDir string, pat string) bool {
	if pat == "..." {
		return true
	}
	if strings.HasSuffix(pat, "/...") {
		prefix := strings.TrimSuffix(pat, "/...")
		return relDir == prefix || strings.HasPrefix(relDir, prefix+"/")
	}
	return relDir == pat
}

func discoverConventionFiles(repoRoot string, patterns []string) (map[string][]string, error) {
	filesByDir := make(map[string][]string)
	err := filepath.WalkDir(repoRoot, func(fullPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(repoRoot, fullPath)
		if err != nil {
			return err
		}
		slashRel := filepath.ToSlash(rel)
		if d.IsDir() {
			if shouldSkipConventionDir(slashRel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isConventionTargetGoFile(slashRel) {
			return nil
		}
		relDir := path.Dir(slashRel)
		if matchesAnyPkgPattern(relDir, patterns) {
			filesByDir[relDir] = append(filesByDir[relDir], slashRel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filesByDir, nil
}

func shouldSkipConventionDir(slashRel string) bool {
	if slashRel == "." {
		return false
	}
	base := path.Base(slashRel)
	if base == ".git" || base == ".jj" || base == "node_modules" || base == "vendor" {
		return true
	}
	return slashRel == "web" || slashRel == "pkg/generated" || strings.HasPrefix(slashRel, "pkg/generated/")
}

func isConventionTargetGoFile(slashRel string) bool {
	if !strings.HasSuffix(slashRel, ".go") || strings.HasSuffix(slashRel, "_test.go") {
		return false
	}
	return !strings.HasPrefix(path.Base(slashRel), "zzz_")
}

func matchesAnyPkgPattern(relDir string, patterns []string) bool {
	for _, pat := range patterns {
		if matchPkgPattern(relDir, pat) {
			return true
		}
	}
	return false
}

func sortedDirKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func checkDirConventions(repoRoot string, relPaths []string) ([]ConventionViolation, error) {
	slices.Sort(relPaths)
	fset := token.NewFileSet()
	byPkg := make(map[string][]parsedConvFile)
	var pkgOrder []string

	for _, relPath := range relPaths {
		src, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", relPath, err)
		}
		file, err := parser.ParseFile(fset, relPath, src, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", relPath, err)
		}
		if IsGeneratedAST(file) {
			continue
		}
		pkgName := file.Name.Name
		if _, exists := byPkg[pkgName]; !exists {
			pkgOrder = append(pkgOrder, pkgName)
		}
		byPkg[pkgName] = append(byPkg[pkgName], parsedConvFile{relPath: relPath, astFile: file})
	}

	var violations []ConventionViolation
	for _, pkgName := range pkgOrder {
		violations = append(violations, checkPackageFiles(fset, byPkg[pkgName])...)
	}
	return violations, nil
}

func checkPackageFiles(fset *token.FileSet, files []parsedConvFile) []ConventionViolation {
	exportedTypes := collectExportedPackageTypes(files)
	var violations []ConventionViolation
	for _, pf := range files {
		chk := &convChecker{
			fset:          fset,
			relPath:       pf.relPath,
			exportedTypes: exportedTypes,
		}
		chk.checkFile(pf.astFile)
		violations = append(violations, chk.violations...)
	}
	return violations
}

func collectExportedPackageTypes(files []parsedConvFile) map[string]bool {
	exported := make(map[string]bool)
	for _, pf := range files {
		for _, decl := range pf.astFile.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				if ast.IsExported(ts.Name.Name) {
					exported[ts.Name.Name] = true
				}
			}
		}
	}
	return exported
}

func (c *convChecker) checkFile(file *ast.File) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			c.checkGenDecl(d)
		case *ast.FuncDecl:
			c.checkFuncOrder(d)
		}
	}
}

func (c *convChecker) checkGenDecl(gd *ast.GenDecl) {
	switch gd.Tok {
	case token.CONST:
		c.checkConstOrder(gd)
	case token.VAR:
		c.checkVarOrder(gd)
	case token.TYPE:
		c.checkTypeOrder(gd)
		c.checkExportedMemberDoc(gd)
	}
}

func (c *convChecker) checkConstOrder(gd *ast.GenDecl) {
	hasExported, hasUnexported := classifyConstBlock(gd)
	if !hasExported && !hasUnexported {
		return
	}
	if hasExported {
		if c.seenUnexpConst {
			c.addViolation(gd.Pos(), RuleDeclVisibilityOrder, "exported const block must appear before unexported const blocks")
		}
		return
	}
	c.seenUnexpConst = true
}

func classifyConstBlock(gd *ast.GenDecl) (bool, bool) {
	hasExported := false
	hasUnexported := false
	for _, spec := range gd.Specs {
		vs := spec.(*ast.ValueSpec)
		for _, ident := range vs.Names {
			if ident.Name == "_" {
				continue
			}
			if ast.IsExported(ident.Name) {
				hasExported = true
			} else {
				hasUnexported = true
			}
		}
	}
	return hasExported, hasUnexported
}

func (c *convChecker) checkVarOrder(gd *ast.GenDecl) {
	for _, spec := range gd.Specs {
		vs := spec.(*ast.ValueSpec)
		hasNamed, hasExported := classifyValueSpec(vs)
		if !hasNamed {
			continue
		}
		if hasExported {
			if c.seenUnexpVar {
				c.addViolation(vs.Pos(), RuleDeclVisibilityOrder, "exported var declaration must appear before unexported var declarations")
			}
			continue
		}
		c.seenUnexpVar = true
	}
}

func classifyValueSpec(vs *ast.ValueSpec) (bool, bool) {
	hasNamed := false
	hasExported := false
	for _, ident := range vs.Names {
		if ident.Name == "_" {
			continue
		}
		hasNamed = true
		if ast.IsExported(ident.Name) {
			hasExported = true
		}
	}
	return hasNamed, hasExported
}

func (c *convChecker) checkTypeOrder(gd *ast.GenDecl) {
	for _, spec := range gd.Specs {
		ts := spec.(*ast.TypeSpec)
		if ts.Name.Name == "_" {
			continue
		}
		if ast.IsExported(ts.Name.Name) {
			if c.seenUnexpType {
				c.addViolation(ts.Pos(), RuleDeclVisibilityOrder, fmt.Sprintf("exported type %s must appear before unexported types", ts.Name.Name))
			}
			continue
		}
		c.seenUnexpType = true
	}
}

func (c *convChecker) checkFuncOrder(fd *ast.FuncDecl) {
	tier := classifyFuncTier(fd, c.exportedTypes)
	if tier < c.maxFuncTier {
		_, display := FormatFuncIdentity(fd)
		c.addViolation(fd.Pos(), RuleDeclVisibilityOrder, fmt.Sprintf("%s must appear before %s declarations", display, tierLabel(c.maxFuncTier)))
		return
	}
	c.maxFuncTier = tier
}

func classifyFuncTier(fd *ast.FuncDecl, exportedTypes map[string]bool) int {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		name := fd.Name.Name
		if name == "init" {
			return 0
		}
		if ast.IsExported(name) || isExportedTypeConstructor(fd, exportedTypes) {
			return 1
		}
		return 2
	}
	recvType, _ := unwrapReceiverName(fd.Recv.List[0].Type)
	if ast.IsExported(recvType) {
		return 1
	}
	return 2
}

func isExportedTypeConstructor(fd *ast.FuncDecl, exportedTypes map[string]bool) bool {
	name := fd.Name.Name
	if !strings.HasPrefix(name, "new") && !strings.HasPrefix(name, "must") {
		return false
	}
	if fd.Type.Results == nil {
		return false
	}
	for _, field := range fd.Type.Results.List {
		if exprReferencesExportedType(field.Type, exportedTypes) {
			return true
		}
	}
	return false
}

func exprReferencesExportedType(expr ast.Expr, exportedTypes map[string]bool) bool {
	for {
		switch e := expr.(type) {
		case *ast.StarExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		case *ast.Ident:
			return exportedTypes[e.Name]
		default:
			return false
		}
	}
}

func tierLabel(tier int) string {
	if tier == 1 {
		return "exported API"
	}
	return "unexported"
}

func (c *convChecker) checkExportedMemberDoc(gd *ast.GenDecl) {
	for _, spec := range gd.Specs {
		ts := spec.(*ast.TypeSpec)
		if !ast.IsExported(ts.Name.Name) {
			continue
		}
		switch t := ts.Type.(type) {
		case *ast.StructType:
			c.checkFieldListDoc(ts.Name.Name, "struct field", t.Fields)
		case *ast.InterfaceType:
			c.checkFieldListDoc(ts.Name.Name, "interface method", t.Methods)
		}
	}
}

func (c *convChecker) checkFieldListDoc(typeName string, kindLabel string, fl *ast.FieldList) {
	for _, field := range fl.List {
		for _, name := range field.Names {
			if !ast.IsExported(name.Name) {
				continue
			}
			if msg := validateMemberDoc(field.Doc, typeName, name.Name, kindLabel); msg != "" {
				c.addViolation(name.Pos(), RuleExportedMemberDoc, msg)
			}
		}
	}
}

func validateMemberDoc(doc *ast.CommentGroup, typeName string, memberName string, kindLabel string) string {
	if doc == nil {
		return fmt.Sprintf("exported %s %s.%s must have a GoDoc comment", kindLabel, typeName, memberName)
	}
	text := strings.TrimSpace(doc.Text())
	if text == "" {
		return fmt.Sprintf("exported %s %s.%s must have a non-empty GoDoc comment", kindLabel, typeName, memberName)
	}
	if !strings.HasSuffix(text, ".") {
		return fmt.Sprintf("GoDoc comment for exported %s %s.%s must end with a period", kindLabel, typeName, memberName)
	}
	return ""
}

func (c *convChecker) addViolation(pos token.Pos, ruleID string, msg string) {
	p := c.fset.Position(pos)
	c.violations = append(c.violations, ConventionViolation{
		File:    c.relPath,
		Line:    p.Line,
		Column:  p.Column,
		Rule:    ruleID,
		Message: msg,
	})
}

func sortConventionViolations(violations []ConventionViolation) {
	slices.SortFunc(violations, func(a, b ConventionViolation) int {
		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.Column, b.Column),
			cmp.Compare(a.Rule, b.Rule),
			cmp.Compare(a.Message, b.Message),
		)
	})
}
