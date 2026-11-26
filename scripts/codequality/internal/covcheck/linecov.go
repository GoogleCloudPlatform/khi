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

package covcheck

import (
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// LineState records whether a 1-based source line is executable, covered by tests, or excluded boilerplate.
type LineState struct {
	// Executable reports whether the line belongs to an executable Go statement.
	Executable bool
	// Covered reports whether tests executed the line.
	Covered bool
	// Boilerplate reports whether the line is excluded as non-domain boilerplate.
	Boilerplate bool
}

// FileAnalysis holds split raw lines and per-line coverage states for a Go source file.
type FileAnalysis struct {
	// RawLines contains 0-indexed raw lines of the source file.
	RawLines []string
	// CodeMask marks 1-based substantive executable lines.
	CodeMask []bool
	// States holds 1-based per-line coverage and boilerplate states.
	States []LineState
}

// AnalyzeFileLines maps coverprofile blocks onto 1-based source lines and marks AST boilerplate lines.
func AnalyzeFileLines(relPath string, src []byte, blocks []gotestutil.CoverBlock) FileAnalysis {
	rawLines := rule.SplitSourceLines(string(src))
	codeLines := classifyExecutableCodeLines(src, len(rawLines))
	states := mapBlocksToLines(rawLines, codeLines, blocks)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, src, 0)
	if err == nil {
		for ln := range rule.BoilerplateLineSet(fset, file) {
			states[ln].Boilerplate = true
		}
	}
	return FileAnalysis{
		RawLines: rawLines,
		CodeMask: codeLines,
		States:   states,
	}
}

// MeasurePatchTotals counts raw changed executable and covered lines on changedSet for dashboard display.
func MeasurePatchTotals(fa FileAnalysis, changedSet map[int]struct{}) (covered int, executable int) {
	for lineNum := 1; lineNum <= len(fa.RawLines); lineNum++ {
		if _, ok := changedSet[lineNum]; !ok {
			continue
		}
		st := fa.States[lineNum]
		if !st.Executable {
			continue
		}
		executable++
		if st.Covered {
			covered++
		}
	}
	return covered, executable
}

// MeasureFuncUncovered counts substantive uncovered lines on changedSet and builds UncoveredSpans inside fn.
func MeasureFuncUncovered(
	relFile string,
	fa FileAnalysis,
	fn astcheck.FuncMetrics,
	changedSet map[int]struct{},
) (int, []rule.UncoveredSpan) {
	var uncovered []int
	for ln := fn.StartLine; ln <= fn.EndLine && ln <= len(fa.RawLines); ln++ {
		if _, changed := changedSet[ln]; !changed {
			continue
		}
		st := fa.States[ln]
		if !st.Executable || st.Covered || st.Boilerplate {
			continue
		}
		uncovered = append(uncovered, ln)
	}
	spans := rule.BuildUncoveredSpans(relFile, fa.RawLines, fa.CodeMask, uncovered)
	return len(uncovered), spans
}

func classifyExecutableCodeLines(src []byte, numLines int) []bool {
	hasToken := make([]bool, numLines+1)
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var sc scanner.Scanner
	sc.Init(file, src, func(_ token.Position, _ string) {}, 0)

	for {
		pos, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		if isNonSubstantiveToken(tok, lit) {
			continue
		}
		markScannedTokenLines(file, pos, lit, hasToken)
	}
	return hasToken
}

func isNonSubstantiveToken(tok token.Token, lit string) bool {
	return (tok == token.SEMICOLON && lit == "\n") || tok == token.RBRACE || tok == token.RPAREN
}

func markScannedTokenLines(file *token.File, pos token.Pos, lit string, hasToken []bool) {
	startLine := file.Position(pos).Line
	endLine := startLine
	endPos := pos + token.Pos(len(lit))
	if len(lit) > 0 && int(endPos) <= file.Base()+file.Size() {
		endLine = file.Position(endPos).Line
	}
	for ln := startLine; ln <= endLine && ln < len(hasToken); ln++ {
		hasToken[ln] = true
	}
}

func mapBlocksToLines(rawLines []string, codeLines []bool, blocks []gotestutil.CoverBlock) []LineState {
	states := make([]LineState, len(rawLines)+1)
	earliest := computeEarliestStartCols(rawLines, blocks)
	for _, b := range blocks {
		effStart, effEnd := effectiveBlockSpan(rawLines, earliest, b)
		applyBlockCoverage(states, codeLines, effStart, effEnd, b.Count)
	}
	return states
}

func computeEarliestStartCols(rawLines []string, blocks []gotestutil.CoverBlock) []int {
	numLines := len(rawLines)
	earliest := make([]int, numLines+1)
	for _, b := range blocks {
		if b.StartLine >= 1 && b.StartLine <= numLines {
			if earliest[b.StartLine] == 0 || b.StartCol < earliest[b.StartLine] {
				earliest[b.StartLine] = b.StartCol
			}
		}
		if b.EndLine >= 1 && b.EndLine <= numLines && b.StartLine < b.EndLine {
			if !isOnlyClosingBracePrefix(rawLines[b.EndLine-1], b.EndCol) {
				earliest[b.EndLine] = 1
			}
		}
	}
	return earliest
}

func effectiveBlockSpan(rawLines []string, earliest []int, b gotestutil.CoverBlock) (int, int) {
	numLines := len(rawLines)
	effStart := b.StartLine
	effEnd := b.EndLine
	if b.EndLine > b.StartLine && b.EndLine >= 1 && b.EndLine <= numLines {
		if isOnlyClosingBracePrefix(rawLines[b.EndLine-1], b.EndCol) {
			effEnd = b.EndLine - 1
		}
	}
	if b.EndLine > b.StartLine && b.StartLine >= 1 && b.StartLine <= numLines {
		if earliest[b.StartLine] > 0 && earliest[b.StartLine] < b.StartCol &&
			isOnlyOpeningBraceSuffix(rawLines[b.StartLine-1], b.StartCol) {
			effStart = b.StartLine + 1
		}
	}
	return effStart, effEnd
}

func applyBlockCoverage(states []LineState, codeLines []bool, start int, end int, count int) {
	for ln := start; ln <= end; ln++ {
		if ln < 1 || ln >= len(states) || !codeLines[ln] {
			continue
		}
		if !states[ln].Executable {
			states[ln].Executable = true
			states[ln].Covered = count > 0
		} else if count == 0 {
			states[ln].Covered = false
		}
	}
}

func isOnlyClosingBracePrefix(line string, endCol int) bool {
	limit := min(endCol-1, len(line))
	if limit <= 0 {
		return true
	}
	prefix := strings.TrimSpace(line[:limit])
	return prefix == "" || prefix == "}"
}

func isOnlyOpeningBraceSuffix(line string, startCol int) bool {
	idx := startCol - 1
	if idx < 0 || idx >= len(line) {
		return false
	}
	suffix := line[idx:]
	if commentIdx := strings.Index(suffix, "//"); commentIdx >= 0 {
		suffix = suffix[:commentIdx]
	}
	return strings.TrimSpace(suffix) == "{"
}
