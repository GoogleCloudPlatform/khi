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

package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const maxRenderedHotspots = 5

// RenderText writes the human- and agent-readable terminal Go Quality report to w.
func RenderText(w io.Writer, rep *rule.Report, style gotestutil.TermStyle) {
	renderDashboardHeader(w, rep, style)
	if len(rep.DirectiveErrors) > 0 {
		renderDirectiveErrors(w, rep.DirectiveErrors, style)
	}
	if len(rep.Findings) > 0 {
		renderFindingsTable(w, rep.Findings, style)
		renderFindingDetails(w, rep.Findings, style)
	}
	if len(rep.Suppressed) > 0 {
		renderSuppressedSummary(w, rep.Suppressed, style)
	}
	if len(rep.Similarity.Advisories) > 0 {
		renderSimilarityAdvisories(w, rep.LanguageTitle(), rep.Similarity.Advisories, style)
	}
}

// FormatBudgetValue formats a Finding's measured value, optional base value, and limit concisely.
func FormatBudgetValue(f rule.Finding) string {
	if f.HasBefore {
		return fmt.Sprintf("%d -> %d (<= %d)", f.Before, f.After, f.Limit)
	}
	return fmt.Sprintf("%d (<= %d)", f.After, f.Limit)
}

func renderDashboardHeader(w io.Writer, rep *rule.Report, style gotestutil.TermStyle) {
	modeNote := ""
	if rep.FastMode {
		modeNote = " [fast AST mode]"
	}
	title := fmt.Sprintf("══ %s Quality Report (base: %s)%s ══", rep.LanguageTitle(), rep.BaseLabel, modeNote)
	fmt.Fprintln(w, style.BoldCyan(title))
	fmt.Fprintf(w, "  Scope:      %d prod file(s), %d test file(s), %d unit(s) evaluated\n",
		rep.ProdFilesChanged, rep.TestFilesChanged, rep.FuncsEvaluated)

	if rep.Coverage.Enabled {
		renderCoverageLine(w, rep.Coverage, style)
	}
	if rep.Mutation.Enabled {
		renderMutationLine(w, rep.Mutation, style)
	}
	if rep.Similarity.Enabled {
		renderSimilarityHeaderLine(w, rep.Similarity, style)
	}
	renderVerdictLine(w, rep, style)
}

func renderCoverageLine(w io.Writer, cov rule.CoverageSummary, style gotestutil.TermStyle) {
	patchTxt := fmt.Sprintf("%.1f%% (%d/%d changed executable lines)", cov.PatchPercent(), cov.PatchCovered, cov.PatchExecutable)
	line := "  Coverage:   Patch " + style.ScoreText(cov.PatchPercent(), patchTxt)
	if cov.HasOverall {
		ovTxt := fmt.Sprintf("%.1f%% (%d/%d stmts)", cov.OverallPercent(), cov.OverallCovered, cov.OverallStatements)
		line += "  |  Overall " + style.Dim(ovTxt)
	}
	fmt.Fprintln(w, line)
}

func renderMutationLine(w io.Writer, mut rule.MutationSummary, style gotestutil.TermStyle) {
	scoreTxt := fmt.Sprintf("%.1f%% (%d/%d killed, %d survived", mut.Score(), mut.DetectedCount(), mut.TotalEvaluated(), mut.Survived)
	if mut.Suppressed > 0 {
		scoreTxt += fmt.Sprintf(", %d suppressed", mut.Suppressed)
	}
	if mut.SkippedBudget > 0 {
		scoreTxt += fmt.Sprintf(", %d budget-skipped", mut.SkippedBudget)
	}
	scoreTxt += ")"
	fmt.Fprintf(w, "  Mutation:   %s\n", style.ScoreText(mut.Score(), scoreTxt))
}

func renderVerdictLine(w io.Writer, rep *rule.Report, style gotestutil.TermStyle) {
	if !rep.HasIssues() {
		fmt.Fprintf(w, "  Verdict:    %s\n", style.BoldGreen("PASS (0 violations)"))
		return
	}
	if rep.Advisory {
		msg := fmt.Sprintf("ADVISORY (%d finding(s), %d directive error(s))", len(rep.Findings), len(rep.DirectiveErrors))
		fmt.Fprintf(w, "  Verdict:    %s\n", style.Yellow(msg))
		return
	}
	msg := fmt.Sprintf("FAIL (%d violation(s), %d directive error(s))", len(rep.Findings), len(rep.DirectiveErrors))
	fmt.Fprintf(w, "  Verdict:    %s\n", style.BoldRed(msg))
}

func renderDirectiveErrors(w io.Writer, errs []rule.DirectiveError, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, style.BoldRed(fmt.Sprintf("── Invalid or Unused CODEQUALITY_ALLOW Directives (%d) ──", len(errs))))
	for _, e := range errs {
		fmt.Fprintf(w, "  %s:%d: %s\n", e.File, e.Line, e.Message)
	}
}

