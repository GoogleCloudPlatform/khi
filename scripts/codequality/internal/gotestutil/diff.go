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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// DiffOptions configures how VCS changes are collected.
type DiffOptions struct {
	// VCS selects the version control backend ("auto", "jj", or "git").
	VCS string
	// Base specifies the base revision for differential comparison.
	Base string
	// Target specifies the target revision for differential comparison.
	Target string
	// CurrentOnly scopes the diff to the active working-copy or latest commit when Base is empty.
	CurrentOnly bool
}

// DiffResult holds the resolved VCS metadata and changed 1-based line numbers per file.
type DiffResult struct {
	// VCS is the resolved version control backend ("jj" or "git").
	VCS string
	// BaseLabel is the human-readable base revision label.
	BaseLabel string
	// ResolvedBaseRev is the resolved revision identifier used to read base files.
	ResolvedBaseRev string
	// ChangedLines maps modified Go production file paths to changed 1-based line sets.
	ChangedLines map[string]map[int]struct{}
	// ChangedTestLines maps modified Go test file paths to changed 1-based line sets.
	ChangedTestLines map[string]map[int]struct{}
	// TSChangedLines maps modified TypeScript production file paths to changed 1-based line sets.
	TSChangedLines map[string]map[int]struct{}
	// TSChangedTestLines maps modified TypeScript spec file paths to changed 1-based line sets.
	TSChangedTestLines map[string]map[int]struct{}
	// DeletedGoFiles lists Go file paths deleted relative to the base revision.
	DeletedGoFiles []string
	// OtherChangedGoFiles lists modified Go files outside primary analysis targets.
	OtherChangedGoFiles []string
	// ChangedFiles lists all added or modified relative file paths in deterministic order.
	ChangedFiles []string
	// DeletedFiles lists all deleted relative file paths in deterministic order.
	DeletedFiles []string
}

type gitDiffPlan struct {
	baseLabel   string
	resolvedRev string
	args        []string
}

// IsTargetGoFile reports whether relPath is a non-test Go source file eligible for coverage and mutation analysis.
func IsTargetGoFile(relPath string) bool {
	clean := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	if clean == "" || !strings.HasSuffix(clean, ".go") || strings.HasSuffix(clean, "_test.go") {
		return false
	}
	return isAllowedGoPath(clean)
}

// IsTargetGoTestFile reports whether relPath is a Go unit test (*_test.go) file eligible for analysis.
func IsTargetGoTestFile(relPath string) bool {
	clean := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	if !strings.HasSuffix(clean, "_test.go") {
		return false
	}
	return isAllowedGoPath(clean)
}

// IsTargetProductionTSFile reports whether relPath is a production TypeScript source file eligible for analysis.
func IsTargetProductionTSFile(relPath string) bool {
	clean := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	if !strings.HasPrefix(clean, "web/src/") || !strings.HasSuffix(clean, ".ts") {
		return false
	}
	if strings.HasSuffix(clean, ".spec.ts") ||
		strings.HasSuffix(clean, ".stories.ts") ||
		strings.HasSuffix(clean, ".d.ts") ||
		strings.HasSuffix(clean, ".mock.ts") {
		return false
	}
	if strings.HasPrefix(clean, "web/src/app/generated/") ||
		strings.HasPrefix(clean, "web/src/environments/") {
		return false
	}
	if clean == "web/src/test.ts" || clean == "web/src/polyfills.ts" {
		return false
	}
	if strings.Contains(clean, "/mock/") || strings.Contains(clean, "/test/") {
		return false
	}
	baseName := path.Base(clean)
	return !strings.HasPrefix(baseName, "zzz_") && !strings.HasPrefix(baseName, "zzz-")
}

// IsTargetTSTestFile reports whether relPath is a TypeScript unit spec (*.spec.ts) file under web/src/.
func IsTargetTSTestFile(relPath string) bool {
	clean := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	if !strings.HasPrefix(clean, "web/src/") || !strings.HasSuffix(clean, ".spec.ts") {
		return false
	}
	if strings.HasPrefix(clean, "web/src/app/generated/") ||
		strings.HasPrefix(clean, "web/src/environments/") {
		return false
	}
	baseName := path.Base(clean)
	return !strings.HasPrefix(baseName, "zzz_") && !strings.HasPrefix(baseName, "zzz-")
}

// CollectChangedLines executes VCS diff in repoRoot and returns changed line numbers for matching Go and TypeScript files.
func CollectChangedLines(ctx context.Context, repoRoot string, opts DiffOptions) (DiffResult, error) {
	vcs, err := resolveVCS(ctx, repoRoot, opts.VCS)
	if err != nil {
		return DiffResult{}, err
	}

	switch vcs {
	case "jj":
		res, jjErr := collectJJDiff(ctx, repoRoot, opts)
		if jjErr != nil && opts.VCS != "jj" && hasGitRepo(repoRoot) {
			if gitRes, gitErr := collectGitDiff(ctx, repoRoot, opts); gitErr == nil {
				return gitRes, nil
			}
		}
		return res, jjErr
	case "git":
		return collectGitDiff(ctx, repoRoot, opts)
	default:
		return DiffResult{}, fmt.Errorf("unsupported vcs %q", vcs)
	}
}

