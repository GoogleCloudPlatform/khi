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
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// CoverBlock represents a single statement block entry from a Go coverprofile.
type CoverBlock struct {
	StartLine     int
	StartCol      int
	EndLine       int
	EndCol        int
	NumStatements int
	Count         int
}

// SourceLine holds a 1-based line number and its raw source text.
type SourceLine struct {
	Number int
	Text   string
}

// LineRange represents an inclusive 1-based line interval.
type LineRange struct {
	StartLine int
	EndLine   int
}

// UncoveredSpan represents a contiguous span of changed executable lines that were not covered by tests.
type UncoveredSpan struct {
	File            string
	StartLine       int
	EndLine         int
	ExecutableLines int
	Lines           []SourceLine
}

// FileCoverageReport holds differential and file-level coverage metrics for a single changed file.
type FileCoverageReport struct {
	File                   string
	ChangedExecutableLines int
	ChangedCoveredLines    int
	TotalStatements        int
	CoveredStatements      int
	UncoveredRanges        []LineRange
	UncoveredSpans         []UncoveredSpan
}

// PatchPercent returns the percentage of changed executable lines covered in this file.
func (f *FileCoverageReport) PatchPercent() float64 {
	if f.ChangedExecutableLines == 0 {
		return 100.0
	}
	return float64(f.ChangedCoveredLines) * 100.0 / float64(f.ChangedExecutableLines)
}

// FilePercent returns the statement coverage percentage across the entire file.
func (f *FileCoverageReport) FilePercent() float64 {
	if f.TotalStatements == 0 {
		return 0.0
	}
	return float64(f.CoveredStatements) * 100.0 / float64(f.TotalStatements)
}

// CoverageReport aggregates differential and overall coverage results.
type CoverageReport struct {
	BaseLabel            string
	PatchExecutableLines int
	PatchCoveredLines    int
	OverallStatements    int
	OverallCovered       int
	HasOverallCoverage   bool
	Files                []FileCoverageReport
}

// PatchPercent returns the overall percentage of changed executable lines covered across all changed files.
func (r *CoverageReport) PatchPercent() float64 {
	if r.PatchExecutableLines == 0 {
		return 100.0
	}
	return float64(r.PatchCoveredLines) * 100.0 / float64(r.PatchExecutableLines)
}

// OverallPercent returns the total statement coverage percentage across all measured production packages.
func (r *CoverageReport) OverallPercent() float64 {
	if r.OverallStatements == 0 {
		return 0.0
	}
	return float64(r.OverallCovered) * 100.0 / float64(r.OverallStatements)
}

// SourceReader loads raw file bytes for a repository-relative slash path.
type SourceReader func(relPath string) ([]byte, error)

// ReadModulePath extracts the Go module path from a go.mod file.
func ReadModulePath(goModPath string) (string, error) {
	f, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %w", goModPath, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("failed reading %s: %w", goModPath, err)
	}
	return "", fmt.Errorf("module directive not found in %s", goModPath)
}

type blockKey struct {
	file      string
	startLine int
	startCol  int
	endLine   int
	endCol    int
	numStmt   int
}

