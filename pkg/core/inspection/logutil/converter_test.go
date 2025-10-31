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

package logutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStripSpecialSequences(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strip nothing",
			input: "this is text",
			want:  "this is text",
		},
		{
			name:  "strip escape sequences",
			input: "this is\\r\\n text\\r\\n",
			want:  "this is text",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			breaklineStripper := &SequenceConverter{From: []string{"\\r", "\\n"}}
			got := ConvertSpecialSequences(tc.input, breaklineStripper)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Convert() result mismatch (-want,+got)\n%s", diff)
			}
		})
	}
}

func TestRegexSequenceConverter(t *testing.T) {
	testCases := []struct {
		desc            string
		regex           string
		repl            string
		inputStr        string
		wantStr         string
		wantErrOnCreate bool
	}{
		{
			desc:            "simple replacement",
			regex:           "foo",
			repl:            "bar",
			inputStr:        "this is foo text",
			wantStr:         "this is bar text",
			wantErrOnCreate: false,
		},
		{
			desc:            "no match",
			regex:           "xyz",
			repl:            "abc",
			inputStr:        "this is foo text",
			wantStr:         "this is foo text",
			wantErrOnCreate: false,
		},
		{
			desc:            "multiple matches",
			regex:           "foo",
			repl:            "bar",
			inputStr:        "foo bar foo",
			wantStr:         "bar bar bar",
			wantErrOnCreate: false,
		},
		{
			desc:            "invalid regex",
			regex:           "[",
			repl:            "bar",
			inputStr:        "this is foo text",
			wantStr:         "",
			wantErrOnCreate: true,
		},
		{
			desc:            "empty regex",
			regex:           "",
			repl:            "bar",
			inputStr:        "this is text",
			wantStr:         "this is text",
			wantErrOnCreate: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			converter, err := NewRegexSequenceConverter(tc.regex, tc.repl)
			if tc.wantErrOnCreate {
				if err == nil {
					t.Fatalf("NewRegexSequenceConverter() expected an error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewRegexSequenceConverter() unexpected error: %v", err)
			}
			gotStr := converter.Convert(tc.inputStr)
			if diff := cmp.Diff(tc.wantStr, gotStr); diff != "" {
				t.Errorf("Convert() result mismatch (-want,+got)\n%s", diff)
			}
		})
	}
}

func TestANSIEscapeSequenceStripper(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strip nothing",
			input: "this is text",
			want:  "this is text",
		},
		{
			name:  "strip ansi escape sequences",
			input: `\x1b[31mthis is red text\x1b[0m`,
			want:  "this is red text",
		},
		{
			name:  "strip ansi escape sequences with multiple begin sequences",
			input: `\u001B[31mthis is red text\033[0m`,
			want:  "this is red text",
		},
		{
			name:  "strip ansi escape sequences with incomplete sequence",
			input: `\x1b[31mthis is red text\x1b[`,
			want:  `this is red text\x1b[`,
		},
		{
			name:  "strip ansi escape sequence with including non ANSI [ in sequence",
			input: `\x1b[K[ \x1b[0;31m*\x1b[0;1;31m*\x1b[0m\x1b[0;31m*  \x1b[0m] Job cri-containerd-372daaaaac81f0de\xe2\x80\xa6/stop running (1min 3s / 1min 30s)\r\n`,
			want:  `[ ***  ] Job cri-containerd-372daaaaac81f0de\xe2\x80\xa6/stop running (1min 3s / 1min 30s)\r\n`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stripper := ANSIEscapeSequenceStripper{}
			got := stripper.Convert(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Convert() result mismatch (-want,+got)\n%s", diff)
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
