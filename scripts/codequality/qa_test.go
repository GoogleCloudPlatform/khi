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
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/report"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestCheckAndQAConventionViolationsAndEnvelopes verifies fast convention violations under check/qa and comment-envelope/summary emission under qa.
func TestCheckAndQAConventionViolationsAndEnvelopes(t *testing.T) {
	cleanRepo := initTempGitRepo(t)
	summaryPath := filepath.Join(t.TempDir(), "qa_summary.md")
	var cleanOut, cleanErr bytes.Buffer
	code, err := runCLI(context.Background(), cliEnv{
		args: []string{
			"qa", "--repo-root=" + cleanRepo, "--vcs=git", "--base=HEAD~1",
			"--speed=fast", "--filter=must",
			"--skip-sources=addlicense,gofmt,golangci_lint,go_test_short",
			"--comment-envelope=true", "--summary-file=" + summaryPath, "--color=never",
		},
		stdout: &cleanOut,
		stderr: &cleanErr,
		getenv: func(string) string { return "" },
	})
	if err != nil || code != 0 {
		t.Fatalf("runCLI(qa clean) = (%d, %v), stderr=%s", code, err, cleanErr.String())
	}
	decoded, found, decErr := report.ExtractCommentEnvelope(cleanOut.String())
	if decErr != nil || !found || !strings.Contains(decoded, "Go Quality Report") {
		t.Errorf("ExtractCommentEnvelope() = (found=%v, err=%v), decoded=%q", found, decErr, decoded)
	}
	if sumBytes, readErr := os.ReadFile(summaryPath); readErr != nil || !strings.Contains(string(sumBytes), "Go Quality Report") {
		t.Errorf("summary file = (%q, %v), want Go Quality Report", string(sumBytes), readErr)
	}

	var allOut, allErr bytes.Buffer
	allCode, allRunErr := runCLI(context.Background(), cliEnv{
		args: []string{
			"qa", "--repo-root=" + cleanRepo, "--vcs=git", "--base=HEAD~1",
			"--speed=all", "--filter=all", "--decisions", "--similarity",
			"--skip-sources=addlicense,gofmt,golangci_lint,go_test_short,unused_exports,deadcode,go_test_race,build_go,decisions,similarity",
			"--format=json",
		},
		stdout: &allOut,
		stderr: &allErr,
		getenv: func(string) string { return "" },
	})
	if allRunErr != nil || allCode != 0 || !strings.Contains(allOut.String(), `"gofmt"`) || !strings.Contains(allOut.String(), `"decisions"`) || !strings.Contains(allOut.String(), `"similarity"`) {
		t.Errorf("runCLI(qa all) = (%d, %v), want code 0 with gofmt, decisions, and similarity in skipped_sources:\n%s", allCode, allRunErr, allOut.String())
	}

	qaDiff, diffErr := collectScopedDiff(context.Background(), cliFlags{subcommand: subcmdQA, vcs: "git", base: "HEAD~1"}, cleanRepo)
	if diffErr != nil {
		t.Fatalf("collectScopedDiff(cleanRepo) error: %v", diffErr)
	}
	var phaseErr bytes.Buffer
	phaseReports, _, phaseRunErr := evaluateQASubcommand(context.Background(), checkStagesRequest{
		flags: cliFlags{
			subcommand:  subcmdQA,
			lang:        "go",
			vcs:         "git",
			base:        "HEAD~1",
			speed:       "all",
			filter:      "all",
			decisions:   true,
			similarity:  true,
			skipSources: "addlicense,gofmt,golangci_lint,go_test_short,unused_exports,deadcode,go_test_race,build_go,decisions,similarity",
		},
		speed:    rule.SpeedAll,
		filter:   rule.FilterAll,
		repoRoot: cleanRepo,
		diffRes:  qaDiff,
		progress: newProgressReporter(cliEnv{stderr: &phaseErr, getenv: func(string) string { return "" }}),
		env:      cliEnv{stderr: &phaseErr, getenv: func(string) string { return "" }},
	})
	if phaseRunErr != nil || len(phaseReports) < 3 {
		t.Fatalf("evaluateQASubcommand(all) = (%d reports, %v), want >= 3", len(phaseReports), phaseRunErr)
	}
	if !phaseReports[0].FastMode || phaseReports[0].Similarity.Enabled {
		t.Errorf("phaseReports[0] = (FastMode=%v, Similarity.Enabled=%v), want (true, false)", phaseReports[0].FastMode, phaseReports[0].Similarity.Enabled)
	}

	// Verify evaluateUnusedExportsForQA reports unused exports, respects path filters, and propagates repo root errors.
	unusedRepo := t.TempDir()
	writeFiles(t, unusedRepo, map[string]string{
		"go.mod":            "module example.com/unusedmod\n\ngo 1.24\n",
		"pkg/a/unused_a.go": "package a\n\n// UnusedA is never referenced.\nconst UnusedA = 1\n",
		"pkg/b/unused_b.go": "package b\n\n// UnusedB is never referenced.\nconst UnusedB = 2\n",
	})
	allUnused, unusedErr := evaluateUnusedExportsForQA(unusedRepo, nil)
	if unusedErr != nil || len(allUnused) != 2 || allUnused[0].Source != "unused_exports" || allUnused[0].File != "pkg/a/unused_a.go" || allUnused[0].Line == 0 {
		t.Fatalf("evaluateUnusedExportsForQA(all) = (%+v, %v), want 2 unused_exports items", allUnused, unusedErr)
	}
	filteredUnused, filterErr := evaluateUnusedExportsForQA(unusedRepo, []string{"pkg/b"})
	wantFilteredID := "unused_exports:unused-exported-decl:pkg/b/unused_b.go:4:1"
	if filterErr != nil || len(filteredUnused) != 1 || filteredUnused[0].File != "pkg/b/unused_b.go" || filteredUnused[0].ID != wantFilteredID || filteredUnused[0].Summary == "" || filteredUnused[0].Detail == "" {
		t.Fatalf("evaluateUnusedExportsForQA(pkg/b) = (%+v, %v), want 1 item with ID %q", filteredUnused, filterErr, wantFilteredID)
	}
	catchAllUnused, catchAllErr := evaluateUnusedExportsForQA(unusedRepo, []string{"."})
	if catchAllErr != nil || len(catchAllUnused) != 2 {
		t.Fatalf("evaluateUnusedExportsForQA(.) = (%+v, %v), want 2 items", catchAllUnused, catchAllErr)
	}
	if _, badErr := evaluateUnusedExportsForQA(t.TempDir(), nil); badErr == nil {
		t.Errorf("evaluateUnusedExportsForQA(missing go.mod) = nil, want error")
	}

	convRepo := initTempGitRepo(t)
	writeFiles(t, convRepo, map[string]string{
		"pkg/svc/badconv.go": "package svc\n\n// BadConv lacks GoDoc on Missing.\ntype BadConv struct {\n\tMissing int\n}\n",
	})
	runGitCmd(t, convRepo, "add", ".")
	runGitCmd(t, convRepo, "commit", "-m", "add convention violation")

	for _, sub := range []string{"check", "qa"} {
		var outBuf, errBuf bytes.Buffer
		args := []string{
			sub, "--repo-root=" + convRepo, "--vcs=git", "--base=HEAD~1",
			"--speed=fast", "--filter=must", "--color=never",
		}
		if sub == "qa" {
			args = append(args, "--skip-sources=addlicense,gofmt,golangci_lint,go_test_short")
		}
		c, runErr := runCLI(context.Background(), cliEnv{
			args:   args,
			stdout: &outBuf,
			stderr: &errBuf,
			getenv: func(string) string { return "" },
		})
		if runErr != nil || c != 1 || !strings.Contains(outBuf.String(), "exported-member-doc") {
			t.Errorf("runCLI(%s) = (%d, %v), want code 1 with exported-member-doc in:\n%s", sub, c, runErr, outBuf.String())
		}
	}
}

