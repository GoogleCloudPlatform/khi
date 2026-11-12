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
	"sort"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const (
	maxGitHubMutantDiffs     = 25
	maxGitHubUncoveredBlocks = 25
)

// RenderGitHubAnnotations writes GitHub Actions workflow warning commands for uncovered blocks, surviving mutants, invalid directives, and redundant tests.
func RenderGitHubAnnotations(w io.Writer, report *MutationReport, maxAnnotations int, minUncoveredLines int) {
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

	if emitted >= maxAnnotations {
		return
	}
	allSpans := collectAllCoverageSpans(report.Coverage)
	var largeSpans []UncoveredSpan
	for i := range allSpans {
		if allSpans[i].ExecutableLines >= minUncoveredLines {
			largeSpans = append(largeSpans, allSpans[i])
		}
	}
	// Highlight the largest untested code blocks first rather than 1-3 line guards.
	sortSpansBySizeDesc(largeSpans)
	if remain := maxAnnotations - emitted; len(largeSpans) > remain {
		largeSpans = largeSpans[:remain]
	}
	sort.Slice(largeSpans, func(i, j int) bool {
		if largeSpans[i].File != largeSpans[j].File {
			return largeSpans[i].File < largeSpans[j].File
		}
		return largeSpans[i].StartLine < largeSpans[j].StartLine
	})
	for i := range largeSpans {
		sp := &largeSpans[i]
		rangeLabel := formatLineSpan(sp.StartLine, sp.EndLine)
		msg := fmt.Sprintf("Changed Go block (%s, %d executable lines) is not covered by unit tests.", rangeLabel, sp.ExecutableLines)
		fmt.Fprintf(w, "::warning file=%s,line=%d,endLine=%d,title=%s::%s\n",
			gotestutil.EscapeAnnotationProperty(sp.File),
			sp.StartLine,
			sp.EndLine,
			gotestutil.EscapeAnnotationProperty("Uncovered Lines in PR Diff"),
			gotestutil.EscapeAnnotationData(msg),
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
		title := fmt.Sprintf("Redundant Test in PR Diff (%s)", redundantKindBadge(rt.Kind))
		covLoc := formatCoveringLocation(rt.File, rt.DominatedByFile, rt.DominatedByLine)
		msg := fmt.Sprintf("Test '%s' is already covered by '%s' (%s). %s %s",
			rt.TestName,
			rt.DominatedBy,
			covLoc,
			explainSingleRedundantWhy(rt, covLoc),
			explainRedundantFix(rt.Kind, false),
		)
		fmt.Fprintf(w, "::warning file=%s,line=%d,title=%s::%s\n",
			gotestutil.EscapeAnnotationProperty(rt.File),
			rt.Line,
			gotestutil.EscapeAnnotationProperty(title),
			gotestutil.EscapeAnnotationData(msg),
		)
		emitted++
	}
}

// RenderGitHubSummary writes a unified GitHub-Flavored Markdown coverage and mutation report suitable for GITHUB_STEP_SUMMARY.
func RenderGitHubSummary(w io.Writer, report *MutationReport) {
	fmt.Fprintln(w, "## Go Differential Test & Mutation Report")
	fmt.Fprintln(w)

	if !report.HasCandidates() && len(report.Warnings) == 0 && len(report.RedundantTests) == 0 {
		fmt.Fprintln(w, "No executable Go production lines were added or modified in this change.")
		if report.Coverage != nil && report.Coverage.HasOverallCoverage && report.Coverage.OverallStatements > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "- **Overall Go Statement Coverage**: `%.1f%%` (%d / %d statements)\n",
				report.Coverage.OverallPercent(), report.Coverage.OverallCovered, report.Coverage.OverallStatements)
		}
		return
	}

	if report.Coverage != nil {
		if report.Coverage.PatchExecutableLines > 0 {
			fmt.Fprintf(w, "- **Patch Coverage (Changed Lines)**: **`%.1f%%`** (%d / %d executable lines covered)\n",
				report.Coverage.PatchPercent(), report.Coverage.PatchCoveredLines, report.Coverage.PatchExecutableLines)
		}
		if report.Coverage.HasOverallCoverage && report.Coverage.OverallStatements > 0 {
			fmt.Fprintf(w, "- **Overall Go Statement Coverage**: `%.1f%%` (%d / %d statements)\n",
				report.Coverage.OverallPercent(), report.Coverage.OverallCovered, report.Coverage.OverallStatements)
		}
	}

	suppressedLabel := formatSuppressedPhrase(report.TotalSuppressed, report.SuppressedByCategory)
	if report.Tested() == 0 {
		fmt.Fprintf(w, "- **Mutation Score (Changed Lines)**: `n/a` (0 tested, %d uncovered, %s)\n",
			report.TotalUncovered, suppressedLabel)
	} else {
		fmt.Fprintf(w, "- **Mutation Score (Changed Lines)**: **`%.1f%%`** (%d killed / %d tested)\n",
			report.ScorePercent(), report.EffectiveKilled(), report.Tested())
		fmt.Fprintf(w, "- **Skipped / Filtered Mutants**: %d uncovered, %s",
			report.TotalUncovered, suppressedLabel)
		if report.TotalSkipped > 0 {
			fmt.Fprintf(w, ", %d budget-skipped", report.TotalSkipped)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)

	if len(report.Files) > 0 {
		hasCov := report.Coverage != nil
		if hasCov {
			fmt.Fprintln(w, "| File | Patch Coverage | Mutation Score | Killed | Survived | Suppressed | Uncovered Lines |")
			fmt.Fprintln(w, "| :--- | ---: | ---: | ---: | ---: | ---: | :--- |")
		} else {
			fmt.Fprintln(w, "| File | Mutation Score | Killed | Survived | Uncovered | Suppressed | Uncovered Lines |")
			fmt.Fprintln(w, "| :--- | ---: | ---: | ---: | ---: | ---: | :--- |")
		}
		for i := range report.Files {
			f := &report.Files[i]
			mutStr := "`n/a`"
			if f.Tested() > 0 {
				mutStr = fmt.Sprintf("`%.1f%%` (%d/%d)", f.ScorePercent(), f.EffectiveKilled(), f.Tested())
			}
			uncovCell := "-"
			if ranges := f.UncoveredLineRanges(); ranges != "-" {
				uncovCell = "`" + gotestutil.EscapeMarkdownTable(ranges) + "`"
			}
			if hasCov {
				patchStr := "`n/a`"
				if f.LineCoverage != nil && f.LineCoverage.ChangedExecutableLines > 0 {
					patchStr = fmt.Sprintf("`%.1f%%` (%d/%d)",
						f.LineCoverage.PatchPercent(),
						f.LineCoverage.ChangedCoveredLines,
						f.LineCoverage.ChangedExecutableLines,
					)
				}
				fmt.Fprintf(w, "| `%s` | %s | %s | %d | %d | %d | %s |\n",
					gotestutil.EscapeMarkdownTable(f.File),
					patchStr,
					mutStr,
					f.EffectiveKilled(),
					f.Survived,
					f.Suppressed,
					uncovCell,
				)
				continue
			}
			fmt.Fprintf(w, "| `%s` | %s | %d | %d | %d | %d | %s |\n",
				gotestutil.EscapeMarkdownTable(f.File),
				mutStr,
				f.EffectiveKilled(),
				f.Survived,
				f.Uncovered,
				f.Suppressed,
				uncovCell,
			)
		}
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "> [!WARNING]")
		fmt.Fprintln(w, "> **Invalid `NOMUTANTS` Directives**")
		for _, warn := range report.Warnings {
			fmt.Fprintf(w, "> - `%s:%d`: %s\n", warn.File, warn.Line, warn.Message)
		}
	}

	spans := collectAllCoverageSpans(report.Coverage)
	if len(spans) > 0 {
		sortSpansBySizeDesc(spans)
		fmt.Fprintln(w)
		fmt.Fprintf(w, "<details>\n<summary>Uncovered Changed Code Snippets (%d)</summary>\n\n", len(spans))
		limit := len(spans)
		if limit > maxGitHubUncoveredBlocks {
			limit = maxGitHubUncoveredBlocks
		}
		for i := 0; i < limit; i++ {
			sp := &spans[i]
			fmt.Fprintf(w, "**`%s:%s`** (%d executable lines)\n\n", sp.File, formatLineSpan(sp.StartLine, sp.EndLine), sp.ExecutableLines)
			fmt.Fprintln(w, "```go")
			for _, ln := range sp.Lines {
				fmt.Fprintf(w, "%4d | %s\n", ln.Number, ln.Text)
			}
			fmt.Fprintln(w, "```")
			fmt.Fprintln(w)
		}
		if len(spans) > limit {
			fmt.Fprintf(w, "_...and %d additional smaller uncovered spans omitted._\n\n", len(spans)-limit)
		}
		fmt.Fprintln(w, "</details>")
	}

	surviving := report.SurvivingMutants()
	if len(surviving) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "<details>\n<summary>Surviving Mutants (%d)</summary>\n\n", len(surviving))
		limit := len(surviving)
		if limit > maxGitHubMutantDiffs {
			limit = maxGitHubMutantDiffs
		}
		for i := 0; i < limit; i++ {
			m := surviving[i]
			fmt.Fprintf(w, "**`[M%d]` `%s` (%s) in `%s:%d`**\n\n", m.GlobalIndex, m.Operator, m.Operator.Label(), m.File, m.Line)
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
		fmt.Fprintf(w, "<summary>Redundant Tests in Diff (%d)</summary>\n\n", len(report.RedundantTests))
		fmt.Fprintln(w, "| Added/Changed Test | Already Covered By | Why It Is Redundant |")
		fmt.Fprintln(w, "| :--- | :--- | :--- |")
		for _, rt := range report.RedundantTests {
			covLoc := formatCoveringLocation(rt.File, rt.DominatedByFile, rt.DominatedByLine)
			covCell := fmt.Sprintf("`%s` (%s)", gotestutil.EscapeMarkdownTable(rt.DominatedBy), gotestutil.EscapeMarkdownTable(covLoc))
			whyCell := fmt.Sprintf("**%s** — %s", redundantKindBadge(rt.Kind), gotestutil.EscapeMarkdownTable(explainSingleRedundantWhy(rt, covLoc)))
			fmt.Fprintf(w, "| `%s` (`%s:%d`) | %s | %s |\n",
				gotestutil.EscapeMarkdownTable(rt.TestName),
				gotestutil.EscapeMarkdownTable(rt.File),
				rt.Line,
				covCell,
				whyCell,
			)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "_Tip: Keep cases that document distinct domain regressions or input encodings; delete duplicate equivalence-class cases._")
		fmt.Fprintln(w, "</details>")
	}
}

const remediationGuideMarkdown = `> [!TIP]
> **How to Triage Surviving Mutants (Without Test Bloat)**
> 1. **Delete redundant code first**: If flipping or skipping a branch cannot alter observable behavior because an invariant already holds, simplify or delete the production branch. Invariant or unreachable guards must **not** be suppressed with ` + "`NOMUTANTS`" + `.
> 2. **Strengthen existing assertions**: Assert full output structs with ` + "`cmp.Diff`" + ` or add one boundary row to an existing table test rather than introducing mock-heavy tests.
> 3. **Suppress categorized equivalent mutants**: For genuine cases (` + "`equivalent`, `perf`, `observability`, `external`" + `), add ` + "`// NOMUTANTS(<category>): <reason>`" + ` above or on the line.
> 4. **Add a focused unit test**: Only when substantive domain behavior is genuinely unverified.
`