// ParseCoverProfile parses a Go coverprofile stream, merging duplicate block entries and filtering out non-target files.
func ParseCoverProfile(r io.Reader, modulePath string) (map[string][]CoverBlock, error) {
	merged := make(map[blockKey]int)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	modulePrefix := strings.TrimSuffix(modulePath, "/") + "/"
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		colonIdx := strings.LastIndexByte(line, ':')
		if colonIdx < 0 {
			return nil, fmt.Errorf("invalid coverprofile line %d: missing ':' in %q", lineNum, line)
		}
		rawFile := line[:colonIdx]
		rest := line[colonIdx+1:]

		var relFile string
		switch {
		case strings.HasPrefix(rawFile, modulePrefix):
			relFile = strings.TrimPrefix(rawFile, modulePrefix)
		default:
			relFile = strings.TrimPrefix(filepath.ToSlash(rawFile), "./")
		}
		if !IsTargetProductionGoFile(relFile) {
			continue
		}

		fields := strings.Fields(rest)
		if len(fields) != 3 {
			return nil, fmt.Errorf("invalid coverprofile line %d: unexpected fields in %q", lineNum, line)
		}
		startLine, startCol, endLine, endCol, err := parseBlockCoords(fields[0])
		if err != nil {
			return nil, fmt.Errorf("invalid coverprofile line %d: %w", lineNum, err)
		}
		numStmt, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("invalid statement count on line %d: %w", lineNum, err)
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("invalid execution count on line %d: %w", lineNum, err)
		}

		key := blockKey{
			file:      relFile,
			startLine: startLine,
			startCol:  startCol,
			endLine:   endLine,
			endCol:    endCol,
			numStmt:   numStmt,
		}
		merged[key] += count
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("failed scanning coverprofile: %w", err)
	}

	byFile := make(map[string][]CoverBlock)
	for k, count := range merged {
		byFile[k.file] = append(byFile[k.file], CoverBlock{
			StartLine:     k.startLine,
			StartCol:      k.startCol,
			EndLine:       k.endLine,
			EndCol:        k.endCol,
			NumStatements: k.numStmt,
			Count:         count,
		})
	}
	for file := range byFile {
		blocks := byFile[file]
		sort.Slice(blocks, func(i, j int) bool {
			if blocks[i].StartLine != blocks[j].StartLine {
				return blocks[i].StartLine < blocks[j].StartLine
			}
			if blocks[i].StartCol != blocks[j].StartCol {
				return blocks[i].StartCol < blocks[j].StartCol
			}
			if blocks[i].EndLine != blocks[j].EndLine {
				return blocks[i].EndLine < blocks[j].EndLine
			}
			return blocks[i].EndCol < blocks[j].EndCol
		})
	}
	return byFile, nil
}

func parseBlockCoords(spec string) (int, int, int, int, error) {
	parts := strings.Split(spec, ",")
	if len(parts) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("expected start,end coordinates in %q", spec)
	}
	sLine, sCol, err := parseLineCol(parts[0])
	if err != nil {
		return 0, 0, 0, 0, err
	}
	eLine, eCol, err := parseLineCol(parts[1])
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return sLine, sCol, eLine, eCol, nil
}

func parseLineCol(s string) (int, int, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected line.col in %q", s)
	}
	line, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	col, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return line, col, nil
}

// AnalyzeCoverage computes differential and overall coverage metrics from parsed coverprofile blocks and diff lines.
func AnalyzeCoverage(
	baseLabel string,
	blocksByFile map[string][]CoverBlock,
	changedLinesByFile map[string]map[int]struct{},
	readSource SourceReader,
	includeOverall bool,
) (*CoverageReport, error) {
	report := &CoverageReport{
		BaseLabel:          baseLabel,
		HasOverallCoverage: includeOverall,
	}

	if includeOverall {
		for _, blocks := range blocksByFile {
			for _, b := range blocks {
				report.OverallStatements += b.NumStatements
				if b.Count > 0 {
					report.OverallCovered += b.NumStatements
				}
			}
		}
	}

	files := make([]string, 0, len(changedLinesByFile))
	for f := range changedLinesByFile {
		files = append(files, f)
	}
	sort.Strings(files)

	for _, relFile := range files {
		changedSet := changedLinesByFile[relFile]
		if len(changedSet) == 0 {
			continue
		}
		src, err := readSource(relFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read %s: %w", relFile, err)
		}

		fileReport := analyzeFile(relFile, src, blocksByFile[relFile], changedSet)
		if fileReport.ChangedExecutableLines == 0 {
			continue
		}
		report.PatchExecutableLines += fileReport.ChangedExecutableLines
		report.PatchCoveredLines += fileReport.ChangedCoveredLines
		report.Files = append(report.Files, fileReport)
	}

	return report, nil
}

type lineCoverageState struct {
	executable bool
	covered    bool
}

