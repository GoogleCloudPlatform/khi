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
	"strings"
	"text/tabwriter"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const maxSummarySnippets = 25

// RenderTextReport writes a human- and agent-friendly differential coverage summary to w.
func RenderTextReport(w io.Writer, report *CoverageReport) {
	if report.PatchExecutableLines == 0 {
		fmt.Fprintf(w, "Go Differential Coverage (base: %s): no executable Go production lines changed.\n", report.BaseLabel)
		if report.HasOverallCoverage && report.OverallStatements > 0 {
			fmt.Fprintf(w, "Overall Go Statement Coverage: %.1f%% (%d/%d statements)\n",
				report.OverallPercent(), report.OverallCovered, report.OverallStatements)
		}
		return
	}

	fmt.Fprintf(w, "Go Differential Coverage (base: %s): %.1f%% (%d/%d executable changed lines covered)\n",
		report.BaseLabel, report.PatchPercent(), report.PatchCoveredLines, report.PatchExecutableLines)
	if report.HasOverallCoverage && report.OverallStatements > 0 {
		fmt.Fprintf(w, "Overall Go Statement Coverage: %.1f%% (%d/%d statements)\n",
			report.OverallPercent(), report.OverallCovered, report.OverallStatements)
	}
	fmt.Fprintln(w)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FILE\tPATCH COV\tFILE COV\tUNCOVERED DIFF LINES")
	for i := range report.Files {
		f := &report.Files[i]
		fmt.Fprintf(tw, "%s\t%.1f%% (%d/%d)\t%.1f%%\t%s\n",
			f.File,
			f.PatchPercent(),
			f.ChangedCoveredLines,
			f.ChangedExecutableLines,
			f.FilePercent(),
			formatLineRanges(f.UncoveredRanges),
		)
	}
	_ = tw.Flush()

	spans := collectAllSpans(report)
	if len(spans) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Uncovered changed code:")
	for i := range spans {
		sp := &spans[i]
		fmt.Fprintf(w, "  %s:%s\n", sp.File, formatSingleRange(sp.StartLine, sp.EndLine))
		for _, ln := range sp.Lines {
			fmt.Fprintf(w, "    %4d | %s\n", ln.Number, ln.Text)
		}
	}
}

// RenderGitHubAnnotations writes GitHub Actions workflow warning commands for substantive uncovered diff spans.
func RenderGitHubAnnotations(w io.Writer, report *CoverageReport, maxAnnotations int, minAnnotationLines int) {
	if maxAnnotations <= 0 {
		return
	}
	if minAnnotationLines < 1 {
		minAnnotationLines = 1
	}

	allSpans := collectAllSpans(report)
	spans := make([]UncoveredSpan, 0, len(allSpans))
	for i := range allSpans {
		if allSpans[i].ExecutableLines >= minAnnotationLines {
			spans = append(spans, allSpans[i])
		}
	}
	if len(spans) == 0 {
		return
	}

	// Highlight the largest untested code blocks first rather than small 1-3 line defensive or error guards.
	sortSpansBySizeDesc(spans)

	if len(spans) > maxAnnotations {
		spans = spans[:maxAnnotations]
	}

	sort.Slice(spans, func(i, j int) bool {
		if spans[i].File != spans[j].File {
			return spans[i].File < spans[j].File
		}
		return spans[i].StartLine < spans[j].StartLine
	})

	for i := range spans {
		sp := &spans[i]
		rangeLabel := formatSingleRange(sp.StartLine, sp.EndLine)
		msg := fmt.Sprintf("Changed Go block (%s, %d executable lines) is not covered by unit tests.", rangeLabel, sp.ExecutableLines)
		fmt.Fprintf(w, "::warning file=%s,line=%d,endLine=%d,title=%s::%s\n",
			gotestutil.EscapeAnnotationProperty(sp.File),
			sp.StartLine,
			sp.EndLine,
			gotestutil.EscapeAnnotationProperty("Uncovered Lines in PR Diff"),
			gotestutil.EscapeAnnotationData(msg),
		)
	}
}

