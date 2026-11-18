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
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/decisioncheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const defaultDecisionMaxTargets = 20

type deciderFactory func(ctx context.Context, opts decisioncheck.DeciderOptions) (decisioncheck.Decider, string, error)

type resolvedDecider struct {
	decider    decisioncheck.Decider
	skipReason string
}

type decisionsSession struct {
	enabled bool
	flags   cliFlags
	getenv  func(string) string
	resolve func() (resolvedDecider, error)
	mu      sync.Mutex
	cache   *decisioncheck.DecisionCache
}

func parseTrimmedEnv(getenv func(string) string, key string) string {
	if getenv == nil {
		return ""
	}
	return strings.TrimSpace(getenv(key))
}

func parseBoolEnv(getenv func(string) string, key string) bool {
	normalized := strings.ToLower(parseTrimmedEnv(getenv, key))
	return normalized == "true" || normalized == "1"
}

func parseDecisionsEnv(getenv func(string) string) bool {
	return parseBoolEnv(getenv, "CODEQUALITY_DECISIONS")
}

func parseDecisionsProjectEnv(getenv func(string) string) string {
	return parseTrimmedEnv(getenv, "CODEQUALITY_DECISIONS_PROJECT")
}

func parseDecisionsFailEnv(getenv func(string) string) bool {
	return parseBoolEnv(getenv, "CODEQUALITY_DECISIONS_FAIL_ON_VIOLATIONS")
}

func newDecisionsSession(ctx context.Context, flags cliFlags, factory deciderFactory, getenv func(string) string) *decisionsSession {
	session := &decisionsSession{
		enabled: flags.decisions || flags.decisionsOnly,
		flags:   flags,
		getenv:  getenv,
	}
	session.resolve = sync.OnceValues(func() (resolvedDecider, error) {
		if factory == nil {
			return resolvedDecider{skipReason: "decision client not configured"}, nil
		}
		decider, reason, err := factory(ctx, decisioncheck.DeciderOptions{
			ProjectID: flags.decisionsProject,
			Getenv:    getenv,
		})
		if err != nil {
			return resolvedDecider{}, err
		}
		return resolvedDecider{decider: decider, skipReason: reason}, nil
	})
	return session
}

func (s *decisionsSession) Close() error {
	return closeGuarded(s, func(sess *decisionsSession) (*sync.Mutex, **decisioncheck.DecisionCache) {
		return &sess.mu, &sess.cache
	})
}

func (s *decisionsSession) evaluate(ctx context.Context, repoRoot string, diffRes gotestutil.DiffResult) (*rule.Report, error) {
	if s == nil || !s.enabled {
		return nil, nil
	}
	if !s.flags.full && len(diffRes.ChangedFiles) == 0 && len(diffRes.DeletedFiles) == 0 {
		return nil, nil
	}
	trustedRoot, err := resolveTrustedRoot(repoRoot, s.flags.trustedRoot)
	if err != nil {
		return nil, err
	}
	rulesDir := s.flags.decisionsRulesDir
	if rulesDir == "" {
		rulesDir = filepath.Join(trustedRoot, "scripts", "codequality", "rules")
	}
	specs, err := decisioncheck.LoadRulesDir(rulesDir)
	if err != nil {
		return nil, err
	}
	cache, err := s.ensureCache(ctx)
	if err != nil {
		return nil, err
	}
	sum, err := decisioncheck.Evaluate(ctx, decisioncheck.Options{
		Rules:         specs,
		RuleFilter:    parseCSVSet(s.flags.decisionsRules),
		ForceBlocking: s.flags.decisionsFailOnViolations,
		TargetOpts: decisioncheck.ResolveOptions{
			RepoRoot:     repoRoot,
			TrustedRoot:  trustedRoot,
			Full:         s.flags.full,
			MaxTargets:   s.flags.decisionsMaxTargets,
			ChangedFiles: diffRes.ChangedFiles,
			DeletedFiles: diffRes.DeletedFiles,
			ChangedLines: mergeChangedLines(diffRes.ChangedLines, diffRes.ChangedTestLines, diffRes.TSChangedLines, diffRes.TSChangedTestLines),
			ReadDiff: func(relPaths []string) (string, error) {
				return collectPathsUnifiedDiff(ctx, repoRoot, diffRes, relPaths)
			},
		},
		Cache: cache,
		ResolveDecider: func(context.Context) (decisioncheck.Decider, string, error) {
			resolved, resolveErr := s.resolve()
			return resolved.decider, resolved.skipReason, resolveErr
		},
	})
	if err != nil {
		return nil, err
	}
	if !sum.Skipped && sum.TargetsEvaluated == 0 {
		return nil, nil
	}
	return &rule.Report{
		Language:  "Decisions",
		BaseLabel: diffRes.BaseLabel,
		Decisions: sum,
	}, nil
}

func (s *decisionsSession) ensureCache(ctx context.Context) (*decisioncheck.DecisionCache, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache != nil {
		return s.cache, nil
	}
	// Fall back to an in-memory cache ("") when DefaultCachePath cannot resolve a user cache directory.
	cachePath, _ := decisioncheck.DefaultCachePath(s.getenv)
	cache, err := decisioncheck.OpenDecisionCache(ctx, cachePath)
	if err != nil {
		return nil, err
	}
	s.cache = cache
	return cache, nil
}

func resolveTrustedRoot(repoRoot string, trustedFlag string) (string, error) {
	if strings.TrimSpace(trustedFlag) == "" {
		return repoRoot, nil
	}
	abs, err := filepath.Abs(trustedFlag)
	if err != nil {
		return "", fmt.Errorf("failed resolving --trusted-root %q: %w", trustedFlag, err)
	}
	return abs, nil
}

func parseCSVSet(raw string) map[string]struct{} {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	out := make(map[string]struct{})
	for part := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out[trimmed] = struct{}{}
		}
	}
	return out
}

func collectPathsUnifiedDiff(ctx context.Context, repoRoot string, diffRes gotestutil.DiffResult, relPaths []string) (string, error) {
	if len(relPaths) == 0 || diffRes.ResolvedBaseRev == "" {
		return "", nil
	}
	cmdName := "git"
	var args []string
	if diffRes.VCS == "jj" {
		cmdName = "jj"
		args = append([]string{"diff", "--git", "--from", diffRes.ResolvedBaseRev, "--"}, relPaths...)
	} else {
		args = append([]string{"diff", diffRes.ResolvedBaseRev, "--"}, relPaths...)
	}
	out, err := gotestutil.RunCommandOutput(ctx, repoRoot, cmdName, args...)
	return string(out), err
}

func mergeChangedLines(maps ...map[string]map[int]struct{}) map[string]map[int]struct{} {
	total := 0
	for _, linesByFile := range maps {
		total += len(linesByFile)
	}
	if total == 0 {
		return nil
	}
	merged := make(map[string]map[int]struct{}, total)
	for _, linesByFile := range maps {
		for filePath, lines := range linesByFile {
			merged[filePath] = lines
		}
	}
	return merged
}

func closeGuarded[Owner any, Closer interface {
	io.Closer
	comparable
}](owner *Owner, fields func(*Owner) (*sync.Mutex, *Closer)) error {
	if owner == nil {
		return nil
	}
	mu, target := fields(owner)
	mu.Lock()
	closer := *target
	var zero Closer
	*target = zero
	mu.Unlock()
	if closer == zero {
		return nil
	}
	return closer.Close()
}
