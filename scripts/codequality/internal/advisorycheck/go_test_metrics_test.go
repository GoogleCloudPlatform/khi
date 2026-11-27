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

package advisorycheck

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestGoTestMetricsTableAndHelperAggregation verifies named/generic test tables, nested-slice non-double-counting, and bypass-proof helper assertion aggregation.
func TestGoTestMetricsTableAndHelperAggregation(t *testing.T) {
	testCases := []struct {
		name         string
		baseSrc      string
		targetSrc    string
		want         []advisoryCheckSnapshot
		wantEvidence map[string]any
	}{
		// Consolidating standalone cmp.Diff subtests into a generic []flagTestCase[T] table passed to runFlagTestCases[T] with scalar != emits zero false positives.
		{
			name: "generic table passed to generic helper with scalar assert has zero false positives",
			baseSrc: `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestString(t *testing.T) {
	t.Run("a", func(t *testing.T) {
		if d := cmp.Diff("a", parse("a")); d != "" {
			t.Errorf("mismatch: %s", d)
		}
	})
	t.Run("b", func(t *testing.T) {
		if d := cmp.Diff("b", parse("b")); d != "" {
			t.Errorf("mismatch: %s", d)
		}
	})
}
`,
			targetSrc: `package svc

import "testing"

type flagTestCase[T any] struct {
	name string
	arg  string
	want T
}

func runFlagTestCases[T comparable](t *testing.T, cases []flagTestCase[T], fn func(string) T) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fn(tc.arg); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	cases := []flagTestCase[string]{
		{name: "a", arg: "a", want: "a"},
		{name: "b", arg: "b", want: "b"},
	}
	runFlagTestCases[string](t, cases, parse)
}
`,
			want: nil,
		},
		// Multi-type-argument generic table ([]pairCase[string, int]) passed to runPairCases[string, int] counts rows accurately without false positives.
		{
			name: "multi-type-param generic table passed to helper has zero false positives",
			baseSrc: `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPairs(t *testing.T) {
	t.Run("one", func(t *testing.T) {
		if d := cmp.Diff(1, parseInt("1")); d != "" {
			t.Errorf("mismatch: %s", d)
		}
	})
	t.Run("two", func(t *testing.T) {
		if d := cmp.Diff(2, parseInt("2")); d != "" {
			t.Errorf("mismatch: %s", d)
		}
	})
}
`,
			targetSrc: `package svc

import "testing"

type pairCase[K any, V any] struct {
	name string
	in   K
	want V
}

func runPairCases[K any, V comparable](t *testing.T, cases []pairCase[K, V], fn func(K) V) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fn(tc.in); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPairs(t *testing.T) {
	runPairCases[string, int](t, []pairCase[string, int]{
		{name: "one", in: "1", want: 1},
		{name: "two", in: "2", want: 2},
	}, parseInt)
}
`,
			want: nil,
		},
		// Consolidating standalone subtests into a named local struct table iterated with t.Run produces zero false positives.
		{
			name: "named local struct table in for range with t.Run has zero false positives",
			baseSrc: `package svc

import "testing"

func TestNamedTable(t *testing.T) {
	t.Run("a", func(t *testing.T) {
		if parse("a") != "a" {
			t.Errorf("bad a")
		}
	})
	t.Run("b", func(t *testing.T) {
		if parse("b") != "b" {
			t.Errorf("bad b")
		}
	})
}
`,
			targetSrc: `package svc

import "testing"

type namedCase struct {
	name string
	want string
}

func TestNamedTable(t *testing.T) {
	cases := []namedCase{
		{name: "a", want: "a"},
		{name: "b", want: "b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if parse(tc.name) != tc.want {
				t.Errorf("bad %s", tc.name)
			}
		})
	}
}
`,
			want: nil,
		},
		// Nested struct slices inside a table row are not double-counted when shrinking a 2-row table to 1 row.
		{
			name: "nested slice inside table row is not double counted",
			baseSrc: `package svc

import "testing"

type nestedItem struct {
	val int
}

type outerCase struct {
	name  string
	items []nestedItem
}

func TestNestedSlice(t *testing.T) {
	cases := []outerCase{
		{name: "a", items: []nestedItem{{val: 1}, {val: 2}}},
		{name: "b", items: []nestedItem{{val: 3}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.items) == 0 {
				t.Errorf("empty")
			}
		})
	}
}
`,
			targetSrc: `package svc

import "testing"

type nestedItem struct {
	val int
}

type outerCase struct {
	name  string
	items []nestedItem
}

func TestNestedSlice(t *testing.T) {
	cases := []outerCase{
		{name: "a", items: []nestedItem{{val: 1}, {val: 2}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.items) == 0 {
				t.Errorf("empty")
			}
		})
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestCasesReduced, File: "pkg/svc/svc_test.go", Symbol: "TestNestedSlice"},
			},
			wantEvidence: map[string]any{"before_count": 2, "after_count": 1},
		},
		// Replacing cmp.Diff with a same-file helper that only checks err != nil still triggers test_assertion_weakened.
		{
			name: "err != nil only helper still triggers test_assertion_weakened",
			baseSrc: `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestErrOnlyHelper(t *testing.T) {
	if d := cmp.Diff("ok", run()); d != "" {
		t.Errorf("mismatch: %s", d)
	}
}
`,
			targetSrc: `package svc

import "testing"

func checkErrOnly(t *testing.T) {
	t.Helper()
	if _, err := runErr(); err != nil {
		t.Fatal(err)
	}
}

func TestErrOnlyHelper(t *testing.T) {
	checkErrOnly(t)
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestErrOnlyHelper"},
			},
		},
		// Replacing assertions with a same-file t.Log-only helper still triggers test_assertion_weakened.
		{
			name: "t.Log only helper still triggers test_assertion_weakened",
			baseSrc: `package svc

import "testing"

func TestLogOnlyHelper(t *testing.T) {
	if run() != "ok" {
		t.Errorf("bad")
	}
}
`,
			targetSrc: `package svc

import "testing"

func logOnly(t *testing.T) {
	t.Helper()
	t.Log("ran")
}

func TestLogOnlyHelper(t *testing.T) {
	logOnly(t)
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestLogOnlyHelper"},
			},
		},
		// Replacing assertions with direct t.Log still triggers test_assertion_weakened.
		{
			name: "direct t.Log bypass still triggers test_assertion_weakened",
			baseSrc: `package svc

import "testing"

func TestDirectLog(t *testing.T) {
	if run() != "ok" {
		t.Errorf("bad")
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestDirectLog(t *testing.T) {
	t.Log(run())
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestDirectLog"},
			},
		},
		// Replacing assertions with a zero-assert setup(t) helper still triggers test_assertion_weakened.
		{
			name: "zero assert setup helper still triggers test_assertion_weakened",
			baseSrc: `package svc

import "testing"

func TestSetupOnly(t *testing.T) {
	if run() != "ok" {
		t.Errorf("bad")
	}
}
`,
			targetSrc: `package svc

import "testing"

func setup(t *testing.T) {
	t.Helper()
	initFixture()
}

func TestSetupOnly(t *testing.T) {
	setup(t)
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestSetupOnly"},
			},
		},
		// Non-table expected slice (want := []item{{name: "a"}}) is not counted as a test table and still flags reduced cases.
		{
			name: "non-table expected slice is not counted as test table",
			baseSrc: `package svc

import "testing"

func TestWantSlice(t *testing.T) {
	t.Run("a", func(t *testing.T) {
		if run("a") != "a" {
			t.Errorf("bad a")
		}
	})
	t.Run("b", func(t *testing.T) {
		if run("b") != "b" {
			t.Errorf("bad b")
		}
	})
}
`,
			targetSrc: `package svc

import "testing"

type item struct {
	name string
}

func TestWantSlice(t *testing.T) {
	t.Run("a", func(t *testing.T) {
		want := []item{{name: "a"}, {name: "b"}}
		if len(want) != 2 {
			t.Errorf("bad len")
		}
		if want[0].name != "a" {
			t.Errorf("bad item")
		}
	})
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestCasesReduced, File: "pkg/svc/svc_test.go", Symbol: "TestWantSlice"},
			},
			wantEvidence: map[string]any{"before_count": 2, "after_count": 1},
		},
		// Replacing assertions with a local zero-assert closure still triggers test_assertion_weakened.
		{
			name: "local zero-assert closure bypass still triggers test_assertion_weakened",
			baseSrc: `package svc

import "testing"

func TestLocalClosureBypass(t *testing.T) {
	if run("a") != "a" {
		t.Errorf("bad a")
	}
	if run("b") != "b" {
		t.Errorf("bad b")
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestLocalClosureBypass(t *testing.T) {
	noop := func(t *testing.T) {}
	noop(t)
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestLocalClosureBypass"},
			},
		},
		// Replacing assertions with a var-bound zero-assert closure still triggers test_assertion_weakened.
		{
			name: "var-bound zero-assert closure still triggers test_assertion_weakened",
			baseSrc: `package svc

import "testing"

func TestVarClosureBypass(t *testing.T) {
	if run("a") != "a" {
		t.Errorf("bad a")
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestVarClosureBypass(t *testing.T) {
	var noop func(*testing.T)
	noop = func(t *testing.T) {}
	noop(t)
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestVarClosureBypass"},
			},
		},
		// Mutual recursive same-file helpers terminate safely under cycle guarding and still detect weakened assertions.
		{
			name: "recursive helper cycle terminates safely",
			baseSrc: `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRecursiveHelpers(t *testing.T) {
	if d := cmp.Diff("ok", run()); d != "" {
		t.Errorf("mismatch: %s", d)
	}
}
`,
			targetSrc: `package svc

import "testing"

type recCase struct {
	name string
}

func recA(t *testing.T, cases []recCase) {
	t.Helper()
	recB(t, cases)
}

func recB(t *testing.T, cases []recCase) {
	t.Helper()
	if len(cases) > 0 {
		recA(t, cases[1:])
	}
}

func TestRecursiveHelpers(t *testing.T) {
	cases := []recCase{{name: "a"}, {name: "b"}}
	recA(t, cases)
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestRecursiveHelpers"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeRepoFiles(t, repoRoot, map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": tc.targetSrc,
			})
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: []string{"pkg/svc/svc_test.go"},
				ReadBaseFile: func(rel string) ([]byte, bool, error) {
					if rel == "pkg/svc/svc_test.go" {
						return []byte(tc.baseSrc), true, nil
					}
					return nil, false, nil
				},
			}

			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}

			assertAdvisorySnapshots(t, items, tc.want)
			if tc.wantEvidence != nil && len(items) > 0 {
				if diff := cmp.Diff(tc.wantEvidence, items[0].Evidence); diff != "" {
					t.Errorf("evidence mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestGoTestMetricsReceiverAwareAsserts verifies that non-testing .Error/.Errorf calls (status.Error, fmt.Errorf, err.Error) are excluded while testing receivers and h.t.Errorf are recognized.
func TestGoTestMetricsReceiverAwareAsserts(t *testing.T) {
	testCases := []struct {
		name      string
		baseSrc   string
		targetSrc string
		want      []advisoryCheckSnapshot
	}{
		// Removing status.Error and fmt.Errorf helper constructions while keeping t.Errorf produces zero false positives.
		{
			name: "removing status.Error and fmt.Errorf has zero false positives",
			baseSrc: `package svc
import ("fmt"; "testing")
func TestErrors(t *testing.T) {
	wantErr := status.Error(3, "bad")
	_ = fmt.Errorf("wrap: %w", wantErr)
	if got := run(); got != "ok" { t.Errorf("got %q, want ok", got) }
}
`,
			targetSrc: `package svc
import "testing"
func TestErrors(t *testing.T) {
	if got := run(); got != "ok" { t.Errorf("got %q, want ok", got) }
}
`,
			want: nil,
		},
		// Replacing t.Errorf with fmt.Errorf still triggers test_assertion_weakened.
		{
			name: "replacing t.Errorf with fmt.Errorf triggers test_assertion_weakened",
			baseSrc: `package svc
import "testing"
func TestReplaceFmtErrorf(t *testing.T) {
	if got := run(); got != "ok" { t.Errorf("bad: %v", got) }
}
`,
			targetSrc: `package svc
import ("fmt"; "testing")
func TestReplaceFmtErrorf(t *testing.T) {
	if got := run(); got != "ok" { err := fmt.Errorf("bad: %v", got); _ = err }
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestReplaceFmtErrorf"},
			},
		},
		// Replacing t.Fatal with status.Error still triggers test_assertion_weakened.
		{
			name: "replacing t.Fatal with status.Error triggers test_assertion_weakened",
			baseSrc: `package svc
import "testing"
func TestReplaceStatusError(t *testing.T) {
	if got := run(); got != "ok" { t.Fatal("bad") }
}
`,
			targetSrc: `package svc
import "testing"
func TestReplaceStatusError(t *testing.T) {
	if got := run(); got != "ok" { err := status.Error(3, "bad"); _ = err }
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestReplaceStatusError"},
			},
		},
		// Subtest parameter subT.Fatalf is recognized as a testing assertion and detects removal when replaced by errors.Errorf.
		{
			name: "subtest param subT.Fatalf replaced by errors.Errorf triggers",
			baseSrc: `package svc
import "testing"
func TestSubtestReceiver(t *testing.T) {
	t.Run("case", func(subT *testing.T) {
		if got := run(); got != "ok" { subT.Fatalf("bad: %v", got) }
	})
}
`,
			targetSrc: `package svc
import "testing"
func TestSubtestReceiver(t *testing.T) {
	t.Run("case", func(subT *testing.T) {
		if got := run(); got != "ok" { err := errors.Errorf("bad: %v", got); _ = err }
	})
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestSubtestReceiver"},
			},
		},
		// Struct field receiver h.t.Errorf is recognized as a testing assertion when refactored to direct t.Errorf.
		{
			name: "struct field receiver h.t.Errorf recognized as testing assert",
			baseSrc: `package svc
import "testing"
func TestFieldReceiver(t *testing.T) {
	h := struct{ t *testing.T }{t: t}
	if got := run(); got != "ok" { h.t.Errorf("bad: %v", got) }
}
`,
			targetSrc: `package svc
import "testing"
func TestFieldReceiver(t *testing.T) {
	if got := run(); got != "ok" { t.Errorf("bad: %v", got) }
}
`,
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeRepoFiles(t, repoRoot, map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": tc.targetSrc,
			})
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: []string{"pkg/svc/svc_test.go"},
				ReadBaseFile: func(rel string) ([]byte, bool, error) {
					if rel == "pkg/svc/svc_test.go" {
						return []byte(tc.baseSrc), true, nil
					}
					return nil, false, nil
				},
			}

			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			assertAdvisorySnapshots(t, items, tc.want)
		})
	}
}

