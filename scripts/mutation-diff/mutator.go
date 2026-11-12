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

package main

import (
	"cmp"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
)

// MutantStatus represents the evaluation state of a single mutant candidate.
type MutantStatus string

const (
	// MutantPending indicates the mutant is awaiting test execution.
	MutantPending MutantStatus = "PENDING"
	// MutantKilled indicates at least one unit test failed when the mutant was active.
	MutantKilled MutantStatus = "KILLED"
	// MutantTimedOut indicates the unit test timed out when the mutant was active (counted as killed).
	MutantTimedOut MutantStatus = "TIMED_OUT"
	// MutantSurvived indicates all unit tests still passed despite the active mutation.
	MutantSurvived MutantStatus = "SURVIVED"
	// MutantUncovered indicates unit tests never executed the mutated expression or statement.
	MutantUncovered MutantStatus = "UNCOVERED"
	// MutantSuppressed indicates the mutant was suppressed by a valid // NOMUTANTS: <reason> comment.
	MutantSuppressed MutantStatus = "SUPPRESSED"
	// MutantSkippedBudget indicates evaluation was skipped due to --max-mutants or --timeout limits.
	MutantSkippedBudget MutantStatus = "SKIPPED_BUDGET"
	// MutantCompileSkipped indicates the mutation site was dropped by compile fallback.
	MutantCompileSkipped MutantStatus = "COMPILE_SKIPPED"
)

// MutantOperator identifies the category of AST mutation applied to a Go expression or statement.
type MutantOperator string

const (
	// OperatorROR represents Relational Operator Replacement (<, <=, >, >=, ==, !=).
	OperatorROR MutantOperator = "ROR"
	// OperatorLCR represents Logical Connector Replacement (&&, ||).
	OperatorLCR MutantOperator = "LCR"
	// OperatorBCR represents Branch Condition Replacement (forcing non-comparison if conditions to true/false).
	OperatorBCR MutantOperator = "BCR"
	// OperatorAOR represents Arithmetic and Concatenation Operator Replacement (+, -, *, /).
	OperatorAOR MutantOperator = "AOR"
	// OperatorUOI represents Unary Operator Removal (!x -> x).
	OperatorUOI MutantOperator = "UOI"
	// OperatorBLR represents Boolean Literal Replacement in return statements (true <-> false).
	OperatorBLR MutantOperator = "BLR"
	// OperatorSBR represents Statement and Side-Effect Removal (=, +=, -=, ++, --).
	OperatorSBR MutantOperator = "SBR"
)

// IsValid reports whether o is a recognized MutantOperator value.
func (o MutantOperator) IsValid() bool {
	switch o {
	case OperatorROR, OperatorLCR, OperatorBCR, OperatorAOR, OperatorUOI, OperatorBLR, OperatorSBR:
		return true
	default:
		return false
	}
}

// Mutant describes a single candidate logic change on a modified Go line.
type Mutant struct {
	// ID is the 1-based package-scoped identifier assigned to executable mutants (0 when suppressed/uncovered pre-compile).
	ID int
	// GlobalIndex is the 1-based display index across the entire run (e.g. [M1], [M2]).
	GlobalIndex int
	// File is the repository-relative path of the Go source file.
	File string
	// Line is the 1-based line number of the mutation site.
	Line int
	// Operator is the mutator category applied at this site.
	Operator MutantOperator
	// OriginalExpr is the original expression or statement summary.
	OriginalExpr string
	// MutatedExpr is the mutated expression or statement summary.
	MutatedExpr string
	// OriginalLine is the trimmed original source line for diff display.
	OriginalLine string
	// MutatedLine is the trimmed mutated source line for diff display.
	MutatedLine string
	// SuppressionCategory records the // NOMUTANTS(<category>) classification when Status == MutantSuppressed.
	SuppressionCategory SuppressionCategory
	// Status holds the evaluation outcome of this mutant.
	Status MutantStatus
}

type siteKind int

const (
	siteOrderedCmp siteKind = iota
	siteEqCmp
	siteLogical
	siteCond
	siteArith
	siteUnaryNot
	siteBoolLit
	siteStmtSkip
)

