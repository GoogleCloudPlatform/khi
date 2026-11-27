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

package advisorycheck

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

const (
	tsTokIdent   tsTokenKind = "ident"
	tsTokString  tsTokenKind = "string"
	tsTokComment tsTokenKind = "comment"
	tsTokPunct   tsTokenKind = "punct"
)

type tsTokenKind string

type tsToken struct {
	kind tsTokenKind
	text string
	line int
}

type tsFileInput struct {
	relPath      string
	targetSrc    []byte
	baseSrc      []byte
	hasBase      bool
	changedLines map[int]struct{}
	addedByDir   map[string][]string
}

type tsSpecEntry struct {
	name string
	line int
}

type tsSpecMetrics struct {
	specs          []tsSpecEntry
	itCount        int
	eqAssertCount  int
	expectCount    int
	skipHits       []lineSubject
	doubleCastHits []int
	suppressHits   []lineSubject
}

func analyzeDeletedTSTestFile(req rule.AdvisoryRequest, relPath string, addedByDir map[string][]string) []rule.QAItem {
	dir := filepath.ToSlash(filepath.Dir(relPath))
	candidates := addedByDir[dir]
	baseSrc, hasBase := readBaseBytes(req.ReadBaseFile, relPath)
	if hasBase {
		metrics := analyzeTSMetrics(baseSrc)
		if len(metrics.specs) > 0 {
			var items []rule.QAItem
			for _, sp := range metrics.specs {
				item := makeAdvisoryItem(checkTestRemoved, relPath, sp.line, sp.name, fmt.Sprintf("spec %q removed with deleted file", sp.name))
				item.Evidence = map[string]any{"test_name": sp.name, "renamed_candidates": candidates}
				item.ReviewPrompt = "Verify whether this deleted spec was intentionally merged elsewhere or dropped."
				items = append(items, item)
			}
			return items
		}
	}
	item := makeAdvisoryItem(checkTestRemoved, relPath, 1, filepath.Base(relPath), "spec file deleted")
	item.Evidence = map[string]any{"deleted_file": relPath, "renamed_candidates": candidates}
	item.ReviewPrompt = "Verify whether this deleted spec file was intentionally removed with user confirmation."
	return []rule.QAItem{item}
}

func extractAddedTSTestNames(baseSrc []byte, targetSrc []byte) []string {
	targetM := analyzeTSMetrics(targetSrc)
	baseSet := make(map[string]bool)
	if len(baseSrc) > 0 {
		for _, sp := range analyzeTSMetrics(baseSrc).specs {
			baseSet[sp.name] = true
		}
	}
	var added []string
	for _, sp := range targetM.specs {
		if !baseSet[sp.name] {
			added = append(added, sp.name)
		}
	}
	slices.Sort(added)
	return slices.Compact(added)
}

func analyzeChangedTSFile(in tsFileInput) []rule.QAItem {
	targetM := analyzeTSMetrics(in.targetSrc)
	var baseM tsSpecMetrics
	if in.hasBase {
		baseM = analyzeTSMetrics(in.baseSrc)
	}

	var items []rule.QAItem
	items = append(items, detectTSSuppressions(in, baseM, targetM)...)
	if gotestutil.IsTargetProductionTSFile(in.relPath) {
		items = append(items, detectTSDoubleCasts(in, baseM, targetM)...)
		return items
	}
	if !gotestutil.IsTargetTSTestFile(in.relPath) {
		return items
	}
	items = append(items, detectTSSpecSkips(in, baseM, targetM)...)
	if in.hasBase {
		items = append(items, detectTSSpecDiffSignals(in, baseM, targetM)...)
	}
	return items
}

func detectTSSuppressions(in tsFileInput, baseM tsSpecMetrics, targetM tsSpecMetrics) []rule.QAItem {
	if len(targetM.suppressHits) <= len(baseM.suppressHits) {
		return nil
	}
	var items []rule.QAItem
	for _, hit := range targetM.suppressHits {
		if !isLineInScope(hit.line, in.changedLines) {
			continue
		}
		item := makeAdvisoryItem(checkSuppressionAdded, in.relPath, hit.line, "<comment>", "newly added ESLint or TypeScript suppression comment")
		item.Evidence = map[string]any{"marker": hit.subject, "before_count": len(baseM.suppressHits), "after_count": len(targetM.suppressHits)}
		item.ReviewPrompt = "Verify that this suppression comment is justified and not hiding an avoidable type or lint defect."
		items = append(items, item)
	}
	return items
}

