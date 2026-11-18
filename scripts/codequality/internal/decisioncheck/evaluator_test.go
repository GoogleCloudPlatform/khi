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

package decisioncheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type fakeDecider struct {
	calls int
	resp  DecisionResponse
}

func (f *fakeDecider) Decide(_ context.Context, _ DecisionRequest) (DecisionResponse, error) {
	f.calls++
	return f.resp, nil
}

// TestEvaluateNumbersBlockingFindings verifies that uncached evaluation invokes the decider once and numbers findings deterministically.
func TestEvaluateNumbersBlockingFindings(t *testing.T) {
	ctx := context.Background()
	repo, spec := setupAlphaRuleFixture(t)
	cache := openTestMemoryCache(t, ctx)
	pLow := 0.10
	dec := &fakeDecider{
		resp: DecisionResponse{
			Answers: map[string]WireAnswer{
				"unit_0": {Noul: &pLow, Confidence: 0.90},
			},
		},
	}
	var resolverCalls int
	opts := Options{
		Rules:         []RuleSpec{spec},
		ForceBlocking: true,
		TargetOpts: ResolveOptions{
			RepoRoot:     repo,
			ChangedFiles: []string{"pkg/alpha/alpha.go"},
		},
		Cache: cache,
		ResolveDecider: func(context.Context) (Decider, string, error) {
			resolverCalls++
			return dec, "", nil
		},
	}

	gotSummary, evalErr := Evaluate(ctx, opts)

	if evalErr != nil {
		t.Fatalf("Evaluate() unexpected error: %v", evalErr)
	}
	wantSummary := rule.DecisionSummary{
		Enabled:            true,
		RulesEvaluated:     1,
		TargetsEvaluated:   1,
		QuestionsEvaluated: 1,
		CacheHits:          0,
		APICalls:           1,
		Findings: []rule.DecisionItemFinding{
			{
				Index:      1,
				RuleID:     "pkg_doc_alignment",
				RuleName:   "Package Doc Alignment",
				QuestionID: "unit",
				Blocking:   true,
				Target:     "pkg/alpha",
				File:       "pkg/alpha/alpha.go",
				Line:       1,
				EndLine:    3,
				Subject:    "alpha.go (Engine)",
				Outcome:    "p_true=0.10 (want true >= 0.50, conf=0.90)",
				Why:        "alpha.go (Engine) missing from README.md.",
				Fix:        "Document alpha.go (Engine) in README.md.",
			},
		},
	}
	if diff := cmp.Diff(wantSummary, gotSummary); diff != "" {
		t.Errorf("Evaluate() mismatch (-want +got):\n%s", diff)
	}
	if dec.calls != 1 || resolverCalls != 1 {
		t.Errorf("call counts = (decCalls=%d, resolverCalls=%d), want (1, 1)", dec.calls, resolverCalls)
	}
}

// TestEvaluateServesWarmSQLiteCacheWithoutDecider verifies that a warm SQLite cache entry satisfies Evaluate with zero API calls even when ResolveDecider is nil.
func TestEvaluateServesWarmSQLiteCacheWithoutDecider(t *testing.T) {
	ctx := context.Background()
	repo, spec := setupAlphaRuleFixture(t)
	cache := openTestMemoryCache(t, ctx)
	pLow := 0.10
	dec := &fakeDecider{
		resp: DecisionResponse{
			Answers: map[string]WireAnswer{
				"unit_0": {Noul: &pLow, Confidence: 0.90},
			},
		},
	}
	seedOpts := Options{
		Rules:         []RuleSpec{spec},
		ForceBlocking: true,
		TargetOpts: ResolveOptions{
			RepoRoot:     repo,
			ChangedFiles: []string{"pkg/alpha/alpha.go"},
		},
		Cache: cache,
		ResolveDecider: func(context.Context) (Decider, string, error) {
			return dec, "", nil
		},
	}
	if _, err := Evaluate(ctx, seedOpts); err != nil {
		t.Fatalf("seed Evaluate() unexpected error: %v", err)
	}
	cachedOpts := seedOpts
	cachedOpts.ResolveDecider = nil

	gotSummary, evalErr := Evaluate(ctx, cachedOpts)

	if evalErr != nil {
		t.Fatalf("Evaluate(cached) unexpected error: %v", evalErr)
	}
	if gotSummary.CacheHits != 1 || gotSummary.APICalls != 0 || dec.calls != 1 {
		t.Errorf("cached stats = (hits=%d, api=%d, decCalls=%d), want (1, 0, 1)", gotSummary.CacheHits, gotSummary.APICalls, dec.calls)
	}
}

