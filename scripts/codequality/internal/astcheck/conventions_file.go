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

package astcheck

import (
	"path"
	"path/filepath"
)

// CheckFileConventions checks Go conventions on only the explicitly listed repository-relative paths without walking or reading sibling files.
func CheckFileConventions(repoRoot string, relPaths []string) ([]ConventionViolation, error) {
	if len(relPaths) == 0 {
		return nil, nil
	}
	filesByDir := groupFilesByDir(relPaths)
	var violations []ConventionViolation
	for _, dir := range sortedDirKeys(filesByDir) {
		dirViolations, err := checkDirConventions(repoRoot, filesByDir[dir])
		if err != nil {
			return nil, err
		}
		violations = append(violations, dirViolations...)
	}
	sortConventionViolations(violations)
	return violations, nil
}

func groupFilesByDir(relPaths []string) map[string][]string {
	filesByDir := make(map[string][]string)
	seen := make(map[string]struct{}, len(relPaths))
	for _, relPath := range relPaths {
		slashRel := path.Clean(filepath.ToSlash(relPath))
		if slashRel == "." || slashRel == "" {
			continue
		}
		if _, ok := seen[slashRel]; ok {
			continue
		}
		seen[slashRel] = struct{}{}
		dir := path.Dir(slashRel)
		filesByDir[dir] = append(filesByDir[dir], slashRel)
	}
	return filesByDir
}
