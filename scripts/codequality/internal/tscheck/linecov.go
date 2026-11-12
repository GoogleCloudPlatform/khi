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

package tscheck

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

// TSCoverageOptions configures TypeScript differential and overall coverage analysis.
type TSCoverageOptions struct {
	CovByFile          map[string]gotestutil.IstanbulFileCoverage
	ChangedLinesByFile map[string]map[int]struct{}
	ReadSource         func(relPath string) ([]byte, error)
	IncludeOverall     bool
}

// TSCoverageResult holds aggregate coverage summary metrics and per-file uncovered line findings.
type TSCoverageResult struct {
	Summary  rule.CoverageSummary
	Findings []rule.Finding
}

// AnalyzeTSCoverage maps Istanbul spans onto changed TypeScript lines and builds uncovered line findings.
func AnalyzeTSCoverage(opts TSCoverageOptions) (*TSCoverageResult, error) {
	res := &TSCoverageResult{
		Summary: rule.CoverageSummary{
			Enabled:    true,
			HasOverall: opts.IncludeOverall,
		},
	}
	if opts.IncludeOverall {
		accumulateOverallTSStatements(&res.Summary, opts.CovByFile)
	}

	files := sortedNonEmptyLineFiles(opts.ChangedLinesByFile)
	for _, relFile := range files {
		out, err := analyzeSingleTSFile(relFile, opts)
		if err != nil {
			return nil, err
		}
		res.Summary.PatchExecutable += out.execCount
		res.Summary.PatchCovered += out.covCount
		if out.finding != nil {
			res.Findings = append(res.Findings, *out.finding)
		}
	}
	return res, nil
}

func accumulateOverallTSStatements(sum *rule.CoverageSummary, covByFile map[string]gotestutil.IstanbulFileCoverage) {
	for _, fc := range covByFile {
		for _, st := range fc.Statements {
			if st.Skip {
				continue
			}
			sum.OverallStatements++
			if st.Count > 0 {
				sum.OverallCovered++
			}
		}
	}
}

type tsFileCovOutcome struct {
	finding   *rule.Finding
	execCount int
	covCount  int
}

func analyzeSingleTSFile(relFile string, opts TSCoverageOptions) (tsFileCovOutcome, error) {
	fc, ok := opts.CovByFile[relFile]
	if !ok {
		return tsFileCovOutcome{}, nil
	}
	src, err := opts.ReadSource(relFile)
	if err != nil {
		if os.IsNotExist(err) {
			return tsFileCovOutcome{}, nil
		}
		return tsFileCovOutcome{}, fmt.Errorf("failed to read %s: %w", relFile, err)
	}
	rawLines := rule.SplitSourceLines(string(src))
	codeLines := classifyExecutableTSLines(src, len(rawLines))
	states := mapIstanbulToLines(rawLines, codeLines, fc)

	changedSet := opts.ChangedLinesByFile[relFile]
	execCount, covCount, uncovered := collectChangedTSLineCounts(states, len(rawLines), changedSet)
	if len(uncovered) == 0 {
		return tsFileCovOutcome{execCount: execCount, covCount: covCount}, nil
	}
	spans := rule.BuildUncoveredSpans(relFile, rawLines, codeLines, uncovered)
	finding := &rule.Finding{
		Rule:           rule.RuleFuncUncoveredLines,
		Kind:           rule.ViolationHardLimit,
		File:           relFile,
		Line:           spans[0].StartLine,
		EndLine:        spans[len(spans)-1].EndLine,
		Subject:        "<file>",
		After:          len(uncovered),
		Limit:          0,
		Detail:         fmt.Sprintf("%d uncovered executable line(s) across %d span(s)", len(uncovered), len(spans)),
		UncoveredSpans: spans,
	}
	return tsFileCovOutcome{finding: finding, execCount: execCount, covCount: covCount}, nil
}

func collectChangedTSLineCounts(states []tsLineState, numLines int, changedSet map[int]struct{}) (int, int, []int) {
	execCount := 0
	covCount := 0
	var uncovered []int
	for lineNum := 1; lineNum <= numLines; lineNum++ {
		if _, changed := changedSet[lineNum]; !changed || !states[lineNum].executable {
			continue
		}
		execCount++
		if states[lineNum].covered {
			covCount++
		} else {
			uncovered = append(uncovered, lineNum)
		}
	}
	return execCount, covCount, uncovered
}

