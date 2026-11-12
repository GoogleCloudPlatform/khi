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
	"io"
	"strconv"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

// RenderTextReport writes a terminal-friendly unified differential coverage and mutation report to w using style.
func RenderTextReport(w io.Writer, report *MutationReport, style gotestutil.TermStyle) {
	langName := report.Language.DisplayName()
	if !report.HasCandidates() && len(report.Warnings) == 0 && len(report.RedundantTests) == 0 {
		fmt.Fprintf(w, "%s\n", style.Dim(fmt.Sprintf("%s Differential Test Report (base: %s): no executable %s production lines changed.", langName, report.BaseLabel, langName)))
		if report.Coverage != nil && report.Coverage.HasOverallCoverage && report.Coverage.OverallStatements > 0 {
			fmt.Fprintf(w, "Overall %s Statement Coverage: %.1f%% (%d/%d statements)\n",
				langName, report.Coverage.OverallPercent(), report.Coverage.OverallCovered, report.Coverage.OverallStatements)
		}
		return
	}

	headerTitle := fmt.Sprintf("══ %s Differential Test Report (base: %s) ══", langName, report.BaseLabel)
	fmt.Fprintln(w, style.BoldCyan(headerTitle))

	if report.Coverage != nil && report.Coverage.PatchExecutableLines > 0 {
		patchPlain := fmt.Sprintf("%.1f%%", report.Coverage.PatchPercent())
		covLine := fmt.Sprintf("  %s %s (%d / %d changed lines covered)",
			style.Bold("Patch Coverage:"),
			style.ScoreText(report.Coverage.PatchPercent(), patchPlain),
			report.Coverage.PatchCoveredLines,
			report.Coverage.PatchExecutableLines,
		)
		if report.Coverage.HasOverallCoverage && report.Coverage.OverallStatements > 0 {
			overallPlain := fmt.Sprintf("%.1f%%", report.Coverage.OverallPercent())
			covLine += fmt.Sprintf("  |  %s %s (%d / %d statements)",
				style.Bold("Overall:"),
				style.ScoreText(report.Coverage.OverallPercent(), overallPlain),
				report.Coverage.OverallCovered,
				report.Coverage.OverallStatements,
			)
		}
		fmt.Fprintln(w, covLine)
	}

	suppressedLabel := formatSuppressedSummary(report.TotalSuppressed, report.SuppressedByCategory)
	if report.Tested() == 0 {
		fmt.Fprintf(w, "  %s %s  |  Uncovered Mutants: %d  |  Suppressed: %s\n",
			style.Bold("Mutation Score:"),
			style.Dim("n/a (0 tested)"),
			report.TotalUncovered,
			suppressedLabel,
		)
	} else {
		pctPlain := fmt.Sprintf("%.1f%%", report.ScorePercent())
		line := fmt.Sprintf("  %s %s (%d killed / %d tested)  |  Uncovered Mutants: %d  |  Suppressed: %s",
			style.Bold("Mutation Score:"),
			style.ScoreText(report.ScorePercent(), pctPlain),
			report.EffectiveKilled(),
			report.Tested(),
			report.TotalUncovered,
			suppressedLabel,
		)
		if report.TotalSkipped > 0 {
			line += fmt.Sprintf("  |  Budget-skipped: %d", report.TotalSkipped)
		}
		fmt.Fprintln(w, line)
	}

	if len(report.Files) > 0 {
		fmt.Fprintln(w)
		renderFilesSummaryTable(w, report, style)
	}

	if len(report.RedundantTests) > 0 {
		renderRedundantSummaryTable(w, report.RedundantTests, style)
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, style.BoldYellow(fmt.Sprintf("── Invalid NOMUTANTS Directives (%d) ──", len(report.Warnings))))
		for _, warn := range report.Warnings {
			loc := fmt.Sprintf("%s:%d", warn.File, warn.Line)
			fmt.Fprintf(w, "  %s: %s\n", style.Bold(loc), warn.Message)
		}
	}

	surviving := report.SurvivingMutants()
	if len(surviving) > 0 {
		fmt.Fprintln(w)
		secTitle := fmt.Sprintf("── Surviving Mutants (%d) — tests still passed when logic changed ──", len(surviving))
		fmt.Fprintln(w, style.BoldRed(secTitle))

		curFile := ""
		for _, m := range surviving {
			if m.File != curFile {
				if curFile != "" {
					fmt.Fprintln(w)
				}
				curFile = m.File
				fmt.Fprintf(w, "  %s\n", style.Bold(curFile))
			}
			idTag := style.BoldYellow(fmt.Sprintf("[M%d]", m.GlobalIndex))
			lineTag := style.Cyan(fmt.Sprintf("line %d", m.Line))
			opHint := style.Dim(fmt.Sprintf("(%s: %s)", m.Operator.Label(), m.Operator))
			fmt.Fprintf(w, "    %s %s  %s\n", idTag, lineTag, opHint)

			styledOld, styledNew := gotestutil.HighlightInlineDiff(m.OriginalLine, m.MutatedLine, style)
			fmt.Fprintf(w, "      %s %s\n", style.Red("-"), styledOld)
			fmt.Fprintf(w, "      %s %s\n", style.Green("+"), styledNew)
		}
		fmt.Fprintln(w)
		tipText := "Tip: Delete redundant/unreachable branches first, or strengthen assertions (cmp.Diff / boundary inputs) rather than adding mock-heavy tests."
		if report.Language == LangTS {
			tipText = "Tip: Delete redundant/unreachable branches first, or strengthen assertions in *.spec.ts rather than adding mock-heavy boilerplate."
		}
		fmt.Fprintf(w, "  %s\n", style.Dim(tipText))
	}

	renderUncoveredSectionText(w, report, style)

	if len(report.RedundantTests) > 0 {
		renderRedundantTestsText(w, report.Language, report.RedundantTests, style)
	}
}