// MutationSite represents one AST span in a file that expands into Super-Mutant runtime code.
type MutationSite struct {
	StartOffset  int
	EndOffset    int
	Kind         siteKind
	Op           token.Token
	AddKeepRight bool
	BoolVal      bool
	LeftStart    int
	LeftEnd      int
	RightStart   int
	RightEnd     int
	Mutants      []*Mutant
}

// FilePlan holds parsed metadata, extracted mutants, and active Super-Mutant sites for a single Go file.
type FilePlan struct {
	File        string
	PackageName string
	Source      []byte
	Sites       []MutationSite
	Mutants     []*Mutant
	Warnings    []DirectiveWarning
}

// MutateFileOptions configures how candidate mutations are extracted from a Go source file.
type MutateFileOptions struct {
	ChangedLines map[int]struct{}
	// CoveredLines optionally restricts executable sites to lines covered in a pre-existing coverprofile.
	CoveredLines map[int]struct{}
	Rules        []ExclusionRule
	NextMutantID *int
}

// PlanFileMutations parses src and extracts all candidate mutants on ChangedLines.
func PlanFileMutations(filePath string, src []byte, opts *MutateFileOptions) (*FilePlan, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	if hasBuildIgnoreDirective(file) {
		return &FilePlan{
			File:        filePath,
			PackageName: file.Name.Name,
			Source:      src,
		}, nil
	}

	rules := opts.Rules
	if rules == nil {
		rules = DefaultExclusionRules()
	}
	nextID := opts.NextMutantID
	if nextID == nil {
		localID := 1
		nextID = &localID
	}

	suppressions := ParseSuppressions(filePath, fset, file, src)
	lineStarts := computeLineOffsets(src)

	plan := &FilePlan{
		File:        filePath,
		PackageName: file.Name.Name,
		Source:      src,
		Warnings:    suppressions.Warnings,
	}

	stmtListMembers := collectBlockListStmts(file)

	for _, decl := range file.Decls {
		fnDecl, ok := decl.(*ast.FuncDecl)
		if !ok || fnDecl.Body == nil {
			continue
		}
		ancestors := []ast.Node{file, fnDecl}
		walkFuncAST(fnDecl.Body, ancestors, fset, file, src, lineStarts, stmtListMembers, &suppressions, opts, rules, nextID, plan)
	}

	return plan, nil
}

func walkFuncAST(
	node ast.Node,
	ancestors []ast.Node,
	fset *token.FileSet,
	file *ast.File,
	src []byte,
	lineStarts []int,
	stmtListMembers map[ast.Stmt]struct{},
	suppressions *SuppressionIndex,
	opts *MutateFileOptions,
	rules []ExclusionRule,
	nextID *int,
	plan *FilePlan,
) {
	if node == nil {
		return
	}

	ctx := &ExclusionContext{
		FilePath:  plan.File,
		Fset:      fset,
		File:      file,
		Source:    src,
		Ancestors: ancestors,
	}
	scope := EvaluateExclusionRules(rules, ctx, node)
	if scope == ExclusionSubtree {
		return
	}

	if scope == ExclusionNone {
		extractNodeSite(node, ctx, lineStarts, stmtListMembers, suppressions, opts, nextID, plan)
	}

	ancestors = append(ancestors, node)
	ast.Inspect(node, func(child ast.Node) bool {
		if child == node {
			return true
		}
		if child != nil {
			walkFuncAST(child, ancestors, fset, file, src, lineStarts, stmtListMembers, suppressions, opts, rules, nextID, plan)
		}
		return false
	})
}

