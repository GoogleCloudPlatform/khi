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
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPlanFileMutationsAndRenderSuperMutant(t *testing.T) {
	src := `package sample

type state struct {
	count int
}

func evaluate(s *state, x int, ready bool) (int, bool) {
	unchanged := x + 100
	if x < 10 && ready {
		s.count += x - 1
		return unchanged, true
	}
	if ready {
		s.count++
	}
	if !ready {
		s.count -= 0 + x
		_ = s.count
	}
	check := func() bool {
		return true
	}
	if check() {
		return s.count, false
	}
	return 0, false
}

func isZero(v int) bool {
	return v == 0 || false
}
`
	changedLines := map[int]struct{}{
		9:  {}, // if x < 10 && ready {
		10: {}, // s.count += x - 1
		11: {}, // return unchanged, true
		13: {}, // if ready {
		14: {}, // s.count++
		16: {}, // if !ready {
		17: {}, // s.count -= 0 + x
		18: {}, // _ = s.count (blank LHS should NOT generate SBR)
		21: {}, // return true (inside FuncLit closure, slot 0)
		24: {}, // return s.count, false
		30: {}, // return v == 0 || false (1-slot bool FuncDecl)
	}

	nextID := 1
	plan, err := PlanFileMutations("pkg/sample/sample.go", []byte(src), &MutateFileOptions{
		ChangedLines: changedLines,
		NextMutantID: &nextID,
	})
	if err != nil {
		t.Fatalf("PlanFileMutations() unexpected error: %v", err)
	}

	type mutantSummary struct {
		ID       int
		Line     int
		Operator MutantOperator
		Orig     string
		Mut      string
		MutLine  string
	}
	var got []mutantSummary
	for _, m := range plan.Mutants {
		got = append(got, mutantSummary{
			ID:       m.ID,
			Line:     m.Line,
			Operator: m.Operator,
			Orig:     m.OriginalExpr,
			Mut:      m.MutatedExpr,
			MutLine:  m.MutatedLine,
		})
	}

	want := []mutantSummary{
		{ID: 1, Line: 9, Operator: OperatorLCR, Orig: "x < 10 && ready", Mut: "x < 10 || ready", MutLine: "if x < 10 || ready {"},
		{ID: 2, Line: 9, Operator: OperatorROR, Orig: "x < 10", Mut: "x <= 10", MutLine: "if x <= 10 && ready {"},
		{ID: 3, Line: 9, Operator: OperatorROR, Orig: "x < 10", Mut: "x >= 10", MutLine: "if x >= 10 && ready {"},
		{ID: 4, Line: 10, Operator: OperatorSBR, Orig: "s.count += x - 1", Mut: "/* skipped assignment */", MutLine: "/* skipped: s.count += x - 1 */"},
		{ID: 5, Line: 10, Operator: OperatorAOR, Orig: "x - 1", Mut: "x + 1", MutLine: "s.count += x + 1"},
		{ID: 6, Line: 11, Operator: OperatorBLR, Orig: "true", Mut: "false", MutLine: "return unchanged, false"},
		{ID: 7, Line: 13, Operator: OperatorBCR, Orig: "ready", Mut: "true", MutLine: "if true {"},
		{ID: 8, Line: 13, Operator: OperatorBCR, Orig: "ready", Mut: "false", MutLine: "if false {"},
		{ID: 9, Line: 14, Operator: OperatorSBR, Orig: "s.count++", Mut: "/* skipped s.count++ */", MutLine: "/* skipped: s.count++ */"},
		{ID: 10, Line: 16, Operator: OperatorUOI, Orig: "!ready", Mut: "ready", MutLine: "if ready {"},
		{ID: 11, Line: 17, Operator: OperatorSBR, Orig: "s.count -= 0 + x", Mut: "/* skipped assignment */", MutLine: "/* skipped: s.count -= 0 + x */"},
		{ID: 12, Line: 17, Operator: OperatorAOR, Orig: "0 + x", Mut: "x", MutLine: "s.count -= x"},
		{ID: 13, Line: 21, Operator: OperatorBLR, Orig: "true", Mut: "false", MutLine: "return false"},
		{ID: 14, Line: 24, Operator: OperatorBLR, Orig: "false", Mut: "true", MutLine: "return s.count, true"},
		{ID: 15, Line: 30, Operator: OperatorLCR, Orig: "v == 0 || false", Mut: "v == 0 && false", MutLine: "return v == 0 && false"},
		{ID: 16, Line: 30, Operator: OperatorROR, Orig: "v == 0", Mut: "v != 0", MutLine: "return v != 0 || false"},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("PlanFileMutations() mutants mismatch (-want +got):\n%s", diff)
	}

	rendered := RenderSuperMutantSource(plan.Source, plan.Sites)
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "sample.go", rendered, 0); err != nil {
		t.Fatalf("rendered Super-Mutant source failed Go syntax parsing: %v\nsource:\n%s", err, string(rendered))
	}
}