// TestPipelineStageGatingAndASTCloneScan verifies collectScopedDiff CurrentOnly, executeCheckStages decisions gating, executeGoPipeline/runGoMustStages, executeTSPipeline, and runASTStage clone scanning.
func TestPipelineStageGatingAndASTCloneScan(t *testing.T) {
	ctx := context.Background()
	repoDir := initTempGitRepo(t)
	qaDiff, err := collectScopedDiff(ctx, cliFlags{subcommand: subcmdQA, vcs: "git"}, repoDir)
	if err != nil || qaDiff.BaseLabel != "HEAD~1" {
		t.Fatalf("collectScopedDiff(qa) = (%q, %v), want BaseLabel HEAD~1", qaDiff.BaseLabel, err)
	}
	checkDiff, err := collectScopedDiff(ctx, cliFlags{subcommand: subcmdCheck, vcs: "git"}, repoDir)
	if err != nil || checkDiff.BaseLabel != "main" {
		t.Fatalf("collectScopedDiff(check) = (%q, %v), want BaseLabel main", checkDiff.BaseLabel, err)
	}

	// Verify executeCheckStages skips decisions unless both fast and advisory are active.
	missingRules := filepath.Join(t.TempDir(), "missing-rules")
	for _, pair := range []struct {
		speed  rule.Speed
		filter rule.Filter
	}{
		{rule.SpeedFast, rule.FilterMust},
		{rule.SpeedSlow, rule.FilterAdvisory},
	} {
		var errBuf bytes.Buffer
		_, stageErr := executeCheckStages(ctx, checkStagesRequest{
			flags:    cliFlags{lang: "ts", speed: "all", filter: "all", decisions: true, decisionsRulesDir: missingRules},
			speed:    pair.speed,
			filter:   pair.filter,
			repoRoot: repoDir,
			diffRes:  gotestutil.DiffResult{BaseLabel: "main", ChangedFiles: []string{"pkg/svc/svc.go"}},
			progress: newProgressReporter(cliEnv{stderr: &errBuf, getenv: func(string) string { return "" }}),
			env:      cliEnv{stderr: &errBuf, getenv: func(string) string { return "" }},
		})
		if stageErr != nil {
			t.Fatalf("executeCheckStages(%s, %s) unexpected error: %v", pair.speed, pair.filter, stageErr)
		}
	}

	// Verify executeGoPipeline fast vs slow, FuncsEvaluated, Suppressed, DirectiveErrors, and clone scanning.
	cloneRepo := initTempGitRepo(t)
	writeFiles(t, cloneRepo, map[string]string{
		"pkg/other/other.go": `package other

// DupB computes a multi-step sum in the base commit.
func DupB(a, b, c int) int {
	total := 0
	step1 := a + 1
	step2 := b + 2
	step3 := c + 3
	step4 := step1 * 2
	step5 := step2 * 3
	step6 := step3 * 4
	total += step4 + step5 + step6
	for i := 0; i < c; i++ {
		v := a + i
		if v%2 == 0 {
			total += v * b
		} else {
			total += v + b
		}
		if total > 100 {
			total -= 7
		}
	}
	if total < 0 {
		return -total
	}
	return total
}
`,
	})
	runGitCmd(t, cloneRepo, "add", ".")
	runGitCmd(t, cloneRepo, "commit", "-m", "add base clone target")

	writeFiles(t, cloneRepo, map[string]string{
		"pkg/svc/dup.go": `package svc

// DupA computes a multi-step sum matching pkg/other.
func DupA(a, b, c int) int {
	total := 0
	step1 := a + 1
	step2 := b + 2
	step3 := c + 3
	step4 := step1 * 2
	step5 := step2 * 3
	step6 := step3 * 4
	total += step4 + step5 + step6
	for i := 0; i < c; i++ {
		v := a + i
		if v%2 == 0 {
			total += v * b
		} else {
			total += v + b
		}
		if total > 100 {
			total -= 7
		}
	}
	if total < 0 {
		return -total
	}
	return total
}

// BadDoc lacks GoDoc on Field.
type BadDoc struct {
	Field int
}

// CODEQUALITY_ALLOW(func_params, protocol): verified intentional.
func SuppressedParams(a, b, c, d, e, f int) int { return a + b + c + d + e + f }

// CODEQUALITY_ALLOW(func_params, protocol): unused directive.
func UnusedDirective(x int) int { return x }

// CODEQUALITY_ALLOW(unknown_rule, protocol): invalid rule.
func MalformedDirective(x int) int { return x }
`,
	})
	runGitCmd(t, cloneRepo, "add", ".")
	runGitCmd(t, cloneRepo, "commit", "-m", "add clone, suppressed, and convention")

	diffRes, err := gotestutil.CollectChangedLines(ctx, cloneRepo, gotestutil.DiffOptions{VCS: "git", Base: "HEAD~1"})
	if err != nil {
		t.Fatalf("CollectChangedLines() error: %v", err)
	}
	fastAST, err := runASTStage(ctx, cloneRepo, diffRes, rule.SpeedFast)
	if err != nil {
		t.Fatalf("runASTStage(fast) error: %v", err)
	}
	slowAST, err := runASTStage(ctx, cloneRepo, diffRes, rule.SpeedSlow)
	if err != nil {
		t.Fatalf("runASTStage(slow) error: %v", err)
	}
	if len(slowAST.Findings) <= len(fastAST.Findings) {
		t.Errorf("slowAST findings (%d) <= fastAST findings (%d), expected clone finding in slow", len(slowAST.Findings), len(fastAST.Findings))
	}
	notADirFile := filepath.Join(t.TempDir(), "not-a-dir")
	if writeErr := os.WriteFile(notADirFile, []byte("x"), 0o600); writeErr != nil {
		t.Fatalf("WriteFile(not-a-dir) error: %v", writeErr)
	}
	invalidRoot := filepath.Join(notADirFile, "sub")
	if _, err := runASTStage(ctx, invalidRoot, gotestutil.DiffResult{}, rule.SpeedSlow); err != nil {
		t.Errorf("runASTStage(empty diff, invalid root) unexpected error: %v", err)
	}
	if _, err := runASTStage(ctx, invalidRoot, diffRes, rule.SpeedSlow); err == nil {
		t.Errorf("runASTStage(non-empty diff, invalid root) error = nil, want ENOTDIR error")
	}

	var goFastErrBuf bytes.Buffer
	fastProg := newProgressReporter(cliEnv{stderr: &goFastErrBuf, getenv: func(string) string { return "" }})
	fastPenv := pipelineEnv{
		flags:    cliFlags{lang: "go"},
		repoRoot: cloneRepo,
		diffRes:  diffRes,
		progress: fastProg,
		errOut:   &goFastErrBuf,
	}
	goFastRep, err := executeGoPipeline(ctx, fastPenv, rule.SpeedFast, rule.FilterMust)
	if err != nil {
		t.Fatalf("executeGoPipeline(fast must) error: %v", err)
	}
	if !goFastRep.FastMode || goFastRep.Coverage.Enabled || goFastRep.Mutation.Enabled {
		t.Errorf("goFastRep = (FastMode=%v, Coverage.Enabled=%v, Mutation.Enabled=%v), want (true, false, false)", goFastRep.FastMode, goFastRep.Coverage.Enabled, goFastRep.Mutation.Enabled)
	}
	if !strings.Contains(goFastErrBuf.String(), "[2/2]") {
		t.Errorf("stderr missing [2/2] progress in fast mode: %q", goFastErrBuf.String())
	}

	var goErrBuf bytes.Buffer
	prog := newProgressReporter(cliEnv{stderr: &goErrBuf, getenv: func(string) string { return "" }})
	penv := pipelineEnv{
		flags:      cliFlags{lang: "go", coverprofile: ""},
		repoRoot:   cloneRepo,
		diffRes:    diffRes,
		progress:   prog,
		errOut:     &goErrBuf,
		simSession: newSimilaritySession(ctx, cliFlags{speed: "slow", filter: "advisory", similarity: true}, nil, nil),
	}
	goSlowRep, err := executeGoPipeline(ctx, penv, rule.SpeedSlow, rule.FilterMust)
	if err != nil {
		t.Fatalf("executeGoPipeline(slow must) error: %v", err)
	}
	if goSlowRep.FastMode || goSlowRep.FuncsEvaluated == 0 || !goSlowRep.Coverage.Enabled || len(goSlowRep.DirectiveErrors) < 2 || len(goSlowRep.Suppressed) == 0 {
		t.Errorf("goSlowRep = %+v, want FastMode=false, FuncsEvaluated>0, Coverage.Enabled=true, DirectiveErrors>=2, Suppressed>0", goSlowRep)
	}
	for _, f := range goSlowRep.Findings {
		if f.Rule == "exported-member-doc" {
			t.Errorf("slow-only Go pipeline unexpectedly ran fast convention check: %+v", f)
		}
	}
	if !strings.Contains(goErrBuf.String(), "[2/4]") {
		t.Errorf("stderr missing [2/4] progress in slow mode: %q", goErrBuf.String())
	}

	goFastAdvRep, err := executeGoPipeline(ctx, penv, rule.SpeedFast, rule.FilterAdvisory)
	if err != nil {
		t.Fatalf("executeGoPipeline(fast advisory) error: %v", err)
	}
	if !goFastAdvRep.FastMode || goFastAdvRep.FuncsEvaluated != 0 || goFastAdvRep.Similarity.Enabled {
		t.Errorf("goFastAdvRep = %+v, want FastMode=true, FuncsEvaluated=0, Similarity.Enabled=false", goFastAdvRep)
	}

	goSlowAdvRep, err := executeGoPipeline(ctx, penv, rule.SpeedSlow, rule.FilterAdvisory)
	if err != nil {
		t.Fatalf("executeGoPipeline(slow advisory) error: %v", err)
	}
	if !goSlowAdvRep.Similarity.Enabled {
		t.Errorf("goSlowAdvRep.Similarity.Enabled = false, want true")
	}

	tsCovPath := filepath.Join(t.TempDir(), "ts-cov.json")
	if err := os.WriteFile(tsCovPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("WriteFile(tsCovPath) error: %v", err)
	}
	writeFiles(t, cloneRepo, map[string]string{
		"web/package.json": `{"name":"test-web"}`,
		"web/src/a.ts":     "export function hello(): number { return 1; }\n",
	})
	tsDiff := gotestutil.DiffResult{BaseLabel: "HEAD~1"}
	if realRoot, rootErr := gotestutil.ResolveRepoRoot(""); rootErr == nil {
		if _, statErr := os.Stat(filepath.Join(realRoot, "web", "node_modules", "typescript")); statErr == nil {
			if symErr := os.Symlink(filepath.Join(realRoot, "web", "node_modules"), filepath.Join(cloneRepo, "web", "node_modules")); symErr != nil {
				t.Fatalf("Symlink(node_modules) error: %v", symErr)
			}
			tsDiff.TSChangedLines = map[string]map[int]struct{}{"web/src/a.ts": {1: {}}}
		}
	}
	tsPenv := pipelineEnv{
		flags:      cliFlags{lang: "ts", tsCoverage: tsCovPath},
		repoRoot:   cloneRepo,
		diffRes:    tsDiff,
		progress:   prog,
		errOut:     &goErrBuf,
		simSession: newSimilaritySession(ctx, cliFlags{speed: "slow", filter: "advisory", similarity: true, skipSources: "similarity"}, nil, nil),
	}
	tsMustRep, err := executeTSPipeline(ctx, tsPenv, rule.FilterMust)
	if err != nil {
		t.Fatalf("executeTSPipeline(must) error: %v", err)
	}
	if !tsMustRep.Coverage.Enabled || tsMustRep.Similarity.Enabled {
		t.Errorf("tsMustRep = %+v, want Coverage.Enabled=true and Similarity.Enabled=false", tsMustRep)
	}
	tsAdvRep, err := executeTSPipeline(ctx, tsPenv, rule.FilterAdvisory)
	if err != nil {
		t.Fatalf("executeTSPipeline(advisory) error: %v", err)
	}
	if tsAdvRep.Coverage.Enabled || !tsAdvRep.Advisory || (len(tsDiff.TSChangedLines) > 0 && !tsAdvRep.Similarity.Enabled) {
		t.Errorf("tsAdvRep = %+v, want Coverage.Enabled=false, Advisory=true, Similarity.Enabled=%v", tsAdvRep, len(tsDiff.TSChangedLines) > 0)
	}
}
