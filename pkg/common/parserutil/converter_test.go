// Copyright 2024 Google LLC
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

package parserutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestStripSpecialSequences(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strip nothing",
			input:    "this is text",
			expected: "this is text",
		},
		{
			name:     "strip escape sequences",
			input:    "this is\\r\\n text\\r\\n",
			expected: "this is text",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			breaklineStripper := &SequenceConverter{From: []string{"\\r", "\\n"}}
			actual := ConvertSpecialSequences(tc.input, breaklineStripper)
			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Errorf("the result is not matching with the expected result\n%s", diff)
			}
		})
	}
}

func TestANSIEscapeSequenceStripper(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strip nothing",
			input:    "this is text",
			expected: "this is text",
		},
		{
			name:     "strip ansi escape sequences",
			input:    "\\x1b[31mthis is red text\\x1b[0m",
			expected: "this is red text",
		},
		{
			name:     "strip ansi escape sequences with multiple begin sequences",
			input:    "\\u001B[31mthis is red text\\033[0m",
			expected: "this is red text",
		},
		{
			name:     "strip ansi escape sequences with incomplete sequence",
			input:    "\\x1b[31mthis is red text\\x1b[",
			expected: "this is red text\\x1b[",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stripper := ANSIEscapeSequenceStripper{}
			actual := stripper.Convert(tc.input)
			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Errorf("the result is not matching with the expected result\n%s", diff)
			}
		})
	}
}

func TestUnicodeUnquoteConverter_Convert(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "simple",
			input: "Job cri-containerd-06a622d26bbe9788\\xe2\\x80\\xa6/stop running (1min 7s / 1min 30s)",
			want:  "Job cri-containerd-06a622d26bbe9788…/stop running (1min 7s / 1min 30s)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UnicodeUnquoteConverter{}
			got := u.Convert(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("UnicodeUnquoteConverter.Convert() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
