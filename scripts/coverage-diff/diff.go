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

// DiffResult holds the resolved VCS metadata and changed 1-based line numbers per file.
type DiffResult struct {
	VCS          string
	BaseLabel    string
	ChangedLines map[string]map[int]struct{}
}

// IsTargetProductionGoFile reports whether relPath is a production Go source file eligible for coverage measurement.
func IsTargetProductionGoFile(relPath string) bool {
	clean := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	if clean == "" || !strings.HasSuffix(clean, ".go") || strings.HasSuffix(clean, "_test.go") {
		return false
	}
	excludedPrefixes := []string{
		"pkg/generated/",
		"pkg/testutil/",
		"scripts/",
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

// ParseUnifiedDiff extracts added and modified 1-based line numbers per production Go file from a unified diff stream.
func ParseUnifiedDiff(r io.Reader) (map[string]map[int]struct{}, error) {
	result := make(map[string]map[int]struct{})
	scanner := bufio.NewScanner(r)
	// Allow up to 16 MiB per line so diffs containing large generated JSON/lockfiles never fail scanning.
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var currentFile string
	inHunk := false
	newLine := 0

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			currentFile = ""
			inHunk = false
		case strings.HasPrefix(line, "--- "):
			inHunk = false
		case strings.HasPrefix(line, "+++ "):
			inHunk = false
			rawPath := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			if rawPath == "/dev/null" {
				currentFile = ""
				continue
			}
			rawPath = strings.TrimPrefix(rawPath, "b/")
			if IsTargetProductionGoFile(rawPath) {
				currentFile = strings.TrimPrefix(filepath.ToSlash(rawPath), "./")
			} else {
				currentFile = ""
			}
		case strings.HasPrefix(line, "@@ "):
			matches := hunkHeaderRegex.FindStringSubmatch(line)
			if matches == nil {
				inHunk = false
				continue
			}
			start, err := strconv.Atoi(matches[1])
			if err != nil {
				return nil, fmt.Errorf("failed to parse hunk start line in %q: %w", line, err)
			}
			newLine = start
			inHunk = currentFile != ""
		case inHunk:
			if line == "" {
				newLine++
				continue
			}
			switch line[0] {
			case '+':
				lines, ok := result[currentFile]
				if !ok {
					lines = make(map[int]struct{})
					result[currentFile] = lines
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
		return nil, fmt.Errorf("failed while scanning unified diff: %w", err)
	}
	return result, nil
}

// CollectChangedLines executes VCS diff in repoRoot and returns changed line numbers for production Go files.
func CollectChangedLines(ctx context.Context, repoRoot string, opts DiffOptions) (DiffResult, error) {
	vcs, err := resolveVCS(ctx, repoRoot, opts.VCS)
	if err != nil {
		return DiffResult{}, err
	}

	switch vcs {
	case "jj":
		return collectJJDiff(ctx, repoRoot, opts)
	case "git":
		return collectGitDiff(ctx, repoRoot, opts)
	default:
		return DiffResult{}, fmt.Errorf("unsupported vcs %q", vcs)
	}
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
	diffBytes, err := runCommandOutput(ctx, repoRoot, "jj", "diff", "--git", "--context=0", "--from", fromRevset, "--to", target)
	if err != nil {
		// Fall back to direct --from when fork_point cannot resolve (e.g., disjoint histories).
		diffBytes, err = runCommandOutput(ctx, repoRoot, "jj", "diff", "--git", "--context=0", "--from", base, "--to", target)
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to run jj diff (base=%s, target=%s): %w", base, target, err)
		}
	}

	changed, err := ParseUnifiedDiff(bytes.NewReader(diffBytes))
	if err != nil {
		return DiffResult{}, err
	}
	return DiffResult{
		VCS:          "jj",
		BaseLabel:    base,
		ChangedLines: changed,
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
		if mbOut, err := runCommandOutput(ctx, repoRoot, "git", "merge-base", base, "HEAD"); err == nil {
			if trimmed := strings.TrimSpace(string(mbOut)); trimmed != "" {
				diffBase = trimmed
			}
		}
		args = []string{"diff", "--unified=0", diffBase}
	}

	diffBytes, err := runCommandOutput(ctx, repoRoot, "git", args...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to run git %s: %w", strings.Join(args, " "), err)
	}

	changed, err := ParseUnifiedDiff(bytes.NewReader(diffBytes))
	if err != nil {
		return DiffResult{}, err
	}
	return DiffResult{
		VCS:          "git",
		BaseLabel:    base,
		ChangedLines: changed,
	}, nil
}

func resolveDefaultGitBase(ctx context.Context, repoRoot string) (string, error) {
	candidates := []string{"origin/main", "main", "HEAD~1", "HEAD"}
	for _, ref := range candidates {
		if _, err := runCommandOutput(ctx, repoRoot, "git", "rev-parse", "--verify", ref); err == nil {
			return ref, nil
		}
	}
	return "", fmt.Errorf("could not resolve a default git base revision (tried %s)", strings.Join(candidates, ", "))
}

func runCommandOutput(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
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
