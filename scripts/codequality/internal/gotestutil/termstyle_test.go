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
	"bytes"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type fakeFileInfo struct {
	mode fs.FileMode
}

func (f fakeFileInfo) Name() string       { return "fake" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() fs.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

type fakeStatBuffer struct {
	bytes.Buffer
	mode fs.FileMode
	err  error
}

func (b *fakeStatBuffer) Stat() (fs.FileInfo, error) {
	if b.err != nil {
		return nil, b.err
	}
	return fakeFileInfo{mode: b.mode}, nil
}

func TestParseColorMode(t *testing.T) {
	testCases := []struct {
		name    string
		raw     string
		want    ColorMode
		wantErr bool
	}{
		{name: "empty defaults to auto", raw: "", want: ColorAuto},
		{name: "auto uppercase", raw: "AUTO", want: ColorAuto},
		{name: "always", raw: "always", want: ColorAlways},
		{name: "never", raw: "never", want: ColorNever},
		{name: "invalid value", raw: "rainbow", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseColorMode(tc.raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseColorMode(%q) error = %v, wantErr %v", tc.raw, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ParseColorMode(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestResolveColorEnabled(t *testing.T) {
	ttyWriter := &fakeStatBuffer{mode: fs.ModeCharDevice}
	fileWriter := &fakeStatBuffer{mode: 0}
	errWriter := &fakeStatBuffer{err: errors.New("stat failed")}
	plainBuf := &bytes.Buffer{}

	testCases := []struct {
		name   string
		mode   ColorMode
		writer interface{ Write([]byte) (int, error) }
		env    map[string]string
		want   bool
	}{
		{
			name:   "always overrides NO_COLOR and non-tty",
			mode:   ColorAlways,
			writer: plainBuf,
			env:    map[string]string{"NO_COLOR": "1"},
			want:   true,
		},
		{
			name:   "never overrides tty and FORCE_COLOR",
			mode:   ColorNever,
			writer: ttyWriter,
			env:    map[string]string{"FORCE_COLOR": "1"},
			want:   false,
		},
		{
			name:   "auto enables on char device tty",
			mode:   ColorAuto,
			writer: ttyWriter,
			env:    map[string]string{"TERM": "xterm-256color"},
			want:   true,
		},
		{
			name:   "auto disables when NO_COLOR is set",
			mode:   ColorAuto,
			writer: ttyWriter,
			env:    map[string]string{"NO_COLOR": "1"},
			want:   false,
		},
		{
			name:   "auto disables when TERM is dumb",
			mode:   ColorAuto,
			writer: ttyWriter,
			env:    map[string]string{"TERM": "dumb"},
			want:   false,
		},
		{
			name:   "auto enables on non-tty when FORCE_COLOR is 1",
			mode:   ColorAuto,
			writer: plainBuf,
			env:    map[string]string{"FORCE_COLOR": "1"},
			want:   true,
		},
		{
			name:   "auto ignores FORCE_COLOR=0",
			mode:   ColorAuto,
			writer: plainBuf,
			env:    map[string]string{"FORCE_COLOR": "0"},
			want:   false,
		},
		{
			name:   "auto disables in GITHUB_ACTIONS",
			mode:   ColorAuto,
			writer: ttyWriter,
			env:    map[string]string{"GITHUB_ACTIONS": "true"},
			want:   false,
		},
		{
			name:   "auto disables on regular file",
			mode:   ColorAuto,
			writer: fileWriter,
			want:   false,
		},
		{
			name:   "auto disables when stat fails",
			mode:   ColorAuto,
			writer: errWriter,
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(k string) string {
				return tc.env[k]
			}
			got := ResolveColorEnabled(tc.mode, tc.writer, getenv)
			if got != tc.want {
				t.Errorf("ResolveColorEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveInteractive(t *testing.T) {
	ttyWriter := &fakeStatBuffer{mode: fs.ModeCharDevice}
	plainBuf := &bytes.Buffer{}

	testCases := []struct {
		name   string
		writer interface{ Write([]byte) (int, error) }
		getenv func(string) string
		want   bool
	}{
		{
			name:   "disables on non-tty writer",
			writer: plainBuf,
			getenv: func(string) string { return "" },
			want:   false,
		},
		{
			name:   "enables on char-device tty with standard terminal",
			writer: ttyWriter,
			getenv: func(k string) string {
				if k == "TERM" {
					return "xterm-256color"
				}
				return ""
			},
			want: true,
		},
		{
			name:   "enables on char-device tty with nil getenv",
			writer: ttyWriter,
			getenv: nil,
			want:   true,
		},
		{
			name:   "disables when TERM is dumb",
			writer: ttyWriter,
			getenv: func(k string) string {
				if k == "TERM" {
					return "dumb"
				}
				return ""
			},
			want: false,
		},
		{
			name:   "disables in GITHUB_ACTIONS",
			writer: ttyWriter,
			getenv: func(k string) string {
				if k == "GITHUB_ACTIONS" {
					return "true"
				}
				return ""
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveInteractive(tc.writer, tc.getenv); got != tc.want {
				t.Errorf("ResolveInteractive() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTermStyleScoreAndFormatting(t *testing.T) {
	off := TermStyle{Enabled: false}
	if got := off.BoldRed("alert"); got != "alert" {
		t.Errorf("disabled style BoldRed() = %q, want %q", got, "alert")
	}

	on := TermStyle{Enabled: true}
	if got := on.Bold(""); got != "" {
		t.Errorf("empty string Bold() = %q, want empty", got)
	}
	if got := on.ScoreText(80.0, "80.0%"); got != "\x1b[1;32m80.0%\x1b[0m" {
		t.Errorf("ScoreText(80.0) = %q, want bold green", got)
	}
	if got := on.ScoreText(60.0, "60.0%"); got != "\x1b[1;33m60.0%\x1b[0m" {
		t.Errorf("ScoreText(60.0) = %q, want bold yellow", got)
	}
	if got := on.ScoreText(59.9, "59.9%"); got != "\x1b[1;31m59.9%\x1b[0m" {
		t.Errorf("ScoreText(59.9) = %q, want bold red", got)
	}
}

func TestRenderAlignedTable(t *testing.T) {
	style := TermStyle{Enabled: true}
	headers := []string{"FILE", "SCORE", "SURVIVED"}
	rows := [][]TableCell{
		{
			{Plain: "pkg/a.go"},
			{Plain: "100.0% (4/4)", Styled: style.Green("100.0% (4/4)"), AlignRight: true},
			{Plain: "0", Styled: style.Dim("0"), AlignRight: true},
		},
		{
			{Plain: "pkg/longer/b.go"},
			{Plain: "50.0% (1/2)", Styled: style.Red("50.0% (1/2)"), AlignRight: true},
			{Plain: "1", Styled: style.BoldRed("1"), AlignRight: true},
		},
	}

	var plainBuf bytes.Buffer
	plainRows := [][]TableCell{
		{
			{Plain: "pkg/a.go"},
			{Plain: "100.0% (4/4)", AlignRight: true},
			{Plain: "0", AlignRight: true},
		},
		{
			{Plain: "pkg/longer/b.go"},
			{Plain: "50.0% (1/2)", AlignRight: true},
			{Plain: "1", AlignRight: true},
		},
	}
	RenderAlignedTable(&plainBuf, "  ", headers, plainRows, TermStyle{Enabled: false})
	wantPlain := `  FILE                    SCORE  SURVIVED
  pkg/a.go         100.0% (4/4)         0
  pkg/longer/b.go   50.0% (1/2)         1
`
	if diff := cmp.Diff(wantPlain, plainBuf.String()); diff != "" {
		t.Errorf("RenderAlignedTable(plain) mismatch (-want +got):\n%s", diff)
	}

	var colorBuf bytes.Buffer
	RenderAlignedTable(&colorBuf, "  ", headers, rows, style)
	if !strings.Contains(colorBuf.String(), "\x1b[1;31m1\x1b[0m") {
		t.Errorf("expected styled survived count in color table, got:\n%s", colorBuf.String())
	}
}

func TestHighlightInlineDiff(t *testing.T) {
	style := TermStyle{Enabled: true}

	t.Run("plain mode returns unchanged strings", func(t *testing.T) {
		gotOld, gotNew := HighlightInlineDiff("if x > 0 {", "if x >= 0 {", TermStyle{Enabled: false})
		if gotOld != "if x > 0 {" || gotNew != "if x >= 0 {" {
			t.Errorf("got (%q, %q), want unmodified lines", gotOld, gotNew)
		}
	})

	t.Run("highlights operator token without splitting", func(t *testing.T) {
		gotOld, gotNew := HighlightInlineDiff("if x > 0 {", "if x >= 0 {", style)
		wantOld := "\x1b[2mif x \x1b[0m\x1b[1;31m>\x1b[0m\x1b[2m 0 {\x1b[0m"
		wantNew := "\x1b[2mif x \x1b[0m\x1b[1;32m>=\x1b[0m\x1b[2m 0 {\x1b[0m"
		if gotOld != wantOld {
			t.Errorf("gotOld = %q, want %q", gotOld, wantOld)
		}
		if gotNew != wantNew {
			t.Errorf("gotNew = %q, want %q", gotNew, wantNew)
		}
	})

	t.Run("keeps whole words intact on shared suffix rune", func(t *testing.T) {
		gotOld, gotNew := HighlightInlineDiff("return true", "return false", style)
		wantOld := "\x1b[2mreturn \x1b[0m\x1b[1;31mtrue\x1b[0m"
		wantNew := "\x1b[2mreturn \x1b[0m\x1b[1;32mfalse\x1b[0m"
		if gotOld != wantOld {
			t.Errorf("gotOld = %q, want %q", gotOld, wantOld)
		}
		if gotNew != wantNew {
			t.Errorf("gotNew = %q, want %q", gotNew, wantNew)
		}
	})
}
