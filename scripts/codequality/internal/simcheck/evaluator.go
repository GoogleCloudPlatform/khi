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
	"cmp"
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
)

type rankRequest struct {
	analyzer   LanguageAnalyzer
	embedder   Embedder
	newUnits   []Unit
	corpus     []Unit
	thresholds Thresholds
}

type corpusIndex struct {
	analyzer   LanguageAnalyzer
	units      []Unit
	normVecs   [][]float32
	newIDSet   map[string]bool
	thresholds Thresholds
}

// Evaluate identifies newly added callable units for opts.Analyzer and ranks similar existing units in the same language.
func Evaluate(ctx context.Context, opts Options) (*Result, error) {
	if opts.Analyzer == nil {
		return nil, fmt.Errorf("simcheck: LanguageAnalyzer is required")
	}
	thresholds := normalizeThresholds(opts.Thresholds)
	baseRes := &Result{
		Language:      opts.Analyzer.ID(),
		LanguageTitle: opts.Analyzer.DisplayName(),
		CodeFence:     opts.Analyzer.CodeFence(),
	}

	newUnits, err := collectNewUnits(ctx, opts, thresholds)
	if err != nil {
		return nil, err
	}
	baseRes.NewUnitsEvaluated = len(newUnits)
	if len(newUnits) == 0 {
		return baseRes, nil
	}

	embedder, skipReason, err := obtainEmbedder(ctx, opts)
	if err != nil {
		baseRes.Skipped = true
		baseRes.SkipReason = err.Error()
		return baseRes, nil
	}
	if embedder == nil {
		if skipReason == "" {
			skipReason = "embedding client not configured"
		}
		baseRes.Skipped = true
		baseRes.SkipReason = skipReason
		return baseRes, nil
	}

	corpus, err := collectCorpusUnits(ctx, opts, thresholds)
	if err != nil {
		return nil, err
	}
	advisories, err := rankSimilarCandidates(ctx, rankRequest{
		analyzer:   opts.Analyzer,
		embedder:   embedder,
		newUnits:   newUnits,
		corpus:     corpus,
		thresholds: thresholds,
	})
	if err != nil {
		baseRes.Skipped = true
		baseRes.SkipReason = fmt.Sprintf("embedding unavailable: %v", err)
		return baseRes, nil
	}
	baseRes.CorpusUnitsCount = len(corpus)
	baseRes.Advisories = advisories
	return baseRes, nil
}

func obtainEmbedder(ctx context.Context, opts Options) (Embedder, string, error) {
	if opts.Embedder != nil {
		return opts.Embedder, "", nil
	}
	if opts.ResolveEmbedder != nil {
		return opts.ResolveEmbedder(ctx)
	}
	return nil, "", nil
}

func normalizeThresholds(th Thresholds) Thresholds {
	def := DefaultThresholds()
	if th.MinStmts <= 0 {
		th.MinStmts = def.MinStmts
	}
	if th.MinCodeLines <= 0 {
		th.MinCodeLines = def.MinCodeLines
	}
	if th.MinScore <= 0 {
		th.MinScore = def.MinScore
	}
	if th.SharedMinScore <= 0 {
		th.SharedMinScore = min(def.SharedMinScore, th.MinScore)
	}
	if th.SameChangeMinScore <= 0 {
		th.SameChangeMinScore = max(def.SameChangeMinScore, th.MinScore)
	}
	if th.TopK <= 0 {
		th.TopK = def.TopK
	}
	return th
}

