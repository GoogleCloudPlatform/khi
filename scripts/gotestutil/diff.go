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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderRegex = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// DiffOptions configures how VCS changes are collected.
type DiffOptions struct {
	VCS    string
	Base   string
	Target string
}

// UnifiedDiffLines holds added and modified 1-based line numbers partitioned by language and role.
type UnifiedDiffLines struct {
	Go     map[string]map[int]struct{}
	GoTest map[string]map[int]struct{}
	TS     map[string]map[int]struct{}
	TSTest map[string]map[int]struct{}
}

// DiffResult holds the resolved VCS metadata and changed 1-based line numbers per file.
type DiffResult struct {
	VCS                string
	BaseLabel          string
	ChangedLines       map[string]map[int]struct{}
	ChangedTestLines   map[string]map[int]struct{}
	TSChangedLines     map[string]map[int]struct{}
	TSChangedTestLines map[string]map[int]struct{}
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

// ParseUnifiedDiff extracts added and modified 1-based line numbers per production Go file from a unified diff stream.
func ParseUnifiedDiff(r io.Reader) (map[string]map[int]struct{}, error) {
	parsed, err := ParseUnifiedDiffAll(r)
	return parsed.Go, err
}

// ParseUnifiedDiffSplit extracts added and modified 1-based line numbers split into production Go files and *_test.go files.
func ParseUnifiedDiffSplit(r io.Reader) (map[string]map[int]struct{}, map[string]map[int]struct{}, error) {
	parsed, err := ParseUnifiedDiffAll(r)
	return parsed.Go, parsed.GoTest, err
}

// ParseUnifiedDiffAll extracts added and modified 1-based line numbers for Go, *_test.go, TypeScript, and *.spec.ts files.
func ParseUnifiedDiffAll(r io.Reader) (UnifiedDiffLines, error) {
	parsed := UnifiedDiffLines{
		Go:     make(map[string]map[int]struct{}),
		GoTest: make(map[string]map[int]struct{}),
		TS:     make(map[string]map[int]struct{}),
		TSTest: make(map[string]map[int]struct{}),
	}
	scanner := bufio.NewScanner(r)
	// Allow up to 16 MiB per line so diffs containing large generated JSON/lockfiles never fail scanning.
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var currentFile string
	var targetMap map[string]map[int]struct{}
	inHunk := false
	newLine := 0

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			currentFile = ""
			targetMap = nil
			inHunk = false
		case strings.HasPrefix(line, "--- "):
			inHunk = false
		case strings.HasPrefix(line, "+++ "):
			inHunk = false
			rawPath := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			if rawPath == "/dev/null" {
				currentFile = ""
				targetMap = nil
				continue
			}
			rawPath = strings.TrimPrefix(rawPath, "b/")
			clean := strings.TrimPrefix(filepath.ToSlash(rawPath), "./")
			switch {
			case IsTargetGoFile(clean):
				currentFile = clean
				targetMap = parsed.Go
			case IsTargetGoTestFile(clean):
				currentFile = clean
				targetMap = parsed.GoTest
			case IsTargetProductionTSFile(clean):
				currentFile = clean
				targetMap = parsed.TS
			case IsTargetTSTestFile(clean):
				currentFile = clean
				targetMap = parsed.TSTest
			default:
				currentFile = ""
				targetMap = nil
			}
		case strings.HasPrefix(line, "@@ "):
			matches := hunkHeaderRegex.FindStringSubmatch(line)
			if matches == nil {
				inHunk = false
				continue
			}
			start, err := strconv.Atoi(matches[1])
			if err != nil {
				return UnifiedDiffLines{}, fmt.Errorf("failed to parse hunk start line in %q: %w", line, err)
			}
			newLine = start
			inHunk = currentFile != "" && targetMap != nil
		case inHunk:
			if line == "" {
				newLine++
				continue
			}
			switch line[0] {
			case '+':
				lines, ok := targetMap[currentFile]
				if !ok {
					lines = make(map[int]struct{})
					targetMap[currentFile] = lines
				}
				lines[newLine] = struct{}{}
				newLine++
			case ' ':
				newLine++
			case '-', '\\':
				// Deleted lines and "\ No newline at end of file" markers do not advance the new-file line counter.
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return UnifiedDiffLines{}, fmt.Errorf("failed while scanning unified diff: %w", err)
	}
	return parsed, nil
}

// CollectChangedLines executes VCS diff in repoRoot and returns changed line numbers for matching Go files.
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
		base = "trunk()"
	}
	target := opts.Target
	if target == "" {
		target = "@"
	}

	fromRevset := fmt.Sprintf("fork_point((%s) | (%s))", target, base)
	diffBytes, err := RunCommandOutput(ctx, repoRoot, "jj", "diff", "--git", "--context=0", "--from", fromRevset, "--to", target)
	if err != nil {
		diffBytes, err = RunCommandOutput(ctx, repoRoot, "jj", "diff", "--git", "--context=0", "--from", base, "--to", target)
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to run jj diff (base=%s, target=%s): %w", base, target, err)
		}
	}

	parsed, err := ParseUnifiedDiffAll(bytes.NewReader(diffBytes))
	if err != nil {
		return DiffResult{}, err
	}
	return DiffResult{
		VCS:                "jj",
		BaseLabel:          base,
		ChangedLines:       parsed.Go,
		ChangedTestLines:   parsed.GoTest,
		TSChangedLines:     parsed.TS,
		TSChangedTestLines: parsed.TSTest,
	}, nil
}

func collectGitDiff(ctx context.Context, repoRoot string, opts DiffOptions) (DiffResult, error) {
	base := opts.Base
	if base == "" {
		resolved, err := resolveDefaultGitBase(ctx, repoRoot)
		if err != nil {
			return DiffResult{}, err
		}
		base = resolved
	}

	var args []string
	if opts.Target != "" {
		args = []string{"diff", "--unified=0", base, opts.Target}
	} else {
		diffBase := base
		if mbOut, err := RunCommandOutput(ctx, repoRoot, "git", "merge-base", base, "HEAD"); err == nil {
			if trimmed := strings.TrimSpace(string(mbOut)); trimmed != "" {
				diffBase = trimmed
			}
		}
		args = []string{"diff", "--unified=0", diffBase}
	}

	diffBytes, err := RunCommandOutput(ctx, repoRoot, "git", args...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to run git %s: %w", strings.Join(args, " "), err)
	}

	parsed, err := ParseUnifiedDiffAll(bytes.NewReader(diffBytes))
	if err != nil {
		return DiffResult{}, err
	}
	return DiffResult{
		VCS:                "git",
		BaseLabel:          base,
		ChangedLines:       parsed.Go,
		ChangedTestLines:   parsed.GoTest,
		TSChangedLines:     parsed.TS,
		TSChangedTestLines: parsed.TSTest,
	}, nil
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