func extractNodeSite(
	node ast.Node,
	ctx *ExclusionContext,
	lineStarts []int,
	stmtListMembers map[ast.Stmt]struct{},
	suppressions *SuppressionIndex,
	opts *MutateFileOptions,
	nextID *int,
	plan *FilePlan,
) {
	fset := ctx.Fset
	src := ctx.Source

	registerSite := func(line int, op MutantOperator, kind siteKind, site MutationSite, specs [][2]string, customMutLines []string) {
		if _, changed := opts.ChangedLines[line]; !changed {
			return
		}
		var status MutantStatus
		supCat, isSuppressed := suppressions.LookupSuppression(line, op)
		switch {
		case isSuppressed:
			status = MutantSuppressed
		case opts.CoveredLines != nil && !hasLine(opts.CoveredLines, line):
			status = MutantUncovered
		default:
			status = MutantPending
		}

		siteMutants := make([]*Mutant, 0, len(specs))
		for i, sp := range specs {
			origLine, mutLine := buildDiffLines(src, lineStarts, line, site.StartOffset, site.EndOffset, sp[1])
			if i < len(customMutLines) {
				mutLine = customMutLines[i]
			}
			m := &Mutant{
				File:                plan.File,
				Line:                line,
				Operator:            op,
				OriginalExpr:        sp[0],
				MutatedExpr:         sp[1],
				OriginalLine:        origLine,
				MutatedLine:         mutLine,
				SuppressionCategory: supCat,
				Status:              status,
			}
			if status == MutantPending {
				m.ID = *nextID
				*nextID++
			}
			plan.Mutants = append(plan.Mutants, m)
			siteMutants = append(siteMutants, m)
		}

		if status == MutantPending {
			site.Kind = kind
			site.Mutants = siteMutants
			plan.Sites = append(plan.Sites, site)
		}
	}

	switch n := node.(type) {
	case *ast.BinaryExpr:
		opLine := fset.Position(n.OpPos).Line
		start := fset.Position(n.Pos()).Offset
		end := fset.Position(n.End()).Offset
		lStart := fset.Position(n.X.Pos()).Offset
		lEnd := fset.Position(n.X.End()).Offset
		rStart := fset.Position(n.Y.Pos()).Offset
		rEnd := fset.Position(n.Y.End()).Offset
		lhsText := compactWhitespace(string(src[lStart:lEnd]))
		rhsText := compactWhitespace(string(src[rStart:rEnd]))
		origText := compactWhitespace(string(src[start:end]))

		baseSite := MutationSite{
			StartOffset: start,
			EndOffset:   end,
			Op:          n.Op,
			LeftStart:   lStart,
			LeftEnd:     lEnd,
			RightStart:  rStart,
			RightEnd:    rEnd,
		}

		switch n.Op {
		case token.LSS:
			registerSite(opLine, OperatorROR, siteOrderedCmp, baseSite, [][2]string{
				{origText, lhsText + " <= " + rhsText},
				{origText, lhsText + " >= " + rhsText},
			}, nil)
		case token.LEQ:
			registerSite(opLine, OperatorROR, siteOrderedCmp, baseSite, [][2]string{
				{origText, lhsText + " < " + rhsText},
				{origText, lhsText + " > " + rhsText},
			}, nil)
		case token.GTR:
			registerSite(opLine, OperatorROR, siteOrderedCmp, baseSite, [][2]string{
				{origText, lhsText + " >= " + rhsText},
				{origText, lhsText + " <= " + rhsText},
			}, nil)
		case token.GEQ:
			registerSite(opLine, OperatorROR, siteOrderedCmp, baseSite, [][2]string{
				{origText, lhsText + " > " + rhsText},
				{origText, lhsText + " < " + rhsText},
			}, nil)
		case token.EQL:
			registerSite(opLine, OperatorROR, siteEqCmp, baseSite, [][2]string{
				{origText, lhsText + " != " + rhsText},
			}, nil)
		case token.NEQ:
			registerSite(opLine, OperatorROR, siteEqCmp, baseSite, [][2]string{
				{origText, lhsText + " == " + rhsText},
			}, nil)
		case token.LAND:
			registerSite(opLine, OperatorLCR, siteLogical, baseSite, [][2]string{
				{origText, lhsText + " || " + rhsText},
			}, nil)
		case token.LOR:
			registerSite(opLine, OperatorLCR, siteLogical, baseSite, [][2]string{
				{origText, lhsText + " && " + rhsText},
			}, nil)
		case token.ADD:
			keepRight := isZeroOrEmptyLiteral(n.X)
			baseSite.AddKeepRight = keepRight
			keptText := lhsText
			if keepRight {
				keptText = rhsText
			}
			registerSite(opLine, OperatorAOR, siteArith, baseSite, [][2]string{
				{origText, keptText},
			}, nil)
		case token.SUB:
			registerSite(opLine, OperatorAOR, siteArith, baseSite, [][2]string{
				{origText, lhsText + " + " + rhsText},
			}, nil)
		case token.MUL:
			registerSite(opLine, OperatorAOR, siteArith, baseSite, [][2]string{
				{origText, lhsText + " / " + rhsText},
			}, nil)
		case token.QUO:
			registerSite(opLine, OperatorAOR, siteArith, baseSite, [][2]string{
				{origText, lhsText + " * " + rhsText},
			}, nil)
		}

	case *ast.IfStmt:
		if shouldApplyBCR(n.Cond) {
			condLine := fset.Position(n.Cond.Pos()).Line
			cStart := fset.Position(n.Cond.Pos()).Offset
			cEnd := fset.Position(n.Cond.End()).Offset
			origCond := compactWhitespace(string(src[cStart:cEnd]))
			site := MutationSite{
				StartOffset: cStart,
				EndOffset:   cEnd,
			}
			registerSite(condLine, OperatorBCR, siteCond, site, [][2]string{
				{origCond, "true"},
				{origCond, "false"},
			}, nil)
		}

	case *ast.UnaryExpr:
		if n.Op == token.NOT {
			line := fset.Position(n.OpPos).Line
			start := fset.Position(n.Pos()).Offset
			end := fset.Position(n.End()).Offset
			xStart := fset.Position(n.X.Pos()).Offset
			xEnd := fset.Position(n.X.End()).Offset
			origText := compactWhitespace(string(src[start:end]))
			xText := compactWhitespace(string(src[xStart:xEnd]))
			site := MutationSite{
				StartOffset: start,
				EndOffset:   end,
				LeftStart:   xStart,
				LeftEnd:     xEnd,
			}
			registerSite(line, OperatorUOI, siteUnaryNot, site, [][2]string{
				{origText, xText},
			}, nil)
		}

	case *ast.Ident:
		if n.Name == "true" || n.Name == "false" {
			if retStmt, ok := ctx.Parent().(*ast.ReturnStmt); ok && isStandardBoolReturnSlot(ctx.Ancestors, retStmt, n) {
				line := fset.Position(n.Pos()).Line
				start := fset.Position(n.Pos()).Offset
				end := fset.Position(n.End()).Offset
				flipped := "false"
				if n.Name == "false" {
					flipped = "true"
				}
				site := MutationSite{
					StartOffset: start,
					EndOffset:   end,
					BoolVal:     n.Name == "true",
				}
				registerSite(line, OperatorBLR, siteBoolLit, site, [][2]string{
					{n.Name, flipped},
				}, nil)
			}
		}

	case *ast.AssignStmt:
		if _, inBlockList := stmtListMembers[n]; !inBlockList {
			return
		}
		if (n.Tok == token.ASSIGN || n.Tok == token.ADD_ASSIGN || n.Tok == token.SUB_ASSIGN) && !isAllBlankLHS(n.Lhs) {
			line := fset.Position(n.TokPos).Line
			start := fset.Position(n.Pos()).Offset
			end := fset.Position(n.End()).Offset
			origStmt := compactWhitespace(string(src[start:end]))
			site := MutationSite{
				StartOffset: start,
				EndOffset:   end,
			}
			registerSite(line, OperatorSBR, siteStmtSkip, site, [][2]string{
				{origStmt, "/* skipped assignment */"},
			}, []string{"/* skipped: " + firstLineTrimmed(src, lineStarts, line) + " */"})
		}

	case *ast.IncDecStmt:
		if _, inBlockList := stmtListMembers[n]; !inBlockList {
			return
		}
		line := fset.Position(n.TokPos).Line
		start := fset.Position(n.Pos()).Offset
		end := fset.Position(n.End()).Offset
		origStmt := compactWhitespace(string(src[start:end]))
		site := MutationSite{
			StartOffset: start,
			EndOffset:   end,
		}
		registerSite(line, OperatorSBR, siteStmtSkip, site, [][2]string{
			{origStmt, "/* skipped " + origStmt + " */"},
		}, []string{"/* skipped: " + firstLineTrimmed(src, lineStarts, line) + " */"})
	}
}

