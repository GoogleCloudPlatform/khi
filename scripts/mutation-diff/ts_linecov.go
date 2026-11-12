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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

// RunTSTestCoverage executes Angular Karma tests with coverage output written to coverageDir.
func RunTSTestCoverage(ctx context.Context, repoRoot string, plan gotestutil.TSTestPlan, coverageDir string, errOut io.Writer) error {
	if plan.SkipKarma {
		return nil
	}

	port, err := gotestutil.PickKarmaPort(ctx)
	if err != nil {
		return err
	}

	ngArgs := make([]string, 0, 6+len(plan.IncludeSpecs))
	ngArgs = append(ngArgs,
		"ng", "test",
		"--code-coverage",
		"--browsers", "ChromeHeadlessNoSandbox",
		"--watch=false",
		"--progress=false",
	)
	for _, spec := range plan.IncludeSpecs {
		ngArgs = append(ngArgs, "--include="+spec)
	}

	var cmd *exec.Cmd
	if gotestutil.IsContainerExecutionEnabled() {
		cmd, err = buildContainerKarmaCoverageCommand(ctx, repoRoot, coverageDir, port, ngArgs)
		if err != nil {
			return err
		}
	} else {
		cmd = exec.CommandContext(ctx, "npx", ngArgs...)
		cmd.Dir = filepath.Join(repoRoot, "web")
		cmd.Env = append(os.Environ(),
			"KARMA_PORT="+port,
			"KARMA_COVERAGE_DIR="+coverageDir,
		)
	}

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	if err := cmd.Run(); err != nil {
		if errOut != nil && combined.Len() > 0 {
			_, _ = io.Copy(errOut, &combined)
		}
		return fmt.Errorf("ng test failed (npx %s): %w", strings.Join(ngArgs, " "), err)
	}
	return nil
}

func buildContainerKarmaCoverageCommand(
	ctx context.Context,
	repoRoot string,
	coverageDir string,
	port string,
	ngArgs []string,
) (*exec.Cmd, error) {
	containerCmd, builderImage, err := gotestutil.ResolveContainerCommand()
	if err != nil {
		return nil, err
	}

	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	absCov, err := filepath.Abs(coverageDir)
	if err != nil {
		return nil, err
	}

	args := []string{
		"run", "--rm",
		"--shm-size=1g",
		fmt.Sprintf("-u=%d:%d", os.Getuid(), os.Getgid()),
		fmt.Sprintf("-v=%s:/workspace", absRepo),
		fmt.Sprintf("-v=%s:%s", absCov, absCov),
		"-w=/workspace/web",
		"-e=HOME=/tmp",
		"-e=KARMA_PORT=" + port,
		"-e=KARMA_COVERAGE_DIR=" + absCov,
		builderImage,
		"npx",
	}
	args = append(args, ngArgs...)

	cmd := exec.CommandContext(ctx, containerCmd, args...)
	cmd.Dir = absRepo
	return cmd, nil
}

// SupplementMissingTSCoverage populates zero-hit Istanbul coverage entries for changed production TypeScript files
// that were not imported by any executed Karma spec bundle (while skipping pure type/interface-only files).
func SupplementMissingTSCoverage(
	ctx context.Context,
	repoRoot string,
	changedTSFiles []string,
	covByFile map[string]gotestutil.IstanbulFileCoverage,
) error {
	var missing []string
	for _, rawFile := range changedTSFiles {
		clean := strings.TrimPrefix(filepath.ToSlash(rawFile), "./")
		if !gotestutil.IsTargetProductionTSFile(clean) {
			continue
		}
		if _, exists := covByFile[clean]; exists {
			continue
		}
		absFile := filepath.Join(repoRoot, filepath.FromSlash(clean))
		if info, err := os.Stat(absFile); err != nil || info.IsDir() {
			continue
		}
		missing = append(missing, clean)
	}
	if len(missing) == 0 {
		return nil
	}

	scriptPath, err := resolveTSHelperScriptPath(repoRoot, "zero_hit_ts.js")
	if err != nil {
		return err
	}

	args := make([]string, 0, 2+len(missing))
	args = append(args, scriptPath, repoRoot)
	args = append(args, missing...)

	outBytes, err := gotestutil.RunCommandOutput(ctx, repoRoot, "node", args...)
	if err != nil {
		return fmt.Errorf("failed to generate zero-hit TypeScript coverage: %w", err)
	}

	supplemented, err := gotestutil.ParseIstanbulCoverageJSON(bytes.NewReader(outBytes), repoRoot)
	if err != nil {
		return err
	}
	for file, fc := range supplemented {
		covByFile[file] = fc
	}
	return nil
}

