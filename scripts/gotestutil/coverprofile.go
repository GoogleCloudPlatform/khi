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
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// CoverBlock represents a single statement block entry from a Go coverprofile.
type CoverBlock struct {
	StartLine     int
	StartCol      int
	EndLine       int
	EndCol        int
	NumStatements int
	Count         int
}

// ResolveRepoRoot locates the Go module root starting from explicit or the current working directory.
func ResolveRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd, nil
		}
		dir = parent
	}
}

// ReadModulePath extracts the Go module path from a go.mod file.
func ReadModulePath(goModPath string) (string, error) {
	f, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %w", goModPath, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("failed reading %s: %w", goModPath, err)
	}
	return "", fmt.Errorf("module directive not found in %s", goModPath)
}

// LoadCoverProfileIfPresent reads and parses profilePath, returning an empty map if the file does not exist.
func LoadCoverProfileIfPresent(profilePath string, modulePath string, filter FilterOptions) (map[string][]CoverBlock, error) {
	f, err := os.Open(profilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string][]CoverBlock), nil
		}
		return nil, fmt.Errorf("failed to open coverprofile %s: %w", profilePath, err)
	}
	defer f.Close()

	return ParseCoverProfileWithFilter(f, modulePath, filter)
}

type blockKey struct {
	file      string
	startLine int
	startCol  int
	endLine   int
	endCol    int
	numStmt   int
}

// ParseCoverProfile parses a Go coverprofile stream for default production Go files.
func ParseCoverProfile(r io.Reader, modulePath string) (map[string][]CoverBlock, error) {
	return ParseCoverProfileWithFilter(r, modulePath, FilterOptions{})
}

// ParseCoverProfileWithFilter parses a Go coverprofile stream, merging duplicate block entries and filtering files by filter.
func ParseCoverProfileWithFilter(r io.Reader, modulePath string, filter FilterOptions) (map[string][]CoverBlock, error) {
	merged := make(map[blockKey]int)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	modulePrefix := strings.TrimSuffix(modulePath, "/") + "/"
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		colonIdx := strings.LastIndexByte(line, ':')
		if colonIdx <= 0 {
			return nil, fmt.Errorf("invalid coverprofile line %d: missing file or ':' in %q", lineNum, line)
		}
		rawFile := line[:colonIdx]
		rest := line[colonIdx+1:]

		var relFile string
		if strings.HasPrefix(rawFile, modulePrefix) {
			relFile = strings.TrimPrefix(rawFile, modulePrefix)
		} else {
			relFile = strings.TrimPrefix(filepath.ToSlash(rawFile), "./")
		}
		if !IsTargetGoFile(relFile, filter) {
			continue
		}

		fields := strings.Fields(rest)
		if len(fields) != 3 {
			return nil, fmt.Errorf("invalid coverprofile line %d: unexpected fields in %q", lineNum, line)
		}
		startLine, startCol, endLine, endCol, err := parseBlockCoords(fields[0])
		if err != nil {
			return nil, fmt.Errorf("invalid coverprofile line %d: %w", lineNum, err)
		}
		numStmt, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("invalid statement count on line %d: %w", lineNum, err)
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("invalid execution count on line %d: %w", lineNum, err)
		}

		key := blockKey{
			file:      relFile,
			startLine: startLine,
			startCol:  startCol,
			endLine:   endLine,
			endCol:    endCol,
			numStmt:   numStmt,
		}
		merged[key] += count
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("failed scanning coverprofile: %w", err)
	}

	byFile := make(map[string][]CoverBlock)
	for k, count := range merged {
		byFile[k.file] = append(byFile[k.file], CoverBlock{
			StartLine:     k.startLine,
			StartCol:      k.startCol,
			EndLine:       k.endLine,
			EndCol:        k.endCol,
			NumStatements: k.numStmt,
			Count:         count,
		})
	}
	for file := range byFile {
		slices.SortFunc(byFile[file], func(a, b CoverBlock) int {
			return cmp.Or(
				cmp.Compare(a.StartLine, b.StartLine),
				cmp.Compare(a.StartCol, b.StartCol),
				cmp.Compare(a.EndLine, b.EndLine),
				cmp.Compare(a.EndCol, b.EndCol),
			)
		})
	}
	return byFile, nil
}

// CoveredLineSet returns the set of 1-based line numbers touched by at least one executed CoverBlock.
func CoveredLineSet(blocks []CoverBlock) map[int]struct{} {
	covered := make(map[int]struct{})
	for _, b := range blocks {
		if b.Count <= 0 {
			continue
		}
		for line := b.StartLine; line <= b.EndLine; line++ {
			covered[line] = struct{}{}
		}
	}
	return covered
}

func parseBlockCoords(spec string) (int, int, int, int, error) {
	parts := strings.Split(spec, ",")
	if len(parts) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("expected start,end coordinates in %q", spec)
	}
	sLine, sCol, err := parseLineCol(parts[0])
	if err != nil {
		return 0, 0, 0, 0, err
	}
	eLine, eCol, err := parseLineCol(parts[1])
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return sLine, sCol, eLine, eCol, nil
}

func parseLineCol(s string) (int, int, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected line.col in %q", s)
	}
	line, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	col, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return line, col, nil
}
