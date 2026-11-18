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
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// TargetBundle holds all files, diff metadata, and extracted AST index for one evaluation unit.
type TargetBundle struct {
	// ID uniquely names the target within a run (logical package directory or relative file path).
	ID string
	// PrimaryFile is the default repository-relative file path attributed to bundle-level findings.
	PrimaryFile string
	// WebDir locates the frontend web/ directory used to resolve TypeScript AST dependencies.
	WebDir string
	// Files holds bundled files in deterministic order.
	Files []BundledFile
	// ASTSummary holds the optional precomputed Go AST index when Go files are present.
	ASTSummary string
	// SubstantiveGoUnits holds precomputed substantive Go units when Go files are present.
	SubstantiveGoUnits []ExtractedItem
	// ChangedLines maps repository-relative Go and TypeScript file paths to changed 1-based line sets.
	ChangedLines map[string]map[int]struct{}
}

// ResolveOptions configures target discovery and shared-context rendering for a RuleSpec.
type ResolveOptions struct {
	// RepoRoot is the target repository directory being inspected.
	RepoRoot string
	// TrustedRoot is the trusted repository root used to load reference_files (defaults to RepoRoot).
	TrustedRoot string
	// Full evaluates all matching repository targets instead of only VCS-changed targets.
	Full bool
	// MaxTargets caps the number of resolved targets per rule when > 0.
	MaxTargets int
	// ChangedFiles lists added or modified repository-relative paths from VCS diff.
	ChangedFiles []string
	// DeletedFiles lists deleted repository-relative paths from VCS diff.
	DeletedFiles []string
	// ChangedLines maps modified Go and TypeScript files to changed 1-based line sets (`nil` in Full mode).
	ChangedLines map[string]map[int]struct{}
	// ReadDiff optionally returns unified diff text for a slice of repository-relative paths.
	ReadDiff func(relPaths []string) (string, error)
}

// ResolveTargets discovers deterministic TargetBundles matching rule and opts.
func ResolveTargets(rule RuleSpec, opts ResolveOptions) ([]TargetBundle, error) {
	switch rule.Target.Kind {
	case TargetGoLogicalPackage:
		return resolveLogicalPackageTargets(rule, opts)
	case TargetChangedFile:
		return resolveChangedFileTargets(rule, opts)
	default:
		return nil, fmt.Errorf("unsupported target kind %q", rule.Target.Kind)
	}
}

// RenderSharedInput assembles the shared text context sent to the Gemini Decisions API for bundle.
func RenderSharedInput(rule RuleSpec, bundle TargetBundle, opts ResolveOptions) (string, error) {
	var builder strings.Builder
	if err := appendReferenceDocs(&builder, resolveEffectiveTrustedRoot(opts), rule.Context); err != nil {
		return "", err
	}
	fmt.Fprintf(&builder, "=== TARGET: %s ===\n", bundle.ID)
	if rule.Context.IncludeGoASTSummary && bundle.ASTSummary != "" {
		fmt.Fprintf(&builder, "\n=== GO AST STRUCTURAL INDEX ===\n%s", bundle.ASTSummary)
	}
	if rule.Context.IncludeDiff {
		if err := appendUnifiedDiff(&builder, bundle, opts.ReadDiff); err != nil {
			return "", err
		}
	}
	for _, file := range bundle.Files {
		fmt.Fprintf(&builder, "\n=== FILE: %s ===\n", file.RelPath)
		writeFileBody(&builder, string(file.Content), rule.Context)
	}
	return builder.String(), nil
}

// LogicalGoPackageDir maps a repository-relative file path to its logical Go package root directory.
func LogicalGoPackageDir(relPath string) string {
	clean := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	dir := path.Dir(clean)
	if dir == "." || dir == "/" {
		return ""
	}
	if idx := strings.Index(dir, "/internal/"); idx > 0 {
		return dir[:idx]
	}
	if strings.HasSuffix(dir, "/internal") {
		return strings.TrimSuffix(dir, "/internal")
	}
	return strings.TrimSuffix(dir, "/impl")
}

