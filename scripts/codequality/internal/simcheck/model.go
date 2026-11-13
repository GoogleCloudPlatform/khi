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

package simcheck

import (
	"context"
	"math"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

const (
	// MaxPreviewLines caps the number of raw source lines attached to each extracted Unit for terminal display.
	MaxPreviewLines = 8

	// KindFunc identifies a standalone or module-level function declaration or arrow function variable.
	KindFunc UnitKind = "func"
	// KindMethod identifies a receiver-bound or class-bound method, accessor, constructor, or arrow property.
	KindMethod UnitKind = "method"
	// KindClosure identifies a module-level or class-property reactive/task closure (such as DefineInspectionTask or computed).
	KindClosure UnitKind = "closure"
)

// TargetReader loads raw source bytes from the current working tree.
type TargetReader func(relPath string) ([]byte, error)

// BaseReader loads raw source bytes from the base revision, returning exists=false when absent at base.
type BaseReader func(relPath string) ([]byte, bool, error)

// Embedder computes dense vector embeddings aligned 1-to-1 with input texts.
type Embedder interface {
	// EmbedBatch returns L2-normalized embedding vectors aligned 1-to-1 with texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// UnitKind classifies the declaration form of an extracted callable unit across languages.
type UnitKind string

// Unit represents a single function, method, or reactive/task closure eligible for similarity indexing.
type Unit struct {
	// ID uniquely identifies the unit by relative file path and qualified name.
	ID string
	// Key identifies the unit within a single file across revisions ("<kind>:<name>").
	Key string
	// Language identifies the source language of the unit.
	Language LanguageID
	// ModulePath is the slash-separated directory or package path owning the unit.
	ModulePath string
	// FilePath is the slash-separated file path relative to the repository root.
	FilePath string
	// StartLine is the 1-based starting line of the unit declaration.
	StartLine int
	// EndLine is the 1-based ending line of the unit declaration.
	EndLine int
	// Name is the qualified display name of the unit.
	Name string
	// Kind classifies the unit as a function, method, or closure.
	Kind UnitKind
	// StmtCount is the number of executable AST statements in the unit body.
	StmtCount int
	// CodeLines is the number of non-blank, non-comment code lines in the unit span.
	CodeLines int
	// NormText holds the AST-normalized source text sent to the embedding model.
	NormText string
	// Preview holds up to MaxPreviewLines of formatted leading source lines.
	Preview []rule.SourceLine
}

// Thresholds configures complexity filters, AST-normalized similarity cutoffs, and per-unit candidate limits.
type Thresholds struct {
	// MinStmts is the minimum executable AST statement count required to index a unit.
	MinStmts int
	// MinCodeLines is the minimum non-comment code line count required to index a unit.
	MinCodeLines int
	// MinScore is the default minimum cosine similarity for cross-module corpus candidates.
	MinScore float64
	// SharedMinScore is the boosted (lower) minimum cosine similarity for shared utility modules.
	SharedMinScore float64
	// SameChangeMinScore is the stricter minimum cosine similarity between two newly added units in the same change.
	SameChangeMinScore float64
	// TopK caps the number of ranked candidates attached to each advisory.
	TopK int
}

// Options configures differential similarity evaluation for newly added units in a single language.
type Options struct {
	// Analyzer provides language-specific file discovery, AST extraction, and normalization.
	Analyzer LanguageAnalyzer
	// Thresholds configures complexity and cosine similarity cutoffs (zero fields use DefaultThresholds).
	Thresholds Thresholds
	// RepoRoot is the repository root path used for automatic corpus discovery when CorpusFiles is empty.
	RepoRoot string
	// ChangedProdLinesByFile maps changed production file paths to 1-based changed line sets.
	ChangedProdLinesByFile map[string]map[int]struct{}
	// CorpusFiles optionally overrides automatic corpus file discovery with an explicit path list.
	CorpusFiles []string
	// ReadTargetFile loads raw source bytes from the target working tree.
	ReadTargetFile TargetReader
	// ReadBaseFile loads raw source bytes from the base revision.
	ReadBaseFile BaseReader
	// Embedder provides pre-initialized vector embedding computation when non-nil.
	Embedder Embedder
	// ResolveEmbedder lazily initializes an Embedder only when newly added units exist.
	ResolveEmbedder func(ctx context.Context) (Embedder, string, error)
}

// CandidateMatch describes an existing repository unit that closely resembles a newly added unit under AST normalization.
type CandidateMatch struct {
	// Unit is the matching existing or peer unit in the corpus.
	Unit Unit
	// Score is the AST-normalized cosine similarity in [-1, 1].
	Score float64
	// SharedModule reports whether Unit resides in a shared utility module.
	SharedModule bool
	// Guidance describes the recommended reuse or consolidation action.
	Guidance string
}

// Advisory groups a newly added unit with its top-ranked existing similarity candidates.
type Advisory struct {
	// Index is the 1-based advisory index within the Result.
	Index int
	// NewUnit is the newly introduced callable unit.
	NewUnit Unit
	// Candidates lists top-ranked matching corpus units ordered by descending score.
	Candidates []CandidateMatch
}

// Result holds non-blocking similarity advisories and execution metadata for a single language.
type Result struct {
	// Language is the canonical short language identifier.
	Language LanguageID
	// LanguageTitle is the human-readable language display name.
	LanguageTitle string
	// CodeFence is the Markdown code fence language tag.
	CodeFence string
	// BaseLabel describes the base VCS revision used for differential comparison.
	BaseLabel string
	// Skipped reports whether embedding evaluation was skipped.
	Skipped bool
	// SkipReason explains why similarity evaluation was skipped when Skipped is true.
	SkipReason string
	// NewUnitsEvaluated is the number of newly added substantive units checked.
	NewUnitsEvaluated int
	// CorpusUnitsCount is the total number of same-language production corpus units indexed.
	CorpusUnitsCount int
	// Advisories lists newly added units that matched existing corpus units above threshold.
	Advisories []Advisory
}

// ContainsAnyLine reports whether any 1-based line in lines falls within [u.StartLine, u.EndLine].
func (u Unit) ContainsAnyLine(lines map[int]struct{}) bool {
	for line := range lines {
		if line >= u.StartLine && line <= u.EndLine {
			return true
		}
	}
	return false
}

// ToRef converts u into a lightweight rule.SimilarityUnitRef for report rendering.
func (u Unit) ToRef() rule.SimilarityUnitRef {
	return rule.SimilarityUnitRef{
		FilePath:  u.FilePath,
		StartLine: u.StartLine,
		EndLine:   u.EndLine,
		Name:      u.Name,
		Preview:   u.Preview,
	}
}

// BuildPreviewLines formats up to MaxPreviewLines starting at startLine with tabs expanded to 4 spaces.
func BuildPreviewLines(declText string, startLine int) []rule.SourceLine {
	rawLines := strings.Split(strings.TrimRight(declText, "\n"), "\n")
	limit := min(len(rawLines), MaxPreviewLines)
	preview := make([]rule.SourceLine, 0, limit)
	for i := range limit {
		preview = append(preview, rule.SourceLine{
			Number: startLine + i,
			Text:   strings.ReplaceAll(rawLines[i], "\t", "    "),
		})
	}
	return preview
}

// DefaultThresholds returns the empirically calibrated thresholds for gemini-embedding-2 over AST-normalized units.
func DefaultThresholds() Thresholds {
	return Thresholds{
		MinStmts:           3,
		MinCodeLines:       6,
		MinScore:           0.940,
		SharedMinScore:     0.915,
		SameChangeMinScore: 0.960,
		TopK:               3,
	}
}

// ToSummary converts r into a rule.SimilaritySummary suitable for attachment to rule.Report.
func (r *Result) ToSummary() rule.SimilaritySummary {
	if r == nil || r.NewUnitsEvaluated == 0 {
		return rule.SimilaritySummary{}
	}
	advisories := make([]rule.SimilarityAdvisory, len(r.Advisories))
	for i, adv := range r.Advisories {
		cands := make([]rule.SimilarityCandidate, len(adv.Candidates))
		for j, c := range adv.Candidates {
			cands[j] = rule.SimilarityCandidate{
				Unit:         c.Unit.ToRef(),
				Score:        c.Score,
				SharedModule: c.SharedModule,
				Guidance:     c.Guidance,
			}
		}
		advisories[i] = rule.SimilarityAdvisory{
			Index:      adv.Index,
			NewUnit:    adv.NewUnit.ToRef(),
			Candidates: cands,
		}
	}
	return rule.SimilaritySummary{
		Enabled:           true,
		Skipped:           r.Skipped,
		SkipReason:        r.SkipReason,
		NewUnitsEvaluated: r.NewUnitsEvaluated,
		CorpusUnitsCount:  r.CorpusUnitsCount,
		Advisories:        advisories,
	}
}

// NormalizeL2 returns a unit-length copy of vec under the Euclidean norm.
func NormalizeL2(vec []float32) []float32 {
	var sum float64
	for _, x := range vec {
		sum += float64(x) * float64(x)
	}
	out := make([]float32, len(vec))
	if sum == 0 {
		return out
	}
	inv := float32(1.0 / math.Sqrt(sum))
	for i, x := range vec {
		out[i] = x * inv
	}
	return out
}

// CosineNormalized returns the dot product of two L2-normalized vectors clamped to [-1, 1].
func CosineNormalized(a []float32, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot float32
	for i := range a {
		dot += a[i] * b[i]
	}
	return max(-1.0, min(1.0, float64(dot)))
}
