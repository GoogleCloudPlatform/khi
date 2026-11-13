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

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type annotationSpec struct {
	level   string
	file    string
	line    int
	title   string
	message string
}

// RenderGitHubAnnotations emits GitHub Actions ::error or ::warning workflow commands up to maxAnnotations.
func RenderGitHubAnnotations(w io.Writer, rep *rule.Report, maxAnnotations int) {
	level := "error"
	if rep.Advisory {
		level = "warning"
	}
	lang := rep.LanguageTitle()
	emitted := 0
	for _, e := range rep.DirectiveErrors {
		if emitted >= maxAnnotations {
			return
		}
		emitAnnotation(w, annotationSpec{level: level, file: e.File, line: e.Line, title: "Invalid CODEQUALITY_ALLOW Directive", message: e.Message})
		emitted++
	}
	for i, f := range rep.Findings {
		if emitted >= maxAnnotations {
			return
		}
		idTag := fmt.Sprintf("[Q%d]", indexOrFallback(f.GlobalIndex, i+1))
		title := fmt.Sprintf("%s Quality %s (%s)", lang, idTag, f.Rule)
		msg := fmt.Sprintf("%s [%s]: %s", f.Subject, f.Kind, FormatBudgetValue(f))
		if f.Detail != "" {
			msg += " — " + f.Detail
		}
		emitAnnotation(w, annotationSpec{level: level, file: f.File, line: f.Line, title: title, message: msg})
		emitted++
	}
}

// RenderGitHubSummary writes a Markdown summary suitable for GITHUB_STEP_SUMMARY.
func RenderGitHubSummary(w io.Writer, rep *rule.Report) {
	verdict := "✅ PASS"
	if rep.HasIssues() {
		if rep.Advisory {
			verdict = fmt.Sprintf("⚠️ ADVISORY (%d finding(s), %d directive error(s))", len(rep.Findings), len(rep.DirectiveErrors))
		} else {
			verdict = fmt.Sprintf("❌ FAIL (%d violation(s), %d directive error(s))", len(rep.Findings), len(rep.DirectiveErrors))
		}
	}
	fmt.Fprintf(w, "## %s Quality Report (`%s`) — %s\n", rep.LanguageTitle(), rep.BaseLabel, verdict)
	fmt.Fprintln(w)
	renderSummaryMetricsTable(w, rep)

	if len(rep.DirectiveErrors) > 0 {
		renderMarkdownDirectiveErrors(w, rep.DirectiveErrors)
	}
	if len(rep.Findings) > 0 {
		renderMarkdownFindings(w, rep.Findings, rep.CodeFence())
	}
}

func emitAnnotation(w io.Writer, a annotationSpec) {
	fmt.Fprintf(
		w,
		"::%s file=%s,line=%d,title=%s::%s\n",
		a.level,
		gotestutil.EscapeAnnotationProperty(a.file),
		max(a.line, 1),
		gotestutil.EscapeAnnotationProperty(a.title),
		gotestutil.EscapeAnnotationData(a.message),
	)
}

func renderSummaryMetricsTable(w io.Writer, rep *rule.Report) {
	fmt.Fprintln(w, "| Metric | Result |")
	fmt.Fprintln(w, "| :--- | :--- |")
	fmt.Fprintf(w, "| Changed Scope | %d prod file(s), %d test file(s), %d unit(s) |\n",
		rep.ProdFilesChanged, rep.TestFilesChanged, rep.FuncsEvaluated)
	if rep.Coverage.Enabled {
		fmt.Fprintf(w, "| Patch Coverage | %.1f%% (%d/%d lines) |\n",
			rep.Coverage.PatchPercent(), rep.Coverage.PatchCovered, rep.Coverage.PatchExecutable)
	}
	if rep.Mutation.Enabled {
		fmt.Fprintf(w, "| Mutation Score | %.1f%% (%d/%d killed, %d survived) |\n",
			rep.Mutation.Score(), rep.Mutation.DetectedCount(), rep.Mutation.TotalEvaluated(), rep.Mutation.Survived)
	}
	fmt.Fprintln(w)
}

func renderMarkdownDirectiveErrors(w io.Writer, errs []rule.DirectiveError) {
	fmt.Fprintln(w, "### Invalid or Unused `CODEQUALITY_ALLOW` Directives")
	fmt.Fprintln(w)
	for _, e := range errs {
		fmt.Fprintf(w, "- `%s:%d` — %s\n", e.File, e.Line, gotestutil.EscapeMarkdownTable(e.Message))
	}
	fmt.Fprintln(w)
}

func renderMarkdownFindings(w io.Writer, findings []rule.Finding, codeFence string) {
	fmt.Fprintln(w, "### Quality Rule Violations")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| # | Rule | Status | Value (Limit) | Location | Subject |")
	fmt.Fprintln(w, "| :--- | :--- | :--- | :--- | :--- | :--- |")
	for i, f := range findings {
		idTag := fmt.Sprintf("`[Q%d]`", indexOrFallback(f.GlobalIndex, i+1))
		loc := fmt.Sprintf("`%s:%d`", f.File, f.Line)
		fmt.Fprintf(w, "| %s | `%s` | `%s` | `%s` | %s | `%s` |\n",
			idTag,
			f.Rule,
			f.Kind,
			gotestutil.EscapeMarkdownTable(FormatBudgetValue(f)),
			loc,
			gotestutil.EscapeMarkdownTable(f.Subject),
		)
	}
	fmt.Fprintln(w)

	for i, f := range findings {
		renderMarkdownFindingDetails(w, i, f, codeFence)
	}
}

func renderMarkdownFindingDetails(w io.Writer, idx int, f rule.Finding, codeFence string) {
	idTag := fmt.Sprintf("[Q%d]", indexOrFallback(f.GlobalIndex, idx+1))
	fmt.Fprintln(w, "<details>")
	fmt.Fprintf(w, "<summary><strong>%s %s</strong> — <code>%s:%d</code> (<code>%s</code>)</summary>\n",
		idTag, f.Rule, f.File, f.Line, f.Subject)
	fmt.Fprintln(w)
	if spec, ok := rule.LookupRuleSpec(f.Rule); ok {
		fmt.Fprintf(w, "- **Why**: %s\n", spec.Why)
		fmt.Fprintf(w, "- **Fix**: %s\n", spec.Fix)
	}
	if f.Detail != "" {
		fmt.Fprintf(w, "- **Detail**: %s\n", f.Detail)
	}
	for _, m := range f.SurvivingMutants {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "```diff")
		fmt.Fprintf(w, "- %s\n", m.OriginalLine)
		fmt.Fprintf(w, "+ %s\n", m.MutatedLine)
		fmt.Fprintln(w, "```")
	}
	for _, sp := range f.UncoveredSpans {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "```%s\n", codeFence)
		for _, ln := range sp.Lines {
			fmt.Fprintf(w, "%4d | %s\n", ln.Number, ln.Text)
		}
		fmt.Fprintln(w, "```")
	}
	for _, rt := range f.RedundantTests {
		fmt.Fprintf(w, "- `[%s]` `%s` (L%d) dominated by `%s` (`%s:%d`)\n",
			rt.Kind, rt.TestName, rt.Line, rt.DominatedBy, rt.DominatedByFile, rt.DominatedByLine)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "</details>")
	fmt.Fprintln(w)
}
