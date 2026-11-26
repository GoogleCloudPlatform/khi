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
	"go/token"
	"maps"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

var fileStyleRules = []rule.RuleID{
	rule.RuleMultilineStringEscape,
	rule.RuleDeferUnlockBeforeLock,
}

// StyleIssue records a single mechanical Go style or lock-safety defect in a parsed file.
type StyleIssue struct {
	// Rule identifies which Stage-1 AST style rule failed.
	Rule rule.RuleID
	// Line is the 1-based line number of the defect.
	Line int
	// Subject names the enclosing declaration or file marker.
	Subject string
	// Reason describes the specific violation.
	Reason string
}

type formatStringSet map[*ast.BasicLit]struct{}

func evaluateFileStyleIssues(session *evalSession, fm *FileMetrics, ruleIDs []rule.RuleID) {
	baseFM, hasBase := session.base.FileMetrics(fm.File)
	for _, ruleID := range ruleIDs {
		afterIssues := filterStyleIssuesByRule(fm.StyleIssues, ruleID)
		if len(afterIssues) == 0 {
			continue
		}
		before := 0
		if hasBase {
			before = len(filterStyleIssuesByRule(baseFM.StyleIssues, ruleID))
		}
		first := afterIssues[0]
		detail := first.Reason
		if len(afterIssues) > 1 {
			detail = fmt.Sprintf("%s (+%d more occurrence(s) in file)", first.Reason, len(afterIssues)-1)
		}
		hotspots := make([]rule.Hotspot, 0, len(afterIssues))
		for _, issue := range afterIssues {
			hotspots = append(hotspots, rule.Hotspot{
				Line:    issue.Line,
				Delta:   1,
				Summary: issue.Reason,
			})
		}
		session.recordCheck(fm, checkInput{
			ruleID:     ruleID,
			declLine:   first.Line,
			reportLine: first.Line,
			endLine:    first.Line,
			subject:    first.Subject,
			sample: rule.Sample{
				After:     len(afterIssues),
				Limit:     session.opts.Thresholds.Limit(ruleID),
				Before:    before,
				HasBefore: hasBase,
			},
			detail:   detail,
			hotspots: hotspots,
		})
	}
}

func filterStyleIssuesByRule(issues []StyleIssue, ruleID rule.RuleID) []StyleIssue {
	var matched []StyleIssue
	for _, issue := range issues {
		if issue.Rule == ruleID {
			matched = append(matched, issue)
		}
	}
	return matched
}

func analyzeFileStyleIssues(fset *token.FileSet, file *ast.File) []StyleIssue {
	formatLits := collectFormatStringLiterals(file)
	issues := detectMultilineStringEscapes(fset, file, formatLits)
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		issues = append(issues, detectDeferUnlockBeforeLock(fset, fd)...)
	}
	return issues
}

func collectFormatStringLiterals(file *ast.File) formatStringSet {
	result := make(formatStringSet)
	for node := range ast.Preorder(file) {
		callExpr, ok := node.(*ast.CallExpr)
		if !ok || !isFormatCallExpr(callExpr.Fun) {
			continue
		}
		for _, arg := range callExpr.Args {
			if basicLit, isLit := arg.(*ast.BasicLit); isLit && basicLit.Kind == token.STRING {
				result[basicLit] = struct{}{}
			}
		}
	}
	return result
}

func isFormatCallExpr(fun ast.Expr) bool {
	var calleeName string
	switch typedFun := fun.(type) {
	case *ast.Ident:
		calleeName = typedFun.Name
	case *ast.SelectorExpr:
		calleeName = typedFun.Sel.Name
	default:
		return false
	}
	switch calleeName {
	case "Errorf", "Fatalf", "Logf", "Skipf", "Printf", "Fprintf", "Sprintf":
		return true
	default:
		return false
	}
}

func detectMultilineStringEscapes(fset *token.FileSet, file *ast.File, formatLits formatStringSet) []StyleIssue {
	var issues []StyleIssue
	for node := range ast.Preorder(file) {
		basicLit, ok := node.(*ast.BasicLit)
		if !ok || basicLit.Kind != token.STRING {
			continue
		}
		if _, isFormat := formatLits[basicLit]; isFormat {
			continue
		}
		if !isDisallowedEscapedMultilineLiteral(basicLit.Value) {
			continue
		}
		issues = append(issues, StyleIssue{
			Rule:    rule.RuleMultilineStringEscape,
			Line:    fset.Position(basicLit.Pos()).Line,
			Subject: "<file>",
			Reason:  `double-quoted string embeds middle "\n"; use a raw backquote literal`,
		})
	}
	return issues
}

func isDisallowedEscapedMultilineLiteral(rawLiteral string) bool {
	if !strings.HasPrefix(rawLiteral, `"`) {
		return false
	}
	// go/parser validates token.STRING syntax; on error decoded is empty and returns false below.
	decoded, _ := strconv.Unquote(rawLiteral)
	if !strings.Contains(strings.TrimSuffix(decoded, "\n"), "\n") {
		return false
	}
	return !strings.ContainsAny(decoded, "`\r\x1b\x00")
}