// TestIsTestingMethodAssertEmptyParams verifies empty testingParams handling for method field receivers like h.t.Errorf while rejecting status/fmt/errors.
func TestIsTestingMethodAssertEmptyParams(t *testing.T) {
	testCases := []struct {
		name          string
		exprSrc       string
		testingParams map[string]bool
		want          bool
	}{
		// Empty testingParams recognizes direct t.Errorf on a testing receiver identifier.
		{name: "empty params accepts t.Errorf", exprSrc: `t.Errorf("bad")`, want: true},
		// Empty testingParams recognizes h.t.Errorf on a struct field receiver.
		{name: "empty params accepts h.t.Errorf", exprSrc: `h.t.Errorf("bad")`, want: true},
		// Empty testingParams rejects status.Error.
		{name: "empty params rejects status.Error", exprSrc: `status.Error(3, "bad")`, want: false},
		// Empty testingParams rejects fmt.Errorf.
		{name: "empty params rejects fmt.Errorf", exprSrc: `fmt.Errorf("bad")`, want: false},
		// Empty testingParams rejects errors.Errorf.
		{name: "empty params rejects errors.Errorf", exprSrc: `errors.Errorf("bad")`, want: false},
		// Non-empty testingParams accepts custom param name tb.FailNow().
		{name: "non-empty params accepts tb.FailNow", exprSrc: `tb.FailNow()`, testingParams: map[string]bool{"tb": true}, want: true},
		// Non-empty testingParams rejects obj.err.Error().
		{name: "non-empty params rejects obj.err.Error", exprSrc: `obj.err.Error()`, testingParams: map[string]bool{"t": true}, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tc.exprSrc)
			if err != nil {
				t.Fatalf("ParseExpr() unexpected error: %v", err)
			}
			call, ok := expr.(*ast.CallExpr)
			if !ok {
				t.Fatalf("ParseExpr() = %T, want *ast.CallExpr", expr)
			}

			got := isTestingMethodAssert(call, tc.testingParams)

			if got != tc.want {
				t.Errorf("isTestingMethodAssert(%s) = %v, want %v", tc.exprSrc, got, tc.want)
			}
		})
	}
}

