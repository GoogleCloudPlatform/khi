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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDiscoverChangedTestFuncsAndAnalyzeRedundancy(t *testing.T) {
	testSrc := `package calc

import "testing"

func TestClamp(t *testing.T) {
	testCases := []struct {
		name string
		in   int
		want int
	}{
		{
			name: "existing negative",
			in:   -1,
			want: 0,
		},
		{
			name: "added duplicate negative",
			in:   -50,
			want: 0,
		},
		{
			name: "added boundary zero",
			in:   0,
			want: 0,
		},
		{
			name: "added positive",
			in:   10,
			want: 10,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Clamp(tc.in); got != tc.want {
				t.Errorf("Clamp(%d) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestClampSmokeOnlyErr(t *testing.T) {
	t.Run("literal subtest err only", func(t *testing.T) {
		if _, err := ClampChecked(-5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

type helperSuite struct{}

func (helperSuite) TestMethodIgnored(t *testing.T) {}
func TestMain(m *testing.M)                        {}
func Test(t *testing.T)                            {}
func TesthelperLowercase(t *testing.T)             {}
func TestZeroParams()                              {}
func BenchmarkClamp(b *testing.B)                  {}
`
	// Simulate PR diff adding "added duplicate negative" (lines 16-20), "added boundary zero" (21-25),
	// "added positive" (26-30), and the new TestClampSmokeOnlyErr (41-47), while "existing negative" (11-15) was pre-existing.
	changedTestLines := map[int]struct{}{
		16: {}, 17: {}, 18: {}, 19: {}, 20: {},
		21: {}, 22: {}, 23: {}, 24: {}, 25: {},
		26: {}, 27: {}, 28: {}, 29: {}, 30: {},
		41: {}, 42: {}, 43: {}, 44: {}, 45: {}, 46: {}, 47: {},
		51: {}, 52: {}, 53: {}, 54: {}, 55: {}, 56: {},
	}

	funcs, err := DiscoverChangedTestFuncs("pkg/calc/calc_test.go", []byte(testSrc), changedTestLines)
	if err != nil {
		t.Fatalf("DiscoverChangedTestFuncs() unexpected error: %v", err)
	}
	if len(funcs) != 3 {
		t.Fatalf("len(funcs) = %d, want 3 (TestClamp, TestClampSmokeOnlyErr, Test)", len(funcs))
	}

	leafOrder := []string{
		"TestClamp/existing_negative",
		"TestClamp/added_duplicate_negative",
		"TestClamp/added_boundary_zero",
		"TestClamp/added_positive",
		"TestClampSmokeOnlyErr/literal_subtest_err_only",
	}

	polarityByLeaf := map[string]map[string]struct{}{
		"TestClamp/existing_negative":                    {"1:T": {}, "2:T": {}},
		"TestClamp/added_duplicate_negative":             {"1:T": {}, "2:T": {}},
		"TestClamp/added_boundary_zero":                  {"1:F": {}, "2:F": {}},
		"TestClamp/added_positive":                       {"1:F": {}, "2:F": {}, "9:S": {}},
		"TestClampSmokeOnlyErr/literal_subtest_err_only": {"1:T": {}, "2:T": {}},
	}

	ranAll := map[string]struct{}{
		"TestClamp/existing_negative":                    {},
		"TestClamp/added_duplicate_negative":             {},
		"TestClamp/added_boundary_zero":                  {},
		"TestClamp/added_positive":                       {},
		"TestClampSmokeOnlyErr/literal_subtest_err_only": {},
	}

	mutantRuns := map[int]VerboseTestResult{
		1: {
			LeafOrder:   leafOrder,
			RanLeafSet:  ranAll,
			FailedLeafs: map[string]struct{}{},
		},
		2: {
			LeafOrder:  leafOrder,
			RanLeafSet: ranAll,
			FailedLeafs: map[string]struct{}{
				"TestClamp/existing_negative":        {},
				"TestClamp/added_duplicate_negative": {},
				"TestClamp/added_positive":           {},
			},
		},
	}

	got := AnalyzeTestRedundancy(funcs, leafOrder, polarityByLeaf, mutantRuns, map[int]struct{}{9: {}})

	// Expect:
	// 1. "added_duplicate_negative" is flagged DUPLICATE of pre-existing "existing_negative".
	// 2. "added_boundary_zero" is NOT flagged (has real value assertion and 0 kills at boundary fixed point).
	// 3. "added_positive" is NOT flagged (different polarity + touched budget-skipped mutant 9).
	// 4. "TestClampSmokeOnlyErr/literal_subtest_err_only" is flagged WEAK_TEST.
	want := []RedundantTestFinding{
		{
			Kind:        RedundantDuplicate,
			File:        "pkg/calc/calc_test.go",
			Line:        16,
			TestName:    "TestClamp/added_duplicate_negative",
			DominatedBy: "TestClamp/existing_negative",
			SharedKills: 1,
		},
		{
			Kind:        RedundantWeakAssertion,
			File:        "pkg/calc/calc_test.go",
			Line:        42,
			TestName:    "TestClampSmokeOnlyErr/literal_subtest_err_only",
			DominatedBy: "TestClamp/existing_negative",
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("AnalyzeTestRedundancy() mismatch (-want +got):\n%s", diff)
	}
}

func TestParseVerboseTestOutputExtractsLeafFailuresAndPanics(t *testing.T) {
	testCases := []struct {
		name           string
		raw            string
		exitedNonZero  bool
		wantLeafOrder  []string
		wantFailedKeys []string
	}{
		{
			name: "ignores parent test wrapper and keeps leaf subtests",
			raw: `=== RUN   TestClamp
=== RUN   TestClamp/neg
=== RUN   TestClamp/pos
--- FAIL: TestClamp (0.00s)
    --- FAIL: TestClamp/neg (0.00s)
    --- PASS: TestClamp/pos (0.00s)
=== RUN   TestSkipped
--- SKIP: TestSkipped (0.00s)
FAIL
`,
			exitedNonZero:  true,
			wantLeafOrder:  []string{"TestClamp/neg", "TestClamp/pos"},
			wantFailedKeys: []string{"TestClamp/neg"},
		},
		{
			name: "attributes unrecovered goroutine crash to active leaf",
			raw: `=== RUN   TestWorker
=== RUN   TestWorker/crash_case
panic: runtime error: index out of range [0] with length 0
`,
			exitedNonZero:  true,
			wantLeafOrder:  []string{"TestWorker/crash_case"},
			wantFailedKeys: []string{"TestWorker/crash_case"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := ParseVerboseTestOutput([]byte(tc.raw), tc.exitedNonZero)
			if diff := cmp.Diff(tc.wantLeafOrder, res.LeafOrder); diff != "" {
				t.Errorf("LeafOrder mismatch (-want +got):\n%s", diff)
			}
			var gotFailed []string
			for _, leaf := range res.LeafOrder {
				if _, ok := res.FailedLeafs[leaf]; ok {
					gotFailed = append(gotFailed, leaf)
				}
			}
			if diff := cmp.Diff(tc.wantFailedKeys, gotFailed); diff != "" {
				t.Errorf("FailedLeafs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