func detectDeferUnlockBeforeLock(fset *token.FileSet, fd *ast.FuncDecl) []StyleIssue {
	var issues []StyleIssue
	scanBlockForDeferUnlock(fset, fd.Name.Name, fd.Body.List, make(map[string]bool), &issues)
	return issues
}

func scanBlockForDeferUnlock(fset *token.FileSet, subject string, stmts []ast.Stmt, inherited map[string]bool, issues *[]StyleIssue) {
	locked := maps.Clone(inherited)
	for idx, stmt := range stmts {
		if target, ok := matchDeferUnlockStmt(stmt); ok && !locked[target] {
			if blockListAcquiresTarget(stmts[idx+1:], target) {
				*issues = append(*issues, StyleIssue{
					Rule:    rule.RuleDeferUnlockBeforeLock,
					Line:    fset.Position(stmt.Pos()).Line,
					Subject: subject,
					Reason:  "defer Unlock/RUnlock placed before Lock/RLock acquisition in the same block",
				})
			}
		}
		updateLockedTargetsFromStmt(stmt, locked)
		descendNestedBlocksForLockOrder(fset, subject, stmt, locked, issues)
	}
}

func descendNestedBlocksForLockOrder(fset *token.FileSet, subject string, stmt ast.Stmt, locked map[string]bool, issues *[]StyleIssue) {
	for _, list := range childStmtLists(stmt) {
		scanBlockForDeferUnlock(fset, subject, list, locked, issues)
	}
	if ifStmt, ok := stmt.(*ast.IfStmt); ok && ifStmt.Else != nil {
		descendNestedBlocksForLockOrder(fset, subject, ifStmt.Else, locked, issues)
	}
	// Check function literals with an isolated lock set because closures execute outside the enclosing block's lock order.
	ast.Inspect(stmt, func(n ast.Node) bool {
		if fl, ok := n.(*ast.FuncLit); ok && fl.Body != nil {
			scanBlockForDeferUnlock(fset, subject, fl.Body.List, make(map[string]bool), issues)
			return false
		}
		return true
	})
}

func childStmtLists(stmt ast.Stmt) [][]ast.Stmt {
	var body *ast.BlockStmt
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return [][]ast.Stmt{s.List}
	case *ast.IfStmt:
		body = s.Body
	case *ast.ForStmt:
		body = s.Body
	case *ast.RangeStmt:
		body = s.Body
	case *ast.SwitchStmt:
		return clauseBodyLists(s.Body)
	case *ast.TypeSwitchStmt:
		return clauseBodyLists(s.Body)
	case *ast.SelectStmt:
		return clauseBodyLists(s.Body)
	}
	if body == nil {
		return nil
	}
	return [][]ast.Stmt{body.List}
}

func clauseBodyLists(body *ast.BlockStmt) [][]ast.Stmt {
	if body == nil {
		return nil
	}
	var out [][]ast.Stmt
	for _, st := range body.List {
		switch cc := st.(type) {
		case *ast.CaseClause:
			out = append(out, cc.Body)
		case *ast.CommClause:
			out = append(out, cc.Body)
		}
	}
	return out
}

func matchDeferUnlockStmt(stmt ast.Stmt) (string, bool) {
	def, ok := stmt.(*ast.DeferStmt)
	if !ok {
		return "", false
	}
	return matchLockMethodCall(def.Call, "Unlock", "RUnlock")
}

func blockListAcquiresTarget(stmts []ast.Stmt, target string) bool {
	for _, st := range stmts {
		found := false
		ast.Inspect(st, func(n ast.Node) bool {
			if _, isLit := n.(*ast.FuncLit); isLit {
				return false
			}
			if call, ok := n.(*ast.CallExpr); ok {
				if key, isLock := matchLockMethodCall(call, "Lock", "RLock", "TryLock", "TryRLock"); isLock && key == target {
					found = true
					return false
				}
			}
			return true
		})
		if found {
			return true
		}
	}
	return false
}

func updateLockedTargetsFromStmt(stmt ast.Stmt, locked map[string]bool) {
	if _, isDefer := stmt.(*ast.DeferStmt); isDefer {
		return
	}
	ast.Inspect(stmt, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.FuncLit, *ast.BlockStmt:
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if key, isLock := matchLockMethodCall(call, "Lock", "RLock", "TryLock", "TryRLock"); isLock {
			locked[key] = true
		} else if key, isUnlock := matchLockMethodCall(call, "Unlock", "RUnlock"); isUnlock {
			delete(locked, key)
		}
		return true
	})
}

func matchLockMethodCall(call *ast.CallExpr, methods ...string) (string, bool) {
	if call == nil {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	for _, m := range methods {
		if sel.Sel.Name == m {
			key := formatLockTargetKey(sel.X)
			return key, key != ""
		}
	}
	return "", false
}

func formatLockTargetKey(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if p := formatLockTargetKey(e.X); p != "" {
			return p + "." + e.Sel.Name
		}
	case *ast.IndexExpr:
		p, idx := formatLockTargetKey(e.X), formatLockTargetKey(e.Index)
		if p != "" && idx != "" {
			return p + "[" + idx + "]"
		}
	case *ast.ParenExpr:
		return formatLockTargetKey(e.X)
	case *ast.StarExpr:
		if inner := formatLockTargetKey(e.X); inner != "" {
			return "*" + inner
		}
	}
	return ""
}
