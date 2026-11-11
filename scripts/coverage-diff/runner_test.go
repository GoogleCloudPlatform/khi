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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPlanTestExecution(t *testing.T) {
	tmpRepo := t.TempDir()
	mustCreateDummyGoFile(t, filepath.Join(tmpRepo, "pkg", "common", "set", "set.go"))
	mustCreateDummyGoFile(t, filepath.Join(tmpRepo, "pkg", "task", "inspection", "sample", "extractor.go"))
	mustCreateDummyGoFile(t, filepath.Join(tmpRepo, "pkg", "task", "inspection", "sample", "impl", "mapper.go"))

	testCases := []struct {
		name         string
		changedFiles []string
		full         bool
		want         TestPlan
	}{
		{
			name:         "no changed files skips execution in targeted mode",
			changedFiles: nil,
			full:         false,
			want: TestPlan{
				Skip: true,
			},
		},
		{
			name:         "full mode runs entire repository regardless of changed files",
			changedFiles: nil,
			full:         true,
			want: TestPlan{
				Packages: []string{"./..."},
			},
		},
		{
			name: "targeted mode runs only packages containing changed production files",
			changedFiles: []string{
				"pkg/common/set/set.go",
				"pkg/task/inspection/sample/extractor.go",
				"pkg/generated/ignored.go",
			},
			full: false,
			want: TestPlan{
				Packages: []string{
					"./pkg/common/set",
					"./pkg/task/inspection/sample",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := PlanTestExecution(tmpRepo, tc.changedFiles, tc.full)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("PlanTestExecution() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func mustCreateDummyGoFile(t *testing.T, fullPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("package dummy\n"), 0o644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}
}