// RenderGitHubSummary writes a GitHub-Flavored Markdown coverage summary suitable for GITHUB_STEP_SUMMARY.
func RenderGitHubSummary(w io.Writer, report *CoverageReport) {
	fmt.Fprintln(w, "## Go Differential Coverage Report")
	fmt.Fprintln(w)

	if report.PatchExecutableLines == 0 {
		fmt.Fprintln(w, "No executable Go production lines were added or modified in this change.")
		if report.HasOverallCoverage && report.OverallStatements > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "- **Overall Go Statement Coverage**: `%.1f%%` (%d / %d statements)\n",
				report.OverallPercent(), report.OverallCovered, report.OverallStatements)
		}
		return
	}

	fmt.Fprintf(w, "- **Patch Coverage (Changed Lines)**: **`%.1f%%`** (%d / %d executable lines covered)\n",
		report.PatchPercent(), report.PatchCoveredLines, report.PatchExecutableLines)
	if report.HasOverallCoverage && report.OverallStatements > 0 {
		fmt.Fprintf(w, "- **Overall Go Statement Coverage**: `%.1f%%` (%d / %d statements)\n",
			report.OverallPercent(), report.OverallCovered, report.OverallStatements)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "| File | Patch Coverage | File Coverage | Uncovered Diff Lines |")
	fmt.Fprintln(w, "| :--- | ---: | ---: | :--- |")
	for i := range report.Files {
		f := &report.Files[i]
		fmt.Fprintf(w, "| `%s` | `%.1f%%` (%d/%d) | `%.1f%%` | %s |\n",
			gotestutil.EscapeMarkdownTable(f.File),
			f.PatchPercent(),
			f.ChangedCoveredLines,
			f.ChangedExecutableLines,
			f.FilePercent(),
			formatMarkdownRanges(f.UncoveredRanges),
		)
	}

	spans := collectAllSpans(report)
	if len(spans) == 0 {
		return
	}

	sortSpansBySizeDesc(spans)

	fmt.Fprintln(w)
	fmt.Fprintf(w, "<details>\n<summary>Uncovered Changed Code Snippets (%d)</summary>\n\n", len(spans))
	limit := len(spans)
	if limit > maxSummarySnippets {
		limit = maxSummarySnippets
	}
	for i := 0; i < limit; i++ {
		sp := &spans[i]
		fmt.Fprintf(w, "**`%s:%s`** (%d executable lines)\n\n", sp.File, formatSingleRange(sp.StartLine, sp.EndLine), sp.ExecutableLines)
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

func sortSpansBySizeDesc(spans []UncoveredSpan) {
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].ExecutableLines != spans[j].ExecutableLines {
			return spans[i].ExecutableLines > spans[j].ExecutableLines
		}
		lenI := spans[i].EndLine - spans[i].StartLine + 1
		lenJ := spans[j].EndLine - spans[j].StartLine + 1
		if lenI != lenJ {
			return lenI > lenJ
		}
		if spans[i].File != spans[j].File {
			return spans[i].File < spans[j].File
		}
		return spans[i].StartLine < spans[j].StartLine
	})
}

func collectAllSpans(report *CoverageReport) []UncoveredSpan {
	var all []UncoveredSpan
	for i := range report.Files {
		all = append(all, report.Files[i].UncoveredSpans...)
	}
	return all
}

func formatLineRanges(ranges []LineRange) string {
	if len(ranges) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(ranges))
	for _, r := range ranges {
		parts = append(parts, formatSingleRange(r.StartLine, r.EndLine))
	}
	return strings.Join(parts, ", ")
}

func formatMarkdownRanges(ranges []LineRange) string {
	if len(ranges) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(ranges))
	for _, r := range ranges {
		parts = append(parts, "`"+formatSingleRange(r.StartLine, r.EndLine)+"`")
	}
	return strings.Join(parts, ", ")
}

func formatSingleRange(startLine int, endLine int) string {
	if startLine == endLine {
		return fmt.Sprintf("%d", startLine)
	}
	return fmt.Sprintf("%d-%d", startLine, endLine)
}
