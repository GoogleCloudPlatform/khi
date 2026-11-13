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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
	"github.com/google/go-cmp/cmp"
)

func TestRunTSTestCoverage(t *testing.T) {
	t.Run("skips execution when SkipKarma is true", func(t *testing.T) {
		fakeBin := writeExecutableScript(t, "npx", "#!/bin/sh\nexit 1\n")
		t.Setenv("USE_CONTAINER", "false")
		t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

		err := RunTSTestCoverage(context.Background(), KarmaCoverageRequest{
			RepoRoot:    t.TempDir(),
			CoverageDir: t.TempDir(),
			Plan:        gotestutil.TSTestPlan{SkipKarma: true},
		})
		if err != nil {
			t.Fatalf("RunTSTestCoverage(SkipKarma) error = %v", err)
		}
	})

	t.Run("runs host npx in web directory with port and coverage env", func(t *testing.T) {
		repoRoot, covDir, recordFile := setupHostKarmaTestDirs(t)
		script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$PWD\" \"$KARMA_PORT\" \"$KARMA_COVERAGE_DIR\" \"$@\" > %q\n", recordFile)
		fakeBin := writeExecutableScript(t, "npx", script)
		t.Setenv("USE_CONTAINER", "false")
		t.Setenv("KARMA_PORT", "")
		t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

		err := RunTSTestCoverage(context.Background(), KarmaCoverageRequest{
			RepoRoot:    repoRoot,
			CoverageDir: covDir,
			Plan:        gotestutil.TSTestPlan{IncludeSpecs: []string{"src/app/a.spec.ts", "src/app/b.spec.ts"}},
		})
		if err != nil {
			t.Fatalf("RunTSTestCoverage(host) error = %v", err)
		}
		got := readRecordedLines(t, recordFile)
		if len(got) < 2 || got[1] == "" {
			t.Fatalf("expected non-empty allocated KARMA_PORT in recorded lines: %v", got)
		}
		want := []string{
			filepath.Join(repoRoot, "web"),
			got[1],
			covDir,
			"ng", "test", "--code-coverage", "--browsers", "ChromeHeadlessNoSandbox",
			"--watch=false", "--progress=false",
			"--include=src/app/a.spec.ts", "--include=src/app/b.spec.ts",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("host npx invocation mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("runs container command with mounted workspace and coverage dir", func(t *testing.T) {
		repoRoot, covDir, recordFile := setupHostKarmaTestDirs(t)
		dockerScript := filepath.Join(writeExecutableScript(t, "fake-docker", fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$PWD\" \"$@\" > %q\n", recordFile)), "fake-docker")
		t.Setenv("USE_CONTAINER", "true")
		t.Setenv("CONTAINER_CMD", dockerScript)
		t.Setenv("BUILDER_IMAGE", "fake-builder:v1")
		t.Setenv("KARMA_PORT", "19877")

		err := RunTSTestCoverage(context.Background(), KarmaCoverageRequest{
			RepoRoot:    repoRoot,
			CoverageDir: covDir,
			Plan:        gotestutil.TSTestPlan{IncludeSpecs: []string{"src/app/c.spec.ts"}},
		})
		if err != nil {
			t.Fatalf("RunTSTestCoverage(container) error = %v", err)
		}
		want := expectedContainerCoverageInvocation(repoRoot, covDir, "19877", "fake-builder:v1", "src/app/c.spec.ts")
		if diff := cmp.Diff(want, readRecordedLines(t, recordFile)); diff != "" {
			t.Errorf("container invocation mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("propagates container command resolution error", func(t *testing.T) {
		t.Setenv("USE_CONTAINER", "true")
		t.Setenv("CONTAINER_CMD", "")
		t.Setenv("PATH", t.TempDir())

		err := RunTSTestCoverage(context.Background(), KarmaCoverageRequest{
			RepoRoot:    t.TempDir(),
			CoverageDir: t.TempDir(),
		})
		if err == nil || !strings.Contains(err.Error(), "neither docker nor podman was found") {
			t.Fatalf("RunTSTestCoverage(missing container runtime) = %v, want missing docker/podman error", err)
		}
	})

	t.Run("copies combined stdout and stderr to ErrOut on failure", func(t *testing.T) {
		repoRoot, covDir, _ := setupHostKarmaTestDirs(t)
		fakeBin := writeExecutableScript(t, "npx", "#!/bin/sh\necho OUT_LINE\necho ERR_LINE >&2\nexit 1\n")
		t.Setenv("USE_CONTAINER", "false")
		t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

		var errOut bytes.Buffer
		err := RunTSTestCoverage(context.Background(), KarmaCoverageRequest{
			RepoRoot:    repoRoot,
			CoverageDir: covDir,
			ErrOut:      &errOut,
		})
		if err == nil || !strings.Contains(err.Error(), "ng test failed (npx ng test --code-coverage") {
			t.Fatalf("RunTSTestCoverage(failing npx) error = %v", err)
		}
		if diff := cmp.Diff("OUT_LINE\nERR_LINE\n", errOut.String()); diff != "" {
			t.Errorf("ErrOut mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestResolveTSHelperScriptPath(t *testing.T) {
	tmpWithLocal := t.TempDir()
	localScript := filepath.Join(tmpWithLocal, "scripts", "codequality", "internal", "tscheck", "js", "custom_helper.js")
	writeTSTestFile(t, localScript, "// local helper\n")

	_, callerFile, ok := callerFilePath()
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	callerMutateScript := filepath.Join(filepath.Dir(callerFile), "js", "mutate_ts.js")
	emptyRepo := t.TempDir()

	t.Run("prefers repo-relative helper script when present", func(t *testing.T) {
		got, err := resolveTSHelperScriptPath(tmpWithLocal, "custom_helper.js")
		if err != nil || got != localScript {
			t.Errorf("resolveTSHelperScriptPath(repo) = (%q, %v), want (%q, nil)", got, err, localScript)
		}
	})

	t.Run("falls back to caller-relative js directory", func(t *testing.T) {
		got, err := resolveTSHelperScriptPath(emptyRepo, "mutate_ts.js")
		if err != nil || got != callerMutateScript {
			t.Errorf("resolveTSHelperScriptPath(caller) = (%q, %v), want (%q, nil)", got, err, callerMutateScript)
		}
	})

	t.Run("returns error when helper script does not exist", func(t *testing.T) {
		got, err := resolveTSHelperScriptPath(emptyRepo, "nonexistent.js")
		if err == nil || got != "" || !strings.Contains(err.Error(), "nonexistent.js helper script not found under "+emptyRepo) {
			t.Errorf("resolveTSHelperScriptPath(missing) = (%q, %v)", got, err)
		}
	})
}

func TestLoadOrRunTSCoverageBranches(t *testing.T) {
	realRepoRoot := mustFindRepoRootForTSTest(t)
	if _, err := os.Stat(filepath.Join(realRepoRoot, "web", "node_modules", "typescript")); err != nil {
		t.Skipf("skipping TS coverage loader test: web/node_modules not installed (%v)", err)
	}

	tmpRepo := setupSymlinkedWebNodeModules(t, realRepoRoot)
	unimportedRel := "web/src/app/sample/unimported.ts"
	writeTSTestFile(t, filepath.Join(tmpRepo, filepath.FromSlash(unimportedRel)), "export const val = 1;\n")

	t.Run("empty changed files without Full leaves hasTSCov false", func(t *testing.T) {
		cov, hasCov, err := loadOrRunTSCoverage(context.Background(), Options{
			RepoRoot: tmpRepo,
		})
		if err != nil || hasCov || len(cov) != 0 {
			t.Errorf("loadOrRunTSCoverage(empty) = (len=%d, %v, %v), want (0, false, nil)", len(cov), hasCov, err)
		}
	})

	t.Run("missing TSCoveragePath leaves hasTSCov false without supplementing", func(t *testing.T) {
		cov, hasCov, err := loadOrRunTSCoverage(context.Background(), Options{
			RepoRoot:           tmpRepo,
			TSCoveragePath:     filepath.Join(tmpRepo, "missing.json"),
			ChangedLinesByFile: map[string]map[int]struct{}{unimportedRel: {1: {}}},
		})
		if err != nil || hasCov || len(cov) != 0 {
			t.Errorf("loadOrRunTSCoverage(missing) = (len=%d, %v, %v), want (0, false, nil)", len(cov), hasCov, err)
		}
	})

	t.Run("directory TSCoveragePath leaves hasTSCov false", func(t *testing.T) {
		cov, hasCov, err := loadOrRunTSCoverage(context.Background(), Options{
			RepoRoot:           tmpRepo,
			TSCoveragePath:     t.TempDir(),
			ChangedLinesByFile: map[string]map[int]struct{}{unimportedRel: {1: {}}},
		})
		if err != nil || hasCov || len(cov) != 0 {
			t.Errorf("loadOrRunTSCoverage(dir) = (len=%d, %v, %v), want (0, false, nil)", len(cov), hasCov, err)
		}
	})

	t.Run("existing TSCoveragePath sets hasTSCov true and supplements unimported TS file", func(t *testing.T) {
		covFile := filepath.Join(tmpRepo, "empty-cov.json")
		writeTSTestFile(t, covFile, "{}\n")
		cov, hasCov, err := loadOrRunTSCoverage(context.Background(), Options{
			RepoRoot:           tmpRepo,
			TSCoveragePath:     covFile,
			ChangedLinesByFile: map[string]map[int]struct{}{unimportedRel: {1: {}}},
		})
		if err != nil || !hasCov || len(cov) != 1 {
			t.Errorf("loadOrRunTSCoverage(existing) = (len=%d, %v, %v), want (1, true, nil)", len(cov), hasCov, err)
		}
	})

	t.Run("fresh execution skips Karma when changed file has no matching specs", func(t *testing.T) {
		cov, hasCov, err := loadOrRunTSCoverage(context.Background(), Options{
			RepoRoot:           tmpRepo,
			ChangedLinesByFile: map[string]map[int]struct{}{unimportedRel: {1: {}}},
			RunKarmaCoverage: func(_ context.Context, _ KarmaCoverageRequest) error {
				t.Fatal("unexpected RunKarmaCoverage call when SkipKarma is true")
				return nil
			},
		})
		if err != nil || !hasCov || len(cov) != 1 {
			t.Errorf("loadOrRunTSCoverage(skip Karma) = (len=%d, %v, %v), want (1, true, nil)", len(cov), hasCov, err)
		}
	})

	t.Run("fresh full execution uses custom RunKarmaCoverage callback when provided", func(t *testing.T) {
		covJSON := fmt.Sprintf(`{%q: {"path": %q, "statementMap": {"0": {"start": {"line": 1, "column": 0}, "end": {"line": 1, "column": 21}}}, "fnMap": {}, "branchMap": {}, "s": {"0": 7}, "f": {}, "b": {}}}`,
			unimportedRel, unimportedRel)
		t.Setenv("USE_CONTAINER", "false")
		t.Setenv("PATH", t.TempDir())

		cov, hasCov, err := loadOrRunTSCoverage(context.Background(), Options{
			RepoRoot: tmpRepo,
			Full:     true,
			RunKarmaCoverage: func(_ context.Context, req KarmaCoverageRequest) error {
				return os.WriteFile(filepath.Join(req.CoverageDir, "coverage-final.json"), []byte(covJSON), 0o600)
			},
		})
		if err != nil || !hasCov || cov[unimportedRel].Statements[0].Count != 7 {
			t.Errorf("loadOrRunTSCoverage(custom callback) = (%+v, %v, %v)", cov, hasCov, err)
		}
	})

	t.Run("fresh full execution invokes default RunTSTestCoverage and loads generated JSON", func(t *testing.T) {
		covJSON := fmt.Sprintf(`{%q: {"path": %q, "statementMap": {"0": {"start": {"line": 1, "column": 0}, "end": {"line": 1, "column": 21}}}, "fnMap": {}, "branchMap": {}, "s": {"0": 3}, "f": {}, "b": {}}}`,
			unimportedRel, unimportedRel)
		script := fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s' > \"$KARMA_COVERAGE_DIR/coverage-final.json\"\n", covJSON)
		fakeBin := writeExecutableScript(t, "npx", script)
		t.Setenv("USE_CONTAINER", "false")
		t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

		cov, hasCov, err := loadOrRunTSCoverage(context.Background(), Options{
			RepoRoot: tmpRepo,
			Full:     true,
		})
		if err != nil || !hasCov || len(cov[unimportedRel].Statements) != 1 {
			t.Errorf("loadOrRunTSCoverage(full default npx) = (%+v, %v, %v)", cov, hasCov, err)
		}
	})
}

func TestSupplementMissingTSCoverage(t *testing.T) {
	realRepoRoot := mustFindRepoRootForTSTest(t)
	if _, err := os.Stat(filepath.Join(realRepoRoot, "web", "node_modules", "typescript")); err != nil {
		t.Skipf("skipping zero-hit integration test: web/node_modules not installed (%v)", err)
	}

	tmpRepo := setupSymlinkedWebNodeModules(t, realRepoRoot)
	typesOnlyRel := "web/src/app/sample/types.ts"
	untestedImplRel := "web/src/app/sample/untested.ts"
	alreadyCoveredRel := "web/src/app/sample/already-covered.ts"

	writeTSTestFile(t, filepath.Join(tmpRepo, filepath.FromSlash(typesOnlyRel)), "export interface OnlyTypes { readonly id: string; }\n")
	writeTSTestFile(t, filepath.Join(tmpRepo, filepath.FromSlash(untestedImplRel)), "export function computeScore(v: number): number {\n  if (v < 0) {\n    return 0;\n  }\n  return v * 2;\n}\n")
	writeTSTestFile(t, filepath.Join(tmpRepo, filepath.FromSlash(alreadyCoveredRel)), "export const hit = 1;\n")

	covByFile := map[string]gotestutil.IstanbulFileCoverage{
		alreadyCoveredRel: {
			File:       alreadyCoveredRel,
			Statements: []gotestutil.IstanbulSpan{{StartLine: 1, StartCol: 13, EndLine: 1, EndCol: 21, Count: 5}},
		},
	}

	if err := SupplementMissingTSCoverage(context.Background(), tmpRepo, []string{alreadyCoveredRel, "web/karma.conf.ts"}, covByFile); err != nil {
		t.Fatalf("SupplementMissingTSCoverage(none missing) error: %v", err)
	}

	if err := SupplementMissingTSCoverage(context.Background(), tmpRepo, []string{typesOnlyRel, untestedImplRel, alreadyCoveredRel, "web/src/app/sample/deleted.ts"}, covByFile); err != nil {
		t.Fatalf("SupplementMissingTSCoverage() error: %v", err)
	}
	if _, hasTypes := covByFile[typesOnlyRel]; hasTypes {
		t.Errorf("covByFile unexpectedly contained pure type-only file %s", typesOnlyRel)
	}
	if _, hasImpl := covByFile[untestedImplRel]; !hasImpl {
		t.Fatalf("covByFile missing unimported implementation file %s", untestedImplRel)
	}
}

func setupHostKarmaTestDirs(t *testing.T) (string, string, string) {
	t.Helper()
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "web"), 0o755); err != nil {
		t.Fatalf("MkdirAll(web) failed: %v", err)
	}
	covDir := t.TempDir()
	recordFile := filepath.Join(t.TempDir(), "recorded.txt")
	return repoRoot, covDir, recordFile
}

func writeExecutableScript(t *testing.T, name string, content string) string {
	t.Helper()
	binDir := t.TempDir()
	scriptPath := filepath.Join(binDir, name)
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%s) failed: %v", scriptPath, err)
	}
	return binDir
}

func readRecordedLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) failed: %v", path, err)
	}
	trimmed := strings.TrimSuffix(string(data), "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func expectedContainerCoverageInvocation(repoRoot string, covDir string, port string, image string, spec string) []string {
	return []string{
		repoRoot,
		"run", "--rm",
		"--shm-size=1g",
		fmt.Sprintf("-u=%d:%d", os.Getuid(), os.Getgid()),
		fmt.Sprintf("-v=%s:/workspace", repoRoot),
		fmt.Sprintf("-v=%s:%s", covDir, covDir),
		"-w=/workspace/web",
		"-e=HOME=/tmp",
		"-e=KARMA_PORT=" + port,
		"-e=KARMA_COVERAGE_DIR=" + covDir,
		image,
		"npx",
		"ng", "test",
		"--code-coverage",
		"--browsers", "ChromeHeadlessNoSandbox",
		"--watch=false",
		"--progress=false",
		"--include=" + spec,
	}
}

func mustFindRepoRootForTSTest(t *testing.T) string {
	t.Helper()
	_, callerFile, ok := callerFilePath()
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(callerFile), "..", "..", "..", ".."))
}

func callerFilePath() (uintptr, string, bool) {
	pc, file, _, ok := runtime.Caller(1)
	return pc, file, ok
}

func setupSymlinkedWebNodeModules(t *testing.T, realRepoRoot string) string {
	t.Helper()
	tmpRepo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpRepo, "web"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	for _, item := range []string{"package.json", "node_modules"} {
		if err := os.Symlink(filepath.Join(realRepoRoot, "web", item), filepath.Join(tmpRepo, "web", item)); err != nil {
			t.Fatalf("Symlink(%s) failed: %v", item, err)
		}
	}
	return tmpRepo
}

func writeTSTestFile(t *testing.T, absPath string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