func renderFilesSummaryTable(w io.Writer, report *MutationReport, style gotestutil.TermStyle) {
	hasCov := report.Coverage != nil
	var headers []string
	if hasCov {
		headers = []string{"FILE", "PATCH COV", "MUTATION", "KILLED", "SURVIVED", "SUPPRESSED", "UNCOVERED LINES"}
	} else {
		headers = []string{"FILE", "SCORE", "KILLED", "SURVIVED", "UNCOVERED", "SUPPRESSED", "UNCOVERED LINES"}
	}

	rows := make([][]gotestutil.TableCell, 0, len(report.Files))
	for i := range report.Files {
		f := &report.Files[i]
		uncovLinesPlain := f.UncoveredLineRanges()
		uncovLinesStyled := style.Dim(uncovLinesPlain)
		if uncovLinesPlain != "-" {
			uncovLinesStyled = style.Yellow(uncovLinesPlain)
		}

		fileStyled := f.File
		if f.Survived > 0 || uncovLinesPlain != "-" {
			fileStyled = style.Bold(f.File)
		} else if f.Tested() == 0 && (f.LineCoverage == nil || f.LineCoverage.ChangedExecutableLines == 0) {
			fileStyled = style.Dim(f.File)
		}

		mutPlain := "n/a"
		mutStyled := style.Dim("n/a")
		if f.Tested() > 0 {
			mutPlain = fmt.Sprintf("%.1f%% (%d/%d)", f.ScorePercent(), f.EffectiveKilled(), f.Tested())
			mutStyled = style.ScoreText(f.ScorePercent(), mutPlain)
		}

		killedPlain := strconv.Itoa(f.EffectiveKilled())
		killedStyled := style.Dim(killedPlain)
		if f.EffectiveKilled() > 0 {
			killedStyled = style.Green(killedPlain)
		}

		survivedPlain := strconv.Itoa(f.Survived)
		survivedStyled := style.Dim(survivedPlain)
		if f.Survived > 0 {
			survivedStyled = style.BoldRed(survivedPlain)
		}

		suppPlain := strconv.Itoa(f.Suppressed)

		if hasCov {
			patchPlain := "n/a"
			patchStyled := style.Dim("n/a")
			if f.LineCoverage != nil && f.LineCoverage.ChangedExecutableLines > 0 {
				patchPlain = fmt.Sprintf("%.1f%% (%d/%d)",
					f.LineCoverage.PatchPercent(),
					f.LineCoverage.ChangedCoveredLines,
					f.LineCoverage.ChangedExecutableLines,
				)
				patchStyled = style.ScoreText(f.LineCoverage.PatchPercent(), patchPlain)
			}
			rows = append(rows, []gotestutil.TableCell{
				{Plain: f.File, Styled: fileStyled},
				{Plain: patchPlain, Styled: patchStyled, AlignRight: true},
				{Plain: mutPlain, Styled: mutStyled, AlignRight: true},
				{Plain: killedPlain, Styled: killedStyled, AlignRight: true},
				{Plain: survivedPlain, Styled: survivedStyled, AlignRight: true},
				{Plain: suppPlain, Styled: style.Dim(suppPlain), AlignRight: true},
				{Plain: uncovLinesPlain, Styled: uncovLinesStyled},
			})
			continue
		}

		uncoveredPlain := strconv.Itoa(f.Uncovered)
		uncoveredStyled := style.Dim(uncoveredPlain)
		if f.Uncovered > 0 {
			uncoveredStyled = style.Yellow(uncoveredPlain)
		}
		rows = append(rows, []gotestutil.TableCell{
			{Plain: f.File, Styled: fileStyled},
			{Plain: mutPlain, Styled: mutStyled, AlignRight: true},
			{Plain: killedPlain, Styled: killedStyled, AlignRight: true},
			{Plain: survivedPlain, Styled: survivedStyled, AlignRight: true},
			{Plain: uncoveredPlain, Styled: uncoveredStyled, AlignRight: true},
			{Plain: suppPlain, Styled: style.Dim(suppPlain), AlignRight: true},
			{Plain: uncovLinesPlain, Styled: uncovLinesStyled},
		})
	}
	gotestutil.RenderAlignedTable(w, "  ", headers, rows, style)
}

