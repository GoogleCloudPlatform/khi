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

package overlay

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// BaseReader loads a repository-relative file from the base VCS revision.
// It returns (nil, false, nil) when the file did not exist at the base revision.
type BaseReader func(relPath string) ([]byte, bool, error)

// Config specifies the repository paths and changed/deleted file sets to reconstruct at base.
type Config struct {
	// RepoRoot is the repository root directory path.
	RepoRoot string
	// WorkDir is the temporary directory where the base mirror is constructed.
	WorkDir string
	// ProdFiles lists changed production Go file paths relative to RepoRoot.
	ProdFiles []string
	// TestFiles lists changed Go test file paths relative to RepoRoot.
	TestFiles []string
	// DeletedGoFiles lists Go file paths deleted in the target revision.
	DeletedGoFiles []string
	// OtherChangedGoFiles lists other modified Go files excluded from quality checks.
	OtherChangedGoFiles []string
	// ReadBaseFile reads file contents at the base revision.
	ReadBaseFile BaseReader
}

// Snapshot holds a reconstructed base repository mirror where go test, go:embed, and cmd/cover execute hermetically.
type Snapshot struct {
	baseRoot    string
	baseSources map[string][]byte
}

type overlayJSON struct {
	Replace map[string]string `json:"Replace"`
}

type treeMirror struct {
	absRoot    string
	baseRoot   string
	changedSet map[string]struct{}
}

// Build constructs a hermetic regular-file base-revision workspace inside cfg.WorkDir.
func Build(cfg Config) (*Snapshot, error) {
	absRoot, err := filepath.Abs(cfg.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repo root: %w", err)
	}
	absWork, err := filepath.Abs(cfg.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workdir: %w", err)
	}

	paths := collectUniquePaths(cfg.ProdFiles, cfg.TestFiles, cfg.DeletedGoFiles, cfg.OtherChangedGoFiles)
	changedSet := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		changedSet[p] = struct{}{}
	}

	baseRoot := filepath.Join(absWork, "base_repo")
	if err := os.MkdirAll(baseRoot, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create base repo root: %w", err)
	}

	m := &treeMirror{
		absRoot:    absRoot,
		baseRoot:   baseRoot,
		changedSet: changedSet,
	}
	if err := m.mirrorDir("."); err != nil {
		return nil, err
	}

	snap := &Snapshot{
		baseRoot:    baseRoot,
		baseSources: make(map[string][]byte, len(paths)),
	}
	for _, relPath := range paths {
		if err := snap.stageBaseFile(relPath, cfg.ReadBaseFile); err != nil {
			return nil, err
		}
	}
	return snap, nil
}

// WriteOverlayJSON marshals replace into a Go -overlay JSON file named fileName inside dir.
func WriteOverlayJSON(dir string, fileName string, replace map[string]string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create overlay directory: %w", err)
	}
	data, err := json.MarshalIndent(overlayJSON{Replace: replace}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal overlay json: %w", err)
	}
	outPath := filepath.Join(dir, fileName)
	if err := os.WriteFile(outPath, data, 0o600); err != nil {
		return "", fmt.Errorf("failed to write overlay json: %w", err)
	}
	return outPath, nil
}

// RepoRoot returns the absolute path to the reconstructed base workspace root.
func (s *Snapshot) RepoRoot() string {
	return s.baseRoot
}

// BaseSource returns the base file bytes for relPath if relPath existed at the base revision.
func (s *Snapshot) BaseSource(relPath string) ([]byte, bool) {
	clean := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	src, ok := s.baseSources[clean]
	return src, ok
}

func (s *Snapshot) stageBaseFile(relPath string, readBase BaseReader) error {
	src, exists, err := readBase(relPath)
	if err != nil {
		return fmt.Errorf("failed to read base file %s: %w", relPath, err)
	}
	if !exists {
		return nil
	}
	dstAbs := filepath.Join(s.baseRoot, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return fmt.Errorf("failed to create base dir for %s: %w", relPath, err)
	}
	if err := os.WriteFile(dstAbs, src, 0o600); err != nil {
		return fmt.Errorf("failed to write base file %s: %w", relPath, err)
	}
	s.baseSources[relPath] = src
	return nil
}

func (m *treeMirror) mirrorDir(relDir string) error {
	srcDir := filepath.Join(m.absRoot, filepath.FromSlash(relDir))
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read dir %s: %w", relDir, err)
	}
	for _, entry := range entries {
		if err := m.mirrorEntry(relDir, entry); err != nil {
			return err
		}
	}
	return nil
}

func (m *treeMirror) mirrorEntry(relDir string, entry os.DirEntry) error {
	childRel := path.Join(relDir, entry.Name())
	dstChild := filepath.Join(m.baseRoot, filepath.FromSlash(childRel))

	if entry.IsDir() {
		if shouldSkipMirrorDir(childRel, entry.Name()) {
			return nil
		}
		if err := os.MkdirAll(dstChild, 0o755); err != nil {
			return fmt.Errorf("failed to create mirror dir %s: %w", childRel, err)
		}
		return m.mirrorDir(childRel)
	}
	if !entry.Type().IsRegular() {
		return nil
	}
	if _, changed := m.changedSet[childRel]; changed {
		return nil
	}
	origChild := filepath.Join(m.absRoot, filepath.FromSlash(childRel))
	return copyRegularFile(origChild, dstChild)
}

func shouldSkipMirrorDir(childRel string, name string) bool {
	return name == ".git" || name == ".jj" || name == "node_modules" || childRel == "web"
}

func copyRegularFile(srcPath string, dstPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", srcPath, err)
	}
	if err := os.WriteFile(dstPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", dstPath, err)
	}
	return nil
}

func collectUniquePaths(groups ...[]string) []string {
	seen := make(map[string]struct{})
	for _, g := range groups {
		for _, p := range g {
			clean := strings.TrimPrefix(filepath.ToSlash(p), "./")
			if clean != "" {
				seen[clean] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}
