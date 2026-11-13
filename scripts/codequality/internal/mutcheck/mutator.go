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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// Mutant describes a single candidate logic change extracted from a Go function.
type Mutant struct {
	ID                  int
	File                string
	FuncKey             string
	FuncDisplay         string
	Line                int
	Operator            rule.MutantOperator
	OriginalExpr        string
	MutatedExpr         string
	OriginalLine        string
	MutatedLine         string
	SuppressionCategory rule.AllowCategory
	Status              rule.MutantStatus
}

// ToDetail converts m into a report-ready rule.MutantDetail.
func (m *Mutant) ToDetail() rule.MutantDetail {
	return rule.MutantDetail{
		Operator:     m.Operator,
		Line:         m.Line,
		OriginalExpr: m.OriginalExpr,
		MutatedExpr:  m.MutatedExpr,
		OriginalLine: m.OriginalLine,
		MutatedLine:  m.MutatedLine,
	}
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

// MutationSite represents one AST span in a Go file that expands into Super-Mutant runtime code.
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
}

// PlanOptions configures which lines, directives, and exclusion rules apply when extracting mutants.
type PlanOptions struct {
	TargetLines  map[int]struct{}
	CoveredLines map[int]struct{}
	Directives   []rule.Directive
	Rules        []rule.ExclusionRule
	NextMutantID *int
}

// PlanFileMutations parses src and extracts all candidate mutants on opts.TargetLines.
func PlanFileMutations(filePath string, src []byte, opts PlanOptions) (*FilePlan, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}
	plan := &FilePlan{
		File:        filePath,
		PackageName: file.Name.Name,
		Source:      src,
	}
	if hasBuildIgnoreDirective(file) {
		return plan, nil
	}

	rules := opts.Rules
	if rules == nil {
		rules = rule.DefaultExclusionRules()
	}
	nextID := opts.NextMutantID
	if nextID == nil {
		localID := 1
		nextID = &localID
	}

	w := &funcWalker{
		fset:            fset,
		file:            file,
		src:             src,
		lineStarts:      computeLineOffsets(src),
		stmtListMembers: collectBlockListStmts(file),
		opts:            opts,
		rules:           rules,
		nextID:          nextID,
		plan:            plan,
	}
	for _, decl := range file.Decls {
		fnDecl, ok := decl.(*ast.FuncDecl)
		if !ok || fnDecl.Body == nil {
			continue
		}
		w.curFuncKey, w.curFuncDisplay = astcheck.FormatFuncIdentity(fnDecl)
		w.curFuncLine = fset.Position(fnDecl.Pos()).Line
		w.walkNode(fnDecl.Body, []ast.Node{file, fnDecl})
	}
	return plan, nil
}

type funcWalker struct {
	fset            *token.FileSet
	file            *ast.File
	src             []byte
	lineStarts      []int
	stmtListMembers map[ast.Stmt]struct{}
	opts            PlanOptions
	rules           []rule.ExclusionRule
	nextID          *int
	plan            *FilePlan
	curFuncKey      string
	curFuncDisplay  string
	curFuncLine     int
}

type siteProposal struct {
	line           int
	op             rule.MutantOperator
	kind           siteKind
	site           MutationSite
	specs          [][2]string
	customMutLines []string
}

func (w *funcWalker) walkNode(node ast.Node, ancestors []ast.Node) {
	ctx := &rule.ExclusionContext{
		FilePath:  w.plan.File,
		Fset:      w.fset,
		File:      w.file,
		Source:    w.src,
		Ancestors: ancestors,
	}
	scope := rule.EvaluateExclusionRules(w.rules, ctx, node)
	if scope == rule.ExclusionSubtree {
		return
	}
	if scope == rule.ExclusionNone {
		w.extractNodeSite(node, ctx)
	}
	ancestors = append(ancestors, node)
	ast.Inspect(node, func(child ast.Node) bool {
		if child == node {
			return true
		}
		if child != nil {
			w.walkNode(child, ancestors)
		}
		return false
	})
}

func (w *funcWalker) extractNodeSite(node ast.Node, ctx *rule.ExclusionContext) {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		w.extractBinarySite(n)
	case *ast.IfStmt:
		w.extractIfSite(n)
	case *ast.UnaryExpr:
		w.extractUnarySite(n)
	case *ast.Ident:
		w.extractReturnBoolSite(n, ctx)
	case *ast.AssignStmt:
		w.extractAssignSite(n)
	case *ast.IncDecStmt:
		w.extractIncDecSite(n)
	}
}

