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
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

var (
	goDiagFilePattern       = regexp.MustCompile(`(?m)^\s*([^\s:]+\.go):(\d+)(?::\d+)?:`)
	deadcodeDiagFilePattern = regexp.MustCompile(`^([^\s:]+):(\d+)(?::\d+)?:`)
)

// ClassifyGoCompileError determines whether a fast-mode Go compiler or typecheck failure is isolated to files outside targetFiles.
func ClassifyGoCompileError(output string, targetFiles map[string]struct{}, speed rule.Speed) (bool, string) {
	if speed != rule.SpeedFast || strings.TrimSpace(output) == "" {
		return false, ""
	}
	if strings.Contains(output, "--- FAIL:") {
		return false, ""
	}

	matches := goDiagFilePattern.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return false, ""
	}

	externalSet := make(map[string]struct{})
	for _, match := range matches {
		diagFile := normalizeSlashPath(match[1])
		if _, inTarget := targetFiles[diagFile]; inTarget {
			return false, ""
		}
		externalSet[diagFile] = struct{}{}
	}

	externalFiles := make([]string, 0, len(externalSet))
	for file := range externalSet {
		externalFiles = append(externalFiles, file)
	}
	slices.Sort(externalFiles)
	return true, fmt.Sprintf("compile error isolated to external files outside target diff (%s)", strings.Join(externalFiles, ", "))
}

// FilterDeadcodeOutput retains only deadcode diagnostic lines whose file path matches at least one pattern in paths.
func FilterDeadcodeOutput(output string, paths []string) string {
	if len(paths) == 0 {
		return output
	}
	patterns, matchAll := gotestutil.NormalizePathPatterns(paths)
	if matchAll {
		return output
	}

	lines := strings.Split(output, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		match := deadcodeDiagFilePattern.FindStringSubmatch(trimmed)
		if len(match) == 0 {
			continue
		}
		if gotestutil.MatchesAnyPathPattern(match[1], patterns) {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func normalizeSlashPath(raw string) string {
	return strings.TrimPrefix(filepath.ToSlash(raw), "./")
}
