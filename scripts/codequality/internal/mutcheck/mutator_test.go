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
	"go/parser"
	"go/token"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type planCounts struct {
	hasSites   bool
	suppressed int
	operators  int
	total      int
}

func summarizePlanCounts(plan *FilePlan) planCounts {
	seenOps := make(map[rule.MutantOperator]bool)
	supCount := 0
	for _, m := range plan.Mutants {
		seenOps[m.Operator] = true
		if m.Status == rule.MutantSuppressed {
			supCount++
		}
	}
	return planCounts{
		hasSites:   len(plan.Sites) > 0,
		suppressed: supCount,
		operators:  len(seenOps),
		total:      len(plan.Mutants),
	}
}

func TestPlanFileMutationsExtractsAllOperatorsAndSuppressions(t *testing.T) {
	baseSrc := `package demo

var globalFlag = true

func ExternNoBody()

func AllOps(a, b int, ok bool, err error) bool {
	if err != nil {
		return false
	}
	if x := a & b; x > 0 {
		for i := 0; i < 1; i++ {
			a -= 1
		}
	}
	if a < b && (a <= 10 || a > 0 && a >= -1 && a == b || a != 99) {
		a = a +
			1 - 2*b/1
		a += 0 + b
		a *= 2
		a++
	}
	if checkReady(a) {
		return !ok
	}
	// CODEQUALITY_ALLOW(func_surviving_mutants, equivalent, BLR): fallback boolean literal
	return true
}

// CODEQUALITY_ALLOW(func_surviving_mutants, equivalent): whole helper suppressed
func checkReady(v int) bool { return false }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "demo.go", baseSrc, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	directives, dirErrs := rule.ParseDirectives("demo.go", fset, file, []byte(baseSrc))
	if len(dirErrs) != 0 {
		t.Fatalf("unexpected directive errors: %+v", dirErrs)
	}

	testCases := []struct {
		name         string
		src          string
		rules        []rule.ExclusionRule
		coveredLines map[int]struct{}
		want         planCounts
	}{
		{
			name: "extracts all 7 operators with default exclusions and func/stmt CODEQUALITY_ALLOW",
			src:  baseSrc,
			want: planCounts{hasSites: true, suppressed: 2, operators: 7, total: 33},
		},
		{
			name:  "custom empty exclusion rules include error guard mutants",
			src:   baseSrc,
			rules: []rule.ExclusionRule{},
			want:  planCounts{hasSites: true, suppressed: 2, operators: 7, total: 35},
		},
		{
			name:         "marks uncovered lines without emitting compile sites",
			src:          baseSrc,
			coveredLines: map[int]struct{}{11: {}},
			want:         planCounts{hasSites: true, suppressed: 2, operators: 7, total: 33},
		},
		{
			name: "skips build ignore files",
			src:  "//go:build ignore\n\n" + baseSrc,
			want: planCounts{hasSites: false, suppressed: 0, operators: 0, total: 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plan, planErr := PlanFileMutations("demo.go", []byte(tc.src), PlanOptions{
				TargetLines:  lineRangeSet(1, 40),
				CoveredLines: tc.coveredLines,
				Directives:   cloneDirectives(directives),
				Rules:        tc.rules,
			})
			if planErr != nil {
				t.Fatalf("PlanFileMutations() error: %v", planErr)
			}
			got := summarizePlanCounts(plan)
			if got != tc.want {
				t.Errorf("summarizePlanCounts() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
