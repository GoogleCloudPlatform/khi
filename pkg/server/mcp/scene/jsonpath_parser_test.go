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

package scene

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInvalidJSONPathError_Error(t *testing.T) {
	testCases := []struct {
		name string
		err  *InvalidJSONPathError
		want string
	}{
		{
			name: "negative path index formats list error",
			err: &InvalidJSONPathError{
				PathIndex: -1,
				Reason:    "paths must contain at least one JSONPath selector",
			},
			want: "invalid JSONPath list: paths must contain at least one JSONPath selector",
		},
		{
			name: "zero path index formats selector error",
			err: &InvalidJSONPathError{
				PathIndex: 0,
				Path:      "$.spec[",
				Reason:    "unclosed '[' in JSONPath",
			},
			want: `invalid JSONPath 0 ("$.spec["): unclosed '[' in JSONPath`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseJSONPathSteps(t *testing.T) {
	intVal := func(v int) *int { return &v }
	allowCmp := cmp.AllowUnexported(jsonPathStep{}, filterOrExpr{}, filterAndExpr{}, filterAtomExpr{}, filterLiteral{})

	testCases := []struct {
		name            string
		raw             string
		allowEmptySteps bool
		want            []jsonPathStep
	}{
		{
			name:            "dollar root allowed when allowEmptySteps is true",
			raw:             "$",
			allowEmptySteps: true,
		},
		{
			name: "leading dot without dollar parses key steps",
			raw:  ".metadata.name",
			want: []jsonPathStep{
				{kind: stepKindKeys, keys: []string{"metadata"}, raw: ".metadata"},
				{kind: stepKindKeys, keys: []string{"name"}, raw: ".name"},
			},
		},
		{
			name: "wildcard bare root and dot wildcard and recursive wildcard",
			raw:  "*.*..*",
			want: []jsonPathStep{
				{kind: stepKindWildcard, raw: ".*"},
				{kind: stepKindWildcard, raw: ".*"},
				{kind: stepKindWildcard, recursive: true, raw: "..*"},
			},
		},
		{
			name: "recursive bracket wildcard and quoted key union with whitespace",
			raw:  "$..[*]['a' , \"b\"]",
			want: []jsonPathStep{
				{kind: stepKindWildcard, recursive: true, raw: "..[*]"},
				{kind: stepKindKeys, keys: []string{"a", "b"}, raw: `['a' , "b"]`},
			},
		},
		{
			name: "escaped bracket and quote inside quoted key selector",
			raw:  `$['a]b\'c']`,
			want: []jsonPathStep{
				{kind: stepKindKeys, keys: []string{"a]b'c"}, raw: `['a]b\'c']`},
			},
		},
		{
			name: "bare identifier with backslash before non-dot and trailing backslash",
			raw:  `$.a\b\`,
			want: []jsonPathStep{
				{kind: stepKindKeys, keys: []string{`a\b\`}, raw: `.a\b\`},
			},
		},
		{
			name: "array indices union and open slice bounds",
			raw:  "$.items[0, -1][:2][1:]",
			want: []jsonPathStep{
				{kind: stepKindKeys, keys: []string{"items"}, raw: ".items"},
				{kind: stepKindIndices, indices: []int{0, -1}, raw: "[0, -1]"},
				{kind: stepKindSlice, sliceTo: intVal(2), raw: "[:2]"},
				{kind: stepKindSlice, sliceFrom: intVal(1), raw: "[1:]"},
			},
		},
		{
			name: "complex filter expression with parens, or, and, relative brackets, and all comparison ops",
			raw:  `$.items[?((@['a.b[\'c\']'] == 'x' || @.n != null) && @.count <= 5 && @.count >= 1 && @.rank < 9 && @.flag == true && @.disabled == false && @.exists)]`,
			want: []jsonPathStep{
				{kind: stepKindKeys, keys: []string{"items"}, raw: ".items"},
				{
					kind: stepKindFilter,
					raw:  `[?((@['a.b[\'c\']'] == 'x' || @.n != null) && @.count <= 5 && @.count >= 1 && @.rank < 9 && @.flag == true && @.disabled == false && @.exists)]`,
					filter: filterAndExpr{
						left: filterAndExpr{
							left: filterAndExpr{
								left: filterAndExpr{
									left: filterAndExpr{
										left: filterAndExpr{
											left: filterOrExpr{
												left: filterAtomExpr{
													relSteps: []jsonPathStep{{kind: stepKindKeys, keys: []string{"a.b['c']"}, raw: `['a.b[\'c\']']`}},
													op:       filterOpEq,
													rhs:      filterLiteral{kind: literalString, strVal: "x"},
												},
												right: filterAtomExpr{
													relSteps: []jsonPathStep{{kind: stepKindKeys, keys: []string{"n"}, raw: ".n"}},
													op:       filterOpNe,
													rhs:      filterLiteral{kind: literalNull},
												},
											},
											right: filterAtomExpr{
												relSteps: []jsonPathStep{{kind: stepKindKeys, keys: []string{"count"}, raw: ".count"}},
												op:       filterOpLe,
												rhs:      filterLiteral{kind: literalNumber, numVal: 5},
											},
										},
										right: filterAtomExpr{
											relSteps: []jsonPathStep{{kind: stepKindKeys, keys: []string{"count"}, raw: ".count"}},
											op:       filterOpGe,
											rhs:      filterLiteral{kind: literalNumber, numVal: 1},
										},
									},
									right: filterAtomExpr{
										relSteps: []jsonPathStep{{kind: stepKindKeys, keys: []string{"rank"}, raw: ".rank"}},
										op:       filterOpLt,
										rhs:      filterLiteral{kind: literalNumber, numVal: 9},
									},
								},
								right: filterAtomExpr{
									relSteps: []jsonPathStep{{kind: stepKindKeys, keys: []string{"flag"}, raw: ".flag"}},
									op:       filterOpEq,
									rhs:      filterLiteral{kind: literalBool, boolVal: true},
								},
							},
							right: filterAtomExpr{
								relSteps: []jsonPathStep{{kind: stepKindKeys, keys: []string{"disabled"}, raw: ".disabled"}},
								op:       filterOpEq,
								rhs:      filterLiteral{kind: literalBool, boolVal: false},
							},
						},
						right: filterAtomExpr{
							relSteps: []jsonPathStep{{kind: stepKindKeys, keys: []string{"exists"}, raw: ".exists"}},
							op:       filterOpExists,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseJSONPathSteps(tc.raw, tc.allowEmptySteps)
			if err != nil {
				t.Fatalf("parseJSONPathSteps(%q) unexpected error = %v", tc.raw, err)
			}
			if diff := cmp.Diff(tc.want, got, allowCmp); diff != "" {
				t.Errorf("parseJSONPathSteps(%q) mismatch (-want +got):\n%s", tc.raw, diff)
			}
		})
	}
}

func TestCompileJSONPaths_Errors(t *testing.T) {
	testCases := []struct {
		name       string
		paths      []string
		want       *InvalidJSONPathError
		wantSubstr string
	}{
		{
			name:  "nil paths slice",
			paths: nil,
			want:  &InvalidJSONPathError{PathIndex: -1, Reason: "paths must contain at least one JSONPath selector"},
		},
		{
			name:  "empty paths slice",
			paths: []string{},
			want:  &InvalidJSONPathError{PathIndex: -1, Reason: "paths must contain at least one JSONPath selector"},
		},
		{
			name:  "empty path string",
			paths: []string{"   "},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "   ", Reason: "JSONPath must not be empty"},
		},
		{
			name:  "root dollar only",
			paths: []string{"$"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$", Reason: "JSONPath must select a field or element beneath root '$' (e.g. '$.spec' or '$.status')"},
		},
		{
			name:  "trailing dot",
			paths: []string{"$.spec."},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.spec.", Reason: "trailing '.' in JSONPath"},
		},
		{
			name:  "trailing recursive descent",
			paths: []string{"$.spec.."},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.spec..", Reason: "trailing '..' in JSONPath"},
		},
		{
			name:  "dot before bracket",
			paths: []string{"$.spec.[0]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.spec.[0]", Reason: "unexpected '[' after '.'; use '[...]' without a preceding dot"},
		},
		{
			name:  "unclosed bracket",
			paths: []string{"$.spec[0"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.spec[0", Reason: "unclosed '[' in JSONPath"},
		},
		{
			name:  "invalid second path reports index 1",
			paths: []string{"$.spec", "$.status]"},
			want:  &InvalidJSONPathError{PathIndex: 1, Path: "$.status]", Reason: "unexpected character ']' in field name"},
		},
		{
			name:  "unexpected closing bracket after bracket step",
			paths: []string{"$.spec[0]]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.spec[0]]", Reason: "unexpected character ']' at offset 9"},
		},
		{
			name:  "space in bare field name",
			paths: []string{"$.a b"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.a b", Reason: "unexpected character ' ' in field name"},
		},
		{
			name:  "tab in bare field name",
			paths: []string{"$.a\tb"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.a\tb", Reason: "unexpected character '\\t' in field name"},
		},
		{
			name:  "triple dot empty bare field name",
			paths: []string{"$...a"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$...a", Reason: "empty field name in JSONPath"},
		},
		{
			name:  "empty bracket selector",
			paths: []string{"$.spec[]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.spec[]", Reason: "empty bracket selector '[]'"},
		},
		{
			name:  "invalid recursive bracket selector",
			paths: []string{"$..[]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$..[]", Reason: "empty bracket selector '[]'"},
		},
		{
			name:  "missing comma in quoted key union",
			paths: []string{"$.spec['a' 'b']"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.spec['a' 'b']", Reason: "expected ',' or ']' after quoted key in bracket selector"},
		},
		{
			name:  "trailing comma in quoted key union",
			paths: []string{"$.spec['a', ]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.spec['a', ]", Reason: "trailing comma in bracket key selector"},
		},
		{
			name:  "non-quoted key after comma in quoted key union",
			paths: []string{"$.spec['a', 1]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.spec['a', 1]", Reason: "expected quoted string"},
		},
		{
			name:  "slice with three parts",
			paths: []string{"$.items[1:2:3]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[1:2:3]", Reason: `slice selector "1:2:3" supports only [start:end]`},
		},
		{
			name:  "invalid slice start",
			paths: []string{"$.items[a:2]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[a:2]", Reason: `invalid slice start "a"`},
		},
		{
			name:  "invalid slice end",
			paths: []string{"$.items[1:b]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[1:b]", Reason: `invalid slice end "b"`},
		},
		{
			name:  "unquoted bare array index",
			paths: []string{"$.items[foo]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[foo]", Reason: `invalid array selector "foo" (quote string keys as ['foo'])`},
		},
		{
			name:  "empty filter expression",
			paths: []string{"$.items[?]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?]", Reason: "empty filter expression after '?'"},
		},
		{
			name:  "trailing token in filter expression",
			paths: []string{"$.items[?@.a == 1 extra]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?@.a == 1 extra]", Reason: `unexpected trailing token "extra" in filter expression`},
		},
		{
			name:  "unexpected end in filter expression",
			paths: []string{"$.items[?(@.a == 1 && )]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?(@.a == 1 && )]", Reason: "filter predicate must start with '@' (e.g. [?(@.type=='Ready')])"},
		},
		{
			name:  "unexpected end at EOF after binary operator",
			paths: []string{"$.items[?@.a && ]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?@.a && ]", Reason: "unexpected end of filter expression"},
		},
		{
			name:  "unexpected end after or operator",
			paths: []string{"$.items[?@.a || ]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?@.a || ]", Reason: "unexpected end of filter expression"},
		},
		{
			name:  "unclosed paren in filter expression",
			paths: []string{"$.items[?(@.a == 1]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?(@.a == 1]", Reason: "unclosed '(' in filter expression"},
		},
		{
			name:  "invalid sub-expression inside parens",
			paths: []string{"$.items[?(a == 1)]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?(a == 1)]", Reason: "filter predicate must start with '@' (e.g. [?(@.type=='Ready')])"},
		},
		{
			name:       "invalid relative path inside filter",
			paths:      []string{"$.items[?(@.a. == 1)]"},
			wantSubstr: "invalid relative path '@.a.'",
		},
		{
			name:  "single equals in filter",
			paths: []string{"$.items[?(@.type='Ready')]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?(@.type='Ready')]", Reason: "use '==' instead of '=' in filter expression"},
		},
		{
			name:  "missing comparison literal before paren",
			paths: []string{"$.items[?(@.type == )]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?(@.type == )]", Reason: "missing comparison value in filter expression"},
		},
		{
			name:  "missing comparison literal at end of filter",
			paths: []string{"$.items[?@.type == ]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?@.type == ]", Reason: "missing comparison value in filter expression"},
		},
		{
			name:  "unquoted string literal in filter",
			paths: []string{"$.items[?(@.type==Ready)]"},
			want:  &InvalidJSONPathError{PathIndex: 0, Path: "$.items[?(@.type==Ready)]", Reason: `invalid filter literal "Ready" (quote strings with single or double quotes)`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := compileJSONPaths(tc.paths)
			var gotErr *InvalidJSONPathError
			if !errors.As(err, &gotErr) {
				t.Fatalf("compileJSONPaths(%v) err = %T (%v), want *InvalidJSONPathError", tc.paths, err, err)
			}
			if tc.want != nil {
				if diff := cmp.Diff(tc.want, gotErr); diff != "" {
					t.Errorf("InvalidJSONPathError mismatch (-want +got):\n%s", diff)
				}
				return
			}
			if !strings.Contains(gotErr.Reason, tc.wantSubstr) {
				t.Errorf("gotErr.Reason = %q, want substring %q", gotErr.Reason, tc.wantSubstr)
			}
		})
	}
}

func TestParseQuotedString_AndScanRelativeEscapeBounds(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		start   int
		wantVal string
		wantPos int
		wantErr string
	}{
		{
			name:    "parses newline, carriage return, tab, and escaped quote",
			input:   `'a\nb\rc\td\'e'rest`,
			start:   0,
			wantVal: "a\nb\rc\td'e",
			wantPos: 15,
		},
		{
			name:    "start at end of string returns error",
			input:   `'abc'`,
			start:   5,
			wantErr: "expected quoted string",
		},
		{
			name:    "non-quote start returns error",
			input:   `abc`,
			start:   0,
			wantErr: "expected quoted string",
		},
		{
			name:    "trailing backslash inside quote returns unclosed error",
			input:   `'abc\`,
			start:   0,
			wantErr: "unclosed quoted string",
		},
		{
			name:    "unclosed quoted string returns error",
			input:   `"abc`,
			start:   0,
			wantErr: "unclosed quoted string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotVal, gotPos, err := parseQuotedString(tc.input, tc.start)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("parseQuotedString() err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseQuotedString() unexpected error = %v", err)
			}
			if gotVal != tc.wantVal || gotPos != tc.wantPos {
				t.Errorf("parseQuotedString() = (%q, %d), want (%q, %d)", gotVal, gotPos, tc.wantVal, tc.wantPos)
			}
		})
	}

	t.Run("filterParser handles malformed unclosed escape and unclosed quoted literal safely", func(t *testing.T) {
		if _, err := parseFilterExpression(`@['a\`); err == nil {
			t.Fatal("parseFilterExpression(@['a\\) expected error, got nil")
		}
		if _, err := parseFilterExpression(`@.a == 'unclosed`); err == nil {
			t.Fatal("parseFilterExpression(@.a == 'unclosed) expected error, got nil")
		}
	})
}
