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
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderRegex = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// UnifiedDiffDetails holds parsed line maps and deleted/non-target Go files extracted from a unified diff stream.
type UnifiedDiffDetails struct {
	// ProdLines maps modified Go production file paths to changed 1-based line sets.
	ProdLines map[string]map[int]struct{}
	// TestLines maps modified Go test file paths to changed 1-based line sets.
	TestLines map[string]map[int]struct{}
	// TSLines maps modified TypeScript production file paths to changed 1-based line sets.
	TSLines map[string]map[int]struct{}
	// TSTestLines maps modified TypeScript spec file paths to changed 1-based line sets.
	TSTestLines map[string]map[int]struct{}
	// DeletedGoFiles lists Go file paths deleted in the diff.
	DeletedGoFiles []string
	// OtherChangedGoFiles lists modified Go files outside primary analysis targets.
	OtherChangedGoFiles []string
	// ChangedFiles lists all added or modified relative file paths in deterministic order.
	ChangedFiles []string
	// DeletedFiles lists all deleted relative file paths in deterministic order.
	DeletedFiles []string
}

type diffScanState struct {
	prodLines           map[string]map[int]struct{}
	testLines           map[string]map[int]struct{}
	tsLines             map[string]map[int]struct{}
	tsTestLines         map[string]map[int]struct{}
	deletedGoFiles      []string
	otherChangedGoFiles []string
	changedFiles        map[string]struct{}
	deletedFiles        map[string]struct{}
	oldFile             string
	currentFile         string
	targetMap           map[string]map[int]struct{}
	inHunk              bool
	newLine             int
}

// ParseUnifiedDiffDetails extracts changed line numbers for Go, *_test.go, TypeScript, and *.spec.ts files plus deleted Go file paths.
func ParseUnifiedDiffDetails(r io.Reader) (UnifiedDiffDetails, error) {
	state := &diffScanState{
		prodLines:    make(map[string]map[int]struct{}),
		testLines:    make(map[string]map[int]struct{}),
		tsLines:      make(map[string]map[int]struct{}),
		tsTestLines:  make(map[string]map[int]struct{}),
		changedFiles: make(map[string]struct{}),
		deletedFiles: make(map[string]struct{}),
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
		ChangedFiles:        sortedSetKeys(state.changedFiles),
		DeletedFiles:        sortedSetKeys(state.deletedFiles),
	}, nil
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
	if clean != "" {
		s.changedFiles[clean] = struct{}{}
	}
	s.selectTargetMap(clean)
}

func (s *diffScanState) recordDeletedFile() {
	if s.oldFile != "" {
		s.deletedFiles[s.oldFile] = struct{}{}
	}
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
