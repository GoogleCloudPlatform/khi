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

package langts

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

//go:embed extract_ts.js
var extractTSScript string

var ignoredAngularSrcDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"coverage":     true,
	"generated":    true,
	"assets":       true,
	"environments": true,
}

var _ simcheck.LanguageAnalyzer = (*Analyzer)(nil)

// Analyzer implements simcheck.LanguageAnalyzer for TypeScript and Angular source files.
type Analyzer struct {
	webDir string
}

type wireFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type batchRequest struct {
	WebDir       string     `json:"webDir"`
	MinStmts     int        `json:"minStmts"`
	MinCodeLines int        `json:"minCodeLines"`
	KeysOnly     bool       `json:"keysOnly"`
	Files        []wireFile `json:"files"`
}

type rawExtractedUnit struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	StmtCount int    `json:"stmtCount"`
	CodeLines int    `json:"codeLines"`
	RawDecl   string `json:"rawDecl"`
	NormText  string `json:"normText"`
}

type batchFileEntry struct {
	Keys  []string           `json:"keys"`
	Units []rawExtractedUnit `json:"units"`
}

type batchResponse struct {
	Files map[string]batchFileEntry `json:"files"`
}

// NewAnalyzer constructs a TypeScript LanguageAnalyzer resolving the TypeScript compiler from repoRoot/web.
func NewAnalyzer(repoRoot string) *Analyzer {
	webDir := ""
	if repoRoot != "" {
		webDir = filepath.Join(repoRoot, "web")
	}
	return &Analyzer{
		webDir: webDir,
	}
}

// ID returns the canonical language identifier "ts".
func (a *Analyzer) ID() simcheck.LanguageID {
	return simcheck.LangTS
}

// DisplayName returns the human-readable language title "TypeScript".
func (a *Analyzer) DisplayName() string {
	return "TypeScript"
}

// CodeFence returns the Markdown code fence tag "ts".
func (a *Analyzer) CodeFence() string {
	return "ts"
}

// IsEligibleFile reports whether relPath is a non-spec, non-generated production TypeScript source file.
func (a *Analyzer) IsEligibleFile(relPath string) bool {
	return gotestutil.IsTargetProductionTSFile(filepath.ToSlash(relPath))
}

// DiscoverCorpusFiles walks web/src under repoRoot and returns all eligible production TypeScript file paths.
func (a *Analyzer) DiscoverCorpusFiles(repoRoot string) ([]string, error) {
	webSrc := filepath.Join(repoRoot, "web", "src")
	if _, err := os.Stat(webSrc); os.IsNotExist(err) {
		return nil, nil
	}
	var files []string
	err := filepath.WalkDir(webSrc, func(fullPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relRaw, err := filepath.Rel(repoRoot, fullPath)
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(relRaw)
		if d.IsDir() {
			if isIgnoredAngularSrcDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if a.IsEligibleFile(rel) {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}

// ExtractUnits parses TypeScript files in a single batched Node invocation and returns qualifying units per file.
func (a *Analyzer) ExtractUnits(ctx context.Context, files []simcheck.FileInput, minStmts int, minCodeLines int) (map[string][]simcheck.Unit, error) {
	if len(files) == 0 {
		return nil, nil
	}
	resp, err := a.runBatchExtractor(ctx, batchRequest{
		MinStmts:     minStmts,
		MinCodeLines: minCodeLines,
		KeysOnly:     false,
		Files:        toWireFiles(files),
	})
	if err != nil {
		return nil, err
	}

	out := make(map[string][]simcheck.Unit, len(resp.Files))
	for relPath, entry := range resp.Files {
		modPath := filepath.ToSlash(filepath.Dir(relPath))
		converted := make([]simcheck.Unit, 0, len(entry.Units))
		for _, raw := range entry.Units {
			kind := simcheck.UnitKind(raw.Kind)
			converted = append(converted, simcheck.Unit{
				ID:         fmt.Sprintf("%s:%s", relPath, raw.Name),
				Key:        fmt.Sprintf("%s:%s", kind, raw.Name),
				Language:   simcheck.LangTS,
				ModulePath: modPath,
				FilePath:   relPath,
				StartLine:  raw.StartLine,
				EndLine:    raw.EndLine,
				Name:       raw.Name,
				Kind:       kind,
				StmtCount:  raw.StmtCount,
				CodeLines:  raw.CodeLines,
				NormText:   raw.NormText,
				Preview:    simcheck.BuildPreviewLines(raw.RawDecl, raw.StartLine),
			})
		}
		out[relPath] = converted
	}
	return out, nil
}

// ExtractBaseKeys parses base-revision TypeScript files in a single batched Node invocation and returns existing unit keys.
func (a *Analyzer) ExtractBaseKeys(ctx context.Context, files []simcheck.FileInput) (map[string]map[string]struct{}, error) {
	if len(files) == 0 {
		return nil, nil
	}
	resp, err := a.runBatchExtractor(ctx, batchRequest{
		KeysOnly: true,
		Files:    toWireFiles(files),
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string]map[string]struct{}, len(resp.Files))
	for relPath, entry := range resp.Files {
		set := make(map[string]struct{}, len(entry.Keys))
		for _, k := range entry.Keys {
			set[k] = struct{}{}
		}
		out[relPath] = set
	}
	return out, nil
}

// IsSharedModule reports whether modulePath is a shared frontend utility or common directory meant for cross-feature reuse.
func (a *Analyzer) IsSharedModule(modulePath string) bool {
	clean := strings.Trim(filepath.ToSlash(modulePath), "/")
	sharedPrefixes := []string{
		"web/src/app/common",
		"web/src/app/utils",
		"web/src/app/shared",
	}
	for _, p := range sharedPrefixes {
		if clean == p || strings.HasPrefix(clean, p+"/") {
			return true
		}
	}
	return false
}

func (a *Analyzer) runBatchExtractor(ctx context.Context, req batchRequest) (*batchResponse, error) {
	webDir, err := a.resolveWebDir()
	if err != nil {
		return nil, err
	}
	req.WebDir = webDir
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed encoding TypeScript extraction request: %w", err)
	}

	cmd := exec.CommandContext(ctx, "node", "-e", extractTSScript)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return nil, fmt.Errorf("typescript AST extractor failed (%w): %s", err, errMsg)
		}
		return nil, fmt.Errorf("typescript AST extractor failed: %w", err)
	}

	var parsed batchResponse
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		return nil, fmt.Errorf("failed decoding TypeScript AST output: %w", err)
	}
	return &parsed, nil
}

func (a *Analyzer) resolveWebDir() (string, error) {
	if a.webDir != "" {
		if _, err := os.Stat(filepath.Join(a.webDir, "package.json")); err == nil {
			return a.webDir, nil
		}
	}
	repoRoot, err := gotestutil.ResolveRepoRoot("")
	if err != nil {
		return "", fmt.Errorf("could not locate web/package.json for TypeScript parser: %w", err)
	}
	return filepath.Join(repoRoot, "web"), nil
}

func isIgnoredAngularSrcDir(baseName string) bool {
	return ignoredAngularSrcDirs[baseName] || (strings.HasPrefix(baseName, ".") && baseName != ".")
}

func toWireFiles(files []simcheck.FileInput) []wireFile {
	out := make([]wireFile, len(files))
	for i, f := range files {
		out[i] = wireFile{
			Path:    filepath.ToSlash(f.Path),
			Content: string(f.Content),
		}
	}
	return out
}
