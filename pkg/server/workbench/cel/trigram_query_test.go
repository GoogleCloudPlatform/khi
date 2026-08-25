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

package cel

import (
	"regexp/syntax"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRegexToTrigramQuery(t *testing.T) {
	testCases := []struct {
		name       string
		pattern    string
		wantString string
	}{
		{
			name:       "literal shorter than 3 runes becomes ALL",
			pattern:    "ab",
			wantString: "ALL",
		},
		{
			name:       "exact 3-rune literal becomes TERM",
			pattern:    "foo",
			wantString: `TERM("foo")`,
		},
		{
			name:       "case-insensitive literal is converted to lowercase",
			pattern:    "FoObAr",
			wantString: `AND(TERM("foo"), TERM("oob"), TERM("oba"), TERM("bar"))`,
		},
		{
			name:       "concatenation with wildcards eliminates ALL",
			pattern:    "foo.*bar",
			wantString: `AND(TERM("foo"), TERM("bar"))`,
		},
		{
			name:       "alternation with two 3-rune terms",
			pattern:    "foo|bar",
			wantString: `OR(TERM("foo"), TERM("bar"))`,
		},
		{
			name:       "alternation with an unconstrained branch becomes ALL",
			pattern:    "foo|.*",
			wantString: "ALL",
		},
		{
			name:       "alternation with short literal branch becomes ALL",
			pattern:    "foo|ab",
			wantString: "ALL",
		},
		{
			name:       "complex nested conjunction and disjunction",
			pattern:    "(foo|bar).*baz",
			wantString: `AND(OR(TERM("foo"), TERM("bar")), TERM("baz"))`,
		},
		{
			name:       "alternation of conjunctions with outer suffix concatenation",
			pattern:    "((foo.*bar)|(baz.*qux)).*quux",
			wantString: `AND(OR(AND(TERM("foo"), TERM("bar")), AND(TERM("baz"), TERM("qux"))), TERM("quu"), TERM("uux"))`,
		},
		{
			name:       "deeply nested alternation inside conjunction inside alternation",
			pattern:    "(foo|(bar.*(baz|qux))).*quux",
			wantString: `AND(OR(TERM("foo"), AND(TERM("bar"), OR(TERM("baz"), TERM("qux")))), TERM("quu"), TERM("uux"))`,
		},
		{
			name:       "alternation of conjunctions of alternations",
			pattern:    "(a11|b22).*(c33|d44)|(e55|f66).*(g77|h88)",
			wantString: `OR(AND(OR(TERM("a11"), TERM("b22")), OR(TERM("c33"), TERM("d44"))), AND(OR(TERM("e55"), TERM("f66")), OR(TERM("g77"), TERM("h88"))))`,
		},
		{
			name:       "nested groups with short literal branch simplified to ALL and reduced",
			pattern:    "((foo|bar).*baz)|(qux.*(quu|ab))",
			wantString: `OR(AND(OR(TERM("foo"), TERM("bar")), TERM("baz")), TERM("qux"))`,
		},
		{
			name:       "nested captures and plus operators",
			pattern:    "((pod-a)+)",
			wantString: `AND(TERM("pod"), TERM("od-"), TERM("d-a"))`,
		},
		{
			name:       "only wildcards becomes ALL",
			pattern:    ".*",
			wantString: "ALL",
		},
		{
			name:       "unicode multibyte string with 3 runes",
			pattern:    "エラー",
			wantString: `TERM("エラー")`,
		},
		{
			name:       "unicode multibyte string with more than 3 runes",
			pattern:    "エラーログ",
			wantString: `AND(TERM("エラー"), TERM("ラーロ"), TERM("ーログ"))`,
		},
		{
			name:       "unicode multibyte string with fewer than 3 runes becomes ALL",
			pattern:    "バグ",
			wantString: "ALL",
		},
		{
			name:       "mixed unicode and ascii string",
			pattern:    "pod-エラー",
			wantString: `AND(TERM("pod"), TERM("od-"), TERM("d-エ"), TERM("-エラ"), TERM("エラー"))`,
		},
		{
			name:       "unicode case folding with accents",
			pattern:    "CAFÉ",
			wantString: `AND(TERM("caf"), TERM("afé"))`,
		},
		{
			name:       "unicode alternation with full and short branches",
			pattern:    "エラー|バグ",
			wantString: "ALL",
		},
		{
			name:       "unicode alternation with multi-rune branches",
			pattern:    "エラー|警告ログ",
			wantString: `OR(TERM("エラー"), AND(TERM("警告ロ"), TERM("告ログ")))`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			syn, err := syntax.Parse(tc.pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("failed to parse regex: %v", err)
			}
			syn = syn.Simplify()
			query := RegexToTrigramQuery(syn).Simplify()
			gotString := query.String()

			if diff := cmp.Diff(tc.wantString, gotString); diff != "" {
				t.Errorf("RegexToTrigramQuery(%q) mismatch (-want +got):\n%s", tc.pattern, diff)
			}
		})
	}
}