func analyzeFile(
	relFile string,
	src []byte,
	blocks []CoverBlock,
	changedSet map[int]struct{},
) FileCoverageReport {
	rawLines := splitLines(string(src))
	codeLines := classifyExecutableCodeLines(src, len(rawLines))

	fileReport := FileCoverageReport{
		File: relFile,
	}
	for _, b := range blocks {
		fileReport.TotalStatements += b.NumStatements
		if b.Count > 0 {
			fileReport.CoveredStatements += b.NumStatements
		}
	}

	var lineStates []lineCoverageState
	if len(blocks) == 0 {
		lineStates = fallbackUncoveredFuncLines(src, codeLines, len(rawLines))
	} else {
		lineStates = mapBlocksToLines(rawLines, codeLines, blocks)
	}

	var uncoveredLines []int
	for lineNum := 1; lineNum <= len(rawLines); lineNum++ {
		if _, changed := changedSet[lineNum]; !changed {
			continue
		}
		st := lineStates[lineNum]
		if !st.executable {
			continue
		}
		fileReport.ChangedExecutableLines++
		if st.covered {
			fileReport.ChangedCoveredLines++
		} else {
			uncoveredLines = append(uncoveredLines, lineNum)
		}
	}

	fileReport.UncoveredRanges = buildLineRanges(uncoveredLines, codeLines)
	fileReport.UncoveredSpans = buildUncoveredSpans(relFile, rawLines, fileReport.UncoveredRanges, uncoveredLines)
	return fileReport
}

func splitLines(content string) []string {
	trimmed := strings.TrimSuffix(content, "\n")
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	return lines
}

// classifyExecutableCodeLines returns a 1-indexed boolean slice indicating lines that contain non-comment, non-closing-delimiter Go tokens.
func classifyExecutableCodeLines(src []byte, numLines int) []bool {
	hasSubstantiveToken := make([]bool, numLines+1)
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))

	var sc scanner.Scanner
	sc.Init(file, src, func(_ token.Position, _ string) {}, 0)

	for {
		pos, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.SEMICOLON && lit == "\n" {
			continue
		}
		if tok == token.RBRACE || tok == token.RPAREN {
			continue
		}
		startLine := file.Position(pos).Line
		endPos := pos + token.Pos(len(lit))
		endLine := startLine
		if len(lit) > 0 && int(endPos) <= file.Base()+file.Size() {
			endLine = file.Position(endPos).Line
		}
		for l := startLine; l <= endLine && l <= numLines; l++ {
			if l >= 1 {
				hasSubstantiveToken[l] = true
			}
		}
	}
	return hasSubstantiveToken
}

func mapBlocksToLines(rawLines []string, codeLines []bool, blocks []CoverBlock) []lineCoverageState {
	numLines := len(rawLines)
	states := make([]lineCoverageState, numLines+1)

	// Precompute whether each line has a block starting before a trailing '{' block on the same line (e.g., "if cond {").
	earliestStartCol := make([]int, numLines+1)
	for _, b := range blocks {
		if b.StartLine >= 1 && b.StartLine <= numLines {
			if earliestStartCol[b.StartLine] == 0 || b.StartCol < earliestStartCol[b.StartLine] {
				earliestStartCol[b.StartLine] = b.StartCol
			}
		}
		// Outer blocks that started on an earlier line and end on StartLine before StartCol also cover the condition header.
		if b.EndLine >= 1 && b.EndLine <= numLines && b.StartLine < b.EndLine {
			if !isOnlyClosingBracePrefix(rawLines[b.EndLine-1], b.EndCol) {
				earliestStartCol[b.EndLine] = 1
			}
		}
	}

	for _, b := range blocks {
		effStart := b.StartLine
		effEnd := b.EndLine

		if b.EndLine > b.StartLine && b.EndLine >= 1 && b.EndLine <= numLines {
			if isOnlyClosingBracePrefix(rawLines[b.EndLine-1], b.EndCol) {
				effEnd = b.EndLine - 1
			}
		}
		if b.EndLine > b.StartLine && b.StartLine >= 1 && b.StartLine <= numLines {
			if earliestStartCol[b.StartLine] > 0 && earliestStartCol[b.StartLine] < b.StartCol &&
				isOnlyOpeningBraceSuffix(rawLines[b.StartLine-1], b.StartCol) {
				effStart = b.StartLine + 1
			}
		}

		for l := effStart; l <= effEnd; l++ {
			if l < 1 || l > numLines || !codeLines[l] {
				continue
			}
			if !states[l].executable {
				states[l].executable = true
				states[l].covered = b.Count > 0
			} else if b.Count == 0 {
				// Any uncovered statement block on the same line (e.g., single-line "if err != nil { return err }") marks the line uncovered.
				states[l].covered = false
			}
		}
	}
	return states
}

