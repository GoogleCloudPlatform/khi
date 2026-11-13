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

// DiffResult holds the resolved VCS metadata and changed 1-based line numbers per file.
type DiffResult struct {
	VCS                 string
	BaseLabel           string
	ResolvedBaseRev     string
	ChangedLines        map[string]map[int]struct{}
	ChangedTestLines    map[string]map[int]struct{}
	TSChangedLines      map[string]map[int]struct{}
	TSChangedTestLines  map[string]map[int]struct{}
	DeletedGoFiles      []string
	OtherChangedGoFiles []string
}

// UnifiedDiffDetails holds parsed line maps and deleted/non-target Go files extracted from a unified diff stream.
type UnifiedDiffDetails struct {
	ProdLines           map[string]map[int]struct{}
	TestLines           map[string]map[int]struct{}
	TSLines             map[string]map[int]struct{}
	TSTestLines         map[string]map[int]struct{}
	DeletedGoFiles      []string
	OtherChangedGoFiles []string
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

// ParseUnifiedDiffDetails extracts changed line numbers for Go, *_test.go, TypeScript, and *.spec.ts files plus deleted Go file paths.
func ParseUnifiedDiffDetails(r io.Reader) (UnifiedDiffDetails, error) {
	state := &diffScanState{
		prodLines:   make(map[string]map[int]struct{}),
		testLines:   make(map[string]map[int]struct{}),
		tsLines:     make(map[string]map[int]struct{}),
		tsTestLines: make(map[string]map[int]struct{}),
	}
	scanner := bufio.NewScanner(r)
	// Allow up to 16 MiB per line so diffs containing large generated JSON/lockfiles never fail scanning.
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	for scanner.Scan() {
		if err := state.step(scanner.Text()); err != nil {
			return UnifiedDiffDetails{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return UnifiedDiffDetails{}, fmt.Errorf("failed while scanning unified diff: %w", err)
	}
	return UnifiedDiffDetails{
		ProdLines:           state.prodLines,
		TestLines:           state.testLines,
		TSLines:             state.tsLines,
		TSTestLines:         state.tsTestLines,
		DeletedGoFiles:      state.deletedGoFiles,
		OtherChangedGoFiles: state.otherChangedGoFiles,
	}, nil
}

type diffScanState struct {
	prodLines           map[string]map[int]struct{}
	testLines           map[string]map[int]struct{}
	tsLines             map[string]map[int]struct{}
	tsTestLines         map[string]map[int]struct{}
	deletedGoFiles      []string
	otherChangedGoFiles []string
	oldFile             string
	currentFile         string
	targetMap           map[string]map[int]struct{}
	inHunk              bool
	newLine             int
}

func (s *diffScanState) step(line string) error {
	switch {
	case strings.HasPrefix(line, "diff --git "):
		s.oldFile = ""
		s.targetMap = nil
		s.inHunk = false
	case strings.HasPrefix(line, "--- "):
		s.inHunk = false
		s.handleOldHeader(line)
	case strings.HasPrefix(line, "+++ "):
		s.inHunk = false
		s.handleNewHeader(line)
	case strings.HasPrefix(line, "@@ "):
		return s.handleHunkHeader(line)
	case s.inHunk:
		s.handleHunkLine(line)
	}
	return nil
}

func (s *diffScanState) handleOldHeader(line string) {
	rawPath := strings.TrimSpace(strings.TrimPrefix(line, "--- "))
	if rawPath == "/dev/null" {
		s.oldFile = ""
		return
	}
	s.oldFile = strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(rawPath, "a/")), "./")
}

func (s *diffScanState) handleNewHeader(line string) {
	rawPath := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
	if rawPath == "/dev/null" {
		s.recordDeletedFile()
		s.targetMap = nil
		return
	}
	clean := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(rawPath, "b/")), "./")
	s.selectTargetMap(clean)
}

func (s *diffScanState) recordDeletedFile() {
	if !strings.HasSuffix(s.oldFile, ".go") {
		return
	}
	if IsTargetGoFile(s.oldFile) || IsTargetGoTestFile(s.oldFile) {
		s.deletedGoFiles = append(s.deletedGoFiles, s.oldFile)
		return
	}
	s.otherChangedGoFiles = append(s.otherChangedGoFiles, s.oldFile)
}

func (s *diffScanState) selectTargetMap(clean string) {
	switch {
	case IsTargetGoFile(clean):
		s.currentFile = clean
		s.targetMap = s.prodLines
	case IsTargetGoTestFile(clean):
		s.currentFile = clean
		s.targetMap = s.testLines
	case IsTargetProductionTSFile(clean):
		s.currentFile = clean
		s.targetMap = s.tsLines
	case IsTargetTSTestFile(clean):
		s.currentFile = clean
		s.targetMap = s.tsTestLines
	default:
		if strings.HasSuffix(clean, ".go") {
			s.otherChangedGoFiles = append(s.otherChangedGoFiles, clean)
		}
		s.targetMap = nil
	}
}

func (s *diffScanState) handleHunkHeader(line string) error {
	matches := hunkHeaderRegex.FindStringSubmatch(line)
	if matches == nil {
		s.inHunk = false
		return nil
	}
	start, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("failed to parse hunk start line in %q: %w", line, err)
	}
	s.newLine = start
	s.inHunk = s.targetMap != nil
	return nil
}

func (s *diffScanState) handleHunkLine(line string) {
	if line == "" {
		s.newLine++
		return
	}
	switch line[0] {
	case '+':
		lines, ok := s.targetMap[s.currentFile]
		if !ok {
			lines = make(map[int]struct{})
			s.targetMap[s.currentFile] = lines
		}
		lines[s.newLine] = struct{}{}
		s.newLine++
	case ' ':
		s.newLine++
	case '-', '\\':
		// Deleted lines and "\ No newline at end of file" markers do not advance the new-file line counter.
	}
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
	return DiffResult{
		VCS:                 "jj",
		BaseLabel:           base,
		ResolvedBaseRev:     resolvedRev,
		ChangedLines:        details.ProdLines,
		ChangedTestLines:    details.TestLines,
		TSChangedLines:      details.TSLines,
		TSChangedTestLines:  details.TSTestLines,
		DeletedGoFiles:      details.DeletedGoFiles,
		OtherChangedGoFiles: details.OtherChangedGoFiles,
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

	resolvedRev := base
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
		resolvedRev = diffBase
		args = []string{"diff", "--unified=0", diffBase}
	}

	diffBytes, err := RunCommandOutput(ctx, repoRoot, "git", args...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to run git %s: %w", strings.Join(args, " "), err)
	}

	details, err := ParseUnifiedDiffDetails(bytes.NewReader(diffBytes))
	if err != nil {
		return DiffResult{}, err
	}
	return DiffResult{
		VCS:                 "git",
		BaseLabel:           base,
		ResolvedBaseRev:     resolvedRev,
		ChangedLines:        details.ProdLines,
		ChangedTestLines:    details.TestLines,
		TSChangedLines:      details.TSLines,
		TSChangedTestLines:  details.TSTestLines,
		DeletedGoFiles:      details.DeletedGoFiles,
		OtherChangedGoFiles: details.OtherChangedGoFiles,
	}, nil
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
