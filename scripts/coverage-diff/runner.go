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

package main

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

// TestPlan describes which Go packages to execute for differential coverage.
type TestPlan struct {
	Skip     bool
	Packages []string
}

// PlanTestExecution builds a minimal or full Go test execution plan from changed production Go files.
func PlanTestExecution(repoRoot string, changedFiles []string, full bool) TestPlan {
	if full {
		return TestPlan{
			Packages: []string{"./..."},
		}
	}
	if len(changedFiles) == 0 {
		return TestPlan{Skip: true}
	}

	pkgSet := make(map[string]struct{})
	for _, relFile := range changedFiles {
		cleanFile := strings.TrimPrefix(filepath.ToSlash(relFile), "./")
		if !gotestutil.IsTargetProductionGoFile(cleanFile) {
			continue
		}
		relDir := path.Dir(cleanFile)
		absDir := filepath.Join(repoRoot, filepath.FromSlash(relDir))
		if !hasGoFiles(absDir) {
			continue
		}

		pkgSpec := "./" + relDir
		if relDir == "." {
			pkgSpec = "."
		}
		pkgSet[pkgSpec] = struct{}{}
	}

	if len(pkgSet) == 0 {
		return TestPlan{Skip: true}
	}

	return TestPlan{
		Packages: sortedKeys(pkgSet),
	}
}

// RunGoTestCoverage executes go test according to plan and writes the coverage profile to profileOut.
func RunGoTestCoverage(ctx context.Context, repoRoot string, plan TestPlan, profileOut string, errOut io.Writer) error {
	if plan.Skip {
		return nil
	}

	args := make([]string, 0, 2+len(plan.Packages))
	args = append(args, "test", "-coverprofile="+profileOut)
	args = append(args, plan.Packages...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "KHI_SKIP_CLOUD_LOGGING=true")

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	if err := cmd.Run(); err != nil {
		if errOut != nil && combined.Len() > 0 {
			_, _ = io.Copy(errOut, &combined)
		}
		return fmt.Errorf("go test failed (%s): %w", strings.Join(args, " "), err)
	}
	return nil
}

func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
