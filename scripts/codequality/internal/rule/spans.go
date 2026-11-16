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

package rule

import "strings"

// LineRange represents an inclusive 1-based [StartLine, EndLine] interval of uncovered source lines.
type LineRange struct {
	// StartLine is the 1-based inclusive start line of the range.
	StartLine int
	// EndLine is the 1-based inclusive end line of the range.
	EndLine int
}

// SplitSourceLines splits content into 1-based line strings without trailing CR/LF characters.
func SplitSourceLines(content string) []string {
	trimmed := strings.TrimSuffix(content, "\n")
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	return lines
}

// BuildLineRanges merges nearby uncovered 1-based line numbers when no substantive code line intervenes within 3 lines.
func BuildLineRanges(uncovered []int, codeLines []bool) []LineRange {
	if len(uncovered) == 0 {
		return nil
	}
	var ranges []LineRange
	cur := LineRange{StartLine: uncovered[0], EndLine: uncovered[0]}
	for _, ln := range uncovered[1:] {
		if canMergeUncoveredLines(cur.EndLine, ln, codeLines) {
			cur.EndLine = ln
			continue
		}
		ranges = append(ranges, cur)
		cur = LineRange{StartLine: ln, EndLine: ln}
	}
	return append(ranges, cur)
}

// BuildUncoveredSpans builds UncoveredSpan snippets (including a leading header '{' line when present) from uncovered line numbers.
func BuildUncoveredSpans(relFile string, rawLines []string, codeLines []bool, uncovered []int) []UncoveredSpan {
	ranges := BuildLineRanges(uncovered, codeLines)
	if len(ranges) == 0 {
		return nil
	}
	uncovSet := make(map[int]struct{}, len(uncovered))
	for _, ln := range uncovered {
		uncovSet[ln] = struct{}{}
	}
	spans := make([]UncoveredSpan, 0, len(ranges))
	for _, rg := range ranges {
		spans = append(spans, buildSingleSpan(relFile, rawLines, rg, uncovSet))
	}
	return spans
}

func canMergeUncoveredLines(prevLine int, nextLine int, codeLines []bool) bool {
	if nextLine-prevLine > 3 {
		return false
	}
	for ln := prevLine + 1; ln < nextLine; ln++ {
		if ln < len(codeLines) && codeLines[ln] {
			return false
		}
	}
	return true
}

func buildSingleSpan(relFile string, rawLines []string, rg LineRange, uncovSet map[int]struct{}) UncoveredSpan {
	snippetStart := rg.StartLine
	if snippetStart > 1 {
		prevTrim := strings.TrimSpace(rawLines[snippetStart-2])
		if strings.HasSuffix(prevTrim, "{") && !strings.HasPrefix(prevTrim, "//") {
			snippetStart--
		}
	}
	var lines []SourceLine
	execCount := 0
	for ln := snippetStart; ln <= rg.EndLine && ln <= len(rawLines); ln++ {
		lines = append(lines, SourceLine{Number: ln, Text: rawLines[ln-1]})
		if ln >= rg.StartLine {
			if _, ok := uncovSet[ln]; ok {
				execCount++
			}
		}
	}
	return UncoveredSpan{
		File:            relFile,
		StartLine:       rg.StartLine,
		EndLine:         rg.EndLine,
		ExecutableLines: execCount,
		Lines:           lines,
	}
}
