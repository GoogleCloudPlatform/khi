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
	"slices"
	"sort"
	"strconv"
	"strings"
)

// FileMutationSummary holds aggregated coverage and mutation counts for a single modified Go file.
type FileMutationSummary struct {
	File         string
	Killed       int
	TimedOut     int
	Survived     int
	Uncovered    int
	Suppressed   int
	Skipped      int
	Mutants      []*Mutant
	LineCoverage *FileCoverageReport
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

// HasCandidates reports whether the file has any changed executable lines or non-compile-skipped mutants.
func (f *FileMutationSummary) HasCandidates() bool {
	if f.LineCoverage != nil && f.LineCoverage.ChangedExecutableLines > 0 {
		return true
	}
	return f.Tested() > 0 || f.Uncovered > 0 || f.Suppressed > 0 || f.Skipped > 0
}

// UncoveredLineRanges returns compact 1-based line ranges (e.g. "42-44, 58" or "-") combining line coverage and uncovered mutants.
func (f *FileMutationSummary) UncoveredLineRanges() string {
	seen := make(map[int]struct{})
	var lines []int
	if f.LineCoverage != nil {
		for _, rg := range f.LineCoverage.UncoveredRanges {
			for ln := rg.StartLine; ln <= rg.EndLine; ln++ {
				if _, ok := seen[ln]; !ok {
					seen[ln] = struct{}{}
					lines = append(lines, ln)
				}
			}
		}
	}
	for _, m := range f.Mutants {
		if m.Status == MutantUncovered && m.Line > 0 {
			if _, ok := seen[m.Line]; !ok {
				seen[m.Line] = struct{}{}
				lines = append(lines, m.Line)
			}
		}
	}
	slices.Sort(lines)
	return collapseIntRanges(lines)
}

func collapseIntRanges(sortedLines []int) string {
	if len(sortedLines) == 0 {
		return "-"
	}
	var parts []string
	start := sortedLines[0]
	end := sortedLines[0]
	for _, ln := range sortedLines[1:] {
		if ln == end+1 {
			end = ln
			continue
		}
		parts = append(parts, formatLineSpan(start, end))
		start = ln
		end = ln
	}
	parts = append(parts, formatLineSpan(start, end))
	return strings.Join(parts, ", ")
}

func formatLineSpan(start int, end int) string {
	if start == end {
		return strconv.Itoa(start)
	}
	return fmt.Sprintf("%d-%d", start, end)
}

// MutationReport holds the complete unified results of differential coverage, mutation testing, and redundant test detection.
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
	Coverage             *CoverageReport
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

// HasCandidates reports whether any changed executable Go lines, mutant candidates, or suppressions were found.
func (r *MutationReport) HasCandidates() bool {
	if r.Coverage != nil && r.Coverage.PatchExecutableLines > 0 {
		return true
	}
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

// AttachLineCoverage merges differential line coverage metrics (including 0-mutant Go files) into report.
func AttachLineCoverage(report *MutationReport, cov *CoverageReport) {
	if report == nil || cov == nil {
		return
	}
	report.Coverage = cov

	idxByFile := make(map[string]int, len(report.Files))
	for i := range report.Files {
		idxByFile[report.Files[i].File] = i
	}

	addedNew := false
	for i := range cov.Files {
		fc := &cov.Files[i]
		if existingIdx, ok := idxByFile[fc.File]; ok {
			report.Files[existingIdx].LineCoverage = fc
			continue
		}
		report.Files = append(report.Files, FileMutationSummary{
			File:         fc.File,
			LineCoverage: fc,
		})
		addedNew = true
	}

	if addedNew {
		slices.SortFunc(report.Files, func(a, b FileMutationSummary) int {
			return cmp.Compare(a.File, b.File)
		})
	}
}

func redundantKindBadge(kind RedundantTestKind) string {
	if kind == RedundantWeakAssertion {
		return "NO OUTPUT CHECK"
	}
	return "SAME PATH"
}

func trimSharedTopFunc(fullName string, sharedTop string) string {
	if sharedTop != "" && strings.HasPrefix(fullName, sharedTop+"/") {
		return strings.TrimPrefix(fullName, sharedTop+"/")
	}
	return fullName
}

func formatCoveringLocation(candFile string, domFile string, domLine int) string {
	if domFile != "" && domFile != candFile {
		if domLine > 0 {
			return fmt.Sprintf("%s:%d", domFile, domLine)
		}
		return domFile
	}
	if domLine > 0 {
		return fmt.Sprintf("line %d", domLine)
	}
	return "existing test"
}

func explainSingleRedundantWhy(rt RedundantTestFinding, covLoc string) string {
	if rt.Kind == RedundantWeakAssertion {
		return fmt.Sprintf("Only checks 'err != nil' without asserting return values or state changes (catches 0 bugs), on a path already tested at %s.", covLoc)
	}
	if rt.DominatorKills > rt.CandidateKills && rt.DominatorKills > 0 {
		return fmt.Sprintf("Runs the exact same if/else path, and catches fewer logic bugs (%d vs %d) than %s.", rt.CandidateKills, rt.DominatorKills, covLoc)
	}
	return fmt.Sprintf("Runs the exact same code path and checks the exact same %d logic condition(s) as %s (no new branches tested).", rt.CandidateKills, covLoc)
}

func explainRedundantFix(kind RedundantTestKind, plural bool) string {
	if kind == RedundantWeakAssertion {
		if plural {
			return "Assert returned outputs/state changes or delete these smoke tests."
		}
		return "Assert the returned output/state or delete this smoke test."
	}
	if plural {
		return "Delete duplicate cases unless keeping a distinct regression input."
	}
	return "Delete this duplicate case unless it documents a distinct regression input."
}

func collectAllCoverageSpans(cov *CoverageReport) []UncoveredSpan {
	if cov == nil {
		return nil
	}
	var all []UncoveredSpan
	for i := range cov.Files {
		all = append(all, cov.Files[i].UncoveredSpans...)
	}
	return all
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

func formatSuppressedSummary(total int, byCat map[SuppressionCategory]int) string {
	cats := formatSuppressedCategories(byCat)
	if total == 0 || cats == "" {
		return strconv.Itoa(total)
	}
	return fmt.Sprintf("%d (%s)", total, cats)
}

func formatSuppressedPhrase(total int, byCat map[SuppressionCategory]int) string {
	cats := formatSuppressedCategories(byCat)
	if total == 0 || cats == "" {
		return fmt.Sprintf("%d suppressed", total)
	}
	return fmt.Sprintf("%d suppressed (%s)", total, cats)
}

func formatSuppressedCategories(byCat map[SuppressionCategory]int) string {
	if len(byCat) == 0 {
		return ""
	}
	var parts []string
	for _, cat := range AllSuppressionCategories {
		if n := byCat[cat]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s: %d", cat, n))
		}
	}
	return strings.Join(parts, ", ")
}