func (w *funcWalker) register(p siteProposal) {
	if _, changed := w.opts.TargetLines[p.line]; !changed {
		return
	}
	status, supCat := w.classifyInitialStatus(p.line, p.op)
	siteMutants := make([]*Mutant, 0, len(p.specs))
	for i, sp := range p.specs {
		origLine, mutLine := w.buildDiffLines(p.line, p.site, sp[1])
		if i < len(p.customMutLines) {
			mutLine = p.customMutLines[i]
		}
		m := &Mutant{
			File:                w.plan.File,
			FuncKey:             w.curFuncKey,
			FuncDisplay:         w.curFuncDisplay,
			Line:                p.line,
			Operator:            p.op,
			OriginalExpr:        sp[0],
			MutatedExpr:         sp[1],
			OriginalLine:        origLine,
			MutatedLine:         mutLine,
			SuppressionCategory: supCat,
			Status:              status,
		}
		if status == rule.MutantPending {
			m.ID = *w.nextID
			*w.nextID++
		}
		w.plan.Mutants = append(w.plan.Mutants, m)
		siteMutants = append(siteMutants, m)
	}
	if status == rule.MutantPending {
		p.site.Kind = p.kind
		p.site.Mutants = siteMutants
		w.plan.Sites = append(w.plan.Sites, p.site)
	}
}

func (w *funcWalker) classifyInitialStatus(line int, op rule.MutantOperator) (rule.MutantStatus, rule.AllowCategory) {
	if d := rule.MatchLineDirective(w.opts.Directives, rule.RuleFuncSurvivingMutants, line, op); d != nil {
		return rule.MutantSuppressed, d.Category
	}
	if d := rule.MatchLineDirective(w.opts.Directives, rule.RuleFuncSurvivingMutants, w.curFuncLine, op); d != nil {
		return rule.MutantSuppressed, d.Category
	}
	if w.opts.CoveredLines != nil {
		if _, ok := w.opts.CoveredLines[line]; !ok {
			return rule.MutantUncovered, ""
		}
	}
	return rule.MutantPending, ""
}

func (w *funcWalker) extractBinarySite(n *ast.BinaryExpr) {
	opLine := w.fset.Position(n.OpPos).Line
	start := w.fset.Position(n.Pos()).Offset
	end := w.fset.Position(n.End()).Offset
	lStart := w.fset.Position(n.X.Pos()).Offset
	lEnd := w.fset.Position(n.X.End()).Offset
	rStart := w.fset.Position(n.Y.Pos()).Offset
	rEnd := w.fset.Position(n.Y.End()).Offset
	lhs := compactWhitespace(string(w.src[lStart:lEnd]))
	rhs := compactWhitespace(string(w.src[rStart:rEnd]))
	orig := compactWhitespace(string(w.src[start:end]))

	base := MutationSite{
		StartOffset: start,
		EndOffset:   end,
		Op:          n.Op,
		LeftStart:   lStart,
		LeftEnd:     lEnd,
		RightStart:  rStart,
		RightEnd:    rEnd,
	}
	specRes, ok := buildBinaryMutantSpecs(n, orig, lhs, rhs, &base)
	if !ok {
		return
	}
	w.register(siteProposal{line: opLine, op: specRes.op, kind: specRes.kind, site: base, specs: specRes.specs})
}

type binarySpecResult struct {
	op    rule.MutantOperator
	kind  siteKind
	specs [][2]string
}

func buildBinaryMutantSpecs(
	n *ast.BinaryExpr,
	orig string,
	lhs string,
	rhs string,
	base *MutationSite,
) (binarySpecResult, bool) {
	switch n.Op {
	case token.LSS:
		return binarySpecResult{op: rule.OperatorROR, kind: siteOrderedCmp, specs: [][2]string{{orig, lhs + " <= " + rhs}, {orig, lhs + " >= " + rhs}}}, true
	case token.LEQ:
		return binarySpecResult{op: rule.OperatorROR, kind: siteOrderedCmp, specs: [][2]string{{orig, lhs + " < " + rhs}, {orig, lhs + " > " + rhs}}}, true
	case token.GTR:
		return binarySpecResult{op: rule.OperatorROR, kind: siteOrderedCmp, specs: [][2]string{{orig, lhs + " >= " + rhs}, {orig, lhs + " <= " + rhs}}}, true
	case token.GEQ:
		return binarySpecResult{op: rule.OperatorROR, kind: siteOrderedCmp, specs: [][2]string{{orig, lhs + " > " + rhs}, {orig, lhs + " < " + rhs}}}, true
	case token.EQL:
		return binarySpecResult{op: rule.OperatorROR, kind: siteEqCmp, specs: [][2]string{{orig, lhs + " != " + rhs}}}, true
	case token.NEQ:
		return binarySpecResult{op: rule.OperatorROR, kind: siteEqCmp, specs: [][2]string{{orig, lhs + " == " + rhs}}}, true
	case token.LAND:
		return binarySpecResult{op: rule.OperatorLCR, kind: siteLogical, specs: [][2]string{{orig, lhs + " || " + rhs}}}, true
	case token.LOR:
		return binarySpecResult{op: rule.OperatorLCR, kind: siteLogical, specs: [][2]string{{orig, lhs + " && " + rhs}}}, true
	default:
		return buildArithMutantSpecs(n, orig, lhs, rhs, base)
	}
}

