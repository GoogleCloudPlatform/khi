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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

func TestPlanTSFileMutationsAndExclusions(t *testing.T) {
	realRepoRoot := mustFindRepoRootForTSTest(t)
	if _, err := os.Stat(filepath.Join(realRepoRoot, "web", "node_modules", "typescript")); err != nil {
		t.Skipf("skipping TS mutation AST test: web/node_modules not installed (%v)", err)
	}

	tmpRepo := setupSymlinkedWebNodeModules(t, realRepoRoot)
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
    // CODEQUALITY_ALLOW(func_surviving_mutants, perf): fast-path cache check does not alter functional result
    if (coerced) {
      return chosen ?? 'ok';
    }
    // CODEQUALITY_ALLOW(func_surviving_mutants, invariant): impossible state
    if (!flag && chosen) {
      this.active.set(true);
    }
    // CODEQUALITY_ALLOW(func_surviving_mutants, equivalent): unused allow comment on pure assignment
    const marker = 1;
    void marker;
    return flag ? 'yes' : 'no';
  }

  isDisabled(): boolean {
    return false;
  }
}
`
	writeTSTestFile(t, filepath.Join(tmpRepo, filepath.FromSlash(relFile)), src)

	extractions, err := InvokeTSMutatorScript(context.Background(), tmpRepo, []string{relFile})
	if err != nil {
		t.Fatalf("InvokeTSMutatorScript() error: %v", err)
	}
	ext, ok := extractions[relFile]
	if !ok {
		t.Fatalf("missing extraction entry for %s", relFile)
	}

	allChanged := make(map[int]struct{})
	for line := 1; line <= 65; line++ {
		allChanged[line] = struct{}{}
	}
	nextID := 1
	fp, sites := PlanTSFileMutations(PlanTSFileInput{
		FilePath:     relFile,
		Source:       []byte(src),
		Extraction:   ext,
		ChangedLines: allChanged,
		NextID:       &nextID,
	})

	assertTSPlanExpectations(t, fp)
	verifyRenderedTSSyntax(t, tmpRepo, []byte(src), sites)

	// Verify probe extraction on unchanged lines and uncovered line marking when HasTSCoverage is true.
	probeID := 1
	nextCovID := 1
	covPlan, probeSites := PlanTSFileMutations(PlanTSFileInput{
		FilePath:      relFile,
		Source:        []byte(src),
		Extraction:    ext,
		ChangedLines:  map[int]struct{}{52: {}},
		CoveredLines:  map[int]struct{}{},
		HasTSCoverage: true,
		NextID:        &nextCovID,
		NextProbeID:   &probeID,
	})
	if len(covPlan.Mutants) == 0 || covPlan.Mutants[0].Status != rule.MutantUncovered || len(probeSites) == 0 {
		t.Errorf("expected uncovered mutant and unchanged-line probes, got mutants=%+v probes=%d", covPlan.Mutants, len(probeSites))
	}
}

func TestPlanTSFileMutationsGuardsAndProbes(t *testing.T) {
	t.Run("filters out-of-bounds duplicate invalid-op and empty-candidate sites", func(t *testing.T) {
		src := []byte("a < b")
		nextID := 1
		fp, sites := PlanTSFileMutations(PlanTSFileInput{
			FilePath:     "web/src/app/bounds.ts",
			Source:       src,
			ChangedLines: map[int]struct{}{1: {}},
			NextID:       &nextID,
			Extraction: RawTSFileExtraction{
				Sites: []RawTSExtracted{
					{Kind: "ordered_cmp", Operator: "ROR", Line: 1, StartOffset: -1, EndOffset: 3, Candidates: []RawTSCandidate{{OrigExpr: "a", MutExpr: "b", ReplacementPreview: "b"}}},
					{Kind: "ordered_cmp", Operator: "ROR", Line: 1, StartOffset: 0, EndOffset: 6, Candidates: []RawTSCandidate{{OrigExpr: "a", MutExpr: "b", ReplacementPreview: "b"}}},
					{Kind: "ordered_cmp", Operator: "ROR", Line: 1, StartOffset: 2, EndOffset: 2, Candidates: []RawTSCandidate{{OrigExpr: "a", MutExpr: "b", ReplacementPreview: "b"}}},
					{Kind: "ordered_cmp", Operator: "ROR", Line: 1, StartOffset: 3, EndOffset: 2, Candidates: []RawTSCandidate{{OrigExpr: "a", MutExpr: "b", ReplacementPreview: "b"}}},
					{Kind: "ordered_cmp", Operator: "INVALID_OP", Line: 1, StartOffset: 0, EndOffset: 2, Candidates: []RawTSCandidate{{OrigExpr: "a", MutExpr: "b", ReplacementPreview: "b"}}},
					{Kind: "ordered_cmp", Operator: "ROR", Line: 1, StartOffset: 1, EndOffset: 3, Candidates: nil},
					{Kind: "ordered_cmp", Op: "<", Operator: "ROR", Line: 1, StartOffset: 0, EndOffset: 5, LeftStart: 0, LeftEnd: 1, RightStart: 4, RightEnd: 5, Candidates: []RawTSCandidate{{OrigExpr: "a < b", MutExpr: "a <= b", ReplacementPreview: "a <= b"}}},
					{Kind: "ordered_cmp", Op: "<", Operator: "ROR", Line: 1, StartOffset: 0, EndOffset: 5, LeftStart: 0, LeftEnd: 1, RightStart: 4, RightEnd: 5, Candidates: []RawTSCandidate{{OrigExpr: "a < b", MutExpr: "a >= b", ReplacementPreview: "a >= b"}}},
				},
			},
		})
		wantMutant := &TSMutant{
			ID:           1,
			File:         "web/src/app/bounds.ts",
			Line:         1,
			Operator:     rule.OperatorROR,
			OriginalExpr: "a < b",
			MutatedExpr:  "a <= b",
			OriginalLine: "a < b",
			MutatedLine:  "a <= b",
			Status:       rule.MutantPending,
		}
		if diff := cmp.Diff([]*TSMutant{wantMutant}, fp.Mutants); diff != "" {
			t.Errorf("fp.Mutants mismatch (-want +got):\n%s", diff)
		}
		wantSites := []TSMutationSite{{
			Kind: "ordered_cmp", Op: "<", StartOffset: 0, EndOffset: 5, LeftStart: 0, LeftEnd: 1, RightStart: 4, RightEnd: 5,
			Mutants: []*TSMutant{wantMutant},
		}}
		if diff := cmp.Diff(wantSites, sites); diff != "" {
			t.Errorf("activeSites mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("plans dual-slot and single-slot unchanged branch probes while skipping non-branch and duplicate spans", func(t *testing.T) {
		src := []byte("01234567890123456789")
		nextID := 1
		probeID := 1
		_, sites := PlanTSFileMutations(PlanTSFileInput{
			FilePath:     "web/src/app/probes.ts",
			Source:       src,
			ChangedLines: map[int]struct{}{},
			NextID:       &nextID,
			NextProbeID:  &probeID,
			Extraction: RawTSFileExtraction{
				Sites: []RawTSExtracted{
					{Kind: "arith", Line: 1, StartOffset: 0, EndOffset: 2},
					{Kind: "ordered_cmp", Line: 1, StartOffset: 2, EndOffset: 4},
					{Kind: "ordered_cmp", Line: 1, StartOffset: 2, EndOffset: 4},
					{Kind: "nullish", Line: 1, StartOffset: 4, EndOffset: 6},
					{Kind: "cond", Line: 1, StartOffset: 6, EndOffset: 8},
					{Kind: "eq_cmp", Line: 1, StartOffset: 8, EndOffset: 10},
				},
			},
		})
		if probeID != 5 {
			t.Errorf("probeID = %d, want 5", probeID)
		}
		p1, p2, p3, p4 := &TSMutant{ID: -1}, &TSMutant{ID: -2}, &TSMutant{ID: -3}, &TSMutant{ID: -4}
		wantSites := []TSMutationSite{
			{Kind: "ordered_cmp", StartOffset: 2, EndOffset: 4, Mutants: []*TSMutant{p1, p1}},
			{Kind: "nullish", StartOffset: 4, EndOffset: 6, Mutants: []*TSMutant{p2, p2}},
			{Kind: "cond", StartOffset: 6, EndOffset: 8, Mutants: []*TSMutant{p3, p3}},
			{Kind: "eq_cmp", StartOffset: 8, EndOffset: 10, Mutants: []*TSMutant{p4}},
		}
		if diff := cmp.Diff(wantSites, sites); diff != "" {
			t.Errorf("probe sites mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestBuildTSDiffLinesAndOffsets(t *testing.T) {
	testCases := []struct {
		name     string
		src      string
		raw      RawTSExtracted
		preview  string
		wantOrig string
		wantMut  string
	}{
		{
			name:     "line zero returns empty original and preview fallback",
			src:      "const a = 1;",
			raw:      RawTSExtracted{Line: 0, StartOffset: 0, EndOffset: 5},
			preview:  "mut",
			wantOrig: "",
			wantMut:  "mut",
		},
		{
			name:     "line past total lines returns empty original and preview fallback",
			src:      "first\nsecond",
			raw:      RawTSExtracted{Line: 3, StartOffset: 0, EndOffset: 5},
			preview:  "mut",
			wantOrig: "",
			wantMut:  "mut",
		},
		{
			name:     "first line starting at byte zero with crlf trims cleanly",
			src:      "a < b;\r\nnext",
			raw:      RawTSExtracted{Line: 1, StartOffset: 0, EndOffset: 5},
			preview:  "a <= b",
			wantOrig: "a < b;",
			wantMut:  "a <= b;",
		},
		{
			name:     "last line without trailing newline splices up to eof",
			src:      "first\nreturn a < b",
			raw:      RawTSExtracted{Line: 2, StartOffset: 13, EndOffset: 18},
			preview:  "a <= b",
			wantOrig: "return a < b",
			wantMut:  "return a <= b",
		},
		{
			name:     "multiline span exceeding line end falls back to preview",
			src:      "return a &&\n  b;",
			raw:      RawTSExtracted{Line: 1, StartOffset: 7, EndOffset: 15},
			preview:  "a || b",
			wantOrig: "return a &&",
			wantMut:  "a || b",
		},
		{
			name:     "span starting before line start falls back to preview",
			src:      "first\nsecond",
			raw:      RawTSExtracted{Line: 2, StartOffset: 2, EndOffset: 10},
			preview:  "alt",
			wantOrig: "second",
			wantMut:  "alt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			starts := computeTSLineOffsets([]byte(tc.src))
			gotOrig, gotMut := buildTSDiffLines([]byte(tc.src), starts, tc.raw, tc.preview)
			if diff := cmp.Diff([2]string{tc.wantOrig, tc.wantMut}, [2]string{gotOrig, gotMut}); diff != "" {
				t.Errorf("buildTSDiffLines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func assertTSPlanExpectations(t *testing.T, fp *TSFilePlan) {
	t.Helper()
	if len(fp.DirectiveErrors) != 2 {
		t.Fatalf("len(DirectiveErrors) = %d, want 2: %+v", len(fp.DirectiveErrors), fp.DirectiveErrors)
	}
	if !strings.Contains(fp.DirectiveErrors[0].Message, "invariant or unreachable branches cannot be suppressed") {
		t.Errorf("DirectiveErrors[0] = %q, want invariant error", fp.DirectiveErrors[0].Message)
	}
	if !strings.Contains(fp.DirectiveErrors[1].Message, "unused CODEQUALITY_ALLOW") {
		t.Errorf("DirectiveErrors[1] = %q, want unused directive error", fp.DirectiveErrors[1].Message)
	}
	if len(fp.Suppressed) != 1 {
		t.Errorf("len(Suppressed) = %d, want 1", len(fp.Suppressed))
	}

	suppressedCount := 0
	var gotOps []rule.MutantOperator
	for _, m := range fp.Mutants {
		if m.Status == rule.MutantSuppressed {
			suppressedCount++
			continue
		}
		gotOps = append(gotOps, m.Operator)
	}
	if suppressedCount != 2 {
		t.Errorf("suppressedCount = %d, want 2", suppressedCount)
	}
	wantOps := []rule.MutantOperator{
		rule.OperatorLCR,
		rule.OperatorLCR,
		rule.OperatorROR,
		rule.OperatorROR,
		rule.OperatorLCR,
		rule.OperatorUOI,
		rule.OperatorSBR,
		rule.OperatorBCR,
		rule.OperatorBCR,
		rule.OperatorBLR,
	}
	if diff := cmp.Diff(wantOps, gotOps); diff != "" {
		t.Errorf("extracted active TypeScript mutant operators mismatch (-want +got):\n%s", diff)
	}
}

func verifyRenderedTSSyntax(t *testing.T, tmpRepo string, src []byte, sites []TSMutationSite) {
	t.Helper()
	rendered := RenderTSSuperMutantSource(src, sites)
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
	writeTSTestFile(t, renderedPath, string(rendered))
	writeTSTestFile(t, verifyScript, verifyCode)
	if _, err := gotestutil.RunCommandOutput(context.Background(), tmpRepo, "node", verifyScript, tmpRepo, renderedPath); err != nil {
		t.Fatalf("rendered Super-Mutant TypeScript failed syntax parse: %v\n---\n%s", err, string(rendered))
	}
}
