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

package tscheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/google/go-cmp/cmp"
)

func TestPrepareTSMutationWorkspaceAndApplyOutcomes(t *testing.T) {
	repoDir := t.TempDir()
	workDir := t.TempDir()
	writeTSTestFile(t, filepath.Join(repoDir, "web", "node_modules", ".keep"), "")
	writeTSTestFile(t, filepath.Join(repoDir, "web", "package.json"), "{}\n")
	writeTSTestFile(t, filepath.Join(repoDir, "web", "tsconfig.json"), "{}\n")
	origCalcSrc := "export const x = 1 + 2;\n"
	origOtherSrc := "export const other = 42;\n"
	writeTSTestFile(t, filepath.Join(repoDir, "web", "src", "app", "calc.ts"), origCalcSrc)
	writeTSTestFile(t, filepath.Join(repoDir, "web", "src", "app", "other.ts"), origOtherSrc)

	m1 := &TSMutant{ID: 1, File: "web/src/app/calc.ts", Status: rule.MutantPending}
	m2 := &TSMutant{ID: 2, File: "web/src/app/calc.ts", Status: rule.MutantPending}
	m3 := &TSMutant{ID: 3, File: "web/src/app/calc.ts", Status: rule.MutantPending}
	m4 := &TSMutant{ID: 4, File: "web/src/app/calc.ts", Status: rule.MutantPending}
	m5 := &TSMutant{ID: 5, File: "web/src/app/calc.ts", Status: rule.MutantPending}

	staged := []tsStagedFile{{
		plan: &TSFilePlan{File: "web/src/app/calc.ts", Source: []byte(origCalcSrc)},
		tsSites: []TSMutationSite{{
			Kind: "arith", Op: "+", StartOffset: 17, EndOffset: 22, LeftStart: 17, LeftEnd: 18, RightStart: 21, RightEnd: 22,
			Mutants: []*TSMutant{m1},
		}},
	}}

	resultsPath, cfgPath, err := prepareTSMutationWorkspace(karmaWorkspaceRequest{
		repoRoot:       repoDir,
		workDir:        workDir,
		stagedFiles:    staged,
		pendingMutants: []*TSMutant{m1, m2, m3, m4, m5},
		maxMutantID:    5,
		maxProbeID:     1,
		maxMutants:     80,
		overallTimeout: time.Minute,
	})
	if err != nil {
		t.Fatalf("prepareTSMutationWorkspace() error = %v", err)
	}

	assertStagedMutantWorkspace(t, repoDir, workDir, origCalcSrc, origOtherSrc)
	assertPluginConfigEquals(t, cfgPath, karmaPluginConfig{
		CandidateIDs: []int{1, 2, 3, 4, 5},
		FileByMutantID: map[string]string{
			"1": "web/src/app/calc.ts",
			"2": "web/src/app/calc.ts",
			"3": "web/src/app/calc.ts",
			"4": "web/src/app/calc.ts",
			"5": "web/src/app/calc.ts",
		},
		MaxMutants:            80,
		MinPerMutantTimeoutMs: 3000,
		OverallTimeoutMs:      30000,
		ResultsPath:           resultsPath,
	})

	// Short overall timeout clamps plugin budget floor to 15 seconds.
	shortWork := t.TempDir()
	shortResPath, shortCfgPath, err := writeKarmaPluginConfigFile(shortWork, karmaWorkspaceRequest{
		pendingMutants: []*TSMutant{m1},
		maxMutants:     10,
		overallTimeout: 35 * time.Second,
	})
	if err != nil {
		t.Fatalf("writeKarmaPluginConfigFile(short) error = %v", err)
	}
	assertPluginConfigEquals(t, shortCfgPath, karmaPluginConfig{
		CandidateIDs:          []int{1},
		FileByMutantID:        map[string]string{"1": "web/src/app/calc.ts"},
		MaxMutants:            10,
		MinPerMutantTimeoutMs: 3000,
		OverallTimeoutMs:      15000,
		ResultsPath:           shortResPath,
	})

	assertApplyKarmaOutcomes(t, []*TSMutant{m1, m2, m3, m4, m5})
}