type uncoveredLineEntry struct {
	file string
	line int
	code string
}

func collectUncoveredMutantLines(report *MutationReport, skipCoveredSpans map[string]map[int]struct{}) []uncoveredLineEntry {
	var out []uncoveredLineEntry
	for i := range report.Files {
		f := &report.Files[i]
		for _, m := range f.Mutants {
			if m.Status != MutantUncovered {
				continue
			}
			if fileSet, ok := skipCoveredSpans[f.File]; ok {
				if _, alreadyShown := fileSet[m.Line]; alreadyShown {
					continue
				}
			}
			code := m.OriginalLine
			if code == "" {
				code = m.OriginalExpr
			}
			if len(out) > 0 && out[len(out)-1].file == f.File && out[len(out)-1].line == m.Line {
				continue
			}
			out = append(out, uncoveredLineEntry{
				file: f.File,
				line: m.Line,
				code: code,
			})
		}
	}
	return out
}

func renderUncoveredSectionText(w io.Writer, report *MutationReport, style gotestutil.TermStyle) {
	spans := collectAllCoverageSpans(report.Coverage)
	shownBySpan := make(map[string]map[int]struct{})
	for i := range spans {
		sp := &spans[i]
		set, ok := shownBySpan[sp.File]
		if !ok {
			set = make(map[int]struct{})
			shownBySpan[sp.File] = set
		}
		for ln := sp.StartLine; ln <= sp.EndLine; ln++ {
			set[ln] = struct{}{}
		}
	}

	extraMutantLines := collectUncoveredMutantLines(report, shownBySpan)
	if len(spans) == 0 && len(extraMutantLines) == 0 {
		return
	}

	fmt.Fprintln(w)
	if len(spans) > 0 {
		fmt.Fprintln(w, style.BoldYellow(fmt.Sprintf("── Uncovered Changed Code (%d block(s)) — never executed by tests ──", len(spans))))
		curFile := ""
		for i := range spans {
			sp := &spans[i]
			if sp.File != curFile {
				if curFile != "" {
					fmt.Fprintln(w)
				}
				curFile = sp.File
				fmt.Fprintf(w, "  %s\n", style.Bold(curFile))
			}
			fmt.Fprintf(w, "    %s\n", style.Cyan("lines "+formatLineSpan(sp.StartLine, sp.EndLine)))
			for _, ln := range sp.Lines {
				numTag := style.Dim(fmt.Sprintf("%4d |", ln.Number))
				var codeText string
				if ln.Number < sp.StartLine {
					codeText = style.Dim(ln.Text)
				} else {
					codeText = style.Yellow(ln.Text)
				}
				fmt.Fprintf(w, "      %s %s\n", numTag, codeText)
			}
		}
		if len(extraMutantLines) == 0 {
			return
		}
		fmt.Fprintln(w)
	}

	secTitle := fmt.Sprintf("── Uncovered Changed Lines (%d lines, %d mutants) — never executed by tests ──", len(extraMutantLines), report.TotalUncovered)
	fmt.Fprintln(w, style.BoldYellow(secTitle))

	curFile := ""
	for _, e := range extraMutantLines {
		if e.file != curFile {
			if curFile != "" {
				fmt.Fprintln(w)
			}
			curFile = e.file
			fmt.Fprintf(w, "  %s\n", style.Bold(curFile))
		}
		numTag := style.Dim(fmt.Sprintf("%4d |", e.line))
		fmt.Fprintf(w, "    %s %s\n", numTag, style.Yellow(e.code))
	}
}

