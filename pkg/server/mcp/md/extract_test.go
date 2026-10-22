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

package md

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractInlineCodeSpans(t *testing.T) {
	testCases := []struct {
		name      string
		markdown  string
		wantSpans []InlineCodeSpan
	}{
		{
			name:     "extracts single and multiple inline code spans with line numbers",
			markdown: "# Report\nPod `pod-a` failed with `OOMKilled`.\nInsert ID: `abc123`.",
			wantSpans: []InlineCodeSpan{
				{Text: "pod-a", Line: 2},
				{Text: "OOMKilled", Line: 2},
				{Text: "abc123", Line: 3},
			},
		},
		{
			name:     "skips backtick and tilde fenced code blocks including blockquote fences",
			markdown: "Before `outside-1`\n```yaml\nname: `inside-backtick-fence`\n```\nMiddle `outside-2`\n> ~~~bash\n> echo `inside-tilde-fence`\n> ~~~\nAfter `outside-3`",
			wantSpans: []InlineCodeSpan{
				{Text: "outside-1", Line: 1},
				{Text: "outside-2", Line: 5},
				{Text: "outside-3", Line: 9},
			},
		},
		{
			name:     "supports multi-backtick inline code spans and strips one symmetric space",
			markdown: "Inline `` `nested` `` and ```triple-inline``` on one line.",
			wantSpans: []InlineCodeSpan{
				{Text: "`nested`", Line: 1},
				{Text: "triple-inline", Line: 1},
			},
		},
		{
			name:      "ignores unmatched backticks, escaped backticks, and whitespace-only code spans",
			markdown:  "Unmatched `backtick on line 1\nEmpty `   ` span on line 2.\nEscaped \\`not-a-span\\` and \\\\`valid-after-double-backslash` on line 3.",
			wantSpans: []InlineCodeSpan{{Text: "valid-after-double-backslash", Line: 3}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractInlineCodeSpans(tc.markdown)
			if diff := cmp.Diff(tc.wantSpans, got); diff != "" {
				t.Errorf("ExtractInlineCodeSpans() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