func isOnlyClosingBracePrefix(line string, endCol int) bool {
	limit := endCol - 1
	if limit > len(line) {
		limit = len(line)
	}
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

func buildLineRanges(uncoveredLines []int, codeLines []bool) []LineRange {
	if len(uncoveredLines) == 0 {
		return nil
	}
	var ranges []LineRange
	cur := LineRange{StartLine: uncoveredLines[0], EndLine: uncoveredLines[0]}
	for _, line := range uncoveredLines[1:] {
		if canMergeUncoveredLines(cur.EndLine, line, codeLines) {
			cur.EndLine = line
			continue
		}
		ranges = append(ranges, cur)
		cur = LineRange{StartLine: line, EndLine: line}
	}
	ranges = append(ranges, cur)
	return ranges
}

func canMergeUncoveredLines(prevLine int, nextLine int, codeLines []bool) bool {
	if nextLine == prevLine+1 {
		return true
	}
	// Merge across at most 2 intervening non-executable lines (comments, blank lines, or closing braces) inside the same block.
	if nextLine-prevLine > 3 {
		return false
	}
	for l := prevLine + 1; l < nextLine; l++ {
		if l < len(codeLines) && codeLines[l] {
			return false
		}
	}
	return true
}

func buildUncoveredSpans(relFile string, rawLines []string, ranges []LineRange, uncoveredLines []int) []UncoveredSpan {
	if len(ranges) == 0 {
		return nil
	}
	uncoveredSet := make(map[int]struct{}, len(uncoveredLines))
	for _, lineNum := range uncoveredLines {
		uncoveredSet[lineNum] = struct{}{}
	}

	spans := make([]UncoveredSpan, 0, len(ranges))
	for _, rg := range ranges {
		snippetStart := rg.StartLine
		// Include the preceding block header line (e.g., "if cond {") as context when it opens the uncovered block.
		if snippetStart > 1 && snippetStart-1 <= len(rawLines) {
			prevTrimmed := strings.TrimSpace(rawLines[snippetStart-2])
			if strings.HasSuffix(prevTrimmed, "{") && !strings.HasPrefix(prevTrimmed, "//") {
				snippetStart--
			}
		}

		var lines []SourceLine
		execCount := 0
		for l := snippetStart; l <= rg.EndLine && l <= len(rawLines); l++ {
			text := rawLines[l-1]
			lines = append(lines, SourceLine{
				Number: l,
				Text:   text,
			})
			if l >= rg.StartLine {
				if _, ok := uncoveredSet[l]; ok {
					execCount++
				}
			}
		}
		spans = append(spans, UncoveredSpan{
			File:            relFile,
			StartLine:       rg.StartLine,
			EndLine:         rg.EndLine,
			ExecutableLines: execCount,
			Lines:           lines,
		})
	}
	return spans
}

// fallbackUncoveredFuncLines marks substantive lines inside function and closure bodies as uncovered when no coverprofile blocks exist for a file.
func fallbackUncoveredFuncLines(src []byte, codeLines []bool, numLines int) []lineCoverageState {
	states := make([]lineCoverageState, numLines+1)
	fset := token.NewFileSet()
	fileAST, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return states
	}
	ast.Inspect(fileAST, func(n ast.Node) bool {
		fnBody := funcBodyBlock(n)
		if fnBody == nil || len(fnBody.List) == 0 {
			return true
		}
		for _, stmt := range fnBody.List {
			startLine := fset.Position(stmt.Pos()).Line
			endLine := fset.Position(stmt.End()).Line
			for l := startLine; l <= endLine && l <= numLines; l++ {
				if l >= 1 && codeLines[l] {
					states[l].executable = true
					states[l].covered = false
				}
			}
		}
		return true
	})
	return states
}

func funcBodyBlock(n ast.Node) *ast.BlockStmt {
	switch fn := n.(type) {
	case *ast.FuncDecl:
		return fn.Body
	case *ast.FuncLit:
		return fn.Body
	default:
		return nil
	}
}
