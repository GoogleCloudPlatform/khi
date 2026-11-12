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
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
	"github.com/google/go-cmp/cmp"
)

func TestPlanTSFileMutationsAndExclusions(t *testing.T) {
	realRepoRoot := mustFindRepoRootForTSMutationTest(t)
	if _, err := os.Stat(filepath.Join(realRepoRoot, "web", "node_modules", "typescript")); err != nil {
		t.Skipf("skipping TS mutation AST test: web/node_modules not installed (%v)", err)
	}

	tmpRepo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpRepo, "web"), 0o755); err != nil {
		t.Fatalf("failed creating web dir: %v", err)
	}
	if err := os.Symlink(filepath.Join(realRepoRoot, "web", "package.json"), filepath.Join(tmpRepo, "web", "package.json")); err != nil {
		t.Fatalf("failed symlinking package.json: %v", err)
	}
	if err := os.Symlink(filepath.Join(realRepoRoot, "web", "node_modules"), filepath.Join(tmpRepo, "web", "node_modules")); err != nil {
		t.Fatalf("failed symlinking node_modules: %v", err)
	}

	relFile := "web/src/app/sample/widget.component.ts"
	src := `import { Component, inject, input, signal, viewChild } from '@angular/core';

export interface WidgetModel {
  readonly id: string;
}

export enum WidgetMode {
  Compact = 'compact',
}

const HOUR_MS = 60 * 60 * 1000;

@Component({
  selector: 'khi-widget',
  template: '<div>{{ title() }}</div>',
})
export class WidgetComponent {
  protected readonly MAX_ROWS = 100 + 20;
  private readonly dep = inject(Object);
  private readonly destroyRef = inject(Object) as { destroyed: boolean; onDestroy: (cb: () => void) => void };
  readonly title = input<string>('init' + 'ial');
  private readonly box = viewChild('box');
  readonly active = signal(false);

  evaluate(tsNs: bigint, primary?: string, secondary?: string, flag?: boolean): string {
    console.warn('evaluating ' + String(tsNs));
    this.destroyRef.onDestroy(() => { this.active.set(false); });
    if (this.destroyRef.destroyed) {
      return '';
    }
    const coerced = !!flag;
    const trivialDefault = primary ?? '';
    const chosen = primary ?? secondary;
    if (tsNs < 0n) {
      throw new Error('negative ' + String(tsNs));
    }
    // NOMUTANTS(perf): fast-path cache check does not alter functional result
    if (coerced) {
      return chosen ?? 'ok';
    }
    // NOMUTANTS(invariant): impossible state
    if (!flag && chosen) {
      this.active.set(true);
    }
    return flag ? 'yes' : 'no';
  }

  isDisabled(): boolean {
    return false;
  }
}
`
	absFile := filepath.Join(tmpRepo, filepath.FromSlash(relFile))
	if err := os.MkdirAll(filepath.Dir(absFile), 0o755); err != nil {
		t.Fatalf("failed creating sample dir: %v", err)
	}
	if err := os.WriteFile(absFile, []byte(src), 0o600); err != nil {
		t.Fatalf("failed writing sample TS file: %v", err)
	}

	extractions, err := invokeTSMutatorScript(context.Background(), tmpRepo, []string{relFile})
	if err != nil {
		t.Fatalf("invokeTSMutatorScript() unexpected error: %v", err)
	}
	ext, ok := extractions[relFile]
	if !ok {
		t.Fatalf("missing extraction entry for %s", relFile)
	}

	allChanged := make(map[int]struct{})
	for line := 1; line <= 60; line++ {
		allChanged[line] = struct{}{}
	}

	nextID := 1
	fp, sites := PlanTSFileMutations(relFile, []byte(src), ext, allChanged, nil, false, &nextID, nil)

	if len(fp.Warnings) != 1 || !strings.Contains(fp.Warnings[0].Message, "invariant or unreachable conditions cannot be suppressed") {
		t.Errorf("expected exactly 1 forbidden invariant NOMUTANTS warning, got: %+v", fp.Warnings)
	}

	suppressedCount := 0
	var gotOps []MutantOperator
	for _, m := range fp.Mutants {
		if m.Line <= 31 {
			t.Errorf("unexpected mutant generated inside excluded declarations/decorators/console/teardown/!!/trivial-?? on line %d: %+v", m.Line, m)
		}
		if m.Line == 35 {
			t.Errorf("unexpected mutant generated inside new Error(...) message argument on line 35: %+v", m)
		}
		if m.Status == MutantSuppressed {
			suppressedCount++
			continue
		}
		gotOps = append(gotOps, m.Operator)
	}

	if suppressedCount != 2 {
		t.Errorf("suppressedCount = %d, want 2 (BCR true/false on line 33)", suppressedCount)
	}

	wantOps := []MutantOperator{
		OperatorLCR, // primary ?? secondary -> primary
		OperatorLCR, // primary ?? secondary -> secondary
		OperatorROR, // tsNs < 0n -> <=
		OperatorROR, // tsNs < 0n -> >=
		OperatorLCR, // !flag && chosen -> ||
		OperatorUOI, // !flag -> flag
		OperatorSBR, // skip this.active.set(true);
		OperatorBCR, // flag ? 'yes' : 'no' -> true
		OperatorBCR, // flag ? 'yes' : 'no' -> false
		OperatorBLR, // return false -> true
	}
	if diff := cmp.Diff(wantOps, gotOps); diff != "" {
		t.Errorf("extracted active TypeScript mutant operators mismatch (-want +got):\n%s", diff)
	}

	rendered := RenderTSSuperMutantSource([]byte(src), sites)
	verifyScript := filepath.Join(tmpRepo, "verify_parse.js")
	verifyCode := `
const fs = require('fs');
const path = require('path');
const { createRequire } = require('module');
const ts = createRequire(path.join(process.argv[2], 'web', 'package.json'))('typescript');
const code = fs.readFileSync(process.argv[3], 'utf8');
const sf = ts.createSourceFile('mutant.ts', code, ts.ScriptTarget.ES2022, true, ts.ScriptKind.TS);
if (sf.parseDiagnostics && sf.parseDiagnostics.length > 0) {
  console.error(sf.parseDiagnostics.map(d => d.messageText).join('\n'));
  process.exit(1);
}
`
	renderedPath := filepath.Join(tmpRepo, "rendered.ts")
	if err := os.WriteFile(renderedPath, rendered, 0o600); err != nil {
		t.Fatalf("failed writing rendered.ts: %v", err)
	}
	if err := os.WriteFile(verifyScript, []byte(verifyCode), 0o600); err != nil {
		t.Fatalf("failed writing verify_parse.js: %v", err)
	}
	if _, err := gotestutil.RunCommandOutput(context.Background(), tmpRepo, "node", verifyScript, tmpRepo, renderedPath); err != nil {
		t.Fatalf("rendered Super-Mutant TypeScript failed syntax parse: %v\n---\n%s", err, string(rendered))
	}
}