func collectNewUnits(ctx context.Context, opts Options, th Thresholds) ([]Unit, error) {
	targetBatch, orderedPaths, err := loadChangedTargetFiles(opts)
	if err != nil {
		return nil, err
	}
	if len(targetBatch) == 0 {
		return nil, nil
	}
	unitsByFile, err := opts.Analyzer.ExtractUnits(ctx, targetBatch, th.MinStmts, th.MinCodeLines)
	if err != nil {
		return nil, err
	}

	candidateFiles := filterFilesWithTouchedUnits(orderedPaths, unitsByFile, opts.ChangedProdLinesByFile)
	if len(candidateFiles) == 0 {
		return nil, nil
	}

	baseKeysByFile, err := loadBatchBaseKeys(ctx, opts.Analyzer, candidateFiles, opts.ReadBaseFile)
	if err != nil {
		return nil, err
	}

	var added []Unit
	for _, relPath := range candidateFiles {
		changedLines := opts.ChangedProdLinesByFile[relPath]
		baseKeys := baseKeysByFile[relPath]
		for _, unit := range unitsByFile[relPath] {
			if !unit.ContainsAnyLine(changedLines) {
				continue
			}
			if _, existedAtBase := baseKeys[unit.Key]; existedAtBase {
				continue
			}
			added = append(added, unit)
		}
	}
	return added, nil
}

func loadChangedTargetFiles(opts Options) ([]FileInput, []string, error) {
	var batch []FileInput
	var paths []string
	for _, relPath := range slices.Sorted(maps.Keys(opts.ChangedProdLinesByFile)) {
		if len(opts.ChangedProdLinesByFile[relPath]) == 0 || !opts.Analyzer.IsEligibleFile(relPath) {
			continue
		}
		src, err := opts.ReadTargetFile(relPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, fmt.Errorf("failed to read target file %s: %w", relPath, err)
		}
		batch = append(batch, FileInput{Path: relPath, Content: src})
		paths = append(paths, relPath)
	}
	return batch, paths, nil
}

func filterFilesWithTouchedUnits(
	orderedPaths []string,
	unitsByFile map[string][]Unit,
	changedByFile map[string]map[int]struct{},
) []string {
	var candidates []string
	for _, relPath := range orderedPaths {
		changedLines := changedByFile[relPath]
		for _, u := range unitsByFile[relPath] {
			if u.ContainsAnyLine(changedLines) {
				candidates = append(candidates, relPath)
				break
			}
		}
	}
	return candidates
}

func loadBatchBaseKeys(
	ctx context.Context,
	analyzer LanguageAnalyzer,
	paths []string,
	readBase BaseReader,
) (map[string]map[string]struct{}, error) {
	if readBase == nil {
		return nil, nil
	}
	var baseBatch []FileInput
	for _, relPath := range paths {
		baseSrc, exists, err := readBase(relPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read base file %s: %w", relPath, err)
		}
		if exists {
			baseBatch = append(baseBatch, FileInput{Path: relPath, Content: baseSrc})
		}
	}
	if len(baseBatch) == 0 {
		return nil, nil
	}
	return analyzer.ExtractBaseKeys(ctx, baseBatch)
}

func collectCorpusUnits(ctx context.Context, opts Options, th Thresholds) ([]Unit, error) {
	corpusFiles := opts.CorpusFiles
	if len(corpusFiles) == 0 && opts.RepoRoot != "" {
		discovered, err := opts.Analyzer.DiscoverCorpusFiles(opts.RepoRoot)
		if err != nil {
			return nil, err
		}
		corpusFiles = discovered
	}
	allFiles := mergeUniquePaths(corpusFiles, slices.Sorted(maps.Keys(opts.ChangedProdLinesByFile)))
	var batch []FileInput
	var ordered []string
	for _, relPath := range allFiles {
		if !opts.Analyzer.IsEligibleFile(relPath) {
			continue
		}
		src, err := opts.ReadTargetFile(relPath)
		if err != nil {
			continue
		}
		batch = append(batch, FileInput{Path: relPath, Content: src})
		ordered = append(ordered, relPath)
	}
	byFile, err := opts.Analyzer.ExtractUnits(ctx, batch, th.MinStmts, th.MinCodeLines)
	if err != nil {
		return nil, err
	}
	var corpus []Unit
	for _, relPath := range ordered {
		corpus = append(corpus, byFile[relPath]...)
	}
	return corpus, nil
}