func resolveLogicalPackageTargets(rule RuleSpec, opts ResolveOptions) ([]TargetBundle, error) {
	candidates, err := candidatePackageDirs(opts)
	if err != nil {
		return nil, err
	}
	var bundles []TargetBundle
	for _, pkgDir := range candidates {
		if !isEligiblePackageDir(opts.RepoRoot, pkgDir, rule.Target) {
			continue
		}
		bundle, ok, buildErr := buildPackageBundle(opts.RepoRoot, pkgDir, rule, opts)
		if buildErr != nil {
			return nil, buildErr
		}
		if !ok {
			continue
		}
		bundles = append(bundles, bundle)
		if opts.MaxTargets > 0 && len(bundles) >= opts.MaxTargets {
			break
		}
	}
	return bundles, nil
}

func candidatePackageDirs(opts ResolveOptions) ([]string, error) {
	seen := make(map[string]struct{})
	if opts.Full {
		err := walkTreeFiles(opts.RepoRoot, func(relPath string) {
			if pkgDir := LogicalGoPackageDir(relPath); pkgDir != "" {
				seen[pkgDir] = struct{}{}
			}
		})
		if err != nil {
			return nil, err
		}
	} else {
		for _, relPath := range slices.Concat(opts.ChangedFiles, opts.DeletedFiles) {
			if !MatchesFilter(relPath, nil, []string{".git/**", ".jj/**"}) {
				continue
			}
			if pkgDir := LogicalGoPackageDir(relPath); pkgDir != "" {
				seen[pkgDir] = struct{}{}
			}
		}
	}
	return slices.Sorted(maps.Keys(seen)), nil
}

func isEligiblePackageDir(repoRoot string, pkgDir string, target TargetSpec) bool {
	if !MatchesFilter(pkgDir, target.Include, target.Exclude) {
		return false
	}
	absPkg := filepath.Join(repoRoot, filepath.FromSlash(pkgDir))
	info, err := os.Stat(absPkg)
	if err != nil || !info.IsDir() {
		return false
	}
	for _, req := range target.RequireFiles {
		reqInfo, reqErr := os.Stat(filepath.Join(absPkg, filepath.FromSlash(req)))
		if reqErr != nil || reqInfo.IsDir() {
			return false
		}
	}
	return true
}

func buildPackageBundle(repoRoot string, pkgDir string, rule RuleSpec, opts ResolveOptions) (TargetBundle, bool, error) {
	inc := rule.Context.IncludeFiles
	if len(inc) == 0 {
		inc = []string{"README.md", "*.go", "impl/*.go", "internal/**/*.go"}
	}
	files, err := collectPackageFiles(repoRoot, pkgDir, inc, rule.Context.ExcludeFiles, rule.Target.Exclude)
	if err != nil {
		return TargetBundle{}, false, err
	}
	if len(files) == 0 {
		return TargetBundle{}, false, nil
	}
	units, astSummary, err := ExtractGoSubstantiveUnits(pkgDir, files)
	if err != nil {
		return TargetBundle{}, false, err
	}
	primary := files[0].RelPath
	if len(rule.Target.RequireFiles) > 0 {
		primary = path.Join(pkgDir, rule.Target.RequireFiles[0])
	}
	var changedMap map[string]map[int]struct{}
	if !opts.Full {
		changedMap = opts.ChangedLines
	}
	return TargetBundle{
		ID:                 pkgDir,
		PrimaryFile:        primary,
		WebDir:             resolveBundleWebDir(opts),
		Files:              files,
		ASTSummary:         astSummary,
		SubstantiveGoUnits: units,
		ChangedLines:       changedMap,
	}, true, nil
}

