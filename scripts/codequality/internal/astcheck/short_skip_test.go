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

package astcheck

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCheckConventionsNoShortSkip(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  []ConventionViolation
	}{
		// Allows parameter scale-down, if/else parameter assignment, value-returning helpers, nested closure returns, and non-short skips.
		{
			name: "allowed patterns and non-short constructs produce no violations",
			files: map[string]string{
				"pkg/sample/valid_test.go": `package sample

import (
	"os"
	"testing"
)

func externalDecl()

func noop() {}

func pair() (int, int) { return 1, 2 }

type fakePkg struct{}

func (fakePkg) Short() bool { return false }

func (fakePkg) Other() bool { return false }

func stressCount() int {
	if testing.Short() {
		return 10
	}
	return 1000
}

func TestAllowedPatterns(t *testing.T) {
	attempts := 1000
	if testing.Short() {
		attempts = 5
		t.Log("scaled")
		noop()
	}
	var n int
	if testing.Short() {
		n = 10
	} else {
		n = 1000
	}
	if testing.Short() {
		fn := func() {
			return
		}
		_ = fn
	}
	if os.Getenv("RUN_LIVE_TESTS") != "true" {
		t.Skip("live tests disabled")
	}
	flag := false
	if !flag || attempts < 1 || attempts == n {
		_ = stressCount()
	}
	var other fakePkg
	if !other.Short() || !other.Other() {
		_ = n
	}
	if flag == testing.Short() {
		_ = n
	}
	_ = testing.Short()
	a, b := pair()
	arr := []int{0}
	arr[0] = a + b
	short := testing.Short()
	short = false
	if short {
		t.Skip("reassigned variable is not testing.Short()")
	}
	if noop(); flag {
		_ = n
	}
	switch testing.Short() {
	case true:
		n = 10
	case false:
		n = 1000
	}
	switch testing.Short() {
	case true:
	}
	switch attempts {
	case 5:
		n = 5
	}
	switch {
	case testing.Short():
		n = 10
	default:
		n = 1000
	}
	switch {
	case testing.Short():
	}
	switch {
	case flag:
		n = 1
	}
	s := false
	if s := testing.Short(); s {
		attempts = 5
	}
	if !s {
		t.Log("outer s after if-init shadow")
	}
	if flag {
		s := testing.Short()
		if s {
			attempts = 5
		}
	}
	if !s {
		t.Log("outer s after block shadow")
	}
	switch s := testing.Short(); {
	case s:
		attempts = 5
	}
	if !s {
		t.Log("outer s after switch-init shadow")
	}
}
`,
				"pkg/sample/blank_test.go": `package sample

import _ "testing"

func helperBlank() {}
`,
				"pkg/sample/notesting_test.go": `package sample

func helperNoTesting() {}
`,
			},
			want: nil,
		},
		// Reports Skip, Skipf, and SkipNow calls guarded by testing.Short() in both test files and non-test test helpers.
		{
			name: "forbids Skip, Skipf, and SkipNow inside if testing.Short()",
			files: map[string]string{
				"pkg/sample/skip_test.go": `package sample

import "testing"

func TestSkipCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("skip")
	}
	if testing.Short() {
		t.Skipf("%s", "skipf")
	}
}
`,
				"pkg/sample/helper.go": `package sample

import "testing"

// SkipHelper demonstrates that non-test Go files importing testing are also checked.
func SkipHelper(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
}
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/sample/helper.go",
					Line:    8,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
				{
					File:    "pkg/sample/skip_test.go",
					Line:    7,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
				{
					File:    "pkg/sample/skip_test.go",
					Line:    10,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
			},
		},
		// Reports Skip calls and early returns under compound logical, equality, parenthesized, and double-negated testing.Short() conditions.
		{
			name: "forbids compound conditions with testing.Short()",
			files: map[string]string{
				"pkg/sample/compound_test.go": `package sample

import "testing"

func TestCompound(t *testing.T) {
	fast := false
	ok := true
	if (testing.Short()) || fast {
		t.Skip("or")
	}
	if ok && testing.Short() {
		return
	}
	if testing.Short() == true {
		t.Skip("eq true")
	}
	if false != testing.Short() {
		t.Skip("false neq")
	}
	if !!testing.Short() {
		t.Skip("double not")
	}
}
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/sample/compound_test.go",
					Line:    9,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
				{
					File:    "pkg/sample/compound_test.go",
					Line:    12,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEarlyReturn,
				},
				{
					File:    "pkg/sample/compound_test.go",
					Line:    15,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
				{
					File:    "pkg/sample/compound_test.go",
					Line:    18,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
				{
					File:    "pkg/sample/compound_test.go",
					Line:    21,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
			},
		},
		// Reports bare early returns guarded by testing.Short() in test functions, subtest closures, and named-return helpers.
		{
			name: "forbids early return in void function, subtest closure, and named return",
			files: map[string]string{
				"pkg/sample/early_return_test.go": `package sample

import "testing"

func TestEarlyReturn(t *testing.T) {
	if testing.Short() {
		return
	}
	t.Run("sub", func(t *testing.T) {
		if testing.Short() {
			return
		}
	})
}

func namedReturn() (n int) {
	if testing.Short() {
		return
	}
	return 10
}
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/sample/early_return_test.go",
					Line:    7,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEarlyReturn,
				},
				{
					File:    "pkg/sample/early_return_test.go",
					Line:    11,
					Column:  4,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEarlyReturn,
				},
				{
					File:    "pkg/sample/early_return_test.go",
					Line:    18,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEarlyReturn,
				},
			},
		},
		// Reports negated testing.Short() guards including unary negation, boolean equality/inequality, and compound negated conditions.
		{
			name: "forbids negated testing.Short() guards",
			files: map[string]string{
				"pkg/sample/negated_test.go": `package sample

import "testing"

func TestNegated(t *testing.T) {
	ok := true
	if !testing.Short() {
		_ = ok
	}
	if testing.Short() == false {
		_ = ok
	}
	if false == testing.Short() {
		_ = ok
	}
	if testing.Short() != true {
		_ = ok
	}
	if true != testing.Short() {
		_ = ok
	}
	if ok && !testing.Short() {
		_ = ok
	}
	if !testing.Short() || ok {
		_ = ok
	}
}
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/sample/negated_test.go",
					Line:    7,
					Column:  5,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
				{
					File:    "pkg/sample/negated_test.go",
					Line:    10,
					Column:  5,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
				{
					File:    "pkg/sample/negated_test.go",
					Line:    13,
					Column:  5,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
				{
					File:    "pkg/sample/negated_test.go",
					Line:    16,
					Column:  5,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
				{
					File:    "pkg/sample/negated_test.go",
					Line:    19,
					Column:  5,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
				{
					File:    "pkg/sample/negated_test.go",
					Line:    22,
					Column:  5,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
				{
					File:    "pkg/sample/negated_test.go",
					Line:    25,
					Column:  5,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
			},
		},
		// Reports an empty if testing.Short() {} else { ... } branch as equivalent to a negated guard.
		{
			name: "forbids empty if testing.Short() with else branch",
			files: map[string]string{
				"pkg/sample/empty_else_test.go": `package sample

import "testing"

func TestEmptyElse(t *testing.T) {
	if testing.Short() {
	} else {
		t.Log("heavy")
	}
}
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/sample/empty_else_test.go",
					Line:    6,
					Column:  2,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEmptyBranch,
				},
			},
		},
		// Reports skips, early returns, and negated guards routed through local aliases, var declarations, if-init bindings, and closures.
		{
			name: "forbids local variable aliases to testing.Short()",
			files: map[string]string{
				"pkg/sample/alias_var_test.go": `package sample

import "testing"

func TestAliasVars(t *testing.T) {
	short := testing.Short()
	if short {
		t.Skip("alias")
	}
	var notShort = !testing.Short()
	if notShort {
		t.Log("heavy")
	}
	if s := testing.Short(); s {
		return
	}
	t.Run("captured", func(t *testing.T) {
		if short {
			t.Skip("captured")
		}
	})
}
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/sample/alias_var_test.go",
					Line:    8,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
				{
					File:    "pkg/sample/alias_var_test.go",
					Line:    11,
					Column:  5,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
				{
					File:    "pkg/sample/alias_var_test.go",
					Line:    15,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEarlyReturn,
				},
				{
					File:    "pkg/sample/alias_var_test.go",
					Line:    19,
					Column:  4,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
			},
		},
		// Reports tagless and tagged switch statements that skip, return early, use empty short cases, or guard with negated polarity.
		{
			name: "forbids switch bypasses on testing.Short()",
			files: map[string]string{
				"pkg/sample/switch_test.go": `package sample

import "testing"

func TestSwitchBypass(t *testing.T) {
	switch {
	case testing.Short():
		t.Skip("tagless skip")
	}
	switch {
	case !testing.Short():
		t.Log("heavy")
	}
	switch {
	case testing.Short():
	default:
		t.Log("heavy default")
	}
	switch testing.Short() {
	case true:
		return
	}
	switch testing.Short() {
	case false:
		t.Log("heavy false")
	}
	switch testing.Short() {
	case true:
	default:
		t.Log("heavy tagged default")
	}
	switch !testing.Short() {
	case true:
		t.Log("negated tag")
	}
	switch testing.Short() {
	case true:
	case false:
		t.Log("heavy empty true and false")
	}
}
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/sample/switch_test.go",
					Line:    8,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
				{
					File:    "pkg/sample/switch_test.go",
					Line:    11,
					Column:  7,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
				{
					File:    "pkg/sample/switch_test.go",
					Line:    15,
					Column:  2,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEmptyBranch,
				},
				{
					File:    "pkg/sample/switch_test.go",
					Line:    21,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEarlyReturn,
				},
				{
					File:    "pkg/sample/switch_test.go",
					Line:    24,
					Column:  2,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
				{
					File:    "pkg/sample/switch_test.go",
					Line:    28,
					Column:  2,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEmptyBranch,
				},
				{
					File:    "pkg/sample/switch_test.go",
					Line:    32,
					Column:  9,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
				{
					File:    "pkg/sample/switch_test.go",
					Line:    37,
					Column:  2,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEmptyBranch,
				},
				{
					File:    "pkg/sample/switch_test.go",
					Line:    38,
					Column:  2,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortNegatedGuard,
				},
			},
		},
		// Resolves aliased and dot imports of the testing package when checking testing.Short() calls.
		{
			name: "detects violations with aliased and dot imports of testing",
			files: map[string]string{
				"pkg/alias/alias_test.go": `package alias

import tpkg "testing"

func TestAliasedImport(t *tpkg.T) {
	if tpkg.Short() {
		t.Skip("aliased")
	}
}
`,
				"pkg/dot/dot_test.go": `package dot

import . "testing"

func otherFunc() bool { return false }

func TestDotImport(t *T) {
	if !otherFunc() {
		_ = t
	}
	if Short() {
		t.Skip("dot")
	}
}
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/alias/alias_test.go",
					Line:    7,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
				{
					File:    "pkg/dot/dot_test.go",
					Line:    12,
					Column:  3,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
			},
		},
		// Reports nested testing.Short() guards once without duplicates and walks else branches and non-short switch clauses.
		{
			name: "reports nested short guards once and checks else and non-short switch bodies",
			files: map[string]string{
				"pkg/nested/nested_test.go": `package nested

import "testing"

func TestNested(t *testing.T) {
	if testing.Short() {
		if testing.Short() {
			t.Skip("nested if")
		}
		switch testing.Short() {
		case true:
			return
		}
	}
	flag := false
	if flag {
		t.Log("then")
	} else {
		if testing.Short() {
			t.Skip("in else")
		}
	}
	mode := 1
	switch mode {
	case 1:
		if testing.Short() {
			t.Skip("in non-short switch")
		}
	}
}
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/nested/nested_test.go",
					Line:    8,
					Column:  4,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
				{
					File:    "pkg/nested/nested_test.go",
					Line:    12,
					Column:  4,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortEarlyReturn,
				},
				{
					File:    "pkg/nested/nested_test.go",
					Line:    20,
					Column:  4,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
				{
					File:    "pkg/nested/nested_test.go",
					Line:    27,
					Column:  4,
					Rule:    RuleNoShortSkip,
					Message: msgNoShortSkipCall,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeConvFixtureRepo(t, tc.files)

			got, err := CheckConventions(root, "./...")

			if err != nil {
				t.Fatalf("CheckConventions() error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("CheckConventions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
