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

package covcheck

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

// PlanRequest specifies how to choose Go packages for coverage execution.
type PlanRequest struct {
	RepoRoot  string
	ProdFiles []string
	Full      bool
}

// RunRequest configures a `go test -coverprofile` execution.
type RunRequest struct {
	RepoRoot    string
	ModulePath  string
	Packages    []string
	WorkDir     string
	ProfileName string
	ErrOut      io.Writer
}

// PlanPackages returns sorted Go package specifiers (e.g. "./pkg/foo") for req.
func PlanPackages(req PlanRequest) []string {
	if req.Full {
		return []string{"./..."}
	}
	pkgSet := make(map[string]struct{})
	for _, relFile := range req.ProdFiles {
		clean := strings.TrimPrefix(filepath.ToSlash(relFile), "./")
		if !gotestutil.IsTargetGoFile(clean) {
			continue
		}
		relDir := path.Dir(clean)
		absDir := filepath.Join(req.RepoRoot, filepath.FromSlash(relDir))
		if !dirHasGoFiles(absDir) {
			continue
		}
		pkgSet[formatPkgSpec(relDir)] = struct{}{}
	}
	out := make([]string, 0, len(pkgSet))
	for pkg := range pkgSet {
		out = append(out, pkg)
	}
	slices.Sort(out)
	return out
}

func formatPkgSpec(relDir string) string {
	if relDir == "." {
		return "."
	}
	return "./" + relDir
}

func dirHasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			return true
		}
	}
	return false
}

// RunCoverProfile runs go test with -coverprofile on req.Packages and returns parsed CoverBlocks per file.
func RunCoverProfile(ctx context.Context, req RunRequest) (map[string][]gotestutil.CoverBlock, error) {
	if len(req.Packages) == 0 {
		return map[string][]gotestutil.CoverBlock{}, nil
	}
	if err := os.MkdirAll(req.WorkDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create coverage workdir: %w", err)
	}
	profilePath := filepath.Join(req.WorkDir, req.ProfileName)
	args := make([]string, 0, 2+len(req.Packages))
	args = append(args, "test", "-coverprofile="+profilePath)
	args = append(args, req.Packages...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = req.RepoRoot
	cmd.Env = append(os.Environ(), "KHI_SKIP_CLOUD_LOGGING=true")

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		if req.ErrOut != nil && combined.Len() > 0 {
			_, _ = io.Copy(req.ErrOut, &combined)
		}
		return nil, fmt.Errorf("go test coverage failed (%s): %w", strings.Join(args, " "), err)
	}
	return gotestutil.LoadCoverProfileIfPresent(profilePath, req.ModulePath)
}
