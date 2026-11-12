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
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const maxGitHubMutantDiffs = 25

// FileMutationSummary holds aggregated mutation counts for a single modified Go file.
type FileMutationSummary struct {
	File       string
	Killed     int
	TimedOut   int
	Survived   int
	Uncovered  int
	Suppressed int
	Skipped    int
	Mutants    []*Mutant
}

// EffectiveKilled returns the total count of killed and timed-out mutants.
func (f *FileMutationSummary) EffectiveKilled() int {
	return f.Killed + f.TimedOut
}

// Tested returns the count of reached mutants evaluated by tests (killed + timed out + survived).
func (f *FileMutationSummary) Tested() int {
	return f.EffectiveKilled() + f.Survived
}

// ScorePercent returns the file's mutation score in [0, 100], or 100 when Tested() == 0.
func (f *FileMutationSummary) ScorePercent() float64 {
	tested := f.Tested()
	if tested == 0 {
		return 100.0
	}
	return 100.0 * float64(f.EffectiveKilled()) / float64(tested)
}

// HasCandidates reports whether the file has any non-compile-skipped mutants.
func (f *FileMutationSummary) HasCandidates() bool {
	return f.Tested() > 0 || f.Uncovered > 0 || f.Suppressed > 0 || f.Skipped > 0
}

// MutationReport holds the complete results of a differential mutation testing run.
type MutationReport struct {
	BaseLabel            string
	TotalKilled          int
	TotalTimedOut        int
	TotalSurvived        int
	TotalUncovered       int
	TotalSuppressed      int
	SuppressedByCategory map[SuppressionCategory]int
	TotalSkipped         int
	Files                []FileMutationSummary
	Warnings             []DirectiveWarning
	RedundantTests       []RedundantTestFinding
}

// EffectiveKilled returns the total count of killed and timed-out mutants across all files.
func (r *MutationReport) EffectiveKilled() int {
	return r.TotalKilled + r.TotalTimedOut
}

// Tested returns the total count of reached mutants evaluated across all files.
func (r *MutationReport) Tested() int {
	return r.EffectiveKilled() + r.TotalSurvived
}

// ScorePercent returns the overall differential mutation score in [0, 100].
func (r *MutationReport) ScorePercent() float64 {
	tested := r.Tested()
	if tested == 0 {
		return 100.0
	}
	return 100.0 * float64(r.EffectiveKilled()) / float64(tested)
}

// HasCandidates reports whether any mutant candidates or suppressions were found in the diff.
func (r *MutationReport) HasCandidates() bool {
	return len(r.Files) > 0
}

// SurvivingMutants returns all mutants with Status == MutantSurvived in deterministic file/line order.
func (r *MutationReport) SurvivingMutants() []*Mutant {
	var out []*Mutant
	for i := range r.Files {
		for _, m := range r.Files[i].Mutants {
			if m.Status == MutantSurvived {
				out = append(out, m)
			}
		}
	}
	return out
}

// BuildMutationReport aggregates per-file mutation plans, warnings, and redundant test findings into a MutationReport.
func BuildMutationReport(
	baseLabel string,
	plans []*FilePlan,
	warnings []DirectiveWarning,
	redundantTests []RedundantTestFinding,
) *MutationReport {
	report := &MutationReport{
		BaseLabel:            baseLabel,
		SuppressedByCategory: make(map[SuppressionCategory]int),
		Warnings:             warnings,
		RedundantTests:       redundantTests,
	}

	sortedPlans := make([]*FilePlan, len(plans))
	copy(sortedPlans, plans)
	slices.SortFunc(sortedPlans, func(a, b *FilePlan) int {
		return cmp.Compare(a.File, b.File)
	})

	for _, p := range sortedPlans {
		if len(p.Mutants) == 0 {
			continue
		}
		fs := FileMutationSummary{
			File:    p.File,
			Mutants: p.Mutants,
		}
		for _, m := range p.Mutants {
			switch m.Status {
			case MutantKilled:
				fs.Killed++
				report.TotalKilled++
			case MutantTimedOut:
				fs.TimedOut++
				report.TotalTimedOut++
			case MutantSurvived:
				fs.Survived++
				report.TotalSurvived++
			case MutantUncovered:
				fs.Uncovered++
				report.TotalUncovered++
			case MutantSuppressed:
				fs.Suppressed++
				report.TotalSuppressed++
				if m.SuppressionCategory != "" {
					report.SuppressedByCategory[m.SuppressionCategory]++
				}
			case MutantSkippedBudget:
				fs.Skipped++
				report.TotalSkipped++
			}
		}
		if fs.HasCandidates() {
			report.Files = append(report.Files, fs)
		}
	}
	return report
}

