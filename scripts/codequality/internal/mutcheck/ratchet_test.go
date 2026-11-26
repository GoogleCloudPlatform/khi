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

package mutcheck

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestCollectSurvivingMutantFindings verifies direct ViolationAdvisory emission when surviving mutants exceed the threshold.
func TestCollectSurvivingMutantFindings(t *testing.T) {
	testCases := []struct {
		name  string
		limit int
		plans []*FilePlan
		want  []rule.Finding
	}{
		// Skips functions whose surviving mutant count is within the configured limit.
		{
			name:  "within limit produces zero findings",
			limit: 1,
			plans: []*FilePlan{
				{
					File: "svc/svc.go",
					Mutants: []*Mutant{
						{File: "svc/svc.go", FuncKey: "Check", Line: 12, Operator: rule.OperatorROR, Status: rule.MutantSurvived},
					},
				},
			},
			want: nil,
		},
		// Emits a ViolationAdvisory finding with EndLine and mutant details when surviving count exceeds limit.
		{
			name:  "above limit emits ViolationAdvisory directly",
			limit: 0,
			plans: []*FilePlan{
				{
					File: "svc/svc.go",
					Mutants: []*Mutant{
						{File: "svc/svc.go", FuncKey: "Check", Line: 12, Operator: rule.OperatorROR, OriginalExpr: "x > 0", MutatedExpr: "x >= 0", Status: rule.MutantSurvived},
						{File: "svc/svc.go", FuncKey: "Check", Line: 14, Operator: rule.OperatorAOR, OriginalExpr: "x + 1", MutatedExpr: "x", Status: rule.MutantKilled},
					},
				},
			},
			want: []rule.Finding{
				{
					Rule:    rule.RuleFuncSurvivingMutants,
					Kind:    rule.ViolationAdvisory,
					File:    "svc/svc.go",
					Line:    12,
					EndLine: 20,
					Subject: "Check",
					After:   1,
					Limit:   0,
					Detail:  "1 mutant(s) survived unit tests",
					SurvivingMutants: []rule.MutantDetail{
						{Operator: rule.OperatorROR, Line: 12, OriginalExpr: "x > 0", MutatedExpr: "x >= 0"},
					},
				},
			},
		},
	}

	astRes := &astcheck.Result{
		Files: map[string]*astcheck.FileMetrics{
			"svc/svc.go": {File: "svc/svc.go"},
		},
		TouchedFuncs: map[string][]astcheck.FuncMetrics{
			"svc/gen.go": {{Key: "Gen", DisplayName: "Gen", StartLine: 1, EndLine: 3}},
			"svc/svc.go": {{Key: "Check", DisplayName: "Check", StartLine: 10, EndLine: 20}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := &Result{}
			th := rule.Thresholds{Limits: map[rule.RuleID]int{rule.RuleFuncSurvivingMutants: tc.limit}}

			collectSurvivingMutantFindings(Options{Thresholds: th, ASTResult: astRes}, tc.plans, res)

			if diff := cmp.Diff(tc.want, res.Findings); diff != "" {
				t.Errorf("Findings mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
