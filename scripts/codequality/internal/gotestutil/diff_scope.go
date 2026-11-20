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

package gotestutil

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// FilterDiffByPaths narrows all file and line collections in res to paths matching any pattern in paths.
func FilterDiffByPaths(res DiffResult, paths []string) DiffResult {
	if len(paths) == 0 {
		return res
	}
	patterns, matchAll := NormalizePathPatterns(paths)
	if matchAll {
		return res
	}
	return filterDiffResult(res, func(filePath string) bool {
		return MatchesAnyPathPattern(filePath, patterns)
	})
}

// IntersectDiffChangedSince narrows baseRes to files changed or deleted between sinceRev and targetRev while preserving base anchor metadata.
func IntersectDiffChangedSince(ctx context.Context, repoRoot string, baseRes DiffResult, sinceRev string, targetRev string) (DiffResult, error) {
	if sinceRev == "" {
		return baseRes, nil
	}
	vcs, err := resolveVCS(ctx, repoRoot, baseRes.VCS)
	if err != nil {
		return DiffResult{}, err
	}
	sinceFiles, err := collectChangedFileSetBetween(ctx, repoRoot, vcs, sinceRev, targetRev)
	if err != nil {
		return DiffResult{}, err
	}
	return filterDiffResult(baseRes, func(filePath string) bool {
		_, ok := sinceFiles[filePath]
		return ok
	}), nil
}

// NormalizePathPatterns normalizes repository-relative path prefixes or Go package patterns and reports whether any pattern matches all paths.
func NormalizePathPatterns(paths []string) ([]string, bool) {
	normalized := make([]string, 0, len(paths))
	for _, raw := range paths {
		clean := strings.TrimSpace(filepath.ToSlash(raw))
		if clean == "" {
			continue
		}
		clean = strings.TrimPrefix(clean, "./")
		clean = strings.TrimSuffix(clean, "/...")
		clean = strings.TrimSuffix(clean, "/")
		if clean == "" || clean == "." || clean == "..." {
			return nil, true
		}
		normalized = append(normalized, clean)
	}
	return normalized, false
}

// MatchesAnyPathPattern reports whether filePath matches any normalized path prefix pattern in patterns.
func MatchesAnyPathPattern(filePath string, patterns []string) bool {
	clean := strings.TrimPrefix(filepath.ToSlash(filePath), "./")
	for _, pattern := range patterns {
		if clean == pattern || strings.HasPrefix(clean, pattern+"/") {
			return true
		}
	}
	return false
}

func filterDiffResult(res DiffResult, keep func(string) bool) DiffResult {
	return DiffResult{
		VCS:                 res.VCS,
		BaseLabel:           res.BaseLabel,
		ResolvedBaseRev:     res.ResolvedBaseRev,
		ChangedLines:        filterLineMap(res.ChangedLines, keep),
		ChangedTestLines:    filterLineMap(res.ChangedTestLines, keep),
		TSChangedLines:      filterLineMap(res.TSChangedLines, keep),
		TSChangedTestLines:  filterLineMap(res.TSChangedTestLines, keep),
		DeletedGoFiles:      filterFileList(res.DeletedGoFiles, keep),
		OtherChangedGoFiles: filterFileList(res.OtherChangedGoFiles, keep),
		ChangedFiles:        filterFileList(res.ChangedFiles, keep),
		DeletedFiles:        filterFileList(res.DeletedFiles, keep),
	}
}

func filterLineMap(src map[string]map[int]struct{}, keep func(string) bool) map[string]map[int]struct{} {
	if len(src) == 0 {
		return nil
	}
	filtered := make(map[string]map[int]struct{})
	for filePath, lines := range src {
		if keep(filePath) {
			filtered[filePath] = lines
		}
	}
	return filtered
}

func filterFileList(src []string, keep func(string) bool) []string {
	if len(src) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(src))
	for _, filePath := range src {
		if keep(filePath) {
			filtered = append(filtered, filePath)
		}
	}
	return filtered
}

func collectChangedFileSetBetween(ctx context.Context, repoRoot string, vcs string, sinceRev string, targetRev string) (map[string]struct{}, error) {
	var out []byte
	var err error
	switch vcs {
	case "jj":
		target := targetRev
		if target == "" {
			target = "@"
		}
		out, err = RunCommandOutput(ctx, repoRoot, "jj", "diff", "--name-only", "--from", sinceRev, "--to", target)
	case "git":
		args := []string{"diff", "--name-only", "--no-renames", sinceRev}
		if targetRev != "" {
			args = append(args, targetRev)
		}
		out, err = RunCommandOutput(ctx, repoRoot, "git", args...)
	default:
		return nil, fmt.Errorf("unsupported vcs %q", vcs)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to collect changed files since %q: %w", sinceRev, err)
	}
	files := make(map[string]struct{})
	for _, line := range strings.Split(string(out), "\n") {
		clean := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(line)), "./")
		if clean != "" {
			files[clean] = struct{}{}
		}
	}
	return files, nil
}
