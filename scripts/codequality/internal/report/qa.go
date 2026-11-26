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
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// RenderQAJSON writes rep as a single normalized JSON object to w.
func RenderQAJSON(w io.Writer, rep *rule.QAReport) error {
	var normalized rule.QAReport
	if rep != nil {
		normalized = *rep
	}
	normalized.Normalize()
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(normalized)
}

// ConvertReportsToQAReport converts legacy rule.Report values into a normalized QAReport.
func ConvertReportsToQAReport(baseLabel string, speed rule.Speed, filter rule.Filter, reports []*rule.Report) *rule.QAReport {
	qa := &rule.QAReport{
		BaseLabel: baseLabel,
		Speed:     speed,
		Filter:    filter,
	}
	for _, rep := range reports {
		if rep == nil {
			continue
		}
		appendReportToQA(qa, rep)
	}
	if filter == rule.FilterMust {
		qa.Advisory = nil
	} else if filter == rule.FilterAdvisory {
		qa.MustFix = nil
	}
	qa.Normalize()
	return qa
}

// RenderQAText writes a human- and agent-readable QA summary to w.
func RenderQAText(w io.Writer, rep *rule.QAReport, style gotestutil.TermStyle) {
	if rep == nil {
		rep = &rule.QAReport{}
	}
	title := fmt.Sprintf("══ KHI QA Report (base: %s, speed: %s, filter: %s) ══", rep.BaseLabel, rep.Speed, rep.Filter)
	fmt.Fprintln(w, style.BoldCyan(title))
	fmt.Fprintf(w, "  Summary:    %d mustfix, %d advisory, %d skipped\n", len(rep.MustFix), len(rep.Advisory), len(rep.SkippedSources))
	if len(rep.MustFix) > 0 {
		renderQAItemSection(w, style.BoldRed(fmt.Sprintf("── MUSTFIX (%d) ──", len(rep.MustFix))), rep.MustFix, style)
	}
	if len(rep.Advisory) > 0 {
		renderQAItemSection(w, style.BoldYellow(fmt.Sprintf("── ADVISORY (%d) ──", len(rep.Advisory))), rep.Advisory, style)
	}
	if len(rep.SkippedSources) > 0 {
		renderQASkippedSection(w, rep.SkippedSources, style)
	}
}

// RenderQAGitHub writes GitHub Actions workflow commands for MustFix, Advisory, and SkippedSources in rep.
func RenderQAGitHub(w io.Writer, rep *rule.QAReport) {
	if rep == nil {
		return
	}
	if len(rep.MustFix) == 0 && len(rep.Advisory) == 0 && len(rep.SkippedSources) == 0 {
		title := fmt.Sprintf("KHI QA (%s)", rep.BaseLabel)
		msg := fmt.Sprintf("clean (speed=%s, filter=%s)", rep.Speed, rep.Filter)
		fmt.Fprintf(w, "::notice title=%s::%s\n", gotestutil.EscapeAnnotationProperty(title), gotestutil.EscapeAnnotationData(msg))
		return
	}
	for _, item := range rep.MustFix {
		emitAnnotation(w, annotationSpec{
			level:   "error",
			file:    item.File,
			line:    item.Line,
			title:   fmt.Sprintf("MUSTFIX (%s)", formatQACheckLabel(item)),
			message: formatQAGitHubMessage(item),
		})
	}
	for _, item := range rep.Advisory {
		emitAnnotation(w, annotationSpec{
			level:   "warning",
			file:    item.File,
			line:    item.Line,
			title:   fmt.Sprintf("ADVISORY (%s)", formatQACheckLabel(item)),
			message: formatQAGitHubMessage(item),
		})
	}
	for _, skip := range rep.SkippedSources {
		fmt.Fprintf(
			w,
			"::notice title=%s::%s\n",
			gotestutil.EscapeAnnotationProperty(fmt.Sprintf("Skipped (%s)", skip.Source)),
			gotestutil.EscapeAnnotationData(skip.Reason),
		)
	}
}

func appendReportToQA(qa *rule.QAReport, rep *rule.Report) {
	source := reportSourceName(rep)
	for i, d := range rep.DirectiveErrors {
		qa.MustFix = append(qa.MustFix, rule.QAItem{
			ID:      fmt.Sprintf("%s:directive:%s:%d:%d", source, d.File, d.Line, i+1),
			Source:  source,
			Check:   "directive",
			File:    d.File,
			Line:    d.Line,
			Summary: d.Message,
			Detail:  d.Message,
		})
	}
	for i, f := range rep.Findings {
		item := findingToQAItem(source, i, f)
		if rep.Advisory || f.IsAdvisory() {
			qa.Advisory = append(qa.Advisory, item)
		} else {
			qa.MustFix = append(qa.MustFix, item)
		}
	}
	appendDecisionsToQA(qa, rep)
	appendSimilarityToQA(qa, rep)
}