func collectPackageFiles(repoRoot string, pkgDir string, includeFiles []string, excludeFiles []string, targetExclude []string) ([]BundledFile, error) {
	absPkg := filepath.Join(repoRoot, filepath.FromSlash(pkgDir))
	var relMatched []string
	err := walkTreeFiles(absPkg, func(relToPkg string) {
		relToRepo := path.Join(pkgDir, relToPkg)
		if MatchesFilter(relToPkg, includeFiles, excludeFiles) && MatchesFilter(relToRepo, nil, targetExclude) {
			relMatched = append(relMatched, relToRepo)
		}
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(relMatched)
	return readBundledFiles(repoRoot, relMatched)
}

func resolveChangedFileTargets(rule RuleSpec, opts ResolveOptions) ([]TargetBundle, error) {
	candidates, err := candidateRepoFiles(opts)
	if err != nil {
		return nil, err
	}
	var bundles []TargetBundle
	for _, relPath := range candidates {
		bundle, ok, buildErr := buildSingleChangedFileBundle(relPath, rule, opts)
		if buildErr != nil {
			return nil, buildErr
		}
		if !ok {
			continue
		}
		bundles = append(bundles, bundle)
		if opts.MaxTargets > 0 && len(bundles) >= opts.MaxTargets {
			break
		}
	}
	return bundles, nil
}

func buildSingleChangedFileBundle(relPath string, rule RuleSpec, opts ResolveOptions) (TargetBundle, bool, error) {
	if !MatchesFilter(relPath, rule.Target.Include, rule.Target.Exclude) {
		return TargetBundle{}, false, nil
	}
	if _, statErr := os.Stat(filepath.Join(opts.RepoRoot, filepath.FromSlash(relPath))); statErr != nil {
		if os.IsNotExist(statErr) {
			return TargetBundle{}, false, nil
		}
		return TargetBundle{}, false, statErr
	}
	bundledFiles, err := buildChangedFileSlice(opts.RepoRoot, relPath, rule.Context)
	if err != nil {
		return TargetBundle{}, false, err
	}
	units, astSummary, err := buildChangedFileASTMetadata(opts.RepoRoot, relPath, bundledFiles, rule.Context)
	if err != nil {
		return TargetBundle{}, false, err
	}
	var changedMap map[string]map[int]struct{}
	if !opts.Full {
		changedMap = opts.ChangedLines
	}
	return TargetBundle{
		ID:                 relPath,
		PrimaryFile:        relPath,
		WebDir:             resolveBundleWebDir(opts),
		Files:              bundledFiles,
		ASTSummary:         astSummary,
		SubstantiveGoUnits: units,
		ChangedLines:       changedMap,
	}, true, nil
}

func candidateRepoFiles(opts ResolveOptions) ([]string, error) {
	if !opts.Full {
		seen := make(map[string]struct{}, len(opts.ChangedFiles))
		for _, relPath := range opts.ChangedFiles {
			seen[relPath] = struct{}{}
		}
		return slices.Sorted(maps.Keys(seen)), nil
	}
	var all []string
	err := walkTreeFiles(opts.RepoRoot, func(relPath string) {
		all = append(all, relPath)
	})
	slices.Sort(all)
	return all, err
}

func walkTreeFiles(rootDir string, visit func(relPath string)) error {
	return filepath.WalkDir(rootDir, func(cur string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if cur != rootDir && shouldSkipWalkDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(rootDir, cur)
		if err != nil {
			return err
		}
		visit(filepath.ToSlash(rel))
		return nil
	})
}

func shouldSkipWalkDir(name string) bool {
	switch name {
	case ".git", ".jj", "node_modules", ".angular", "dist", "coverage", "vrt-output":
		return true
	default:
		return false
	}
}

func readBundledFiles(repoRoot string, relPaths []string) ([]BundledFile, error) {
	out := make([]BundledFile, 0, len(relPaths))
	for _, rel := range relPaths {
		raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("failed reading %s: %w", rel, err)
		}
		out = append(out, BundledFile{RelPath: rel, Content: raw})
	}
	return out, nil
}

func appendReferenceDocs(builder *strings.Builder, trustedRoot string, ctxSpec ContextSpec) error {
	for _, ref := range ctxSpec.ReferenceFiles {
		raw, err := os.ReadFile(filepath.Join(trustedRoot, filepath.FromSlash(ref)))
		if err != nil {
			return fmt.Errorf("failed reading reference file %s: %w", ref, err)
		}
		fmt.Fprintf(builder, "=== REFERENCE DOCUMENT: %s ===\n", ref)
		writeFileBody(builder, string(raw), ctxSpec)
		builder.WriteByte('\n')
	}
	return nil
}

func appendUnifiedDiff(builder *strings.Builder, bundle TargetBundle, readDiff func([]string) (string, error)) error {
	if readDiff == nil {
		return nil
	}
	paths := make([]string, len(bundle.Files))
	for i, file := range bundle.Files {
		paths[i] = file.RelPath
	}
	diffTxt, err := readDiff(paths)
	if err != nil {
		return err
	}
	if strings.TrimSpace(diffTxt) != "" {
		fmt.Fprintf(builder, "\n=== VCS UNIFIED DIFF ===\n%s\n", diffTxt)
	}
	return nil
}

func writeFileBody(builder *strings.Builder, content string, ctxSpec ContextSpec) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if !ctxSpec.LineNumbers {
		builder.WriteString(normalized)
		if !strings.HasSuffix(normalized, "\n") {
			builder.WriteByte('\n')
		}
		return
	}
	lines := strings.Split(normalized, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for idx, line := range lines {
		fmt.Fprintf(builder, "%4d | %s\n", idx+1, line)
	}
}

func buildChangedFileSlice(repoRoot string, relPath string, ctxSpec ContextSpec) ([]BundledFile, error) {
	paths := []string{relPath}
	if companion, ok := findCompanionFile(repoRoot, relPath, ctxSpec); ok {
		paths = []string{companion, relPath}
	}
	return readBundledFiles(repoRoot, paths)
}

func findCompanionFile(repoRoot string, relPath string, ctxSpec ContextSpec) (string, bool) {
	var companion string
	switch {
	case ctxSpec.IncludeCompanionGo && strings.HasSuffix(relPath, "_test.go"):
		companion = strings.TrimSuffix(relPath, "_test.go") + ".go"
	case ctxSpec.IncludeCompanionTS && strings.HasSuffix(relPath, ".spec.ts"):
		companion = strings.TrimSuffix(relPath, ".spec.ts") + ".ts"
	default:
		return "", false
	}
	info, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(companion)))
	if err != nil || info.IsDir() {
		return "", false
	}
	return companion, true
}

