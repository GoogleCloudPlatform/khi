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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type fakeLangAnalyzer struct {
	discovered  []string
	discoverErr error
}

func (f *fakeLangAnalyzer) ID() LanguageID               { return "mocklang" }
func (f *fakeLangAnalyzer) DisplayName() string          { return "MockLang" }
func (f *fakeLangAnalyzer) CodeFence() string            { return "mock" }
func (f *fakeLangAnalyzer) IsEligibleFile(p string) bool { return strings.HasSuffix(p, ".mock") }
func (f *fakeLangAnalyzer) DiscoverCorpusFiles(string) ([]string, error) {
	return f.discovered, f.discoverErr
}
func (f *fakeLangAnalyzer) IsSharedModule(mod string) bool {
	return strings.HasSuffix(mod, "/common")
}
func (f *fakeLangAnalyzer) ExtractUnits(_ context.Context, files []FileInput, _ int, _ int) (map[string][]Unit, error) {
	out := make(map[string][]Unit, len(files))
	for _, file := range files {
		lines := strings.Split(strings.TrimSpace(string(file.Content)), "\n")
		for idx, ln := range lines {
			parts := strings.SplitN(strings.TrimSpace(ln), ":", 2)
			if len(parts) != 2 {
				continue
			}
			fnName, body := parts[0], parts[1]
			lineNo := idx + 1
			out[file.Path] = append(out[file.Path], Unit{
				ID:         file.Path + ":" + fnName,
				Key:        "func:" + fnName,
				Language:   "mocklang",
				ModulePath: filepath.ToSlash(filepath.Dir(file.Path)),
				FilePath:   file.Path,
				StartLine:  lineNo,
				EndLine:    lineNo,
				Name:       fnName,
				Kind:       KindFunc,
				StmtCount:  5,
				CodeLines:  8,
				NormText:   body,
			})
		}
	}
	return out, nil
}
func (f *fakeLangAnalyzer) ExtractBaseKeys(ctx context.Context, files []FileInput) (map[string]map[string]struct{}, error) {
	unitsByFile, err := f.ExtractUnits(ctx, files, 0, 0)
	if err != nil {
		return nil, err
	}
	res := make(map[string]map[string]struct{}, len(unitsByFile))
	for p, units := range unitsByFile {
		set := make(map[string]struct{}, len(units))
		for _, u := range units {
			set[u.Key] = struct{}{}
		}
		res[p] = set
	}
	return res, nil
}

type stubEmbedder struct {
	err error
}

func (s *stubEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make([][]float32, len(texts))
	for i, text := range texts {
		if strings.Contains(text, "DEDUP_ALGO") {
			out[i] = NormalizeL2([]float32{0.99, 0.01})
		} else {
			out[i] = NormalizeL2([]float32{0.01, 0.99})
		}
	}
	return out, nil
}