type tsLineState struct {
	executable bool
	covered    bool
}

type tsCandidateKind int

const (
	tsCandidateStatement tsCandidateKind = iota
	tsCandidateFnDecl
	tsCandidateFnBody
	tsCandidateBranchArm
)

type tsLineCandidate struct {
	kind          tsCandidateKind
	branchType    string
	effStart      int
	effEnd        int
	origStartLine int
	origStartCol  int
	origEndLine   int
	origEndCol    int
	count         int
}

func mapIstanbulToLines(rawLines []string, codeLines []bool, fc gotestutil.IstanbulFileCoverage) []tsLineState {
	numLines := len(rawLines)
	states := make([]tsLineState, numLines+1)
	if numLines == 0 {
		return states
	}
	earliestStmtCol := collectEarliestTSStmtCols(numLines, fc)
	candidates := collectTSCandidates(rawLines, earliestStmtCol, fc)
	for ln := 1; ln <= numLines; ln++ {
		if codeLines[ln] {
			states[ln] = evaluateTSLineState(ln, candidates)
		}
	}
	return states
}

func collectEarliestTSStmtCols(numLines int, fc gotestutil.IstanbulFileCoverage) []int {
	earliest := make([]int, numLines+1)
	for i := range earliest {
		earliest[i] = -1
	}
	record := func(line int, col int) {
		if line < 1 || line > numLines {
			return
		}
		if earliest[line] < 0 {
			earliest[line] = col
		} else {
			earliest[line] = min(earliest[line], col)
		}
	}
	for _, st := range fc.Statements {
		if !st.Skip {
			record(st.StartLine, st.StartCol)
		}
	}
	for _, fn := range fc.Functions {
		if !fn.Skip {
			record(fn.Decl.StartLine, fn.Decl.StartCol)
		}
	}
	return earliest
}

func collectTSCandidates(rawLines []string, earliestStmtCol []int, fc gotestutil.IstanbulFileCoverage) []tsLineCandidate {
	var candidates []tsLineCandidate
	add := func(kind tsCandidateKind, branchType string, sp gotestutil.IstanbulSpan) {
		if cand, ok := buildTSCandidate(rawLines, earliestStmtCol, kind, branchType, sp); ok {
			candidates = append(candidates, cand)
		}
	}
	for _, st := range fc.Statements {
		add(tsCandidateStatement, "", st)
	}
	for _, fn := range fc.Functions {
		if !fn.Skip {
			add(tsCandidateFnDecl, "", fn.Decl)
			add(tsCandidateFnBody, "", fn.Body)
		}
	}
	for _, br := range fc.Branches {
		if br.BranchType != "if" {
			add(tsCandidateBranchArm, br.BranchType, br.Arm)
		}
	}
	return candidates
}

func buildTSCandidate(
	rawLines []string,
	earliestStmtCol []int,
	kind tsCandidateKind,
	branchType string,
	sp gotestutil.IstanbulSpan,
) (tsLineCandidate, bool) {
	if sp.Skip {
		return tsLineCandidate{}, false
	}
	numLines := len(rawLines)
	effStart := sp.StartLine
	effEnd := sp.EndLine
	resolvedEndCol := resolveTSEndCol(rawLines, sp.EndLine, sp.EndCol)
	if sp.EndLine > sp.StartLine {
		if sp.EndLine <= numLines && isOnlyTSClosingChars(rawLines[sp.EndLine-1][:resolvedEndCol]) {
			effEnd = sp.EndLine - 1
		}
		if sp.StartLine <= numLines &&
			earliestStmtCol[sp.StartLine] >= 0 && earliestStmtCol[sp.StartLine] < sp.StartCol &&
			isOnlyTSOpeningSuffix(rawLines[sp.StartLine-1], sp.StartCol) {
			effStart = sp.StartLine + 1
		}
	}
	if effStart > effEnd {
		return tsLineCandidate{}, false
	}
	return tsLineCandidate{
		kind:          kind,
		branchType:    branchType,
		effStart:      effStart,
		effEnd:        effEnd,
		origStartLine: sp.StartLine,
		origStartCol:  sp.StartCol,
		origEndLine:   sp.EndLine,
		origEndCol:    resolvedEndCol,
		count:         sp.Count,
	}, true
}

