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
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// ExclusionScope specifies how far an exclusion decision applies during AST traversal.
type ExclusionScope int

const (
	// ExclusionNone allows normal mutation analysis on the node and its children.
	ExclusionNone ExclusionScope = iota
	// ExclusionNodeOnly skips generating mutants directly on this node while still visiting its children.
	ExclusionNodeOnly
	// ExclusionSubtree skips generating mutants on this node and all of its AST descendants.
	ExclusionSubtree
)

// ExclusionContext provides AST, token position, and ancestor chain context to ExclusionRule implementations.
type ExclusionContext struct {
	// FilePath is the repository-relative slash-separated path of the Go source file.
	FilePath string
	// Fset holds token position metadata for File.
	Fset *token.FileSet
	// File is the parsed AST of the Go source file.
	File *ast.File
	// Source contains the raw UTF-8 bytes of the Go source file.
	Source []byte
	// Ancestors contains enclosing AST nodes ordered from outermost (*ast.File) to immediate parent.
	Ancestors []ast.Node
}

// Parent returns the immediate enclosing AST node, or nil when at the root.
func (c *ExclusionContext) Parent() ast.Node {
	if len(c.Ancestors) == 0 {
		return nil
	}
	return c.Ancestors[len(c.Ancestors)-1]
}

// ExclusionRule determines whether an AST node or its entire subtree should be excluded from mutation.
type ExclusionRule interface {
	// Name returns a stable identifier for this exclusion rule.
	Name() string
	// Evaluate inspects node within ctx and returns the exclusion scope to apply.
	Evaluate(ctx *ExclusionContext, node ast.Node) ExclusionScope
}

// DefaultExclusionRules returns the standard set of KHI noise-suppression rules.
func DefaultExclusionRules() []ExclusionRule {
	return []ExclusionRule{
		&constAndTypeRule{},
		&errorGuardRule{},
		&loggingAndDiagCallRule{},
		&allocationHintRule{},
		&deferAndPanicRule{},
	}
}

// EvaluateExclusionRules runs rules against node and returns the strongest ExclusionScope matched.
func EvaluateExclusionRules(rules []ExclusionRule, ctx *ExclusionContext, node ast.Node) ExclusionScope {
	scope := ExclusionNone
	for _, rule := range rules {
		switch rule.Evaluate(ctx, node) {
		case ExclusionSubtree:
			return ExclusionSubtree
		case ExclusionNodeOnly:
			scope = ExclusionNodeOnly
		}
	}
	return scope
}

type constAndTypeRule struct{}

var _ ExclusionRule = (*constAndTypeRule)(nil)

func (r *constAndTypeRule) Name() string {
	return "const_and_type"
}

func (r *constAndTypeRule) Evaluate(_ *ExclusionContext, node ast.Node) ExclusionScope {
	switch n := node.(type) {
	case *ast.GenDecl:
		if n.Tok == token.CONST || n.Tok == token.TYPE {
			return ExclusionSubtree
		}
	case *ast.ArrayType, *ast.StructType, *ast.InterfaceType, *ast.MapType, *ast.ChanType, *ast.FuncType:
		return ExclusionSubtree
	case *ast.BinaryExpr:
		if isPureLiteralExpr(n) {
			return ExclusionSubtree
		}
	}
	return ExclusionNone
}

type errorGuardRule struct{}

var _ ExclusionRule = (*errorGuardRule)(nil)

func (r *errorGuardRule) Name() string {
	return "error_guard"
}

func (r *errorGuardRule) Evaluate(_ *ExclusionContext, node ast.Node) ExclusionScope {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		if isErrorNilComparison(n) {
			return ExclusionSubtree
		}
	case *ast.IfStmt:
		if n.Else == nil && isErrorNotNilGuard(n.Cond) && endsWithReturnStmt(n.Body) {
			return ExclusionSubtree
		}
	}
	return ExclusionNone
}

type loggingAndDiagCallRule struct{}

var _ ExclusionRule = (*loggingAndDiagCallRule)(nil)

