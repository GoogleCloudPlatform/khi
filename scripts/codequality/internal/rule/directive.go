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

package rule

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

const minDirectiveReasonLen = 8

// RawComment holds a 1-based line number and raw comment text (starting with // or /*).
type RawComment struct {
	// Line is the 1-based source line number of the comment.
	Line int `json:"line"`
	// Text is the raw comment text including comment delimiters.
	Text string `json:"text"`
}

// Directive represents a validated // CODEQUALITY_ALLOW(...) suppression comment.
type Directive struct {
	// CommentLine is the 1-based line number containing the directive comment.
	CommentLine int
	// TargetLine is the 1-based line number of the first code line following the comment.
	TargetLine int
	// Rule is the suppressed RuleID.
	Rule RuleID
	// Category is the justification AllowCategory.
	Category AllowCategory
	// AllOperators indicates whether the directive applies to all mutant operators.
	AllOperators bool
	// Operators specifies the subset of mutant operators suppressed when AllOperators is false.
	Operators map[MutantOperator]struct{}
	// Reason is the human-written explanation for the suppression.
	Reason string
	// Used records whether the directive matched at least one violation during evaluation.
	Used bool
}

// UnusedScope defines which file, stages, and target source lines were evaluated during a run.
type UnusedScope struct {
	// File is the slash-separated relative path of the evaluated file.
	File string
	// IncludeDynamic indicates whether StageDynamic rules were evaluated.
	IncludeDynamic bool
	// EvaluatedLines contains 1-based line numbers covered by the evaluation.
	EvaluatedLines map[int]struct{}
}

// ParseDirectives extracts and validates all CODEQUALITY_ALLOW comments in a Go AST file.
func ParseDirectives(relPath string, fset *token.FileSet, file *ast.File, src []byte) ([]Directive, []DirectiveError) {
	codeMask := BuildCodeLineMask(fset, file, src)
	var raw []RawComment
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			raw = append(raw, RawComment{
				Line: fset.Position(c.Pos()).Line,
				Text: c.Text,
			})
		}
	}
	return parseDirectivesWithMask(relPath, raw, codeMask)
}

// ParseDirectivesFromComments extracts and validates CODEQUALITY_ALLOW comments from language-agnostic comments and source bytes.
func ParseDirectivesFromComments(relPath string, comments []RawComment, src []byte) ([]Directive, []DirectiveError) {
	codeMask := buildRawSourceCodeMask(src, comments)
	return parseDirectivesWithMask(relPath, comments, codeMask)
}

// BuildCodeLineMask returns a 1-indexed boolean slice marking lines that contain non-comment, non-blank Go source.
func BuildCodeLineMask(fset *token.FileSet, file *ast.File, src []byte) []bool {
	stripped := make([]byte, len(src))
	copy(stripped, src)
	tf := fset.File(file.Pos())
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			blankCommentSpan(tf, stripped, c)
		}
	}
	lines := strings.Split(string(stripped), "\n")
	mask := make([]bool, len(lines)+1)
	for i, line := range lines {
		mask[i+1] = strings.TrimSpace(line) != ""
	}
	return mask
}

// MatchDeclDirective finds and marks used a CODEQUALITY_ALLOW directive for id targeting declStartLine.
func MatchDeclDirective(directives []Directive, id RuleID, declStartLine int) *Directive {
	for i := range directives {
		d := &directives[i]
		if d.Rule == id && (d.TargetLine == declStartLine || d.CommentLine == declStartLine) {
			d.Used = true
			return d
		}
	}
	return nil
}

// MatchLineDirective finds and marks used a CODEQUALITY_ALLOW directive for id targeting line (and op when non-empty).
func MatchLineDirective(directives []Directive, id RuleID, line int, op MutantOperator) *Directive {
	for i := range directives {
		d := &directives[i]
		if d.Rule != id || (d.TargetLine != line && d.CommentLine != line) {
			continue
		}
		if op != "" && !d.AllOperators {
			if _, ok := d.Operators[op]; !ok {
				continue
			}
		}
		d.Used = true
		return d
	}
	return nil
}

