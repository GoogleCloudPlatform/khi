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
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestCountTableElementsLocalStructIsolation verifies that helper-local struct declarations do not leak into idx.localStructs across sibling Test* functions.
func TestCountTableElementsLocalStructIsolation(t *testing.T) {
	testCases := []struct {
		name           string
		src            string
		wantLeakedKey  bool
		wantTableElems map[string]int
	}{
		// Local struct declared inside a table helper does not make an external struct type count as local in a sibling test.
		{
			name: "helper local struct does not leak to sibling test",
			src: `package svc

import "testing"

type hCase struct {
	name string
}

func runH(t *testing.T, cases []hCase) {
	t.Helper()
	type leaked struct {
		name string
	}
	_ = leaked{name: "helper-only"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "" {
				t.Errorf("empty")
			}
		})
	}
}

func TestFirst(t *testing.T) {
	cases := []hCase{{name: "x"}}
	runH(t, cases)
}

func TestSecond(t *testing.T) {
	items := []leaked{{name: "a"}, {name: "b"}}
	for _, tc := range items {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "" {
				t.Errorf("empty")
			}
		})
	}
}
`,
			wantLeakedKey: false,
			wantTableElems: map[string]int{
				"TestFirst":  1,
				"TestSecond": 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "svc_test.go", []byte(tc.src), 0)
			if err != nil {
				t.Fatalf("ParseFile() unexpected error: %v", err)
			}
			idx := newTestFileIndex(file)

			// Act.
			summaries := collectGoTestFuncSummaries(fset, file)
			gotElems := make(map[string]int, len(summaries))
			gotDirect := make(map[string]int, len(summaries))
			for _, summary := range summaries {
				gotElems[summary.name] = summary.tableElems
				gotDirect[summary.name] = countTableElements(idx, summary.decl)
			}

			// Assert.
			if idx.localStructs["leaked"] != tc.wantLeakedKey {
				t.Errorf("idx.localStructs[\"leaked\"] = %v, want %v", idx.localStructs["leaked"], tc.wantLeakedKey)
			}
			if diff := cmp.Diff(tc.wantTableElems, gotElems); diff != "" {
				t.Errorf("summary tableElems mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantTableElems, gotDirect); diff != "" {
				t.Errorf("direct countTableElements mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGoTestTablesNonTestingRunLoop verifies that non-testing 2-arg .Run calls in a loop are not counted as table tests while real testing.TB.Run loops still are.
func TestGoTestTablesNonTestingRunLoop(t *testing.T) {
	testCases := []struct {
		name         string
		baseSrc      string
		targetSrc    string
		want         []advisoryCheckSnapshot
		wantEvidence map[string]any
	}{
		// Dropping standalone subtests while adding a loop over svc.Run(ctx, it) still flags test_cases_reduced.
		{
			name: "non-testing svc.Run loop does not count as table",
			baseSrc: `package svc

import "testing"

func TestServiceRunLoop(t *testing.T) {
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

func TestServiceRunLoop(t *testing.T) {
	items := []item{{name: "a"}, {name: "b"}}
	for _, it := range items {
		svc.Run(ctx, it)
	}
	t.Run("a", func(t *testing.T) {
		if run("a") != "a" {
			t.Errorf("bad a")
		}
		if len(items) != 2 {
			t.Errorf("bad len")
		}
	})
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestCasesReduced, File: "pkg/svc/svc_test.go", Symbol: "TestServiceRunLoop"},
			},
			wantEvidence: map[string]any{"before_count": 2, "after_count": 1},
		},
		// Consolidating standalone subtests into a testing.TB.Run helper loop has zero false positives.
		{
			name: "testing.TB.Run helper loop counts table rows accurately",
			baseSrc: `package svc

import "testing"

func TestTBHelperLoop(t *testing.T) {
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

type tbCase struct {
	name string
}

func runTBHelper(tb testing.TB, cases []tbCase) {
	tb.Helper()
	for _, tc := range cases {
		tb.Run(tc.name, func(t *testing.T) {
			if run(tc.name) != tc.name {
				t.Errorf("bad %s", tc.name)
			}
		})
	}
}

func TestTBHelperLoop(t *testing.T) {
	cases := []tbCase{{name: "a"}, {name: "b"}}
	runTBHelper(t, cases)
}
`,
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
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

			// Act.
			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}

			// Assert.
			assertAdvisorySnapshots(t, items, tc.want)
			if tc.wantEvidence != nil && len(items) > 0 {
				if diff := cmp.Diff(tc.wantEvidence, items[0].Evidence); diff != "" {
					t.Errorf("evidence mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