func TestGenerateTSBrowserRuntimeSemantics(t *testing.T) {
	tmpDir := t.TempDir()
	runtimeFile := filepath.Join(tmpDir, tsBrowserRuntimeFileName)
	if err := os.WriteFile(runtimeFile, GenerateTSBrowserRuntimeSource(10, 5), 0o600); err != nil {
		t.Fatalf("failed writing runtime JS: %v", err)
	}

	driverFile := filepath.Join(tmpDir, "driver.js")
	driverSrc := `
const fs = require('fs');
const vm = require('vm');
const runtimeCode = fs.readFileSync(process.argv[2], 'utf8');

function runWithArgs(args, fn) {
  let reported = null;
  let registeredReporter = null;
  const sandbox = {
    __karma__: {
      config: { args },
      info: (payload) => { reported = payload; },
      start: () => {},
    },
    jasmine: {
      getEnv: () => ({
        addReporter: (rep) => { registeredReporter = rep; },
      }),
    },
  };
  sandbox.globalThis = sandbox;
  vm.createContext(sandbox);
  vm.runInContext(runtimeCode, sandbox);
  sandbox.__karma__.start();
  const value = fn(sandbox.__khiMut, registeredReporter);
  return { value, reported };
}

// 1. Baseline Pass 0: preserves JS value-returning && / || / ?? and bigint comparisons while recording hits & polarities.
const base = runWithArgs(['--khi-mutant=0'], (m, rep) => {
  const topHit = m.lt(1, 2, 5n, 10n);
  rep.specStarted({ id: 'spec0', fullName: 'SampleSuite handles values' });
  const andVal = m.and(3, () => 'truthy', () => 'rhs');
  const coalVal = m.coal(4, 5, () => null, () => 'fallback');
  const probeVal = m.lt(-2, -2, 1, 10);
  rep.specDone({ id: 'spec0', fullName: 'SampleSuite handles values', status: 'passed' });
  rep.jasmineDone();
  return { topHit, andVal, coalVal, probeVal };
});

// 2. Active mutant 3: && -> || returns left truthy operand without calling rhs.
let rhsCalled = false;
const mutAnd = runWithArgs(['--khi-mutant=3'], (m) =>
  m.and(3, () => 'leftVal', () => { rhsCalled = true; return 'rightVal'; })
);

// 3. Active mutant 4: a ?? b -> keep left (returns null instead of fallback).
const mutCoalLeft = runWithArgs(['--khi-mutant=4'], (m) =>
  m.coal(4, 5, () => null, () => 'fallback')
);

process.stdout.write(JSON.stringify({
  baseValue: base.value,
  specHits: base.reported.khiMutantReport.specHits,
  specOrder: base.reported.khiMutantReport.specOrder,
  specPolarities: base.reported.khiMutantReport.specPolarities,
  topLevelHits: base.reported.khiMutantReport.topLevelHits,
  mutAnd: mutAnd.value,
  rhsCalled,
  mutCoalLeft: mutCoalLeft.value,
}));
`
	if err := os.WriteFile(driverFile, []byte(driverSrc), 0o600); err != nil {
		t.Fatalf("failed writing driver.js: %v", err)
	}

	outBytes, err := gotestutil.RunCommandOutput(context.Background(), tmpDir, "node", driverFile, runtimeFile)
	if err != nil {
		t.Fatalf("runtime driver failed: %v", err)
	}

	type driverOutput struct {
		BaseValue struct {
			TopHit  bool   `json:"topHit"`
			AndVal  string `json:"andVal"`
			CoalVal string `json:"coalVal"`
		} `json:"baseValue"`
		SpecHits       map[string][]int    `json:"specHits"`
		SpecOrder      []TSSpecOrderEntry  `json:"specOrder"`
		SpecPolarities map[string][]string `json:"specPolarities"`
		TopLevelHits   []int               `json:"topLevelHits"`
		MutAnd         string              `json:"mutAnd"`
		RhsCalled      bool                `json:"rhsCalled"`
		MutCoalLeft    *string             `json:"mutCoalLeft"`
	}

	var got driverOutput
	if err := json.Unmarshal(outBytes, &got); err != nil {
		t.Fatalf("failed decoding driver JSON: %v", err)
	}

	if !got.BaseValue.TopHit || got.BaseValue.AndVal != "rhs" || got.BaseValue.CoalVal != "fallback" {
		t.Errorf("unexpected baseline semantics: %+v", got.BaseValue)
	}
	if diff := cmp.Diff([]int{1, 2}, got.TopLevelHits); diff != "" {
		t.Errorf("topLevelHits mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(map[string][]int{"SampleSuite handles values": {3, 4, 5}}, got.SpecHits); diff != "" {
		t.Errorf("specHits mismatch (-want +got):\n%s", diff)
	}
	wantOrder := []TSSpecOrderEntry{{ID: "spec0", FullName: "SampleSuite handles values"}}
	if diff := cmp.Diff(wantOrder, got.SpecOrder); diff != "" {
		t.Errorf("specOrder mismatch (-want +got):\n%s", diff)
	}
	wantPolarities := map[string][]string{"spec0": {"3:T", "4:F", "5:F", "P2:T"}}
	if diff := cmp.Diff(wantPolarities, got.SpecPolarities); diff != "" {
		t.Errorf("specPolarities mismatch (-want +got):\n%s", diff)
	}
	if got.MutAnd != "leftVal" || got.RhsCalled {
		t.Errorf("mutated && -> || expected short-circuited 'leftVal', got %q (rhsCalled=%v)", got.MutAnd, got.RhsCalled)
	}
	if got.MutCoalLeft != nil {
		t.Errorf("mutated ?? keep-left expected nil, got %v", *got.MutCoalLeft)
	}
}

func TestDiscoverAndAnalyzeTSTestRedundancy(t *testing.T) {
	realRepoRoot := mustFindRepoRootForTSMutationTest(t)
	if _, err := os.Stat(filepath.Join(realRepoRoot, "web", "node_modules", "typescript")); err != nil {
		t.Skipf("skipping TS redundant test AST test: web/node_modules not installed (%v)", err)
	}

	tmpRepo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpRepo, "web"), 0o755); err != nil {
		t.Fatalf("failed creating web dir: %v", err)
	}
	if err := os.Symlink(filepath.Join(realRepoRoot, "web", "package.json"), filepath.Join(tmpRepo, "web", "package.json")); err != nil {
		t.Fatalf("failed symlinking package.json: %v", err)
	}
	if err := os.Symlink(filepath.Join(realRepoRoot, "web", "node_modules"), filepath.Join(tmpRepo, "web", "node_modules")); err != nil {
		t.Fatalf("failed symlinking node_modules: %v", err)
	}

	relSpec := "web/src/app/sample/sign.component.spec.ts"
	specSrc := `describe('SignComponent', () => {
  let component: { isNegative: (v: number) => boolean };
  let fixture: { nativeElement: { querySelector: (s: string) => unknown } };

  beforeEach(() => {
    component = { isNegative: (v: number) => v < 0 };
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should render badge element', () => {
    const badge = fixture.nativeElement.querySelector('.badge');
    expect(badge).not.toBeNull();
  });

  describe('isNegative', () => {
    const testCases = [
      { desc: 'neg one', in: -1, want: true },
      { desc: 'neg hundred', in: -100, want: true },
      { desc: 'zero boundary', in: 0, want: false },
    ];

    for (const tc of testCases) {
      it(` + "`should return ${tc.want} for ${tc.desc}`" + `, () => {
        expect(component.isNegative(tc.in)).toBe(tc.want);
      });
    }
  });
});
`
	absSpec := filepath.Join(tmpRepo, filepath.FromSlash(relSpec))
	if err := os.MkdirAll(filepath.Dir(absSpec), 0o755); err != nil {
		t.Fatalf("failed creating spec dir: %v", err)
	}
	if err := os.WriteFile(absSpec, []byte(specSrc), 0o600); err != nil {
		t.Fatalf("failed writing spec file: %v", err)
	}

	extractions, err := invokeTSMutatorScript(context.Background(), tmpRepo, []string{relSpec})
	if err != nil {
		t.Fatalf("invokeTSMutatorScript() unexpected error: %v", err)
	}
	ext, ok := extractions[relSpec]
	if !ok {
		t.Fatalf("missing extraction entry for %s", relSpec)
	}

	// Mark line 9 ('should create') and line 21 ('neg hundred' table row) as newly added in the PR diff.
	changedLines := map[int]struct{}{
		9:  {},
		21: {},
	}
	discovered := DiscoverChangedTSSpecs(relSpec, ext.SpecCases, changedLines)

	wantDiscovered := []ChangedTSSpec{
		{
			File:          relSpec,
			FullName:      "SignComponent should create",
			Line:          9,
			InDiff:        true,
			WeakAssertion: true,
		},
		{
			File:          relSpec,
			FullName:      "SignComponent should render badge element",
			Line:          13,
			InDiff:        false,
			WeakAssertion: false,
		},
		{
			File:          relSpec,
			FullName:      "SignComponent isNegative should return true for neg one",
			Line:          20,
			InDiff:        false,
			WeakAssertion: false,
		},
		{
			File:          relSpec,
			FullName:      "SignComponent isNegative should return true for neg hundred",
			Line:          21,
			InDiff:        true,
			WeakAssertion: false,
		},
		{
			File:          relSpec,
			FullName:      "SignComponent isNegative should return false for zero boundary",
			Line:          22,
			InDiff:        false,
			WeakAssertion: false,
		},
	}
	if diff := cmp.Diff(wantDiscovered, discovered); diff != "" {
		t.Fatalf("DiscoverChangedTSSpecs() mismatch (-want +got):\n%s", diff)
	}

	// Simulate Pass 0 polarities and mutant 1 (v < 0 -> v <= 0) + mutant 2 (v < 0 -> v >= 0):
	// - spec0 ('should create') touches mutant 1/2 via beforeEach setup with :F polarity, kills 0 mutants.
	// - spec2 ('neg one') and spec3 ('neg hundred') both have :T polarity and both kill mutant 2 (v >= 0).
	// - spec4 ('zero boundary') has :F polarity and kills BOTH mutant 1 (v <= 0) and mutant 2 (v >= 0).
	specOrder := []TSSpecOrderEntry{
		{ID: "spec0", FullName: "SignComponent should create"},
		{ID: "spec1", FullName: "SignComponent should render badge element"},
		{ID: "spec2", FullName: "SignComponent isNegative should return true for neg one"},
		{ID: "spec3", FullName: "SignComponent isNegative should return true for neg hundred"},
		{ID: "spec4", FullName: "SignComponent isNegative should return false for zero boundary"},
	}
	specPolarities := map[string][]string{
		"spec0": {"1:F", "2:F"},
		"spec2": {"1:T", "2:T"},
		"spec3": {"1:T", "2:T"},
		"spec4": {"1:F", "2:F"},
	}
	mutantRuns := map[string]TSMutantSpecRun{
		"1": {
			RanSpecIDs:    []string{"spec0", "spec2", "spec3", "spec4"},
			FailedSpecIDs: []string{"spec4"},
		},
		"2": {
			RanSpecIDs:    []string{"spec0", "spec2", "spec3", "spec4"},
			FailedSpecIDs: []string{"spec2", "spec3", "spec4"},
		},
	}

	mutantFileByID := map[int]string{
		1:  "web/src/app/sample/sign.component.ts",
		2:  "web/src/app/sample/sign.component.ts",
		99: "web/src/app/services/unrelated.service.ts",
	}
	findings := AnalyzeTSTestRedundancy(discovered, specOrder, specPolarities, mutantRuns, mutantFileByID, nil)
	wantFindings := []RedundantTestFinding{
		{
			Kind:            RedundantWeakAssertion,
			File:            relSpec,
			Line:            9,
			TestName:        "SignComponent should create",
			DominatedBy:     "SignComponent isNegative should return false for zero boundary",
			DominatedByFile: relSpec,
			DominatedByLine: 22,
			CandidateKills:  0,
			DominatorKills:  2,
		},
		{
			Kind:            RedundantDuplicate,
			File:            relSpec,
			Line:            21,
			TestName:        "SignComponent isNegative should return true for neg hundred",
			DominatedBy:     "SignComponent isNegative should return true for neg one",
			DominatedByFile: relSpec,
			DominatedByLine: 20,
			CandidateKills:  1,
			DominatorKills:  1,
		},
	}
	if diff := cmp.Diff(wantFindings, findings); diff != "" {
		t.Errorf("AnalyzeTSTestRedundancy() mismatch (-want +got):\n%s", diff)
	}
}

func mustFindRepoRootForTSMutationTest(t *testing.T) string {
	t.Helper()
	_, callerFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(callerFile), "..", ".."))
}
