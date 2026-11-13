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
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/overlay"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func TestEvaluateLazyBaseRatchetAndHardLimits(t *testing.T) {
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

func LegacyUnchanged(x int) int {
	if x < 0 {
		return -1
	}
	return x
}

func LegacyWorsened(x int) int {
	if x < 0 {
		return -1
	}
	if x > 100 {
		return 100
	}
	return x
}
`
	targetTest := `package svc

import "testing"

func TestSvc(t *testing.T) {
	if CleanCovered(1) != 2 || NewUncovered(1) != 1 || AllowedExternal(1) != 1 || LegacyUnchanged(1) != 1 || LegacyWorsened(1) != 1 {
		t.Fatal("unexpected")
	}
}
`
	baseSvc := `package svc

func LegacyUnchanged(x int) int {
	if x < 0 {
		return -1
	}
	return x
}

func LegacyWorsened(x int) int {
	if x < 0 {
		return -1
	}
	return x
}
`
	baseTest := `package svc

import "testing"

func TestSvc(t *testing.T) {
	if LegacyUnchanged(1) != 1 || LegacyWorsened(1) != 1 {
		t.Fatal("unexpected")
	}
}
`
	setupEvalCovRepo(t, repoDir, targetSvc, targetTest)

	changedProd := map[string]map[int]struct{}{
		"svc/svc.go": {4: {}, 8: {}, 16: {}, 23: {}, 31: {}},
		"svc/gen.go": {2: {}},
	}
	readBase := func(relPath string) ([]byte, bool, error) {
		switch relPath {
		case "svc/svc.go":
			return []byte(baseSvc), true, nil
		case "svc/svc_test.go":
			return []byte(baseTest), true, nil
		default:
			return nil, false, nil
		}
	}
	readTarget := func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(relPath)))
	}

	th := rule.DefaultThresholds()
	astRes, err := astcheck.Evaluate(astcheck.Options{
		Thresholds:             th,
		ChangedProdLinesByFile: changedProd,
		ReadTargetFile:         readTarget,
		ReadBaseFile:           readBase,
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
		EnsureBaseOverlay: func() (*overlay.Snapshot, error) {
			return overlay.Build(overlay.Config{
				RepoRoot:     repoDir,
				WorkDir:      filepath.Join(workDir, "ov"),
				ProdFiles:    []string{"svc/svc.go"},
				TestFiles:    []string{"svc/svc_test.go"},
				ReadBaseFile: readBase,
			})
		},
	})
	if evalErr != nil {
		t.Fatalf("covcheck.Evaluate failed: %v", evalErr)
	}

	assertCovResult(t, covRes, covExpectation{
		subjects:     []string{"NewUncovered", "LegacyWorsened"},
		kinds:        []rule.ViolationKind{rule.ViolationHardLimit, rule.ViolationRatchetRegression},
		suppressed:   1,
		baseOverlays: 1,
		hasOverall:   false,
		overallStmts: 0,
	})

	// Re-use the generated target.coverprofile as PrecomputedProfilePath and verify clean diff skips base overlay.
	cleanChanged := map[string]map[int]struct{}{"svc/svc.go": {4: {}}}
	cleanAST, err := astcheck.Evaluate(astcheck.Options{
		Thresholds:             th,
		ChangedProdLinesByFile: cleanChanged,
		ReadTargetFile:         readTarget,
		ReadBaseFile:           readBase,
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
		EnsureBaseOverlay: func() (*overlay.Snapshot, error) {
			t.Fatal("EnsureBaseOverlay must not run when pending is empty")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("Evaluate(precomputed) failed: %v", err)
	}
	assertCovResult(t, precompRes, covExpectation{
		hasOverall:   true,
		overallStmts: 15,
		overallCov:   10,
	})
}

func TestEvaluateEdgeModesAndBrokenBaseFallback(t *testing.T) {
	repoDir := t.TempDir()
	workDir := t.TempDir()
	targetSvc := `package svc

func LegacyBrokenBase(x int) int {
	if x < 0 {
		return -1
	}
	return x
}
`
	targetTest := `package svc

import "testing"