func TestWriteMutatedWebFilesEdgeCases(t *testing.T) {
	staged := []tsStagedFile{{
		plan: &TSFilePlan{File: "web/src/app/new_file.ts", Source: []byte("export const a = true;\n")},
		tsSites: []TSMutationSite{{
			Kind: "bool_lit", BoolVal: true, StartOffset: 17, EndOffset: 21,
			Mutants: []*TSMutant{{ID: 1}},
		}},
	}}

	t.Run("writes mutated file when destination symlink does not exist", func(t *testing.T) {
		tmpWebDir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(tmpWebDir, "src", "app"), 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		if err := writeMutatedWebFiles(tmpWebDir, staged, 1, 0); err != nil {
			t.Fatalf("writeMutatedWebFiles() error = %v", err)
		}
		got, err := os.ReadFile(filepath.Join(tmpWebDir, "src", "app", "new_file.ts"))
		if err != nil || !strings.Contains(string(got), "__khiMut.flip(1, true)") {
			t.Errorf("written mutant file = (%q, %v)", string(got), err)
		}
	})

	t.Run("returns error when destination cannot be removed", func(t *testing.T) {
		tmpWebDir := t.TempDir()
		nonEmptyDir := filepath.Join(tmpWebDir, "src", "app", "new_file.ts", "nested")
		if err := os.MkdirAll(nonEmptyDir, 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		err := writeMutatedWebFiles(tmpWebDir, staged, 1, 0)
		if err == nil || !strings.Contains(err.Error(), "failed to remove symlink before writing mutant") {
			t.Fatalf("writeMutatedWebFiles(non-empty dir) = %v, want remove error", err)
		}
	})
}

func TestRunKarmaMutationSession(t *testing.T) {
	t.Run("runs host npx in staged web directory with config env", func(t *testing.T) {
		repoRoot, workDir, cfgPath, recordFile := setupKarmaSessionDirs(t)
		script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$PWD\" \"$KARMA_PORT\" \"$KHI_MUTATION_CONFIG_FILE\" \"$@\" > %q\n", recordFile)
		fakeBin := writeExecutableScript(t, "npx", script)
		t.Setenv("USE_CONTAINER", "false")
		t.Setenv("KARMA_PORT", "")
		t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

		err := runKarmaMutationSession(context.Background(), karmaSessionRequest{
			repoRoot:     repoRoot,
			workDir:      workDir,
			cfgPath:      cfgPath,
			includeSpecs: []string{"src/app/calc.spec.ts"},
		})
		if err != nil {
			t.Fatalf("runKarmaMutationSession(host) error = %v", err)
		}
		got := readRecordedLines(t, recordFile)
		if len(got) < 2 || got[1] == "" {
			t.Fatalf("expected non-empty allocated KARMA_PORT in recorded lines: %v", got)
		}
		want := []string{
			filepath.Join(workDir, "web"),
			got[1],
			cfgPath,
			"ng", "test", "--browsers=ChromeHeadlessNoSandbox", "--watch=false", "--progress=false",
			"--include=src/app/calc.spec.ts",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("host mutation session invocation mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("runs container command with read-only repo and writable work mounts", func(t *testing.T) {
		repoRoot, workDir, cfgPath, recordFile := setupKarmaSessionDirs(t)
		dockerScript := filepath.Join(writeExecutableScript(t, "fake-docker", fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$PWD\" \"$@\" > %q\n", recordFile)), "fake-docker")
		t.Setenv("USE_CONTAINER", "true")
		t.Setenv("CONTAINER_CMD", dockerScript)
		t.Setenv("BUILDER_IMAGE", "mut-builder:v2")
		t.Setenv("KARMA_PORT", "29877")

		err := runKarmaMutationSession(context.Background(), karmaSessionRequest{
			repoRoot:     repoRoot,
			workDir:      workDir,
			cfgPath:      cfgPath,
			includeSpecs: []string{"src/app/calc.spec.ts"},
		})
		if err != nil {
			t.Fatalf("runKarmaMutationSession(container) error = %v", err)
		}
		want := expectedContainerMutationInvocation(repoRoot, workDir, cfgPath, "29877", "mut-builder:v2", "src/app/calc.spec.ts")
		if diff := cmp.Diff(want, readRecordedLines(t, recordFile)); diff != "" {
			t.Errorf("container mutation invocation mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("propagates container runtime lookup error", func(t *testing.T) {
		repoRoot, workDir, cfgPath, _ := setupKarmaSessionDirs(t)
		t.Setenv("USE_CONTAINER", "true")
		t.Setenv("CONTAINER_CMD", "")
		t.Setenv("PATH", t.TempDir())

		err := runKarmaMutationSession(context.Background(), karmaSessionRequest{
			repoRoot: repoRoot,
			workDir:  workDir,
			cfgPath:  cfgPath,
		})
		if err == nil || !strings.Contains(err.Error(), "neither docker nor podman was found") {
			t.Fatalf("runKarmaMutationSession(missing runtime) = %v", err)
		}
	})

	t.Run("copies combined failure output to errOut", func(t *testing.T) {
		repoRoot, workDir, cfgPath, _ := setupKarmaSessionDirs(t)
		fakeBin := writeExecutableScript(t, "npx", "#!/bin/sh\necho MUT_STDOUT\necho MUT_STDERR >&2\nexit 1\n")
		t.Setenv("USE_CONTAINER", "false")
		t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

		var errBuf bytes.Buffer
		err := runKarmaMutationSession(context.Background(), karmaSessionRequest{
			repoRoot: repoRoot,
			workDir:  workDir,
			cfgPath:  cfgPath,
			errOut:   &errBuf,
		})
		if err == nil || !strings.Contains(err.Error(), "TypeScript Karma mutation run failed (npx ng test") {
			t.Fatalf("runKarmaMutationSession(failing npx) = %v", err)
		}
		if diff := cmp.Diff("MUT_STDOUT\nMUT_STDERR\n", errBuf.String()); diff != "" {
			t.Errorf("errOut mismatch (-want +got):\n%s", diff)
		}
	})
}

func assertStagedMutantWorkspace(t *testing.T, repoDir string, workDir string, origCalcSrc string, origOtherSrc string) {
	t.Helper()
	gotRepoCalc, err := os.ReadFile(filepath.Join(repoDir, "web", "src", "app", "calc.ts"))
	if err != nil || string(gotRepoCalc) != origCalcSrc {
		t.Errorf("original repo calc.ts modified = (%q, %v)", string(gotRepoCalc), err)
	}
	gotMutCalc, err := os.ReadFile(filepath.Join(workDir, "web", "src", "app", "calc.ts"))
	if err != nil || !strings.Contains(string(gotMutCalc), "__khiMut.addL(1,") {
		t.Errorf("staged mutant calc.ts = (%q, %v)", string(gotMutCalc), err)
	}
	gotSymlinkedOther, err := os.ReadFile(filepath.Join(workDir, "web", "src", "app", "other.ts"))
	if err != nil || string(gotSymlinkedOther) != origOtherSrc {
		t.Errorf("staged symlink other.ts = (%q, %v)", string(gotSymlinkedOther), err)
	}
}

func assertPluginConfigEquals(t *testing.T, cfgPath string, want karmaPluginConfig) {
	t.Helper()
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) failed: %v", cfgPath, err)
	}
	var got karmaPluginConfig
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal(%s) failed: %v", cfgPath, err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("karmaPluginConfig mismatch (-want +got):\n%s", diff)
	}
}

