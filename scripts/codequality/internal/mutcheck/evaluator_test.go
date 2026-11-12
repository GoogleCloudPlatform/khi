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
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/overlay"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
	"github.com/google/go-cmp/cmp"
)

func TestEvaluateSurvivingMutantsRatchetAndRedundancy(t *testing.T) {
	repoDir := t.TempDir()
	workDir := t.TempDir()

	targetSvc := `package svc

func KilledClean(x int) int {
	return x + 1
}

func NewWeak(x int) bool {
	return x >= 10
}

func LegacyUnchangedWeak(x int) bool {
	return x >= 10
}

func LegacyWorsenedWeak(x int, y int) bool {
	return x >= 10 && y >= 10
}

func UnreachedByRuntime(x int) int {
	return x - 1
}

func UncoveredByProfile(x int) int {
	return x * 2
}
`
	targetTest := `package svc

import "testing"

func TestSvc(t *testing.T) {
	testCases := []struct {
		name string
		x    int
		want int
	}{
		{name: "case one", x: 1, want: 2},
		{name: "duplicate case", x: 5, want: 6},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := KilledClean(tc.x); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
	if !NewWeak(20) || !LegacyUnchangedWeak(20) || !LegacyWorsenedWeak(20, 20) {
		t.Fatal("expected true")
	}
}
`
	baseSvc := `package svc

func LegacyUnchangedWeak(x int) bool {
	return x >= 10
}

func LegacyWorsenedWeak(x int, y int) bool {
	return x >= 10
}
`
	baseTest := `package svc

import "testing"

func TestSvc(t *testing.T) {
	if !LegacyUnchangedWeak(20) || !LegacyWorsenedWeak(20, 20) {
		t.Fatal("expected true")
	}
}
`

	setupMutEvalRepo(t, repoDir, targetSvc, targetTest)

	changedProd := map[string]map[int]struct{}{
		"bisectdrop/drop.go": {6: {}},
		"empty/empty.go":     {3: {}},
		"notest/notest.go":   {3: {}},
		"suppkg/sup.go":      {5: {}},
		"svc/svc.go":         {4: {}, 8: {}, 12: {}, 16: {}, 20: {}, 24: {}},
	}
	changedTest := map[string]map[int]struct{}{
		"svc/svc_test.go": {12: {}},
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
		ChangedTestLinesByFile: changedTest,
		ReadTargetFile:         readTarget,
		ReadBaseFile:           readBase,
	})
	if err != nil {
		t.Fatalf("astcheck.Evaluate failed: %v", err)
	}

	prog := &spyProgress{}
	mutRes, evalErr := Evaluate(context.Background(), Options{
		RepoRoot:               repoDir,
		WorkDir:                workDir,
		Thresholds:             th,
		ChangedProdLinesByFile: changedProd,
		ChangedTestLinesByFile: changedTest,
		HasCoverProfile:        true,
		CoverBlocksByFile: map[string][]gotestutil.CoverBlock{
			"bisectdrop/drop.go": {{StartLine: 1, EndLine: 8, Count: 1}},
			"notest/notest.go":   {{StartLine: 1, EndLine: 5, Count: 1}},
			"suppkg/sup.go":      {{StartLine: 1, EndLine: 6, Count: 1}},
			"svc/svc.go":         {{StartLine: 1, EndLine: 21, Count: 1}},
		},
		ASTResult:  astRes,
		Workers:    2,
		MaxMutants: 80,
		Timeout:    30 * time.Second,
		Progress:   prog,
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
		t.Fatalf("mutcheck.Evaluate failed: %v", evalErr)
	}

	assertMutEvalResult(t, mutRes, prog)

	// Verify canceled context exits package loop cleanly and summary status counter covers all variants.
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Evaluate(canceledCtx, Options{
		ChangedProdLinesByFile: changedProd,
		ASTResult:              astRes,
	}); err != nil {
		t.Errorf("Evaluate(canceled) unexpected error: %v", err)
	}
	verifyAllSummaryStatuses(t)
}