// RenderSuperMutantSource splices activeSites into src from innermost to outermost AST spans.
func RenderSuperMutantSource(src []byte, activeSites []MutationSite) []byte {
	if len(activeSites) == 0 {
		return src
	}

	sorted := make([]MutationSite, len(activeSites))
	copy(sorted, activeSites)
	slices.SortFunc(sorted, func(a, b MutationSite) int {
		return cmp.Or(
			cmp.Compare(a.StartOffset, b.StartOffset),
			cmp.Compare(b.EndOffset, a.EndOffset),
		)
	})

	rendered := renderSpan(src, sorted, 0, len(src), -1)
	return []byte(rendered)
}

func renderSpan(src []byte, sites []MutationSite, start int, end int, excludeIdx int) string {
	var sb strings.Builder
	cursor := start

	for i := range sites {
		if i == excludeIdx {
			continue
		}
		s := &sites[i]
		if s.StartOffset < cursor || s.EndOffset > end {
			continue
		}
		if excludeIdx >= 0 {
			parent := &sites[excludeIdx]
			if s.StartOffset == parent.StartOffset && s.EndOffset == parent.EndOffset {
				continue
			}
		}

		sb.Write(src[cursor:s.StartOffset])
		sb.WriteString(renderSingleSite(src, sites, i))
		cursor = s.EndOffset
	}

	sb.Write(src[cursor:end])
	return sb.String()
}