// ReadBaseFile reads relPath from the base revision recorded in res.
// It returns (nil, false, nil) when relPath did not exist at the base revision.
func ReadBaseFile(ctx context.Context, repoRoot string, res DiffResult, relPath string) ([]byte, bool, error) {
	cleanPath := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	if cleanPath == "" {
		return nil, false, fmt.Errorf("empty relative file path")
	}
	if res.ResolvedBaseRev == "" {
		return nil, false, fmt.Errorf("missing resolved base revision for vcs %q", res.VCS)
	}
	switch res.VCS {
	case "jj":
		return readJJBaseFile(ctx, repoRoot, res.ResolvedBaseRev, cleanPath)
	case "git":
		return readGitBaseFile(ctx, repoRoot, res.ResolvedBaseRev, cleanPath)
	default:
		return nil, false, fmt.Errorf("unsupported vcs %q", res.VCS)
	}
}

// RunCommandOutput executes name with args in dir and returns captured stdout or an enriched error containing stderr.
func RunCommandOutput(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return nil, fmt.Errorf("%w: %s", err, errMsg)
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

func isAllowedGoPath(clean string) bool {
	excludedPrefixes := []string{
		"pkg/generated/",
		"pkg/testutil/",
		"web/",
	}
	for _, prefix := range excludedPrefixes {
		if strings.HasPrefix(clean, prefix) {
			return false
		}
	}
	baseName := path.Base(clean)
	return !strings.HasPrefix(baseName, "zzz_")
}

func hasGitRepo(repoRoot string) bool {
	_, err := os.Stat(filepath.Join(repoRoot, ".git"))
	return err == nil
}

func resolveVCS(ctx context.Context, repoRoot string, requested string) (string, error) {
	switch requested {
	case "jj", "git":
		return requested, nil
	case "", "auto":
		if os.Getenv("GITHUB_ACTIONS") != "true" {
			if info, err := os.Stat(filepath.Join(repoRoot, ".jj")); err == nil && info.IsDir() {
				if _, err := exec.LookPath("jj"); err == nil {
					return "jj", nil
				}
			}
			cmd := exec.CommandContext(ctx, "jj", "root")
			cmd.Dir = repoRoot
			if err := cmd.Run(); err == nil {
				return "jj", nil
			}
		}
		return "git", nil
	default:
		return "", fmt.Errorf("invalid --vcs value %q (want auto, jj, or git)", requested)
	}
}

func collectJJDiff(ctx context.Context, repoRoot string, opts DiffOptions) (DiffResult, error) {
	base := opts.Base
	if base == "" {
		if opts.CurrentOnly {
			base = "@-"
		} else {
			base = "trunk()"
		}
	}
	target := opts.Target
	if target == "" {
		target = "@"
	}

	fromRevset := fmt.Sprintf("fork_point((%s) | (%s))", target, base)
	resolvedRev := fromRevset
	diffBytes, err := RunCommandOutput(ctx, repoRoot, "jj", "diff", "--git", "--context=0", "--from", fromRevset, "--to", target)
	if err != nil {
		resolvedRev = base
		diffBytes, err = RunCommandOutput(ctx, repoRoot, "jj", "diff", "--git", "--context=0", "--from", base, "--to", target)
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to run jj diff (base=%s, target=%s): %w", base, target, err)
		}
	}
	if idOut, idErr := RunCommandOutput(ctx, repoRoot, "jj", "log", "--ignore-working-copy", "--no-graph", "-T", "commit_id", "-r", resolvedRev); idErr == nil {
		if trimmed := strings.TrimSpace(string(idOut)); trimmed != "" {
			resolvedRev = trimmed
		}
	}

	details, err := ParseUnifiedDiffDetails(bytes.NewReader(diffBytes))
	if err != nil {
		return DiffResult{}, err
	}
	return buildDiffResult("jj", base, resolvedRev, details), nil
}

func collectGitDiff(ctx context.Context, repoRoot string, opts DiffOptions) (DiffResult, error) {
	plan, err := resolveGitDiffBaseAndArgs(ctx, repoRoot, opts)
	if err != nil {
		return DiffResult{}, err
	}

	diffBytes, err := RunCommandOutput(ctx, repoRoot, "git", plan.args...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to run git %s: %w", strings.Join(plan.args, " "), err)
	}

	details, err := ParseUnifiedDiffDetails(bytes.NewReader(diffBytes))
	if err != nil {
		return DiffResult{}, err
	}
	return buildDiffResult("git", plan.baseLabel, plan.resolvedRev, details), nil
}