func renderRedundantSummaryTable(w io.Writer, findings []RedundantTestFinding, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n", style.BoldYellow(fmt.Sprintf("Redundant Tests Summary (%d):", len(findings))))
	headers := []string{"ID", "LOCATION", "REDUNDANT TEST", "KIND", "COVERED BY"}
	rows := make([][]gotestutil.TableCell, 0, len(findings))
	for _, rt := range findings {
		idPlain := fmt.Sprintf("[R%d]", rt.GlobalIndex)
		locPlain := fmt.Sprintf("%s:%d", rt.File, rt.Line)
		kindPlain := redundantKindBadge(rt.Kind)

		candTop, candSub := splitTopAndSubtest(rt.TestName)
		domTop, domSub := splitTopAndSubtest(rt.DominatedBy)
		sharedTop := ""
		if candSub != "" && domSub != "" && candTop == domTop && (rt.DominatedByFile == "" || rt.DominatedByFile == rt.File) {
			sharedTop = candTop
		}
		covLoc := formatCoveringLocation(rt.File, rt.DominatedByFile, rt.DominatedByLine)
		covPlain := fmt.Sprintf("%s (%s)", covLoc, trimSharedTopFunc(rt.DominatedBy, sharedTop))

		rows = append(rows, []gotestutil.TableCell{
			{Plain: idPlain, Styled: style.BoldYellow(idPlain)},
			{Plain: locPlain, Styled: style.Cyan(locPlain)},
			{Plain: rt.TestName, Styled: style.Bold(rt.TestName)},
			{Plain: kindPlain, Styled: style.Yellow(kindPlain)},
			{Plain: covPlain, Styled: style.Dim(covPlain)},
		})
	}
	gotestutil.RenderAlignedTable(w, "  ", headers, rows, style)
}

type redundantDisplayGroup struct {
	file            string
	kind            RedundantTestKind
	dominatedBy     string
	dominatedByFile string
	dominatedByLine int
	dominatorKills  int
	sharedTopFunc   string
	items           []RedundantTestFinding
}

func groupRedundantFindings(findings []RedundantTestFinding) []redundantDisplayGroup {
	var groups []redundantDisplayGroup
	for _, rt := range findings {
		if len(groups) > 0 {
			last := &groups[len(groups)-1]
			if last.file == rt.File &&
				last.kind == rt.Kind &&
				last.dominatedBy == rt.DominatedBy &&
				last.dominatedByFile == rt.DominatedByFile &&
				last.dominatedByLine == rt.DominatedByLine {
				last.items = append(last.items, rt)
				if topFn, _ := splitTopAndSubtest(rt.TestName); topFn != last.sharedTopFunc {
					last.sharedTopFunc = ""
				}
				continue
			}
		}
		candTop, candSub := splitTopAndSubtest(rt.TestName)
		domTop, domSub := splitTopAndSubtest(rt.DominatedBy)
		sharedTop := ""
		if candSub != "" && domSub != "" && candTop == domTop && (rt.DominatedByFile == "" || rt.DominatedByFile == rt.File) {
			sharedTop = candTop
		}
		groups = append(groups, redundantDisplayGroup{
			file:            rt.File,
			kind:            rt.Kind,
			dominatedBy:     rt.DominatedBy,
			dominatedByFile: rt.DominatedByFile,
			dominatedByLine: rt.DominatedByLine,
			dominatorKills:  rt.DominatorKills,
			sharedTopFunc:   sharedTop,
			items:           []RedundantTestFinding{rt},
		})
	}
	return groups
}