func rankSimilarCandidates(ctx context.Context, req rankRequest) ([]Advisory, error) {
	if len(req.corpus) == 0 {
		return nil, nil
	}
	newNorm, err := embedNormUnits(ctx, req.embedder, req.newUnits)
	if err != nil {
		return nil, err
	}
	corpNorm, err := embedNormUnits(ctx, req.embedder, req.corpus)
	if err != nil {
		return nil, err
	}

	idx := &corpusIndex{
		analyzer:   req.analyzer,
		units:      req.corpus,
		normVecs:   corpNorm,
		newIDSet:   buildIDSet(req.newUnits),
		thresholds: req.thresholds,
	}

	var advisories []Advisory
	for i, nu := range req.newUnits {
		matches := idx.findTopMatches(nu, newNorm[i])
		if len(matches) == 0 {
			continue
		}
		advisories = append(advisories, Advisory{
			Index:      len(advisories) + 1,
			NewUnit:    nu,
			Candidates: matches,
		})
	}
	return advisories, nil
}

func buildIDSet(units []Unit) map[string]bool {
	set := make(map[string]bool, len(units))
	for _, u := range units {
		set[u.ID] = true
	}
	return set
}

func embedNormUnits(ctx context.Context, embedder Embedder, units []Unit) ([][]float32, error) {
	normTexts := make([]string, len(units))
	for i, u := range units {
		normTexts[i] = u.NormText
	}
	return embedder.EmbedBatch(ctx, normTexts)
}

func (idx *corpusIndex) findTopMatches(query Unit, qNorm []float32) []CandidateMatch {
	var matches []CandidateMatch
	for j, cand := range idx.units {
		cutoff, isShared, eligible := idx.candidateCutoff(query, cand)
		if !eligible {
			continue
		}
		score := CosineNormalized(qNorm, idx.normVecs[j])
		if score < cutoff {
			continue
		}
		matches = append(matches, CandidateMatch{
			Unit:         cand,
			Score:        score,
			SharedModule: isShared,
			Guidance:     idx.buildGuidance(query, cand),
		})
	}
	slices.SortFunc(matches, compareCandidateMatches)
	return matches[:min(len(matches), idx.thresholds.TopK)]
}

func (idx *corpusIndex) candidateCutoff(query Unit, cand Unit) (float64, bool, bool) {
	if cand.ID == query.ID {
		return 0, false, false
	}
	isShared := idx.analyzer.IsSharedModule(cand.ModulePath)
	if idx.newIDSet[cand.ID] {
		if cand.FilePath == query.FilePath {
			return 0, false, false
		}
		return idx.thresholds.SameChangeMinScore, isShared, true
	}
	if isShared {
		return idx.thresholds.SharedMinScore, true, true
	}
	return idx.thresholds.MinScore, false, true
}

func (idx *corpusIndex) buildGuidance(query Unit, cand Unit) string {
	if idx.newIDSet[cand.ID] {
		return "Both units are newly added in this change; consolidate into a single shared helper."
	}
	if idx.analyzer.IsSharedModule(cand.ModulePath) {
		return fmt.Sprintf("Existing helper in shared module %q; inspect and reuse directly instead of duplicating logic.", cand.ModulePath)
	}
	if query.ModulePath == cand.ModulePath {
		return "Existing implementation in the same module; reuse or parameterize the existing code."
	}
	return "Similar logic exists in another module; extract a shared helper or verify intentional symmetry."
}

func compareCandidateMatches(a CandidateMatch, b CandidateMatch) int {
	return cmp.Or(
		cmp.Compare(b.Score, a.Score),
		cmp.Compare(a.Unit.FilePath, b.Unit.FilePath),
		cmp.Compare(a.Unit.StartLine, b.Unit.StartLine),
	)
}

func mergeUniquePaths(a []string, b []string) []string {
	merged := slices.Concat(a, b)
	slices.Sort(merged)
	return slices.Compact(merged)
}
