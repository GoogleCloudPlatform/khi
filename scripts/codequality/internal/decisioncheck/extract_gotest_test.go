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

package decisioncheck

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractChangedGoTestFuncs(t *testing.T) {
	testSrc := `package sample

import "testing"

type testHelper struct{}

func (h *testHelper) TestMethod(t *testing.T) {}

func TestMain(m *testing.M) {}

func Test(t *testing.T) {}

func Testhelper(t *testing.T) {}

func helper() {}

// TestStandaloneCallback verifies that progress events are recorded in order.
func TestStandaloneCallback(t *testing.T) {}
`
	nonTestSrc := `package sample

func TestIgnoredInNonTestFile() {}
`
	files := []BundledFile{
		{RelPath: "pkg/sample/sample.go", Content: []byte(nonTestSrc)},
		{RelPath: "pkg/sample/sample_test.go", Content: []byte(testSrc)},
	}

	testCases := []struct {
		name    string
		changed map[string]map[int]struct{}
		want    []ExtractedItem
	}{
		// Extracts Test* functions (including bare Test) while skipping TestMain, Testhelper, methods, helpers, and non-test Go files.
		{
			name:    "full mode filters non-unit-test declarations",
			changed: nil,
			want: []ExtractedItem{
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      11,
					StartLine: 11,
					EndLine:   11,
					Name:      "Test",
					Kind:      "test func",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      18,
					StartLine: 17,
					EndLine:   18,
					Name:      "TestStandaloneCallback",
					Kind:      "test func",
					Doc:       "TestStandaloneCallback verifies that progress events are recorded in order.",
				},
			},
		},
		// Includes a test function when changedLines touches only its preceding GoDoc comment line and excludes untouched tests.
		{
			name: "differential mode matches GoDoc span and skips untouched tests",
			changed: map[string]map[int]struct{}{
				"pkg/sample/sample_test.go": {17: {}},
			},
			want: []ExtractedItem{
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      18,
					StartLine: 17,
					EndLine:   18,
					Name:      "TestStandaloneCallback",
					Kind:      "test func",
					Doc:       "TestStandaloneCallback verifies that progress events are recorded in order.",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractChangedGoTestFuncs("pkg/sample/sample_test.go", files, tc.changed)

			if err != nil {
				t.Fatalf("ExtractChangedGoTestFuncs() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractChangedGoTestFuncs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestExtractChangedGoTestFuncs_InvalidSyntax verifies that malformed Go test source returns a parse error and no items.
func TestExtractChangedGoTestFuncs_InvalidSyntax(t *testing.T) {
	files := []BundledFile{
		{
			RelPath: "pkg/sample/broken_test.go",
			Content: []byte(`package broken

func TestBroken( {
`),
		},
	}

	got, err := ExtractChangedGoTestFuncs("pkg/sample/broken_test.go", files, nil)

	if err == nil {
		t.Fatal("ExtractChangedGoTestFuncs() error = nil, want non-nil parse error")
	}
	if got != nil {
		t.Errorf("ExtractChangedGoTestFuncs() items = %v, want nil on error", got)
	}
}

func TestExtractChangedGoTestCases(t *testing.T) {
	tableVarsSrc := `package sample

import "testing"

const pkgConst = 1

var pkgCases = []struct {
	name string
}{
	// Uses package-level table literal.
	{name: "pkg one"},
}

func pair() (int, int) { return 1, 2 }
func getModes() []string { return []string{"fast"} }
func checkItem(string) {}

func TestTables(t *testing.T) {
	type localCfg struct {
		name string
		flag bool
	}
	var uninit []string
	var varCases = []localCfg{
		// Uses local var table literal.
		{name: "var one", flag: true},
	}
	plain := 1
	_ = plain
	a, b := pair()
	_, _ = a, b
	holder := struct{ items []int }{}
	holder.items = []int{1}
	_ = uninit
	ignoredItems := []string{"ignored"}
	for _, item := range ignoredItems {
		checkItem(item)
		t.Log(item)
	}
	modeCases := []struct{ name string }{
		// Extracts inner subtest table wrapped inside an unresolved outer range loop.
		{name: "mode one"},
	}
	for _, mode := range getModes() {
		for _, tc := range modeCases {
			_ = mode
			t.Run(tc.name, func(t *testing.T) {})
		}
	}
	for _, tc := range varCases {
		t.Run(tc.name, func(t *testing.T) {})
	}
	for _, tc := range varCases {
		t.Run(tc.name, func(t *testing.T) {})
	}
	innerCases := []struct{ name string }{
		// Extracts nested subtest table inside outer subtest loop.
		{name: "inner one"},
	}
	for _, tc := range pkgCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, inner := range innerCases {
				t.Run(inner.name, func(t *testing.T) {})
			}
		})
	}
	shadowed := []struct{ name string }{
		// Keeps slice table when inner scope shadows the variable with a struct literal.
		{name: "kept slice"},
	}
	for _, tc := range shadowed {
		t.Run(tc.name, func(t *testing.T) {
			shadowed := localCfg{flag: true}
			_ = shadowed
		})
	}
}
`

	labelsCommentsSrc := `package sample

import "testing"

func TestLabelsAndComments(t *testing.T) {
	for _, tc := range []struct {
		name  string
		desc  string
		input int
	}{
		{name: "first"}, // trailing note on first
		{name: "second"},
		{ // Inside opening brace comment.
			name: "inside opening",
		},
		{
			// Inside next line comment.
			name: "inside next",
		},
		{
			name: "field inline", // inline on firstInner
		},
		{name: "", desc: "desc fallback"},
		{input: 42},
		{},
	} {
		t.Run(tc.name, func(t *testing.T) {})
	}
	mapCases := map[string]struct {
		name string
		val  int
	}{
		// Map entry comment above key.
		"map key label": {val: 1},
		"": { // Inside map value brace.
			name: "empty key fallback",
		},
	}
	for k := range mapCases {
		t.Run(k, func(t *testing.T) {})
	}
	posCases := []struct {
		label string
		code  int
	}{
		{"positional label", 1},
	}
	for _, tc := range posCases {
		t.Run(tc.label, func(t *testing.T) {})
	}
	numCases := []struct {
		code int
	}{
		{42},
	}
	for _, tc := range numCases {
		t.Run("num", func(t *testing.T) { _ = tc })
	}
	nestedMaps := []map[string]int{
		{"alpha": 1},
	}
	for _, m := range nestedMaps {
		t.Run("map", func(t *testing.T) { _ = m })
	}
	rawStrs := []string{"raw-entry"}
	for _, s := range rawStrs {
		t.Run(s, func(t *testing.T) {})
	}
}

// TestEmptyTable verifies that an empty table falls back to a standalone test.
func TestEmptyTable(t *testing.T) {
	empty := []struct{ name string }{}
	for _, tc := range empty {
		t.Run(tc.name, func(t *testing.T) {})
	}
}
`

	diffSrc := `package sample

import "testing"

func TestLookup(t *testing.T) {
	testCases := []struct {
		name string
	}{
		// Matches zero.
		{name: "zero"},
		// Matches one.
		{name: "one"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {})
	}
}

// TestStandalone verifies standalone filtering.
func TestStandalone(t *testing.T) {}
`

	testCases := []struct {
		name    string
		src     string
		changed map[string]map[int]struct{}
		want    []ExtractedItem
	}{
		// Resolves package-level and local var tables, skips non-subtest loops and shadowed struct literals, and deduplicates nested or repeated loops.
		{
			name:    "package local var nested and shadowed tables",
			src:     tableVarsSrc,
			changed: nil,
			want: []ExtractedItem{
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      42,
					StartLine: 41,
					EndLine:   42,
					Name:      `TestTables / "mode one"`,
					Kind:      "test case",
					Doc:       "Extracts inner subtest table wrapped inside an unresolved outer range loop.",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      26,
					StartLine: 25,
					EndLine:   26,
					Name:      `TestTables / "var one"`,
					Kind:      "test case",
					Doc:       "Uses local var table literal.",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      11,
					StartLine: 10,
					EndLine:   11,
					Name:      `TestTables / "pkg one"`,
					Kind:      "test case",
					Doc:       "Uses package-level table literal.",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      58,
					StartLine: 57,
					EndLine:   58,
					Name:      `TestTables / "inner one"`,
					Kind:      "test case",
					Doc:       "Extracts nested subtest table inside outer subtest loop.",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      69,
					StartLine: 68,
					EndLine:   69,
					Name:      `TestTables / "kept slice"`,
					Kind:      "test case",
					Doc:       "Keeps slice table when inner scope shadows the variable with a struct literal.",
				},
			},
		},
		// Extracts comments inside braces, avoids stealing trailing comments, resolves map/positional/fallback labels, and treats empty tables as standalone tests.
		{
			name:    "comment placement map positional fallback labels and empty table",
			src:     labelsCommentsSrc,
			changed: nil,
			want: []ExtractedItem{
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      11,
					StartLine: 11,
					EndLine:   11,
					Name:      `TestLabelsAndComments / "first"`,
					Kind:      "test case",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      12,
					StartLine: 12,
					EndLine:   12,
					Name:      `TestLabelsAndComments / "second"`,
					Kind:      "test case",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      13,
					StartLine: 13,
					EndLine:   15,
					Name:      `TestLabelsAndComments / "inside opening"`,
					Kind:      "test case",
					Doc:       "Inside opening brace comment.",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      16,
					StartLine: 16,
					EndLine:   19,
					Name:      `TestLabelsAndComments / "inside next"`,
					Kind:      "test case",
					Doc:       "Inside next line comment.",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      20,
					StartLine: 20,
					EndLine:   22,
					Name:      `TestLabelsAndComments / "field inline"`,
					Kind:      "test case",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      23,
					StartLine: 23,
					EndLine:   23,
					Name:      `TestLabelsAndComments / "desc fallback"`,
					Kind:      "test case",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      24,
					StartLine: 24,
					EndLine:   24,
					Name:      `TestLabelsAndComments / "case #7"`,
					Kind:      "test case",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      25,
					StartLine: 25,
					EndLine:   25,
					Name:      `TestLabelsAndComments / "case #8"`,
					Kind:      "test case",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      34,
					StartLine: 33,
					EndLine:   34,
					Name:      `TestLabelsAndComments / "map key label"`,
					Kind:      "test case",
					Doc:       "Map entry comment above key.",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      35,
					StartLine: 35,
					EndLine:   37,
					Name:      `TestLabelsAndComments / "empty key fallback"`,
					Kind:      "test case",
					Doc:       "Inside map value brace.",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      46,
					StartLine: 46,
					EndLine:   46,
					Name:      `TestLabelsAndComments / "positional label"`,
					Kind:      "test case",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      54,
					StartLine: 54,
					EndLine:   54,
					Name:      `TestLabelsAndComments / "case #1"`,
					Kind:      "test case",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      60,
					StartLine: 60,
					EndLine:   60,
					Name:      `TestLabelsAndComments / "case #1"`,
					Kind:      "test case",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      65,
					StartLine: 65,
					EndLine:   65,
					Name:      `TestLabelsAndComments / "case #1"`,
					Kind:      "test case",
				},
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      72,
					StartLine: 71,
					EndLine:   77,
					Name:      "TestEmptyTable",
					Kind:      "standalone test",
					Doc:       "TestEmptyTable verifies that an empty table falls back to a standalone test.",
				},
			},
		},
		// Extracts only the table case whose comment span overlaps changedLines while skipping untouched cases and standalone tests.
		{
			name: "differential mode extracts only touched table case",
			src:  diffSrc,
			changed: map[string]map[int]struct{}{
				"pkg/sample/sample_test.go": {11: {}},
			},
			want: []ExtractedItem{
				{
					Target:    "pkg/sample/sample_test.go",
					File:      "pkg/sample/sample_test.go",
					Line:      12,
					StartLine: 11,
					EndLine:   12,
					Name:      `TestLookup / "one"`,
					Kind:      "test case",
					Doc:       "Matches one.",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			files := []BundledFile{{RelPath: "pkg/sample/sample_test.go", Content: []byte(tc.src)}}

			got, err := ExtractChangedGoTestCases("pkg/sample/sample_test.go", files, tc.changed)

			if err != nil {
				t.Fatalf("ExtractChangedGoTestCases() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractChangedGoTestCases() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