func renderRedundantTestsText(w io.Writer, lang Language, findings []RedundantTestFinding, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	secTitle := fmt.Sprintf("── Redundant Tests in Diff (%d) — already covered by another test ──", len(findings))
	fmt.Fprintln(w, style.BoldYellow(secTitle))

	groups := groupRedundantFindings(findings)
	curHeader := ""
	for _, g := range groups {
		headerPlain := g.file
		headerStyled := style.Bold(g.file)
		if g.sharedTopFunc != "" {
			headerPlain += "  (in " + g.sharedTopFunc + ")"
			headerStyled += "  " + style.Dim("(in "+g.sharedTopFunc+")")
		}
		if headerPlain != curHeader {
			if curHeader != "" {
				fmt.Fprintln(w)
			}
			curHeader = headerPlain
			fmt.Fprintf(w, "  %s\n", headerStyled)
		}

		covLoc := formatCoveringLocation(g.file, g.dominatedByFile, g.dominatedByLine)
		covName := trimSharedTopFunc(g.dominatedBy, g.sharedTopFunc)
		badge := style.Yellow("[" + redundantKindBadge(g.kind) + "]")

		if len(g.items) == 1 {
			rt := g.items[0]
			idTag := style.BoldYellow(fmt.Sprintf("[R%d]", rt.GlobalIndex))
			lineTag := style.Cyan(fmt.Sprintf("line %d:", rt.Line))
			candName := trimSharedTopFunc(rt.TestName, g.sharedTopFunc)
			fmt.Fprintf(w, "    %s %s %s  %s\n", idTag, lineTag, candName, badge)
			fmt.Fprintf(w, "         %s %s: %s\n", style.Bold("Covered by"), style.Cyan(covLoc), covName)
			fmt.Fprintf(w, "         %s %s\n", style.Bold("Why:"), explainSingleRedundantWhy(lang, rt, covLoc))
			fmt.Fprintf(w, "         %s %s\n", style.Bold("Fix:"), explainRedundantFix(rt.Kind, false))
			continue
		}

		fmt.Fprintf(w, "    %s %s: %s\n", style.Bold("Covered by"), style.Cyan(covLoc), covName)
		for _, rt := range g.items {
			idTag := style.BoldYellow(fmt.Sprintf("[R%d]", rt.GlobalIndex))
			lineTag := style.Cyan(fmt.Sprintf("line %d:", rt.Line))
			candName := trimSharedTopFunc(rt.TestName, g.sharedTopFunc)
			fmt.Fprintf(w, "      - %s %s %s  %s\n", idTag, lineTag, candName, badge)
		}
		fmt.Fprintf(w, "      %s %s\n", style.Bold("Why:"), explainGroupRedundantWhy(lang, g, covLoc))
		fmt.Fprintf(w, "      %s %s\n", style.Bold("Fix:"), explainRedundantFix(g.kind, true))
	}
}

func explainGroupRedundantWhy(lang Language, g redundantDisplayGroup, covLoc string) string {
	if g.kind == RedundantWeakAssertion {
		if lang == LangTS {
			return fmt.Sprintf("All %d cases only perform trivial smoke checks without asserting outputs or DOM/state changes, on a path already tested at %s.", len(g.items), covLoc)
		}
		return fmt.Sprintf("All %d cases only check 'err != nil' without asserting outputs or state changes, on a path already tested at %s.", len(g.items), covLoc)
	}
	allEqual := true
	for _, it := range g.items {
		if it.CandidateKills != g.dominatorKills {
			allEqual = false
			break
		}
	}
	if allEqual && g.dominatorKills > 0 {
		return fmt.Sprintf("All %d cases run the exact same code path and check the exact same %d logic condition(s) as %s (no new branches tested).", len(g.items), g.dominatorKills, covLoc)
	}
	return fmt.Sprintf("All %d cases run the exact same if/else path as %s without catching any additional logic bugs.", len(g.items), covLoc)
}
