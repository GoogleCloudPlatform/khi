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

import "strings"

// InlineCodeSpan represents a single inline code span extracted from a Markdown document.
type InlineCodeSpan struct {
	// Text is the unescaped content inside the inline backticks.
	Text string
	// Line is the 1-based line number where the inline code span starts.
	Line int
}

// ExtractInlineCodeSpans extracts non-empty CommonMark inline code spans from a Markdown document while skipping fenced code blocks.
func ExtractInlineCodeSpans(markdown string) []InlineCodeSpan {
	normalized := strings.ReplaceAll(markdown, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")

	var spans []InlineCodeSpan
	inFence := false
	var fenceChar byte
	fenceLen := 0

	for i, line := range lines {
		lineNum := i + 1
		stripped := stripBlockquoteAndIndent(line)

		if !inFence {
			if ch, count, ok := parseOpeningFence(stripped); ok {
				inFence = true
				fenceChar = ch
				fenceLen = count
				continue
			}
			spans = append(spans, extractLineInlineCodeSpans(line, lineNum)...)
		} else if isClosingFence(stripped, fenceChar, fenceLen) {
			inFence = false
			fenceChar = 0
			fenceLen = 0
		}
	}

	return spans
}

// stripBlockquoteAndIndent removes leading blockquote markers ('>') and whitespace from a line so fence markers inside blockquotes or lists are recognized.
func stripBlockquoteAndIndent(line string) string {
	s := line
	for {
		s = strings.TrimLeft(s, " \t")
		if strings.HasPrefix(s, ">") {
			s = s[1:]
			continue
		}
		return s
	}
}

// parseOpeningFence checks whether a line begins a CommonMark fenced code block of at least 3 backticks or tildes.
func parseOpeningFence(stripped string) (byte, int, bool) {
	if len(stripped) < 3 {
		return 0, 0, false
	}
	ch := stripped[0]
	if ch != '`' && ch != '~' {
		return 0, 0, false
	}
	count := countLeadingByte(stripped, ch)
	if count < 3 {
		return 0, 0, false
	}
	info := stripped[count:]
	// Per CommonMark spec, a backtick fence info string cannot contain backtick characters.
	if ch == '`' && strings.IndexByte(info, '`') >= 0 {
		return 0, 0, false
	}
	return ch, count, true
}

// isClosingFence checks whether a line closes an active fenced code block.
func isClosingFence(stripped string, fenceChar byte, minLen int) bool {
	if len(stripped) < minLen || stripped[0] != fenceChar {
		return false
	}
	count := countLeadingByte(stripped, fenceChar)
	if count < minLen {
		return false
	}
	return strings.TrimSpace(stripped[count:]) == ""
}

func countLeadingByte(s string, b byte) int {
	i := 0
	for i < len(s) && s[i] == b {
		i++
	}
	return i
}

// extractLineInlineCodeSpans extracts non-empty CommonMark inline code spans from a single line.
func extractLineInlineCodeSpans(line string, lineNum int) []InlineCodeSpan {
	var spans []InlineCodeSpan
	i := 0
	n := len(line)

	for i < n {
		if line[i] != '`' {
			i++
			continue
		}
		backslashCount := 0
		for k := i - 1; k >= 0 && line[k] == '\\'; k-- {
			backslashCount++
		}
		if backslashCount%2 != 0 {
			i++
			continue
		}
		openLen := countLeadingByte(line[i:], '`')
		contentStart := i + openLen

		closeStart := findClosingBacktickRun(line, contentStart, openLen)
		if closeStart < 0 {
			i = contentStart
			continue
		}

		content := normalizeInlineCodeContent(line[contentStart:closeStart])
		if strings.TrimSpace(content) != "" {
			spans = append(spans, InlineCodeSpan{
				Text: content,
				Line: lineNum,
			})
		}
		i = closeStart + openLen
	}

	return spans
}

// findClosingBacktickRun finds the index of the next backtick run of exact length targetLen starting at or after start.
func findClosingBacktickRun(line string, start, targetLen int) int {
	j := start
	n := len(line)
	for j < n {
		if line[j] != '`' {
			j++
			continue
		}
		runLen := countLeadingByte(line[j:], '`')
		if runLen == targetLen {
			return j
		}
		j += runLen
	}
	return -1
}

// normalizeInlineCodeContent strips a single leading and trailing space if both are present and the span is not all spaces, matching CommonMark code span normalization.
func normalizeInlineCodeContent(s string) string {
	if len(s) >= 2 && s[0] == ' ' && s[len(s)-1] == ' ' && strings.Trim(s, " ") != "" {
		return s[1 : len(s)-1]
	}
	return s
}
