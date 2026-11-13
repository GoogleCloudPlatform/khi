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

package langgo

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

var _ simcheck.LanguageAnalyzer = (*Analyzer)(nil)

// Analyzer implements simcheck.LanguageAnalyzer for Go source files using go/parser and go/ast.
type Analyzer struct{}

// NewAnalyzer returns a Go LanguageAnalyzer instance.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// ID returns the canonical language identifier "go".
func (a *Analyzer) ID() simcheck.LanguageID {
	return simcheck.LangGo
}

// DisplayName returns the human-readable language title "Go".
func (a *Analyzer) DisplayName() string {
	return "Go"
}

// CodeFence returns the Markdown code fence tag "go".
func (a *Analyzer) CodeFence() string {
	return "go"
}

// IsEligibleFile reports whether relPath is a non-test, non-generated production Go source file.
func (a *Analyzer) IsEligibleFile(relPath string) bool {
	clean := filepath.ToSlash(relPath)
	return gotestutil.IsTargetGoFile(clean) && !strings.HasSuffix(clean, ".pb.go")
}

// DiscoverCorpusFiles walks repoRoot via astcheck.DiscoverProdGoFiles and returns eligible production Go file paths.
func (a *Analyzer) DiscoverCorpusFiles(repoRoot string) ([]string, error) {
	candidates, err := astcheck.DiscoverProdGoFiles(repoRoot)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(candidates))
	for _, rel := range candidates {
		if a.IsEligibleFile(rel) {
			files = append(files, rel)
		}
	}
	return files, nil
}

// ExtractUnits parses Go files in memory and returns qualifying functions, methods, and task closures per file.
func (a *Analyzer) ExtractUnits(_ context.Context, files []simcheck.FileInput, minStmts int, minCodeLines int) (map[string][]simcheck.Unit, error) {
	result := make(map[string][]simcheck.Unit, len(files))
	for _, f := range files {
		units, err := ExtractUnitsFromSource(f.Path, f.Content, minStmts, minCodeLines)
		if err != nil {
			return nil, err
		}
		if len(units) > 0 {
			result[f.Path] = units
		}
	}
	return result, nil
}

// ExtractBaseKeys parses base-revision Go files without size filtering and returns unit key sets per file.
func (a *Analyzer) ExtractBaseKeys(_ context.Context, files []simcheck.FileInput) (map[string]map[string]struct{}, error) {
	result := make(map[string]map[string]struct{}, len(files))
	for _, f := range files {
		units, err := ExtractUnitsFromSource(f.Path, f.Content, 0, 0)
		if err != nil {
			return nil, err
		}
		keys := make(map[string]struct{}, len(units))
		for _, u := range units {
			keys[u.Key] = struct{}{}
		}
		result[f.Path] = keys
	}
	return result, nil
}

// IsSharedModule reports whether modulePath is a shared Go utility or domain-common package meant for cross-package reuse.
func (a *Analyzer) IsSharedModule(modulePath string) bool {
	clean := strings.Trim(modulePath, "/")
	if clean == "pkg/common" || strings.HasPrefix(clean, "pkg/common/") {
		return true
	}
	if clean == "scripts/gotestutil" || strings.HasPrefix(clean, "scripts/gotestutil/") {
		return true
	}
	segments := strings.Split(clean, "/")
	last := segments[len(segments)-1]
	if last == "impl" {
		return false
	}
	return strings.HasSuffix(last, "common") || clean == "pkg/task/inspection/common/k8saudit"
}