// TestEvaluateSkipsGracefullyWithoutDecider verifies that uncached runs with a nil ResolveDecider return Skipped=true with a non-empty skip reason.
func TestEvaluateSkipsGracefullyWithoutDecider(t *testing.T) {
	ctx := context.Background()
	repo, spec := setupAlphaRuleFixture(t)

	gotSummary, evalErr := Evaluate(ctx, Options{
		Rules: []RuleSpec{spec},
		TargetOpts: ResolveOptions{
			RepoRoot:     repo,
			ChangedFiles: []string{"pkg/alpha/alpha.go"},
		},
	})

	if evalErr != nil {
		t.Fatalf("Evaluate() unexpected error: %v", evalErr)
	}
	if !gotSummary.Skipped || gotSummary.SkipReason == "" {
		t.Errorf("gotSummary = (Skipped=%v, SkipReason=%q), want Skipped=true with reason", gotSummary.Skipped, gotSummary.SkipReason)
	}
}

// TestEvaluateNoMatchingRules verifies that Evaluate returns a skipped summary when no enabled rules match.
func TestEvaluateNoMatchingRules(t *testing.T) {
	ctx := context.Background()
	repo, spec := setupAlphaRuleFixture(t)
	spec.Enabled = false

	gotSummary, evalErr := Evaluate(ctx, Options{
		Rules: []RuleSpec{spec},
		TargetOpts: ResolveOptions{
			RepoRoot:     repo,
			ChangedFiles: []string{"pkg/alpha/alpha.go"},
		},
	})

	if evalErr != nil {
		t.Fatalf("Evaluate() unexpected error: %v", evalErr)
	}
	wantSummary := rule.DecisionSummary{
		Enabled:    true,
		Skipped:    true,
		SkipReason: "no enabled decision rules matched",
	}
	if diff := cmp.Diff(wantSummary, gotSummary); diff != "" {
		t.Errorf("Evaluate() mismatch (-want +got):\n%s", diff)
	}
}

