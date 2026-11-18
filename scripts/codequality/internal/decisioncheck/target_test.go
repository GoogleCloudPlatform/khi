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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLogicalGoPackageDir(t *testing.T) {
	testCases := []struct {
		name    string
		relPath string
		want    string
	}{
		// Maps files inside an impl/ subdirectory back to the parent logical feature package root.
		{
			name:    "impl file resolves to parent feature root",
			relPath: "pkg/task/inspection/common/k8saudit/impl/module.go",
			want:    "pkg/task/inspection/common/k8saudit",
		},
		// Maps direct files inside an internal/ directory back to the enclosing logical package root.
		{
			name:    "direct internal file resolves to parent package root",
			relPath: "pkg/task/feat/internal/helper.go",
			want:    "pkg/task/feat",
		},
		// Maps nested internal/ helper files back to the enclosing logical package root.
		{
			name:    "nested internal helper resolves to parent package root",
			relPath: "pkg/task/inspection/common/k8saudit/internal/parse/parse.go",
			want:    "pkg/task/inspection/common/k8saudit",
		},
		// Preserves top-level /internal paths without an enclosing package prefix and without trimming /impl.
		{
			name:    "leading slash internal path keeps full directory",
			relPath: "/internal/foo/bar.go",
			want:    "/internal/foo",
		},
		// Returns an empty string for files placed at the repository root.
		{
			name:    "repository root file returns empty package dir",
			relPath: "README.md",
			want:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := LogicalGoPackageDir(tc.relPath)

			if got != tc.want {
				t.Errorf("LogicalGoPackageDir(%q) = %q, want %q", tc.relPath, got, tc.want)
			}
		})
	}
}

// TestResolveTargetsUnsupportedKind verifies that ResolveTargets returns an explicit error for unknown target kinds.
func TestResolveTargetsUnsupportedKind(t *testing.T) {
	repo := t.TempDir()
	spec := RuleSpec{
		ID:          "bad_target",
		Name:        "Bad Target",
		Enforcement: EnforcementAdvisory,
		Target:      TargetSpec{Kind: TargetKind("unknown")},
	}

	_, err := ResolveTargets(spec, ResolveOptions{RepoRoot: repo})

	if err == nil {
		t.Fatal("ResolveTargets() error = nil, want unsupported target kind error")
	}
	if !strings.Contains(err.Error(), "unsupported target kind") {
		t.Errorf("ResolveTargets() error = %q, want substring %q", err.Error(), "unsupported target kind")
	}
}

