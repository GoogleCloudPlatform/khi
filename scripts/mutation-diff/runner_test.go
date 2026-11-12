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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestBuildMutantEnvStripsOuterMutantVariables(t *testing.T) {
	outer := []string{
		"PATH=/usr/bin",
		"KHI_MUTANT_ID=99",
		"KHI_MUTANT_TRACE_FILE=/tmp/outer.trace",
	}
	got := buildMutantEnv(outer, 7, "/tmp/inner.trace")
	want := []string{
		"PATH=/usr/bin",
		"KHI_SKIP_CLOUD_LOGGING=true",
		"KHI_MUTANT_ID=7",
		"KHI_MUTANT_TRACE_FILE=/tmp/inner.trace",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("buildMutantEnv() mismatch (-want +got):\n%s", diff)
	}
}

func TestApplyMutantBudgetRoundRobin(t *testing.T) {
	mA1 := &Mutant{File: "pkg/a/a.go", ID: 1, Status: MutantPending}
	mA2 := &Mutant{File: "pkg/a/a.go", ID: 2, Status: MutantPending}
	mA3 := &Mutant{File: "pkg/a/a.go", ID: 3, Status: MutantPending}
	mB1 := &Mutant{File: "pkg/b/b.go", ID: 1, Status: MutantPending}

	jobs := []mutantJob{
		{mutant: mA1},
		{mutant: mA2},
		{mutant: mA3},
		{mutant: mB1},
	}

	selected := applyMutantBudget(jobs, 3)
	var gotSelected []string
	for _, j := range selected {
		gotSelected = append(gotSelected, j.mutant.File+":"+string(rune('0'+j.mutant.ID)))
	}
	wantSelected := []string{
		"pkg/a/a.go:1",
		"pkg/b/b.go:1",
		"pkg/a/a.go:2",
	}
	if diff := cmp.Diff(wantSelected, gotSelected); diff != "" {
		t.Errorf("applyMutantBudget() selected mismatch (-want +got):\n%s", diff)
	}
	if mA3.Status != MutantSkippedBudget {
		t.Errorf("unselected mutant status = %v, want %v", mA3.Status, MutantSkippedBudget)
	}
}

func TestComputePerMutantTimeout(t *testing.T) {
	testCases := []struct {
		name     string
		baseline time.Duration
		want     time.Duration
	}{
		{
			name:     "enforces minimum 3s floor for fast packages",
			baseline: 100 * time.Millisecond,
			want:     3 * time.Second,
		},
		{
			name:     "scales 5x for slower baseline packages",
			baseline: 2 * time.Second,
			want:     10 * time.Second,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := computePerMutantTimeout(tc.baseline)
			if got != tc.want {
				t.Errorf("computePerMutantTimeout(%v) = %v, want %v", tc.baseline, got, tc.want)
			}
		})
	}
}

func TestExecuteMutationRunE2E(t *testing.T) {
	repoDir := t.TempDir()

	goMod := `module example.com/muttest

go 1.23
`
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	pkgDir := filepath.Join(repoDir, "pkg", "quota")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("failed to mkdir pkg/quota: %v", err)
	}

	// Production file with:
	// 1. Killed mutant: ValidateExact checks limit == 10 and test verifies 10 and 9.
	// 2. Surviving SBR mutant: UpdateAudit updates q.Updated = true, but test only checks err == nil!
	// 3. Uncovered mutant: NeverCalled is not called by any test.
	// 4. Short-circuit safety: SafeCheck uses q != nil && q.Limit > 0 with nil test input.
	prodSrc := `package quota

type Tracker struct {
	Limit   int
	Updated bool
}

func SafeCheck(q *Tracker) bool {
	return q != nil && q.Limit > 0
}

func UpdateAudit(q *Tracker) error {
	q.Updated = true
	return nil
}

func NeverCalled(x int) int {
	return x + 1
}
`
	prodFile := filepath.Join(pkgDir, "quota.go")
	if err := os.WriteFile(prodFile, []byte(prodSrc), 0o600); err != nil {
		t.Fatalf("failed to write quota.go: %v", err)
	}

	testSrc := `package quota

import "testing"

func TestSafeCheck(t *testing.T) {
	if SafeCheck(nil) {
		t.Fatal("expected false for nil tracker")
	}
	if SafeCheck(&Tracker{Limit: 0}) {
		t.Fatal("expected false for zero limit")
	}
	if !SafeCheck(&Tracker{Limit: 1}) {
		t.Fatal("expected true for positive limit")
	}
}

func TestUpdateAuditWeakAssertion(t *testing.T) {
	tr := &Tracker{}
	if err := UpdateAudit(tr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Intentionally forgets to check tr.Updated!
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "quota_test.go"), []byte(testSrc), 0o600); err != nil {
		t.Fatalf("failed to write quota_test.go: %v", err)
	}

	report, err := ExecuteMutationRun(context.Background(), &RunnerOptions{
		RepoRoot:  repoDir,
		BaseLabel: "main",
		ChangedLinesByFile: map[string]map[int]struct{}{
			"pkg/quota/quota.go": {
				9:  {}, // return q != nil && q.Limit > 0
				13: {}, // q.Updated = true
				18: {}, // return x + 1
			},
		},
		Workers:    2,
		MaxMutants: 50,
		Timeout:    time.Minute,
	})
	if err != nil {
		t.Fatalf("ExecuteMutationRun() unexpected error: %v", err)
	}

	// Verify working tree file was not modified by overlay execution.
	afterBytes, err := os.ReadFile(prodFile)
	if err != nil {
		t.Fatalf("failed reading quota.go after run: %v", err)
	}
	if string(afterBytes) != prodSrc {
		t.Errorf("working tree file was mutated on disk")
	}

	if report.TotalKilled != 4 {
		t.Errorf("TotalKilled = %d, want 4 (LCR &&->||, ROR !=->==, ROR >->>= and >->=0)", report.TotalKilled)
	}
	if report.TotalSurvived != 1 {
		t.Errorf("TotalSurvived = %d, want 1 (SBR on q.Updated = true)", report.TotalSurvived)
	}
	if report.TotalUncovered != 1 {
		t.Errorf("TotalUncovered = %d, want 1 (NeverCalled x + 1)", report.TotalUncovered)
	}

	surviving := report.SurvivingMutants()
	if len(surviving) != 1 || !strings.Contains(surviving[0].OriginalLine, "q.Updated = true") {
		t.Errorf("unexpected surviving mutants: %+v", surviving)
	}
}
