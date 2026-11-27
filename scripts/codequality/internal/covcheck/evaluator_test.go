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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestEvaluateDifferentialCoverageAdvisories verifies changed-line scoping, direct ViolationAdvisory emission, and precomputed profile loading.
func TestEvaluateDifferentialCoverageAdvisories(t *testing.T) {
	repoDir := t.TempDir()
	workDir := t.TempDir()

	targetSvc := `package svc

func CleanCovered(x int) int {
	return x + 1
}

func NewUncovered(x int) int {
	if x < 0 {
		return -1
	}
	return x
}

// CODEQUALITY_ALLOW(func_uncovered_lines, external): external tty boundary
func AllowedExternal(x int) int {
	if x < 0 {
		return -1
	}
	return x
}

func UnchangedUncoveredInTouched(x int) int {
	if x < 0 {
		return -1
	}
	return x
}
`
	targetTest := `package svc

import "testing"

func TestSvc(t *testing.T) {
	if CleanCovered(1) != 2 || NewUncovered(1) != 1 || AllowedExternal(1) != 1 || UnchangedUncoveredInTouched(1) != 1 {
		t.Fatal("unexpected")
	}
}
`
	setupEvalCovRepo(t, repoDir, targetSvc, targetTest)

	// Line 9 (NewUncovered) and line 17 (AllowedExternal) are changed and uncovered;
	// line 22 (UnchangedUncoveredInTouched header/if) is touched without touching uncovered line 24.
	changedProd := map[string]map[int]struct{}{
		"svc/svc.go": {4: {}, 9: {}, 17: {}, 23: {}},
		"svc/gen.go": {2: {}},
	}
	readTarget := func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(relPath)))
	}

	th := rule.DefaultThresholds()
	astRes, err := astcheck.Evaluate(astcheck.Options{
		Thresholds:             th,
		ChangedProdLinesByFile: changedProd,
		ReadTargetFile:         readTarget,
	})
	if err != nil {
		t.Fatalf("astcheck.Evaluate failed: %v", err)
	}

	covRes, evalErr := Evaluate(context.Background(), Options{
		RepoRoot:               repoDir,
		ModulePath:             "example.com/evalcov",
		WorkDir:                workDir,
		Thresholds:             th,
		ChangedProdLinesByFile: changedProd,
		ASTResult:              astRes,
		ReadTargetFile:         readTarget,
	})
	if evalErr != nil {
		t.Fatalf("covcheck.Evaluate failed: %v", evalErr)
	}

	assertCovResult(t, covRes, covExpectation{
		subjects:     []string{"NewUncovered", "AllowedExternal"},
		kinds:        []rule.ViolationKind{rule.ViolationAdvisory, rule.ViolationAdvisory},
		hasOverall:   false,
		overallStmts: 0,
	})

	// Re-use target.coverprofile as PrecomputedProfilePath on a clean changed line.
	cleanChanged := map[string]map[int]struct{}{"svc/svc.go": {4: {}}}
	cleanAST, err := astcheck.Evaluate(astcheck.Options{
		Thresholds:             th,
		ChangedProdLinesByFile: cleanChanged,
		ReadTargetFile:         readTarget,
	})
	if err != nil {
		t.Fatalf("astcheck.Evaluate(clean) failed: %v", err)
	}
	precompRes, err := Evaluate(context.Background(), Options{
		RepoRoot:               repoDir,
		ModulePath:             "example.com/evalcov",
		WorkDir:                workDir,
		PrecomputedProfilePath: filepath.Join(workDir, "target.coverprofile"),
		Thresholds:             th,
		ChangedProdLinesByFile: cleanChanged,
		ASTResult:              cleanAST,
		ReadTargetFile:         readTarget,
	})
	if err != nil {
		t.Fatalf("Evaluate(precomputed) failed: %v", err)
	}
	assertCovResult(t, precompRes, covExpectation{
		hasOverall:   true,
		overallStmts: 10,
		overallCov:   7,
	})
}