func buildChangedFileASTMetadata(repoRoot string, relPath string, bundledFiles []BundledFile, ctxSpec ContextSpec) ([]ExtractedItem, string, error) {
	units, summary, err := ExtractGoSubstantiveUnits(relPath, bundledFiles)
	if err != nil {
		return nil, "", err
	}
	if !ctxSpec.IncludeGoASTSummary || !strings.HasSuffix(relPath, ".go") {
		return units, summary, nil
	}
	siblingFiles, sibErr := readSiblingProdGoFiles(repoRoot, path.Dir(relPath))
	if sibErr != nil {
		return units, summary, nil
	}
	if len(siblingFiles) == 0 {
		return units, summary, nil
	}
	_, dirSummary, sumErr := ExtractGoSubstantiveUnits(relPath, siblingFiles)
	if sumErr != nil {
		return units, summary, nil
	}
	return units, dirSummary, nil
}

func readSiblingProdGoFiles(repoRoot string, relDir string) ([]BundledFile, error) {
	absDir := filepath.Join(repoRoot, filepath.FromSlash(relDir))
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}
	var relPaths []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || strings.HasPrefix(name, "zzz_") {
			continue
		}
		relPaths = append(relPaths, path.Join(relDir, name))
	}
	slices.Sort(relPaths)
	return readBundledFiles(repoRoot, relPaths)
}

func resolveEffectiveTrustedRoot(opts ResolveOptions) string {
	if opts.TrustedRoot != "" {
		return opts.TrustedRoot
	}
	return opts.RepoRoot
}

func resolveBundleWebDir(opts ResolveOptions) string {
	root := resolveEffectiveTrustedRoot(opts)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "web")
}
