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

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func renderDecisionsReportText(w io.Writer, rep *rule.Report, style gotestutil.TermStyle) {
	title := fmt.Sprintf("══ Decision Rules Report (base: %s) ══", rep.BaseLabel)
	fmt.Fprintln(w, style.BoldCyan(title))
	dec := rep.Decisions
	if dec.Skipped {
		fmt.Fprintf(w, "  Status:     %s (%s)\n", style.Yellow("SKIPPED"), dec.SkipReason)
		return
	}
	fmt.Fprintf(w, "  Scope:      %d rule(s), %d target(s), %d question(s) evaluated (cache: %d hit(s), %d API call(s))\n",
		dec.RulesEvaluated, dec.TargetsEvaluated, dec.QuestionsEvaluated, dec.CacheHits, dec.APICalls)
	renderDecisionsVerdictLine(w, rep, style)
	if len(dec.Findings) > 0 {
		renderDecisionsTable(w, dec.Findings, style)
		renderDecisionsDetails(w, dec.Findings, style)
	}
}

func renderDecisionsVerdictLine(w io.Writer, rep *rule.Report, style gotestutil.TermStyle) {
	dec := rep.Decisions
	renderVerdictStatusLine(w, verdictStatus{
		hasIssues:       len(dec.Findings) > 0,
		blocking:        !rep.Advisory && dec.HasBlockingFindings(),
		passMessage:     "PASS (0 findings)",
		advisoryMessage: fmt.Sprintf("ADVISORY (%d finding(s))", len(dec.Findings)),
		failMessage:     fmt.Sprintf("FAIL (%d blocking, %d advisory)", dec.BlockingCount(), dec.AdvisoryCount()),
	}, style)
}

func renderDecisionsTable(w io.Writer, findings []rule.DecisionItemFinding, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, style.BoldYellow(fmt.Sprintf("── Decision Rule Findings (%d) ──", len(findings))))
	headers := []string{"#", "RULE / QUESTION", "SEVERITY", "OUTCOME", "LOCATION"}
	rows := make([][]gotestutil.TableCell, 0, len(findings))
	for i, f := range findings {
		tag, tagStyled := formatDecisionTag(f, i+1, style)
		ruleCol := f.RuleID + "/" + f.QuestionID
		sevPlain, sevStyled := formatDecisionSeverity(f.Blocking, style)
		loc := formatDecisionLocation(f)
		rows = append(rows, []gotestutil.TableCell{
			{Plain: tag, Styled: tagStyled},
			{Plain: ruleCol, Styled: style.Bold(ruleCol)},
			{Plain: sevPlain, Styled: sevStyled},
			{Plain: f.Outcome, Styled: f.Outcome},
			{Plain: loc, Styled: loc},
		})
	}
	gotestutil.RenderAlignedTable(w, "  ", headers, rows, style)
}

func renderDecisionsDetails(w io.Writer, findings []rule.DecisionItemFinding, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	for i, f := range findings {
		_, tagStyled := formatDecisionTag(f, i+1, style)
		fmt.Fprintf(w, "  %s %s/%s — %s (%q)\n",
			tagStyled, style.Bold(f.RuleID), f.QuestionID, formatDecisionSpan(f), f.Subject)
		fmt.Fprintf(w, "      Outcome: %s\n", f.Outcome)
		fmt.Fprintf(w, "      Why:     %s\n", style.Dim(f.Why))
		fmt.Fprintf(w, "      Fix:     %s\n", f.Fix)
	}
}

func renderDecisionsAnnotations(w io.Writer, rep *rule.Report, maxAnnotations int) {
	emitted := 0
	for i, f := range rep.Decisions.Findings {
		if emitted >= maxAnnotations {
			return
		}
		level := "warning"
		if !rep.Advisory && f.Blocking {
			level = "error"
		}
		tag := fmt.Sprintf("[D%d]", indexOrFallback(f.Index, i+1))
		title := fmt.Sprintf("Decision %s (%s/%s)", tag, f.RuleID, f.QuestionID)
		msg := fmt.Sprintf("%s [%s] — Why: %s | Fix: %s", f.Subject, f.Outcome, f.Why, f.Fix)
		emitAnnotation(w, annotationSpec{
			level:   level,
			file:    f.File,
			line:    f.Line,
			title:   title,
			message: msg,
		})
		emitted++
	}
}