// RenderTextReport writes a terminal-friendly differential mutation report to w.
func RenderTextReport(w io.Writer, report *MutationReport) {
	if !report.HasCandidates() && len(report.Warnings) == 0 {
		fmt.Fprintf(w, "Go Differential Mutation Score (base: %s): no mutable Go production expressions changed.\n", report.BaseLabel)
		return
	}

	suppressedLabel := formatSuppressedSummary(report.TotalSuppressed, report.SuppressedByCategory)
	if report.Tested() == 0 {
		fmt.Fprintf(w, "Go Differential Mutation Score (base: %s): n/a (0 tested, %d uncovered, %s)\n",
			report.BaseLabel, report.TotalUncovered, suppressedLabel)
	} else {
		fmt.Fprintf(w, "Go Differential Mutation Score (base: %s): %.1f%% (%d killed / %d tested, %d uncovered, %s)\n",
			report.BaseLabel, report.ScorePercent(), report.EffectiveKilled(), report.Tested(), report.TotalUncovered, suppressedLabel)
	}

	if len(report.Files) > 0 {
		fmt.Fprintln(w)
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "FILE\tSCORE\tKILLED\tSURVIVED\tUNCOVERED\tSUPPRESSED")
		for i := range report.Files {
			f := &report.Files[i]
			scoreStr := "n/a"
			if f.Tested() > 0 {
				scoreStr = fmt.Sprintf("%.1f%% (%d/%d)", f.ScorePercent(), f.EffectiveKilled(), f.Tested())
			}
			fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\t%d\n",
				f.File, scoreStr, f.EffectiveKilled(), f.Survived, f.Uncovered, f.Suppressed)
		}
		_ = tw.Flush()
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Warnings (invalid NOMUTANTS directives):")
		for _, warn := range report.Warnings {
			fmt.Fprintf(w, "  %s:%d: %s\n", warn.File, warn.Line, warn.Message)
		}
	}

	surviving := report.SurvivingMutants()
	if len(surviving) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Surviving mutants (tests passed despite logic change):")
		for _, m := range surviving {
			fmt.Fprintf(w, "  [M%d] %s %s:%d\n", m.GlobalIndex, m.Operator, m.File, m.Line)
			fmt.Fprintf(w, "    - %s\n", m.OriginalLine)
			fmt.Fprintf(w, "    + %s\n", m.MutatedLine)
		}
	}

	if len(report.RedundantTests) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Potentially redundant tests in diff (subsumed branch polarity & mutant kills):")
		for _, rt := range report.RedundantTests {
			fmt.Fprintf(w, "  [R%d] %s %s:%d %s\n", rt.GlobalIndex, rt.Kind, rt.File, rt.Line, rt.TestName)
			fmt.Fprintf(w, "    -> %s\n", formatRedundantReason(rt))
		}
	}
}

