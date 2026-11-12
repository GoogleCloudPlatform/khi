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

package tscheck

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestClassifyExecutableTSLines(t *testing.T) {
	testCases := []struct {
		name     string
		src      string
		numLines int
		want     []bool
	}{
		{
			name: "comments quotes escapes and block-comment boundary tokens",
			src: `// leading line comment
/* block comment with // slash and * star
   still inside block comment */
'only single';
"only double";
const s1 = 'it\'s /* not comment';
const afterS1 = 1;
const s2 = 'closed'; /*
inside block after s2
*/
const d1 = "qu\"ote /* not comment";
const afterD1 = 1;
const d2 = "closed"; /*
inside block after d2
*/
x*y
/*/
const hiddenInsideSlashStarSlash = 1;
*/
  });
`,
			numLines: 20,
			want: []bool{
				false,
				false, false, false,
				true, true,
				true, true,
				true, false, false,
				true, true,
				true, false, false,
				true,
				false, false, false,
				false,
			},
		},
		{
			name: "template literals whitespace-only lines and trailing backslash before newline",
			src: "`   \n" +
				"   \n" +
				"\t\n" +
				"\r\n" +
				"escaped \\` still template /*\n" +
				"closing` /*\n" +
				"inside block after template\n" +
				"*/\n" +
				"'trailing backslash before newline \\\n" +
				"/*\n" +
				"*/\n" +
				"`template backslash before newline \\\n" +
				"end` /*\n" +
				"still block\n" +
				"*/\n",
			numLines: 15,
			want: []bool{
				false,
				true, false, false, false, true, true, false, false,
				true, false, false,
				true, true, false, false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyExecutableTSLines([]byte(tc.src), tc.numLines)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("classifyExecutableTSLines() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("eof without trailing newline numLines cap and trailing slash", func(t *testing.T) {
		if diff := cmp.Diff([]bool{false, true, true}, classifyExecutableTSLines([]byte("const a = 1;\nconst b = 2"), 2)); diff != "" {
			t.Errorf("no-newline eof mismatch:\n%s", diff)
		}
		if diff := cmp.Diff([]bool{false, true, true}, classifyExecutableTSLines([]byte("const a = 1;\nconst b = 2;\nconst c = 3;"), 2)); diff != "" {
			t.Errorf("numLines cap mismatch:\n%s", diff)
		}
		if diff := cmp.Diff([]bool{false, true}, classifyExecutableTSLines([]byte("/"), 1)); diff != "" {
			t.Errorf("trailing slash mismatch:\n%s", diff)
		}
	})
}