func TestPlanFileMutationsSkipsForHeaderIncDecAndCustomBoolReturn(t *testing.T) {
	src := `package sample

type customBool bool

func loop(n int) customBool {
	sum := 0
	for i := 0; i < n; i++ {
		sum += i
	}
	_ = sum
	return true
}
`
	changedLines := map[int]struct{}{
		7:  {}, // for i := 0; i < n; i++ (i++ in Post must NOT trigger SBR)
		11: {}, // return true (customBool return slot must NOT trigger BLR)
	}

	plan, err := PlanFileMutations("sample.go", []byte(src), &MutateFileOptions{
		ChangedLines: changedLines,
	})
	if err != nil {
		t.Fatalf("PlanFileMutations() unexpected error: %v", err)
	}

	type opExpr struct {
		Op   MutantOperator
		Expr string
	}
	var ops []opExpr
	for _, m := range plan.Mutants {
		ops = append(ops, opExpr{Op: m.Operator, Expr: m.MutatedExpr})
	}
	want := []opExpr{
		{Op: OperatorROR, Expr: "i <= n"},
		{Op: OperatorROR, Expr: "i >= n"},
	}
	if diff := cmp.Diff(want, ops); diff != "" {
		t.Errorf("operators mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanFileMutationsPreCoverProfileAndSuppression(t *testing.T) {
	src := `package sample

func calc(a int) int {
	// NOMUTANTS(perf): fast-path zero check does not change return value
	if a == 0 {
		return 0
	}
	if a > 5 {
		return a + 1
	}
	return a - 1
}
`
	changedLines := map[int]struct{}{
		5:  {},
		8:  {},
		9:  {},
		11: {},
	}
	coveredLines := map[int]struct{}{
		5:  {},
		8:  {},
		11: {},
		// Line 9 (return a + 1) is NOT covered.
	}

	plan, err := PlanFileMutations("sample.go", []byte(src), &MutateFileOptions{
		ChangedLines: changedLines,
		CoveredLines: coveredLines,
	})
	if err != nil {
		t.Fatalf("PlanFileMutations() error: %v", err)
	}

	type statusRecord struct {
		Line     int
		Op       MutantOperator
		Status   MutantStatus
		Category SuppressionCategory
		HasID    bool
	}
	var got []statusRecord
	for _, m := range plan.Mutants {
		got = append(got, statusRecord{
			Line:     m.Line,
			Op:       m.Operator,
			Status:   m.Status,
			Category: m.SuppressionCategory,
			HasID:    m.ID > 0,
		})
	}

	want := []statusRecord{
		{Line: 5, Op: OperatorROR, Status: MutantSuppressed, Category: SuppressionPerf, HasID: false},
		{Line: 8, Op: OperatorROR, Status: MutantPending, HasID: true},
		{Line: 8, Op: OperatorROR, Status: MutantPending, HasID: true},
		{Line: 9, Op: OperatorAOR, Status: MutantUncovered, HasID: false},
		{Line: 11, Op: OperatorAOR, Status: MutantPending, HasID: true},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("status records mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanFileMutationsSkipsBuildIgnoreFiles(t *testing.T) {
	src := `//go:build ignore

package main

func helper(a int) int {
	return a + 1
}
`
	plan, err := PlanFileMutations("gen.go", []byte(src), &MutateFileOptions{
		ChangedLines: map[int]struct{}{6: {}},
	})
	if err != nil {
		t.Fatalf("PlanFileMutations() unexpected error: %v", err)
	}
	if len(plan.Mutants) != 0 {
		t.Errorf("len(plan.Mutants) = %d, want 0 for //go:build ignore file", len(plan.Mutants))
	}
}
