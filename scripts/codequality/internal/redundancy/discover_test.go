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
)

func TestDiscoverChangedTestFuncs(t *testing.T) {
	src := `package svc

import "testing"

type suite struct{}

func (suite) TestMethod(t *testing.T) {}
func TestMain(m *testing.M) {}
func Testhelper(t *testing.T) {}
func MakeHelper(t *testing.T) {}
func TestNoParams() {}
func TestTwoParams(t *testing.T, extra int) {}
func TestDeclOnly(t *testing.T)
func TestUntouched(t *testing.T) {}

func TestHeaderErrOnly(t *testing.T) {
	t.Helper()
	err := runOp(1); if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestTableAndLiteral(t *testing.T) {
	type row struct {
		name     string
		desc     string
		testName string
		other    string
		in       int
	}
	dynName := "var_row"
	extra := row{name: dynName}
	holder := struct{ items []row }{
		items: []row{{name: "in_struct_comp", in: 1}},
	}
	_ = holder
	_ = []any{[]row{{testName: "in_nested_slice", in: 2}}}
	_ = [][]string{{0: "indexed_elem"}}
	testCases := []row{
		{name: "first case", in: 1},
		{desc: "first case", in: 2},
		{},
		{name: ""},
		{name: dynName},
		{other: "no_name_field", in: 2},
		{"", "", "", "empty_pos", 3},
		{"positional\x01case", "", "", "", 3},
		extra,
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := runOp(tc.in); err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			t.Run("in_dyn_run", func(t *testing.T) {})
		})
	}
	pairCall(1, func() {
		t.Run("in_non_sel", func(t *testing.T) {})
	})
	helperObj.Other("not_subtest", func(t *testing.T) {
		t.Run("in_other_sel", func(t *testing.T) {})
	})
	t.Run("wrapped_fn", wrapSub(func(t *testing.T) {
		t.Run("in_wrapped_fn", func(t *testing.T) {})
	}))
	t.Run(("explicit_assert"), func(t *testing.T) {
		if got := runOp(3); got != nil {
			t.Errorf("got %v", got)
		}
		t.Run("nested_lit", func(t *testing.T) {})
	})
}
`
	if empty, err := DiscoverChangedTestFuncs("svc_test.go", []byte(src), nil); err != nil || empty != nil {
		t.Fatalf("DiscoverChangedTestFuncs(nil) = (%v, %v), want (nil, nil)", empty, err)
	}
	if _, err := DiscoverChangedTestFuncs("bad_test.go", []byte("package !bad"), map[int]struct{}{1: {}}); err == nil {
		t.Fatalf("DiscoverChangedTestFuncs(invalid syntax) = nil error, want non-nil")
	}

	changedLines := map[int]struct{}{
		7: {}, 8: {}, 9: {}, 10: {}, 11: {}, 12: {}, 13: {},
		16: {}, 40: {}, 68: {},
	}
	funcs, err := DiscoverChangedTestFuncs("svc_test.go", []byte(src), changedLines)
	if err != nil {
		t.Fatalf("DiscoverChangedTestFuncs() unexpected error: %v", err)
	}

	wantFuncs := []ChangedTestFunc{
		{
			File:              "svc_test.go",
			FuncName:          "TestHeaderErrOnly",
			Line:              16,
			EndLine:           21,
			EntireFuncChanged: true,
			OnlyErrorNil:      true,
			Subtests:          map[string]SubtestMeta{},
		},
		{
			File:              "svc_test.go",
			FuncName:          "TestTableAndLiteral",
			Line:              23,
			EndLine:           73,
			EntireFuncChanged: false,
			OnlyErrorNil:      false,
			Subtests: map[string]SubtestMeta{
				"in_dyn_run":         {NormalizedName: "in_dyn_run", Line: 55, InDiff: false, OnlyErrorNil: true},
				"in_non_sel":         {NormalizedName: "in_non_sel", Line: 59, InDiff: false, OnlyErrorNil: true},
				"in_other_sel":       {NormalizedName: "in_other_sel", Line: 62, InDiff: false, OnlyErrorNil: true},
				"in_wrapped_fn":      {NormalizedName: "in_wrapped_fn", Line: 65, InDiff: false, OnlyErrorNil: true},
				"explicit_assert":    {NormalizedName: "explicit_assert", Line: 67, InDiff: true, OnlyErrorNil: false},
				"nested_lit":         {NormalizedName: "nested_lit", Line: 71, InDiff: false, OnlyErrorNil: true},
				"in_struct_comp":     {NormalizedName: "in_struct_comp", Line: 34, InDiff: false, OnlyErrorNil: true},
				"in_nested_slice":    {NormalizedName: "in_nested_slice", Line: 37, InDiff: false, OnlyErrorNil: true},
				"first_case":         {NormalizedName: "first_case", Line: 40, InDiff: true, OnlyErrorNil: true},
				`positional\x01case`: {NormalizedName: `positional\x01case`, Line: 47, InDiff: false, OnlyErrorNil: true},
			},
		},
	}
	if diff := cmp.Diff(wantFuncs, funcs); diff != "" {
		t.Errorf("DiscoverChangedTestFuncs() mismatch (-want +got):\n%s", diff)
	}
	if got := BuildTestFuncsRunRegex(funcs); got != "^(?:TestHeaderErrOnly|TestTableAndLiteral)$" {
		t.Errorf("BuildTestFuncsRunRegex() = %q", got)
	}
}