func renderFindingsTable(w io.Writer, findings []rule.Finding, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, style.BoldRed(fmt.Sprintf("── Quality Rule Violations (%d) ──", len(findings))))
	headers := []string{"#", "RULE", "STATUS", "VALUE (LIMIT)", "TARGET"}
	rows := make([][]gotestutil.TableCell, 0, len(findings))
	for i, f := range findings {
		idTag := fmt.Sprintf("[Q%d]", indexOrFallback(f.GlobalIndex, i+1))
		statusPlain := string(f.Kind)
		valPlain := FormatBudgetValue(f)
		targetPlain := fmt.Sprintf("%s:%d · %s", f.File, f.Line, f.Subject)
		rows = append(rows, []gotestutil.TableCell{
			{Plain: idTag, Styled: style.BoldRed(idTag)},
			{Plain: string(f.Rule), Styled: style.Bold(string(f.Rule))},
			{Plain: statusPlain, Styled: style.Red(statusPlain)},
			{Plain: valPlain, Styled: style.Yellow(valPlain)},
			{Plain: targetPlain, Styled: targetPlain},
		})
	}
	gotestutil.RenderAlignedTable(w, "  ", headers, rows, style)
}

func renderFindingDetails(w io.Writer, findings []rule.Finding, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	for i, f := range findings {
		idTag := fmt.Sprintf("[Q%d]", indexOrFallback(f.GlobalIndex, i+1))
		header := fmt.Sprintf("%s %s — %s:%d (%s) [%s: %s]",
			style.BoldRed(idTag), style.Bold(string(f.Rule)), f.File, f.Line, f.Subject, f.Kind, FormatBudgetValue(f))
		fmt.Fprintf(w, "  %s\n", header)
		if f.Detail != "" {
			fmt.Fprintf(w, "      Detail: %s\n", f.Detail)
		}
		if spec, ok := rule.LookupRuleSpec(f.Rule); ok {
			fmt.Fprintf(w, "      Why:    %s\n", style.Dim(spec.Why))
			fmt.Fprintf(w, "      Fix:    %s\n", spec.Fix)
		}
		renderFindingEvidence(w, f, style)
	}
}

func renderFindingEvidence(w io.Writer, f rule.Finding, style gotestutil.TermStyle) {
	if len(f.Hotspots) > 0 {
		renderHotspots(w, f.Hotspots, style)
	}
	renderUncoveredSpans(w, f.UncoveredSpans, style)
	renderSurvivingMutants(w, f.SurvivingMutants, style)
	renderRedundantTests(w, f.RedundantTests, style)
}

func renderHotspots(w io.Writer, hotspots []rule.Hotspot, style gotestutil.TermStyle) {
	limit := min(len(hotspots), maxRenderedHotspots)
	parts := make([]string, 0, limit)
	for _, h := range hotspots[:limit] {
		parts = append(parts, fmt.Sprintf("L%d %s", h.Line, h.Summary))
	}
	extra := ""
	if len(hotspots) > limit {
		extra = fmt.Sprintf(" (+%d more)", len(hotspots)-limit)
	}
	fmt.Fprintf(w, "      Hotspots: %s%s\n", style.Dim(strings.Join(parts, "; ")), extra)
}

func renderUncoveredSpans(w io.Writer, spans []rule.UncoveredSpan, style gotestutil.TermStyle) {
	for _, sp := range spans {
		fmt.Fprintf(w, "      Uncovered L%d-%d:\n", sp.StartLine, sp.EndLine)
		for _, ln := range sp.Lines {
			fmt.Fprintf(w, "        %s\n", style.Red(fmt.Sprintf("%4d | %s", ln.Number, ln.Text)))
		}
	}
}

func renderSurvivingMutants(w io.Writer, mutants []rule.MutantDetail, style gotestutil.TermStyle) {
	for _, m := range mutants {
		oldStr, newStr := gotestutil.HighlightInlineDiff(m.OriginalLine, m.MutatedLine, style)
		fmt.Fprintf(w, "      Mutant L%d [%s — %s]:\n", m.Line, m.Operator, m.Operator.Label())
		fmt.Fprintf(w, "        - %s\n", oldStr)
		fmt.Fprintf(w, "        + %s\n", newStr)
	}
}

func renderRedundantTests(w io.Writer, tests []rule.RedundantDetail, style gotestutil.TermStyle) {
	for _, rt := range tests {
		fmt.Fprintf(w, "      [%s] L%d %s -> dominated by %s (%s:%d, kills %d vs %d)\n",
			style.Yellow(string(rt.Kind)), rt.Line, rt.TestName, rt.DominatedBy, rt.DominatedByFile, rt.DominatedByLine, rt.CandidateKills, rt.DominatorKills)
	}
}

func renderSuppressedSummary(w io.Writer, suppressed []rule.SuppressedItem, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, style.Dim(fmt.Sprintf("── Suppressed via CODEQUALITY_ALLOW (%d) ──", len(suppressed))))
	for _, s := range suppressed {
		fmt.Fprintf(w, "  %s:%d (%s) [%s / %s]: %s\n", s.File, s.Line, s.Subject, s.Rule, s.Category, s.Reason)
	}
}

func indexOrFallback(globalIdx int, fallback int) int {
	if globalIdx > 0 {
		return globalIdx
	}
	return fallback
}