// RenderGitHubAnnotations writes GitHub Actions workflow warning commands for surviving mutants and invalid directives.
func RenderGitHubAnnotations(w io.Writer, report *MutationReport, maxAnnotations int) {
	if maxAnnotations <= 0 {
		return
	}
	emitted := 0

	for _, warn := range report.Warnings {
		if emitted >= maxAnnotations {
			return
		}
		fmt.Fprintf(w, "::warning file=%s,line=%d,title=%s::%s\n",
			gotestutil.EscapeAnnotationProperty(warn.File),
			warn.Line,
			gotestutil.EscapeAnnotationProperty("Invalid NOMUTANTS Directive"),
			gotestutil.EscapeAnnotationData(warn.Message),
		)
		emitted++
	}

	for _, m := range report.SurvivingMutants() {
		if emitted >= maxAnnotations {
			return
		}
		title := fmt.Sprintf("Surviving Mutant in PR Diff (%s)", m.Operator)
		msg := fmt.Sprintf(
			"Unit tests still pass when line %d is mutated from '%s' to '%s'. Prefer deleting redundant branches or strengthening existing assertions over adding mock-heavy boilerplate.",
			m.Line, m.OriginalExpr, m.MutatedExpr,
		)
		fmt.Fprintf(w, "::warning file=%s,line=%d,title=%s::%s\n",
			gotestutil.EscapeAnnotationProperty(m.File),
			m.Line,
			gotestutil.EscapeAnnotationProperty(title),
			gotestutil.EscapeAnnotationData(msg),
		)
		emitted++
	}

	for _, rt := range report.RedundantTests {
		if emitted >= maxAnnotations {
			return
		}
		title := fmt.Sprintf("Potentially Redundant Test in PR Diff (%s)", rt.Kind)
		msg := fmt.Sprintf("Test '%s' %s. Remove pure equivalence-class duplicates unless preserving a distinct regression fixture.", rt.TestName, formatRedundantReason(rt))
		fmt.Fprintf(w, "::warning file=%s,line=%d,title=%s::%s\n",
			gotestutil.EscapeAnnotationProperty(rt.File),
			rt.Line,
			gotestutil.EscapeAnnotationProperty(title),
			gotestutil.EscapeAnnotationData(msg),
		)
		emitted++
	}
}

