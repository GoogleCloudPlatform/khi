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

// SimilarityUnitRef identifies an extracted callable unit and its source preview for similarity reporting.
type SimilarityUnitRef struct {
	// FilePath is the slash-separated path relative to the repository root.
	FilePath string
	// StartLine is the 1-based starting line of the unit declaration.
	StartLine int
	// EndLine is the 1-based ending line of the unit declaration.
	EndLine int
	// Name is the qualified display name of the unit.
	Name string
	// Preview holds leading source lines for terminal display.
	Preview []SourceLine
}

// SimilarityCandidate describes an existing repository unit ranked as structurally similar to a newly added unit.
type SimilarityCandidate struct {
	// Unit identifies the matching existing unit in the corpus.
	Unit SimilarityUnitRef
	// Score is the AST-normalized cosine similarity in [-1, 1].
	Score float64
	// SharedModule reports whether the candidate lives in a shared utility module.
	SharedModule bool
	// Guidance provides an actionable reuse or refactoring recommendation.
	Guidance string
}

// SimilarityAdvisory groups a newly added unit with its top-ranked existing similarity candidates.
type SimilarityAdvisory struct {
	// Index is the 1-based advisory index within the report.
	Index int
	// NewUnit identifies the newly introduced callable unit.
	NewUnit SimilarityUnitRef
	// Candidates lists top-ranked similar corpus units ordered by descending score.
	Candidates []SimilarityCandidate
}

// SimilaritySummary holds non-blocking AST-normalized function similarity metadata and advisories for a Report.
type SimilaritySummary struct {
	// Enabled reports whether similarity evaluation was active for newly added units.
	Enabled bool
	// Skipped reports whether embedding evaluation was skipped (for example due to missing credentials).
	Skipped bool
	// SkipReason explains why similarity evaluation was skipped when Skipped is true.
	SkipReason string
	// NewUnitsEvaluated is the number of newly added substantive units checked.
	NewUnitsEvaluated int
	// CorpusUnitsCount is the total number of same-language production corpus units indexed.
	CorpusUnitsCount int
	// Advisories lists newly added units that exceeded similarity thresholds.
	Advisories []SimilarityAdvisory
}