func renderSingleSite(src []byte, sites []MutationSite, idx int) string {
	s := &sites[idx]
	switch s.Kind {
	case siteOrderedCmp:
		lhs := renderSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		rhs := renderSpan(src, sites, s.RightStart, s.RightEnd, idx)
		fn := "__khiMutLT"
		switch s.Op {
		case token.LEQ:
			fn = "__khiMutLE"
		case token.GTR:
			fn = "__khiMutGT"
		case token.GEQ:
			fn = "__khiMutGE"
		}
		return fmt.Sprintf("%s(%d, %d, %s, %s)", fn, s.Mutants[0].ID, s.Mutants[1].ID, lhs, rhs)

	case siteEqCmp:
		lhs := renderSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		rhs := renderSpan(src, sites, s.RightStart, s.RightEnd, idx)
		opStr := "=="
		if s.Op == token.NEQ {
			opStr = "!="
		}
		return fmt.Sprintf("__khiMutFlip(%d, %s %s %s)", s.Mutants[0].ID, lhs, opStr, rhs)

	case siteLogical:
		lhs := renderSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		rhs := renderSpan(src, sites, s.RightStart, s.RightEnd, idx)
		fn := "__khiMutAnd"
		if s.Op == token.LOR {
			fn = "__khiMutOr"
		}
		return fmt.Sprintf("%s(%d, func() bool { return %s }, func() bool { return %s })", fn, s.Mutants[0].ID, lhs, rhs)

	case siteCond:
		inner := renderSpan(src, sites, s.StartOffset, s.EndOffset, idx)
		return fmt.Sprintf("__khiMutCond(%d, %d, %s)", s.Mutants[0].ID, s.Mutants[1].ID, inner)

	case siteArith:
		lhs := renderSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		rhs := renderSpan(src, sites, s.RightStart, s.RightEnd, idx)
		fn := "__khiMutAddL"
		switch s.Op {
		case token.ADD:
			if s.AddKeepRight {
				fn = "__khiMutAddR"
			}
		case token.SUB:
			fn = "__khiMutSub"
		case token.MUL:
			fn = "__khiMutMul"
		case token.QUO:
			fn = "__khiMutDiv"
		}
		return fmt.Sprintf("%s(%d, %s, %s)", fn, s.Mutants[0].ID, lhs, rhs)

	case siteUnaryNot:
		operand := renderSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		return fmt.Sprintf("__khiMutFlip(%d, !(%s))", s.Mutants[0].ID, operand)

	case siteBoolLit:
		lit := "false"
		if s.BoolVal {
			lit = "true"
		}
		return fmt.Sprintf("__khiMutFlip(%d, %s)", s.Mutants[0].ID, lit)

	case siteStmtSkip:
		stmtText := renderSpan(src, sites, s.StartOffset, s.EndOffset, idx)
		return fmt.Sprintf("if !__khiMutActive(%d) { %s }", s.Mutants[0].ID, stmtText)

	default:
		return string(src[s.StartOffset:s.EndOffset])
	}
}