func resolveGitDiffBaseAndArgs(ctx context.Context, repoRoot string, opts DiffOptions) (gitDiffPlan, error) {
	if opts.Base == "" && opts.CurrentOnly {
		return resolveCurrentOnlyGitDiff(ctx, repoRoot, opts.Target)
	}
	base := opts.Base
	if base == "" {
		resolved, err := resolveDefaultGitBase(ctx, repoRoot)
		if err != nil {
			return gitDiffPlan{}, err
		}
		base = resolved
	}
	if opts.Target != "" {
		return gitDiffPlan{
			baseLabel:   base,
			resolvedRev: base,
			args:        []string{"diff", "--unified=0", base, opts.Target},
		}, nil
	}
	diffBase := base
	if mbOut, err := RunCommandOutput(ctx, repoRoot, "git", "merge-base", base, "HEAD"); err == nil {
		if trimmed := strings.TrimSpace(string(mbOut)); trimmed != "" {
			diffBase = trimmed
		}
	}
	return gitDiffPlan{
		baseLabel:   base,
		resolvedRev: diffBase,
		args:        []string{"diff", "--unified=0", diffBase},
	}, nil
}

func resolveCurrentOnlyGitDiff(ctx context.Context, repoRoot string, target string) (gitDiffPlan, error) {
	statusOut, err := RunCommandOutput(ctx, repoRoot, "git", "status", "--porcelain")
	if err != nil {
		return gitDiffPlan{}, fmt.Errorf("failed to check git working tree status: %w", err)
	}
	if strings.TrimSpace(string(statusOut)) != "" {
		resolvedRev, err := resolveGitRevision(ctx, repoRoot, "HEAD")
		if err != nil {
			return gitDiffPlan{}, err
		}
		args := []string{"diff", "--unified=0", "HEAD"}
		if target != "" {
			args = append(args, target)
		}
		return gitDiffPlan{
			baseLabel:   "HEAD",
			resolvedRev: resolvedRev,
			args:        args,
		}, nil
	}
	resolvedRev, err := resolveGitRevision(ctx, repoRoot, "HEAD~1")
	if err != nil {
		return gitDiffPlan{}, err
	}
	if target == "" {
		target = "HEAD"
	}
	return gitDiffPlan{
		baseLabel:   "HEAD~1",
		resolvedRev: resolvedRev,
		args:        []string{"diff", "--unified=0", "HEAD~1", target},
	}, nil
}

func resolveGitRevision(ctx context.Context, repoRoot string, rev string) (string, error) {
	out, err := RunCommandOutput(ctx, repoRoot, "git", "rev-parse", "--verify", rev)
	if err != nil {
		return "", fmt.Errorf("failed to resolve git revision %q: %w", rev, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func buildDiffResult(vcs string, baseLabel string, resolvedRev string, details UnifiedDiffDetails) DiffResult {
	return DiffResult{
		VCS:                 vcs,
		BaseLabel:           baseLabel,
		ResolvedBaseRev:     resolvedRev,
		ChangedLines:        details.ProdLines,
		ChangedTestLines:    details.TestLines,
		TSChangedLines:      details.TSLines,
		TSChangedTestLines:  details.TSTestLines,
		DeletedGoFiles:      details.DeletedGoFiles,
		OtherChangedGoFiles: details.OtherChangedGoFiles,
		ChangedFiles:        details.ChangedFiles,
		DeletedFiles:        details.DeletedFiles,
	}
}

func readJJBaseFile(ctx context.Context, repoRoot string, rev string, cleanPath string) ([]byte, bool, error) {
	listed, err := RunCommandOutput(ctx, repoRoot, "jj", "file", "list", "--ignore-working-copy", "-r", rev, "--", cleanPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to query jj base file %s at %s: %w", cleanPath, rev, err)
	}
	if strings.TrimSpace(string(listed)) == "" {
		return nil, false, nil
	}
	data, err := RunCommandOutput(ctx, repoRoot, "jj", "file", "show", "--ignore-working-copy", "-r", rev, "--", cleanPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read jj base file %s at %s: %w", cleanPath, rev, err)
	}
	return data, true, nil
}

func readGitBaseFile(ctx context.Context, repoRoot string, rev string, cleanPath string) ([]byte, bool, error) {
	spec := rev + ":" + cleanPath
	if _, err := RunCommandOutput(ctx, repoRoot, "git", "cat-file", "-e", spec); err != nil {
		if _, revErr := RunCommandOutput(ctx, repoRoot, "git", "rev-parse", "--verify", rev); revErr != nil {
			return nil, false, fmt.Errorf("invalid git base revision %q: %w", rev, revErr)
		}
		return nil, false, nil
	}
	data, err := RunCommandOutput(ctx, repoRoot, "git", "show", spec)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read git base file %s: %w", spec, err)
	}
	return data, true, nil
}

func resolveDefaultGitBase(ctx context.Context, repoRoot string) (string, error) {
	candidates := []string{"origin/main", "main", "HEAD~1", "HEAD"}
	for _, ref := range candidates {
		if _, err := RunCommandOutput(ctx, repoRoot, "git", "rev-parse", "--verify", ref); err == nil {
			return ref, nil
		}
	}
	return "", fmt.Errorf("could not resolve a default git base revision (tried %s)", strings.Join(candidates, ", "))
}
