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

// Hotspot records a single line's contribution to cognitive complexity.
type Hotspot struct {
	Line    int
	Delta   int
	Summary string
}

// SourceLine pairs a 1-based line number with its raw source text for snippet rendering.
type SourceLine struct {
	Number int
	Text   string
}

// UncoveredSpan represents a contiguous span of substantive uncovered lines inside a touched function.
type UncoveredSpan struct {
	File            string
	StartLine       int
	EndLine         int
	ExecutableLines int
	Lines           []SourceLine
}

// MutantDetail holds diff and operator metadata for a single surviving mutant inside a touched function.
type MutantDetail struct {
	Operator     MutantOperator
	Line         int
	OriginalExpr string
	MutatedExpr  string
	OriginalLine string
	MutatedLine  string
}

// RedundantKind classifies why a touched test or subtest was flagged by test_redundancy.
type RedundantKind string

const (
	// RedundantSamePath marks a subtest with an identical branch-polarity path whose kills are subsumed by another test.
	RedundantSamePath RedundantKind = "SAME PATH"
	// RedundantNoOutputCheck marks an error-nil-only subtest with 0 kills on a path already asserted by another test.
	RedundantNoOutputCheck RedundantKind = "NO OUTPUT CHECK"
)

// RedundantDetail describes one subsumed test or subtest inside a touched Test* function.
type RedundantDetail struct {
	Kind            RedundantKind
	Line            int
	TestName        string
	DominatedBy     string
	DominatedByFile string
	DominatedByLine int
	CandidateKills  int
	DominatorKills  int
}

// Finding represents a single blocking quality rule violation (HARD_LIMIT or RATCHET_REGRESSION).
type Finding struct {
	GlobalIndex      int
	Rule             RuleID
	Kind             ViolationKind
	File             string
	Line             int
	EndLine          int
	Subject          string
	Before           int
	HasBefore        bool
	After            int
	Limit            int
	Detail           string
	Hotspots         []Hotspot
	UncoveredSpans   []UncoveredSpan
	SurvivingMutants []MutantDetail
	RedundantTests   []RedundantDetail
}

// SuppressedItem records a HARD_LIMIT violation legitimately suppressed via CODEQUALITY_ALLOW.
type SuppressedItem struct {
	Rule     RuleID
	File     string
	Line     int
	Subject  string
	Category AllowCategory
	Reason   string
	After    int
	Limit    int
}

// DirectiveError describes a malformed, disallowed, or unused CODEQUALITY_ALLOW directive.
type DirectiveError struct {
	File    string
	Line    int
	Message string
}

// CoverageSummary holds informational patch and overall statement coverage totals for dashboard display.
type CoverageSummary struct {
	Enabled           bool
	PatchCovered      int
	PatchExecutable   int
	HasOverall        bool
	OverallCovered    int
	OverallStatements int
	BaseOverlayRuns   int
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

// MutationSummary holds aggregate mutant evaluation counts for dashboard display.
type MutationSummary struct {
	Enabled         bool
	Killed          int
	TimedOut        int
	Survived        int
	Uncovered       int
	Suppressed      int
	SkippedBudget   int
	CompileSkipped  int
	BaseOverlayRuns int
	ByCategory      map[AllowCategory]int
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

// Report aggregates all static and dynamic rule results for a single language ("Go" or "TypeScript") in a codequality run.
type Report struct {
	Language         string
	BaseLabel        string
	FastMode         bool
	Advisory         bool
	ProdFilesChanged int
	TestFilesChanged int
	FuncsEvaluated   int
	Coverage         CoverageSummary
	Mutation         MutationSummary
	Findings         []Finding
	Suppressed       []SuppressedItem
	DirectiveErrors  []DirectiveError
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
