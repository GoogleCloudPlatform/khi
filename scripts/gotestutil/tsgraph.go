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
	"context"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// DefaultBuilderImage is the container image used when USE_CONTAINER=true and BUILDER_IMAGE is unset.
const DefaultBuilderImage = "gcr.io/kubernetes-history-inspector/builder:latest"

var tsImportSpecRegex = regexp.MustCompile(`(?m)(?:\bfrom\s+|import\s*\(\s*|\bimport\s+)['"]([^'"]+)['"]`)

// TSTestPlan describes which Angular Karma spec files to execute for differential testing.
type TSTestPlan struct {
	// SkipKarma reports whether Karma execution can be skipped because no relevant specs matched.
	SkipKarma bool
	// IncludeSpecs lists web/-relative spec paths (e.g. "src/app/foo.spec.ts") to pass via --include.
	IncludeSpecs []string
}

// PlanTSTestExecution builds a minimal or full Angular Karma test execution plan from changed production TypeScript files.
func PlanTSTestExecution(repoRoot string, changedTSFiles []string, full bool) (TSTestPlan, error) {
	if full {
		return TSTestPlan{
			SkipKarma: false,
		}, nil
	}
	if len(changedTSFiles) == 0 {
		return TSTestPlan{SkipKarma: true}, nil
	}

	webSrcDir := filepath.Join(repoRoot, "web", "src")
	if info, err := os.Stat(webSrcDir); err != nil || !info.IsDir() {
		return TSTestPlan{SkipKarma: true}, nil
	}

	knownFiles, contentsByFile, err := loadWebTSFiles(repoRoot, webSrcDir)
	if err != nil {
		return TSTestPlan{}, err
	}

	reverseImporters := buildReverseImportGraph(knownFiles, contentsByFile)

	allowMainSpec := false
	queue := make([]string, 0, len(changedTSFiles))
	visited := make(map[string]struct{})
	specSet := make(map[string]struct{})

	for _, rawFile := range changedTSFiles {
		clean := strings.TrimPrefix(filepath.ToSlash(rawFile), "./")
		if !IsTargetProductionTSFile(clean) {
			continue
		}
		if clean == "web/src/main.ts" || clean == "web/src/app/app.route.ts" {
			allowMainSpec = true
		}
		if _, seen := visited[clean]; !seen {
			visited[clean] = struct{}{}
			queue = append(queue, clean)
		}

		// Also include a direct sibling <basename>.spec.ts if present on disk.
		siblingSpec := strings.TrimSuffix(clean, ".ts") + ".spec.ts"
		if _, ok := knownFiles[siblingSpec]; ok {
			specSet[strings.TrimPrefix(siblingSpec, "web/")] = struct{}{}
		}
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, importer := range reverseImporters[cur] {
			if importer == "web/src/main.spec.ts" && !allowMainSpec {
				continue
			}
			if strings.HasSuffix(importer, ".spec.ts") {
				specSet[strings.TrimPrefix(importer, "web/")] = struct{}{}
				continue
			}
			if !allowMainSpec && (importer == "web/src/main.ts" || importer == "web/src/app/app.route.ts") {
				continue
			}
			if _, seen := visited[importer]; !seen {
				visited[importer] = struct{}{}
				queue = append(queue, importer)
			}
		}
	}

	if len(specSet) == 0 {
		return TSTestPlan{SkipKarma: true}, nil
	}

	return TSTestPlan{
		SkipKarma:    false,
		IncludeSpecs: sortedSetKeys(specSet),
	}, nil
}

// IsContainerExecutionEnabled reports whether USE_CONTAINER is enabled in the process environment.
func IsContainerExecutionEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("USE_CONTAINER")))
	return v == "true" || v == "1"
}

// ResolveContainerCommand resolves the container runtime binary (docker/podman) and builder image name.
func ResolveContainerCommand() (string, string, error) {
	containerCmd := strings.TrimSpace(os.Getenv("CONTAINER_CMD"))
	if containerCmd == "" {
		if p, err := exec.LookPath("docker"); err == nil {
			containerCmd = p
		} else if p, err := exec.LookPath("podman"); err == nil {
			containerCmd = p
		} else {
			return "", "", fmt.Errorf("USE_CONTAINER is enabled, but neither docker nor podman was found")
		}
	}

	builderImage := strings.TrimSpace(os.Getenv("BUILDER_IMAGE"))
	if builderImage == "" {
		builderImage = DefaultBuilderImage
	}
	return containerCmd, builderImage, nil
}

// PickKarmaPort returns KARMA_PORT if set, or allocates an ephemeral localhost TCP port.
func PickKarmaPort(ctx context.Context) (string, error) {
	if envPort := strings.TrimSpace(os.Getenv("KARMA_PORT")); envPort != "" {
		return envPort, nil
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return "9876", nil
	}
	defer ln.Close()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok || addr.Port <= 0 {
		return "9876", nil
	}
	return strconv.Itoa(addr.Port), nil
}

func loadWebTSFiles(repoRoot string, webSrcDir string) (map[string]struct{}, map[string][]byte, error) {
	known := make(map[string]struct{})
	contents := make(map[string][]byte)

	err := filepath.WalkDir(webSrcDir, func(absPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "generated" || name == "assets" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".ts") || strings.HasSuffix(d.Name(), ".d.ts") {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, absPath)
		if err != nil {
			return err
		}
		slashRel := filepath.ToSlash(rel)
		data, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		known[slashRel] = struct{}{}
		contents[slashRel] = data
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed walking %s: %w", webSrcDir, err)
	}
	return known, contents, nil
}

func buildReverseImportGraph(knownFiles map[string]struct{}, contentsByFile map[string][]byte) map[string][]string {
	reverseSet := make(map[string]map[string]struct{})
	for importerRel, src := range contentsByFile {
		matches := tsImportSpecRegex.FindAllSubmatch(src, -1)
		for _, m := range matches {
			targetRel := resolveTSImportSpecifier(importerRel, string(m[1]), knownFiles)
			if targetRel == "" || targetRel == importerRel {
				continue
			}
			importers, ok := reverseSet[targetRel]
			if !ok {
				importers = make(map[string]struct{})
				reverseSet[targetRel] = importers
			}
			importers[importerRel] = struct{}{}
		}
	}

	reverse := make(map[string][]string, len(reverseSet))
	for target, set := range reverseSet {
		reverse[target] = sortedSetKeys(set)
	}
	return reverse
}

func resolveTSImportSpecifier(importerRel string, spec string, knownFiles map[string]struct{}) string {
	var base string
	switch {
	case strings.HasPrefix(spec, "src/"):
		base = "web/" + spec
	case strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../"):
		base = path.Clean(path.Join(path.Dir(importerRel), spec))
	default:
		return ""
	}

	candidates := []string{
		base,
		strings.TrimSuffix(base, ".js") + ".ts",
		base + "/index.ts",
	}
	for _, cand := range candidates {
		if _, ok := knownFiles[cand]; ok {
			return cand
		}
	}
	return ""
}

func sortedSetKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
