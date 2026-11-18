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

import "testing"

func TestMatchGlobAndFilter(t *testing.T) {
	testCases := []struct {
		name    string
		relPath string
		include []string
		exclude []string
		want    bool
	}{
		// Matches nested Go implementation files under pkg/** when not excluded.
		{
			name:    "matches recursive pkg glob",
			relPath: "pkg/task/inspection/common/k8saudit/impl/module.go",
			include: []string{"pkg/**"},
			exclude: []string{"pkg/generated/**", "**/testdata/**"},
			want:    true,
		},
		// Rejects files inside any nested testdata directory matched by **/testdata/**.
		{
			name:    "excludes nested testdata path via leading double star",
			relPath: "pkg/server/mcp/testdata/run_inspection.golden.md",
			include: []string{"pkg/**"},
			exclude: []string{"**/testdata/**"},
			want:    false,
		},
		// Rejects generated protobuf code under pkg/generated/**.
		{
			name:    "excludes generated prefix",
			relPath: "pkg/generated/khifile/v6/file.pb.go",
			include: []string{"pkg/**"},
			exclude: []string{"pkg/generated/**"},
			want:    false,
		},
		// Allows ** to match zero intermediate path segments for files directly under pkg/.
		{
			name:    "matches root-level pattern when double star matches zero segments",
			relPath: "pkg/GEMINI.md",
			include: []string{"pkg/**/*.md"},
			want:    true,
		},
		// Treats an empty include slice as matching every path not rejected by exclude.
		{
			name:    "empty include matches all non-excluded files",
			relPath: "README.md",
			exclude: []string{"*_test.go"},
			want:    true,
		},
		// Rejects paths outside all configured include globs.
		{
			name:    "non-matching include returns false",
			relPath: "web/src/app/app.component.ts",
			include: []string{"pkg/**"},
			want:    false,
		},
		// Fails closed to false when a glob pattern contains malformed bracket syntax.
		{
			name:    "invalid bracket syntax fails closed",
			relPath: "pkg/a.go",
			include: []string{"pkg/[unclosed"},
			want:    false,
		},
		// Matches an empty relative path against an empty pattern literal.
		{
			name:    "empty pattern only matches empty path",
			relPath: "",
			include: []string{""},
			want:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchesFilter(tc.relPath, tc.include, tc.exclude)

			if got != tc.want {
				t.Errorf("MatchesFilter(%q, %v, %v) = %v, want %v", tc.relPath, tc.include, tc.exclude, got, tc.want)
			}
		})
	}
}
