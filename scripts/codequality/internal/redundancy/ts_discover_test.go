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

func TestDiscoverChangedTSSpecs(t *testing.T) {
	rawSpecs := []RawTSSpecCase{
		{FullName: ""},
		{FullName: "SignComponent should create", Line: 9, StartLine: 9, EndLine: 11, WeakAssertion: true},
		{FullName: "SignComponent isNegative param", Line: 20, StartLine: 20, EndLine: 20},
		{FullName: "SignComponent isNegative param", Line: 21, ItStartLine: 21, ItEndLine: 21},
		{FullName: "SignComponent isNegative zero boundary", Line: 22, SuiteHeaderLines: []int{5, 30}},
	}

	if got := DiscoverChangedTSSpecs("x.spec.ts", rawSpecs, nil); got != nil {
		t.Errorf("DiscoverChangedTSSpecs(nil lines) = %v, want nil", got)
	}
	if got := DiscoverChangedTSSpecs("x.spec.ts", nil, map[int]struct{}{9: {}}); got != nil {
		t.Errorf("DiscoverChangedTSSpecs(nil specs) = %v, want nil", got)
	}
	if got := DiscoverChangedTSSpecs("x.spec.ts", rawSpecs, map[int]struct{}{999: {}}); got != nil {
		t.Errorf("DiscoverChangedTSSpecs(no overlap) = %v, want nil", got)
	}

	relSpec := "web/src/app/sample/sign.component.spec.ts"
	discovered := DiscoverChangedTSSpecs(relSpec, rawSpecs, map[int]struct{}{10: {}, 21: {}, 30: {}})
	wantDiscovered := []ChangedTSSpec{
		{File: relSpec, FullName: "SignComponent should create", Line: 9, InDiff: true, WeakAssertion: true},
		{File: relSpec, FullName: "SignComponent isNegative param", Line: 20, InDiff: false, WeakAssertion: false},
		{File: relSpec, FullName: "SignComponent isNegative param", Line: 21, InDiff: true, WeakAssertion: false},
		{File: relSpec, FullName: "SignComponent isNegative zero boundary", Line: 22, InDiff: true, WeakAssertion: false},
	}
	if diff := cmp.Diff(wantDiscovered, discovered); diff != "" {
		t.Fatalf("DiscoverChangedTSSpecs() mismatch (-want +got):\n%s", diff)
	}

	if !IsSubjectMutantForTSSpec("./web/src/app/sample/sign.component.spec.ts", "./web/src/app/sample/sign.component.ts") {
		t.Errorf("IsSubjectMutantForTSSpec(same dir) = false, want true")
	}
	if !IsSubjectMutantForTSSpec(relSpec, "web/src/app/sample/sub/helper.ts") {
		t.Errorf("IsSubjectMutantForTSSpec(child subdir) = false, want true")
	}
	if IsSubjectMutantForTSSpec(relSpec, "web/src/app/sample2/sibling.ts") {
		t.Errorf("IsSubjectMutantForTSSpec(sibling prefix dir) = true, want false")
	}
	if IsSubjectMutantForTSSpec(relSpec, "web/src/app/parent.ts") {
		t.Errorf("IsSubjectMutantForTSSpec(parent dir) = true, want false")
	}
}

