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

package decisioncheck

import "strings"

// BundledFile pairs a repository-relative path with its raw file bytes in deterministic bundle order.
type BundledFile struct {
	// RelPath is the slash-separated repository-relative path.
	RelPath string
	// Content holds the raw file contents.
	Content []byte
}

type rawHeadingSpan struct {
	heading   string
	startLine int
	endLine   int
}

// ExtractMarkdownSections returns substantive H1 through H4 heading sections from content.
func ExtractMarkdownSections(targetID string, relFile string, content string) []ExtractedItem {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	spans := scanHeadingSpans(lines)
	items := make([]ExtractedItem, 0, len(spans))
	for _, span := range spans {
		trimmedEnd, substantive := substantiveSectionEnd(lines, span.startLine, span.endLine)
		if !substantive {
			continue
		}
		items = append(items, ExtractedItem{
			Target:    targetID,
			File:      relFile,
			Line:      span.startLine,
			StartLine: span.startLine,
			EndLine:   trimmedEnd,
			Heading:   span.heading,
		})
	}
	return items
}

func scanHeadingSpans(lines []string) []rawHeadingSpan {
	var spans []rawHeadingSpan
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if heading, ok := parseMarkdownHeading(trimmed); ok {
			if len(spans) > 0 {
				spans[len(spans)-1].endLine = i
			}
			spans = append(spans, rawHeadingSpan{
				heading:   heading,
				startLine: i + 1,
				endLine:   len(lines),
			})
		}
	}
	return spans
}

func parseMarkdownHeading(trimmed string) (string, bool) {
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	hashes := 0
	for hashes < len(trimmed) && trimmed[hashes] == '#' {
		hashes++
	}
	if hashes > 4 || hashes >= len(trimmed) || trimmed[hashes] != ' ' {
		return "", false
	}
	return strings.TrimSpace(trimmed[hashes:]), true
}

func substantiveSectionEnd(lines []string, startLine int, endLine int) (int, bool) {
	end := startLine
	hasBody := false
	for idx := startLine; idx < endLine; idx++ {
		trimmed := strings.TrimSpace(lines[idx])
		if trimmed == "" {
			continue
		}
		end = idx + 1
		if !strings.HasPrefix(trimmed, "```") && !strings.HasPrefix(trimmed, "~~~") {
			hasBody = true
		}
	}
	return end, hasBody
}

func firstNonEmpty(candidates ...string) string {
	for _, candidate := range candidates {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}