// TestGoTestMetricsFormattingCmpDiff verifies that formatting-only cmp.Diff inside t.Error* arguments is not double-counted while genuine cmp.Diff/cmp.Equal removals still trigger.
func TestGoTestMetricsFormattingCmpDiff(t *testing.T) {
	testCases := []struct {
		name      string
		baseSrc   string
		targetSrc string
		want      []advisoryCheckSnapshot
	}{
		// Replacing formatting-only cmp.Diff inside t.Errorf with %d/%q produces zero false positives.
		{
			name: "formatting-only cmp.Diff inside t.Errorf replaced by %d has zero false positives",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestFormatOnlyCmpDiff(t *testing.T) {
	if got := parse(); got != "want" { t.Errorf("mismatch:\n%s", cmp.Diff("want", got)) }
}
`,
			targetSrc: `package svc
import "testing"
func TestFormatOnlyCmpDiff(t *testing.T) {
	if got := parse(); got != "want" { t.Errorf("got %q, want %q", got, "want") }
}
`,
			want: nil,
		},
		// Refactoring if !cmp.Equal(...) { t.Errorf(..., cmp.Diff(...)) } to if diff := cmp.Diff(...); diff != "" preserves cmpAsserts=1 with zero false positives.
		{
			name: "cmp.Equal guard with formatting cmp.Diff refactored to cmp.Diff guard has zero false positives",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestEqualWithFormatDiff(t *testing.T) {
	if got := parse(); !cmp.Equal("want", got) { t.Errorf("mismatch:\n%s", cmp.Diff("want", got)) }
}
`,
			targetSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestEqualWithFormatDiff(t *testing.T) {
	if diff := cmp.Diff("want", parse()); diff != "" { t.Errorf("mismatch:\n%s", diff) }
}
`,
			want: nil,
		},
		// Replacing genuine cmp.Diff guard with weak nil guard and formatting-only cmp.Diff inside t.Errorf still triggers test_assertion_weakened.
		{
			name: "bypassing genuine cmp.Diff via nil guard and t.Errorf format cmp.Diff triggers",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestBypassFormatCmpDiff(t *testing.T) {
	if diff := cmp.Diff("want", parse()); diff != "" { t.Errorf("mismatch:\n%s", diff) }
}
`,
			targetSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestBypassFormatCmpDiff(t *testing.T) {
	if got := parse(); got != nil { t.Errorf("mismatch:\n%s", cmp.Diff("want", got)) }
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestBypassFormatCmpDiff"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeRepoFiles(t, repoRoot, map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": tc.targetSrc,
			})
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: []string{"pkg/svc/svc_test.go"},
				ReadBaseFile: func(rel string) ([]byte, bool, error) {
					if rel == "pkg/svc/svc_test.go" {
						return []byte(tc.baseSrc), true, nil
					}
					return nil, false, nil
				},
			}

			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			assertAdvisorySnapshots(t, items, tc.want)
		})
	}
}