func renderDecisionsGitHubSummary(w io.Writer, rep *rule.Report) {
	dec := rep.Decisions
	verdict := decisionMarkdownVerdict(rep)
	fmt.Fprintf(w, "## Decision Rules Report (`%s`) — %s\n\n", rep.BaseLabel, verdict)
	fmt.Fprintln(w, "| Metric | Result |")
	fmt.Fprintln(w, "| :--- | :--- |")
	if dec.Skipped {
		fmt.Fprintf(w, "| Status | Skipped (%s) |\n\n", gotestutil.EscapeMarkdownTable(dec.SkipReason))
		return
	}
	fmt.Fprintf(w, "| Evaluated Scope | %d rule(s), %d target(s), %d question(s) |\n",
		dec.RulesEvaluated, dec.TargetsEvaluated, dec.QuestionsEvaluated)
	fmt.Fprintf(w, "| Cache & API | %d cache hit(s), %d live API call(s) |\n\n",
		dec.CacheHits, dec.APICalls)
	if len(dec.Findings) > 0 {
		renderMarkdownDecisionsFindings(w, dec.Findings)
	}
}

func renderMarkdownDecisionsFindings(w io.Writer, findings []rule.DecisionItemFinding) {
	fmt.Fprintln(w, "### Decision Rule Findings")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| # | Rule / Question | Severity | Outcome | Location | Subject |")
	fmt.Fprintln(w, "| :--- | :--- | :--- | :--- | :--- | :--- |")
	for i, f := range findings {
		tag := fmt.Sprintf("`[D%d]`", indexOrFallback(f.Index, i+1))
		sev := "Advisory"
		if f.Blocking {
			sev = "Blocking"
		}
		fmt.Fprintf(w, "| %s | `%s/%s` | `%s` | `%s` | `%s` | `%s` |\n",
			tag,
			gotestutil.EscapeMarkdownTable(f.RuleID),
			gotestutil.EscapeMarkdownTable(f.QuestionID),
			sev,
			gotestutil.EscapeMarkdownTable(f.Outcome),
			formatDecisionSpan(f),
			gotestutil.EscapeMarkdownTable(f.Subject),
		)
	}
	fmt.Fprintln(w)
	for i, f := range findings {
		renderMarkdownDecisionDetail(w, i, f)
	}
}

func renderMarkdownDecisionDetail(w io.Writer, idx int, f rule.DecisionItemFinding) {
	tag := fmt.Sprintf("[D%d]", indexOrFallback(f.Index, idx+1))
	fmt.Fprintln(w, "<details>")
	fmt.Fprintf(w, "<summary><strong>%s %s/%s</strong> — <code>%s</code> (<code>%s</code>)</summary>\n\n",
		tag, f.RuleID, f.QuestionID, formatDecisionSpan(f), f.Subject)
	fmt.Fprintf(w, "- **Outcome**: `%s`\n", f.Outcome)
	fmt.Fprintf(w, "- **Why**: %s\n", f.Why)
	fmt.Fprintf(w, "- **Fix**: %s\n\n", f.Fix)
	fmt.Fprintln(w, "</details>")
	fmt.Fprintln(w)
}

func decisionMarkdownVerdict(rep *rule.Report) string {
	dec := rep.Decisions
	if dec.Skipped {
		return "⚪ SKIPPED"
	}
	if len(dec.Findings) == 0 {
		return "✅ PASS"
	}
	if !rep.Advisory && dec.HasBlockingFindings() {
		return fmt.Sprintf("❌ FAIL (%d blocking, %d advisory)", dec.BlockingCount(), dec.AdvisoryCount())
	}
	return fmt.Sprintf("⚠️ ADVISORY (%d finding(s))", len(dec.Findings))
}

func formatDecisionTag(f rule.DecisionItemFinding, fallbackIdx int, style gotestutil.TermStyle) (string, string) {
	tag := fmt.Sprintf("[D%d]", indexOrFallback(f.Index, fallbackIdx))
	if f.Blocking {
		return tag, style.BoldRed(tag)
	}
	return tag, style.BoldYellow(tag)
}

func formatDecisionSeverity(blocking bool, style gotestutil.TermStyle) (string, string) {
	if blocking {
		return "BLOCKING", style.Red("BLOCKING")
	}
	return "ADVISORY", style.Yellow("ADVISORY")
}

func formatDecisionLocation(f rule.DecisionItemFinding) string {
	return fmt.Sprintf("%s · %s", formatDecisionSpan(f), f.Subject)
}

func formatDecisionSpan(f rule.DecisionItemFinding) string {
	if f.EndLine > f.Line {
		return fmt.Sprintf("%s:%d-%d", f.File, f.Line, f.EndLine)
	}
	return fmt.Sprintf("%s:%d", f.File, max(f.Line, 1))
}
