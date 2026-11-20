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

package qarunner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// CommandResult holds the captured output, exit status, and launch error of an external command.
type CommandResult struct {
	// Stdout is the captured standard output of the command.
	Stdout string
	// Stderr is the captured standard error of the command.
	Stderr string
	// ExitCode is the process exit code (0 on success).
	ExitCode int
	// Err is a non-nil error when the command could not be started or waited on.
	Err error
}

// CommandRunner executes an external command in dir with optional extra environment variables.
type CommandRunner func(ctx context.Context, dir string, env []string, name string, args ...string) CommandResult

type domainScope struct {
	licenseFiles      []string
	goFiles           []string
	goDirs            []string
	goAllPkgs         bool
	hasGo             bool
	targetGoFiles     map[string]struct{}
	hasWeb            bool
	hasWebScripts     bool
	prettierFiles     []string
	miscPrettierFiles []string
	tsFiles           []string
	scssFiles         []string
	hasProto          bool
	mdFiles           []string
}

// ExecCommandRunner runs external commands using os/exec.CommandContext.
func ExecCommandRunner(ctx context.Context, dir string, env []string, name string, args ...string) CommandResult {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return CommandResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: 0}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return CommandResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitErr.ExitCode()}
	}
	return CommandResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: -1, Err: err}
}

func buildDomainScope(repoRoot string, diff gotestutil.DiffResult, lang string) domainScope {
	scope := domainScope{targetGoFiles: make(map[string]struct{})}
	goDirSet := make(map[string]struct{})
	allowGo := lang == "" || lang == "all" || lang == "go"
	allowWeb := lang == "" || lang == "all" || lang == "ts"
	allowMisc := lang == "" || lang == "all"

	for _, rawPath := range diff.ChangedFiles {
		clean := normalizeSlashPath(rawPath)
		if clean == "" {
			continue
		}
		if allowGo {
			scope.collectGoFile(repoRoot, clean, goDirSet)
		}
		if allowWeb {
			scope.collectWebFile(repoRoot, clean)
		}
		if allowMisc {
			scope.collectMiscFile(repoRoot, clean)
		}
		if fileExists(repoRoot, clean) && isLicenseAllowedForLang(clean, lang) && isLicenseCandidate(clean) {
			scope.licenseFiles = append(scope.licenseFiles, clean)
		}
	}

	scope.finalize(goDirSet)
	return scope
}

func (s *domainScope) collectGoFile(repoRoot string, clean string, goDirSet map[string]struct{}) {
	if clean == "go.mod" || clean == "go.sum" || clean == ".golangci.yaml" || strings.HasPrefix(clean, "scripts/backend-codegen/") {
		s.hasGo = true
		s.goAllPkgs = true
		return
	}
	if !isTargetGoSource(clean) {
		return
	}
	s.hasGo = true
	if fileExists(repoRoot, clean) {
		s.goFiles = append(s.goFiles, clean)
		s.targetGoFiles[clean] = struct{}{}
	}
	relDir := path.Dir(clean)
	if dirHasGoFiles(repoRoot, relDir) {
		if relDir == "." {
			goDirSet["."] = struct{}{}
		} else {
			goDirSet["./"+relDir] = struct{}{}
		}
	} else {
		s.goAllPkgs = true
	}
}

func (s *domainScope) collectWebFile(repoRoot string, clean string) {
	if isGlobalWebTrigger(clean) {
		s.hasWeb = true
		return
	}
	if !strings.HasPrefix(clean, "web/") {
		return
	}
	s.hasWeb = true
	if strings.HasPrefix(clean, "web/scripts/") {
		s.hasWebScripts = true
	}
	if !fileExists(repoRoot, clean) {
		return
	}
	webRel := strings.TrimPrefix(clean, "web/")
	switch {
	case strings.HasSuffix(clean, ".ts"):
		s.tsFiles = append(s.tsFiles, webRel)
		s.prettierFiles = append(s.prettierFiles, webRel)
	case strings.HasSuffix(clean, ".scss"):
		s.scssFiles = append(s.scssFiles, webRel)
		s.prettierFiles = append(s.prettierFiles, webRel)
	case strings.HasSuffix(clean, ".html"), strings.HasSuffix(clean, ".json"), strings.HasSuffix(clean, ".mjs"):
		s.prettierFiles = append(s.prettierFiles, webRel)
	}
}

