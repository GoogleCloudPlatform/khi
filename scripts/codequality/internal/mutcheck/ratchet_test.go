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

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/overlay"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func TestRecordMutOutcome(t *testing.T) {
	fm := &astcheck.FileMetrics{File: "pkg/a/a.go"}
	fn := astcheck.FuncMetrics{Key: "Fn", DisplayName: "Fn", StartLine: 10, EndLine: 20}

	testCases := []struct {
		name      string
		sample    rule.Sample
		wantCount int
		wantKind  rule.ViolationKind
	}{
		{
			name:      "compliant sample records zero findings",
			sample:    rule.Sample{After: 0, Limit: 0},
			wantCount: 0,
		},
		{
			name:      "worsened legacy sample records ratchet regression",
			sample:    rule.Sample{After: 3, Limit: 0, Before: 2, HasBefore: true},
			wantCount: 1,
			wantKind:  rule.ViolationRatchetRegression,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := &Result{}
			recordMutOutcome(res, fm, fn, tc.sample, []rule.MutantDetail{{Line: 12, Operator: rule.OperatorROR}})
			if len(res.Findings) != tc.wantCount {
				t.Fatalf("len(Findings) = %d, want %d", len(res.Findings), tc.wantCount)
			}
			if tc.wantCount > 0 && res.Findings[0].Kind != tc.wantKind {
				t.Errorf("Kind = %q, want %q", res.Findings[0].Kind, tc.wantKind)
			}
		})
	}
}