// CollectUnusedDirectives returns errors for unused directives within scope.
func CollectUnusedDirectives(directives []Directive, scope UnusedScope) []DirectiveError {
	var errs []DirectiveError
	for _, d := range directives {
		if d.Used {
			continue
		}
		spec, _ := LookupRuleSpec(d.Rule)
		if !scope.IncludeDynamic && spec.Stage == StageDynamic {
			continue
		}
		if _, inScope := scope.EvaluatedLines[d.TargetLine]; !inScope {
			continue
		}
		errs = append(errs, DirectiveError{
			File:    scope.File,
			Line:    d.CommentLine,
			Message: fmt.Sprintf("unused CODEQUALITY_ALLOW(%s, %s): target has no suppressible HARD_LIMIT violation", d.Rule, d.Category),
		})
	}
	return errs
}

func parseDirectivesWithMask(relPath string, comments []RawComment, codeMask []bool) ([]Directive, []DirectiveError) {
	var directives []Directive
	var errs []DirectiveError
	for _, c := range comments {
		d, errMsg, matched := parseCommentDirective(c.Text, c.Line, codeMask)
		if !matched {
			continue
		}
		if errMsg != "" {
			errs = append(errs, DirectiveError{File: relPath, Line: c.Line, Message: errMsg})
			continue
		}
		directives = append(directives, d)
	}
	return directives, errs
}

func buildRawSourceCodeMask(src []byte, comments []RawComment) []bool {
	lines := strings.Split(string(src), "\n")
	standaloneComment := make(map[int]struct{}, len(comments))
	for _, c := range comments {
		if c.Line >= 1 && c.Line <= len(lines) && strings.TrimSpace(lines[c.Line-1]) == strings.TrimSpace(c.Text) {
			standaloneComment[c.Line] = struct{}{}
		}
	}
	mask := make([]bool, len(lines)+1)
	for i, line := range lines {
		ln := i + 1
		if _, onlyComment := standaloneComment[ln]; onlyComment {
			continue
		}
		trimmed := strings.TrimSpace(line)
		mask[ln] = trimmed != "" && !strings.HasPrefix(trimmed, "//")
	}
	return mask
}

func blankCommentSpan(tf *token.File, buf []byte, c *ast.Comment) {
	start := tf.Offset(c.Pos())
	end := start + len(c.Text)
	for i := start; i < end; i++ {
		if buf[i] != '\n' {
			buf[i] = ' '
		}
	}
}