func isGlobalWebTrigger(clean string) bool {
	return clean == ".node-version" ||
		strings.HasPrefix(clean, "scripts/frontend-codegen/") ||
		strings.HasPrefix(clean, "scripts/fontlist-gen/") ||
		strings.HasPrefix(clean, "scripts/msdf-generator/") ||
		clean == "scripts/generate-angular-json.sh" ||
		clean == "scripts/generate-version.sh"
}

func (s *domainScope) collectMiscFile(repoRoot string, clean string) {
	if strings.HasPrefix(clean, "proto/") || clean == "buf.yaml" || clean == "buf.gen.yaml" {
		s.hasProto = true
	}
	if fileExists(repoRoot, clean) && strings.HasSuffix(clean, ".md") {
		s.mdFiles = append(s.mdFiles, clean)
	}
	if fileExists(repoRoot, clean) && isMiscPrettierFile(clean) {
		s.miscPrettierFiles = append(s.miscPrettierFiles, clean)
	}
}

func isMiscPrettierFile(clean string) bool {
	inMiscDir := strings.HasPrefix(clean, ".cloudbuild/") ||
		strings.HasPrefix(clean, ".github/") ||
		strings.HasPrefix(clean, ".vscode/") ||
		strings.HasPrefix(clean, "scripts/codequality/rules/")
	if !inMiscDir {
		return false
	}
	return strings.HasSuffix(clean, ".yaml") ||
		strings.HasSuffix(clean, ".yml") ||
		strings.HasSuffix(clean, ".json")
}

func (s *domainScope) finalize(goDirSet map[string]struct{}) {
	s.goDirs = make([]string, 0, len(goDirSet))
	for dir := range goDirSet {
		s.goDirs = append(s.goDirs, dir)
	}
	slices.Sort(s.goDirs)
	slices.Sort(s.goFiles)
	slices.Sort(s.licenseFiles)
	slices.Sort(s.prettierFiles)
	slices.Sort(s.miscPrettierFiles)
	slices.Sort(s.tsFiles)
	slices.Sort(s.scssFiles)
	slices.Sort(s.mdFiles)
}

func (s *domainScope) goPkgArgs() []string {
	if s.goAllPkgs || len(s.goDirs) == 0 {
		return []string{"./..."}
	}
	return s.goDirs
}

func isLicenseAllowedForLang(clean string, lang string) bool {
	switch lang {
	case "go":
		return !strings.HasPrefix(clean, "web/") && !strings.HasSuffix(clean, ".md") && !strings.HasPrefix(clean, "proto/")
	case "ts":
		return strings.HasPrefix(clean, "web/")
	default:
		return true
	}
}

func isLicenseCandidate(clean string) bool {
	excludedSuffixes := []string{
		".md", ".json", ".png", ".jpg", ".jpeg", ".gif", ".svg",
		".woff", ".woff2", ".ttf", ".ico", ".khi", ".txt",
	}
	for _, suffix := range excludedSuffixes {
		if strings.HasSuffix(clean, suffix) {
			return false
		}
	}
	if strings.Contains(clean, ".golden") {
		return false
	}
	base := path.Base(clean)
	switch base {
	case "LICENSE", "VERSION", ".gitignore", ".prettierignore", ".dockerignore", ".gitattributes", ".node-version", ".root":
		return false
	default:
		return true
	}
}

func isTargetGoSource(clean string) bool {
	if !strings.HasSuffix(clean, ".go") {
		return false
	}
	if strings.HasPrefix(clean, "web/") || strings.HasPrefix(clean, "pkg/generated/") {
		return false
	}
	return !strings.HasPrefix(path.Base(clean), "zzz_")
}

func fileExists(repoRoot string, clean string) bool {
	info, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(clean)))
	return err == nil && !info.IsDir()
}

func dirHasGoFiles(repoRoot string, relDir string) bool {
	entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(relDir)))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

func isSourceSkipped(skipSources map[string]struct{}, source string, rep *rule.QAReport) bool {
	if _, ok := skipSources[source]; !ok {
		return false
	}
	rep.SkippedSources = append(rep.SkippedSources, rule.SkippedSource{
		Source: source,
		Reason: "skipped by configuration",
	})
	return true
}

func combineCommandOutput(res CommandResult) string {
	out := strings.TrimSpace(res.Stdout)
	errOut := strings.TrimSpace(res.Stderr)
	if out != "" && errOut != "" {
		return out + "\n" + errOut
	}
	if out != "" {
		return out
	}
	return errOut
}
