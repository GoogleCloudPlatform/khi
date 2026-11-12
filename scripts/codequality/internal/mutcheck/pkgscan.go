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

package mutcheck

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/redundancy"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func collectPackageChangedTestFuncs(
	repoRoot string,
	relTestFiles []string,
	changedTestLines map[string]map[int]struct{},
) ([]redundancy.ChangedTestFunc, error) {
	var out []redundancy.ChangedTestFunc
	for _, relFile := range relTestFiles {
		absPath := filepath.Join(repoRoot, filepath.FromSlash(relFile))
		src, err := os.ReadFile(absPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("failed to read test file %s: %w", relFile, err)
		}
		funcs, err := redundancy.DiscoverChangedTestFuncs(relFile, src, changedTestLines[relFile])
		if err != nil {
			return nil, err
		}
		out = append(out, funcs...)
	}
	return out, nil
}

func hasPackageTestFiles(pkgAbsDir string) (bool, error) {
	entries, err := os.ReadDir(pkgAbsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read package directory %s: %w", pkgAbsDir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
			return true, nil
		}
	}
	return false, nil
}

func markPendingMutants(plans []*FilePlan, status rule.MutantStatus) {
	for _, p := range plans {
		for _, m := range p.Mutants {
			if m.Status == rule.MutantPending {
				m.Status = status
			}
		}
	}
}

func groupSurvivedByFunc(plans []*FilePlan) map[string][]rule.MutantDetail {
	out := make(map[string][]rule.MutantDetail)
	for _, p := range plans {
		for _, m := range p.Mutants {
			if m.Status == rule.MutantSurvived {
				key := p.File + "|" + m.FuncKey
				out[key] = append(out[key], m.ToDetail())
			}
		}
	}
	return out
}

func collectBudgetSkippedMutantIDs(plans []*FilePlan) map[int]struct{} {
	skipped := make(map[int]struct{})
	for _, p := range plans {
		for _, m := range p.Mutants {
			if m.Status == rule.MutantSkippedBudget && m.ID > 0 {
				skipped[m.ID] = struct{}{}
			}
		}
	}
	return skipped
}

func lineRangeSet(startLine int, endLine int) map[int]struct{} {
	set := make(map[int]struct{}, endLine-startLine+1)
	for ln := startLine; ln <= endLine; ln++ {
		set[ln] = struct{}{}
	}
	return set
}

func cloneDirectives(in []rule.Directive) []rule.Directive {
	out := make([]rule.Directive, len(in))
	copy(out, in)
	return out
}

func sortedMapKeys(m map[string][]astcheck.FuncMetrics) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func formatPkgLabel(pkgDir string) string {
	if pkgDir == "." {
		return "."
	}
	return "./" + pkgDir
}
