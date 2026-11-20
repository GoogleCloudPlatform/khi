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
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck/langgo"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck/langts"
)

type embedderFactory func(ctx context.Context, opts simcheck.EmbedderOptions) (simcheck.Embedder, string, error)

type resolvedEmbedder struct {
	embedder   simcheck.Embedder
	skipReason string
}

type similaritySession struct {
	enabled           bool
	explicitlySkipped bool
	resolve           func() (resolvedEmbedder, error)
	mu                sync.Mutex
	closer            io.Closer
}

type simLangRequest struct {
	repoRoot     string
	diffRes      gotestutil.DiffResult
	analyzer     simcheck.LanguageAnalyzer
	changedLines map[string]map[int]struct{}
}

func parseSimilarityProjectEnv(getenv func(string) string) string {
	return parseTrimmedEnv(getenv, "CODEQUALITY_SIMILARITY_PROJECT")
}

func newSimilaritySession(ctx context.Context, flags cliFlags, factory embedderFactory, getenv func(string) string) *similaritySession {
	speed, _ := rule.ParseSpeed(flags.speed)
	filter, _ := rule.ParseFilter(flags.filter)
	skipSources := parseCSVSet(flags.skipSources)
	_, explicitlySkipped := skipSources["similarity"]
	s := &similaritySession{
		enabled:           flags.similarity && speed.IncludesSlow() && filter.IncludesAdvisory(),
		explicitlySkipped: explicitlySkipped,
	}
	s.resolve = sync.OnceValues(func() (resolvedEmbedder, error) {
		if factory == nil {
			return resolvedEmbedder{skipReason: "embedding client not configured"}, nil
		}
		emb, reason, err := factory(ctx, simcheck.EmbedderOptions{
			ProjectID: flags.similarityProject,
			Getenv:    getenv,
		})
		if err != nil {
			return resolvedEmbedder{}, err
		}
		s.mu.Lock()
		s.closer, _ = emb.(io.Closer)
		s.mu.Unlock()
		return resolvedEmbedder{embedder: emb, skipReason: reason}, nil
	})
	return s
}

func (s *similaritySession) Close() error {
	return closeGuarded(s, func(sess *similaritySession) (*sync.Mutex, *io.Closer) {
		return &sess.mu, &sess.closer
	})
}

func (s *similaritySession) evaluateGo(ctx context.Context, repoRoot string, diffRes gotestutil.DiffResult) (rule.SimilaritySummary, error) {
	return s.evaluateLanguage(ctx, simLangRequest{
		repoRoot:     repoRoot,
		diffRes:      diffRes,
		analyzer:     langgo.NewAnalyzer(),
		changedLines: diffRes.ChangedLines,
	})
}

func (s *similaritySession) evaluateTS(ctx context.Context, repoRoot string, diffRes gotestutil.DiffResult) (rule.SimilaritySummary, error) {
	return s.evaluateLanguage(ctx, simLangRequest{
		repoRoot:     repoRoot,
		diffRes:      diffRes,
		analyzer:     langts.NewAnalyzer(repoRoot),
		changedLines: diffRes.TSChangedLines,
	})
}

func (s *similaritySession) evaluateLanguage(ctx context.Context, req simLangRequest) (rule.SimilaritySummary, error) {
	if s == nil || !s.enabled || len(req.changedLines) == 0 {
		return rule.SimilaritySummary{}, nil
	}
	if s.explicitlySkipped {
		return rule.SimilaritySummary{
			Enabled:    true,
			Skipped:    true,
			SkipReason: "explicitly skipped via --skip-sources",
		}, nil
	}
	res, err := simcheck.Evaluate(ctx, simcheck.Options{
		Analyzer:               req.analyzer,
		Thresholds:             simcheck.DefaultThresholds(),
		RepoRoot:               req.repoRoot,
		ChangedProdLinesByFile: req.changedLines,
		ReadTargetFile: func(relPath string) ([]byte, error) {
			return os.ReadFile(filepath.Join(req.repoRoot, filepath.FromSlash(relPath)))
		},
		ReadBaseFile: func(relPath string) ([]byte, bool, error) {
			return gotestutil.ReadBaseFile(ctx, req.repoRoot, req.diffRes, relPath)
		},
		ResolveEmbedder: func(context.Context) (simcheck.Embedder, string, error) {
			res, err := s.resolve()
			return res.embedder, res.skipReason, err
		},
	})
	if err != nil {
		return rule.SimilaritySummary{}, err
	}
	return res.ToSummary(), nil
}
