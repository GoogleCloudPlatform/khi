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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/redundancy"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/google/go-cmp/cmp"
)

type tsSpyProgress struct {
	phases int
}

func (s *tsSpyProgress) Phase(_ int, _ int, _ string) { s.phases++ }

func TestEvaluateEndToEndWithMockKarmaSession(t *testing.T) {
	realRepoRoot := mustFindRepoRootForTSTest(t)
	if _, err := os.Stat(filepath.Join(realRepoRoot, "web", "node_modules", "typescript")); err != nil {
		t.Skipf("skipping TS evaluator test: web/node_modules not installed (%v)", err)
	}

	tmpRepo := setupSymlinkedWebNodeModules(t, realRepoRoot)
	writeTSTestFile(t, filepath.Join(tmpRepo, "web", "tsconfig.json"), "{}\n")

	prodRel := "web/src/app/sample/calc.ts"
	specRel := "web/src/app/sample/calc.spec.ts"
	prodSrc := `export function calc(v: number): number {
  if (v === -999) { return -1; }
  if (v === -998) { return -2; }
  if (v < 0) {
    return 0;
  }
  return v + 1;
}
`
	specSrc := `describe('calc', () => {
  it('handles positive one', () => {
    expect(calc(1)).toBe(2);
  });
  it('handles positive five duplicate', () => {
    expect(calc(5)).toBe(6);
  });
});
`
	writeTSTestFile(t, filepath.Join(tmpRepo, filepath.FromSlash(prodRel)), prodSrc)
	writeTSTestFile(t, filepath.Join(tmpRepo, filepath.FromSlash(specRel)), specSrc)

	covJSONPath := filepath.Join(tmpRepo, "coverage-final.json")
	covJSON := `{
  "` + filepath.Join(tmpRepo, filepath.FromSlash(prodRel)) + `": {
    "path": "` + filepath.Join(tmpRepo, filepath.FromSlash(prodRel)) + `",
    "statementMap": {
      "0": {"start": {"line": 2, "column": 2}, "end": {"line": 2, "column": 32}},
      "1": {"start": {"line": 3, "column": 2}, "end": {"line": 3, "column": 32}},
      "2": {"start": {"line": 4, "column": 2}, "end": {"line": 6, "column": 3}},
      "3": {"start": {"line": 5, "column": 4}, "end": {"line": 5, "column": 13}},
      "4": {"start": {"line": 7, "column": 2}, "end": {"line": 7, "column": 15}}
    },
    "fnMap": {},
    "branchMap": {},
    "s": {"0": 1, "1": 1, "2": 1, "3": 0, "4": 1},
    "f": {},
    "b": {}
  }
}`
	writeTSTestFile(t, covJSONPath, covJSON)

	prog := &tsSpyProgress{}
	var runtimeCode string
	rep, err := Evaluate(context.Background(), Options{
		RepoRoot:       tmpRepo,
		BaseLabel:      "trunk()",
		TSCoveragePath: covJSONPath,
		ChangedLinesByFile: map[string]map[int]struct{}{
			prodRel: {4: {}, 5: {}, 7: {}},
		},
		ChangedTestLinesByFile: map[string]map[int]struct{}{
			specRel: {5: {}},
		},
		Progress: prog,
		RunKarmaSession: func(_ context.Context, req karmaSessionRequest) error {
			raw, readErr := os.ReadFile(filepath.Join(req.workDir, "web", tsBrowserRuntimeFileName))
			if readErr != nil {
				return readErr
			}
			runtimeCode = string(raw)
			return writeFakeKarmaPluginResults(req.cfgPath)
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() error: %v", err)
	}
	if !rep.Advisory || rep.HasBlockingIssues() || rep.FuncsEvaluated != 1 {
		t.Errorf("expected advisory non-blocking report with FuncsEvaluated=1, got Advisory=%v Blocking=%v FuncsEvaluated=%d", rep.Advisory, rep.HasBlockingIssues(), rep.FuncsEvaluated)
	}
	if !rep.Coverage.Enabled || rep.Coverage.PatchCovered != 2 || rep.Coverage.PatchExecutable != 3 {
		t.Errorf("unexpected rep.Coverage = %+v, want PatchCovered=2 PatchExecutable=3", rep.Coverage)
	}
	if !strings.Contains(runtimeCode, "if (id === 0 || id >= 4)") || !strings.Contains(runtimeCode, "if (pId >= 3)") {
		t.Errorf("expected mutant cap 4 and probe cap 3 in generated runtime, got:\n%s", runtimeCode)
	}

	assertFindingsRulesAndGlobalIndices(t, rep.Findings, []rule.RuleID{
		rule.RuleTestRedundancy,
		rule.RuleFuncSurvivingMutants,
		rule.RuleFuncUncoveredLines,
	})
	if prog.phases != 3 {
		t.Errorf("expected 3 progress phase callbacks, got %d", prog.phases)
	}

	// Empty diff returns clean PASS report without invoking node or Karma.
	cleanRep, err := Evaluate(context.Background(), Options{
		RepoRoot:         tmpRepo,
		BaseLabel:        "trunk()",
		FailOnViolations: true,
	})
	if err != nil || cleanRep.HasIssues() || cleanRep.Advisory {
		t.Errorf("Evaluate(empty, FailOnViolations=true) = (%+v, %v)", cleanRep, err)
	}
}

func TestEvaluatorPlanningAndMutationSessionBranches(t *testing.T) {
	t.Run("resolves default and explicit mutation timeouts", func(t *testing.T) {
		gotZeroOverall, gotZeroHard := resolveTSMutationTimeouts(0)
		if gotZeroOverall != 3*time.Minute || gotZeroHard != 3*time.Minute+35*time.Second {
			t.Errorf("resolveTSMutationTimeouts(0) = (%v, %v)", gotZeroOverall, gotZeroHard)
		}
		gotCustomOverall, gotCustomHard := resolveTSMutationTimeouts(45 * time.Second)
		if gotCustomOverall != 45*time.Second || gotCustomHard != 80*time.Second {
			t.Errorf("resolveTSMutationTimeouts(45s) = (%v, %v)", gotCustomOverall, gotCustomHard)
		}
	})

	t.Run("plans single changed file probe gating and missing file fallbacks", testPlanSingleTSChangedFileBranches)
	t.Run("registers planned pending suppressed and probe mutants", testRegisterPlannedTSFileBranches)
	t.Run("executes pending mutants skip budget and default session branches", testRunTSPendingMutantsBranches)

	t.Run("fails validation when Karma baseline reports failure", func(t *testing.T) {
		resPath := filepath.Join(t.TempDir(), "results.json")
		writeTSTestFile(t, resPath, `{"baselineFailed": true, "errorMessage": "Expected true to be false"}`)
		_, err := readAndValidateKarmaResults(resPath, nil)
		if err == nil || !strings.Contains(err.Error(), "baseline TypeScript Karma tests failed: Expected true to be false") {
			t.Fatalf("readAndValidateKarmaResults(baselineFailed) = %v", err)
		}
	})
}

func testPlanSingleTSChangedFileBranches(t *testing.T) {
	tmpRepo := t.TempDir()
	prodRel := "web/src/app/sample/item.ts"
	specRel := "web/src/app/sample/item.spec.ts"
	otherSpecRel := "web/src/app/other/other.spec.ts"
	writeTSTestFile(t, filepath.Join(tmpRepo, filepath.FromSlash(prodRel)), "export const ok = a == b;\nexport const next = c + d;\n")

	extMap := map[string]RawTSFileExtraction{
		prodRel: {
			Sites: []RawTSExtracted{
				{Kind: "eq_cmp", Op: "==", Operator: "ROR", Line: 1, StartOffset: 18, EndOffset: 24, LeftStart: 18, LeftEnd: 19, RightStart: 23, RightEnd: 24, Candidates: []RawTSCandidate{{OrigExpr: "a == b", MutExpr: "a != b", ReplacementPreview: "a != b"}}},
				{Kind: "arith", Op: "+", Operator: "AOR", Line: 2, StartOffset: 46, EndOffset: 51, LeftStart: 46, LeftEnd: 47, RightStart: 50, RightEnd: 51, Candidates: []RawTSCandidate{{OrigExpr: "c + d", MutExpr: "c - d", ReplacementPreview: "c - d"}}},
			},
		},
		"web/src/app/sample/deleted.ts": {},
	}
	opts := Options{
		RepoRoot:           tmpRepo,
		ChangedLinesByFile: map[string]map[int]struct{}{prodRel: {2: {}}},
	}

	// Missing extraction key returns nil without planning.
	emptyBundle := tsMutPlanBundle{mutantFileByID: make(map[int]string)}
	if err := planSingleTSChangedFile(opts, tsSingleFileCtx{relFile: "web/src/app/unknown.ts", extractions: extMap}, &emptyBundle); err != nil || len(emptyBundle.allPlans) != 0 {
		t.Errorf("missing extraction = (%v, %d)", err, len(emptyBundle.allPlans))
	}

	// Deleted file on disk returns nil without planning.
	if err := planSingleTSChangedFile(opts, tsSingleFileCtx{relFile: "web/src/app/sample/deleted.ts", extractions: extMap}, &emptyBundle); err != nil || len(emptyBundle.allPlans) != 0 {
		t.Errorf("deleted source file = (%v, %d)", err, len(emptyBundle.allPlans))
	}

	// Matching spec with changedSpecs > 0 allocates unchanged probe.
	mutID, probeID := 1, 1
	withProbe := tsMutPlanBundle{
		mutantFileByID: make(map[int]string),
		changedSpecs:   []redundancy.ChangedTSSpec{{File: specRel}},
	}
	if err := planSingleTSChangedFile(opts, tsSingleFileCtx{
		relFile: prodRel, extractions: extMap, changedSpecFiles: []string{specRel},
		nextMutID: &mutID, nextProbeID: &probeID,
	}, &withProbe); err != nil || probeID != 2 || withProbe.mutantFileByID[-1] != prodRel {
		t.Errorf("matching spec probe = (err=%v, probeID=%d, fileByID=%v)", err, probeID, withProbe.mutantFileByID)
	}

	// Matching spec file with zero changedSpecs skips unchanged probe.
	mutID2, probeID2 := 1, 1
	noChangedSpecs := tsMutPlanBundle{mutantFileByID: make(map[int]string)}
	if err := planSingleTSChangedFile(opts, tsSingleFileCtx{
		relFile: prodRel, extractions: extMap, changedSpecFiles: []string{specRel},
		nextMutID: &mutID2, nextProbeID: &probeID2,
	}, &noChangedSpecs); err != nil || probeID2 != 1 {
		t.Errorf("zero changedSpecs probeID = (%v, %d), want 1", err, probeID2)
	}

	// Non-matching spec file skips unchanged probe.
	mutID3, probeID3 := 1, 1
	nonMatchingSpec := tsMutPlanBundle{
		mutantFileByID: make(map[int]string),
		changedSpecs:   []redundancy.ChangedTSSpec{{File: otherSpecRel}},
	}
	if err := planSingleTSChangedFile(opts, tsSingleFileCtx{
		relFile: prodRel, extractions: extMap, changedSpecFiles: []string{otherSpecRel},
		nextMutID: &mutID3, nextProbeID: &probeID3,
	}, &nonMatchingSpec); err != nil || probeID3 != 1 {
		t.Errorf("non-matching spec probeID = (%v, %d), want 1", err, probeID3)
	}
}

func testRegisterPlannedTSFileBranches(t *testing.T) {
	out := tsMutPlanBundle{mutantFileByID: make(map[int]string)}
	suppressedPlan := &TSFilePlan{
		File:    "web/src/app/suppressed.ts",
		Mutants: []*TSMutant{{ID: 0, Status: rule.MutantSuppressed}},
	}
	registerPlannedTSFile(suppressedPlan.File, suppressedPlan, nil, &out)

	pendingMut := &TSMutant{ID: 1, File: "web/src/app/active.ts", Status: rule.MutantPending}
	activePlan := &TSFilePlan{
		File:    "web/src/app/active.ts",
		Mutants: []*TSMutant{pendingMut},
	}
	sites := []TSMutationSite{
		{Mutants: []*TSMutant{pendingMut, {ID: 0}, {ID: -1}}},
	}
	registerPlannedTSFile(activePlan.File, activePlan, sites, &out)

	if diff := cmp.Diff([]string{"web/src/app/active.ts"}, out.filesPending); diff != "" {
		t.Errorf("filesPending mismatch (-want +got):\n%s", diff)
	}
	if len(out.stagedFiles) != 1 || len(out.allPlans) != 2 {
		t.Errorf("staged=%d allPlans=%d, want 1 and 2", len(out.stagedFiles), len(out.allPlans))
	}
	wantByID := map[int]string{1: "web/src/app/active.ts", -1: "web/src/app/active.ts"}
	if diff := cmp.Diff(wantByID, out.mutantFileByID); diff != "" {
		t.Errorf("mutantFileByID mismatch (-want +got):\n%s", diff)
	}
}

func testRunTSPendingMutantsBranches(t *testing.T) {
	// Empty pendingMutants returns nil immediately.
	res, err := runTSPendingMutantsIfNeeded(context.Background(), Options{}, &tsMutPlanBundle{}, time.Minute)
	if err != nil || res != nil {
		t.Fatalf("empty pending = (%+v, %v), want (nil, nil)", res, err)
	}

	// Orphan file with no matching specs marks pending mutants uncovered without launching Karma.
	orphanRepo := t.TempDir()
	writeTSTestFile(t, filepath.Join(orphanRepo, "web", "src", "app", "orphan.ts"), "export const x = 1;\n")
	orphanMut := &TSMutant{ID: 1, File: "web/src/app/orphan.ts", Status: rule.MutantPending}
	orphanBundle := &tsMutPlanBundle{
		pendingMutants: []*TSMutant{orphanMut},
		filesPending:   []string{"web/src/app/orphan.ts"},
	}
	res, err = runTSPendingMutantsIfNeeded(context.Background(), Options{RepoRoot: orphanRepo}, orphanBundle, time.Minute)
	if err != nil || res != nil || orphanMut.Status != rule.MutantUncovered {
		t.Fatalf("orphan SkipKarma = (%+v, %v, status=%v)", res, err, orphanMut.Status)
	}

	// Default MaxMutants (80) and nil RunKarmaSession execute runKarmaMutationSession via hermetic npx.
	repoDir := setupMinimalMutantRepoWithSpec(t)
	script := `#!/bin/sh
node -e '
const fs = require("fs");
const cfg = JSON.parse(fs.readFileSync(process.env.KHI_MUTATION_CONFIG_FILE, "utf8"));
fs.writeFileSync(cfg.resultsPath, JSON.stringify({
  outcomes: {"1": String(cfg.maxMutants === 80 ? "KILLED" : "SURVIVED")}
}));
'
`
	fakeBin := writeExecutableScript(t, "npx", script)
	t.Setenv("USE_CONTAINER", "false")
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	mDefault := &TSMutant{ID: 1, File: "web/src/app/calc.ts", Status: rule.MutantPending}
	bundleDefault := newSingleMutantBundle(mDefault)
	if _, err := runTSPendingMutantsIfNeeded(context.Background(), Options{RepoRoot: repoDir}, bundleDefault, time.Minute); err != nil || mDefault.Status != rule.MutantKilled {
		t.Fatalf("default MaxMutants session = (err=%v, status=%v), want KILLED", err, mDefault.Status)
	}

	// Explicit MaxMutants (12) passes custom budget to plugin config.
	var gotCustomMax int
	mCustom := &TSMutant{ID: 1, File: "web/src/app/calc.ts", Status: rule.MutantPending}
	bundleCustom := newSingleMutantBundle(mCustom)
	_, err = runTSPendingMutantsIfNeeded(context.Background(), Options{
		RepoRoot:   repoDir,
		MaxMutants: 12,
		RunKarmaSession: func(_ context.Context, req karmaSessionRequest) error {
			raw, readErr := os.ReadFile(req.cfgPath)
			if readErr != nil {
				return readErr
			}
			var cfg karmaPluginConfig
			if unmarshalErr := json.Unmarshal(raw, &cfg); unmarshalErr != nil {
				return unmarshalErr
			}
			gotCustomMax = cfg.MaxMutants
			return os.WriteFile(cfg.ResultsPath, []byte(`{"outcomes":{"1":"KILLED"}}`), 0o600)
		},
	}, bundleCustom, time.Minute)
	if err != nil || gotCustomMax != 12 {
		t.Fatalf("custom MaxMutants = (%d, %v), want (12, nil)", gotCustomMax, err)
	}
}

func TestRecordTSBundleAndFileSortHelpers(t *testing.T) {
	rep := &rule.Report{
		Mutation: rule.MutationSummary{
			Enabled:    true,
			ByCategory: make(map[rule.AllowCategory]int),
		},
	}
	fp := &TSFilePlan{
		File:       "web/src/app/calc.ts",
		Suppressed: []rule.SuppressedItem{{Rule: rule.RuleFuncSurvivingMutants, File: "web/src/app/calc.ts", Line: 10, Category: rule.AllowCategory("perf")}},
		DirectiveErrors: []rule.DirectiveError{{
			File: "web/src/app/calc.ts", Line: 12, Message: "bad allow",
		}},
		Mutants: []*TSMutant{
			{ID: 1, File: "web/src/app/calc.ts", Line: 2, Operator: rule.OperatorROR, Status: rule.MutantKilled},
			{ID: 2, File: "web/src/app/calc.ts", Line: 3, Operator: rule.OperatorAOR, Status: rule.MutantTimedOut},
			{ID: 3, File: "web/src/app/calc.ts", Line: 4, Operator: rule.OperatorBCR, OriginalExpr: "ok", MutatedExpr: "false", Status: rule.MutantSurvived},
			{ID: 4, File: "web/src/app/calc.ts", Line: 5, Operator: rule.OperatorLCR, Status: rule.MutantUncovered},
			{ID: 0, File: "web/src/app/calc.ts", Line: 10, Operator: rule.OperatorBCR, SuppressionCategory: rule.AllowCategory("perf"), Status: rule.MutantSuppressed},
			{ID: 6, File: "web/src/app/calc.ts", Line: 11, Operator: rule.OperatorUOI, Status: rule.MutantSkippedBudget},
			{ID: 0, File: "web/src/app/calc.ts", Line: 12, Operator: rule.OperatorUOI, Status: rule.MutantSkippedBudget},
		},
	}
	cleanPlan := &TSFilePlan{
		File: "web/src/app/clean.ts",
		Mutants: []*TSMutant{
			{ID: 7, File: "web/src/app/clean.ts", Line: 2, Operator: rule.OperatorROR, Status: rule.MutantKilled},
		},
	}
	bundle := &tsMutPlanBundle{
		allPlans:       []*TSFilePlan{fp, cleanPlan},
		changedSpecs:   []redundancy.ChangedTSSpec{{File: "web/src/app/calc.spec.ts", FullName: "calc budget exempt", InDiff: true, WeakAssertion: true}},
		mutantFileByID: map[int]string{1: "web/src/app/calc.ts", 6: "web/src/app/calc.ts"},
	}
	recordTSBundleIntoReport(rep, bundle, &karmaPluginResults{
		SpecOrder: []redundancy.TSSpecOrderEntry{
			{ID: "s0", FullName: "calc base killer"},
			{ID: "s1", FullName: "calc budget exempt"},
		},
		SpecPolarities: map[string][]string{
			"s0": {"1:S", "6:S"},
			"s1": {"1:S", "6:S"},
		},
		MutantRuns: map[string]redundancy.TSMutantSpecRun{
			"1": {RanSpecIDs: []string{"s0", "s1"}, FailedSpecIDs: []string{"s0"}},
		},
	})

	wantMut := rule.MutationSummary{
		Enabled:       true,
		Killed:        2,
		Survived:      1,
		TimedOut:      1,
		Uncovered:     1,
		Suppressed:    1,
		SkippedBudget: 2,
		ByCategory:    map[rule.AllowCategory]int{rule.AllowCategory("perf"): 1},
	}
	if diff := cmp.Diff(wantMut, rep.Mutation); diff != "" {
		t.Errorf("MutationSummary mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(fp.Suppressed, rep.Suppressed); diff != "" {
		t.Errorf("Suppressed mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(fp.DirectiveErrors, rep.DirectiveErrors); diff != "" {
		t.Errorf("DirectiveErrors mismatch (-want +got):\n%s", diff)
	}
	if len(rep.Findings) != 1 || len(rep.Findings[0].SurvivingMutants) != 1 || rep.Findings[0].SurvivingMutants[0].Line != 4 {
		t.Errorf("unexpected surviving mutant findings: %+v", rep.Findings)
	}

	assertFileFilterAndSortHelpers(t)
}

func assertFileFilterAndSortHelpers(t *testing.T) {
	t.Helper()
	gotProd := sortedTSProdFiles(map[string]map[int]struct{}{
		"web/src/app/b.ts":      {1: {}},
		"web/src/app/empty.ts":  {},
		"web/src/app/a.spec.ts": {1: {}},
		"web/src/app/a.ts":      {2: {}},
	})
	if diff := cmp.Diff([]string{"web/src/app/a.ts", "web/src/app/b.ts"}, gotProd); diff != "" {
		t.Errorf("sortedTSProdFiles() mismatch (-want +got):\n%s", diff)
	}

	gotSpecs := sortedTSSpecFiles(map[string]map[int]struct{}{
		"web/src/app/b.spec.ts":     {1: {}},
		"web/src/app/empty.spec.ts": {},
		"web/src/app/a.ts":          {1: {}},
		"web/src/app/a.spec.ts":     {2: {}},
	})
	if diff := cmp.Diff([]string{"web/src/app/a.spec.ts", "web/src/app/b.spec.ts"}, gotSpecs); diff != "" {
		t.Errorf("sortedTSSpecFiles() mismatch (-want +got):\n%s", diff)
	}

	if !hasMatchingTSSpec("web/src/app/foo.ts", []string{"web/src/other/bar.spec.ts", "web/src/app/foo.spec.ts"}) {
		t.Errorf("hasMatchingTSSpec(matching) = false, want true")
	}
	if hasMatchingTSSpec("web/src/app/foo.ts", []string{"web/src/other/bar.spec.ts"}) {
		t.Errorf("hasMatchingTSSpec(non-matching) = true, want false")
	}

	sortRep := &rule.Report{
		Findings: []rule.Finding{
			{File: "b.ts", Line: 1, Rule: rule.RuleFuncUncoveredLines},
			{File: "a.ts", Line: 5, Rule: rule.RuleFuncUncoveredLines},
			{File: "a.ts", Line: 5, Rule: rule.RuleFuncSurvivingMutants},
		},
	}
	sortTSFindings(sortRep)
	assertFindingsRulesAndGlobalIndices(t, sortRep.Findings, []rule.RuleID{
		rule.RuleFuncSurvivingMutants,
		rule.RuleFuncUncoveredLines,
		rule.RuleFuncUncoveredLines,
	})
}

func assertFindingsRulesAndGlobalIndices(t *testing.T, findings []rule.Finding, wantRules []rule.RuleID) {
	t.Helper()
	var gotRules []rule.RuleID
	var gotIdx []int
	var wantIdx []int
	for i, f := range findings {
		gotRules = append(gotRules, f.Rule)
		gotIdx = append(gotIdx, f.GlobalIndex)
		wantIdx = append(wantIdx, i+1)
	}
	if diff := cmp.Diff(wantRules, gotRules); diff != "" {
		t.Errorf("Finding rules mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantIdx, gotIdx); diff != "" {
		t.Errorf("Finding GlobalIndex mismatch (-want +got):\n%s", diff)
	}
}

func setupMinimalMutantRepoWithSpec(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()
	writeTSTestFile(t, filepath.Join(repoDir, "web", "node_modules", ".keep"), "")
	writeTSTestFile(t, filepath.Join(repoDir, "web", "package.json"), "{}\n")
	writeTSTestFile(t, filepath.Join(repoDir, "web", "tsconfig.json"), "{}\n")
	writeTSTestFile(t, filepath.Join(repoDir, "web", "src", "app", "calc.ts"), "export const x = 1 + 2;\n")
	writeTSTestFile(t, filepath.Join(repoDir, "web", "src", "app", "calc.spec.ts"), "import './calc';\n")
	return repoDir
}

func newSingleMutantBundle(m *TSMutant) *tsMutPlanBundle {
	src := []byte("export const x = 1 + 2;\n")
	fp := &TSFilePlan{File: m.File, Source: src, Mutants: []*TSMutant{m}}
	sites := []TSMutationSite{{
		Kind: "arith", Op: "+", StartOffset: 17, EndOffset: 22, LeftStart: 17, LeftEnd: 18, RightStart: 21, RightEnd: 22,
		Mutants: []*TSMutant{m},
	}}
	return &tsMutPlanBundle{
		allPlans:       []*TSFilePlan{fp},
		stagedFiles:    []tsStagedFile{{plan: fp, tsSites: sites}},
		pendingMutants: []*TSMutant{m},
		filesPending:   []string{m.File},
		mutantFileByID: map[int]string{m.ID: m.File},
		maxMutantID:    m.ID,
	}
}

func writeFakeKarmaPluginResults(cfgPath string) error {
	cfgBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}
	var cfg karmaPluginConfig
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		return err
	}
	res := karmaPluginResults{
		Outcomes: map[string]string{
			"1": "SURVIVED",
			"2": "KILLED",
			"3": "KILLED",
		},
		SpecOrder: []redundancy.TSSpecOrderEntry{
			{ID: "spec0", FullName: "calc handles positive one"},
			{ID: "spec1", FullName: "calc handles positive five duplicate"},
		},
		SpecPolarities: map[string][]string{
			"spec0": {"1:F", "2:F", "3:S"},
			"spec1": {"1:F", "2:F", "3:S"},
		},
		MutantRuns: map[string]redundancy.TSMutantSpecRun{
			"3": {RanSpecIDs: []string{"spec0", "spec1"}, FailedSpecIDs: []string{"spec0", "spec1"}},
		},
	}
	outBytes, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.ResultsPath, outBytes, 0o600)
}