func findingToQAItem(source string, idx int, f rule.Finding) rule.QAItem {
	summary := cmp.Or(f.Detail, fmt.Sprintf("%s [%s]: %s", f.Subject, f.Kind, FormatBudgetValue(f)))
	return rule.QAItem{
		ID:           fmt.Sprintf("%s:%s:%s:%d:%d", source, f.Rule, f.File, f.Line, idx+1),
		Source:       source,
		Check:        string(f.Rule),
		File:         f.File,
		Line:         f.Line,
		EndLine:      f.EndLine,
		Symbol:       f.Subject,
		Summary:      summary,
		Detail:       formatFindingDetail(f),
		Evidence:     buildFindingEvidence(f),
		ReviewPrompt: reviewPromptForRule(f.Rule),
	}
}

func buildFindingEvidence(f rule.Finding) map[string]any {
	ev := map[string]any{
		"kind":  string(f.Kind),
		"after": f.After,
		"limit": f.Limit,
	}
	if len(f.UncoveredSpans) > 0 {
		ev["uncovered_spans"] = buildUncoveredSpansEvidence(f.UncoveredSpans)
	}
	if len(f.SurvivingMutants) > 0 {
		ev["surviving_mutants"] = buildSurvivingMutantsEvidence(f.SurvivingMutants)
	}
	if len(f.RedundantTests) > 0 {
		ev["redundant_tests"] = buildRedundantTestsEvidence(f.RedundantTests)
	}
	return ev
}

func buildUncoveredSpansEvidence(spans []rule.UncoveredSpan) []map[string]any {
	out := make([]map[string]any, 0, len(spans))
	for _, sp := range spans {
		out = append(out, map[string]any{
			"file":             sp.File,
			"start_line":       sp.StartLine,
			"end_line":         sp.EndLine,
			"executable_lines": sp.ExecutableLines,
		})
	}
	return out
}

func buildSurvivingMutantsEvidence(mutants []rule.MutantDetail) []map[string]any {
	out := make([]map[string]any, 0, len(mutants))
	for _, m := range mutants {
		out = append(out, map[string]any{
			"operator": string(m.Operator),
			"line":     m.Line,
			"original": m.OriginalExpr,
			"mutated":  m.MutatedExpr,
		})
	}
	return out
}

func buildRedundantTestsEvidence(tests []rule.RedundantDetail) []map[string]any {
	out := make([]map[string]any, 0, len(tests))
	for _, rt := range tests {
		out = append(out, map[string]any{
			"kind":              string(rt.Kind),
			"line":              rt.Line,
			"test_name":         rt.TestName,
			"dominated_by":      rt.DominatedBy,
			"dominated_by_file": rt.DominatedByFile,
			"dominated_by_line": rt.DominatedByLine,
			"candidate_kills":   rt.CandidateKills,
			"dominator_kills":   rt.DominatorKills,
		})
	}
	return out
}

func reviewPromptForRule(id rule.RuleID) string {
	switch id {
	case rule.RuleFuncUncoveredLines:
		return "Inspect uncovered spans in context: add targeted tests for missing branches, or retain legitimate defensive/pre-sizing guards instead of deleting safety checks."
	case rule.RuleFuncSurvivingMutants:
		return "Inspect surviving mutants in context: strengthen assertions for missing edge checks, or retain legitimate defensive guards, pre-sizing, or equivalent expressions."
	case rule.RuleTestRedundancy:
		return "Inspect flagged redundant subtests: add distinct output assertions or merge duplicates introduced in this change, and never delete pre-existing regression tests without user confirmation."
	default:
		return ""
	}
}

func formatFindingDetail(f rule.Finding) string {
	var parts []string
	if f.Detail != "" {
		parts = append(parts, f.Detail)
	}
	if s := formatUncoveredSpansDetail(f.UncoveredSpans); s != "" {
		parts = append(parts, s)
	}
	if s := formatMutantsDetail(f.SurvivingMutants); s != "" {
		parts = append(parts, s)
	}
	if s := formatRedundantTestsDetail(f.RedundantTests); s != "" {
		parts = append(parts, s)
	}
	if s := formatHotspotsDetail(f.Hotspots); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, "; ")
}

func formatUncoveredSpansDetail(spans []rule.UncoveredSpan) string {
	if len(spans) == 0 {
		return ""
	}
	labels := make([]string, 0, len(spans))
	for _, sp := range spans {
		if sp.StartLine == sp.EndLine {
			labels = append(labels, fmt.Sprintf("L%d", sp.StartLine))
		} else {
			labels = append(labels, fmt.Sprintf("L%d-L%d", sp.StartLine, sp.EndLine))
		}
	}
	return "uncovered: " + strings.Join(labels, ", ")
}

func formatMutantsDetail(mutants []rule.MutantDetail) string {
	if len(mutants) == 0 {
		return ""
	}
	labels := make([]string, 0, len(mutants))
	for _, m := range mutants {
		labels = append(labels, fmt.Sprintf("L%d (%s): %s -> %s", m.Line, m.Operator, m.OriginalExpr, m.MutatedExpr))
	}
	return "survived: " + strings.Join(labels, ", ")
}

