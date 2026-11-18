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

package decisioncheck

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const (
	tsExtractModeDecls = "decls"
	tsExtractModeSpecs = "specs"
)

//go:embed extract_ts.js
var extractTSScript string

type tsWireFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type tsExtractRequest struct {
	WebDir string       `json:"webDir"`
	Mode   string       `json:"mode"`
	Files  []tsWireFile `json:"files"`
}

type tsRawItem struct {
	Line         int      `json:"line"`
	StartLine    int      `json:"startLine"`
	EndLine      int      `json:"endLine"`
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Doc          string   `json:"doc"`
	IsCallable   bool     `json:"isCallable"`
	MemberRanges [][2]int `json:"memberRanges"`
}

type tsExtractResponse struct {
	Files map[string][]tsRawItem `json:"files"`
}

type tsExtractSpec struct {
	mode      string
	matchFile func(string) bool
	keepItem  func(tsRawItem) bool
}

// ExtractChangedTSDecls extracts production TypeScript declarations (classes, interfaces, enums, types, functions, and methods) overlapping changedLines.
func ExtractChangedTSDecls(targetID string, webDir string, files []BundledFile, changedLines map[string]map[int]struct{}) ([]ExtractedItem, error) {
	return extractChangedTSItems(targetID, webDir, files, changedLines, tsExtractSpec{
		mode:      tsExtractModeDecls,
		matchFile: isProdTSFile,
		keepItem:  keepAllTSItems,
	})
}

// ExtractChangedTSFuncs extracts production TypeScript functions and class methods overlapping changedLines.
func ExtractChangedTSFuncs(targetID string, webDir string, files []BundledFile, changedLines map[string]map[int]struct{}) ([]ExtractedItem, error) {
	return extractChangedTSItems(targetID, webDir, files, changedLines, tsExtractSpec{
		mode:      tsExtractModeDecls,
		matchFile: isProdTSFile,
		keepItem:  keepCallableTSItems,
	})
}

// ExtractChangedTSSpecs extracts Jasmine spec blocks (it/fit/xit) in *.spec.ts files overlapping changedLines.
func ExtractChangedTSSpecs(targetID string, webDir string, files []BundledFile, changedLines map[string]map[int]struct{}) ([]ExtractedItem, error) {
	return extractChangedTSItems(targetID, webDir, files, changedLines, tsExtractSpec{
		mode:      tsExtractModeSpecs,
		matchFile: isSpecTSFile,
		keepItem:  keepAllTSItems,
	})
}

func extractChangedTSItems(
	targetID string,
	webDir string,
	files []BundledFile,
	changedLines map[string]map[int]struct{},
	spec tsExtractSpec,
) ([]ExtractedItem, error) {
	wireFiles := filterTSWireFiles(files, changedLines, spec.matchFile)
	if len(wireFiles) == 0 {
		return nil, nil
	}
	resolvedWebDir, err := gotestutil.ResolveWebDir(webDir)
	if err != nil {
		return nil, err
	}
	// tsExtractRequest contains only strings and slices, so json.Marshal cannot fail.
	payload, _ := json.Marshal(tsExtractRequest{
		WebDir: resolvedWebDir,
		Mode:   spec.mode,
		Files:  wireFiles,
	})
	rawOutput, err := runTSNodeScript(payload)
	if err != nil {
		return nil, err
	}
	var parsed tsExtractResponse
	if err := json.Unmarshal(rawOutput, &parsed); err != nil {
		return nil, fmt.Errorf("failed decoding TypeScript AST output: %w", err)
	}
	return collectExtractedTSItems(targetID, wireFiles, &parsed, changedLines, spec.keepItem), nil
}

func filterTSWireFiles(files []BundledFile, changedLines map[string]map[int]struct{}, matchFile func(string) bool) []tsWireFile {
	var wireFiles []tsWireFile
	for _, file := range files {
		relPath := filepath.ToSlash(file.RelPath)
		if !matchFile(relPath) {
			continue
		}
		if changedLines != nil && len(changedLines[relPath]) == 0 {
			continue
		}
		wireFiles = append(wireFiles, tsWireFile{
			Path:    relPath,
			Content: string(file.Content),
		})
	}
	return wireFiles
}

func runTSNodeScript(payload []byte) ([]byte, error) {
	cmd := exec.CommandContext(context.Background(), "node", "-e", extractTSScript)
	cmd.Stdin = bytes.NewReader(payload)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return nil, formatTSExtractorError(err, stderrBuf.String())
	}
	return stdoutBuf.Bytes(), nil
}

func formatTSExtractorError(err error, stderr string) error {
	if errMsg := strings.TrimSpace(stderr); errMsg != "" {
		return fmt.Errorf("typescript AST extractor failed (%w): %s", err, errMsg)
	}
	return fmt.Errorf("typescript AST extractor failed: %w", err)
}

func collectExtractedTSItems(
	targetID string,
	wireFiles []tsWireFile,
	resp *tsExtractResponse,
	changedLines map[string]map[int]struct{},
	keepItem func(tsRawItem) bool,
) []ExtractedItem {
	var out []ExtractedItem
	for _, wireFile := range wireFiles {
		lineSet := changedLines[wireFile.Path]
		for _, rawItem := range resp.Files[wireFile.Path] {
			if !keepItem(rawItem) || !rawOverlapsChangedLines(rawItem, lineSet) {
				continue
			}
			out = append(out, ExtractedItem{
				Target:    targetID,
				File:      wireFile.Path,
				Line:      rawItem.Line,
				StartLine: rawItem.StartLine,
				EndLine:   rawItem.EndLine,
				Name:      rawItem.Name,
				Kind:      rawItem.Kind,
				Doc:       rawItem.Doc,
			})
		}
	}
	return out
}

func rawOverlapsChangedLines(rawItem tsRawItem, lineSet map[int]struct{}) bool {
	if lineSet == nil {
		return true
	}
	for lineNum := rawItem.StartLine; lineNum <= rawItem.EndLine; lineNum++ {
		if _, touched := lineSet[lineNum]; touched && !lineInsideRanges(lineNum, rawItem.MemberRanges) {
			return true
		}
	}
	return false
}

func lineInsideRanges(lineNum int, ranges [][2]int) bool {
	for _, span := range ranges {
		if lineNum >= span[0] && lineNum <= span[1] {
			return true
		}
	}
	return false
}

func keepAllTSItems(tsRawItem) bool {
	return true
}

func keepCallableTSItems(rawItem tsRawItem) bool {
	return rawItem.IsCallable
}

func isProdTSFile(relPath string) bool {
	return strings.HasSuffix(relPath, ".ts") &&
		!strings.HasSuffix(relPath, ".spec.ts") &&
		!strings.HasSuffix(relPath, ".d.ts") &&
		!strings.HasSuffix(relPath, ".stories.ts")
}

func isSpecTSFile(relPath string) bool {
	return strings.HasSuffix(relPath, ".spec.ts")
}
