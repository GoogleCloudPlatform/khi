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

package rule

const (
	// RedundantSamePath marks a subtest with an identical branch-polarity path whose kills are subsumed by another test.
	RedundantSamePath RedundantKind = "SAME PATH"
	// RedundantNoOutputCheck marks an error-nil-only subtest with 0 kills on a path already asserted by another test.
	RedundantNoOutputCheck RedundantKind = "NO OUTPUT CHECK"
)

// Hotspot records a single line's contribution to cognitive complexity.
type Hotspot struct {
	// Line is the 1-based source line number of the complexity contributor.
	Line int
	// Delta is the cognitive complexity increment added at Line.
	Delta int
	// Summary describes the construct causing the complexity increment.
	Summary string
}

// SourceLine pairs a 1-based line number with its raw source text for snippet rendering.
type SourceLine struct {
	// Number is the 1-based source line number.
	Number int
	// Text is the raw source line text.
	Text string
}

// UncoveredSpan represents a contiguous span of substantive uncovered lines inside a touched function.
type UncoveredSpan struct {
	// File is the slash-separated relative path containing the span.
	File string
	// StartLine is the 1-based starting line of the uncovered span.
	StartLine int
	// EndLine is the 1-based ending line of the uncovered span.
	EndLine int
	// ExecutableLines is the count of substantive uncovered lines in the span.
	ExecutableLines int
	// Lines holds surrounding source lines for report snippet rendering.
	Lines []SourceLine
}

// MutantDetail holds diff and operator metadata for a single surviving mutant inside a touched function.
type MutantDetail struct {
	// Operator identifies the mutation operator category.
	Operator MutantOperator
	// Line is the 1-based source line of the mutation.
	Line int
	// OriginalExpr is the compact original expression text.
	OriginalExpr string
	// MutatedExpr is the compact mutated expression text.
	MutatedExpr string
	// OriginalLine is the trimmed original source line.
	OriginalLine string
	// MutatedLine is the rendered mutated source line.
	MutatedLine string
}

// RedundantKind classifies why a touched test or subtest was flagged by test_redundancy.
type RedundantKind string

// RedundantDetail describes one subsumed test or subtest inside a touched Test* function.
type RedundantDetail struct {
	// Kind classifies whether the test was dominated on the same path or lacked output checks.
	Kind RedundantKind
	// Line is the 1-based source line of the redundant test or table row.
	Line int
	// TestName is the redundant test or subtest name.
	TestName string
	// DominatedBy is the name of the dominating test or subtest.
	DominatedBy string
	// DominatedByFile is the file containing the dominating test.
	DominatedByFile string
	// DominatedByLine is the 1-based line number of the dominating test.
	DominatedByLine int
	// CandidateKills is the number of mutants killed by the candidate test.
	CandidateKills int
	// DominatorKills is the number of mutants killed by the dominating test.
	DominatorKills int
}

// Finding represents a single blocking quality rule violation (HARD_LIMIT or RATCHET_REGRESSION).
type Finding struct {
	// GlobalIndex is the 1-based report-wide index assigned during final sorting.
	GlobalIndex int
	// Rule is the violated RuleID.
	Rule RuleID
	// Kind classifies the violation as HARD_LIMIT or RATCHET_REGRESSION.
	Kind ViolationKind
	// File is the slash-separated relative path of the finding.
	File string
	// Line is the 1-based primary source line of the finding.
	Line int
	// EndLine is the 1-based ending source line of the finding scope.
	EndLine int
	// Subject identifies the function, file, or test unit associated with the finding.
	Subject string
	// Before is the measured metric value at the base revision.
	Before int
	// HasBefore reports whether a base-revision measurement was available.
	HasBefore bool
	// After is the measured metric value at the target revision.
	After int
	// Limit is the configured rule budget threshold.
	Limit int
	// Detail provides a concise human-readable explanation of the finding.
	Detail string
	// Hotspots lists complexity hotspots for complexity findings.
	Hotspots []Hotspot
	// UncoveredSpans lists uncovered line spans for coverage findings.
	UncoveredSpans []UncoveredSpan
	// SurvivingMutants lists surviving mutant details for mutation findings.
	SurvivingMutants []MutantDetail
	// RedundantTests lists redundant subtest details for test redundancy findings.
	RedundantTests []RedundantDetail
}

// SuppressedItem records a HARD_LIMIT violation legitimately suppressed via CODEQUALITY_ALLOW.
type SuppressedItem struct {
	// Rule is the suppressed RuleID.
	Rule RuleID
	// File is the slash-separated relative path containing the suppression.
	File string
	// Line is the 1-based line number of the suppressed unit.
	Line int
	// Subject identifies the suppressed function or unit.
	Subject string
	// Category is the justification AllowCategory.
	Category AllowCategory
	// Reason is the human-written explanation for the suppression.
	Reason string
	// After is the measured metric value at the target revision.
	After int
	// Limit is the rule budget threshold that was exceeded.
	Limit int
}

