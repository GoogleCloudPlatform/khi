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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func TestCompileBisectFallbackAndEnv(t *testing.T) {
	repoDir := t.TempDir()
	workDir := t.TempDir()
	pkgDir := filepath.Join(repoDir, "calc")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	src := `package calc

func Add(a, b int) int {
	return a + b
}
`
	testSrc := `package calc

import "testing"

func TestAdd(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Fatal("bad")
	}
}
`
	for rel, content := range map[string]string{
		"go.mod": "module example.com/bisectmod\n\ngo 1.24\n",
		"root.go": `package root
func R() int { return 1 }
`,
		"root_test.go": `package root
import "testing"
func TestR(t *testing.T) { _ = R() }
`,
		"calc/calc.go":      src,
		"calc/calc_test.go": testSrc,
	} {
		if err := os.WriteFile(filepath.Join(repoDir, filepath.FromSlash(rel)), []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	// Empty site list returns ("", nil) immediately.
	if emptyBin, err := compilePackageWithFallback(context.Background(), compileRequest{repoRoot: repoDir, pkgDir: "calc"}); err != nil || emptyBin != "" {
		t.Fatalf("compilePackageWithFallback(empty) = (%q, %v), want empty nil", emptyBin, err)
	}

	// Root package "." compiles cleanly via runGoTestCompile.
	if err := runGoTestCompile(context.Background(), repoDir, "", filepath.Join(workDir, "root.test"), "."); err != nil {
		t.Fatalf("runGoTestCompile(.) failed: %v", err)
	}

	plan, err := PlanFileMutations("calc/calc.go", []byte(src), PlanOptions{TargetLines: lineRangeSet(1, 10)})
	if err != nil {
		t.Fatalf("PlanFileMutations failed: %v", err)
	}
	bogusMut := &Mutant{ID: 2, File: "calc/calc.go", Status: rule.MutantPending}
	plan.Mutants = append(plan.Mutants, bogusMut)
	plan.Sites = append(plan.Sites, MutationSite{
		StartOffset: 0,
		EndOffset:   7,
		Kind:        siteBoolLit,
		Mutants:     []*Mutant{bogusMut},
	})

	var logBuf bytes.Buffer
	binPath, compErr := compilePackageWithFallback(context.Background(), compileRequest{
		repoRoot:    repoDir,
		pkgDir:      "calc",
		pkgPlans:    []*FilePlan{plan},
		maxMutantID: 2,
		pkgWorkDir:  workDir,
		logWriter:   &logBuf,
	})
	if compErr != nil || binPath == "" {
		t.Fatalf("compilePackageWithFallback() = (%q, %v)", binPath, compErr)
	}
	if !strings.Contains(logBuf.String(), "bisecting") {
		t.Errorf("expected bisect log message, got %q", logBuf.String())
	}
	if bogusMut.Status != rule.MutantCompileSkipped {
		t.Errorf("bogusMut.Status = %q, want %q", bogusMut.Status, rule.MutantCompileSkipped)
	}

	// Single broken site with nil logWriter returns ("", nil) without panicking.
	onlyBogusPlan := &FilePlan{
		File:        "calc/calc.go",
		PackageName: "calc",
		Source:      []byte(src),
		Sites:       []MutationSite{{StartOffset: 0, EndOffset: 7, Kind: siteBoolLit, Mutants: []*Mutant{{ID: 1, Status: rule.MutantPending}}}},
	}
	if nilLogBin, err := compilePackageWithFallback(context.Background(), compileRequest{
		repoRoot:    repoDir,
		pkgDir:      "calc",
		pkgPlans:    []*FilePlan{onlyBogusPlan},
		maxMutantID: 1,
		pkgWorkDir:  filepath.Join(workDir, "only_bogus"),
		logWriter:   nil,
	}); err != nil || nilLogBin != "" {
		t.Errorf("compilePackageWithFallback(only broken, nil logWriter) = (%q, %v), want empty nil", nilLogBin, err)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	attempts := 0
	if refs := bisectCompilableRefs(canceledCtx, compileRequest{}, []flatSiteRef{{}}, &attempts); refs != nil {
		t.Errorf("bisectCompilableRefs(canceled) = %v, want nil", refs)
	}

	byPkg, pkgs := groupFilesByPackage(map[string]map[int]struct{}{
		"empty.go":     nil,
		"calc/calc.go": {1: {}},
	})
	if len(pkgs) != 1 || len(byPkg["calc"]) != 1 {
		t.Errorf("groupFilesByPackage() = (%v, %v), want single calc entry", byPkg, pkgs)
	}

	env := buildMutantEnv([]string{"KHI_MUTANT_ID=99", "KHI_SKIP_CLOUD_LOGGING=true", "FOO=bar"}, 3, "/tmp/t.trace")
	if len(env) != 4 {
		t.Errorf("len(buildMutantEnv) = %d, want 4", len(env))
	}
	wantDefaultWorkers := max(runtime.NumCPU(), 1)
	if got := resolveWorkerCount(0); got != wantDefaultWorkers {
		t.Errorf("resolveWorkerCount(0) = %d, want %d", got, wantDefaultWorkers)
	}
	if got := resolveWorkerCount(-1); got != wantDefaultWorkers {
		t.Errorf("resolveWorkerCount(-1) = %d, want %d", got, wantDefaultWorkers)
	}
	if got := resolveWorkerCount(1); got != 1 {
		t.Errorf("resolveWorkerCount(1) = %d, want 1", got)
	}
	if got := resolveWorkerCount(9999); got != 9999 {
		t.Errorf("resolveWorkerCount(9999) = %d, want 9999", got)
	}
}
