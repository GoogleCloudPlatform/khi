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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type stubClosableEmbedder struct {
	closed atomic.Int32
}

func (s *stubClosableEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{0.6, 0.8}
	}
	return out, nil
}

func (s *stubClosableEmbedder) Close() error {
	s.closed.Add(1)
	return nil
}

type stubNonCloserEmbedder struct{}

func (s *stubNonCloserEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{0.6, 0.8}
	}
	return out, nil
}

func TestSimilaritySessionOptInAndMultiLanguageIntegration(t *testing.T) {
	realRepoRoot, err := gotestutil.ResolveRepoRoot("")
	if err != nil {
		t.Fatalf("ResolveRepoRoot failed: %v", err)
	}

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init", "-b", "main")
	writeFiles(t, repoDir, map[string]string{
		"go.mod": "module example.com/simmod\n\ngo 1.27\n",
		"pkg/common/dedup.go": `package common

func DedupItems(arr []string) []string {
	seen := map[string]bool{}
	for _, v := range arr {
		seen[v] = true
	}
	var out []string
	for k := range seen {
		out = append(out, k)
	}
	return out
}
`,
		"web/package.json": `{"name":"sim-web"}`,
		"web/src/app/utils/byte-format.ts": `export function formatBytes(bytes: number): string {
  if (bytes <= 0 || !Number.isFinite(bytes)) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB'];
  const idx = Math.min(Math.floor(Math.log2(bytes) / 10), units.length - 1);
  return (bytes / Math.pow(1024, idx)).toFixed(1) + ' ' + units[idx];
}
`,
	})
	runGitCmd(t, repoDir, "add", ".")
	runGitCmd(t, repoDir, "commit", "-m", "base")

	writeFiles(t, repoDir, map[string]string{
		"pkg/feature/feature.go": `package feature

func dedupNames(names []string) []string {
	seen := map[string]bool{}
	for _, v := range names {
		seen[v] = true
	}
	var out []string
	for k := range seen {
		out = append(out, k)
	}
	return out
}
`,
		"web/src/app/timeline/size.ts": `export function renderSizeLabel(sizeBytes: number): string {
  if (sizeBytes <= 0 || !Number.isFinite(sizeBytes)) {
    return '0 B';
  }
  const suffixes = ['B', 'KB', 'MB', 'GB'];
  const idx = Math.min(Math.floor(Math.log2(sizeBytes) / 10), suffixes.length - 1);
  return (sizeBytes / Math.pow(1024, idx)).toFixed(1) + ' ' + suffixes[idx];
}
`,
	})
	runGitCmd(t, repoDir, "add", ".")
	runGitCmd(t, repoDir, "commit", "-m", "add similar helpers")

	emb := &stubClosableEmbedder{}
	var factoryCalls atomic.Int32
	summaryPath := filepath.Join(t.TempDir(), "sim_summary.md")

	var defaultOffOut, stderr bytes.Buffer
	code, err := runCLI(context.Background(), cliEnv{
		args: []string{
			"--repo-root=" + repoDir,
			"--vcs=git",
			"--base=HEAD~1",
			"--fast",
			"--color=never",
		},
		stdout: &defaultOffOut,
		stderr: &stderr,
		getenv: func(string) string { return "" },
		embedFactory: func(context.Context, simcheck.EmbedderOptions) (simcheck.Embedder, string, error) {
			factoryCalls.Add(1)
			return emb, "", nil
		},
	})
	if err != nil || code != 0 {
		t.Fatalf("runCLI(default opt-out) = (%d, %v), stderr=%s", code, err, stderr.String())
	}
	if factoryCalls.Load() != 0 || strings.Contains(defaultOffOut.String(), "Similarity:") {
		t.Errorf("default run invoked similarity unexpectedly: calls=%d, out:\n%s", factoryCalls.Load(), defaultOffOut.String())
	}

	var gotGoProject string
	var stdout bytes.Buffer
	code, err = runCLI(context.Background(), cliEnv{
		args: []string{
			"--repo-root=" + repoDir,
			"--vcs=git",
			"--base=HEAD~1",
			"--fast",
			"--color=never",
			"--summary-file=" + summaryPath,
		},
		stdout: &stdout,
		stderr: &stderr,
		getenv: func(k string) string {
			if k == "CODEQUALITY_SIMILARITY" {
				return " true "
			}
			if k == "CODEQUALITY_SIMILARITY_PROJECT" {
				return " env-sim-project "
			}
			return ""
		},
		embedFactory: func(_ context.Context, opts simcheck.EmbedderOptions) (simcheck.Embedder, string, error) {
			factoryCalls.Add(1)
			gotGoProject = opts.ProjectID
			return emb, "", nil
		},
	})
	if err != nil || code != 0 {
		t.Fatalf("runCLI(CODEQUALITY_SIMILARITY=true) = (%d, %v), stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Similar Existing Go Functions") || !strings.Contains(stdout.String(), "DedupItems") {
		t.Errorf("stdout missing Go similarity advisory:\n%s", stdout.String())
	}
	if gotGoProject != "env-sim-project" {
		t.Errorf("gotGoProject = %q, want env-sim-project", gotGoProject)
	}
	if factoryCalls.Load() != 1 || emb.closed.Load() != 1 {
		t.Errorf("factoryCalls=%d, closed=%d; want 1 each", factoryCalls.Load(), emb.closed.Load())
	}

	var gotGoOverrideProject string
	var goOverrideOut bytes.Buffer
	code, err = runCLI(context.Background(), cliEnv{
		args: []string{
			"--repo-root=" + repoDir,
			"--vcs=git",
			"--base=HEAD~1",
			"--fast",
			"--similarity-project=cli-override-project",
			"--color=never",
		},
		stdout: &goOverrideOut,
		stderr: &stderr,
		getenv: func(k string) string {
			if k == "CODEQUALITY_SIMILARITY" {
				return "1"
			}
			if k == "CODEQUALITY_SIMILARITY_PROJECT" {
				return "env-sim-project"
			}
			return ""
		},
		embedFactory: func(_ context.Context, opts simcheck.EmbedderOptions) (simcheck.Embedder, string, error) {
			gotGoOverrideProject = opts.ProjectID
			return &stubNonCloserEmbedder{}, "", nil
		},
	})
	if err != nil || code != 0 {
		t.Fatalf("runCLI(Go CLI project override) = (%d, %v), stderr=%s", code, err, stderr.String())
	}
	if gotGoOverrideProject != "cli-override-project" {
		t.Errorf("gotGoOverrideProject = %q, want cli-override-project", gotGoOverrideProject)
	}
	if !strings.Contains(goOverrideOut.String(), "Similar Existing Go Functions") || !strings.Contains(goOverrideOut.String(), "DedupItems") {
		t.Errorf("go override stdout missing Go similarity advisory:\n%s", goOverrideOut.String())
	}

	t.Run("TypeScript", func(t *testing.T) {
		verifySimilarityTypeScriptCLI(t, realRepoRoot, repoDir)
	})

	var nilFactoryOut bytes.Buffer
	code, err = runCLI(context.Background(), cliEnv{
		args: []string{
			"--repo-root=" + repoDir,
			"--vcs=git",
			"--base=HEAD~1",
			"--fast",
			"--similarity=true",
			"--color=never",
		},
		stdout: &nilFactoryOut,
		stderr: &stderr,
		getenv: func(string) string { return "" },
	})
	if err != nil || code != 0 || !strings.Contains(nilFactoryOut.String(), "SKIPPED (embedding client not configured)") {
		t.Errorf("runCLI(nil factory) = (%d, %v), out:\n%s", code, err, nilFactoryOut.String())
	}

	var disabledOut bytes.Buffer
	code, err = runCLI(context.Background(), cliEnv{
		args: []string{
			"--repo-root=" + repoDir,
			"--vcs=git",
			"--base=HEAD~1",
			"--fast",
			"--similarity=false",
			"--color=never",
		},
		stdout: &disabledOut,
		stderr: &stderr,
		getenv: func(k string) string {
			if k == "CODEQUALITY_SIMILARITY" {
				return "true"
			}
			return ""
		},
		embedFactory: func(context.Context, simcheck.EmbedderOptions) (simcheck.Embedder, string, error) {
			return emb, "", nil
		},
	})
	if err != nil || code != 0 || strings.Contains(disabledOut.String(), "Similarity:") {
		t.Errorf("runCLI(--similarity=false) = (%d, %v), out:\n%s", code, err, disabledOut.String())
	}
}

func TestSimilaritySessionIdempotentCloseAndNilSafety(t *testing.T) {
	if parseSimilarityEnv(nil) {
		t.Errorf("parseSimilarityEnv(nil) = true, want false")
	}
	if gotProj := parseSimilarityProjectEnv(nil); gotProj != "" {
		t.Errorf("parseSimilarityProjectEnv(nil) = %q, want empty", gotProj)
	}

	ctx := context.Background()
	var nilSess *similaritySession
	if err := nilSess.Close(); err != nil {
		t.Errorf("nilSession.Close() = %v, want nil", err)
	}

	emb := &stubClosableEmbedder{}
	sess := newSimilaritySession(ctx, cliFlags{similarity: true}, func(context.Context, simcheck.EmbedderOptions) (simcheck.Embedder, string, error) {
		return emb, "", nil
	}, nil)
	if _, err := sess.resolve(); err != nil {
		t.Fatalf("sess.resolve() failed: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Fatalf("first sess.Close() failed: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Fatalf("second sess.Close() failed: %v", err)
	}
	if emb.closed.Load() != 1 {
		t.Errorf("emb.closed = %d, want 1 after repeated Close()", emb.closed.Load())
	}

	emptySum, err := sess.evaluateTS(ctx, t.TempDir(), gotestutil.DiffResult{})
	if err != nil || emptySum.Enabled {
		t.Errorf("evaluateTS(empty diff) = (%+v, %v), want disabled", emptySum, err)
	}
}

func verifySimilarityTypeScriptCLI(t *testing.T, realRepoRoot string, repoDir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(realRepoRoot, "web", "node_modules", "typescript")); err != nil {
		t.Skipf("skipping TypeScript similarity CLI integration check: web/node_modules/typescript not installed (%v)", err)
	}
	if err := os.Symlink(filepath.Join(realRepoRoot, "web", "node_modules"), filepath.Join(repoDir, "web", "node_modules")); err != nil {
		t.Fatalf("Symlink node_modules failed: %v", err)
	}

	tsCovPath := filepath.Join(t.TempDir(), "coverage-final.json")
	if err := os.WriteFile(tsCovPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("WriteFile(tsCovPath) failed: %v", err)
	}

	var gotTSProject string
	var tsOut, stderr bytes.Buffer
	code, err := runCLI(context.Background(), cliEnv{
		args: []string{
			"--repo-root=" + repoDir,
			"--lang=ts",
			"--ts-coverage=" + tsCovPath,
			"--similarity-project=cli-override-project",
			"--vcs=git",
			"--base=HEAD~1",
			"--color=never",
		},
		stdout: &tsOut,
		stderr: &stderr,
		getenv: func(k string) string {
			if k == "CODEQUALITY_SIMILARITY" {
				return "1"
			}
			if k == "CODEQUALITY_SIMILARITY_PROJECT" {
				return "env-sim-project"
			}
			return ""
		},
		embedFactory: func(_ context.Context, opts simcheck.EmbedderOptions) (simcheck.Embedder, string, error) {
			gotTSProject = opts.ProjectID
			return &stubNonCloserEmbedder{}, "", nil
		},
	})
	if err != nil || code != 0 {
		t.Fatalf("runCLI(--lang=ts) = (%d, %v), stderr=%s", code, err, stderr.String())
	}
	if gotTSProject != "cli-override-project" {
		t.Errorf("gotTSProject = %q, want cli-override-project", gotTSProject)
	}
	if !strings.Contains(tsOut.String(), "Similar Existing TypeScript Functions") || !strings.Contains(tsOut.String(), "formatBytes") {
		t.Errorf("ts stdout missing TypeScript similarity advisory:\n%s", tsOut.String())
	}
}