// RenderGitHubSummary writes a GitHub-Flavored Markdown mutation report suitable for GITHUB_STEP_SUMMARY.
func RenderGitHubSummary(w io.Writer, report *MutationReport) {
	fmt.Fprintln(w, "## Go Differential Mutation Report")
	fmt.Fprintln(w)

	if !report.HasCandidates() && len(report.Warnings) == 0 {
		fmt.Fprintln(w, "No mutable Go production expressions were added or modified in this change.")
		return
	}

	suppressedLabel := formatSuppressedSummary(report.TotalSuppressed, report.SuppressedByCategory)
	if report.Tested() == 0 {
		fmt.Fprintf(w, "- **Mutation Score (Changed Lines)**: `n/a` (0 tested, %d uncovered, %s)\n",
			report.TotalUncovered, suppressedLabel)
	} else {
		fmt.Fprintf(w, "- **Mutation Score (Changed Lines)**: **`%.1f%%`** (%d killed / %d tested)\n",
			report.ScorePercent(), report.EffectiveKilled(), report.Tested())
		fmt.Fprintf(w, "- **Skipped / Filtered**: %d uncovered, %s",
			report.TotalUncovered, suppressedLabel)
		if report.TotalSkipped > 0 {
			fmt.Fprintf(w, ", %d budget-skipped", report.TotalSkipped)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)

	if len(report.Files) > 0 {
		fmt.Fprintln(w, "| File | Mutation Score | Killed | Survived | Uncovered | Suppressed |")
		fmt.Fprintln(w, "| :--- | ---: | ---: | ---: | ---: | ---: |")
		for i := range report.Files {
			f := &report.Files[i]
			scoreCell := "`n/a`"
			if f.Tested() > 0 {
				scoreCell = fmt.Sprintf("`%.1f%%` (%d/%d)", f.ScorePercent(), f.EffectiveKilled(), f.Tested())
			}
			fmt.Fprintf(w, "| `%s` | %s | %d | %d | %d | %d |\n",
				gotestutil.EscapeMarkdownTable(f.File),
				scoreCell,
				f.EffectiveKilled(),
				f.Survived,
				f.Uncovered,
				f.Suppressed,
			)
		}
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "### Invalid `NOMUTANTS` Directives")
		fmt.Fprintln(w)
		for _, warn := range report.Warnings {
			fmt.Fprintf(w, "- `%s:%d`: %s\n", warn.File, warn.Line, warn.Message)
		}
	}

	surviving := report.SurvivingMutants()
	if len(surviving) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "<details>\n")
		fmt.Fprintf(w, "<summary>Surviving Mutants (%d)</summary>\n\n", len(surviving))

		limit := min(len(surviving), maxGitHubMutantDiffs)
		for i := 0; i < limit; i++ {
			m := surviving[i]
			fmt.Fprintf(w, "**`[M%d]` `%s` in `%s:%d`**\n\n", m.GlobalIndex, m.Operator, m.File, m.Line)
			fmt.Fprintln(w, "```diff")
			fmt.Fprintf(w, "- %s\n", m.OriginalLine)
			fmt.Fprintf(w, "+ %s\n", m.MutatedLine)
			fmt.Fprintln(w, "```")
			fmt.Fprintln(w)
		}
		if len(surviving) > limit {
			fmt.Fprintf(w, "_...and %d additional surviving mutants omitted._\n\n", len(surviving)-limit)
		}
		fmt.Fprintln(w, "</details>")
		fmt.Fprintln(w)
		fmt.Fprint(w, remediationGuideMarkdown)
	}

	if len(report.RedundantTests) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "<details>\n")
		fmt.Fprintf(w, "<summary>Potentially Redundant Changed Tests (%d)</summary>\n\n", len(report.RedundantTests))
		fmt.Fprintln(w, "| Test | Location | Kind | Reason |")
		fmt.Fprintln(w, "| :--- | :--- | :--- | :--- |")
		for _, rt := range report.RedundantTests {
			fmt.Fprintf(w, "| `%s` | `%s:%d` | `%s` | %s |\n",
				gotestutil.EscapeMarkdownTable(rt.TestName),
				gotestutil.EscapeMarkdownTable(rt.File),
				rt.Line,
				rt.Kind,
				gotestutil.EscapeMarkdownTable(formatRedundantReason(rt)),
			)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "_Note: Keep cases that document distinct domain regressions or input encodings; remove pure equivalence-class duplicates._")
		fmt.Fprintln(w, "</details>")
	}
}

func formatRedundantReason(rt RedundantTestFinding) string {
	if rt.Kind == RedundantWeakAssertion {
		return fmt.Sprintf("executes changed code with 0 mutant kills and only error-nil checks (already covered by %s)", rt.DominatedBy)
	}
	return fmt.Sprintf("exercises identical branch polarities and kills no additional mutants beyond %s (%d shared kill(s))", rt.DominatedBy, rt.SharedKills)
}

func formatSuppressedSummary(total int, byCat map[SuppressionCategory]int) string {
	if total == 0 || len(byCat) == 0 {
		return fmt.Sprintf("%d suppressed", total)
	}
	var parts []string
	for _, cat := range AllSuppressionCategories {
		if n := byCat[cat]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s: %d", cat, n))
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d suppressed", total)
	}
	return fmt.Sprintf("%d suppressed (%s)", total, strings.Join(parts, ", "))
}

const remediationGuideMarkdown = `> [!TIP]
> **How to Triage Surviving Mutants (Without Test Bloat)**
> 1. **Delete redundant code first**: If flipping or skipping a branch cannot alter observable behavior because an invariant already holds, simplify or delete the production branch. Invariant or unreachable guards must **not** be suppressed with ` + "`NOMUTANTS`" + `.
> 2. **Strengthen existing assertions**: Assert full output structs with ` + "`cmp.Diff`" + ` or add one boundary row to an existing table test rather than introducing mock-heavy tests.
> 3. **Suppress categorized equivalent mutants**: For genuine cases (` + "`equivalent`, `perf`, `observability`, `external`" + `), add ` + "`// NOMUTANTS(<category>): <reason>`" + ` above or on the line.
> 4. **Add a focused unit test**: Only when substantive domain behavior is genuinely unverified.
`