func detectTSDoubleCasts(in tsFileInput, baseM tsSpecMetrics, targetM tsSpecMetrics) []rule.QAItem {
	if len(targetM.doubleCastHits) <= len(baseM.doubleCastHits) {
		return nil
	}
	var items []rule.QAItem
	for _, ln := range targetM.doubleCastHits {
		if !isLineInScope(ln, in.changedLines) {
			continue
		}
		item := makeAdvisoryItem(checkTSDoubleCastAdded, in.relPath, ln, "<type-cast>", "newly added `as unknown as` double type assertion")
		item.Evidence = map[string]any{"before_count": len(baseM.doubleCastHits), "after_count": len(targetM.doubleCastHits)}
		item.ReviewPrompt = "Verify whether `as unknown as` can be replaced by proper typing or type guards."
		items = append(items, item)
	}
	return items
}

func detectTSSpecSkips(in tsFileInput, baseM tsSpecMetrics, targetM tsSpecMetrics) []rule.QAItem {
	if len(targetM.skipHits) <= len(baseM.skipHits) {
		return nil
	}
	var items []rule.QAItem
	for _, sk := range targetM.skipHits {
		if !isLineInScope(sk.line, in.changedLines) {
			continue
		}
		item := makeAdvisoryItem(checkTestSkipAdded, in.relPath, sk.line, sk.subject, fmt.Sprintf("newly added skipped spec (%s)", sk.subject))
		item.Evidence = map[string]any{"before_count": len(baseM.skipHits), "after_count": len(targetM.skipHits)}
		item.ReviewPrompt = "Verify that this skipped Jasmine spec is not masking a broken test."
		items = append(items, item)
	}
	return items
}

func detectTSSpecDiffSignals(in tsFileInput, baseM tsSpecMetrics, targetM tsSpecMetrics) []rule.QAItem {
	var items []rule.QAItem
	targetNames := make(map[string]bool, len(targetM.specs))
	for _, sp := range targetM.specs {
		targetNames[sp.name] = true
	}
	addedInFile := extractAddedTSTestNames(in.baseSrc, in.targetSrc)
	dir := filepath.ToSlash(filepath.Dir(in.relPath))
	candidates := mergeRenamedCandidates(addedInFile, in.addedByDir[dir])

	for _, sp := range baseM.specs {
		if !targetNames[sp.name] {
			item := makeAdvisoryItem(checkTestRemoved, in.relPath, sp.line, sp.name, fmt.Sprintf("spec %q removed", sp.name))
			item.Evidence = map[string]any{"test_name": sp.name, "renamed_candidates": candidates}
			item.ReviewPrompt = "Verify whether this removed spec was renamed/merged intentionally or dropped."
			items = append(items, item)
		}
	}
	if targetM.itCount < baseM.itCount {
		item := makeAdvisoryItem(checkTestCasesReduced, in.relPath, 1, filepath.Base(in.relPath), fmt.Sprintf("spec count reduced from %d to %d", baseM.itCount, targetM.itCount))
		item.Evidence = map[string]any{"before_count": baseM.itCount, "after_count": targetM.itCount}
		item.ReviewPrompt = "Confirm whether specs were consolidated equivalently or dropped."
		items = append(items, item)
	}
	if targetM.eqAssertCount < baseM.eqAssertCount {
		item := makeAdvisoryItem(checkTestAssertionWeakened, in.relPath, 1, filepath.Base(in.relPath), fmt.Sprintf("toEqual/toStrictEqual assertions reduced from %d to %d", baseM.eqAssertCount, targetM.eqAssertCount))
		item.Evidence = map[string]any{"before_eq_asserts": baseM.eqAssertCount, "after_eq_asserts": targetM.eqAssertCount}
		item.ReviewPrompt = "Verify that removing deep equality checks did not weaken spec assertions."
		items = append(items, item)
	} else if targetM.expectCount < baseM.expectCount {
		item := makeAdvisoryItem(checkTestAssertionWeakened, in.relPath, 1, filepath.Base(in.relPath), fmt.Sprintf("expect() assertions reduced from %d to %d", baseM.expectCount, targetM.expectCount))
		item.Evidence = map[string]any{"before_expect_count": baseM.expectCount, "after_expect_count": targetM.expectCount}
		item.ReviewPrompt = "Verify that reducing expect() assertions preserved test strength."
		items = append(items, item)
	}
	return items
}

