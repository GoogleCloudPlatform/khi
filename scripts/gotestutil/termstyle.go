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
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// ColorAuto enables ANSI color only on interactive terminals unless overridden by environment variables.
	ColorAuto ColorMode = "auto"
	// ColorAlways forces ANSI color escape sequences regardless of output destination.
	ColorAlways ColorMode = "always"
	// ColorNever disables all ANSI color escape sequences.
	ColorNever ColorMode = "never"
)

// ColorMode controls whether ANSI terminal escape sequences are emitted.
type ColorMode string

// TermStyle formats terminal text with optional ANSI SGR escape sequences.
type TermStyle struct {
	// Enabled reports whether ANSI escape sequences are active.
	Enabled bool
}

// TableCell holds a table column's plain text (for width calculation) and pre-styled text.
type TableCell struct {
	// Plain is the unstyled string used to compute visible rune column width.
	Plain string
	// Styled is the rendered string (including optional ANSI codes); defaults to Plain when empty.
	Styled string
	// AlignRight right-aligns the cell within its column when true.
	AlignRight bool
}

type statWriter interface {
	Stat() (os.FileInfo, error)
}

// ParseColorMode validates and normalizes a --color flag value.
func ParseColorMode(raw string) (ColorMode, error) {
	switch ColorMode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", ColorAuto:
		return ColorAuto, nil
	case ColorAlways:
		return ColorAlways, nil
	case ColorNever:
		return ColorNever, nil
	default:
		return "", fmt.Errorf("invalid --color %q (want auto, always, or never)", raw)
	}
}

// IsTerminal reports whether w is backed by a character-device terminal.
func IsTerminal(w io.Writer) bool {
	if sw, ok := w.(statWriter); ok {
		if info, err := sw.Stat(); err == nil {
			return info.Mode()&os.ModeCharDevice != 0
		}
	}
	return false
}

// ResolveColorEnabled determines whether ANSI color sequences should be emitted for mode and w.
func ResolveColorEnabled(mode ColorMode, w io.Writer, getenv func(string) string) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	}

	if getenv != nil {
		if getenv("NO_COLOR") != "" || getenv("TERM") == "dumb" {
			return false
		}
		if fc := getenv("FORCE_COLOR"); fc != "" && fc != "0" {
			return true
		}
		if getenv("GITHUB_ACTIONS") == "true" {
			return false
		}
	}

	return IsTerminal(w)
}

// ResolveInteractive reports whether w supports live in-place terminal progress updates.
func ResolveInteractive(w io.Writer, getenv func(string) string) bool {
	if !IsTerminal(w) {
		return false
	}
	if getenv != nil && (getenv("TERM") == "dumb" || getenv("GITHUB_ACTIONS") == "true") {
		return false
	}
	return true
}

// Bold formats text in bold weight when styling is enabled.
func (s TermStyle) Bold(text string) string { return s.wrap("1", text) }

// Dim formats text in faint/dim intensity when styling is enabled.
func (s TermStyle) Dim(text string) string { return s.wrap("2", text) }

// Red formats text in standard red when styling is enabled.
func (s TermStyle) Red(text string) string { return s.wrap("31", text) }

// Green formats text in standard green when styling is enabled.
func (s TermStyle) Green(text string) string { return s.wrap("32", text) }

// Yellow formats text in standard yellow when styling is enabled.
func (s TermStyle) Yellow(text string) string { return s.wrap("33", text) }

// Cyan formats text in standard cyan when styling is enabled.
func (s TermStyle) Cyan(text string) string { return s.wrap("36", text) }

// BoldRed formats text in bold red when styling is enabled.
func (s TermStyle) BoldRed(text string) string { return s.wrap("1;31", text) }

// BoldGreen formats text in bold green when styling is enabled.
func (s TermStyle) BoldGreen(text string) string { return s.wrap("1;32", text) }

// BoldYellow formats text in bold yellow when styling is enabled.
func (s TermStyle) BoldYellow(text string) string { return s.wrap("1;33", text) }

// BoldCyan formats text in bold cyan when styling is enabled.
func (s TermStyle) BoldCyan(text string) string { return s.wrap("1;36", text) }

// ScoreText colors text according to percentage thresholds (>=80 green, >=60 yellow, <60 red).
func (s TermStyle) ScoreText(pct float64, text string) string {
	if pct >= 80.0 {
		return s.BoldGreen(text)
	}
	if pct >= 60.0 {
		return s.BoldYellow(text)
	}
	return s.BoldRed(text)
}

