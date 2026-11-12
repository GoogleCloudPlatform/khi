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
	"bytes"
	"context"
	"fmt"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

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

// CoverageTestPlan describes which Go packages to execute for differential line coverage.
type CoverageTestPlan struct {
	Skip     bool
	Packages []string
}

// PlanCoverageExecution builds a minimal or full Go test execution plan from changed production Go files.
func PlanCoverageExecution(repoRoot string, changedFiles []string, full bool) CoverageTestPlan {
	if full {
		return CoverageTestPlan{
			Packages: []string{"./..."},
		}
	}
	if len(changedFiles) == 0 {
		return CoverageTestPlan{Skip: true}
	}

	pkgSet := make(map[string]struct{})
	for _, relFile := range changedFiles {
		cleanFile := strings.TrimPrefix(filepath.ToSlash(relFile), "./")
		if !gotestutil.IsTargetGoFile(cleanFile) {
			continue
		}
		relDir := path.Dir(cleanFile)
		absDir := filepath.Join(repoRoot, filepath.FromSlash(relDir))
		if !dirHasGoFiles(absDir) {
			continue
		}

		pkgSpec := "./" + relDir
		if relDir == "." {
			pkgSpec = "."
		}
		pkgSet[pkgSpec] = struct{}{}
	}

	if len(pkgSet) == 0 {
		return CoverageTestPlan{Skip: true}
	}

	out := make([]string, 0, len(pkgSet))
	for k := range pkgSet {
		out = append(out, k)
	}
	slices.Sort(out)
	return CoverageTestPlan{
		Packages: out,
	}
}

// RunGoTestCoverage executes go test according to plan and writes the coverage profile to profileOut.
func RunGoTestCoverage(ctx context.Context, repoRoot string, plan CoverageTestPlan, profileOut string, errOut io.Writer) error {
	if plan.Skip {
		return nil
	}

	args := make([]string, 0, 2+len(plan.Packages))
	args = append(args, "test", "-coverprofile="+profileOut)
	args = append(args, plan.Packages...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "KHI_SKIP_CLOUD_LOGGING=true")

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	if err := cmd.Run(); err != nil {
		if errOut != nil && combined.Len() > 0 {
			_, _ = io.Copy(errOut, &combined)
		}
		return fmt.Errorf("go test failed (%s): %w", strings.Join(args, " "), err)
	}
	return nil
}

func dirHasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

// AnalyzeCoverage computes differential and overall coverage metrics from parsed coverprofile blocks and diff lines.
func AnalyzeCoverage(
	baseLabel string,
	blocksByFile map[string][]gotestutil.CoverBlock,
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
	slices.Sort(files)

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

		fileReport := analyzeFileCoverage(relFile, src, blocksByFile[relFile], changedSet)
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

func analyzeFileCoverage(
	relFile string,
	src []byte,
	blocks []gotestutil.CoverBlock,
	changedSet map[int]struct{},
) FileCoverageReport {
	rawLines := splitSourceLines(string(src))
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

	lineStates := mapBlocksToLines(rawLines, codeLines, blocks)

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

func splitSourceLines(content string) []string {
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

func mapBlocksToLines(rawLines []string, codeLines []bool, blocks []gotestutil.CoverBlock) []lineCoverageState {
	numLines := len(rawLines)
	states := make([]lineCoverageState, numLines+1)

	earliestStartCol := make([]int, numLines+1)
	for _, b := range blocks {
		if b.StartLine >= 1 && b.StartLine <= numLines {
			if earliestStartCol[b.StartLine] == 0 || b.StartCol < earliestStartCol[b.StartLine] {
				earliestStartCol[b.StartLine] = b.StartCol
			}
		}
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
