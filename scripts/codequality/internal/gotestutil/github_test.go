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

package gotestutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEscapeGitHubHelpers(t *testing.T) {
	testCases := []struct {
		name string
		fn   func(string) string
		in   string
		want string
	}{
		{
			name: "data escapes percent carriage return and newline",
			fn:   EscapeAnnotationData,
			in:   "100%\r\nline2:ok,yes",
			want: "100%25%0D%0Aline2:ok,yes",
		},
		{
			name: "property also escapes colon and comma",
			fn:   EscapeAnnotationProperty,
			in:   "pkg/a:b,c%1\n",
			want: "pkg/a%3Ab%2Cc%251%0A",
		},
		{
			name: "markdown table escapes pipe",
			fn:   EscapeMarkdownTable,
			in:   "a|b",
			want: `a\|b`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.fn(tc.in)
			if got != tc.want {
				t.Errorf("%s(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
			}
		})
	}
}

func TestAppendSummaryFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "summary.md")
	if err := AppendSummaryFile(tmpFile, []byte("first\n")); err != nil {
		t.Fatalf("first AppendSummaryFile() failed: %v", err)
	}
	if err := AppendSummaryFile(tmpFile, []byte("second\n")); err != nil {
		t.Fatalf("second AppendSummaryFile() failed: %v", err)
	}

	raw, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed reading summary file: %v", err)
	}
	want := `first
second
`
	if string(raw) != want {
		t.Errorf("summary content = %q, want %q", string(raw), want)
	}
}