func TestRatchetEdgeCases(t *testing.T) {
	repoDir := t.TempDir()
	workDir := t.TempDir()
	svcDir := filepath.Join(repoDir, "svc")
	if err := os.MkdirAll(svcDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	svcSrc := `package svc

func Empty() {}

func HasMutant(x int) bool {
	return x > 0
}
`
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module example.com/ratch\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(svcDir, "svc.go"), []byte(svcSrc), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Zero-mutant function returns (0, nil, nil).
	if n, det, err := countFullFuncSurvived(context.Background(), fullFuncRequest{
		repoRoot: repoDir,
		workDir:  workDir,
		relFile:  "svc/svc.go",
		fn:       astcheck.FuncMetrics{StartLine: 3, EndLine: 3},
	}); err != nil || n != 0 || det != nil {
		t.Errorf("countFullFuncSurvived(Empty) = (%d, %v, %v), want (0, nil, nil)", n, det, err)
	}

	// Package without _test.go returns len(plan.Mutants) when isBase is true, and 0 when false.
	if baseN, _, err := countFullFuncSurvived(context.Background(), fullFuncRequest{
		repoRoot: repoDir,
		workDir:  workDir,
		relFile:  "svc/svc.go",
		fn:       astcheck.FuncMetrics{StartLine: 5, EndLine: 7},
		isBase:   true,
	}); err != nil || baseN != 2 {
		t.Errorf("countFullFuncSurvived(isBase without tests) = (%d, %v), want (2, nil)", baseN, err)
	}
	if targN, _, err := countFullFuncSurvived(context.Background(), fullFuncRequest{
		repoRoot: repoDir,
		workDir:  workDir,
		relFile:  "svc/svc.go",
		fn:       astcheck.FuncMetrics{StartLine: 5, EndLine: 7},
		isBase:   false,
	}); err != nil || targN != 0 {
		t.Errorf("countFullFuncSurvived(target without tests) = (%d, %v), want (0, nil)", targN, err)
	}

	// Function returning a custom bool type with unary ! drops cleanly via bisect (binPath == "").
	supDir := filepath.Join(repoDir, "sup")
	if err := os.MkdirAll(supDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(sup) failed: %v", err)
	}
	supSrc := "package sup\n\ntype myBool bool\n\nfunc Sup(b myBool) myBool {\n\treturn !b\n}\n"
	if err := os.WriteFile(filepath.Join(supDir, "sup.go"), []byte(supSrc), 0o600); err != nil {
		t.Fatalf("WriteFile(sup.go) failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(supDir, "sup_test.go"), []byte("package sup\nimport \"testing\"\nfunc TestSup(t *testing.T) { _ = Sup(true) }\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(sup_test.go) failed: %v", err)
	}
	supFM, err := astcheck.AnalyzeFile("sup/sup.go", []byte(supSrc), 3)
	if err != nil {
		t.Fatalf("AnalyzeFile(sup.go) failed: %v", err)
	}
	if supN, supDet, err := countFullFuncSurvived(context.Background(), fullFuncRequest{
		repoRoot: repoDir,
		workDir:  filepath.Join(workDir, "supwork"),
		relFile:  "sup/sup.go",
		fn:       supFM.Funcs[0],
	}); err != nil || supN != 0 || supDet != nil {
		t.Errorf("countFullFuncSurvived(compile-skipped site) = (%d, %v, %v), want (0, nil, nil)", supN, supDet, err)
	}

	// Empty overlay snapshot returns 0 for missing BaseSource.
	snap, err := overlay.Build(overlay.Config{
		RepoRoot:     repoDir,
		WorkDir:      filepath.Join(workDir, "ov"),
		ReadBaseFile: func(string) ([]byte, bool, error) { return nil, false, nil },
	})
	if err != nil {
		t.Fatalf("overlay.Build failed: %v", err)
	}
	if got := countBaseFuncSurvived(context.Background(), Options{ASTResult: &astcheck.Result{}}, lazyBaseEnv{snap: snap}, 0, pendingBaseMut{baseFile: "svc/svc.go"}); got != 0 {
		t.Errorf("countBaseFuncSurvived(missing BaseSource) = %d, want 0", got)
	}

	// collectSurvivingMutantFindings skips nil FileMetrics and returns nil when pending is empty.
	astRes := &astcheck.Result{
		Files: map[string]*astcheck.FileMetrics{
			"svc/svc.go": {File: "svc/svc.go"},
		},
		TouchedFuncs: map[string][]astcheck.FuncMetrics{
			"svc/gen.go": {{Key: "Gen"}},
			"svc/svc.go": {{Key: "Empty"}},
		},
	}
	if err := collectSurvivingMutantFindings(context.Background(), Options{
		Thresholds: rule.DefaultThresholds(),
		ASTResult:  astRes,
	}, nil, 1, &Result{}); err != nil {
		t.Errorf("collectSurvivingMutantFindings(clean) error: %v", err)
	}

	// evaluateLazyBaseMutants falls back to item.survived when target has 0 tests (fullSurvived empty), and stops on canceled ctx.
	res := &Result{}
	item := pendingBaseMut{
		relFile:  "svc/svc.go",
		fm:       &astcheck.FileMetrics{File: "svc/svc.go"},
		fn:       astcheck.FuncMetrics{Key: "HasMutant", StartLine: 5, EndLine: 7},
		baseFile: "svc/svc.go",
		survived: []rule.MutantDetail{{Line: 6, Operator: rule.OperatorROR}},
	}
	if err := evaluateLazyBaseMutants(context.Background(), Options{
		RepoRoot:   repoDir,
		WorkDir:    workDir,
		Thresholds: rule.DefaultThresholds(),
		ASTResult:  astRes,
		EnsureBaseOverlay: func() (*overlay.Snapshot, error) {
			return snap, nil
		},
	}, []pendingBaseMut{item}, 1, res); err != nil {
		t.Errorf("evaluateLazyBaseMutants() error: %v", err)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := evaluateLazyBaseMutants(canceledCtx, Options{
		Thresholds:        rule.DefaultThresholds(),
		EnsureBaseOverlay: func() (*overlay.Snapshot, error) { return snap, nil },
	}, []pendingBaseMut{item}, 1, res); err != nil {
		t.Errorf("evaluateLazyBaseMutants(canceled) error: %v", err)
	}
}