func TestRunRegexAndSplitLeaf(t *testing.T) {
	if got := BuildTestFuncsRunRegex(nil); got != "" {
		t.Errorf("BuildTestFuncsRunRegex(nil) = %q, want empty", got)
	}
	if got := BuildSingleLeafRunRegex("TestFoo/a+b"); got != `^TestFoo$/^a\+b$` {
		t.Errorf("BuildSingleLeafRunRegex() = %q", got)
	}
	if top, sub := SplitTopAndSubtest("TestTopOnly"); top != "TestTopOnly" || sub != "" {
		t.Errorf("SplitTopAndSubtest(TestTopOnly) = (%q, %q)", top, sub)
	}
	if top, sub := SplitTopAndSubtest("TestTop/sub/leaf"); top != "TestTop" || sub != "sub/leaf" {
		t.Errorf("SplitTopAndSubtest(TestTop/sub/leaf) = (%q, %q)", top, sub)
	}
}

func TestParseVerboseTestOutput(t *testing.T) {
	testCases := []struct {
		name    string
		rawOut  string
		nonZero bool
		want    VerboseTestResult
	}{
		{
			name: "extracts deduplicated leaf tests skips skipped and strips durations",
			rawOut: "=== RUN   \n" + `=== RUN   TestSuite
=== RUN   TestSuite/first_case
=== RUN   TestSuite/first_case
=== RUN   TestSuite/skipped_case
--- SKIP: TestSuite/skipped_case (0.00s)
=== RUN   TestSuite/explicit_assert
--- PASS:   TestSuite/first_case
--- FAIL:   TestSuite/explicit_assert (0.01s)
--- FAIL: TestSuite (0.01s)
=== RUN   TestSuiteExtra
--- PASS: TestSuiteExtra (0.00s)
`,
			nonZero: true,
			want: VerboseTestResult{
				LeafOrder:   []string{"TestSuite/first_case", "TestSuite/explicit_assert", "TestSuiteExtra"},
				RanLeafSet:  map[string]struct{}{"TestSuite/first_case": {}, "TestSuite/explicit_assert": {}, "TestSuiteExtra": {}},
				FailedLeafs: map[string]struct{}{"TestSuite/explicit_assert": {}},
			},
		},
		{
			name: "attributes direct crash without CONT or FAIL line to active leaf",
			rawOut: `=== RUN   TestDirectCrash
panic: boom
`,
			nonZero: true,
			want: VerboseTestResult{
				LeafOrder:   []string{"TestDirectCrash"},
				RanLeafSet:  map[string]struct{}{"TestDirectCrash": {}},
				FailedLeafs: map[string]struct{}{"TestDirectCrash": {}},
			},
		},
		{
			name: "attributes parallel crash after CONT and preserves active leaf across unrelated PASS",
			rawOut: `=== RUN   TestResumed
=== RUN   TestOther
--- PASS: TestOther (0.00s)
=== RUN   TestEarly
=== CONT   TestResumed
--- PASS: TestEarly (0.00s)
panic: boom
`,
			nonZero: true,
			want: VerboseTestResult{
				LeafOrder:   []string{"TestResumed", "TestOther", "TestEarly"},
				RanLeafSet:  map[string]struct{}{"TestResumed": {}, "TestOther": {}, "TestEarly": {}},
				FailedLeafs: map[string]struct{}{"TestResumed": {}},
			},
		},
		{
			name: "does not attribute active leaf when exit code is zero",
			rawOut: `=== RUN   TestIncomplete
`,
			nonZero: false,
			want: VerboseTestResult{
				LeafOrder:   []string{"TestIncomplete"},
				RanLeafSet:  map[string]struct{}{"TestIncomplete": {}},
				FailedLeafs: map[string]struct{}{},
			},
		},
		{
			name: "does not attribute failure when active leaf already passed before non-zero exit",
			rawOut: `=== RUN   TestPassed
--- PASS: TestPassed (0.00s)
`,
			nonZero: true,
			want: VerboseTestResult{
				LeafOrder:   []string{"TestPassed"},
				RanLeafSet:  map[string]struct{}{"TestPassed": {}},
				FailedLeafs: map[string]struct{}{},
			},
		},
		{
			name: "does not attribute active leaf when explicit FAIL was already recorded",
			rawOut: `=== RUN   TestExplicitFail
--- FAIL: TestExplicitFail (0.00s)
=== RUN   TestStillRunning
`,
			nonZero: true,
			want: VerboseTestResult{
				LeafOrder:   []string{"TestExplicitFail", "TestStillRunning"},
				RanLeafSet:  map[string]struct{}{"TestExplicitFail": {}, "TestStillRunning": {}},
				FailedLeafs: map[string]struct{}{"TestExplicitFail": {}},
			},
		},
		{
			name: "does not attribute non-leaf parent crash to FailedLeafs",
			rawOut: `=== RUN   TestParent
=== RUN   TestParent/child
--- PASS: TestParent/child (0.00s)
=== CONT  TestParent
panic: parent cleanup failed
`,
			nonZero: true,
			want: VerboseTestResult{
				LeafOrder:   []string{"TestParent/child"},
				RanLeafSet:  map[string]struct{}{"TestParent/child": {}},
				FailedLeafs: map[string]struct{}{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseVerboseTestOutput([]byte(tc.rawOut), tc.nonZero)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseVerboseTestOutput() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