func (r *loggingAndDiagCallRule) Name() string {
	return "logging_and_diag_call"
}

func (r *loggingAndDiagCallRule) Evaluate(_ *ExclusionContext, node ast.Node) ExclusionScope {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return ExclusionNone
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ExclusionNone
	}
	recvIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return ExclusionNone
	}
	pkgOrVar := recvIdent.Name
	method := sel.Sel.Name

	switch pkgOrVar {
	case "slog", "log", "klog", "otel", "span", "tracer":
		return ExclusionSubtree
	case "fmt":
		if strings.HasPrefix(method, "Print") || strings.HasPrefix(method, "Fprint") || method == "Errorf" {
			return ExclusionSubtree
		}
	case "errors":
		if method == "New" || method == "Join" {
			return ExclusionSubtree
		}
	}
	return ExclusionNone
}

type allocationHintRule struct{}

var _ ExclusionRule = (*allocationHintRule)(nil)

func (r *allocationHintRule) Name() string {
	return "allocation_hint"
}

func (r *allocationHintRule) Evaluate(ctx *ExclusionContext, node ast.Node) ExclusionScope {
	if call, ok := node.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Grow" {
			return ExclusionSubtree
		}
	}

	parentCall, ok := ctx.Parent().(*ast.CallExpr)
	if !ok {
		return ExclusionNone
	}
	funIdent, ok := parentCall.Fun.(*ast.Ident)
	if !ok || funIdent.Name != "make" {
		return ExclusionNone
	}

	if len(parentCall.Args) == 3 && parentCall.Args[2] == node {
		if arrType, ok := parentCall.Args[0].(*ast.ArrayType); ok && arrType.Len == nil {
			return ExclusionSubtree
		}
	}
	if len(parentCall.Args) == 2 && parentCall.Args[1] == node {
		if _, ok := parentCall.Args[0].(*ast.MapType); ok {
			return ExclusionSubtree
		}
	}
	return ExclusionNone
}

type deferAndPanicRule struct{}

var _ ExclusionRule = (*deferAndPanicRule)(nil)

func (r *deferAndPanicRule) Name() string {
	return "defer_and_panic"
}

func (r *deferAndPanicRule) Evaluate(_ *ExclusionContext, node ast.Node) ExclusionScope {
	switch n := node.(type) {
	case *ast.DeferStmt:
		return ExclusionSubtree
	case *ast.CallExpr:
		if ident, ok := n.Fun.(*ast.Ident); ok && ident.Name == "panic" {
			return ExclusionSubtree
		}
	}
	return ExclusionNone
}

// SuppressionCategory classifies why a mutant is legitimately suppressed via // NOMUTANTS.
type SuppressionCategory string

const (
	// SuppressionEquivalent marks mutations that yield identical observable outputs for all valid inputs.
	SuppressionEquivalent SuppressionCategory = "equivalent"
	// SuppressionPerf marks caching, fast-path, or pre-sizing optimizations that do not alter functional output.
	SuppressionPerf SuppressionCategory = "perf"
	// SuppressionObservability marks telemetry, progress, or diagnostic operations outside domain behavior.
	SuppressionObservability SuppressionCategory = "observability"
	// SuppressionExternal marks OS, subprocess, or external I/O boundaries covered by integration tests instead of unit mocks.
	SuppressionExternal SuppressionCategory = "external"
)

// AllSuppressionCategories lists all valid SuppressionCategory values in deterministic reporting order.
var AllSuppressionCategories = []SuppressionCategory{
	SuppressionEquivalent,
	SuppressionPerf,
	SuppressionObservability,
	SuppressionExternal,
}

// IsValid reports whether c is one of the allowed SuppressionCategory values.
func (c SuppressionCategory) IsValid() bool {
	switch c {
	case SuppressionEquivalent, SuppressionPerf, SuppressionObservability, SuppressionExternal:
		return true
	default:
		return false
	}
}

// DirectiveWarning describes an invalid // NOMUTANTS comment encountered in source code.
type DirectiveWarning struct {
	// File is the repository-relative path where the invalid directive was found.
	File string
	// Line is the 1-based line number of the comment.
	Line int
	// Message explains why the directive was rejected.
	Message string
}