func analyzeTSMetrics(src []byte) tsSpecMetrics {
	tokens := lexTypeScript(src)
	var out tsSpecMetrics
	var code []tsToken
	for _, tok := range tokens {
		if tok.kind == tsTokComment {
			if marker := matchTSSuppressionMarker(tok.text); marker != "" {
				out.suppressHits = append(out.suppressHits, lineSubject{line: tok.line, subject: marker})
			}
			continue
		}
		code = append(code, tok)
	}
	scanTSCodeTokens(code, &out)
	return out
}

func scanTSCodeTokens(code []tsToken, out *tsSpecMetrics) {
	for i := 0; i < len(code); i++ {
		if code[i].kind == tsTokIdent {
			recordTSIdentToken(code, i, out)
		}
	}
}

func recordTSIdentToken(code []tsToken, i int, out *tsSpecMetrics) {
	if recordTSSpecIdent(code, i, out) {
		return
	}
	recordTSAssertOrCastIdent(code, i, out)
}

func recordTSSpecIdent(code []tsToken, i int, out *tsSpecMetrics) bool {
	tok := code[i]
	switch tok.text {
	case "it":
		if hasCallStringArg(code, i) {
			out.specs = append(out.specs, tsSpecEntry{name: unquoteTSString(code[i+2].text), line: tok.line})
			out.itCount++
		} else if hasSkipCall(code, i) {
			out.skipHits = append(out.skipHits, lineSubject{line: tok.line, subject: "it.skip"})
		}
		return true
	case "xit", "xdescribe":
		if hasOpenParen(code, i) {
			out.skipHits = append(out.skipHits, lineSubject{line: tok.line, subject: tok.text})
		}
		return true
	case "describe":
		if hasSkipCall(code, i) {
			out.skipHits = append(out.skipHits, lineSubject{line: tok.line, subject: "describe.skip"})
		}
		return true
	default:
		return false
	}
}

func recordTSAssertOrCastIdent(code []tsToken, i int, out *tsSpecMetrics) {
	tok := code[i]
	switch tok.text {
	case "expect":
		if hasOpenParen(code, i) {
			out.expectCount++
		}
	case "toEqual", "toStrictEqual":
		if hasOpenParen(code, i) {
			out.eqAssertCount++
		}
	case "as":
		if i+2 < len(code) && code[i+1].text == "unknown" && code[i+2].text == "as" {
			out.doubleCastHits = append(out.doubleCastHits, tok.line)
		}
	}
}

func hasOpenParen(code []tsToken, i int) bool {
	return i+1 < len(code) && code[i+1].text == "("
}

func hasCallStringArg(code []tsToken, i int) bool {
	return i+2 < len(code) && code[i+1].text == "(" && code[i+2].kind == tsTokString
}

func hasSkipCall(code []tsToken, i int) bool {
	return i+3 < len(code) && code[i+1].text == "." && code[i+2].text == "skip" && code[i+3].text == "("
}

func matchTSSuppressionMarker(comment string) string {
	for _, rawLine := range strings.Split(comment, "\n") {
		body := stripCommentLineDelimiters(rawLine)
		if hasAllowDirectivePrefix(body) {
			return "CODEQUALITY_ALLOW"
		}
		if hasTSDirectivePrefix(body, "eslint-disable", "- \t") {
			return "eslint-disable"
		}
		if hasTSPragmaPrefix(body, "@ts-ignore") {
			return "@ts-ignore"
		}
		if hasTSPragmaPrefix(body, "@ts-expect-error") {
			return "@ts-expect-error"
		}
	}
	return ""
}

func hasTSDirectivePrefix(body string, marker string, allowedNext string) bool {
	after, ok := strings.CutPrefix(body, marker)
	if !ok {
		return false
	}
	return after == "" || strings.IndexByte(allowedNext, after[0]) >= 0
}

func hasTSPragmaPrefix(body string, marker string) bool {
	after, ok := strings.CutPrefix(body, marker)
	if !ok {
		return false
	}
	return after == "" || !isTSIdentPart(after[0])
}

