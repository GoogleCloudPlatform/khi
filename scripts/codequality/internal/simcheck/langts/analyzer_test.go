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

package langts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck"
)

func TestExtractTSUnitsAndFilterBoilerplate(t *testing.T) {
	analyzer := NewAnalyzer("")
	if analyzer.webDir != "" {
		t.Fatalf("NewAnalyzer(\"\").webDir = %q, want empty", analyzer.webDir)
	}
	if gotRoot := NewAnalyzer("/custom/repo").webDir; gotRoot != filepath.Join("/custom/repo", "web") {
		t.Fatalf("NewAnalyzer(\"/custom/repo\").webDir = %q, want %q", gotRoot, filepath.Join("/custom/repo", "web"))
	}
	if analyzer.ID() != simcheck.LangTS || analyzer.DisplayName() != "TypeScript" || analyzer.CodeFence() != "ts" {
		t.Fatalf("unexpected Analyzer metadata: %s / %s / %s", analyzer.ID(), analyzer.DisplayName(), analyzer.CodeFence())
	}

	emptyUnits, err := analyzer.ExtractUnits(context.Background(), nil, 3, 6)
	if err != nil || emptyUnits != nil {
		t.Fatalf("ExtractUnits(nil) = (%v, %v), want (nil, nil)", emptyUnits, err)
	}
	emptyKeys, err := analyzer.ExtractBaseKeys(context.Background(), nil)
	if err != nil || emptyKeys != nil {
		t.Fatalf("ExtractBaseKeys(nil) = (%v, %v), want (nil, nil)", emptyKeys, err)
	}

	src := `import { Component, computed, input, output } from '@angular/core';

export function tinyHelper(x: number): number {
  return x + 1;
}

export function formatBytes(bytes: number): string {
  if (bytes <= 0 || !Number.isFinite(bytes)) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB'];
  const idx = Math.min(Math.floor(Math.log2(bytes) / 10), units.length - 1);
  const val = bytes / Math.pow(1024, idx);
  return val.toFixed(1) + ' ' + units[idx];
}

export const dedupItems = <T>(items: readonly T[]): T[] => {
  const seen = new Set<T>();
  const out: T[] = [];
  for (const item of items) {
    if (!seen.has(item)) {
      seen.add(item);
      out.push(item);
    }
  }
  return out;
};

@Component({ selector: 'khi-sample', template: '' })
export class SampleComponent {
  public readonly items = input.required<string[]>();
  public readonly selected = output<string>();
  protected readonly oneLineComputed = computed(() => this.items().length);

  protected readonly visibleLabels = computed(() => {
    const raw = this.items();
    if (raw.length === 0) {
      return [];
    }
    const filtered = raw.filter((v) => v.trim() !== '');
    return filtered.map((v) => v.toUpperCase());
  });

  public computeTotal(values: number[]): number {
    let sum = 0;
    for (const v of values) {
      if (v > 0) {
        sum += v;
      }
    }
    return sum;
  }
}
`
	byFile, err := analyzer.ExtractUnits(context.Background(), []simcheck.FileInput{
		{Path: "web/src/app/sample/sample.component.ts", Content: []byte(src)},
		{Path: "web/src/app/sample/gen.ts", Content: []byte("// @generated\nexport function gen() {\n  const a = 1;\n  const b = 2;\n  const c = 3;\n  const d = 4;\n  return a + b + c + d;\n}\n")},
	}, 3, 6)
	if err != nil {
		t.Fatalf("ExtractUnits() failed: %v", err)
	}

	if _, hasGen := byFile["web/src/app/sample/gen.ts"]; hasGen {
		t.Errorf("expected @generated file to be skipped")
	}

	type summary struct {
		Name string
		Kind simcheck.UnitKind
		Key  string
	}
	var got []summary
	for _, u := range byFile["web/src/app/sample/sample.component.ts"] {
		got = append(got, summary{Name: u.Name, Kind: u.Kind, Key: u.Key})
	}
	want := []summary{
		{Name: "formatBytes", Kind: simcheck.KindFunc, Key: "func:formatBytes"},
		{Name: "dedupItems", Kind: simcheck.KindFunc, Key: "func:dedupItems"},
		{Name: "SampleComponent.visibleLabels", Kind: simcheck.KindClosure, Key: "closure:SampleComponent.visibleLabels"},
		{Name: "SampleComponent.computeTotal", Kind: simcheck.KindMethod, Key: "method:SampleComponent.computeTotal"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ExtractUnits() mismatch (-want +got):\n%s", diff)
	}

	baseKeys, err := analyzer.ExtractBaseKeys(context.Background(), []simcheck.FileInput{
		{Path: "web/src/app/sample/sample.component.ts", Content: []byte(src)},
	})
	if err != nil {
		t.Fatalf("ExtractBaseKeys() failed: %v", err)
	}
	if _, ok := baseKeys["web/src/app/sample/sample.component.ts"]["func:tinyHelper"]; !ok {
		t.Errorf("expected ExtractBaseKeys to include small helper func:tinyHelper")
	}
}

func TestTSASTNormalizationEquivalence(t *testing.T) {
	analyzer := NewAnalyzer("")

	funcA := `/** Formats cluster labels. */
export function buildSummaryA(inputState: { count: number; items: string[] }): { ok: boolean; count: number } {
  // Destructure count shorthand
  const { count } = inputState;
  let valid = 0;
  for (const label of inputState.items) {
    if (label.trim() !== '') {
      valid += count;
    }
  }
  return { ok: valid > 0, count };
}
`
	funcB := `function computeOtherSummary(payload: { count: number; items: string[] }): { ok: boolean; count: number } {
  const { count: totalCount } = payload;
  let acc = 0;
  for (const entry of payload.items) {
    if (entry.trim() !== '') {
      acc += totalCount;
    }
  }
  return { ok: acc > 0, count: totalCount };
}
`
	byFile, err := analyzer.ExtractUnits(context.Background(), []simcheck.FileInput{
		{Path: "web/src/app/a.ts", Content: []byte(funcA)},
		{Path: "web/src/app/b.ts", Content: []byte(funcB)},
	}, 3, 6)
	if err != nil {
		t.Fatalf("ExtractUnits() failed: %v", err)
	}
	unitsA := byFile["web/src/app/a.ts"]
	unitsB := byFile["web/src/app/b.ts"]
	if len(unitsA) != 1 || len(unitsB) != 1 {
		t.Fatalf("got len(A)=%d, len(B)=%d; want 1 each", len(unitsA), len(unitsB))
	}
	if diff := cmp.Diff(unitsA[0].NormText, unitsB[0].NormText); diff != "" {
		t.Errorf("expected identical TypeScript NormText across renamed variables & shorthand properties (-A +B):\n%s", diff)
	}
}

func TestDiscoverTSCorpusAndSharedModule(t *testing.T) {
	emptyTmp := t.TempDir()
	emptyAnalyzer := NewAnalyzer(emptyTmp)
	emptyCorpus, err := emptyAnalyzer.DiscoverCorpusFiles(emptyTmp)
	if err != nil || emptyCorpus != nil {
		t.Fatalf("DiscoverCorpusFiles(no web/src) = (%v, %v), want (nil, nil)", emptyCorpus, err)
	}

	tmp := t.TempDir()
	files := map[string]string{
		"web/src/app/utils/time.ts":             "export const x = 1;\n",
		"web/src/app/utils/time.spec.ts":        "describe('x', () => {});\n",
		"web/src/app/common/dom-util.ts":        "export const y = 1;\n",
		"web/src/app/dialogs/view.component.ts": "export class V {}\n",
		"web/src/app/dialogs/view.stories.ts":   "export default {};\n",
		"web/src/app/generated/api.ts":          "export const g = 1;\n",
		"web/src/environments/environment.ts":   "export const env = {};\n",
		"web/src/assets/zzz-icons.ts":           "export const z = 1;\n",
		"web/src/.hidden/h.ts":                  "export const h = 1;\n",
		"web/src/google.d.ts":                   "declare const g: unknown;\n",
	}
	for rel, content := range files {
		full := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}

	analyzer := NewAnalyzer(tmp)
	got, err := analyzer.DiscoverCorpusFiles(tmp)
	if err != nil {
		t.Fatalf("DiscoverCorpusFiles() failed: %v", err)
	}
	want := []string{
		"web/src/app/common/dom-util.ts",
		"web/src/app/dialogs/view.component.ts",
		"web/src/app/utils/time.ts",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DiscoverCorpusFiles() mismatch (-want +got):\n%s", diff)
	}

	sharedCases := []struct {
		mod  string
		want bool
	}{
		{"web/src/app/common", true},
		{"web/src/app/common/schema", true},
		{"web/src/app/utils", true},
		{"web/src/app/shared/components", true},
		{"web/src/app/commonly", false},
		{"web/src/app/timeline", false},
		{"web/src/app/dialogs/startup", false},
	}
	for _, sc := range sharedCases {
		if gotShared := analyzer.IsSharedModule(sc.mod); gotShared != sc.want {
			t.Errorf("IsSharedModule(%q) = %v, want %v", sc.mod, gotShared, sc.want)
		}
	}

	brokenRepo := t.TempDir()
	brokenWeb := filepath.Join(brokenRepo, "web")
	if err := os.MkdirAll(brokenWeb, 0o755); err != nil {
		t.Fatalf("mkdir brokenWeb failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(brokenWeb, "package.json"), []byte(`{"name":"no-ts"}`), 0o600); err != nil {
		t.Fatalf("write package.json failed: %v", err)
	}
	brokenAnalyzer := NewAnalyzer(brokenRepo)
	_, tsErr := brokenAnalyzer.ExtractUnits(context.Background(), []simcheck.FileInput{
		{Path: "web/src/app/a.ts", Content: []byte("export function f() {}")},
	}, 1, 1)
	if tsErr == nil || !strings.Contains(tsErr.Error(), "Cannot find module") {
		t.Errorf("ExtractUnits(missing typescript module) err = %v, want stderr mentioning Cannot find module", tsErr)
	}
}

func TestExtractAllRepositoryProductionTSFiles(t *testing.T) {
	analyzer := NewAnalyzer("")
	webDir, err := analyzer.resolveWebDir()
	if err != nil {
		t.Fatalf("resolveWebDir() failed: %v", err)
	}
	repoRoot := filepath.Dir(webDir)

	explicitAnalyzer := NewAnalyzer(repoRoot)
	if gotWebDir, err := explicitAnalyzer.resolveWebDir(); err != nil || gotWebDir != webDir {
		t.Fatalf("explicit resolveWebDir() = (%q, %v), want (%q, nil)", gotWebDir, err, webDir)
	}

	files, err := explicitAnalyzer.DiscoverCorpusFiles(repoRoot)
	if err != nil {
		t.Fatalf("DiscoverCorpusFiles(%s) failed: %v", repoRoot, err)
	}
	if len(files) == 0 {
		t.Fatalf("expected non-empty production TypeScript corpus in %s", repoRoot)
	}

	inputs := make([]simcheck.FileInput, 0, len(files))
	for _, rel := range files {
		raw, readErr := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if readErr != nil {
			t.Fatalf("read %s failed: %v", rel, readErr)
		}
		inputs = append(inputs, simcheck.FileInput{Path: rel, Content: raw})
	}

	byFile, err := explicitAnalyzer.ExtractUnits(context.Background(), inputs, 3, 6)
	if err != nil {
		t.Fatalf("ExtractUnits() over %d real KHI TypeScript files failed: %v", len(inputs), err)
	}
	totalUnits := 0
	for _, units := range byFile {
		totalUnits += len(units)
	}
	t.Logf("Extracted %d qualifying TypeScript units across %d production files", totalUnits, len(files))
	if totalUnits == 0 {
		t.Errorf("expected >0 qualifying TypeScript units across KHI web/src")
	}
}