func setupMutEvalRepo(t *testing.T, repoDir string, targetSvc string, targetTest string) {
	t.Helper()
	for _, sub := range []string{"svc", "notest", "empty", "suppkg", "bisectdrop"} {
		if err := os.MkdirAll(filepath.Join(repoDir, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	}
	for rel, content := range map[string]string{
		"go.mod":                  "module example.com/evalmut\n\ngo 1.24\n",
		"bisectdrop/drop.go":      "package bisectdrop\n\ntype customBool bool\n\nfunc NegCustom(b customBool) customBool {\n\treturn !b\n}\n",
		"bisectdrop/drop_test.go": "package bisectdrop\n\nimport \"testing\"\n\nfunc TestNegCustom(t *testing.T) { _ = NegCustom(true) }\n",
		"empty/empty.go":          "package empty\n\nfunc NoMutants() {}\n",
		"notest/notest.go":        "package notest\n\nfunc Untested(x int) int { return x + 1 }\n",
		"suppkg/sup.go":           "package suppkg\n\n// CODEQUALITY_ALLOW(func_surviving_mutants, equivalent): symmetric add\nfunc Allowed(x int) int {\n\treturn x + 1\n}\n",
		"suppkg/sup_test.go":      "package suppkg\n\nimport \"testing\"\n\nfunc TestAllowed(t *testing.T) { _ = Allowed(1) }\n",
		"svc/svc.go":              targetSvc,
		"svc/svc_test.go":         targetTest,
	} {
		if err := os.WriteFile(filepath.Join(repoDir, filepath.FromSlash(rel)), []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}
}

func assertMutEvalResult(t *testing.T, mutRes *Result, prog *spyProgress) {
	t.Helper()
	wantRules := []rule.RuleID{
		rule.RuleFuncSurvivingMutants,
		rule.RuleFuncSurvivingMutants,
		rule.RuleTestRedundancy,
	}
	wantSubjects := []string{"NewWeak", "LegacyWorsenedWeak", "TestSvc"}
	wantKinds := []rule.ViolationKind{
		rule.ViolationHardLimit,
		rule.ViolationRatchetRegression,
		rule.ViolationHardLimit,
	}
	var gotRules []rule.RuleID
	var gotSubjects []string
	var gotKinds []rule.ViolationKind
	for _, f := range mutRes.Findings {
		gotRules = append(gotRules, f.Rule)
		gotSubjects = append(gotSubjects, f.Subject)
		gotKinds = append(gotKinds, f.Kind)
	}
	if diff := cmp.Diff(wantRules, gotRules); diff != "" {
		t.Errorf("Finding rules mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantSubjects, gotSubjects); diff != "" {
		t.Errorf("Finding subjects mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantKinds, gotKinds); diff != "" {
		t.Errorf("Finding kinds mismatch (-want +got):\n%s", diff)
	}
	if mutRes.Summary.BaseOverlayRuns != 2 {
		t.Errorf("BaseOverlayRuns = %d, want 2", mutRes.Summary.BaseOverlayRuns)
	}
	if prog.phases.Load() == 0 || prog.pools.Load() == 0 || prog.results.Load() == 0 {
		t.Errorf("expected progress callbacks, got phases=%d pools=%d results=%d", prog.phases.Load(), prog.pools.Load(), prog.results.Load())
	}
}

func verifyAllSummaryStatuses(t *testing.T) {
	t.Helper()
	sum := rule.MutationSummary{ByCategory: make(map[rule.AllowCategory]int)}
	plans := []*FilePlan{{Mutants: []*Mutant{
		{Status: rule.MutantKilled},
		{Status: rule.MutantTimedOut},
		{Status: rule.MutantSurvived},
		{Status: rule.MutantUncovered},
		{Status: rule.MutantSuppressed, SuppressionCategory: rule.CategoryEquivalent},
		{Status: rule.MutantSkippedBudget},
		{Status: rule.MutantCompileSkipped},
	}}}
	accumulateSummary(&sum, plans)
	if sum.Killed != 1 || sum.TimedOut != 1 || sum.Survived != 1 || sum.Uncovered != 1 || sum.Suppressed != 1 || sum.SkippedBudget != 1 || sum.CompileSkipped != 1 {
		t.Errorf("unexpected summary counts: %+v", sum)
	}
}