func (s TermStyle) wrap(code string, text string) string {
	if !s.Enabled || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// RenderAlignedTable writes a padded table to w using visible rune widths so ANSI colors never skew columns.
func RenderAlignedTable(w io.Writer, indent string, headers []string, rows [][]TableCell, style TermStyle) {
	numCols := len(headers)
	if numCols == 0 {
		return
	}

	widths := make([]int, numCols)
	rightAlignHeader := make([]bool, numCols)
	for c, h := range headers {
		widths[c] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for c := 0; c < numCols && c < len(row); c++ {
			if n := utf8.RuneCountInString(row[c].Plain); n > widths[c] {
				widths[c] = n
			}
			if row[c].AlignRight {
				rightAlignHeader[c] = true
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(indent)
	for c, h := range headers {
		if c > 0 {
			sb.WriteString("  ")
		}
		pad := widths[c] - utf8.RuneCountInString(h)
		styledH := style.Bold(h)
		switch {
		case rightAlignHeader[c]:
			sb.WriteString(strings.Repeat(" ", pad))
			sb.WriteString(styledH)
		case c < numCols-1:
			sb.WriteString(styledH)
			sb.WriteString(strings.Repeat(" ", pad))
		default:
			sb.WriteString(styledH)
		}
	}
	fmt.Fprintln(w, sb.String())

	for _, row := range rows {
		sb.Reset()
		sb.WriteString(indent)
		for c := 0; c < numCols; c++ {
			if c > 0 {
				sb.WriteString("  ")
			}
			var cell TableCell
			if c < len(row) {
				cell = row[c]
			}
			rendered := cell.Styled
			if rendered == "" {
				rendered = cell.Plain
			}
			pad := widths[c] - utf8.RuneCountInString(cell.Plain)
			switch {
			case cell.AlignRight:
				sb.WriteString(strings.Repeat(" ", pad))
				sb.WriteString(rendered)
			case c < numCols-1:
				sb.WriteString(rendered)
				sb.WriteString(strings.Repeat(" ", pad))
			default:
				sb.WriteString(rendered)
			}
		}
		fmt.Fprintln(w, sb.String())
	}
}

// HighlightInlineDiff returns oldLine and newLine with unchanged context dimmed and the mutated token span highlighted.
func HighlightInlineDiff(oldLine string, newLine string, style TermStyle) (string, string) {
	if !style.Enabled {
		return oldLine, newLine
	}

	oldRunes := []rune(oldLine)
	newRunes := []rune(newLine)

	prefixLen := 0
	for prefixLen < len(oldRunes) && prefixLen < len(newRunes) && oldRunes[prefixLen] == newRunes[prefixLen] {
		prefixLen++
	}

	suffixLen := 0
	for suffixLen < len(oldRunes)-prefixLen &&
		suffixLen < len(newRunes)-prefixLen &&
		oldRunes[len(oldRunes)-1-suffixLen] == newRunes[len(newRunes)-1-suffixLen] {
		suffixLen++
	}

	// Expand diff bounds outward to token delimiters so words or multi-char operators (e.g. true/false, >/>=) stay intact.
	for prefixLen > 0 {
		keepOld := prefixLen < len(oldRunes) && !isDiffBoundary(oldRunes[prefixLen-1]) && !isDiffBoundary(oldRunes[prefixLen])
		keepNew := prefixLen < len(newRunes) && !isDiffBoundary(newRunes[prefixLen-1]) && !isDiffBoundary(newRunes[prefixLen])
		if !keepOld && !keepNew {
			break
		}
		prefixLen--
	}

	for suffixLen > 0 {
		oldSplit := len(oldRunes) - suffixLen
		newSplit := len(newRunes) - suffixLen
		keepOld := oldSplit > 0 && !isDiffBoundary(oldRunes[oldSplit-1]) && !isDiffBoundary(oldRunes[oldSplit])
		keepNew := newSplit > 0 && !isDiffBoundary(newRunes[newSplit-1]) && !isDiffBoundary(newRunes[newSplit])
		if !keepOld && !keepNew {
			break
		}
		suffixLen--
	}

	prefix := string(oldRunes[:prefixLen])
	suffix := string(oldRunes[len(oldRunes)-suffixLen:])
	oldMid := string(oldRunes[prefixLen : len(oldRunes)-suffixLen])
	newMid := string(newRunes[prefixLen : len(newRunes)-suffixLen])

	styledOld := style.Dim(prefix) + style.BoldRed(oldMid) + style.Dim(suffix)
	styledNew := style.Dim(prefix) + style.BoldGreen(newMid) + style.Dim(suffix)
	return styledOld, styledNew
}

func isDiffBoundary(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("()[]{},;:", r)
}