// AnalyzeTSCoverage computes TypeScript differential and overall coverage metrics from Istanbul coverage entries and diff lines.
func AnalyzeTSCoverage(
	baseLabel string,
	covByFile map[string]gotestutil.IstanbulFileCoverage,
	changedLinesByFile map[string]map[int]struct{},
	readSource SourceReader,
	includeOverall bool,
) (*CoverageReport, error) {
	report := &CoverageReport{
		BaseLabel:          baseLabel,
		HasOverallCoverage: includeOverall,
	}

	if includeOverall {
		for _, fc := range covByFile {
			for _, st := range fc.Statements {
				if st.Skip {
					continue
				}
				report.OverallStatements++
				if st.Count > 0 {
					report.OverallCovered++
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

		fc, ok := covByFile[relFile]
		if !ok {
			continue
		}

		fileReport := analyzeTSFileCoverage(relFile, src, fc, changedSet)
		if fileReport.ChangedExecutableLines == 0 {
			continue
		}
		report.PatchExecutableLines += fileReport.ChangedExecutableLines
		report.PatchCoveredLines += fileReport.ChangedCoveredLines
		report.Files = append(report.Files, fileReport)
	}

	return report, nil
}

func analyzeTSFileCoverage(
	relFile string,
	src []byte,
	fc gotestutil.IstanbulFileCoverage,
	changedSet map[int]struct{},
) FileCoverageReport {
	rawLines := splitSourceLines(string(src))
	codeLines := classifyExecutableTSLines(src, len(rawLines))

	fileReport := FileCoverageReport{
		File: relFile,
	}
	for _, st := range fc.Statements {
		if st.Skip {
			continue
		}
		fileReport.TotalStatements++
		if st.Count > 0 {
			fileReport.CoveredStatements++
		}
	}

	lineStates := mapIstanbulToLines(rawLines, codeLines, fc)

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

func mapIstanbulToLines(rawLines []string, codeLines []bool, fc gotestutil.IstanbulFileCoverage) []lineCoverageState {
	numLines := len(rawLines)
	states := make([]lineCoverageState, numLines+1)
	if numLines == 0 {
		return states
	}

	// Track the earliest 0-based start column of any statement or function declaration per line (-1 = none).
	earliestStmtCol := make([]int, numLines+1)
	for i := range earliestStmtCol {
		earliestStmtCol[i] = -1
	}
	recordStart := func(line int, col int) {
		if line >= 1 && line <= numLines {
			if earliestStmtCol[line] < 0 || col < earliestStmtCol[line] {
				earliestStmtCol[line] = col
			}
		}
	}

	for _, st := range fc.Statements {
		if !st.Skip {
			recordStart(st.StartLine, st.StartCol)
		}
	}
	for _, fn := range fc.Functions {
		if !fn.Skip {
			recordStart(fn.Decl.StartLine, fn.Decl.StartCol)
		}
	}

	var candidates []tsLineCandidate
	addCandidate := func(kind tsCandidateKind, branchType string, sp gotestutil.IstanbulSpan) {
		if sp.Skip {
			return
		}
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
			return
		}

		candidates = append(candidates, tsLineCandidate{
			kind:          kind,
			branchType:    branchType,
			effStart:      effStart,
			effEnd:        effEnd,
			origStartLine: sp.StartLine,
			origStartCol:  sp.StartCol,
			origEndLine:   sp.EndLine,
			origEndCol:    resolvedEndCol,
			count:         sp.Count,
		})
	}

	for _, st := range fc.Statements {
		addCandidate(tsCandidateStatement, "", st)
	}
	for _, fn := range fc.Functions {
		if fn.Skip {
			continue
		}
		addCandidate(tsCandidateFnDecl, "", fn.Decl)
		addCandidate(tsCandidateFnBody, "", fn.Body)
	}
	for _, br := range fc.Branches {
		// Istanbul records the entire IfStatement's loc (starting at the "if" keyword) on locations[0]
		// as a badge anchor, while every statement inside consequent/alternate already has its own
		// statementMap entry. Skipping "if" branch anchors keeps evaluated "if (cond) {" headers covered.
		if br.BranchType == "if" {
			continue
		}
		addCandidate(tsCandidateBranchArm, br.BranchType, br.Arm)
	}

	for l := 1; l <= numLines; l++ {
		if !codeLines[l] {
			continue
		}

		hasCoveredEarlierStmtOnLine := false
		earliestCoveredStmtCol := -1
		for i := range candidates {
			c := &candidates[i]
			if c.kind == tsCandidateStatement && c.origStartLine == l && c.count > 0 {
				hasCoveredEarlierStmtOnLine = true
				if earliestCoveredStmtCol < 0 || c.origStartCol < earliestCoveredStmtCol {
					earliestCoveredStmtCol = c.origStartCol
				}
			}
		}

		var best *tsLineCandidate
		hasSingleLineUncoveredOverride := false

		for i := range candidates {
			c := &candidates[i]
			if l < c.effStart || l > c.effEnd {
				continue
			}

			// Single-line defensive nullish/logical fallbacks (e.g., "const items = res.items ?? [];")
			// should not mark an already-executed statement line uncovered.
			if c.kind == tsCandidateBranchArm &&
				(c.branchType == "binary-expr" || c.branchType == "default-arg") &&
				c.origStartLine == l && c.origEndLine == l && hasCoveredEarlierStmtOnLine {
				continue
			}

			// Inline callback declarations (e.g., "readonly x = computed(() => {") should not mark the
			// outer call/property initialization line uncovered when the body spans subsequent lines.
			if c.kind == tsCandidateFnDecl && c.origStartLine == l && c.origEndLine == l &&
				earliestCoveredStmtCol >= 0 && earliestCoveredStmtCol < c.origStartCol {
				continue
			}

			if best == nil || isNarrowerTSCandidate(c, best) {
				best = c
			}

			if c.count == 0 && c.effStart == l && c.effEnd == l {
				if c.kind == tsCandidateStatement ||
					(c.kind == tsCandidateFnBody && c.origStartLine == l && c.origEndLine == l) {
					hasSingleLineUncoveredOverride = true
				}
			}
		}

		if best == nil {
			continue
		}
		states[l].executable = true
		states[l].covered = best.count > 0 && !hasSingleLineUncoveredOverride
	}

	return states
}

func isNarrowerTSCandidate(a, b *tsLineCandidate) bool {
	lineSpanA := a.origEndLine - a.origStartLine
	lineSpanB := b.origEndLine - b.origStartLine
	if lineSpanA != lineSpanB {
		return lineSpanA < lineSpanB
	}
	if lineSpanA == 0 {
		colSpanA := a.origEndCol - a.origStartCol
		colSpanB := b.origEndCol - b.origStartCol
		if colSpanA != colSpanB {
			return colSpanA < colSpanB
		}
	}
	if (a.count == 0) != (b.count == 0) {
		return a.count == 0
	}
	return a.kind > b.kind
}

func resolveTSEndCol(rawLines []string, endLine int, endCol int) int {
	if endLine > len(rawLines) {
		if endCol < 0 {
			return 0
		}
		return endCol
	}
	lineLen := len(rawLines[endLine-1])
	if endCol < 0 || endCol > lineLen {
		return lineLen
	}
	return endCol
}

func isOnlyTSOpeningSuffix(line string, startCol int) bool {
	if startCol < 0 || startCol >= len(line) {
		return false
	}
	suffix := line[startCol:]
	if commentIdx := strings.Index(suffix, "//"); commentIdx >= 0 {
		suffix = suffix[:commentIdx]
	}
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

// classifyExecutableTSLines returns a 1-indexed boolean slice indicating lines that contain non-comment, non-closing-delimiter TypeScript tokens.
func classifyExecutableTSLines(src []byte, numLines int) []bool {
	hasSubstantiveToken := make([]bool, numLines+1)
	if numLines == 0 || len(src) == 0 {
		return hasSubstantiveToken
	}

	type scanMode int
	const (
		modeCode scanMode = iota
		modeLineComment
		modeBlockComment
		modeSingleQuote
		modeDoubleQuote
		modeTemplate
	)

	mode := modeCode
	line := 1

	markLine := func(l int) {
		if l >= 1 && l <= numLines {
			hasSubstantiveToken[l] = true
		}
	}

	for i := 0; i < len(src) && line <= numLines; i++ {
		ch := src[i]
		if ch == '\n' {
			if mode == modeLineComment || mode == modeSingleQuote || mode == modeDoubleQuote {
				mode = modeCode
			}
			line++
			continue
		}

		switch mode {
		case modeLineComment:
			continue

		case modeBlockComment:
			if ch == '*' && i+1 < len(src) && src[i+1] == '/' {
				mode = modeCode
				i++
			}

		case modeSingleQuote:
			markLine(line)
			if ch == '\\' && i+1 < len(src) && src[i+1] != '\n' {
				i++
				continue
			}
			if ch == '\'' {
				mode = modeCode
			}

		case modeDoubleQuote:
			markLine(line)
			if ch == '\\' && i+1 < len(src) && src[i+1] != '\n' {
				i++
				continue
			}
			if ch == '"' {
				mode = modeCode
			}

		case modeTemplate:
			if ch != ' ' && ch != '\t' && ch != '\r' {
				markLine(line)
			}
			if ch == '\\' && i+1 < len(src) && src[i+1] != '\n' {
				i++
				continue
			}
			if ch == '`' {
				mode = modeCode
			}

		case modeCode:
			if ch == ' ' || ch == '\t' || ch == '\r' {
				continue
			}
			if ch == '/' && i+1 < len(src) {
				if src[i+1] == '/' {
					mode = modeLineComment
					i++
					continue
				}
				if src[i+1] == '*' {
					mode = modeBlockComment
					i++
					continue
				}
			}
			switch ch {
			case '\'':
				mode = modeSingleQuote
				markLine(line)
			case '"':
				mode = modeDoubleQuote
				markLine(line)
			case '`':
				mode = modeTemplate
				markLine(line)
			case '}', ')', ']', ';', ',':
				// Closing delimiters and statement separators alone do not make a line executable.
			default:
				markLine(line)
			}
		}
	}

	return hasSubstantiveToken
}