func evaluateTSLineState(line int, candidates []tsLineCandidate) tsLineState {
	earliestCoveredCol := findEarliestCoveredStmtCol(line, candidates)
	var best *tsLineCandidate
	hasUncoveredOverride := false

	for i := range candidates {
		c := &candidates[i]
		if line < c.effStart || line > c.effEnd {
			continue
		}
		if shouldSkipCoveredLineSubspan(line, c, earliestCoveredCol) {
			continue
		}
		if best == nil || isNarrowerTSCandidate(c, best) {
			best = c
		}
		if isSingleLineUncoveredOverride(line, c) {
			hasUncoveredOverride = true
		}
	}
	if best == nil {
		return tsLineState{}
	}
	return tsLineState{
		executable: true,
		covered:    best.count > 0 && !hasUncoveredOverride,
	}
}

func findEarliestCoveredStmtCol(line int, candidates []tsLineCandidate) int {
	earliest := -1
	for i := range candidates {
		c := &candidates[i]
		if c.kind == tsCandidateStatement && c.origStartLine == line && c.count > 0 {
			if earliest < 0 {
				earliest = c.origStartCol
			} else {
				earliest = min(earliest, c.origStartCol)
			}
		}
	}
	return earliest
}

func shouldSkipCoveredLineSubspan(line int, c *tsLineCandidate, earliestCoveredCol int) bool {
	if c.origStartLine != line || c.origEndLine != line || earliestCoveredCol < 0 {
		return false
	}
	if c.kind == tsCandidateBranchArm && (c.branchType == "binary-expr" || c.branchType == "default-arg") {
		return true
	}
	return c.kind == tsCandidateFnDecl && earliestCoveredCol < c.origStartCol
}

func isSingleLineUncoveredOverride(line int, c *tsLineCandidate) bool {
	if c.count != 0 || c.effStart != line || c.effEnd != line {
		return false
	}
	if c.kind == tsCandidateStatement {
		return true
	}
	return c.kind == tsCandidateFnBody && c.origStartLine == line && c.origEndLine == line
}

func isNarrowerTSCandidate(a *tsLineCandidate, b *tsLineCandidate) bool {
	lineSpanA := a.origEndLine - a.origStartLine
	lineSpanB := b.origEndLine - b.origStartLine
	if lineSpanA < lineSpanB {
		return true
	}
	if lineSpanA > lineSpanB {
		return false
	}
	if lineSpanA == 0 {
		colSpanA := a.origEndCol - a.origStartCol
		colSpanB := b.origEndCol - b.origStartCol
		if colSpanA < colSpanB {
			return true
		}
		if colSpanA > colSpanB {
			return false
		}
	}
	if (a.count == 0) != (b.count == 0) {
		return a.count == 0
	}
	return a.kind > b.kind
}

func resolveTSEndCol(rawLines []string, endLine int, endCol int) int {
	if endLine > len(rawLines) {
		return max(endCol, 0)
	}
	lineLen := len(rawLines[endLine-1])
	if endCol < 0 {
		return lineLen
	}
	return min(endCol, lineLen)
}

func isOnlyTSOpeningSuffix(line string, startCol int) bool {
	if startCol < 0 {
		return false
	}
	suffix, _, _ := strings.Cut(line[min(startCol, len(line)):], "//")
	trimmed := strings.TrimSpace(suffix)
	return trimmed == "{" || trimmed == "=> {"
}

func isOnlyTSClosingChars(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\r', '}', ')', ']', ';', ',':
		default:
			return false
		}
	}
	return true
}

func sortedNonEmptyLineFiles(m map[string]map[int]struct{}) []string {
	var files []string
	for f, lines := range m {
		if len(lines) > 0 {
			files = append(files, f)
		}
	}
	slices.Sort(files)
	return files
}
