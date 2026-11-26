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

package redundancy

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func TestAnalyzeAndBuildFindings(t *testing.T) {
	changed := []ChangedTestFunc{
		{
			File:     "pkg/svc/svc_test.go",
			FuncName: "TestLater",
			Line:     60,
			EndLine:  95,
			Subtests: map[string]SubtestMeta{
				"cand_weak_early":     {NormalizedName: "cand_weak_early", Line: 62, InDiff: true, OnlyErrorNil: false},
				"cand_strong_first":   {NormalizedName: "cand_strong_first", Line: 65, InDiff: true, OnlyErrorNil: false},
				"cand_strong_second":  {NormalizedName: "cand_strong_second", Line: 70, InDiff: true, OnlyErrorNil: false},
				"untouched_weak_late": {NormalizedName: "untouched_weak_late", Line: 75, InDiff: false, OnlyErrorNil: false},
			},
		},
		{
			File:              "pkg/a/a_test.go",
			FuncName:          "TestWhole",
			Line:              100,
			EndLine:           120,
			EntireFuncChanged: true,
			OnlyErrorNil:      true,
			Subtests:          map[string]SubtestMeta{},
		},
		{
			File:     "pkg/svc/svc_test.go",
			FuncName: "TestCalc",
			Line:     10,
			EndLine:  55,
			Subtests: map[string]SubtestMeta{
				"":               {NormalizedName: "", Line: 12, InDiff: true, OnlyErrorNil: true},
				"empty_pol":      {NormalizedName: "empty_pol", Line: 13, InDiff: true, OnlyErrorNil: true},
				"budget_dom":     {NormalizedName: "budget_dom", Line: 14, InDiff: false, OnlyErrorNil: false},
				"strong_base":    {NormalizedName: "strong_base", Line: 15, InDiff: false, OnlyErrorNil: false},
				"mid_cand":       {NormalizedName: "mid_cand", Line: 16, InDiff: true, OnlyErrorNil: false},
				"untouched_dup":  {NormalizedName: "untouched_dup", Line: 18, InDiff: false, OnlyErrorNil: false},
				"dup_same":       {NormalizedName: "dup_same", Line: 20, InDiff: true, OnlyErrorNil: false},
				"never_ran":      {NormalizedName: "never_ran", Line: 22, InDiff: true, OnlyErrorNil: true},
				"weak_err":       {NormalizedName: "weak_err", Line: 25, InDiff: true, OnlyErrorNil: true},
				"weak_uncov":     {NormalizedName: "weak_uncov", Line: 28, InDiff: true, OnlyErrorNil: true},
				"zero_kill_ok":   {NormalizedName: "zero_kill_ok", Line: 30, InDiff: true, OnlyErrorNil: false},
				"unique_ran":     {NormalizedName: "unique_ran", Line: 32, InDiff: true, OnlyErrorNil: false},
				"unique_killed":  {NormalizedName: "unique_killed", Line: 34, InDiff: true, OnlyErrorNil: false},
				"diff_branch":    {NormalizedName: "diff_branch", Line: 36, InDiff: true, OnlyErrorNil: false},
				"weak_diff_pol":  {NormalizedName: "weak_diff_pol", Line: 37, InDiff: true, OnlyErrorNil: true},
				"budget_skip":    {NormalizedName: "budget_skip", Line: 38, InDiff: true, OnlyErrorNil: true},
				"subset_pol_kil": {NormalizedName: "subset_pol_kil", Line: 40, InDiff: true, OnlyErrorNil: false},
			},
		},
	}

	leafOrder := []string{
		"TestUntracked/sub",
		"TestCalc/empty_pol",
		"TestCalc/#00",
		"TestCalc/strong_base",
		"TestCalc/mid_cand",
		"TestCalc/dup_same#01",
		"TestCalc/untouched_dup",
		"TestCalc/budget_dom",
		"TestCalc/never_ran",
		"TestCalc/weak_err",
		"TestCalc/weak_uncov",
		"TestCalc/zero_kill_ok",
		"TestCalc/unique_ran",
		"TestCalc/unique_killed",
		"TestCalc/diff_branch",
		"TestCalc/weak_diff_pol",
		"TestCalc/budget_skip",
		"TestCalc/subset_pol_kil",
		"TestWhole/dyn_b",
		"TestWhole/dyn_a#01",
		"TestLater/cand_weak_early",
		"TestLater/cand_strong_first",
		"TestLater/cand_strong_second",
		"TestLater/untouched_weak_late",
	}

	basePol := map[string]struct{}{"1:T": {}, "2:S": {}, "P1:T": {}}
	polarity := map[string]map[string]struct{}{
		"TestUntracked/sub":             basePol,
		"TestCalc/#00":                  {"1:T": {}},
		"TestCalc/strong_base":          basePol,
		"TestCalc/mid_cand":             basePol,
		"TestCalc/dup_same#01":          basePol,
		"TestCalc/untouched_dup":        basePol,
		"TestCalc/budget_dom":           {"1:T": {}, "99:T": {}},
		"TestCalc/never_ran":            {"1:T": {}},
		"TestCalc/weak_err":             {"1:T": {}},
		"TestCalc/weak_uncov":           {"7:T": {}},
		"TestCalc/zero_kill_ok":         {"1:T": {}},
		"TestCalc/unique_ran":           basePol,
		"TestCalc/unique_killed":        basePol,
		"TestCalc/diff_branch":          {"1:F": {}, "2:S": {}, "P1:T": {}},
		"TestCalc/weak_diff_pol":        {"1:F": {}, "8:T": {}, "P1:T": {}},
		"TestCalc/budget_skip":          {"1:T": {}, "99:T": {}},
		"TestCalc/subset_pol_kil":       {"1:T": {}},
		"TestWhole/dyn_b":               {"1:T": {}},
		"TestWhole/dyn_a#01":            {"1:T": {}},
		"TestLater/cand_weak_early":     {"4:T": {}},
		"TestLater/cand_strong_first":   {"4:T": {}},
		"TestLater/cand_strong_second":  {"4:T": {}},
		"TestLater/untouched_weak_late": {"4:T": {}},
	}

	allRan := map[string]struct{}{
		"TestUntracked/sub":             {},
		"TestCalc/empty_pol":            {},
		"TestCalc/#00":                  {},
		"TestCalc/strong_base":          {},
		"TestCalc/mid_cand":             {},
		"TestCalc/dup_same#01":          {},
		"TestCalc/untouched_dup":        {},
		"TestCalc/budget_dom":           {},
		"TestCalc/weak_err":             {},
		"TestCalc/weak_uncov":           {},
		"TestCalc/zero_kill_ok":         {},
		"TestCalc/unique_ran":           {},
		"TestCalc/unique_killed":        {},
		"TestCalc/diff_branch":          {},
		"TestCalc/weak_diff_pol":        {},
		"TestCalc/budget_skip":          {},
		"TestCalc/subset_pol_kil":       {},
		"TestWhole/dyn_b":               {},
		"TestWhole/dyn_a#01":            {},
		"TestLater/cand_weak_early":     {},
		"TestLater/cand_strong_first":   {},
		"TestLater/cand_strong_second":  {},
		"TestLater/untouched_weak_late": {},
	}

	mutantRuns := map[int]VerboseTestResult{
		1: {
			RanLeafSet: allRan,
			FailedLeafs: map[string]struct{}{
				"TestUntracked/sub":       {},
				"TestCalc/strong_base":    {},
				"TestCalc/mid_cand":       {},
				"TestCalc/dup_same#01":    {},
				"TestCalc/untouched_dup":  {},
				"TestCalc/budget_dom":     {},
				"TestCalc/unique_killed":  {},
				"TestCalc/diff_branch":    {},
				"TestCalc/subset_pol_kil": {},
			},
		},
		2: {
			RanLeafSet: allRan,
			FailedLeafs: map[string]struct{}{
				"TestUntracked/sub":    {},
				"TestCalc/strong_base": {},
				"TestCalc/mid_cand":    {},
			},
		},
		3: {
			RanLeafSet:  map[string]struct{}{"TestCalc/unique_ran": {}},
			FailedLeafs: map[string]struct{}{"TestCalc/unique_ran": {}},
		},
		4: {
			RanLeafSet: allRan,
			FailedLeafs: map[string]struct{}{
				"TestUntracked/sub":             {},
				"TestCalc/unique_killed":        {},
				"TestLater/cand_weak_early":     {},
				"TestLater/cand_strong_first":   {},
				"TestLater/cand_strong_second":  {},
				"TestLater/untouched_weak_late": {},
			},
		},
		5: {
			RanLeafSet: allRan,
			FailedLeafs: map[string]struct{}{
				"TestLater/cand_strong_first":  {},
				"TestLater/cand_strong_second": {},
			},
		},
	}

	if got := Analyze(AnalyzeInput{ChangedFuncs: nil, LeafOrder: leafOrder}); got != nil {
		t.Errorf("Analyze(empty ChangedFuncs) = %+v, want nil", got)
	}
	if got := Analyze(AnalyzeInput{ChangedFuncs: changed, LeafOrder: nil}); got != nil {
		t.Errorf("Analyze(empty LeafOrder) = %+v, want nil", got)
	}

	wholeDetails := []rule.RedundantDetail{
		{
			Kind:            rule.RedundantNoOutputCheck,
			Line:            100,
			TestName:        "TestWhole/dyn_a#01",
			DominatedBy:     "TestCalc/strong_base",
			DominatedByFile: "pkg/svc/svc_test.go",
			DominatedByLine: 15,
			CandidateKills:  0,
			DominatorKills:  2,
		},
		{
			Kind:            rule.RedundantNoOutputCheck,
			Line:            100,
			TestName:        "TestWhole/dyn_b",
			DominatedBy:     "TestCalc/strong_base",
			DominatedByFile: "pkg/svc/svc_test.go",
			DominatedByLine: 15,
			CandidateKills:  0,
			DominatorKills:  2,
		},
	}
	calcSkippedDetails := []rule.RedundantDetail{
		{
			Kind:            rule.RedundantSamePath,
			Line:            20,
			TestName:        "TestCalc/dup_same#01",
			DominatedBy:     "TestCalc/strong_base",
			DominatedByFile: "pkg/svc/svc_test.go",
			DominatedByLine: 15,
			CandidateKills:  1,
			DominatorKills:  2,
		},
		{
			Kind:            rule.RedundantNoOutputCheck,
			Line:            25,
			TestName:        "TestCalc/weak_err",
			DominatedBy:     "TestCalc/strong_base",
			DominatedByFile: "pkg/svc/svc_test.go",
			DominatedByLine: 15,
			CandidateKills:  0,
			DominatorKills:  2,
		},
	}
	calcAllDetails := append(append([]rule.RedundantDetail{}, calcSkippedDetails...), rule.RedundantDetail{
		Kind:            rule.RedundantNoOutputCheck,
		Line:            38,
		TestName:        "TestCalc/budget_skip",
		DominatedBy:     "TestCalc/budget_dom",
		DominatedByFile: "pkg/svc/svc_test.go",
		DominatedByLine: 14,
		CandidateKills:  0,
		DominatorKills:  1,
	})
	laterDetails := []rule.RedundantDetail{
		{
			Kind:            rule.RedundantSamePath,
			Line:            62,
			TestName:        "TestLater/cand_weak_early",
			DominatedBy:     "TestLater/cand_strong_first",
			DominatedByFile: "pkg/svc/svc_test.go",
			DominatedByLine: 65,
			CandidateKills:  1,
			DominatorKills:  2,
		},
	}

	testCases := []struct {
		name         string
		skipped      map[int]struct{}
		limit        int
		wantGroups   []FuncRedundancy
		wantFindings []rule.Finding
	}{
		// Flags subsumed candidates sorted by file and line while exempting skipped mutants.
		{
			name:    "flags subsumed candidates sorted by file and line while exempting skipped mutants",
			skipped: map[int]struct{}{99: {}},
			limit:   0,
			wantGroups: []FuncRedundancy{
				{File: "pkg/a/a_test.go", FuncName: "TestWhole", Line: 100, EndLine: 120, Details: wholeDetails},
				{File: "pkg/svc/svc_test.go", FuncName: "TestCalc", Line: 10, EndLine: 55, Details: calcSkippedDetails},
				{File: "pkg/svc/svc_test.go", FuncName: "TestLater", Line: 60, EndLine: 95, Details: laterDetails},
			},
			wantFindings: []rule.Finding{
				{
					Rule:           rule.RuleTestRedundancy,
					Kind:           rule.ViolationAdvisory,
					File:           "pkg/a/a_test.go",
					Line:           100,
					EndLine:        120,
					Subject:        "TestWhole",
					After:          2,
					Detail:         "2 redundant subtest(s) subsumed by stronger tests",
					RedundantTests: wholeDetails,
				},
				{
					Rule:           rule.RuleTestRedundancy,
					Kind:           rule.ViolationAdvisory,
					File:           "pkg/svc/svc_test.go",
					Line:           20,
					EndLine:        55,
					Subject:        "TestCalc",
					After:          2,
					Detail:         "2 redundant subtest(s) subsumed by stronger tests",
					RedundantTests: calcSkippedDetails,
				},
				{
					Rule:           rule.RuleTestRedundancy,
					Kind:           rule.ViolationAdvisory,
					File:           "pkg/svc/svc_test.go",
					Line:           62,
					EndLine:        95,
					Subject:        "TestLater",
					After:          1,
					Detail:         "1 redundant subtest(s) subsumed by stronger tests",
					RedundantTests: laterDetails,
				},
			},
		},
		// Suppresses groups at or below limit and evaluates all candidates when skipped map is nil.
		{
			name:    "limit suppresses groups within threshold and nil skipped map evaluates all candidates",
			skipped: nil,
			limit:   2,
			wantGroups: []FuncRedundancy{
				{File: "pkg/a/a_test.go", FuncName: "TestWhole", Line: 100, EndLine: 120, Details: wholeDetails},
				{File: "pkg/svc/svc_test.go", FuncName: "TestCalc", Line: 10, EndLine: 55, Details: calcAllDetails},
				{File: "pkg/svc/svc_test.go", FuncName: "TestLater", Line: 60, EndLine: 95, Details: laterDetails},
			},
			wantFindings: []rule.Finding{
				{
					Rule:           rule.RuleTestRedundancy,
					Kind:           rule.ViolationAdvisory,
					File:           "pkg/svc/svc_test.go",
					Line:           20,
					EndLine:        55,
					Subject:        "TestCalc",
					After:          3,
					Limit:          2,
					Detail:         "3 redundant subtest(s) subsumed by stronger tests",
					RedundantTests: calcAllDetails,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			groups := Analyze(AnalyzeInput{
				ChangedFuncs:         changed,
				LeafOrder:            leafOrder,
				PolarityByLeaf:       polarity,
				MutantRuns:           mutantRuns,
				BudgetSkippedMutants: tc.skipped,
			})
			if diff := cmp.Diff(tc.wantGroups, groups); diff != "" {
				t.Errorf("Analyze() mismatch (-want +got):\n%s", diff)
			}
			findings := BuildFindings(groups, tc.limit)
			if diff := cmp.Diff(tc.wantFindings, findings); diff != "" {
				t.Errorf("BuildFindings() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAnalyzeDoesNotFlagSamePathAcrossDistinctTestFuncs verifies that a separate top-level Test* function executing the same differential polarity does not subsume another dedicated unit test as SAME PATH.
func TestAnalyzeDoesNotFlagSamePathAcrossDistinctTestFuncs(t *testing.T) {
	// Arrange.
	changed := []ChangedTestFunc{
		{
			File:              "pkg/runner/runner_test.go",
			FuncName:          "TestCompileIntegration",
			Line:              10,
			EndLine:           25,
			EntireFuncChanged: true,
		},
		{
			File:              "pkg/runner/runner_test.go",
			FuncName:          "TestBuildEnvUnit",
			Line:              30,
			EndLine:           45,
			EntireFuncChanged: true,
		},
	}
	sharedPolarity := map[string]struct{}{"1:T": {}, "2:S": {}}
	ranBoth := map[string]struct{}{
		"TestCompileIntegration": {},
		"TestBuildEnvUnit":       {},
	}
	input := AnalyzeInput{
		ChangedFuncs: changed,
		LeafOrder:    []string{"TestCompileIntegration", "TestBuildEnvUnit"},
		PolarityByLeaf: map[string]map[string]struct{}{
			"TestCompileIntegration": sharedPolarity,
			"TestBuildEnvUnit":       sharedPolarity,
		},
		MutantRuns: map[int]VerboseTestResult{
			1: {RanLeafSet: ranBoth, FailedLeafs: ranBoth},
		},
	}

	// Act.
	got := Analyze(input)

	// Assert.
	if len(got) != 0 {
		t.Errorf("Analyze() = %+v, want empty (cross-TestFunc SAME PATH must not fire)", got)
	}
}

// TestAnalyzeDoesNotFlagEqualKillPairs verifies that subtests killing the exact same mutant set are never flagged as SAME PATH regardless of candidate status.
func TestAnalyzeDoesNotFlagEqualKillPairs(t *testing.T) {
	testCases := []struct {
		name     string
		subtests map[string]SubtestMeta
		order    []string
	}{
		// Candidate subtests with identical kills do not subsume each other.
		{
			name: "two candidates with identical kills",
			subtests: map[string]SubtestMeta{
				"case_a": {NormalizedName: "case_a", Line: 12, InDiff: true},
				"case_b": {NormalizedName: "case_b", Line: 16, InDiff: true},
			},
			order: []string{"TestPair/case_b", "TestPair/case_a"},
		},
		// Candidate subtest matching an untouched baseline's exact kill set is not flagged.
		{
			name: "candidate matching baseline identical kills",
			subtests: map[string]SubtestMeta{
				"baseline":  {NormalizedName: "baseline", Line: 12, InDiff: false},
				"candidate": {NormalizedName: "candidate", Line: 16, InDiff: true},
			},
			order: []string{"TestPair/candidate", "TestPair/baseline"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			sharedPol := map[string]struct{}{"1:T": {}, "2:F": {}}
			ranBoth := map[string]struct{}{tc.order[0]: {}, tc.order[1]: {}}
			input := AnalyzeInput{
				ChangedFuncs: []ChangedTestFunc{{
					File:     "pkg/svc/svc_test.go",
					FuncName: "TestPair",
					Line:     10,
					EndLine:  25,
					Subtests: tc.subtests,
				}},
				LeafOrder: tc.order,
				PolarityByLeaf: map[string]map[string]struct{}{
					tc.order[0]: sharedPol,
					tc.order[1]: sharedPol,
				},
				MutantRuns: map[int]VerboseTestResult{
					1: {RanLeafSet: ranBoth, FailedLeafs: ranBoth},
					2: {RanLeafSet: ranBoth, FailedLeafs: ranBoth},
				},
			}

			// Act.
			got := Analyze(input)

			// Assert.
			if len(got) != 0 {
				t.Errorf("Analyze() = %+v, want empty for equal-kill pair", got)
			}
		})
	}
}

// TestAnalyzeDeterministicDominatorAcrossLeafOrder verifies that dominator selection is independent of LeafOrder across both SAME PATH and NO OUTPUT CHECK modes.
func TestAnalyzeDeterministicDominatorAcrossLeafOrder(t *testing.T) {
	testCases := []struct {
		name           string
		changed        []ChangedTestFunc
		forwardOrder   []string
		polarityByLeaf map[string]map[string]struct{}
		mutantRuns     map[int]VerboseTestResult
		want           []FuncRedundancy
	}{
		// Prefers higher kill count over non-candidate status in SAME PATH regardless of execution order.
		{
			name: "same path prefers higher kill count over baseline status",
			changed: []ChangedTestFunc{{
				File: "pkg/svc/svc_test.go", FuncName: "TestOrder", Line: 10, EndLine: 40,
				Subtests: map[string]SubtestMeta{
					"weak":       {NormalizedName: "weak", Line: 12, InDiff: true},
					"base_two":   {NormalizedName: "base_two", Line: 15, InDiff: false},
					"cand_three": {NormalizedName: "cand_three", Line: 20, InDiff: true},
				},
			}},
			forwardOrder: []string{"TestOrder/weak", "TestOrder/base_two", "TestOrder/cand_three"},
			polarityByLeaf: map[string]map[string]struct{}{
				"TestOrder/weak":       {"1:T": {}},
				"TestOrder/base_two":   {"1:T": {}},
				"TestOrder/cand_three": {"1:T": {}},
			},
			mutantRuns: map[int]VerboseTestResult{
				1: {RanLeafSet: map[string]struct{}{"TestOrder/weak": {}, "TestOrder/base_two": {}, "TestOrder/cand_three": {}}, FailedLeafs: map[string]struct{}{"TestOrder/weak": {}, "TestOrder/base_two": {}, "TestOrder/cand_three": {}}},
				2: {RanLeafSet: map[string]struct{}{"TestOrder/weak": {}, "TestOrder/base_two": {}, "TestOrder/cand_three": {}}, FailedLeafs: map[string]struct{}{"TestOrder/base_two": {}, "TestOrder/cand_three": {}}},
				3: {RanLeafSet: map[string]struct{}{"TestOrder/weak": {}, "TestOrder/base_two": {}, "TestOrder/cand_three": {}}, FailedLeafs: map[string]struct{}{"TestOrder/cand_three": {}}},
			},
			want: []FuncRedundancy{{
				File: "pkg/svc/svc_test.go", FuncName: "TestOrder", Line: 10, EndLine: 40,
				Details: []rule.RedundantDetail{{
					Kind: rule.RedundantSamePath, Line: 12, TestName: "TestOrder/weak",
					DominatedBy: "TestOrder/cand_three", DominatedByFile: "pkg/svc/svc_test.go", DominatedByLine: 20,
					CandidateKills: 1, DominatorKills: 3,
				}},
			}},
		},
		// Tie-breaks equal-kill covering dominators by non-candidate then file, line, and name across NO OUTPUT CHECK.
		{
			name: "covering test tie-breaks by non-candidate then file line name",
			changed: []ChangedTestFunc{
				{
					File: "pkg/b/b_test.go", FuncName: "TestB", Line: 5, EndLine: 20,
					Subtests: map[string]SubtestMeta{
						"base_b": {NormalizedName: "base_b", Line: 10, InDiff: false},
					},
				},
				{
					File: "pkg/a/a_test.go", FuncName: "TestA", Line: 5, EndLine: 30,
					Subtests: map[string]SubtestMeta{
						"cand_a": {NormalizedName: "cand_a", Line: 8, InDiff: true},
						"base_z": {NormalizedName: "base_z", Line: 12, InDiff: false},
						"base_a": {NormalizedName: "base_a", Line: 12, InDiff: false},
					},
				},
				{
					File: "pkg/c/c_test.go", FuncName: "TestWeak", Line: 5, EndLine: 15,
					EntireFuncChanged: true, OnlyErrorNil: true,
				},
			},
			forwardOrder: []string{"TestWeak", "TestB/base_b", "TestA/cand_a", "TestA/base_z", "TestA/base_a"},
			polarityByLeaf: map[string]map[string]struct{}{
				"TestWeak":     {"1:T": {}},
				"TestB/base_b": {"1:T": {}},
				"TestA/cand_a": {"1:T": {}},
				"TestA/base_z": {"1:T": {}},
				"TestA/base_a": {"1:T": {}},
			},
			mutantRuns: map[int]VerboseTestResult{
				1: {
					RanLeafSet:  map[string]struct{}{"TestWeak": {}, "TestB/base_b": {}, "TestA/cand_a": {}, "TestA/base_z": {}, "TestA/base_a": {}},
					FailedLeafs: map[string]struct{}{"TestB/base_b": {}, "TestA/cand_a": {}, "TestA/base_z": {}, "TestA/base_a": {}},
				},
			},
			want: []FuncRedundancy{{
				File: "pkg/c/c_test.go", FuncName: "TestWeak", Line: 5, EndLine: 15,
				Details: []rule.RedundantDetail{{
					Kind: rule.RedundantNoOutputCheck, Line: 5, TestName: "TestWeak",
					DominatedBy: "TestA/base_a", DominatedByFile: "pkg/a/a_test.go", DominatedByLine: 12,
					CandidateKills: 0, DominatorKills: 1,
				}},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			reversedOrder := make([]string, len(tc.forwardOrder))
			for i, leaf := range tc.forwardOrder {
				reversedOrder[len(tc.forwardOrder)-1-i] = leaf
			}

			// Act.
			gotForward := Analyze(AnalyzeInput{
				ChangedFuncs: tc.changed, LeafOrder: tc.forwardOrder, PolarityByLeaf: tc.polarityByLeaf, MutantRuns: tc.mutantRuns,
			})
			gotReversed := Analyze(AnalyzeInput{
				ChangedFuncs: tc.changed, LeafOrder: reversedOrder, PolarityByLeaf: tc.polarityByLeaf, MutantRuns: tc.mutantRuns,
			})

			// Assert.
			if diff := cmp.Diff(tc.want, gotForward); diff != "" {
				t.Errorf("Analyze(forward) mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(gotForward, gotReversed); diff != "" {
				t.Errorf("Analyze(reversed) differed from forward order (-forward +reversed):\n%s", diff)
			}
		})
	}
}

// TestAnalyzeIgnoresAbortedMutantRuns verifies that aborted mutant runs do not induce SAME PATH or NO OUTPUT CHECK dominance while non-aborted runs evaluate normally.
func TestAnalyzeIgnoresAbortedMutantRuns(t *testing.T) {
	testCases := []struct {
		name       string
		subtests   map[string]SubtestMeta
		leafOrder  []string
		polarity   map[string]map[string]struct{}
		mutantRuns map[int]VerboseTestResult
		want       []FuncRedundancy
	}{
		// Ignores aborted mutant run so equal non-aborted kills do not induce SAME PATH dominance.
		{
			name: "aborted run does not induce SAME PATH",
			subtests: map[string]SubtestMeta{
				"weak":   {NormalizedName: "weak", Line: 12, InDiff: true},
				"strong": {NormalizedName: "strong", Line: 16, InDiff: false},
			},
			leafOrder: []string{"TestAbort/weak", "TestAbort/strong"},
			polarity: map[string]map[string]struct{}{
				"TestAbort/weak":   {"1:T": {}, "2:T": {}},
				"TestAbort/strong": {"1:T": {}, "2:T": {}},
			},
			mutantRuns: map[int]VerboseTestResult{
				1: {
					RanLeafSet:  map[string]struct{}{"TestAbort/weak": {}, "TestAbort/strong": {}},
					FailedLeafs: map[string]struct{}{"TestAbort/weak": {}, "TestAbort/strong": {}},
				},
				2: {
					RanLeafSet:  map[string]struct{}{"TestAbort/weak": {}, "TestAbort/strong": {}},
					FailedLeafs: map[string]struct{}{"TestAbort/strong": {}},
					Aborted:     true,
				},
			},
			want: []FuncRedundancy{},
		},
		// Ignores aborted mutant run so an aborted-only killer does not induce NO OUTPUT CHECK dominance.
		{
			name: "aborted run does not induce NO OUTPUT CHECK",
			subtests: map[string]SubtestMeta{
				"weak_err":   {NormalizedName: "weak_err", Line: 12, InDiff: true, OnlyErrorNil: true},
				"panic_only": {NormalizedName: "panic_only", Line: 16, InDiff: false},
			},
			leafOrder: []string{"TestAbort/weak_err", "TestAbort/panic_only"},
			polarity: map[string]map[string]struct{}{
				"TestAbort/weak_err":   {"1:T": {}, "2:T": {}},
				"TestAbort/panic_only": {"1:T": {}, "2:T": {}},
			},
			mutantRuns: map[int]VerboseTestResult{
				1: {
					RanLeafSet:  map[string]struct{}{"TestAbort/weak_err": {}, "TestAbort/panic_only": {}},
					FailedLeafs: map[string]struct{}{},
				},
				2: {
					RanLeafSet:  map[string]struct{}{"TestAbort/weak_err": {}, "TestAbort/panic_only": {}},
					FailedLeafs: map[string]struct{}{"TestAbort/panic_only": {}},
					Aborted:     true,
				},
			},
			want: []FuncRedundancy{},
		},
		// Evaluates non-aborted mutant runs normally alongside ignored aborted runs.
		{
			name: "non-aborted runs evaluate normally alongside aborted runs",
			subtests: map[string]SubtestMeta{
				"weak_err": {NormalizedName: "weak_err", Line: 12, InDiff: true, OnlyErrorNil: true},
				"dup":      {NormalizedName: "dup", Line: 15, InDiff: true},
				"strong":   {NormalizedName: "strong", Line: 20, InDiff: false},
			},
			leafOrder: []string{"TestAbort/weak_err", "TestAbort/dup", "TestAbort/strong"},
			polarity: map[string]map[string]struct{}{
				"TestAbort/weak_err": {"1:T": {}, "2:T": {}, "3:T": {}},
				"TestAbort/dup":      {"1:T": {}, "2:T": {}, "3:T": {}},
				"TestAbort/strong":   {"1:T": {}, "2:T": {}, "3:T": {}},
			},
			mutantRuns: map[int]VerboseTestResult{
				1: {
					RanLeafSet:  map[string]struct{}{"TestAbort/weak_err": {}, "TestAbort/dup": {}, "TestAbort/strong": {}},
					FailedLeafs: map[string]struct{}{"TestAbort/dup": {}, "TestAbort/strong": {}},
				},
				2: {
					RanLeafSet:  map[string]struct{}{"TestAbort/weak_err": {}, "TestAbort/dup": {}, "TestAbort/strong": {}},
					FailedLeafs: map[string]struct{}{"TestAbort/strong": {}},
				},
				3: {
					RanLeafSet:  map[string]struct{}{"TestAbort/weak_err": {}, "TestAbort/dup": {}, "TestAbort/strong": {}},
					FailedLeafs: map[string]struct{}{"TestAbort/dup": {}},
					Aborted:     true,
				},
			},
			want: []FuncRedundancy{{
				File:     "pkg/svc/svc_test.go",
				FuncName: "TestAbort",
				Line:     10,
				EndLine:  25,
				Details: []rule.RedundantDetail{
					{
						Kind:            rule.RedundantNoOutputCheck,
						Line:            12,
						TestName:        "TestAbort/weak_err",
						DominatedBy:     "TestAbort/strong",
						DominatedByFile: "pkg/svc/svc_test.go",
						DominatedByLine: 20,
						CandidateKills:  0,
						DominatorKills:  2,
					},
					{
						Kind:            rule.RedundantSamePath,
						Line:            15,
						TestName:        "TestAbort/dup",
						DominatedBy:     "TestAbort/strong",
						DominatedByFile: "pkg/svc/svc_test.go",
						DominatedByLine: 20,
						CandidateKills:  1,
						DominatorKills:  2,
					},
				},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			input := AnalyzeInput{
				ChangedFuncs: []ChangedTestFunc{{
					File:     "pkg/svc/svc_test.go",
					FuncName: "TestAbort",
					Line:     10,
					EndLine:  25,
					Subtests: tc.subtests,
				}},
				LeafOrder:      tc.leafOrder,
				PolarityByLeaf: tc.polarity,
				MutantRuns:     tc.mutantRuns,
			}

			// Act.
			got := Analyze(input)

			// Assert.
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Analyze() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAnalyzeIgnoresParentPanicAfterChildFail verifies that a parent panic after a child failure produces Aborted=true via ParseVerboseTestOutput and does not induce false SAME PATH dominance against unexecuted siblings.
func TestAnalyzeIgnoresParentPanicAfterChildFail(t *testing.T) {
	// Arrange.
	panickedRun := ParseVerboseTestOutput([]byte(`=== RUN   TestParent
=== RUN   TestParent/a
--- FAIL: TestParent/a (0.00s)
=== CONT  TestParent
panic: setup for b crashed
`), true)
	input := AnalyzeInput{
		ChangedFuncs: []ChangedTestFunc{{
			File:     "pkg/svc/svc_test.go",
			FuncName: "TestParent",
			Line:     10,
			EndLine:  30,
			Subtests: map[string]SubtestMeta{
				"a": {NormalizedName: "a", Line: 15, InDiff: false},
				"b": {NormalizedName: "b", Line: 20, InDiff: true},
			},
		}},
		LeafOrder: []string{"TestParent/a", "TestParent/b"},
		PolarityByLeaf: map[string]map[string]struct{}{
			"TestParent/a": {"1:T": {}, "2:T": {}},
			"TestParent/b": {"1:T": {}, "2:T": {}},
		},
		MutantRuns: map[int]VerboseTestResult{
			1: {
				RanLeafSet:  map[string]struct{}{"TestParent/a": {}, "TestParent/b": {}},
				FailedLeafs: map[string]struct{}{"TestParent/a": {}, "TestParent/b": {}},
			},
			2: panickedRun,
		},
	}

	// Act.
	got := Analyze(input)

	// Assert.
	if !panickedRun.Aborted {
		t.Fatalf("ParseVerboseTestOutput() Aborted = false, want true")
	}
	if len(got) != 0 {
		t.Errorf("Analyze() = %+v, want empty when mutant 2 aborted in parent panic", got)
	}
}
