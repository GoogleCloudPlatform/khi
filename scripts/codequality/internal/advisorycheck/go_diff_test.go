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
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type advisoryCheckSnapshot struct {
	Check  string
	File   string
	Symbol string
}

// TestAnalyzeGoTestRemovedAndCasesReduced verifies test_removed (including renamed_candidates and deleted files) and test_cases_reduced (including zero-FP table refactoring).
func TestAnalyzeGoTestRemovedAndCasesReduced(t *testing.T) {
	testCases := []struct {
		name         string
		targetFiles  map[string]string
		baseFiles    map[string]string
		changedFiles []string
		deletedFiles []string
		want         []advisoryCheckSnapshot
		wantCands    []string
		wantEvidence map[string]any
	}{
		// Removing TestOld while keeping TestKeep and adding TestNew, TestMain, and a helper excludes non-new Test* names from renamed_candidates.
		{
			name: "removed test attaches only newly added Test* as renamed candidate",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": "package svc\n",
				"pkg/svc/svc_test.go": `package svc

import "testing"

func TestMain(m *testing.M) {}
func helperFunc(t *testing.T) {}
func TestKeep(t *testing.T) {}
func TestNew(t *testing.T) {}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import "testing"

func TestOld(t *testing.T) {}
func TestKeep(t *testing.T) {}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestRemoved, File: "pkg/svc/svc_test.go", Symbol: "TestOld"},
			},
			wantCands: []string{"TestNew"},
		},
		// Refactoring two standalone t.Run blocks into an equivalent 2-row table keeps case count equal (0 FP).
		{
			name: "refactoring standalone t.Runs to table has zero false positives",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": "package svc\n",
				"pkg/svc/svc_test.go": `package svc

import "testing"

func TestCases(t *testing.T) {
	cases := []struct{ name string }{{name: "a"}, {name: "b"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {})
	}
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import "testing"

func TestCases(t *testing.T) {
	t.Run("a", func(t *testing.T) {})
	t.Run("b", func(t *testing.T) {})
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go"},
			want:         nil,
		},
		// Shrinking a test table from 2 rows to 1 row triggers test_cases_reduced.
		{
			name: "table case reduction triggers test_cases_reduced",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": "package svc\n",
				"pkg/svc/svc_test.go": `package svc

import "testing"

func TestCases(t *testing.T) {
	cases := []struct{ name string }{{name: "a"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {})
	}
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import "testing"

func TestCases(t *testing.T) {
	cases := []struct{ name string }{{name: "a"}, {name: "b"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {})
	}
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestCasesReduced, File: "pkg/svc/svc_test.go", Symbol: "TestCases"},
			},
		},
		// Deleting an entire Go test file emits test_removed with test_name and renamed_candidates evidence.
		{
			name: "deleted go test file emits test_removed with evidence",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": "package svc\n",
			},
			baseFiles: map[string]string{
				"pkg/svc/old_test.go": `package svc

import "testing"

func TestDeleted(t *testing.T) {}
`,
			},
			deletedFiles: []string{"pkg/svc/old_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestRemoved, File: "pkg/svc/old_test.go", Symbol: "TestDeleted"},
			},
			wantEvidence: map[string]any{"test_name": "TestDeleted", "renamed_candidates": []string(nil)},
		},
		// Deleting a helper-only Go test file with zero Test* functions falls back to test file deleted.
		{
			name: "deleted zero-test helper file emits file deletion fallback",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": "package svc\n",
			},
			baseFiles: map[string]string{
				"pkg/svc/helper_test.go": `package svc

import "testing"

func BenchmarkRun(b *testing.B) {}
`,
			},
			deletedFiles: []string{"pkg/svc/helper_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestRemoved, File: "pkg/svc/helper_test.go", Symbol: "helper_test.go"},
			},
			wantEvidence: map[string]any{"deleted_file": "pkg/svc/helper_test.go", "renamed_candidates": []string(nil)},
		},
		// Deleting a Go test file whose base bytes are unavailable falls back to test file deleted.
		{
			name: "deleted file without base bytes emits file deletion fallback",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": "package svc\n",
			},
			deletedFiles: []string{"pkg/svc/missing_base_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestRemoved, File: "pkg/svc/missing_base_test.go", Symbol: "missing_base_test.go"},
			},
			wantEvidence: map[string]any{"deleted_file": "pkg/svc/missing_base_test.go", "renamed_candidates": []string(nil)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeRepoFiles(t, repoRoot, tc.targetFiles)
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: tc.changedFiles,
				DeletedFiles: tc.deletedFiles,
				ReadBaseFile: func(rel string) ([]byte, bool, error) {
					s, ok := tc.baseFiles[rel]
					return []byte(s), ok, nil
				},
			}

			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			assertAdvisorySnapshots(t, items, tc.want)
			if tc.wantCands != nil {
				gotCands, _ := items[0].Evidence["renamed_candidates"].([]string)
				if diff := cmp.Diff(tc.wantCands, gotCands); diff != "" {
					t.Errorf("renamed_candidates mismatch (-want +got):\n%s", diff)
				}
			}
			if tc.wantEvidence != nil {
				if diff := cmp.Diff(tc.wantEvidence, items[0].Evidence); diff != "" {
					t.Errorf("evidence mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestAnalyzeGoAssertionWeakenedAndSkipAdded verifies test_assertion_weakened (cmp.Diff removal, count reduction without helper, full discard) and test_skip_added (excluding testing.Short()).
func TestAnalyzeGoAssertionWeakenedAndSkipAdded(t *testing.T) {
	testCases := []struct {
		name      string
		targetSrc string
		baseSrc   string
		changed   map[int]struct{}
		want      []advisoryCheckSnapshot
	}{
		// Replacing cmp.Diff with no assertion triggers test_assertion_weakened.
		{
			name: "cmp.Diff removed triggers test_assertion_weakened",
			baseSrc: `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDiff(t *testing.T) {
	if d := cmp.Diff(1, 2); d != "" {
		t.Errorf("diff: %s", d)
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestDiff(t *testing.T) {
	_ = 1
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestDiff"},
			},
		},
		// Replacing cmp.Diff with a scalar != assertion prevents false-positive test_assertion_weakened.
		{
			name: "cmp.Diff replaced by scalar != has zero false positives",
			baseSrc: `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFlag(t *testing.T) {
	got := parse()
	if diff := cmp.Diff("ok", got); diff != "" {
		t.Errorf("mismatch: %s", diff)
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestFlag(t *testing.T) {
	got := parse()
	if got != "ok" {
		t.Errorf("got %q, want %q", got, "ok")
	}
}
`,
			want: nil,
		},
		// Consolidating ad-hoc t.Run blocks into a table-driven loop prevents false-positive test_assertion_weakened.
		{
			name: "ad-hoc t.Run consolidated into table has zero false positives",
			baseSrc: `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSeries(t *testing.T) {
	t.Run("a", func(t *testing.T) {
		if diff := cmp.Diff("a", run("a")); diff != "" {
			t.Errorf("mismatch: %s", diff)
		}
	})
	t.Run("b", func(t *testing.T) {
		if diff := cmp.Diff("b", run("b")); diff != "" {
			t.Errorf("mismatch: %s", diff)
		}
	})
}
`,
			targetSrc: `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSeries(t *testing.T) {
	testCases := []struct{ name, in, want string }{
		{name: "a", in: "a", want: "a"},
		{name: "b", in: "b", want: "b"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, run(tc.in)); diff != "" {
				t.Errorf("mismatch: %s", diff)
			}
		})
	}
}
`,
			want: nil,
		},
		// Replacing cmp.Diff with only err != nil still triggers test_assertion_weakened.
		{
			name: "cmp.Diff replaced by err != nil only still triggers",
			baseSrc: `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	got, err := parse()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff("ok", got); diff != "" {
		t.Errorf("mismatch: %s", diff)
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestParse(t *testing.T) {
	_, err := parse()
	if err != nil {
		t.Fatal(err)
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestParse"},
			},
		},
		// Converting standalone t.Runs into a table with zero assertions still triggers test_assertion_weakened.
		{
			name: "table conversion with zero asserts still triggers",
			baseSrc: `package svc

import "testing"

func TestTableZero(t *testing.T) {
	t.Run("a", func(t *testing.T) {
		if run("a") != "a" {
			t.Errorf("bad")
		}
	})
	t.Run("b", func(t *testing.T) {
		if run("b") != "b" {
			t.Errorf("bad")
		}
	})
}
`,
			targetSrc: `package svc

import "testing"

func TestTableZero(t *testing.T) {
	testCases := []struct{ name, in string }{
		{name: "a", in: "a"},
		{name: "b", in: "b"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(tc.in)
		})
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestTableZero"},
			},
		},
		// Adding a row to an existing table while dropping cmp.Diff still triggers test_assertion_weakened.
		{
			name: "existing table plus one row with dropped cmp.Diff still triggers",
			baseSrc: `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExistingTable(t *testing.T) {
	testCases := []struct{ name, want string }{
		{name: "a", want: "a"},
		{name: "b", want: "b"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := run(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch: %s", diff)
			}
		})
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestExistingTable(t *testing.T) {
	testCases := []struct{ name, want string }{
		{name: "a", want: "a"},
		{name: "b", want: "b"},
		{name: "c", want: "c"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := run(tc.name)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestExistingTable"},
			},
		},
		// Extracting assertions into a new helper call prevents false-positive test_assertion_weakened.
		{
			name: "extracting assertions into helper has zero false positives",
			baseSrc: `package svc

import "testing"

func TestHelper(t *testing.T) {
	if 1 != 1 {
		t.Errorf("bad 1")
	}
	if 2 != 2 {
		t.Errorf("bad 2")
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestHelper(t *testing.T) {
	assertBoth(t)
}

func assertBoth(t *testing.T) {
	if 1 != 1 {
		t.Errorf("bad 1")
	}
}
`,
			want: nil,
		},
		// Adding `_ = call()` triggers test_assertion_weakened while idiomatic `_, err := call()` does not.
		{
			name: "full discard triggers while multi-lhs err assignment does not",
			baseSrc: `package svc

import "testing"

func TestDiscard(t *testing.T) {
	_, err := runOK()
	if err != nil {
		t.Fatal(err)
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestDiscard(t *testing.T) {
	_, err := runOK()
	if err != nil {
		t.Fatal(err)
	}
	_ = runOther()
}
`,
			changed: map[int]struct{}{10: {}},
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestDiscard"},
			},
		},
		// Adding t.Skip outside testing.Short() triggers test_skip_added while scaling inside if testing.Short() does not.
		{
			name: "skip outside testing.Short triggers while inside testing.Short does not",
			baseSrc: `package svc

import "testing"

func TestSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("legacy")
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("legacy")
	}
	t.Skip("flaky")
}
`,
			changed: map[int]struct{}{9: {}},
			want: []advisoryCheckSnapshot{
				{Check: checkTestSkipAdded, File: "pkg/svc/svc_test.go", Symbol: "TestSkip"},
			},
		},
		// Replacing cmp.Diff with a nil check (if got == nil) still triggers test_assertion_weakened.
		{
			name: "cmp.Diff replaced by nil check triggers test_assertion_weakened",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestNilBypass(t *testing.T) {
	if d := cmp.Diff("ok", run()); d != "" { t.Errorf("mismatch: %s", d) }
}
`,
			targetSrc: `package svc
import "testing"
func TestNilBypass(t *testing.T) {
	if got := run(); got == nil { t.Fatal("nil") }
}
`,
			want: []advisoryCheckSnapshot{{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestNilBypass"}},
		},
		// Replacing cmp.Diff with renamed error nil checks (gotErr != nil, res.Err != nil) still triggers test_assertion_weakened.
		{
			name: "cmp.Diff replaced by renamed err nil check triggers test_assertion_weakened",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestRenamedErr(t *testing.T) {
	res, gotErr := parse()
	if gotErr != nil || res.Err != nil { t.Fatal(gotErr) }
	if d := cmp.Diff("ok", res.Val); d != "" { t.Errorf("mismatch: %s", d) }
}
`,
			targetSrc: `package svc
import "testing"
func TestRenamedErr(t *testing.T) {
	res, gotErr := parse()
	if gotErr != nil { t.Fatal(gotErr) }
	if res.Err != nil { t.Fatal(res.Err) }
}
`,
			want: []advisoryCheckSnapshot{{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestRenamedErr"}},
		},
		// Keeping an existing scalar check unchanged while dropping cmp.Diff triggers test_assertion_weakened.
		{
			name: "unchanged scalar check with dropped cmp.Diff triggers",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestKeepScalarDropCmp(t *testing.T) {
	got := parse()
	if got.Code != 200 { t.Errorf("code=%d", got.Code) }
	if d := cmp.Diff("ok", got.Body); d != "" { t.Errorf("mismatch: %s", d) }
}
`,
			targetSrc: `package svc
import "testing"
func TestKeepScalarDropCmp(t *testing.T) {
	got := parse()
	if got.Code != 200 { t.Errorf("code=%d", got.Code) }
}
`,
			want: []advisoryCheckSnapshot{{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestKeepScalarDropCmp"}},
		},
		// Partially dropping cmp.Diff assertions from 2 to 1 in a non-table test triggers test_assertion_weakened.
		{
			name: "partial cmp.Diff drop in non-table test triggers",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestPartialCmp(t *testing.T) {
	if d := cmp.Diff(1, a()); d != "" { t.Errorf("a: %s", d) }
	if d := cmp.Diff(2, b()); d != "" { t.Errorf("b: %s", d) }
}
`,
			targetSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestPartialCmp(t *testing.T) {
	if d := cmp.Diff(1, a()); d != "" { t.Errorf("a: %s", d) }
}
`,
			want: []advisoryCheckSnapshot{{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestPartialCmp"}},
		},
		// Partially dropping cmp.Diff assertions from 2 to 1 in an existing table test triggers test_assertion_weakened.
		{
			name: "partial cmp.Diff drop in existing table test triggers",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestPartialTableCmp(t *testing.T) {
	cases := []struct{ name string }{{name: "a"}, {name: "b"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if d := cmp.Diff("x", a()); d != "" { t.Errorf("a: %s", d) }
			if d := cmp.Diff("y", b()); d != "" { t.Errorf("b: %s", d) }
		})
	}
}
`,
			targetSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestPartialTableCmp(t *testing.T) {
	cases := []struct{ name string }{{name: "a"}, {name: "b"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if d := cmp.Diff("x", a()); d != "" { t.Errorf("a: %s", d) }
		})
	}
}
`,
			want: []advisoryCheckSnapshot{{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestPartialTableCmp"}},
		},
		// Consolidating cmp.Diff subtests into a table keeping only err != nil triggers test_assertion_weakened.
		{
			name: "table consolidation dropping cmp.Diff to err only triggers",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestTableErrOnly(t *testing.T) {
	t.Run("a", func(t *testing.T) { if d := cmp.Diff("a", run("a")); d != "" { t.Errorf("m: %s", d) } })
	t.Run("b", func(t *testing.T) { if d := cmp.Diff("b", run("b")); d != "" { t.Errorf("m: %s", d) } })
}
`,
			targetSrc: `package svc
import "testing"
func TestTableErrOnly(t *testing.T) {
	cases := []struct{ name string }{{name: "a"}, {name: "b"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := run(tc.name); err != nil { t.Fatal(err) }
		})
	}
}
`,
			want: []advisoryCheckSnapshot{{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestTableErrOnly"}},
		},
		// Dropping a non-cmp assertion in a non-table test without helper extraction triggers test_assertion_weakened.
		{
			name: "dropping non-cmp assertion in non-table test triggers",
			baseSrc: `package svc
import "testing"
func TestDropScalar(t *testing.T) {
	if a() != 1 { t.Errorf("a") }
	if b() != 2 { t.Errorf("b") }
}
`,
			targetSrc: `package svc
import "testing"
func TestDropScalar(t *testing.T) {
	if a() != 1 { t.Errorf("a") }
}
`,
			want: []advisoryCheckSnapshot{{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestDropScalar"}},
		},
		// Converting assertions into a 1-element struct slice does not trigger table consolidation suppression.
		{
			name: "single-element slice does not suppress test_assertion_weakened",
			baseSrc: `package svc
import "testing"
func TestSingleElemSlice(t *testing.T) {
	if a(1) != 1 { t.Errorf("a") }
	if a(2) != 2 { t.Errorf("b") }
}
`,
			targetSrc: `package svc
import "testing"
func TestSingleElemSlice(t *testing.T) {
	cases := []struct{ in int }{{in: 1}}
	for _, tc := range cases {
		if a(tc.in) != tc.in { t.Errorf("a") }
	}
}
`,
			want: []advisoryCheckSnapshot{{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestSingleElemSlice"}},
		},
		// Replacing cmp.Diff with a scalar check whose body calls a non-assert function triggers test_assertion_weakened.
		{
			name: "scalar check without testing assert triggers",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestNonAssertBody(t *testing.T) {
	if d := cmp.Diff("ok", run()); d != "" { t.Errorf("mismatch: %s", d) }
}
`,
			targetSrc: `package svc
import "testing"
func TestNonAssertBody(t *testing.T) {
	if got := run(); got != "ok" { nonAssertCall() }
}
`,
			want: []advisoryCheckSnapshot{{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestNonAssertBody"}},
		},
		// Replacing cmp.Diff with a scalar == check against tc.want prevents false-positive test_assertion_weakened.
		{
			name: "cmp.Diff replaced by scalar == tc.want has zero false positives",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestScalarEqual(t *testing.T) {
	if d := cmp.Diff(tc.want, run()); d != "" { t.Errorf("mismatch: %s", d) }
}
`,
			targetSrc: `package svc
import "testing"
func TestScalarEqual(t *testing.T) {
	if got := run(); got == tc.want { t.Errorf("unexpected match %v", got) }
}
`,
			want: nil,
		},
		// Replacing cmp.Diff with a relational < check still triggers test_assertion_weakened.
		{
			name: "cmp.Diff replaced by relational < check triggers",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestRelationalCheck(t *testing.T) {
	if d := cmp.Diff(1, run()); d != "" { t.Errorf("mismatch: %s", d) }
}
`,
			targetSrc: `package svc
import "testing"
func TestRelationalCheck(t *testing.T) {
	if got := run(); got < 0 { t.Errorf("negative: %d", got) }
}
`,
			want: []advisoryCheckSnapshot{{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestRelationalCheck"}},
		},
		// Consolidating cmp.Diff subtests into a table with scalar != prevents false-positive test_assertion_weakened.
		{
			name: "table consolidation with scalar != has zero false positives",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestTableScalar(t *testing.T) {
	t.Run("a", func(t *testing.T) { if d := cmp.Diff("a", run("a")); d != "" { t.Errorf("m: %s", d) } })
	t.Run("b", func(t *testing.T) { if d := cmp.Diff("b", run("b")); d != "" { t.Errorf("m: %s", d) } })
}
`,
			targetSrc: `package svc
import "testing"
func TestTableScalar(t *testing.T) {
	cases := []struct{ name, want string }{{name: "a", want: "a"}, {name: "b", want: "b"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := run(tc.name); got != tc.want { t.Errorf("got %q, want %q", got, tc.want) }
		})
	}
}
`,
			want: nil,
		},
		// With ChangedLines set, only overlapping modified Test* functions are flagged for reduced cases/weakened assertions while untouched Test* functions are skipped.
		{
			name: "changed lines scope flags modified weakened test and skips untouched test",
			baseSrc: `package svc
import ("testing"; "github.com/google/go-cmp/cmp")
func TestUntouched(t *testing.T) {
	if a() != 1 { t.Errorf("a") }
	if b() != 2 { t.Errorf("b") }
}
func TestModified(t *testing.T) {
	cases := []struct{ name string }{{name: "a"}, {name: "b"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if d := cmp.Diff("ok", run()); d != "" { t.Errorf("m: %s", d) }
		})
	}
}
`,
			targetSrc: `package svc
import "testing"
func TestUntouched(t *testing.T) {
	if a() != 1 { t.Errorf("a") }
}
func TestModified(t *testing.T) {
	cases := []struct{ name string }{{name: "a"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { run() })
	}
}
`,
			changed: map[int]struct{}{7: {}, 8: {}},
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestModified"},
				{Check: checkTestCasesReduced, File: "pkg/svc/svc_test.go", Symbol: "TestModified"},
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
			var changedLines map[string]map[int]struct{}
			if tc.changed != nil {
				changedLines = map[string]map[int]struct{}{"pkg/svc/svc_test.go": tc.changed}
			}
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: []string{"pkg/svc/svc_test.go"},
				ChangedLines: changedLines,
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

func assertAdvisorySnapshots(t *testing.T, items []rule.QAItem, want []advisoryCheckSnapshot) {
	t.Helper()
	var got []advisoryCheckSnapshot
	for _, item := range items {
		got = append(got, advisoryCheckSnapshot{
			Check:  item.Check,
			File:   item.File,
			Symbol: item.Symbol,
		})
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("advisory items mismatch (-want +got):\n%s", diff)
	}
}