// TestInspectGoTestBodyMetricsDirectCounts verifies exact cmpAsserts, scalarAsserts, and totalAsserts for formatting-only vs genuine cmp.Diff/cmp.Equal calls.
func TestInspectGoTestBodyMetricsDirectCounts(t *testing.T) {
	src := `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestFormatOnly(t *testing.T) {
	if got != want { t.Errorf("diff:\n%s", cmp.Diff(want, got)) }
}
func TestGenuineDiff(t *testing.T) {
	if diff := cmp.Diff(want, got); diff != "" { t.Errorf("diff:\n%s", diff) }
}
func TestInlineCmpCond(t *testing.T) {
	if cmp.Diff(want, got) != "" { t.Errorf("diff") }
}
func TestCmpEqualCond(t *testing.T) {
	if !cmp.Equal(want, got) { t.Errorf("diff") }
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "svc_test.go", []byte(src), 0)
	if err != nil {
		t.Fatalf("ParseFile() unexpected error: %v", err)
	}
	summaries := collectGoTestFuncSummaries(fset, file)
	if len(summaries) != 4 {
		t.Fatalf("got %d summaries, want 4", len(summaries))
	}
	want := []struct {
		name   string
		cmp    int
		scalar int
		total  int
	}{
		{name: "TestFormatOnly", cmp: 0, scalar: 1, total: 1},
		{name: "TestGenuineDiff", cmp: 1, scalar: 0, total: 2},
		{name: "TestInlineCmpCond", cmp: 1, scalar: 0, total: 2},
		{name: "TestCmpEqualCond", cmp: 1, scalar: 0, total: 2},
	}
	for i, w := range want {
		s := summaries[i]
		if s.name != w.name || s.cmpAssertCount != w.cmp || s.scalarAssertCount != w.scalar || s.totalAsserts != w.total {
			t.Errorf("summary[%d] = (%s cmp=%d scalar=%d total=%d), want (%s cmp=%d scalar=%d total=%d)",
				i, s.name, s.cmpAssertCount, s.scalarAssertCount, s.totalAsserts, w.name, w.cmp, w.scalar, w.total)
		}
	}
}
