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

import "testing"

func TestFilterExpr_Eval(t *testing.T) {
	const docYAML = `items:
- name: alpha
  count: 10
  active: true
  emptyVal: null
  meta:
    env: prod
- name: beta
  count: 5
  active: false
  badBool: maybe
  badNum: abc
  emptyVal: non-null
`
	doc, err := parseIndexedYAMLDocument(docYAML)
	if err != nil {
		t.Fatalf("parseIndexedYAMLDocument() error = %v", err)
	}
	alpha := doc.root.byKey["items"].seqItems[0]
	beta := doc.root.byKey["items"].seqItems[1]

	testCases := []struct {
		name      string
		filterRaw string
		candidate *indexedYAMLNode
		wantMatch bool
		wantHLKey string
	}{
		{
			name:      "or matches left branch",
			filterRaw: `@.count == 10 || @.name == 'missing'`,
			candidate: alpha,
			wantMatch: true,
			wantHLKey: "count",
		},
		{
			name:      "or falls through to right branch",
			filterRaw: `@.count == 99 || @.name == 'alpha'`,
			candidate: alpha,
			wantMatch: true,
			wantHLKey: "name",
		},
		{
			name:      "or returns false when both fail",
			filterRaw: `@.count == 99 || @.name == 'missing'`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "and returns false when right side fails",
			filterRaw: `@.count == 10 && @.name == 'beta'`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "exists matches present non-null scalar and highlights it",
			filterRaw: `@.count`,
			candidate: alpha,
			wantMatch: true,
			wantHLKey: "count",
		},
		{
			name:      "exists rejects null and missing fields",
			filterRaw: `@.emptyVal`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "exists rejects missing field",
			filterRaw: `@.noSuchField`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "null literal eq on null node is true",
			filterRaw: `@.emptyVal == null`,
			candidate: alpha,
			wantMatch: true,
		},
		{
			name:      "null literal eq on non-null node is false",
			filterRaw: `@.emptyVal == null`,
			candidate: beta,
			wantMatch: false,
		},
		{
			name:      "null literal ne on non-null node is true",
			filterRaw: `@.emptyVal != null`,
			candidate: beta,
			wantMatch: true,
		},
		{
			name:      "null literal ne on null node is false",
			filterRaw: `@.emptyVal != null`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "null literal ordered comparison returns false",
			filterRaw: `@.emptyVal < null`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "non-null literal ne on null node is true even when scalar text is null",
			filterRaw: `@.emptyVal != 'null'`,
			candidate: alpha,
			wantMatch: true,
		},
		{
			name:      "non-null literal eq on null node is false even when scalar text is null",
			filterRaw: `@.emptyVal == 'null'`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "non-scalar node ne non-null literal is true",
			filterRaw: `@.meta != 'x'`,
			candidate: alpha,
			wantMatch: true,
		},
		{
			name:      "non-scalar node ne null literal is false",
			filterRaw: `@.meta != null`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "non-scalar node eq literal is false",
			filterRaw: `@.meta == 'x'`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "bool eq matches",
			filterRaw: `@.active == true`,
			candidate: alpha,
			wantMatch: true,
		},
		{
			name:      "bool ne matches",
			filterRaw: `@.active != true`,
			candidate: beta,
			wantMatch: true,
		},
		{
			name:      "bool ordered comparison returns false",
			filterRaw: `@.active < true`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "invalid bool scalar returns false",
			filterRaw: `@.badBool == true`,
			candidate: beta,
			wantMatch: false,
		},
		{
			name:      "number eq, ne, lt, le, gt, ge boundaries",
			filterRaw: `@.count == 10 && @.count != 5 && @.count < 11 && @.count <= 10 && @.count > 9 && @.count >= 10`,
			candidate: alpha,
			wantMatch: true,
		},
		{
			name:      "number gt rejects equal boundary",
			filterRaw: `@.count > 10`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "number lt rejects equal boundary",
			filterRaw: `@.count < 10`,
			candidate: alpha,
			wantMatch: false,
		},
		{
			name:      "invalid number scalar returns false",
			filterRaw: `@.badNum == 1`,
			candidate: beta,
			wantMatch: false,
		},
		{
			name:      "string eq, ne, lt, le, gt, ge",
			filterRaw: `@.name == 'alpha' && @.name != 'beta' && @.name < 'b' && @.name <= 'alpha' && @.name > 'a' && @.name >= 'alpha'`,
			candidate: alpha,
			wantMatch: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parseFilterExpression(tc.filterRaw)
			if err != nil {
				t.Fatalf("parseFilterExpression(%q) error = %v", tc.filterRaw, err)
			}
			gotMatch, gotHL := expr.eval(tc.candidate)
			if gotMatch != tc.wantMatch {
				t.Errorf("eval(%q) = %v, want %v", tc.filterRaw, gotMatch, tc.wantMatch)
			}
			if tc.wantHLKey != "" {
				if gotHL == nil || gotHL.keyNode == nil || gotHL.keyNode.Value != tc.wantHLKey {
					t.Errorf("eval(%q) highlight = %+v, want key %q", tc.filterRaw, gotHL, tc.wantHLKey)
				}
			}
		})
	}
}
