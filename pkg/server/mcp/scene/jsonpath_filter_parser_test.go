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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseFilterExpression(t *testing.T) {
	allowCmp := cmp.AllowUnexported(jsonPathStep{}, filterOrExpr{}, filterAndExpr{}, filterAtomExpr{}, filterLiteral{})

	testCases := []struct {
		name string
		raw  string
		want filterExpr
	}{
		{
			name: "single-quoted relative key protects brackets, spaces, and equals from boundary split",
			raw:  `@['a] = b'] == 'x'`,
			want: filterAtomExpr{
				relSteps: []jsonPathStep{{kind: stepKindKeys, keys: []string{"a] = b"}, raw: `['a] = b']`}},
				op:       filterOpEq,
				rhs:      filterLiteral{kind: literalString, strVal: "x"},
			},
		},
		{
			name: "double-quoted relative key with escaped quote protects boundary chars",
			raw:  `@["a\" ] = b"] == 'x'`,
			want: filterAtomExpr{
				relSteps: []jsonPathStep{{kind: stepKindKeys, keys: []string{`a" ] = b`}, raw: `["a\" ] = b"]`}},
				op:       filterOpEq,
				rhs:      filterLiteral{kind: literalString, strVal: "x"},
			},
		},
		{
			name: "unquoted bracket selector keeps internal space inside bracket depth",
			raw:  `@.items[0, 1] == 2`,
			want: filterAtomExpr{
				relSteps: []jsonPathStep{
					{kind: stepKindKeys, keys: []string{"items"}, raw: ".items"},
					{kind: stepKindIndices, indices: []int{0, 1}, raw: "[0, 1]"},
				},
				op:  filterOpEq,
				rhs: filterLiteral{kind: literalNumber, numVal: 2},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFilterExpression(tc.raw)
			if err != nil {
				t.Fatalf("parseFilterExpression(%q) unexpected error = %v", tc.raw, err)
			}
			if diff := cmp.Diff(tc.want, got, allowCmp); diff != "" {
				t.Errorf("parseFilterExpression(%q) mismatch (-want +got):\n%s", tc.raw, diff)
			}
		})
	}
}

func TestParseFilterExpression_RelativePathErrors(t *testing.T) {
	testCases := []struct {
		name       string
		raw        string
		wantSubstr string
	}{
		{
			name:       "unclosed relative path escape stops safely at end of input",
			raw:        `@['a\`,
			wantSubstr: "invalid relative path '@['a\\'",
		},
		{
			name:       "unclosed quoted literal returns error",
			raw:        `@.a == 'unclosed`,
			wantSubstr: "unclosed quoted string",
		},
		{
			name:       "stray closing bracket does not keep bracket depth negative",
			raw:        `@.a] == 1`,
			wantSubstr: "invalid relative path '@.a]': unexpected character ']'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseFilterExpression(tc.raw)
			if err == nil {
				t.Fatalf("parseFilterExpression(%q) expected error, got nil", tc.raw)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("parseFilterExpression(%q) error = %q, want substring %q", tc.raw, err.Error(), tc.wantSubstr)
			}
		})
	}
}