type nomutantsRule struct {
	category     SuppressionCategory
	allOperators bool
	operators    map[MutantOperator]struct{}
}

// SuppressionIndex maps 1-based source lines to parsed // NOMUTANTS suppression directives.
type SuppressionIndex struct {
	byLine   map[int][]nomutantsRule
	Warnings []DirectiveWarning
}

// ParseSuppressions extracts // NOMUTANTS directives and validation warnings from a parsed Go file.
func ParseSuppressions(filePath string, fset *token.FileSet, file *ast.File, src []byte) SuppressionIndex {
	idx := SuppressionIndex{
		byLine: make(map[int][]nomutantsRule),
	}
	sourceLines := strings.Split(string(src), "\n")

	for _, group := range file.Comments {
		for _, c := range group.List {
			pos := fset.Position(c.Slash)
			rule, warnMsg, matched := parseNomutantsComment(c.Text)
			if !matched {
				continue
			}
			if warnMsg != "" {
				idx.Warnings = append(idx.Warnings, DirectiveWarning{
					File:    filePath,
					Line:    pos.Line,
					Message: warnMsg,
				})
				continue
			}

			idx.byLine[pos.Line] = append(idx.byLine[pos.Line], rule)

			// Standalone comment lines apply to the next non-comment, non-blank source line.
			if isStandaloneCommentLine(sourceLines, pos.Line) {
				targetLine := nextCodeLine(sourceLines, pos.Line)
				if targetLine > 0 {
					idx.byLine[targetLine] = append(idx.byLine[targetLine], rule)
				}
			}
		}
	}
	return idx
}

// LookupSuppression returns the matched SuppressionCategory if a mutant of operator on line is suppressed.
func (s *SuppressionIndex) LookupSuppression(line int, operator MutantOperator) (SuppressionCategory, bool) {
	for _, r := range s.byLine[line] {
		if r.allOperators {
			return r.category, true
		}
		if _, ok := r.operators[operator]; ok {
			return r.category, true
		}
	}
	return "", false
}

func parseNomutantsComment(commentText string) (nomutantsRule, string, bool) {
	trimmed := strings.TrimSpace(commentText)
	switch {
	case strings.HasPrefix(trimmed, "//"):
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
	case strings.HasPrefix(trimmed, "/*") && strings.HasSuffix(trimmed, "*/"):
		trimmed = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "/*"), "*/"))
	default:
		return nomutantsRule{}, "", false
	}

	const marker = "NOMUTANTS"
	if !strings.HasPrefix(trimmed, marker) {
		return nomutantsRule{}, "", false
	}
	rest := trimmed[len(marker):]
	if rest != "" && rest[0] != ':' && rest[0] != '(' && rest[0] != ' ' && rest[0] != '\t' {
		return nomutantsRule{}, "", false
	}

	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "(") {
		return nomutantsRule{}, "// NOMUTANTS requires a category and reason: // NOMUTANTS(<category>): <reason> (allowed categories: equivalent, perf, observability, external)", true
	}

	closeIdx := strings.IndexByte(rest, ')')
	if closeIdx < 0 {
		return nomutantsRule{}, "// NOMUTANTS(...) is missing closing ')' and ': <reason>'", true
	}

	rawArgs := strings.TrimSpace(rest[1:closeIdx])
	if rawArgs == "" {
		return nomutantsRule{}, "// NOMUTANTS(...) requires a category (allowed: equivalent, perf, observability, external)", true
	}

	parts := strings.Split(rawArgs, ",")
	rawCat := strings.TrimSpace(parts[0])
	if isForbiddenInvariantCategory(rawCat) {
		return nomutantsRule{}, "invariant or unreachable conditions cannot be suppressed with NOMUTANTS; simplify or delete the redundant branch instead (allowed categories: equivalent, perf, observability, external)", true
	}
	cat := SuppressionCategory(rawCat)
	if !cat.IsValid() {
		return nomutantsRule{}, fmt.Sprintf("unknown NOMUTANTS category %q (allowed categories: equivalent, perf, observability, external; format: // NOMUTANTS(<category>[, <operator>...]): <reason>)", rawCat), true
	}

	rule := nomutantsRule{
		category:     cat,
		allOperators: true,
	}
	if len(parts) > 1 {
		rule.allOperators = false
		rule.operators = make(map[MutantOperator]struct{}, len(parts)-1)
		for _, part := range parts[1:] {
			op := MutantOperator(strings.TrimSpace(part))
			if op == "" {
				continue
			}
			if !op.IsValid() {
				return nomutantsRule{}, fmt.Sprintf("unknown operator %q in // NOMUTANTS(...)", op), true
			}
			rule.operators[op] = struct{}{}
		}
		if len(rule.operators) == 0 {
			return nomutantsRule{}, "// NOMUTANTS(<category>, ...) has empty operator list", true
		}
	}

	rest = strings.TrimSpace(rest[closeIdx+1:])
	if !strings.HasPrefix(rest, ":") {
		return nomutantsRule{}, "// NOMUTANTS(<category>) requires ': <reason>' explaining why mutation is suppressed", true
	}
	reason := strings.TrimSpace(strings.TrimPrefix(rest, ":"))
	if reason == "" {
		return nomutantsRule{}, "// NOMUTANTS(<category>) requires a non-empty reason after ':'", true
	}
	return rule, "", true
}