func TestNormalizeThresholds(t *testing.T) {
	testCases := []struct {
		name  string
		input Thresholds
		want  Thresholds
	}{
		{
			name:  "zero thresholds populate defaults",
			input: Thresholds{},
			want:  DefaultThresholds(),
		},
		{
			name: "positive custom thresholds preserved verbatim",
			input: Thresholds{
				MinStmts:           5,
				MinCodeLines:       10,
				MinScore:           0.90,
				SharedMinScore:     0.85,
				SameChangeMinScore: 0.98,
				TopK:               4,
			},
			want: Thresholds{
				MinStmts:           5,
				MinCodeLines:       10,
				MinScore:           0.90,
				SharedMinScore:     0.85,
				SameChangeMinScore: 0.98,
				TopK:               4,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeThresholds(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("normalizeThresholds() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEvaluateWithPluggableLanguageAnalyzer(t *testing.T) {
	sharedFile := "DedupStrings:DEDUP_ALGO\n"
	baseFeatureFile := "ExistingLogic:OTHER_ALGO\n"
	targetFeatureFile := "ExistingLogic:OTHER_ALGO_MODIFIED\ndedupClusterLabels:DEDUP_ALGO\n"

	type evalOutcome struct {
		LanguageTitle     string
		Skipped           bool
		NewUnitsEvaluated int
		CorpusUnitsCount  int
		AdvisoryIndexes   []int
		AdvisoryNewNames  []string
		TopCandidateNames []string
		TopSharedFlags    []bool
	}

	testCases := []struct {
		name        string
		changedLine int
		embedder    Embedder
		resolveFn   func(context.Context) (Embedder, string, error)
		want        evalOutcome
	}{
		{
			name:        "short-circuits without invoking ResolveEmbedder when only existing unit modified",
			changedLine: 1,
			resolveFn: func(context.Context) (Embedder, string, error) {
				return nil, "", errors.New("should not be called when 0 new units")
			},
			want: evalOutcome{
				LanguageTitle:     "MockLang",
				Skipped:           false,
				NewUnitsEvaluated: 0,
			},
		},
		{
			name:        "skips gracefully when new unit exists and ResolveEmbedder returns nil",
			changedLine: 2,
			resolveFn: func(context.Context) (Embedder, string, error) {
				return nil, "no credentials", nil
			},
			want: evalOutcome{
				LanguageTitle:     "MockLang",
				Skipped:           true,
				NewUnitsEvaluated: 1,
			},
		},
		{
			name:        "skips gracefully when ResolveEmbedder returns initialization error",
			changedLine: 2,
			resolveFn: func(context.Context) (Embedder, string, error) {
				return nil, "", errors.New("cache init failed")
			},
			want: evalOutcome{
				LanguageTitle:     "MockLang",
				Skipped:           true,
				NewUnitsEvaluated: 1,
			},
		},
		{
			name:        "degrades gracefully to skipped when EmbedBatch encounters transient network error",
			changedLine: 2,
			embedder:    &stubEmbedder{err: errors.New("network unreachable")},
			want: evalOutcome{
				LanguageTitle:     "MockLang",
				Skipped:           true,
				NewUnitsEvaluated: 1,
			},
		},
		{
			name:        "flags newly added unit matching shared module helper via lazy corpus discovery",
			changedLine: 2,
			resolveFn: func(context.Context) (Embedder, string, error) {
				return &stubEmbedder{}, "", nil
			},
			want: evalOutcome{
				LanguageTitle:     "MockLang",
				Skipped:           false,
				NewUnitsEvaluated: 1,
				CorpusUnitsCount:  3,
				AdvisoryIndexes:   []int{1},
				AdvisoryNewNames:  []string{"dedupClusterLabels"},
				TopCandidateNames: []string{"DedupStrings"},
				TopSharedFlags:    []bool{true},
			},
		},
	}

	files := map[string]string{
		"src/common/dedup.mock":   sharedFile,
		"src/feature/mapper.mock": targetFeatureFile,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Evaluate(context.Background(), Options{
				Analyzer:   &fakeLangAnalyzer{discovered: []string{"src/common/dedup.mock", "src/feature/mapper.mock"}},
				Thresholds: Thresholds{},
				RepoRoot:   "/virtual/repo",
				ChangedProdLinesByFile: map[string]map[int]struct{}{
					"src/feature/mapper.mock": {tc.changedLine: {}},
				},
				ReadTargetFile: func(relPath string) ([]byte, error) {
					return []byte(files[relPath]), nil
				},
				ReadBaseFile: func(relPath string) ([]byte, bool, error) {
					if relPath == "src/feature/mapper.mock" {
						return []byte(baseFeatureFile), true, nil
					}
					return []byte(sharedFile), true, nil
				},
				Embedder:        tc.embedder,
				ResolveEmbedder: tc.resolveFn,
			})
			if err != nil {
				t.Fatalf("Evaluate() failed: %v", err)
			}

			got := evalOutcome{
				LanguageTitle:     res.LanguageTitle,
				Skipped:           res.Skipped,
				NewUnitsEvaluated: res.NewUnitsEvaluated,
				CorpusUnitsCount:  res.CorpusUnitsCount,
			}
			for _, adv := range res.Advisories {
				got.AdvisoryIndexes = append(got.AdvisoryIndexes, adv.Index)
				got.AdvisoryNewNames = append(got.AdvisoryNewNames, adv.NewUnit.Name)
				got.TopCandidateNames = append(got.TopCandidateNames, adv.Candidates[0].Unit.Name)
				got.TopSharedFlags = append(got.TopSharedFlags, adv.Candidates[0].SharedModule)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Evaluate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEvaluateEdgeCasesAndTieBreaking(t *testing.T) {
	if _, err := Evaluate(context.Background(), Options{}); err == nil {
		t.Errorf("Evaluate(nil analyzer) succeeded, want error")
	}

	if _, err := Evaluate(context.Background(), Options{
		Analyzer: &fakeLangAnalyzer{},
		ChangedProdLinesByFile: map[string]map[int]struct{}{
			"src/unreadable.mock": {1: {}},
		},
		ReadTargetFile: func(string) ([]byte, error) {
			return nil, errors.New("permission denied")
		},
	}); err == nil {
		t.Errorf("Evaluate(target read error) succeeded, want error")
	}

	resIneligibleOnly, err := Evaluate(context.Background(), Options{
		Analyzer: &fakeLangAnalyzer{},
		ChangedProdLinesByFile: map[string]map[int]struct{}{
			"README.md":      {1: {}},
			"src/empty.mock": {},
			"src/gone.mock":  {1: {}},
		},
		ReadTargetFile: func(string) ([]byte, error) {
			return nil, os.ErrNotExist
		},
	})
	if err != nil || resIneligibleOnly.NewUnitsEvaluated != 0 {
		t.Fatalf("Evaluate(empty target batch) = (%+v, %v), want 0 new units", resIneligibleOnly, err)
	}

	resNoTargets, err := Evaluate(context.Background(), Options{
		Analyzer: &fakeLangAnalyzer{},
		ChangedProdLinesByFile: map[string]map[int]struct{}{
			"README.md":           {1: {}},
			"src/empty.mock":      {},
			"src/deleted.mock":    {1: {}},
			"src/outside_fn.mock": {99: {}},
		},
		ReadTargetFile: func(relPath string) ([]byte, error) {
			if relPath == "src/deleted.mock" {
				return nil, os.ErrNotExist
			}
			return []byte("Fn:OTHER_ALGO\n"), nil
		},
		ReadBaseFile: func(string) ([]byte, bool, error) {
			return nil, false, errors.New("should not read base file when no units touched")
		},
	})
	if err != nil || resNoTargets.NewUnitsEvaluated != 0 {
		t.Fatalf("Evaluate(non-unit changes) = (%+v, %v), want 0 new units", resNoTargets, err)
	}

	resNilEmbed, err := Evaluate(context.Background(), Options{
		Analyzer: &fakeLangAnalyzer{},
		ChangedProdLinesByFile: map[string]map[int]struct{}{
			"src/brandnew.mock": {1: {}},
		},
		ReadTargetFile: func(string) ([]byte, error) {
			return []byte("NewFn:DEDUP_ALGO\n"), nil
		},
		ReadBaseFile: nil,
	})
	if err != nil || !resNilEmbed.Skipped || resNilEmbed.SkipReason != "embedding client not configured" {
		t.Fatalf("Evaluate(nil embedder & nil baseReader) = (%+v, %v)", resNilEmbed, err)
	}

	corpusDocs := map[string]string{
		"src/new1/query.mock": "Query:DEDUP_ALGO\nUnmatched:UNIQUE_ALGO\n",
		"src/new2/twin.mock":  "Twin:DEDUP_ALGO\n",
		"src/b/peer.mock":     "PeerB1:DEDUP_ALGO\nPeerB2:DEDUP_ALGO\n",
		"src/a/peer.mock":     "PeerA1:DEDUP_ALGO\n",
	}
	th := Thresholds{
		MinStmts:           3,
		MinCodeLines:       6,
		MinScore:           1.0,
		SharedMinScore:     1.0,
		SameChangeMinScore: 1.0,
		TopK:               2,
	}
	resRanked, err := Evaluate(context.Background(), Options{
		Analyzer:   &fakeLangAnalyzer{discoverErr: errors.New("should not discover when CorpusFiles is provided")},
		Thresholds: th,
		RepoRoot:   "/virtual/repo",
		ChangedProdLinesByFile: map[string]map[int]struct{}{
			"src/new1/query.mock": {1: {}, 2: {}},
			"src/new2/twin.mock":  {1: {}},
		},
		CorpusFiles: []string{"README.md", "src/missing_corpus.mock", "src/a/peer.mock", "src/b/peer.mock", "src/new1/query.mock", "src/new2/twin.mock"},
		ReadTargetFile: func(relPath string) ([]byte, error) {
			if relPath == "src/missing_corpus.mock" {
				return nil, os.ErrNotExist
			}
			return []byte(corpusDocs[relPath]), nil
		},
		ReadBaseFile: func(relPath string) ([]byte, bool, error) {
			if relPath == "src/new1/query.mock" {
				return []byte("Query:STALE_IGNORED\n"), false, nil
			}
			if relPath == "src/new2/twin.mock" {
				return []byte{}, true, nil
			}
			return []byte(corpusDocs[relPath]), true, nil
		},
		Embedder: &stubEmbedder{},
	})
	if err != nil {
		t.Fatalf("Evaluate(tie-break & TopK) failed: %v", err)
	}
	if len(resRanked.Advisories) != 2 {
		t.Fatalf("len(Advisories) = %d, want 2", len(resRanked.Advisories))
	}
	if resRanked.Advisories[0].Index != 1 || resRanked.Advisories[1].Index != 2 {
		t.Errorf("advisory indexes = (%d, %d), want (1, 2)", resRanked.Advisories[0].Index, resRanked.Advisories[1].Index)
	}
	var gotTopCands []string
	for _, c := range resRanked.Advisories[0].Candidates {
		gotTopCands = append(gotTopCands, c.Unit.Name)
	}
	wantTopCands := []string{"PeerA1", "PeerB1"}
	if diff := cmp.Diff(wantTopCands, gotTopCands); diff != "" {
		t.Errorf("tie-broken TopK candidates mismatch (-want +got):\n%s", diff)
	}

	hasBothNewGuidance := false
	for _, adv := range resRanked.Advisories {
		for _, c := range adv.Candidates {
			if strings.Contains(c.Guidance, "Both units are newly added") {
				hasBothNewGuidance = true
			}
		}
	}
	thAll := th
	thAll.TopK = 10
	resAllCands, err := Evaluate(context.Background(), Options{
		Analyzer:   &fakeLangAnalyzer{},
		Thresholds: thAll,
		ChangedProdLinesByFile: map[string]map[int]struct{}{
			"src/new1/query.mock": {1: {}},
			"src/new2/twin.mock":  {1: {}},
		},
		CorpusFiles: []string{"src/new1/query.mock", "src/new2/twin.mock"},
		ReadTargetFile: func(relPath string) ([]byte, error) {
			return []byte(corpusDocs[relPath]), nil
		},
		ReadBaseFile: func(string) ([]byte, bool, error) {
			return nil, false, nil
		},
		Embedder: &stubEmbedder{},
	})
	if err != nil {
		t.Fatalf("Evaluate(same-change pair) failed: %v", err)
	}
	for _, adv := range resAllCands.Advisories {
		for _, c := range adv.Candidates {
			if strings.Contains(c.Guidance, "Both units are newly added") {
				hasBothNewGuidance = true
			}
		}
	}
	if !hasBothNewGuidance {
		t.Errorf("expected same-change guidance via buildIDSet")
	}

	emptyRank, err := rankSimilarCandidates(context.Background(), rankRequest{})
	if err != nil || emptyRank != nil {
		t.Errorf("rankSimilarCandidates(empty corpus) = (%v, %v), want (nil, nil)", emptyRank, err)
	}
}

func TestCorpusIndexGuidanceAndCutoffs(t *testing.T) {
	query := Unit{ID: "src/a/x.mock:Foo", FilePath: "src/a/x.mock", ModulePath: "src/a"}
	th := DefaultThresholds()

	type cutoffOutcome struct {
		Cutoff   float64
		Shared   bool
		Eligible bool
		Guidance string
	}

	testCases := []struct {
		name     string
		cand     Unit
		newIDSet map[string]bool
		want     cutoffOutcome
	}{
		{
			name:     "self candidate is ineligible",
			cand:     query,
			newIDSet: map[string]bool{"src/a/x.mock:Foo": true},
			want:     cutoffOutcome{Cutoff: 0, Shared: false, Eligible: false},
		},
		{
			name:     "another new unit in same file is ineligible",
			cand:     Unit{ID: "src/a/x.mock:Sibling", FilePath: "src/a/x.mock", ModulePath: "src/a"},
			newIDSet: map[string]bool{"src/a/x.mock:Sibling": true},
			want:     cutoffOutcome{Cutoff: 0, Shared: false, Eligible: false},
		},
		{
			name:     "both newly added across different files",
			cand:     Unit{ID: "src/b/y.mock:Bar", FilePath: "src/b/y.mock", ModulePath: "src/b"},
			newIDSet: map[string]bool{"src/b/y.mock:Bar": true},
			want: cutoffOutcome{
				Cutoff:   th.SameChangeMinScore,
				Shared:   false,
				Eligible: true,
				Guidance: "Both units are newly added in this change; consolidate into a single shared helper.",
			},
		},
		{
			name:     "shared module candidate",
			cand:     Unit{ID: "src/common/z.mock:Baz", FilePath: "src/common/z.mock", ModulePath: "src/common"},
			newIDSet: map[string]bool{},
			want: cutoffOutcome{
				Cutoff:   th.SharedMinScore,
				Shared:   true,
				Eligible: true,
				Guidance: `Existing helper in shared module "src/common"; inspect and reuse directly instead of duplicating logic.`,
			},
		},
		{
			name:     "same module candidate",
			cand:     Unit{ID: "src/a/other.mock:Qux", FilePath: "src/a/other.mock", ModulePath: "src/a"},
			newIDSet: map[string]bool{},
			want: cutoffOutcome{
				Cutoff:   th.MinScore,
				Shared:   false,
				Eligible: true,
				Guidance: "Existing implementation in the same module; reuse or parameterize the existing code.",
			},
		},
		{
			name:     "cross module candidate",
			cand:     Unit{ID: "src/c/other.mock:Quux", FilePath: "src/c/other.mock", ModulePath: "src/c"},
			newIDSet: map[string]bool{},
			want: cutoffOutcome{
				Cutoff:   th.MinScore,
				Shared:   false,
				Eligible: true,
				Guidance: "Similar logic exists in another module; extract a shared helper or verify intentional symmetry.",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idx := &corpusIndex{
				analyzer:   &fakeLangAnalyzer{},
				newIDSet:   tc.newIDSet,
				thresholds: th,
			}
			cutoff, shared, elig := idx.candidateCutoff(query, tc.cand)
			var guidance string
			if elig {
				guidance = idx.buildGuidance(query, tc.cand)
			}
			got := cutoffOutcome{
				Cutoff:   cutoff,
				Shared:   shared,
				Eligible: elig,
				Guidance: guidance,
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("candidateCutoff/buildGuidance mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