// DirectiveError describes a malformed, disallowed, or unused CODEQUALITY_ALLOW directive.
type DirectiveError struct {
	// File is the slash-separated relative path containing the directive.
	File string
	// Line is the 1-based line number of the directive comment.
	Line int
	// Message describes the directive syntax or validation error.
	Message string
}

// CoverageSummary holds informational patch and overall statement coverage totals for dashboard display.
type CoverageSummary struct {
	// Enabled reports whether coverage evaluation ran.
	Enabled bool
	// PatchCovered is the count of covered changed executable lines.
	PatchCovered int
	// PatchExecutable is the total count of changed executable lines.
	PatchExecutable int
	// HasOverall reports whether full-repository overall coverage was measured.
	HasOverall bool
	// OverallCovered is the total count of covered statements across the repository.
	OverallCovered int
	// OverallStatements is the total statement count across the repository.
	OverallStatements int
	// BaseOverlayRuns is the number of lazy base-overlay coverage runs executed.
	BaseOverlayRuns int
}

// MutationSummary holds aggregate mutant evaluation counts for dashboard display.
type MutationSummary struct {
	// Enabled reports whether mutation testing ran.
	Enabled bool
	// Killed is the count of mutants detected by test failures.
	Killed int
	// TimedOut is the count of mutants that timed out (counted as detected).
	TimedOut int
	// Survived is the count of mutants that passed all tests.
	Survived int
	// Uncovered is the count of mutants never reached by tests.
	Uncovered int
	// Suppressed is the count of mutants suppressed by CODEQUALITY_ALLOW.
	Suppressed int
	// SkippedBudget is the count of mutants skipped due to budget or timeout limits.
	SkippedBudget int
	// CompileSkipped is the count of mutants dropped during compile bisect fallback.
	CompileSkipped int
	// BaseOverlayRuns is the number of lazy base-overlay mutation runs executed.
	BaseOverlayRuns int
	// ByCategory counts suppressed mutants grouped by AllowCategory.
	ByCategory map[AllowCategory]int
}

// Report aggregates all static and dynamic rule results for a single language ("Go" or "TypeScript") in a codequality run.
type Report struct {
	// Language is the target language label ("Go" or "TypeScript").
	Language string
	// BaseLabel describes the base VCS revision used for differential comparison.
	BaseLabel string
	// FastMode reports whether only fast Stage-1 AST checks ran.
	FastMode bool
	// Advisory reports whether findings are non-blocking warnings.
	Advisory bool
	// ProdFilesChanged is the count of modified production files evaluated.
	ProdFilesChanged int
	// TestFilesChanged is the count of modified test files evaluated.
	TestFilesChanged int
	// FuncsEvaluated is the count of functions or units evaluated.
	FuncsEvaluated int
	// Coverage holds differential and overall coverage summary totals.
	Coverage CoverageSummary
	// Mutation holds aggregate mutation testing counts.
	Mutation MutationSummary
	// Findings lists all rule findings in deterministic order.
	Findings []Finding
	// Suppressed lists all findings suppressed via CODEQUALITY_ALLOW.
	Suppressed []SuppressedItem
	// DirectiveErrors lists invalid or unused CODEQUALITY_ALLOW directive errors.
	DirectiveErrors []DirectiveError
}

// PatchPercent returns the percentage of changed executable lines covered across the diff.
func (c CoverageSummary) PatchPercent() float64 {
	if c.PatchExecutable == 0 {
		return 100.0
	}
	return float64(c.PatchCovered) * 100.0 / float64(c.PatchExecutable)
}

// OverallPercent returns total statement coverage percentage when --full or --coverprofile is provided.
func (c CoverageSummary) OverallPercent() float64 {
	if c.OverallStatements == 0 {
		return 0.0
	}
	return float64(c.OverallCovered) * 100.0 / float64(c.OverallStatements)
}

// DetectedCount returns the number of mutants detected by tests (killed + timed out).
func (m MutationSummary) DetectedCount() int {
	return m.Killed + m.TimedOut
}

// TotalEvaluated returns the denominator of reached and executed mutants (detected + survived).
func (m MutationSummary) TotalEvaluated() int {
	return m.DetectedCount() + m.Survived
}

// Score returns the percentage of evaluated mutants detected by tests (100 when 0 evaluated).
func (m MutationSummary) Score() float64 {
	total := m.TotalEvaluated()
	if total == 0 {
		return 100.0
	}
	return float64(m.DetectedCount()) * 100.0 / float64(total)
}

// LanguageTitle returns the display language name, defaulting to "Go".
func (r *Report) LanguageTitle() string {
	if r.Language != "" {
		return r.Language
	}
	return "Go"
}

// CodeFence returns the Markdown syntax-highlighting language tag ("ts" or "go").
func (r *Report) CodeFence() string {
	if r.Language == "TypeScript" {
		return "ts"
	}
	return "go"
}

// HasIssues reports whether the report contains any findings or directive errors.
func (r *Report) HasIssues() bool {
	return len(r.Findings) > 0 || len(r.DirectiveErrors) > 0
}

// HasBlockingIssues reports whether the report contains blocking rule violations or directive errors.
func (r *Report) HasBlockingIssues() bool {
	return !r.Advisory && r.HasIssues()
}