func formatRedundantTestsDetail(tests []rule.RedundantDetail) string {
	if len(tests) == 0 {
		return ""
	}
	labels := make([]string, 0, len(tests))
	for _, rt := range tests {
		labels = append(labels, fmt.Sprintf("[%s] %s (L%d) dominated by %s", rt.Kind, rt.TestName, rt.Line, rt.DominatedBy))
	}
	return "redundant: " + strings.Join(labels, ", ")
}

func formatHotspotsDetail(hotspots []rule.Hotspot) string {
	if len(hotspots) == 0 {
		return ""
	}
	limit := min(len(hotspots), maxRenderedHotspots)
	labels := make([]string, 0, limit)
	for _, h := range hotspots[:limit] {
		labels = append(labels, fmt.Sprintf("L%d %s", h.Line, h.Summary))
	}
	return "hotspots: " + strings.Join(labels, ", ")
}

func appendDecisionsToQA(qa *rule.QAReport, rep *rule.Report) {
	if !rep.Decisions.Enabled {
		return
	}
	if rep.Decisions.Skipped {
		qa.SkippedSources = append(qa.SkippedSources, rule.SkippedSource{
			Source: "decisions",
			Reason: rep.Decisions.SkipReason,
		})
	}
	for i, df := range rep.Decisions.Findings {
		check := df.RuleID + "/" + df.QuestionID
		item := rule.QAItem{
			ID:           fmt.Sprintf("decisions:%s:%s:%d:%d", check, df.File, df.Line, i+1),
			Source:       "decisions",
			Check:        check,
			File:         df.File,
			Line:         df.Line,
			Symbol:       df.Subject,
			Summary:      cmp.Or(df.Why, df.Outcome, check),
			Detail:       df.Outcome,
			ReviewPrompt: df.Fix,
		}
		if !rep.Advisory && df.Blocking {
			qa.MustFix = append(qa.MustFix, item)
		} else {
			qa.Advisory = append(qa.Advisory, item)
		}
	}
}

func appendSimilarityToQA(qa *rule.QAReport, rep *rule.Report) {
	if !rep.Similarity.Enabled {
		return
	}
	if rep.Similarity.Skipped {
		qa.SkippedSources = append(qa.SkippedSources, rule.SkippedSource{
			Source: "similarity",
			Reason: rep.Similarity.SkipReason,
		})
	}
	for i, adv := range rep.Similarity.Advisories {
		summary := "similar existing function found"
		prompt := ""
		if len(adv.Candidates) > 0 {
			top := adv.Candidates[0]
			summary = fmt.Sprintf("similar to %s (%s:%d, score %.3f)", top.Unit.Name, top.Unit.FilePath, top.Unit.StartLine, top.Score)
			prompt = top.Guidance
		}
		qa.Advisory = append(qa.Advisory, rule.QAItem{
			ID:           fmt.Sprintf("similarity:%s:%d:%d", adv.NewUnit.FilePath, adv.NewUnit.StartLine, i+1),
			Source:       "similarity",
			Check:        "function_similarity",
			File:         adv.NewUnit.FilePath,
			Line:         adv.NewUnit.StartLine,
			Symbol:       adv.NewUnit.Name,
			Summary:      summary,
			ReviewPrompt: prompt,
		})
	}
}

func reportSourceName(rep *rule.Report) string {
	lang := strings.ToLower(strings.TrimSpace(rep.Language))
	if lang == "" {
		return "codequality"
	}
	return lang
}

func renderQAItemSection(w io.Writer, heading string, items []rule.QAItem, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, heading)
	for _, item := range items {
		loc := formatQALocation(item)
		fmt.Fprintf(w, "  [%s] %s — %s\n", formatQACheckLabel(item), loc, item.Summary)
		if item.Detail != "" && item.Detail != item.Summary {
			fmt.Fprintf(w, "      Detail: %s\n", style.Dim(item.Detail))
		}
		if item.ReviewPrompt != "" {
			fmt.Fprintf(w, "      Prompt: %s\n", item.ReviewPrompt)
		}
	}
}

func renderQASkippedSection(w io.Writer, skipped []rule.SkippedSource, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, style.Dim(fmt.Sprintf("── Skipped Sources (%d) ──", len(skipped))))
	for _, skip := range skipped {
		fmt.Fprintf(w, "  %s: %s\n", skip.Source, skip.Reason)
	}
}

func formatQALocation(item rule.QAItem) string {
	loc := fmt.Sprintf("%s:%d", item.File, max(item.Line, 1))
	if item.Symbol != "" {
		return loc + " (" + item.Symbol + ")"
	}
	return loc
}

func formatQACheckLabel(item rule.QAItem) string {
	if item.Check == "" {
		return item.Source
	}
	if item.Source == "" {
		return item.Check
	}
	return item.Source + "/" + item.Check
}

func formatQAGitHubMessage(item rule.QAItem) string {
	msg := item.Summary
	if item.Symbol != "" {
		msg = item.Symbol + ": " + msg
	}
	if item.ReviewPrompt != "" {
		msg += " | Fix: " + item.ReviewPrompt
	}
	return msg
}
