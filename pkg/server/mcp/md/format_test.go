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

import "testing"

func TestCode(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain string without backticks",
			input: "my-id-123",
			want:  "`my-id-123`",
		},
		{
			name:  "empty string",
			input: "",
			want:  "``",
		},
		{
			name:  "embedded single backtick",
			input: "foo`bar",
			want:  "``foo`bar``",
		},
		{
			name:  "leading backtick",
			input: "`foo",
			want:  "`` `foo ``",
		},
		{
			name:  "trailing backtick",
			input: "foo`",
			want:  "`` foo` ``",
		},
		{
			name:  "embedded double backticks",
			input: "foo``bar",
			want:  "```foo``bar```",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Code(tc.input)
			if got != tc.want {
				t.Errorf("Code(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCell(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "escapes pipe character",
			input: "column|value",
			want:  `column\|value`,
		},
		{
			name:  "replaces LF with space",
			input: "line1\nline2",
			want:  "line1 line2",
		},
		{
			name:  "replaces CRLF with space",
			input: "line1\r\nline2",
			want:  "line1 line2",
		},
		{
			name:  "replaces CR with space",
			input: "line1\rline2",
			want:  "line1 line2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Cell(tc.input)
			if got != tc.want {
				t.Errorf("Cell(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFence(t *testing.T) {
	testCases := []struct {
		name     string
		lang     string
		body     string
		wantBody string
	}{
		{
			name:     "simple block",
			lang:     "yaml",
			body:     "key: value",
			wantBody: "```yaml\nkey: value\n```",
		},
		{
			name:     "trims single trailing newline",
			lang:     "yaml",
			body:     "key: value\n",
			wantBody: "```yaml\nkey: value\n```",
		},
		{
			name:     "trims single trailing CRLF",
			lang:     "yaml",
			body:     "key: value\r\n",
			wantBody: "```yaml\nkey: value\n```",
		},
		{
			name:     "enclosing body with backticks",
			lang:     "markdown",
			body:     "```code```",
			wantBody: "````markdown\n```code```\n````",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Fence(tc.lang, tc.body)
			if got != tc.wantBody {
				t.Errorf("Fence(%q, %q) = %q, want %q", tc.lang, tc.body, got, tc.wantBody)
			}
		})
	}
}