func TestAnalyzeTSAndHelpers(t *testing.T) {
	relSpec := "web/src/app/sample/sign.component.spec.ts"
	cleanSpec := "web/src/app/other/clean.spec.ts"
	allChanged := []ChangedTSSpec{
		{File: relSpec, FullName: "SignComponent should create", Line: 9, InDiff: true, WeakAssertion: true},
		{File: relSpec, FullName: "SignComponent isNegative param", Line: 20, InDiff: false},
		{File: relSpec, FullName: "SignComponent isNegative param", Line: 21, InDiff: true},
		{File: relSpec, FullName: "SignComponent isNegative zero boundary", Line: 22, InDiff: true},
		{File: relSpec, FullName: "SignComponent unrelated pol only", Line: 25, InDiff: true, WeakAssertion: true},
		{File: relSpec, FullName: "SignComponent probe distinct", Line: 28, InDiff: true},
		{File: cleanSpec, FullName: "CleanService works", Line: 12, InDiff: true},
	}

	groups := AnalyzeTS(TSAnalyzeInput{
		ChangedSpecs: allChanged,
		SpecOrder: []TSSpecOrderEntry{
			{ID: "", FullName: "SignComponent should create"},
			{ID: "empty_full", FullName: ""},
			{ID: "s_unknown", FullName: "unknown_name"},
			{ID: "spec0", FullName: "SignComponent should create"},
			{ID: "spec0_extra", FullName: "SignComponent should create"},
			{ID: "spec2", FullName: "SignComponent isNegative param"},
			{ID: "spec3", FullName: "SignComponent isNegative param"},
			{ID: "spec4", FullName: "SignComponent isNegative zero boundary"},
			{ID: "spec5", FullName: "SignComponent unrelated pol only"},
			{ID: "spec6", FullName: "SignComponent probe distinct"},
			{ID: "clean0", FullName: "CleanService works"},
		},
		SpecPolarities: map[string][]string{
			"spec0":  {"", "1:F", "2:F", "P5:T", "99:T", "P99:T"},
			"spec2":  {"1:T", "2:T", "P5:T"},
			"spec3":  {"1:T", "2:T", "P5:T", "P99:T"},
			"spec4":  {"1:F", "2:F", "P5:T"},
			"spec5":  {"", "99:T"},
			"spec6":  {"1:T", "2:T", "P5:F"},
			"clean0": {"99:T"},
		},
		MutantRuns: map[string]TSMutantSpecRun{
			"bad": {RanSpecIDs: []string{"spec0"}, FailedSpecIDs: []string{"spec0"}},
			"0":   {RanSpecIDs: []string{"spec0"}, FailedSpecIDs: []string{"spec0"}},
			"-5":  {RanSpecIDs: []string{"spec0"}, FailedSpecIDs: []string{"spec0"}},
			"1":   {RanSpecIDs: []string{"spec0", "spec2", "spec3", "spec4", "spec6"}, FailedSpecIDs: []string{"spec4"}},
			"2":   {RanSpecIDs: []string{"spec0", "spec2", "spec3", "spec4", "spec6"}, FailedSpecIDs: []string{"spec2", "spec3", "spec4", "spec6"}},
			"99":  {RanSpecIDs: []string{"spec0", "spec3", "spec5", "clean0"}, FailedSpecIDs: []string{"spec0", "spec3", "clean0"}},
		},
		MutantFileByID: map[int]string{
			-99: "web/src/app/other/unrelated.service.ts",
			-5:  "web/src/app/sample/sub/helper.ts",
			0:   "web/src/app/sample/sign.component.ts",
			1:   "web/src/app/sample/sign.component.ts",
			2:   "web/src/app/sample/sign.component.ts",
			99:  "web/src/app/other/unrelated.service.ts",
		},
		BudgetSkippedMutants: map[int]struct{}{99: {}},
	})

	wantGroups := []FuncRedundancy{
		{
			File:     relSpec,
			FuncName: "<spec>",
			Line:     9,
			EndLine:  21,
			Details: []rule.RedundantDetail{
				{
					Kind:            rule.RedundantNoOutputCheck,
					Line:            9,
					TestName:        "SignComponent should create",
					DominatedBy:     "SignComponent isNegative zero boundary",
					DominatedByFile: relSpec,
					DominatedByLine: 22,
					CandidateKills:  0,
					DominatorKills:  2,
				},
				{
					Kind:            rule.RedundantSamePath,
					Line:            21,
					TestName:        "SignComponent isNegative param",
					DominatedBy:     "SignComponent isNegative param",
					DominatedByFile: relSpec,
					DominatedByLine: 20,
					CandidateKills:  1,
					DominatorKills:  1,
				},
			},
		},
	}
	if diff := cmp.Diff(wantGroups, groups); diff != "" {
		t.Errorf("AnalyzeTS() mismatch (-want +got):\n%s", diff)
	}

	specsByName := groupTSSpecsByName(allChanged[:1])
	cursors := make(map[string]int)
	if _, ok := matchNextOrderedTSSpec(TSSpecOrderEntry{ID: "", FullName: "SignComponent should create"}, specsByName, cursors); ok {
		t.Errorf("matchNextOrderedTSSpec(empty ID) = true, want false")
	}
	if _, ok := matchNextOrderedTSSpec(TSSpecOrderEntry{ID: "first", FullName: "SignComponent should create"}, specsByName, cursors); !ok {
		t.Errorf("matchNextOrderedTSSpec(first) = false, want true")
	}
	if _, ok := matchNextOrderedTSSpec(TSSpecOrderEntry{ID: "overflow", FullName: "SignComponent should create"}, specsByName, cursors); ok {
		t.Errorf("matchNextOrderedTSSpec(overflow) = true, want false")
	}

	unfilteredPol := filterTSPolarityTokensForSpec(relSpec, []string{"1:T", "P1:T", "bad:T", "Pbad:T"}, nil)
	if diff := cmp.Diff(map[string]struct{}{"1:T": {}, "P1:T": {}}, unfilteredPol); diff != "" {
		t.Errorf("filterTSPolarityTokensForSpec(nil map) mismatch (-want +got):\n%s", diff)
	}
	filteredIDs := filterMutantIDsForSpec("sign.spec.ts", map[int]struct{}{1: {}, 404: {}}, map[int]string{1: "sign.ts"})
	if diff := cmp.Diff(map[int]struct{}{1: {}}, filteredIDs); diff != "" {
		t.Errorf("filterMutantIDsForSpec(missing ID) mismatch (-want +got):\n%s", diff)
	}
	unfilteredIDs := filterMutantIDsForSpec(relSpec, map[int]struct{}{1: {}}, nil)
	if diff := cmp.Diff(map[int]struct{}{1: {}}, unfilteredIDs); diff != "" {
		t.Errorf("filterMutantIDsForSpec(nil map) mismatch (-want +got):\n%s", diff)
	}
}