// TestEvaluateEdgeModes verifies error handling for missing precomputed profiles and clean no-op behavior on empty diffs.
func TestEvaluateEdgeModes(t *testing.T) {
	repoDir := t.TempDir()
	workDir := t.TempDir()
	targetSvc := `package svc

func CleanOnly(x int) int {
	return x
}
`
	targetTest := `package svc

import "testing"

func TestSvc(t *testing.T) {
	if CleanOnly(1) != 1 {
		t.Fatal("unexpected")
	}
}
`
	setupEvalCovRepo(t, repoDir, targetSvc, targetTest)
	th := rule.DefaultThresholds()
	readTarget := func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(relPath)))
	}

	testCases := []struct {
		name        string
		precompPath string
		changed     map[string]map[int]struct{}
		wantErr     bool
	}{
		// Returns an error when PrecomputedProfilePath does not exist.
		{
			name:        "missing precomputed profile returns error",
			precompPath: filepath.Join(workDir, "missing.cov"),
			wantErr:     true,
		},
		// Returns empty results cleanly when no production lines changed.
		{
			name:    "empty changed files returns zero packages cleanly",
			changed: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			astRes, err := astcheck.Evaluate(astcheck.Options{
				Thresholds:             th,
				ChangedProdLinesByFile: tc.changed,
				ReadTargetFile:         readTarget,
			})
			if err != nil {
				t.Fatalf("astcheck.Evaluate failed: %v", err)
			}
			res, evalErr := Evaluate(context.Background(), Options{
				RepoRoot:               repoDir,
				ModulePath:             "example.com/evalcov",
				WorkDir:                filepath.Join(workDir, tc.name),
				PrecomputedProfilePath: tc.precompPath,
				Thresholds:             th,
				ChangedProdLinesByFile: tc.changed,
				ASTResult:              astRes,
				ReadTargetFile:         readTarget,
			})
			if (evalErr != nil) != tc.wantErr {
				t.Fatalf("Evaluate() err = %v, wantErr %v", evalErr, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			assertCovResult(t, res, covExpectation{})
		})
	}
}

type covExpectation struct {
	subjects     []string
	kinds        []rule.ViolationKind
	hasOverall   bool
	overallStmts int
	overallCov   int
}

func setupEvalCovRepo(t *testing.T, repoDir string, targetSvc string, targetTest string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repoDir, "svc"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	for rel, content := range map[string]string{
		"go.mod":          "module example.com/evalcov\n\ngo 1.24\n",
		"svc/svc.go":      targetSvc,
		"svc/svc_test.go": targetTest,
		"svc/gen.go":      "// Code generated by gen. DO NOT EDIT.\npackage svc\nfunc Gen() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(repoDir, filepath.FromSlash(rel)), []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}
}

func assertCovResult(t *testing.T, res *Result, want covExpectation) {
	t.Helper()
	var gotSubjects []string
	var gotKinds []rule.ViolationKind
	for _, f := range res.Findings {
		gotSubjects = append(gotSubjects, f.Subject)
		gotKinds = append(gotKinds, f.Kind)
	}
	if want.subjects != nil {
		if diff := cmp.Diff(want.subjects, gotSubjects); diff != "" {
			t.Errorf("Finding subjects mismatch (-want +got):\n%s", diff)
		}
	}
	if diff := cmp.Diff(want.kinds, gotKinds); diff != "" {
		t.Errorf("Finding kinds mismatch (-want +got):\n%s", diff)
	}
	if res.Summary.HasOverall != want.hasOverall {
		t.Errorf("HasOverall = %v, want %v", res.Summary.HasOverall, want.hasOverall)
	}
	if want.overallStmts > 0 || !want.hasOverall {
		if res.Summary.OverallStatements != want.overallStmts {
			t.Errorf("OverallStatements = %d, want %d", res.Summary.OverallStatements, want.overallStmts)
		}
	}
	if want.overallCov > 0 && res.Summary.OverallCovered != want.overallCov {
		t.Errorf("OverallCovered = %d, want %d", res.Summary.OverallCovered, want.overallCov)
	}
}
