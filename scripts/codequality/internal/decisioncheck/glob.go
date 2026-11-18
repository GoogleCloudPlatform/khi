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
	"path"
	"strings"
)

// MatchGlob reports whether slash-separated relPath matches pattern, supporting *, ?, and ** path segments.
func MatchGlob(pattern string, relPath string) bool {
	patSegs := splitCleanSlash(pattern)
	pathSegs := splitCleanSlash(relPath)
	return matchSegments(patSegs, pathSegs)
}

// MatchesFilter reports whether relPath matches at least one include glob (or empty include) and zero exclude globs.
func MatchesFilter(relPath string, include []string, exclude []string) bool {
	for _, ex := range exclude {
		if MatchGlob(ex, relPath) {
			return false
		}
	}
	if len(include) == 0 {
		return true
	}
	for _, in := range include {
		if MatchGlob(in, relPath) {
			return true
		}
	}
	return false
}

func splitCleanSlash(p string) []string {
	trimmed := strings.Trim(strings.TrimPrefix(p, "./"), "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func matchSegments(pat []string, target []string) bool {
	if len(pat) == 0 {
		return len(target) == 0
	}
	if pat[0] == "**" {
		if matchSegments(pat[1:], target) {
			return true
		}
		return len(target) > 0 && matchSegments(pat, target[1:])
	}
	if len(target) == 0 {
		return false
	}
	matched, err := path.Match(pat[0], target[0])
	if err != nil || !matched {
		return false
	}
	return matchSegments(pat[1:], target[1:])
}
