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

package rule

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMutantOperatorMethods(t *testing.T) {
	testCases := []struct {
		name       string
		ops        []MutantOperator
		wantValid  bool
		wantLabels map[MutantOperator]string
	}{
		{
			name:      "all registered operators are valid with descriptive labels",
			ops:       []MutantOperator{OperatorROR, OperatorLCR, OperatorBCR, OperatorAOR, OperatorUOI, OperatorBLR, OperatorSBR},
			wantValid: true,
			wantLabels: map[MutantOperator]string{
				OperatorROR: "boundary/comparison",
				OperatorLCR: "logical &&/||",
				OperatorBCR: "branch forced true/false",
				OperatorAOR: "arithmetic/concat",
				OperatorUOI: "negation/sign",
				OperatorBLR: "return bool flipped",
				OperatorSBR: "statement skipped",
			},
		},
		{
			name:      "unrecognized operator is invalid and echoes raw code",
			ops:       []MutantOperator{MutantOperator("XYZ")},
			wantValid: false,
			wantLabels: map[MutantOperator]string{
				MutantOperator("XYZ"): "XYZ",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotLabels := make(map[MutantOperator]string, len(tc.ops))
			for _, op := range tc.ops {
				if got := op.IsValid(); got != tc.wantValid {
					t.Errorf("IsValid(%q) = %v, want %v", op, got, tc.wantValid)
				}
				gotLabels[op] = op.Label()
			}
			if diff := cmp.Diff(tc.wantLabels, gotLabels); diff != "" {
				t.Errorf("Labels mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLookupRuleSpecAndThresholds(t *testing.T) {
	specs := AllRuleSpecs()
	if len(specs) != 16 {
		t.Fatalf("len(AllRuleSpecs()) = %d, want 16", len(specs))
	}

	th := DefaultThresholds()
	if th.MinParamsForFlagCheck != 3 {
		t.Errorf("MinParamsForFlagCheck = %d, want 3", th.MinParamsForFlagCheck)
	}
	if th.Limits[RuleFuncCognitive] != 15 {
		t.Errorf("DefaultThresholds().Limits[%q] = %d, want 15", RuleFuncCognitive, th.Limits[RuleFuncCognitive])
	}
	if got := (Thresholds{}).Limit(RuleID("unknown_rule")); got != 0 {
		t.Errorf("Thresholds{}.Limit(unknown_rule) = %d, want 0", got)
	}

	testCases := []struct {
		name      string
		id        RuleID
		customMap map[RuleID]int
		wantFound bool
		wantLimit int
		checkCat  AllowCategory
		wantAllow bool
	}{
		{
			name:      "default cognitive limit and dispatch category",
			id:        RuleFuncCognitive,
			wantFound: true,
			wantLimit: 15,
			checkCat:  CategoryDispatch,
			wantAllow: true,
		},
		{
			name:      "custom override takes precedence over spec default",
			id:        RuleFuncCognitive,
			customMap: map[RuleID]int{RuleFuncCognitive: 10},
			wantFound: true,
			wantLimit: 10,
			checkCat:  CategoryEquivalent,
			wantAllow: false,
		},
		{
			name:      "fallback to spec default when map omits rule",
			id:        RuleFuncCodeLines,
			customMap: map[RuleID]int{},
			wantFound: true,
			wantLimit: 60,
			checkCat:  CategoryProtocol,
			wantAllow: true,
		},
		{
			name:      "unknown rule returns zero limit",
			id:        RuleID("unknown_rule"),
			customMap: map[RuleID]int{},
			wantFound: false,
			wantLimit: 0,
			checkCat:  CategoryProtocol,
			wantAllow: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spec, ok := LookupRuleSpec(tc.id)
			if ok != tc.wantFound {
				t.Errorf("LookupRuleSpec(%q) ok = %v, want %v", tc.id, ok, tc.wantFound)
			}
			if gotAllow := spec.AllowsCategory(tc.checkCat); gotAllow != tc.wantAllow {
				t.Errorf("AllowsCategory(%q) = %v, want %v", tc.checkCat, gotAllow, tc.wantAllow)
			}
			cur := th
			if tc.customMap != nil {
				cur.Limits = tc.customMap
			}
			if got := cur.Limit(tc.id); got != tc.wantLimit {
				t.Errorf("Limit(%q) = %d, want %d", tc.id, got, tc.wantLimit)
			}
		})
	}
}