func isForbiddenInvariantCategory(raw string) bool {
	switch strings.ToLower(raw) {
	case "invariant", "unreachable", "defensive", "dead", "impossible", "redundant", "always":
		return true
	default:
		return false
	}
}

func isStandaloneCommentLine(lines []string, lineNum int) bool {
	if lineNum < 1 || lineNum > len(lines) {
		return false
	}
	trimmed := strings.TrimSpace(lines[lineNum-1])
	return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*")
}

func nextCodeLine(lines []string, startLine int) int {
	for i := startLine; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		return i + 1
	}
	return 0
}

func isPureLiteralExpr(expr ast.Expr) bool {
	switch e := ast.Unparen(expr).(type) {
	case *ast.BasicLit:
		return true
	case *ast.UnaryExpr:
		if e.Op == token.SUB || e.Op == token.ADD {
			return isPureLiteralExpr(e.X)
		}
	case *ast.BinaryExpr:
		return isPureLiteralExpr(e.X) && isPureLiteralExpr(e.Y)
	}
	return false
}

func isErrorNilComparison(bin *ast.BinaryExpr) bool {
	if bin.Op != token.EQL && bin.Op != token.NEQ {
		return false
	}
	if isNilIdent(bin.Y) && isErrorLikeExpr(bin.X) {
		return true
	}
	if isNilIdent(bin.X) && isErrorLikeExpr(bin.Y) {
		return true
	}
	return false
}

func isErrorNotNilGuard(expr ast.Expr) bool {
	bin, ok := ast.Unparen(expr).(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}
	return (isNilIdent(bin.Y) && isErrorLikeExpr(bin.X)) || (isNilIdent(bin.X) && isErrorLikeExpr(bin.Y))
}

func isNilIdent(expr ast.Expr) bool {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	return ok && ident.Name == "nil"
}

func isErrorLikeExpr(expr ast.Expr) bool {
	switch e := ast.Unparen(expr).(type) {
	case *ast.Ident:
		return isErrorName(e.Name)
	case *ast.SelectorExpr:
		return isErrorName(e.Sel.Name)
	case *ast.CallExpr:
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok && len(e.Args) == 0 {
			return sel.Sel.Name == "Err"
		}
	}
	return false
}

func isErrorName(name string) bool {
	return strings.EqualFold(name, "err") || strings.HasSuffix(name, "Err") || strings.HasSuffix(name, "Error")
}

func endsWithReturnStmt(block *ast.BlockStmt) bool {
	if block == nil || len(block.List) == 0 {
		return false
	}
	_, ok := block.List[len(block.List)-1].(*ast.ReturnStmt)
	return ok
}