func parseCommentDirective(rawText string, commentLine int, codeMask []bool) (Directive, string, bool) {
	body := stripCommentDelimiters(rawText)
	if !hasMarkerPrefix(body, "CODEQUALITY_ALLOW") {
		return Directive{}, "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(body, "CODEQUALITY_ALLOW"))
	d, errMsg := parseAllowPayload(rest)
	if errMsg != "" {
		return Directive{}, errMsg, true
	}
	d.CommentLine = commentLine
	d.TargetLine = resolveTargetCodeLine(commentLine, codeMask)
	return d, "", true
}

func stripCommentDelimiters(text string) string {
	trimmed := strings.TrimSpace(text)
	if after, ok := strings.CutPrefix(trimmed, "//"); ok {
		return strings.TrimSpace(after)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(trimmed, "/*"), "*/")
	return strings.TrimSpace(inner)
}

func hasMarkerPrefix(body string, marker string) bool {
	if !strings.HasPrefix(body, marker) {
		return false
	}
	rest := body[len(marker):]
	return rest == "" || rest[0] == '(' || rest[0] == ':' || rest[0] == ' ' || rest[0] == '\t'
}

func resolveTargetCodeLine(commentLine int, codeMask []bool) int {
	for ln := commentLine; ln < len(codeMask); ln++ {
		if codeMask[ln] {
			return ln
		}
	}
	return commentLine
}

func parseAllowPayload(rest string) (Directive, string) {
	rawArgs, reason, errMsg := splitArgsAndReason(rest)
	if errMsg != "" {
		return Directive{}, errMsg
	}
	parts := strings.Split(rawArgs, ",")
	if len(parts) < 2 {
		return Directive{}, "CODEQUALITY_ALLOW requires (<rule_id>, <category>): <reason>"
	}
	ruleID := RuleID(strings.TrimSpace(parts[0]))
	spec, ok := LookupRuleSpec(ruleID)
	if !ok {
		return Directive{}, fmt.Sprintf("unknown CODEQUALITY_ALLOW rule %q", ruleID)
	}
	if len(spec.AllowedCategories) == 0 {
		return Directive{}, fmt.Sprintf("rule %q does not allow CODEQUALITY_ALLOW suppression", ruleID)
	}
	cat, errMsg := validateCategory(spec, strings.TrimSpace(parts[1]))
	if errMsg != "" {
		return Directive{}, errMsg
	}
	ops, errMsg := parseOptionalOperators(ruleID, parts[2:])
	if errMsg != "" {
		return Directive{}, errMsg
	}
	return Directive{
		Rule:         ruleID,
		Category:     cat,
		AllOperators: ops == nil,
		Operators:    ops,
		Reason:       reason,
	}, ""
}

func splitArgsAndReason(rest string) (string, string, string) {
	afterOpen, hasOpen := strings.CutPrefix(rest, "(")
	if !hasOpen {
		return "", "", "CODEQUALITY_ALLOW requires '(<rule_id>, <category>): <reason>'"
	}
	rawArgs, afterClose, hasClose := strings.Cut(afterOpen, ")")
	if !hasClose {
		return "", "", "CODEQUALITY_ALLOW is missing closing ')' and ': <reason>'"
	}
	reasonPart, hasColon := strings.CutPrefix(strings.TrimSpace(afterClose), ":")
	if !hasColon {
		return "", "", "CODEQUALITY_ALLOW requires ': <reason>' after ')'"
	}
	reason := strings.TrimSpace(reasonPart)
	if len(reason) < minDirectiveReasonLen {
		return "", "", "CODEQUALITY_ALLOW requires a descriptive reason (>= 8 chars) after ':'"
	}
	return strings.TrimSpace(rawArgs), reason, ""
}

func validateCategory(spec RuleSpec, rawCat string) (AllowCategory, string) {
	if isForbiddenInvariantCategory(rawCat) {
		return "", "invariant or unreachable branches cannot be suppressed; simplify or delete the redundant branch instead"
	}
	cat := AllowCategory(rawCat)
	if !spec.AllowsCategory(cat) {
		return "", fmt.Sprintf("category %q is not allowed for rule %q", rawCat, spec.ID)
	}
	return cat, ""
}

func parseOptionalOperators(ruleID RuleID, extraParts []string) (map[MutantOperator]struct{}, string) {
	if len(extraParts) == 0 {
		return nil, ""
	}
	if ruleID != RuleFuncSurvivingMutants {
		return nil, fmt.Sprintf("operator filters are only supported for %q", RuleFuncSurvivingMutants)
	}
	ops := make(map[MutantOperator]struct{}, len(extraParts))
	for _, raw := range extraParts {
		op := MutantOperator(strings.TrimSpace(raw))
		if op == "" {
			continue
		}
		if !op.IsValid() {
			return nil, fmt.Sprintf("unknown mutant operator %q in CODEQUALITY_ALLOW", op)
		}
		ops[op] = struct{}{}
	}
	if len(ops) == 0 {
		return nil, "CODEQUALITY_ALLOW operator filter list cannot be empty"
	}
	return ops, ""
}

func isForbiddenInvariantCategory(raw string) bool {
	switch strings.ToLower(raw) {
	case "invariant", "unreachable", "defensive", "dead", "impossible", "redundant", "always":
		return true
	default:
		return false
	}
}
