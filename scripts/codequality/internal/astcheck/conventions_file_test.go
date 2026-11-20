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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCheckFileConventionsPathNormalization(t *testing.T) {
	testCases := []struct {
		name     string
		files    map[string]string
		relPaths []string
		want     []ConventionViolation
	}{
		// Ignores empty and dot entries while deduplicating repeated relative paths so each violation is reported once.
		{
			name: "ignores empty and dot entries and deduplicates repeated paths",
			files: map[string]string{
				"pkg/a/target.go": `package a

type PubType struct {
	Missing int
}
`,
			},
			relPaths: []string{"", ".", "pkg/a/target.go", "./pkg/a/target.go"},
			want: []ConventionViolation{
				{
					File:    "pkg/a/target.go",
					Line:    4,
					Column:  2,
					Rule:    RuleExportedMemberDoc,
					Message: "exported struct field PubType.Missing must have a GoDoc comment",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := writeIsolationFixtureRepo(t, tc.files)

			got, err := CheckFileConventions(repoRoot, tc.relPaths)

			if err != nil {
				t.Fatalf("CheckFileConventions() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("CheckFileConventions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