func assertApplyKarmaOutcomes(t *testing.T, mutants []*TSMutant) {
	t.Helper()
	applyKarmaOutcomes(mutants, map[string]string{
		"1": "KILLED",
		"2": "TIMED_OUT",
		"3": "SURVIVED",
		"4": "UNCOVERED",
	})
	var got []rule.MutantStatus
	for _, m := range mutants {
		got = append(got, m.Status)
	}
	want := []rule.MutantStatus{
		rule.MutantKilled,
		rule.MutantTimedOut,
		rule.MutantSurvived,
		rule.MutantUncovered,
		rule.MutantSkippedBudget,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("applyKarmaOutcomes() mismatch (-want +got):\n%s", diff)
	}
}

func setupKarmaSessionDirs(t *testing.T) (string, string, string, string) {
	t.Helper()
	repoRoot := t.TempDir()
	workDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workDir, "web"), 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir/web) failed: %v", err)
	}
	cfgPath := filepath.Join(workDir, "mutation-config.json")
	writeTSTestFile(t, cfgPath, "{}\n")
	recordFile := filepath.Join(t.TempDir(), "recorded.txt")
	return repoRoot, workDir, cfgPath, recordFile
}

func expectedContainerMutationInvocation(repoRoot string, workDir string, cfgPath string, port string, image string, spec string) []string {
	return []string{
		repoRoot,
		"run", "--rm",
		"--shm-size=1g",
		fmt.Sprintf("-u=%d:%d", os.Getuid(), os.Getgid()),
		fmt.Sprintf("-v=%s:%s:ro", repoRoot, repoRoot),
		fmt.Sprintf("-v=%s:%s", workDir, workDir),
		"-w=" + filepath.Join(workDir, "web"),
		"-e=HOME=/tmp",
		"-e=KARMA_PORT=" + port,
		"-e=KHI_MUTATION_CONFIG_FILE=" + cfgPath,
		image,
		"npx",
		"ng", "test",
		"--browsers=ChromeHeadlessNoSandbox",
		"--watch=false",
		"--progress=false",
		"--include=" + spec,
	}
}