func collectBlockListStmts(file *ast.File) map[ast.Stmt]struct{} {
	members := make(map[ast.Stmt]struct{})
	ast.Inspect(file, func(n ast.Node) bool {
		switch b := n.(type) {
		case *ast.BlockStmt:
			for _, st := range b.List {
				members[st] = struct{}{}
			}
		case *ast.CaseClause:
			for _, st := range b.Body {
				members[st] = struct{}{}
			}
		case *ast.CommClause:
			for _, st := range b.Body {
				members[st] = struct{}{}
			}
		}
		return true
	})
	return members
}

func shouldApplyBCR(cond ast.Expr) bool {
	unwrapped := ast.Unparen(cond)
	switch e := unwrapped.(type) {
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
	}
	return false
}

func isAllBlankLHS(lhs []ast.Expr) bool {
	if len(lhs) == 0 {
		return true
	}
	for _, e := range lhs {
		ident, ok := ast.Unparen(e).(*ast.Ident)
		if !ok || ident.Name != "_" {
			return false
		}
	}
	return true
}

func isStandardBoolReturnSlot(ancestors []ast.Node, ret *ast.ReturnStmt, target *ast.Ident) bool {
	slotIdx := -1
	for i, res := range ret.Results {
		if res == target {
			slotIdx = i
			break
		}
	}
	if slotIdx < 0 {
		return false
	}

	var funcResults *ast.FieldList
ancestorLoop:
	for i := len(ancestors) - 1; i >= 0; i-- {
		switch fn := ancestors[i].(type) {
		case *ast.FuncLit:
			funcResults = fn.Type.Results
			break ancestorLoop
		case *ast.FuncDecl:
			funcResults = fn.Type.Results
			break ancestorLoop
		}
	}
	if funcResults == nil {
		return false
	}

	var slotTypes []ast.Expr
	for _, field := range funcResults.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			slotTypes = append(slotTypes, field.Type)
		}
	}
	if slotIdx >= len(slotTypes) {
		return false
	}

	switch t := slotTypes[slotIdx].(type) {
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

func buildDiffLines(src []byte, lineStarts []int, line int, startOffset int, endOffset int, replacement string) (string, string) {
	origLine := firstLineTrimmed(src, lineStarts, line)
	if line < 1 || line > len(lineStarts) {
		return origLine, replacement
	}
	lineStart := lineStarts[line-1]
	lineEnd := len(src)
	if line < len(lineStarts) {
		lineEnd = lineStarts[line] - 1
	}
	if lineEnd > lineStart && src[lineEnd-1] == '\r' {
		lineEnd--
	}

	if startOffset >= lineStart && endOffset <= lineEnd {
		var sb strings.Builder
		sb.Write(src[lineStart:startOffset])
		sb.WriteString(replacement)
		sb.Write(src[endOffset:lineEnd])
		return strings.TrimSpace(string(src[lineStart:lineEnd])), strings.TrimSpace(sb.String())
	}
	return origLine, replacement
}

func firstLineTrimmed(src []byte, lineStarts []int, line int) string {
	if line < 1 || line > len(lineStarts) {
		return ""
	}
	lineStart := lineStarts[line-1]
	lineEnd := len(src)
	if line < len(lineStarts) {
		lineEnd = lineStarts[line] - 1
	}
	if lineEnd > lineStart && src[lineEnd-1] == '\r' {
		lineEnd--
	}
	return strings.TrimSpace(string(src[lineStart:lineEnd]))
}

func compactWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func hasLine(lines map[int]struct{}, line int) bool {
	_, ok := lines[line]
	return ok
}