func buildArithMutantSpecs(
	n *ast.BinaryExpr,
	orig string,
	lhs string,
	rhs string,
	base *MutationSite,
) (binarySpecResult, bool) {
	switch n.Op {
	case token.ADD:
		keepRight := isZeroOrEmptyLiteral(n.X)
		base.AddKeepRight = keepRight
		kept := lhs
		if keepRight {
			kept = rhs
		}
		return binarySpecResult{op: rule.OperatorAOR, kind: siteArith, specs: [][2]string{{orig, kept}}}, true
	case token.SUB:
		return binarySpecResult{op: rule.OperatorAOR, kind: siteArith, specs: [][2]string{{orig, lhs + " + " + rhs}}}, true
	case token.MUL:
		return binarySpecResult{op: rule.OperatorAOR, kind: siteArith, specs: [][2]string{{orig, lhs + " / " + rhs}}}, true
	case token.QUO:
		return binarySpecResult{op: rule.OperatorAOR, kind: siteArith, specs: [][2]string{{orig, lhs + " * " + rhs}}}, true
	default:
		return binarySpecResult{}, false
	}
}

func (w *funcWalker) extractIfSite(n *ast.IfStmt) {
	if !shouldApplyBCR(n.Cond) {
		return
	}
	condLine := w.fset.Position(n.Cond.Pos()).Line
	cStart := w.fset.Position(n.Cond.Pos()).Offset
	cEnd := w.fset.Position(n.Cond.End()).Offset
	orig := compactWhitespace(string(w.src[cStart:cEnd]))
	w.register(siteProposal{
		line:  condLine,
		op:    rule.OperatorBCR,
		kind:  siteCond,
		site:  MutationSite{StartOffset: cStart, EndOffset: cEnd},
		specs: [][2]string{{orig, "true"}, {orig, "false"}},
	})
}

func (w *funcWalker) extractUnarySite(n *ast.UnaryExpr) {
	if n.Op != token.NOT {
		return
	}
	line := w.fset.Position(n.OpPos).Line
	start := w.fset.Position(n.Pos()).Offset
	end := w.fset.Position(n.End()).Offset
	xStart := w.fset.Position(n.X.Pos()).Offset
	xEnd := w.fset.Position(n.X.End()).Offset
	orig := compactWhitespace(string(w.src[start:end]))
	xText := compactWhitespace(string(w.src[xStart:xEnd]))
	w.register(siteProposal{
		line:  line,
		op:    rule.OperatorUOI,
		kind:  siteUnaryNot,
		site:  MutationSite{StartOffset: start, EndOffset: end, LeftStart: xStart, LeftEnd: xEnd},
		specs: [][2]string{{orig, xText}},
	})
}

func (w *funcWalker) extractReturnBoolSite(n *ast.Ident, ctx *rule.ExclusionContext) {
	if n.Name != "true" && n.Name != "false" {
		return
	}
	retStmt, ok := ctx.Parent().(*ast.ReturnStmt)
	if !ok || !isStandardBoolReturnSlot(ctx.Ancestors, retStmt, n) {
		return
	}
	line := w.fset.Position(n.Pos()).Line
	start := w.fset.Position(n.Pos()).Offset
	end := w.fset.Position(n.End()).Offset
	flipped := "false"
	if n.Name == "false" {
		flipped = "true"
	}
	w.register(siteProposal{
		line:  line,
		op:    rule.OperatorBLR,
		kind:  siteBoolLit,
		site:  MutationSite{StartOffset: start, EndOffset: end, BoolVal: n.Name == "true"},
		specs: [][2]string{{n.Name, flipped}},
	})
}

func (w *funcWalker) extractAssignSite(n *ast.AssignStmt) {
	if _, inBlock := w.stmtListMembers[n]; !inBlock {
		return
	}
	if !isMutatableAssignTok(n.Tok) || isAllBlankLHS(n.Lhs) {
		return
	}
	line := w.fset.Position(n.TokPos).Line
	start := w.fset.Position(n.Pos()).Offset
	end := w.fset.Position(n.End()).Offset
	orig := compactWhitespace(string(w.src[start:end]))
	w.register(siteProposal{
		line:           line,
		op:             rule.OperatorSBR,
		kind:           siteStmtSkip,
		site:           MutationSite{StartOffset: start, EndOffset: end},
		specs:          [][2]string{{orig, "/* skipped assignment */"}},
		customMutLines: []string{"/* skipped: " + firstLineTrimmed(w.src, w.lineStarts, line) + " */"},
	})
}

func (w *funcWalker) extractIncDecSite(n *ast.IncDecStmt) {
	if _, inBlock := w.stmtListMembers[n]; !inBlock {
		return
	}
	line := w.fset.Position(n.TokPos).Line
	start := w.fset.Position(n.Pos()).Offset
	end := w.fset.Position(n.End()).Offset
	orig := compactWhitespace(string(w.src[start:end]))
	w.register(siteProposal{
		line:           line,
		op:             rule.OperatorSBR,
		kind:           siteStmtSkip,
		site:           MutationSite{StartOffset: start, EndOffset: end},
		specs:          [][2]string{{orig, "/* skipped " + orig + " */"}},
		customMutLines: []string{"/* skipped: " + firstLineTrimmed(w.src, w.lineStarts, line) + " */"},
	})
}
