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
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func TestPkgScanHelpers(t *testing.T) {
	dir := t.TempDir()
	// Populate non-test .go file and a directory ending in _test.go so && cannot be mutated to ||.
	if err := os.MkdirAll(filepath.Join(dir, "subdir_test.go"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prod.go"), []byte("package a\n"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	testCases := []struct {
		name      string
		targetDir string
		writeTest bool
		pkgDir    string
		wantTests bool
		wantLabel string
	}{
		{
			name:      "non-existent dir returns false without error",
			targetDir: filepath.Join(dir, "missing"),
			pkgDir:    ".",
			wantTests: false,
			wantLabel: ".",
		},
		{
			name:      "dir with prod file and _test.go subdir still returns false",
			targetDir: dir,
			writeTest: false,
			pkgDir:    ".",
			wantTests: false,
			wantLabel: ".",
		},
		{
			name:      "present test file returns true and subpkg label has slash",
			targetDir: dir,
			writeTest: true,
			pkgDir:    "pkg/foo",
			wantTests: true,
			wantLabel: "./pkg/foo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.writeTest {
				if err := os.WriteFile(filepath.Join(tc.targetDir, "a_test.go"), []byte("package a\n"), 0o600); err != nil {
					t.Fatalf("WriteFile failed: %v", err)
				}
			}
			has, err := hasPackageTestFiles(tc.targetDir)
			if err != nil || has != tc.wantTests {
				t.Errorf("hasPackageTestFiles() = (%v, %v), want (%v, nil)", has, err, tc.wantTests)
			}
			if got := formatPkgLabel(tc.pkgDir); got != tc.wantLabel {
				t.Errorf("formatPkgLabel(%q) = %q, want %q", tc.pkgDir, got, tc.wantLabel)
			}
		})
	}

	mPend := &Mutant{ID: 1, Status: rule.MutantPending}
	mSkip := &Mutant{ID: 4, Status: rule.MutantSkippedBudget}
	plans := []*FilePlan{{Mutants: []*Mutant{mPend, mSkip}}}
	markPendingMutants(plans, rule.MutantUncovered)
	if mPend.Status != rule.MutantUncovered {
		t.Errorf("mPend.Status = %q, want %q", mPend.Status, rule.MutantUncovered)
	}
	skipped := collectBudgetSkippedMutantIDs(plans)
	if _, ok := skipped[4]; !ok {
		t.Errorf("expected mutant 4 in skipped set, got %v", skipped)
	}
}
