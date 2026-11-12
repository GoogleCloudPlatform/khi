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

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/google/go-cmp/cmp"
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
			Line:            16,
			TestName:        "TestCalc/mid_cand",
			DominatedBy:     "TestCalc/strong_base",
			DominatedByFile: "pkg/svc/svc_test.go",
			DominatedByLine: 15,
			CandidateKills:  2,
			DominatorKills:  2,
		},
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
		{
			Kind:            rule.RedundantSamePath,
			Line:            70,
			TestName:        "TestLater/cand_strong_second",
			DominatedBy:     "TestLater/cand_strong_first",
			DominatedByFile: "pkg/svc/svc_test.go",
			DominatedByLine: 65,
			CandidateKills:  2,
			DominatorKills:  2,
		},
	}

	testCases := []struct {
		name         string
		skipped      map[int]struct{}
		beforeMap    map[string]int
		wantGroups   []FuncRedundancy
		wantFindings []rule.Finding
	}{
		{
			name:      "flags subsumed candidates sorted by file and line while exempting skipped mutants",
			skipped:   map[int]struct{}{99: {}},
			beforeMap: nil,
			wantGroups: []FuncRedundancy{
				{File: "pkg/a/a_test.go", FuncName: "TestWhole", Line: 100, EndLine: 120, Details: wholeDetails},
				{File: "pkg/svc/svc_test.go", FuncName: "TestCalc", Line: 10, EndLine: 55, Details: calcSkippedDetails},
				{File: "pkg/svc/svc_test.go", FuncName: "TestLater", Line: 60, EndLine: 95, Details: laterDetails},
			},
			wantFindings: []rule.Finding{
				{
					Rule:           rule.RuleTestRedundancy,
					Kind:           rule.ViolationHardLimit,
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
					Kind:           rule.ViolationHardLimit,
					File:           "pkg/svc/svc_test.go",
					Line:           16,
					EndLine:        55,
					Subject:        "TestCalc",
					After:          3,
					Detail:         "3 redundant subtest(s) subsumed by stronger tests",
					RedundantTests: calcSkippedDetails,
				},
				{
					Rule:           rule.RuleTestRedundancy,
					Kind:           rule.ViolationHardLimit,
					File:           "pkg/svc/svc_test.go",
					Line:           62,
					EndLine:        95,
					Subject:        "TestLater",
					After:          2,
					Detail:         "2 redundant subtest(s) subsumed by stronger tests",
					RedundantTests: laterDetails,
				},
			},
		},
		{
			name:    "ratchet suppresses unchanged counts and nil skipped map evaluates all candidates",
			skipped: nil,
			beforeMap: map[string]int{
				"pkg/a/a_test.go|TestWhole":     2,
				"pkg/svc/svc_test.go|TestCalc":  4,
				"pkg/svc/svc_test.go|TestLater": 2,
			},
			wantGroups: []FuncRedundancy{
				{File: "pkg/a/a_test.go", FuncName: "TestWhole", Line: 100, EndLine: 120, Details: wholeDetails},
				{File: "pkg/svc/svc_test.go", FuncName: "TestCalc", Line: 10, EndLine: 55, Details: calcAllDetails},
				{File: "pkg/svc/svc_test.go", FuncName: "TestLater", Line: 60, EndLine: 95, Details: laterDetails},
			},
			wantFindings: nil,
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
			findings := BuildFindings(groups, 0, tc.beforeMap)
			if diff := cmp.Diff(tc.wantFindings, findings); diff != "" {
				t.Errorf("BuildFindings() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
