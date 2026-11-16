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

package simcheck

import (
	"context"
	"fmt"
	"strings"
)

const (
	// LangGo identifies Go source code.
	LangGo LanguageID = "go"
	// LangTS identifies TypeScript source code.
	LangTS LanguageID = "ts"
)

// LanguageID identifies a supported source language in CLI flags and reports.
type LanguageID string

// FileInput pairs a repository-relative file path with raw UTF-8 source bytes for in-memory batch AST parsing.
type FileInput struct {
	// Path is the slash-separated path relative to the repository root.
	Path string
	// Content holds raw UTF-8 source bytes of the file.
	Content []byte
}

// LanguageAnalyzer encapsulates language-specific file discovery, AST extraction, identifier normalization, and module rules.
type LanguageAnalyzer interface {
	// ID returns the canonical short language identifier (such as "go" or "ts").
	ID() LanguageID
	// DisplayName returns the human-readable language title (such as "Go" or "TypeScript").
	DisplayName() string
	// CodeFence returns the Markdown syntax-highlighting tag (such as "go" or "ts").
	CodeFence() string
	// IsEligibleFile reports whether relPath is a non-test, non-generated production source file for this language.
	IsEligibleFile(relPath string) bool
	// DiscoverCorpusFiles walks repoRoot and returns all eligible production source file paths for this language.
	DiscoverCorpusFiles(repoRoot string) ([]string, error)
	// ExtractUnits parses a batch of files into normalized callable units filtered by minStmts and minCodeLines.
	ExtractUnits(ctx context.Context, files []FileInput, minStmts int, minCodeLines int) (map[string][]Unit, error)
	// ExtractBaseKeys parses base-revision files without size filtering and returns existing unit keys per file.
	ExtractBaseKeys(ctx context.Context, files []FileInput) (map[string]map[string]struct{}, error)
	// IsSharedModule reports whether modulePath is a shared utility package or directory prioritized for reuse.
	IsSharedModule(modulePath string) bool
}

// Registry maintains an ordered collection of LanguageAnalyzer implementations for multi-language execution.
type Registry struct {
	order []LanguageID
	byID  map[LanguageID]LanguageAnalyzer
}

// NewRegistry constructs a Registry populated with the provided analyzers in registration order.
func NewRegistry(analyzers ...LanguageAnalyzer) *Registry {
	r := &Registry{
		byID: make(map[LanguageID]LanguageAnalyzer, len(analyzers)),
	}
	for _, a := range analyzers {
		r.Register(a)
	}
	return r
}

// Register adds or replaces a LanguageAnalyzer in the registry while preserving registration order.
func (r *Registry) Register(a LanguageAnalyzer) {
	if a == nil {
		return
	}
	id := a.ID()
	if _, exists := r.byID[id]; !exists {
		r.order = append(r.order, id)
	}
	r.byID[id] = a
}

// Resolve returns the matching analyzers for langFlag ("all" or a specific registered LanguageID).
func (r *Registry) Resolve(langFlag string) ([]LanguageAnalyzer, error) {
	key := strings.TrimSpace(strings.ToLower(langFlag))
	if key == "" || key == "all" {
		out := make([]LanguageAnalyzer, 0, len(r.order))
		for _, id := range r.order {
			out = append(out, r.byID[id])
		}
		return out, nil
	}
	if a, ok := r.byID[LanguageID(key)]; ok {
		return []LanguageAnalyzer{a}, nil
	}
	allowed := make([]string, 0, len(r.order)+1)
	allowed = append(allowed, "all")
	for _, id := range r.order {
		allowed = append(allowed, string(id))
	}
	return nil, fmt.Errorf("invalid --lang %q (want %s)", langFlag, strings.Join(allowed, ", "))
}