func TestSelectActiveRules(t *testing.T) {
	enabledRule := RuleSpec{ID: "enabled_rule", Enabled: true}
	disabledRule := RuleSpec{ID: "disabled_rule", Enabled: false}
	rules := []RuleSpec{enabledRule, disabledRule}
	testCases := []struct {
		name   string
		filter map[string]struct{}
		want   []RuleSpec
	}{
		// Includes only Enabled rules when RuleFilter is empty.
		{
			name:   "empty filter keeps enabled rules only",
			filter: nil,
			want:   []RuleSpec{enabledRule},
		},
		// Selects explicitly filtered rule IDs even when Enabled is false and excludes unmatched rules.
		{
			name:   "explicit filter selects matching rule regardless of Enabled",
			filter: map[string]struct{}{"disabled_rule": {}},
			want:   []RuleSpec{disabledRule},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := selectActiveRules(rules, tc.filter)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("selectActiveRules() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEvaluateZeroTargetsAndResolverSkip(t *testing.T) {
	ctx := context.Background()
	repo, spec := setupAlphaRuleFixture(t)
	testCases := []struct {
		name              string
		changedFiles      []string
		resolverReason    string
		wantSummary       rule.DecisionSummary
		wantResolverCalls int
	}{
		// Returns zero targets evaluated without invoking ResolveDecider when no repository targets match.
		{
			name:         "no matching targets skips evaluation cleanly",
			changedFiles: []string{"docs/unrelated.md"},
			wantSummary: rule.DecisionSummary{
				Enabled:        true,
				RulesEvaluated: 1,
			},
			wantResolverCalls: 0,
		},
		// Propagates the resolver skip reason when ResolveDecider returns a nil decider without error.
		{
			name:           "nil decider from resolver returns skipped summary",
			changedFiles:   []string{"pkg/alpha/alpha.go"},
			resolverReason: "missing ADC credentials",
			wantSummary: rule.DecisionSummary{
				Enabled:    true,
				Skipped:    true,
				SkipReason: "missing ADC credentials",
			},
			wantResolverCalls: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var resolverCalls int
			opts := Options{
				Rules: []RuleSpec{spec},
				TargetOpts: ResolveOptions{
					RepoRoot:     repo,
					ChangedFiles: tc.changedFiles,
				},
				ResolveDecider: func(context.Context) (Decider, string, error) {
					resolverCalls++
					return nil, tc.resolverReason, nil
				},
			}

			gotSummary, evalErr := Evaluate(ctx, opts)

			if evalErr != nil {
				t.Fatalf("Evaluate() unexpected error: %v", evalErr)
			}
			if diff := cmp.Diff(tc.wantSummary, gotSummary); diff != "" {
				t.Errorf("Evaluate() mismatch (-want +got):\n%s", diff)
			}
			if resolverCalls != tc.wantResolverCalls {
				t.Errorf("ResolveDecider calls = %d, want %d", resolverCalls, tc.wantResolverCalls)
			}
		})
	}
}

func TestEvaluateErrorPropagation(t *testing.T) {
	ctx := context.Background()
	repo, spec := setupAlphaRuleFixture(t)
	testCases := []struct {
		name              string
		targetKind        TargetKind
		closeCache        bool
		resolverErr       error
		wantErrSubstr     string
		wantResolverCalls int
	}{
		// Propagates target-resolution errors before querying cache or resolving a decider.
		{
			name:              "unsupported target kind propagates ResolveTargets error",
			targetKind:        TargetKind("unknown"),
			wantErrSubstr:     "unsupported target kind",
			wantResolverCalls: 0,
		},
		// Propagates SQLite lookup errors from a closed cache without invoking ResolveDecider.
		{
			name:              "closed cache lookup propagates error",
			closeCache:        true,
			wantErrSubstr:     "closed",
			wantResolverCalls: 0,
		},
		// Propagates errors returned by ResolveDecider even if a non-nil decider is also returned.
		{
			name:              "ResolveDecider error propagates",
			resolverErr:       errors.New("credential probe failed"),
			wantErrSubstr:     "credential probe failed",
			wantResolverCalls: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := openTestMemoryCache(t, ctx)
			if tc.closeCache {
				_ = cache.Close()
			}
			specCopy := spec
			if tc.targetKind != "" {
				specCopy.Target.Kind = tc.targetKind
			}
			var resolverCalls int
			opts := Options{
				Rules: []RuleSpec{specCopy},
				TargetOpts: ResolveOptions{
					RepoRoot:     repo,
					ChangedFiles: []string{"pkg/alpha/alpha.go"},
				},
				Cache: cache,
				ResolveDecider: func(context.Context) (Decider, string, error) {
					resolverCalls++
					return &fakeDecider{}, "", tc.resolverErr
				},
			}

			_, evalErr := Evaluate(ctx, opts)

			if evalErr == nil || !strings.Contains(evalErr.Error(), tc.wantErrSubstr) {
				t.Errorf("Evaluate() error = %v, want substring %q", evalErr, tc.wantErrSubstr)
			}
			if resolverCalls != tc.wantResolverCalls {
				t.Errorf("ResolveDecider calls = %d, want %d", resolverCalls, tc.wantResolverCalls)
			}
		})
	}
}

// TestSortAndNumberFindings verifies deterministic multi-key sorting across all tie-breakers and 1-based Index assignment.
func TestSortAndNumberFindings(t *testing.T) {
	findings := []rule.DecisionItemFinding{
		{File: "b.go", Line: 1, RuleID: "r1", QuestionID: "q1", Subject: "s1"},
		{File: "a.go", Line: 20, RuleID: "r1", QuestionID: "q1", Subject: "s1"},
		{File: "a.go", Line: 10, RuleID: "r2", QuestionID: "q1", Subject: "s1"},
		{File: "a.go", Line: 10, RuleID: "r1", QuestionID: "q2", Subject: "s1"},
		{File: "a.go", Line: 10, RuleID: "r1", QuestionID: "q1", Subject: "s2"},
		{File: "a.go", Line: 10, RuleID: "r1", QuestionID: "q1", Subject: "s1"},
	}

	sortAndNumberFindings(findings)

	want := []rule.DecisionItemFinding{
		{Index: 1, File: "a.go", Line: 10, RuleID: "r1", QuestionID: "q1", Subject: "s1"},
		{Index: 2, File: "a.go", Line: 10, RuleID: "r1", QuestionID: "q1", Subject: "s2"},
		{Index: 3, File: "a.go", Line: 10, RuleID: "r1", QuestionID: "q2", Subject: "s1"},
		{Index: 4, File: "a.go", Line: 10, RuleID: "r2", QuestionID: "q1", Subject: "s1"},
		{Index: 5, File: "a.go", Line: 20, RuleID: "r1", QuestionID: "q1", Subject: "s1"},
		{Index: 6, File: "b.go", Line: 1, RuleID: "r1", QuestionID: "q1", Subject: "s1"},
	}
	if diff := cmp.Diff(want, findings); diff != "" {
		t.Errorf("sortAndNumberFindings() mismatch (-want +got):\n%s", diff)
	}
}

func setupAlphaRuleFixture(t *testing.T) (string, RuleSpec) {
	t.Helper()
	repo := t.TempDir()
	pkgDir := filepath.Join(repo, "pkg", "alpha")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "README.md"), []byte("# Alpha\n\nOverview.\n"), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "alpha.go"), []byte("package alpha\n\ntype Engine struct{}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	spec, err := ParseRuleYAML([]byte(`
id: pkg_doc_alignment
name: Package Doc Alignment
target:
  kind: go_logical_package
  include: ["pkg/**"]
  require_files: ["README.md"]
questions:
  - id: unit
    for_each: go_substantive_units
    type: noul
    instructions: "Is {{.Name}} documented?"
    criteria:
      false_description: "{{.Name}} missing from README.md."
    fix_hint: "Document {{.Name}} in README.md."
    pass:
      expect: true
      min_probability: 0.50
`))
	if err != nil {
		t.Fatalf("ParseRuleYAML() unexpected error: %v", err)
	}
	return repo, spec
}

func openTestMemoryCache(t *testing.T, ctx context.Context) *DecisionCache {
	t.Helper()
	t.Chdir(t.TempDir())
	cache, err := OpenDecisionCache(ctx, "")
	if err != nil {
		t.Fatalf("OpenDecisionCache() unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = cache.Close() })
	return cache
}