func TestResolveLogicalPackageTargets(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(t *testing.T) string
		rule        RuleSpec
		opts        func(repo string) ResolveOptions
		wantBundles []TargetBundle
	}{
		// Resolves changed and deleted package paths using default include_files, require_files primary path, changed line preservation, and directory exclusions.
		{
			name: "changed and deleted packages with defaults and exclusions",
			setup: func(t *testing.T) string {
				repo := t.TempDir()
				writeRepoFile(t, repo, "README.md", "# Root\n")
				writeRepoFile(t, repo, "pkg/alpha/A_alpha.go", "package alpha; type Config struct{}\n")
				writeRepoFile(t, repo, "pkg/alpha/README.md", "# Alpha\n")
				writeRepoFile(t, repo, "pkg/alpha/impl/module.go", "package alpha_impl; var Module = M{Tasks: []T{alphaTask}}\n")
				writeRepoFile(t, repo, "pkg/alpha/impl/module_test.go", "package alpha_impl\n")
				writeRepoFile(t, repo, "pkg/alpha/internal/helper.go", "package alpha_internal; func helper() {}\n")
				writeRepoFile(t, repo, "pkg/alpha/internal/node_modules/ignored.go", "package ignored\n")
				writeRepoFile(t, repo, "pkg/alpha/internal/testdata/fixture.go", "package testdata\n")
				writeRepoFile(t, repo, "pkg/alpha/notes.txt", "ignored\n")
				writeRepoFile(t, repo, "pkg/beta/A_keep.go", "package beta; type Keep struct{}\n")
				writeRepoFile(t, repo, "pkg/beta/README.md", "# Beta\n")
				writeRepoFile(t, repo, "pkg/nodoc/taskid.go", "package nodoc\n")
				writeRepoFile(t, repo, "pkg/dirreq/README.md/nested.txt", "not a file\n")
				writeRepoFile(t, repo, "pkg/asfile", "regular file blocking dir\n")
				writeRepoFile(t, repo, "pkg/untouched/u.go", "package untouched\n")
				writeRepoFile(t, repo, "pkg/untouched/README.md", "# Untouched\n")
				writeRepoFile(t, repo, "cmd/outside/main.go", "package main\n")
				writeRepoFile(t, repo, "cmd/outside/README.md", "# Outside\n")
				return repo
			},
			rule: RuleSpec{
				ID:          "pkg_doc",
				Name:        "Pkg Doc",
				Enforcement: EnforcementAdvisory,
				Target: TargetSpec{
					Kind:         TargetGoLogicalPackage,
					Include:      []string{"pkg/**"},
					Exclude:      []string{"**/testdata/**"},
					RequireFiles: []string{"README.md"},
				},
				Context: ContextSpec{
					ExcludeFiles: []string{"**/*_test.go"},
				},
			},
			opts: func(repo string) ResolveOptions {
				return ResolveOptions{
					RepoRoot:       repo,
					ChangedFiles:   []string{"README.md", "pkg/alpha/impl/module.go", "pkg/nodoc/taskid.go", "pkg/dirreq/README.md", "pkg/asfile/phantom.go", "cmd/outside/main.go"},
					DeletedFiles:   []string{"pkg/beta/removed.go"},
					ChangedGoLines: map[string]map[int]struct{}{"pkg/alpha/A_alpha.go": {1: {}}},
				}
			},
			wantBundles: []TargetBundle{
				{
					ID:          "pkg/alpha",
					PrimaryFile: "pkg/alpha/README.md",
					Files: []BundledFile{
						{RelPath: "pkg/alpha/A_alpha.go", Content: []byte("package alpha; type Config struct{}\n")},
						{RelPath: "pkg/alpha/README.md", Content: []byte("# Alpha\n")},
						{RelPath: "pkg/alpha/impl/module.go", Content: []byte("package alpha_impl; var Module = M{Tasks: []T{alphaTask}}\n")},
						{RelPath: "pkg/alpha/internal/helper.go", Content: []byte("package alpha_internal; func helper() {}\n")},
					},
					ASTSummary: `- pkg/alpha/A_alpha.go (package alpha)
  type Config (pkg/alpha/A_alpha.go:1)
- pkg/alpha/impl/module.go (package alpha_impl)
- pkg/alpha/internal/helper.go (package alpha_internal)
`,
					SubstantiveGoUnits: []ExtractedItem{
						{Target: "pkg/alpha", File: "pkg/alpha/impl/module.go", Line: 1, StartLine: 1, EndLine: 1, Name: "alphaTask", Kind: "registered DAG task"},
					},
					ChangedLines: map[string]map[int]struct{}{"pkg/alpha/A_alpha.go": {1: {}}},
				},
				{
					ID:          "pkg/beta",
					PrimaryFile: "pkg/beta/README.md",
					Files: []BundledFile{
						{RelPath: "pkg/beta/A_keep.go", Content: []byte("package beta; type Keep struct{}\n")},
						{RelPath: "pkg/beta/README.md", Content: []byte("# Beta\n")},
					},
					ASTSummary: `- pkg/beta/A_keep.go (package beta)
  type Keep (pkg/beta/A_keep.go:1)
`,
					SubstantiveGoUnits: []ExtractedItem{
						{Target: "pkg/beta", File: "pkg/beta/A_keep.go", Line: 1, StartLine: 1, EndLine: 1, Name: "A_keep.go (Keep)", Kind: "Go source file"},
					},
					ChangedLines: map[string]map[int]struct{}{"pkg/alpha/A_alpha.go": {1: {}}},
				},
			},
		},
		// Respects custom include_files, preserves a package dir named coverage, skips root files, VCS metadata paths, and empty packages, defaults PrimaryFile without require_files, and enforces MaxTargets.
		{
			name: "custom include_files, pkg/coverage preservation, empty package skip, VCS skip, and max_targets cap",
			setup: func(t *testing.T) string {
				repo := t.TempDir()
				writeRepoFile(t, repo, "README.md", "# Root\n")
				writeRepoFile(t, repo, ".jj/state/x.go", "package jj\n")
				writeRepoFile(t, repo, "pkg/a_empty/notes.txt", "only text\n")
				writeRepoFile(t, repo, "pkg/coverage/cov.go", "package coverage; type Cov struct{}\n")
				writeRepoFile(t, repo, "pkg/coverage/internal/skip.go", "package coverage_internal\n")
				writeRepoFile(t, repo, "pkg/delta/delta.go", "package delta; type Delta struct{}\n")
				writeRepoFile(t, repo, "pkg/echo/echo.go", "package echo; type Echo struct{}\n")
				return repo
			},
			rule: RuleSpec{
				ID:          "pkg_go",
				Name:        "Pkg Go",
				Enforcement: EnforcementAdvisory,
				Target:      TargetSpec{Kind: TargetGoLogicalPackage},
				Context:     ContextSpec{IncludeFiles: []string{"*.go"}},
			},
			opts: func(repo string) ResolveOptions {
				return ResolveOptions{
					RepoRoot:     repo,
					MaxTargets:   2,
					ChangedFiles: []string{"README.md", ".jj/state/x.go", "pkg/a_empty/notes.txt", "pkg/coverage/cov.go", "pkg/delta/delta.go", "pkg/echo/echo.go"},
					DeletedFiles: []string{"pkg/gone/old.go"},
				}
			},
			wantBundles: []TargetBundle{
				{
					ID:          "pkg/coverage",
					PrimaryFile: "pkg/coverage/cov.go",
					Files:       []BundledFile{{RelPath: "pkg/coverage/cov.go", Content: []byte("package coverage; type Cov struct{}\n")}},
					ASTSummary: `- pkg/coverage/cov.go (package coverage)
  type Cov (pkg/coverage/cov.go:1)
`,
					SubstantiveGoUnits: []ExtractedItem{
						{Target: "pkg/coverage", File: "pkg/coverage/cov.go", Line: 1, StartLine: 1, EndLine: 1, Name: "cov.go (Cov)", Kind: "Go source file"},
					},
				},
				{
					ID:          "pkg/delta",
					PrimaryFile: "pkg/delta/delta.go",
					Files:       []BundledFile{{RelPath: "pkg/delta/delta.go", Content: []byte("package delta; type Delta struct{}\n")}},
					ASTSummary: `- pkg/delta/delta.go (package delta)
  type Delta (pkg/delta/delta.go:1)
`,
					SubstantiveGoUnits: []ExtractedItem{
						{Target: "pkg/delta", File: "pkg/delta/delta.go", Line: 1, StartLine: 1, EndLine: 1, Name: "delta.go (Delta)", Kind: "Go source file"},
					},
				},
			},
		},
		// Walks all repository packages in Full mode (skipping root files and node_modules) and clears ChangedLines.
		{
			name: "full mode discovers packages and clears changed lines",
			setup: func(t *testing.T) string {
				repo := t.TempDir()
				writeRepoFile(t, repo, "README.md", "# Root\n")
				writeRepoFile(t, repo, "node_modules/ignored/bad.go", "package bad\n")
				writeRepoFile(t, repo, "pkg/one/one.go", "package one; type One struct{}\n")
				writeRepoFile(t, repo, "pkg/two/two.go", "package two; type Two struct{}\n")
				return repo
			},
			rule: RuleSpec{
				ID:          "pkg_full",
				Name:        "Pkg Full",
				Enforcement: EnforcementAdvisory,
				Target:      TargetSpec{Kind: TargetGoLogicalPackage},
			},
			opts: func(repo string) ResolveOptions {
				return ResolveOptions{
					RepoRoot:       repo,
					Full:           true,
					ChangedGoLines: map[string]map[int]struct{}{"pkg/one/one.go": {1: {}}},
				}
			},
			wantBundles: []TargetBundle{
				{
					ID:          "pkg/one",
					PrimaryFile: "pkg/one/one.go",
					Files:       []BundledFile{{RelPath: "pkg/one/one.go", Content: []byte("package one; type One struct{}\n")}},
					ASTSummary: `- pkg/one/one.go (package one)
  type One (pkg/one/one.go:1)
`,
					SubstantiveGoUnits: []ExtractedItem{
						{Target: "pkg/one", File: "pkg/one/one.go", Line: 1, StartLine: 1, EndLine: 1, Name: "one.go (One)", Kind: "Go source file"},
					},
				},
				{
					ID:          "pkg/two",
					PrimaryFile: "pkg/two/two.go",
					Files:       []BundledFile{{RelPath: "pkg/two/two.go", Content: []byte("package two; type Two struct{}\n")}},
					ASTSummary: `- pkg/two/two.go (package two)
  type Two (pkg/two/two.go:1)
`,
					SubstantiveGoUnits: []ExtractedItem{
						{Target: "pkg/two", File: "pkg/two/two.go", Line: 1, StartLine: 1, EndLine: 1, Name: "two.go (Two)", Kind: "Go source file"},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := tc.setup(t)

			gotBundles, err := ResolveTargets(tc.rule, tc.opts(repo))

			if err != nil {
				t.Fatalf("ResolveTargets() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantBundles, gotBundles); diff != "" {
				t.Errorf("ResolveTargets() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveChangedFileTargets(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(t *testing.T) string
		rule        RuleSpec
		opts        func(repo string) ResolveOptions
		wantBundles []TargetBundle
	}{
		// Bundles companion .go when present, skips missing companion/non-matching/missing files, excludes zzz_ siblings, and keeps ChangedLines.
		{
			name: "companion go, sibling struct summary, and filtered/missing changed files",
			setup: func(t *testing.T) string {
				repo := t.TempDir()
				writeRepoFile(t, repo, "pkg/store/types.go", "package store; type Store struct { present []bool }\n")
				writeRepoFile(t, repo, "pkg/store/zzz_gen.go", "package store; type Generated struct{}\n")
				writeRepoFile(t, repo, "pkg/store/query.go", "package store; func (s *Store) Count() int { return len(s.present) }\n")
				writeRepoFile(t, repo, "pkg/store/query_test.go", "package store; import \"testing\"; func TestCount(t *testing.T) {}\n")
				writeRepoFile(t, repo, "pkg/store/standalone_test.go", "package store; import \"testing\"; func TestAlone(t *testing.T) {}\n")
				writeRepoFile(t, repo, "pkg/store/README.md", "# Store\n")
				writeRepoFile(t, repo, "pkg/store/untouched_test.go", "package store\n")
				return repo
			},
			rule: RuleSpec{
				ID:          "go_test_quality",
				Name:        "Go Test Quality",
				Enforcement: EnforcementAdvisory,
				Target: TargetSpec{
					Kind:    TargetChangedFile,
					Include: []string{"pkg/**/*_test.go"},
				},
				Context: ContextSpec{
					IncludeCompanionGo:  true,
					LineNumbers:         true,
					IncludeGoASTSummary: true,
				},
			},
			opts: func(repo string) ResolveOptions {
				return ResolveOptions{
					RepoRoot:       repo,
					ChangedFiles:   []string{"pkg/store/standalone_test.go", "pkg/store/query_test.go", "pkg/store/README.md", "pkg/store/deleted_test.go"},
					ChangedGoLines: map[string]map[int]struct{}{"pkg/store/query_test.go": {1: {}}},
				}
			},
			wantBundles: []TargetBundle{
				{
					ID:          "pkg/store/query_test.go",
					PrimaryFile: "pkg/store/query_test.go",
					Files: []BundledFile{
						{RelPath: "pkg/store/query.go", Content: []byte("package store; func (s *Store) Count() int { return len(s.present) }\n")},
						{RelPath: "pkg/store/query_test.go", Content: []byte("package store; import \"testing\"; func TestCount(t *testing.T) {}\n")},
					},
					ASTSummary: `- pkg/store/query.go (package store)
  func Store.Count (pkg/store/query.go:1)
- pkg/store/types.go (package store)
  type Store (pkg/store/types.go:1)
    field present []bool
`,
					ChangedLines: map[string]map[int]struct{}{"pkg/store/query_test.go": {1: {}}},
				},
				{
					ID:          "pkg/store/standalone_test.go",
					PrimaryFile: "pkg/store/standalone_test.go",
					Files: []BundledFile{
						{RelPath: "pkg/store/standalone_test.go", Content: []byte("package store; import \"testing\"; func TestAlone(t *testing.T) {}\n")},
					},
					ASTSummary: `- pkg/store/query.go (package store)
  func Store.Count (pkg/store/query.go:1)
- pkg/store/types.go (package store)
  type Store (pkg/store/types.go:1)
    field present []bool
`,
					ChangedLines: map[string]map[int]struct{}{"pkg/store/query_test.go": {1: {}}},
				},
			},
		},
		// Omits companion .go and sibling AST summaries when IncludeCompanionGo and IncludeGoASTSummary are false.
		{
			name: "disabled companion go and disabled AST summary",
			setup: func(t *testing.T) string {
				repo := t.TempDir()
				writeRepoFile(t, repo, "pkg/store/types.go", "package store; type Store struct { present []bool }\n")
				writeRepoFile(t, repo, "pkg/store/query.go", "package store; func New() *Store { return nil }\n")
				writeRepoFile(t, repo, "pkg/store/query_test.go", "package store; import \"testing\"; func TestNew(t *testing.T) {}\n")
				return repo
			},
			rule: RuleSpec{
				ID:          "plain_file",
				Name:        "Plain File",
				Enforcement: EnforcementAdvisory,
				Target: TargetSpec{
					Kind:    TargetChangedFile,
					Include: []string{"pkg/**"},
				},
			},
			opts: func(repo string) ResolveOptions {
				return ResolveOptions{
					RepoRoot:     repo,
					ChangedFiles: []string{"pkg/store/query.go", "pkg/store/query_test.go"},
				}
			},
			wantBundles: []TargetBundle{
				{
					ID:          "pkg/store/query.go",
					PrimaryFile: "pkg/store/query.go",
					Files:       []BundledFile{{RelPath: "pkg/store/query.go", Content: []byte("package store; func New() *Store { return nil }\n")}},
					ASTSummary: `- pkg/store/query.go (package store)
  func New (pkg/store/query.go:1)
`,
					SubstantiveGoUnits: []ExtractedItem{
						{Target: "pkg/store/query.go", File: "pkg/store/query.go", Line: 1, StartLine: 1, EndLine: 1, Name: "query.go (New)", Kind: "Go source file"},
					},
				},
				{
					ID:          "pkg/store/query_test.go",
					PrimaryFile: "pkg/store/query_test.go",
					Files:       []BundledFile{{RelPath: "pkg/store/query_test.go", Content: []byte("package store; import \"testing\"; func TestNew(t *testing.T) {}\n")}},
				},
			},
		},
		// Walks repository files in Full mode, handles root-level Go siblings, skips non-Go and empty-sibling dirs, caps at MaxTargets, and clears ChangedLines.
		{
			name: "full mode walks root files, stops at max_targets, and clears changed lines",
			setup: func(t *testing.T) string {
				repo := t.TempDir()
				writeRepoFile(t, repo, "main.go", "package main; type App struct{}\n")
				writeRepoFile(t, repo, "main_test.go", "package main; import \"testing\"; func TestApp(t *testing.T) {}\n")
				writeRepoFile(t, repo, "notes.md", "# Root Notes\n")
				writeRepoFile(t, repo, "pkg/onlytests/only_test.go", "package onlytests; import \"testing\"; func TestOnly(t *testing.T) {}\n")
				writeRepoFile(t, repo, "pkg/z/z.go", "package z; type Z struct{}\n")
				writeRepoFile(t, repo, "node_modules/dep.go", "package dep\n")
				return repo
			},
			rule: RuleSpec{
				ID:          "all_files",
				Name:        "All Files",
				Enforcement: EnforcementAdvisory,
				Target:      TargetSpec{Kind: TargetChangedFile},
				Context:     ContextSpec{IncludeGoASTSummary: true},
			},
			opts: func(repo string) ResolveOptions {
				return ResolveOptions{
					RepoRoot:       repo,
					Full:           true,
					MaxTargets:     4,
					ChangedGoLines: map[string]map[int]struct{}{"main.go": {1: {}}},
				}
			},
			wantBundles: []TargetBundle{
				{
					ID:          "main.go",
					PrimaryFile: "main.go",
					Files:       []BundledFile{{RelPath: "main.go", Content: []byte("package main; type App struct{}\n")}},
					ASTSummary: `- main.go (package main)
  type App (main.go:1)
`,
					SubstantiveGoUnits: []ExtractedItem{
						{Target: "main.go", File: "main.go", Line: 1, StartLine: 1, EndLine: 1, Name: "main.go (App)", Kind: "Go source file"},
					},
				},
				{
					ID:          "main_test.go",
					PrimaryFile: "main_test.go",
					Files:       []BundledFile{{RelPath: "main_test.go", Content: []byte("package main; import \"testing\"; func TestApp(t *testing.T) {}\n")}},
					ASTSummary: `- main.go (package main)
  type App (main.go:1)
`,
				},
				{
					ID:          "notes.md",
					PrimaryFile: "notes.md",
					Files:       []BundledFile{{RelPath: "notes.md", Content: []byte("# Root Notes\n")}},
				},
				{
					ID:          "pkg/onlytests/only_test.go",
					PrimaryFile: "pkg/onlytests/only_test.go",
					Files:       []BundledFile{{RelPath: "pkg/onlytests/only_test.go", Content: []byte("package onlytests; import \"testing\"; func TestOnly(t *testing.T) {}\n")}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := tc.setup(t)

			gotBundles, err := ResolveTargets(tc.rule, tc.opts(repo))

			if err != nil {
				t.Fatalf("ResolveTargets() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantBundles, gotBundles); diff != "" {
				t.Errorf("ResolveTargets() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestReadSiblingProdGoFiles verifies that readSiblingProdGoFiles collects sorted production Go files while excluding directories, non-Go files, tests, and zzz_ files.
func TestReadSiblingProdGoFiles(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "pkg/sample/b.go", "package sample; var B = 2\n")
	writeRepoFile(t, repo, "pkg/sample/a.go", "package sample; var A = 1\n")
	writeRepoFile(t, repo, "pkg/sample/a_test.go", "package sample\n")
	writeRepoFile(t, repo, "pkg/sample/zzz_generated.go", "package sample\n")
	writeRepoFile(t, repo, "pkg/sample/README.md", "# Sample\n")
	if err := os.MkdirAll(filepath.Join(repo, "pkg", "sample", "sub.go"), 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	gotFiles, err := readSiblingProdGoFiles(repo, "pkg/sample")

	if err != nil {
		t.Fatalf("readSiblingProdGoFiles() unexpected error: %v", err)
	}
	wantFiles := []BundledFile{
		{RelPath: "pkg/sample/a.go", Content: []byte("package sample; var A = 1\n")},
		{RelPath: "pkg/sample/b.go", Content: []byte("package sample; var B = 2\n")},
	}
	if diff := cmp.Diff(wantFiles, gotFiles); diff != "" {
		t.Errorf("readSiblingProdGoFiles() mismatch (-want +got):\n%s", diff)
	}
}

// TestIsEligiblePackageDirRegularFile verifies that isEligiblePackageDir rejects a regular file path when no RequireFiles are configured.
func TestIsEligiblePackageDirRegularFile(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "pkg/asfile", "regular file blocking dir\n")
	target := TargetSpec{
		Kind:    TargetGoLogicalPackage,
		Include: []string{"pkg/**"},
	}

	got := isEligiblePackageDir(repo, "pkg/asfile", target)

	if got {
		t.Errorf("isEligiblePackageDir(%q) = %v, want false", "pkg/asfile", got)
	}
}

func TestRenderSharedInput(t *testing.T) {
	testCases := []struct {
		name          string
		setup         func(t *testing.T) (ResolveOptions, *[][]string)
		rule          RuleSpec
		bundle        TargetBundle
		wantDiffCalls int
		wantDiffArgs  [][]string
		wantRendered  string
	}{
		// Loads reference_files strictly from TrustedRoot, includes AST summary and diff, and renders line numbers for files with and without trailing newlines.
		{
			name: "trusted root, AST summary, diff, and numbered lines",
			setup: func(t *testing.T) (ResolveOptions, *[][]string) {
				repo := t.TempDir()
				trusted := t.TempDir()
				writeRepoFile(t, trusted, ".agents/rules/sample.md", "# Trusted Standard\n")
				writeRepoFile(t, repo, ".agents/rules/sample.md", "# Tampered PR Standard\n")
				var diffCalls [][]string
				opts := ResolveOptions{
					RepoRoot:    repo,
					TrustedRoot: trusted,
					ReadDiff: func(paths []string) (string, error) {
						diffCalls = append(diffCalls, append([]string(nil), paths...))
						return "+added line", nil
					},
				}
				return opts, &diffCalls
			},
			rule: RuleSpec{
				ID:          "pkg_doc",
				Name:        "Pkg Doc",
				Enforcement: EnforcementAdvisory,
				Context: ContextSpec{
					ReferenceFiles:      []string{".agents/rules/sample.md"},
					IncludeDiff:         true,
					LineNumbers:         true,
					IncludeGoASTSummary: true,
				},
			},
			bundle: TargetBundle{
				ID:          "pkg/task/feat",
				PrimaryFile: "pkg/task/feat/README.md",
				ASTSummary:  "- pkg/task/feat/taskid.go (package feat)\n",
				Files: []BundledFile{
					{RelPath: "pkg/task/feat/README.md", Content: []byte("# Feat\r\nOverview")},
					{RelPath: "pkg/task/feat/taskid.go", Content: []byte("package feat\n")},
				},
			},
			wantDiffCalls: 1,
			wantDiffArgs:  [][]string{{"pkg/task/feat/README.md", "pkg/task/feat/taskid.go"}},
			wantRendered: `=== REFERENCE DOCUMENT: .agents/rules/sample.md ===
   1 | # Trusted Standard

=== TARGET: pkg/task/feat ===

=== GO AST STRUCTURAL INDEX ===
- pkg/task/feat/taskid.go (package feat)

=== VCS UNIFIED DIFF ===
+added line

=== FILE: pkg/task/feat/README.md ===
   1 | # Feat
   2 | Overview

=== FILE: pkg/task/feat/taskid.go ===
   1 | package feat
`,
		},
		// Defaults TrustedRoot to RepoRoot, omits AST summary and ReadDiff callback when disabled, and appends a missing trailing newline when LineNumbers is false.
		{
			name: "repo root fallback, disabled diff and AST summary, unnumbered body without trailing newline",
			setup: func(t *testing.T) (ResolveOptions, *[][]string) {
				repo := t.TempDir()
				writeRepoFile(t, repo, "docs/ref.md", "Local Ref\n")
				var diffCalls [][]string
				opts := ResolveOptions{
					RepoRoot: repo,
					ReadDiff: func(paths []string) (string, error) {
						diffCalls = append(diffCalls, append([]string(nil), paths...))
						return "+should not appear", nil
					},
				}
				return opts, &diffCalls
			},
			rule: RuleSpec{
				ID:          "raw_ctx",
				Name:        "Raw Context",
				Enforcement: EnforcementAdvisory,
				Context: ContextSpec{
					ReferenceFiles: []string{"docs/ref.md"},
				},
			},
			bundle: TargetBundle{
				ID:          "pkg/a/a.go",
				PrimaryFile: "pkg/a/a.go",
				ASTSummary:  "- ignored\n",
				Files: []BundledFile{
					{RelPath: "pkg/a/a.go", Content: []byte("package a\r\nvar x = 1")},
				},
			},
			wantDiffCalls: 0,
			wantRendered: `=== REFERENCE DOCUMENT: docs/ref.md ===
Local Ref

=== TARGET: pkg/a/a.go ===

=== FILE: pkg/a/a.go ===
package a
var x = 1
`,
		},
		// Omits AST index when ASTSummary is empty, ignores nil ReadDiff even when IncludeDiff is true, and avoids adding a duplicate trailing newline when LineNumbers is false.
		{
			name: "nil ReadDiff, empty AST summary, and unnumbered body preserving trailing newline",
			setup: func(t *testing.T) (ResolveOptions, *[][]string) {
				repo := t.TempDir()
				var diffCalls [][]string
				return ResolveOptions{RepoRoot: repo}, &diffCalls
			},
			rule: RuleSpec{
				ID:          "raw_nl",
				Name:        "Raw Newline",
				Enforcement: EnforcementAdvisory,
				Context: ContextSpec{
					IncludeDiff:         true,
					IncludeGoASTSummary: true,
				},
			},
			bundle: TargetBundle{
				ID:          "pkg/a/a.go",
				PrimaryFile: "pkg/a/a.go",
				Files:       []BundledFile{{RelPath: "pkg/a/a.go", Content: []byte("package a\n")}},
			},
			wantDiffCalls: 0,
			wantRendered: `=== TARGET: pkg/a/a.go ===

=== FILE: pkg/a/a.go ===
package a
`,
		},
		// Invokes ReadDiff when IncludeDiff is true and omits the diff section header when ReadDiff returns only whitespace.
		{
			name: "whitespace-only diff omits diff header",
			setup: func(t *testing.T) (ResolveOptions, *[][]string) {
				repo := t.TempDir()
				var diffCalls [][]string
				opts := ResolveOptions{
					RepoRoot: repo,
					ReadDiff: func(paths []string) (string, error) {
						diffCalls = append(diffCalls, append([]string(nil), paths...))
						return "  \n\t ", nil
					},
				}
				return opts, &diffCalls
			},
			rule: RuleSpec{
				ID:          "blank_diff",
				Name:        "Blank Diff",
				Enforcement: EnforcementAdvisory,
				Context: ContextSpec{
					IncludeDiff: true,
					LineNumbers: true,
				},
			},
			bundle: TargetBundle{
				ID:          "pkg/a/a.go",
				PrimaryFile: "pkg/a/a.go",
				Files:       []BundledFile{{RelPath: "pkg/a/a.go", Content: []byte("package a\n")}},
			},
			wantDiffCalls: 1,
			wantDiffArgs:  [][]string{{"pkg/a/a.go"}},
			wantRendered: `=== TARGET: pkg/a/a.go ===

=== FILE: pkg/a/a.go ===
   1 | package a
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts, gotDiffCalls := tc.setup(t)

			gotRendered, err := RenderSharedInput(tc.rule, tc.bundle, opts)

			if err != nil {
				t.Fatalf("RenderSharedInput() unexpected error: %v", err)
			}
			if len(*gotDiffCalls) != tc.wantDiffCalls {
				t.Errorf("ReadDiff call count = %d, want %d", len(*gotDiffCalls), tc.wantDiffCalls)
			}
			if diff := cmp.Diff(tc.wantDiffArgs, *gotDiffCalls); diff != "" {
				t.Errorf("ReadDiff args mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantRendered, gotRendered); diff != "" {
				t.Errorf("RenderSharedInput() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func writeRepoFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
}