func TestSvc(t *testing.T) {
	if LegacyBrokenBase(1) != 1 {
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
		name            string
		precompPath     string
		changed         map[string]map[int]struct{}
		omitSnapBaseSrc bool
		wantErr         bool
		wantKinds       []rule.ViolationKind
		wantOverlays    int
	}{
		{
			name:        "missing precomputed profile returns error",
			precompPath: filepath.Join(workDir, "missing.cov"),
			wantErr:     true,
		},
		{
			name:    "empty changed files returns zero packages cleanly",
			changed: nil,
		},
		{
			name:            "broken base test isolates healthy sibling package and missing base source returns 0",
			changed:         map[string]map[int]struct{}{"svc/svc.go": {4: {}}, "peer/peer.go": {4: {}}},
			omitSnapBaseSrc: true,
			wantKinds:       []rule.ViolationKind{rule.ViolationHardLimit},
			wantOverlays:    1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			readBase := func(relPath string) ([]byte, bool, error) {
				switch relPath {
				case "svc/svc.go":
					return []byte(targetSvc), true, nil
				case "svc/svc_test.go":
					return []byte("package svc\nbroken_syntax\n"), true, nil
				case "peer/peer.go":
					return []byte("package peer\n\nfunc PeerLegacy(x int) int {\n\tif x < 0 {\n\t\treturn -1\n\t}\n\treturn x\n}\n"), true, nil
				default:
					return nil, false, nil
				}
			}
			astRes, err := astcheck.Evaluate(astcheck.Options{
				Thresholds:             th,
				ChangedProdLinesByFile: tc.changed,
				ReadTargetFile:         readTarget,
				ReadBaseFile:           readBase,
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
				EnsureBaseOverlay:      buildTestBaseOverlayFn(repoDir, filepath.Join(workDir, tc.name, "ov"), tc.omitSnapBaseSrc, readBase),
			})
			if (evalErr != nil) != tc.wantErr {
				t.Fatalf("Evaluate() err = %v, wantErr %v", evalErr, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			assertCovResult(t, res, covExpectation{
				kinds:        tc.wantKinds,
				baseOverlays: tc.wantOverlays,
			})
		})
	}
}

type covExpectation struct {
	subjects     []string
	kinds        []rule.ViolationKind
	suppressed   int
	baseOverlays int
	hasOverall   bool
	overallStmts int
	overallCov   int
}

func setupEvalCovRepo(t *testing.T, repoDir string, targetSvc string, targetTest string) {
	t.Helper()
	for _, sub := range []string{"svc", "peer"} {
		if err := os.MkdirAll(filepath.Join(repoDir, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	}
	for rel, content := range map[string]string{
		"go.mod":            "module example.com/evalcov\n\ngo 1.24\n",
		"svc/svc.go":        targetSvc,
		"svc/svc_test.go":   targetTest,
		"svc/gen.go":        "// Code generated by gen. DO NOT EDIT.\npackage svc\nfunc Gen() {}\n",
		"peer/peer.go":      "package peer\n\nfunc PeerLegacy(x int) int {\n\tif x < 0 {\n\t\treturn -1\n\t}\n\treturn x\n}\n",
		"peer/peer_test.go": "package peer\n\nimport \"testing\"\n\nfunc TestPeer(t *testing.T) {\n\tif PeerLegacy(1) != 1 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n",
	} {
		if err := os.WriteFile(filepath.Join(repoDir, filepath.FromSlash(rel)), []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}
}

func buildTestBaseOverlayFn(
	repoDir string,
	workDir string,
	omitProd bool,
	readBase func(string) ([]byte, bool, error),
) func() (*overlay.Snapshot, error) {
	return func() (*overlay.Snapshot, error) {
		prodFiles := []string{"peer/peer.go"}
		if !omitProd {
			prodFiles = append(prodFiles, "svc/svc.go")
		}
		return overlay.Build(overlay.Config{
			RepoRoot:     repoDir,
			WorkDir:      workDir,
			ProdFiles:    prodFiles,
			TestFiles:    []string{"svc/svc_test.go"},
			ReadBaseFile: readBase,
		})
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
	if len(res.Suppressed) != want.suppressed {
		t.Errorf("len(Suppressed) = %d, want %d", len(res.Suppressed), want.suppressed)
	}
	if res.Summary.BaseOverlayRuns != want.baseOverlays {
		t.Errorf("BaseOverlayRuns = %d, want %d", res.Summary.BaseOverlayRuns, want.baseOverlays)
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