func stripCommentLineDelimiters(rawLine string) string {
	trimmed := strings.TrimSpace(rawLine)
	if trimmed == "" {
		return ""
	}
	if after, ok := strings.CutPrefix(trimmed, "//"); ok {
		return strings.TrimSpace(strings.TrimLeft(after, "/"))
	}
	inner := strings.TrimPrefix(trimmed, "/*")
	inner = strings.TrimSuffix(inner, "*/")
	inner = strings.TrimLeft(strings.TrimSpace(inner), "*")
	return strings.TrimSpace(inner)
}

func hasAllowDirectivePrefix(body string) bool {
	rest, ok := strings.CutPrefix(body, "CODEQUALITY_ALLOW")
	if !ok {
		return false
	}
	rest = strings.TrimLeft(rest, " \t")
	return strings.HasPrefix(rest, "(") || strings.HasPrefix(rest, ":")
}

func mergeRenamedCandidates(sameFile []string, samePkg []string) []string {
	merged := make([]string, 0, len(sameFile)+len(samePkg))
	merged = append(merged, sameFile...)
	for _, c := range samePkg {
		if !slices.Contains(merged, c) {
			merged = append(merged, c)
		}
	}
	return merged
}

func isLineInScope(line int, changed map[int]struct{}) bool {
	if len(changed) == 0 {
		return true
	}
	_, ok := changed[line]
	return ok
}

func unquoteTSString(raw string) string {
	if len(raw) >= 2 {
		return raw[1 : len(raw)-1]
	}
	return raw
}

func lexTypeScript(src []byte) []tsToken {
	var tokens []tsToken
	line := 1
	for i := 0; i < len(src); {
		b := src[i]
		if b == '\n' {
			line++
			i++
			continue
		}
		if b == ' ' || b == '\t' || b == '\r' {
			i++
			continue
		}
		if tok, nextIdx, lines := tryReadTSCommentOrString(src, i, line); nextIdx > i {
			tokens = append(tokens, tok)
			line += lines
			i = nextIdx
			continue
		}
		if isTSIdentStart(b) {
			end := i + 1
			for end < len(src) && isTSIdentPart(src[end]) {
				end++
			}
			tokens = append(tokens, tsToken{kind: tsTokIdent, text: string(src[i:end]), line: line})
			i = end
			continue
		}
		tokens = append(tokens, tsToken{kind: tsTokPunct, text: string(b), line: line})
		i++
	}
	return tokens
}

func tryReadTSCommentOrString(src []byte, start int, line int) (tsToken, int, int) {
	b := src[start]
	if b == '/' && start+1 < len(src) {
		if src[start+1] == '/' {
			end := start + 2
			for end < len(src) && src[end] != '\n' {
				end++
			}
			return tsToken{kind: tsTokComment, text: string(src[start:end]), line: line}, end, 0
		}
		if src[start+1] == '*' {
			return scanTSBlockComment(src, start, line)
		}
	}
	if b == '\'' || b == '"' || b == '`' {
		end, lines := scanTSQuotedSpan(src, start, b)
		return tsToken{kind: tsTokString, text: string(src[start:end]), line: line}, end, lines
	}
	return tsToken{}, start, 0
}

func scanTSBlockComment(src []byte, start int, line int) (tsToken, int, int) {
	end := start + 2
	lines := 0
	for end < len(src) {
		if src[end] == '\n' {
			lines++
		}
		if src[end] == '*' && end+1 < len(src) && src[end+1] == '/' {
			end += 2
			break
		}
		end++
	}
	return tsToken{kind: tsTokComment, text: string(src[start:end]), line: line}, end, lines
}

func scanTSQuotedSpan(src []byte, start int, quote byte) (int, int) {
	end := start + 1
	lines := 0
	for end < len(src) {
		c := src[end]
		if c == '\n' {
			lines++
			if quote != '`' {
				return end, lines
			}
		}
		if c == '\\' {
			end, lines = advanceTSEscapedByte(src, end, lines)
			continue
		}
		end++
		if c == quote {
			break
		}
	}
	return end, lines
}

func advanceTSEscapedByte(src []byte, end int, lines int) (int, int) {
	if end+1 >= len(src) {
		return len(src), lines
	}
	if src[end+1] == '\n' {
		lines++
	}
	return end + 2, lines
}

func isTSIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' || b == '$'
}

func isTSIdentPart(b byte) bool {
	return isTSIdentStart(b) || (b >= '0' && b <= '9')
}
