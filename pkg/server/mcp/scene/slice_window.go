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

package scene

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

const (
	// DefaultExcerptContextLines is the default symmetric context kept around each matched line.
	DefaultExcerptContextLines = 1
	// MaxExcerptContextLines caps before and after context lines per match window.
	MaxExcerptContextLines = 5
	// DefaultExcerptMaxMatches is the default number of occurrences sliced per window.
	DefaultExcerptMaxMatches = 2
	// MaxExcerptMaxMatches caps occurrences sliced per window.
	MaxExcerptMaxMatches = 4
	// DefaultExcerptMaxLines is the default maximum number of content lines rendered in one deck card.
	DefaultExcerptMaxLines = 10
	// MaxExcerptMaxLines caps content lines rendered in one deck card.
	MaxExcerptMaxLines = 16
)

// InvalidSliceWindowError reports an invalid SliceWindow configuration such as an empty query.
type InvalidSliceWindowError struct {
	// WindowIndex is the 0-based index of the invalid window.
	WindowIndex int
	// Reason describes why the slice window configuration is invalid.
	Reason string
}

// SliceQueryNotFoundError reports that a SliceWindow query matched zero lines in the target diff or log.
type SliceQueryNotFoundError struct {
	// WindowIndex is the 0-based index of the window whose query matched nothing.
	WindowIndex int
	// Query is the literal search string that produced zero matches.
	Query string
	// TopLevelKeys lists available top-level YAML keys to help refine the query.
	TopLevelKeys []string
	// Preview holds a short preview of changed diff hunks when slicing a diff.
	Preview string
	// Summary holds the log summary when slicing a log entry.
	Summary string
}

type byteSpan struct {
	start int
	end   int
}

type lineInterval struct {
	start int
	end   int
}

type compiledWindow struct {
	index      int
	query      string
	before     int
	after      int
	maxMatches int
}

// Error formats the invalid slice window error message.
func (e *InvalidSliceWindowError) Error() string {
	return fmt.Sprintf("invalid slice window %d: %s", e.WindowIndex, e.Reason)
}

// Error formats the missing query match error message.
func (e *SliceQueryNotFoundError) Error() string {
	return fmt.Sprintf("slice window %d query %q matched no lines", e.WindowIndex, e.Query)
}

func normalizeMaxLines(maxLines int) int {
	if maxLines <= 0 {
		return DefaultExcerptMaxLines
	}
	return min(maxLines, MaxExcerptMaxLines)
}

func compileSliceWindows(windows []SliceWindow) ([]compiledWindow, error) {
	if len(windows) == 0 {
		return nil, nil
	}
	out := make([]compiledWindow, len(windows))
	for i, w := range windows {
		if w.Query == "" {
			return nil, &InvalidSliceWindowError{
				WindowIndex: i,
				Reason:      "query must not be empty",
			}
		}
		before := clampInt(w.BeforeLines, 0, MaxExcerptContextLines)
		after := clampInt(w.AfterLines, 0, MaxExcerptContextLines)
		maxMatches := w.MaxMatches
		if maxMatches <= 0 {
			maxMatches = DefaultExcerptMaxMatches
		}
		maxMatches = min(maxMatches, MaxExcerptMaxMatches)

		out[i] = compiledWindow{
			index:      i,
			query:      w.Query,
			before:     before,
			after:      after,
			maxMatches: maxMatches,
		}
	}
	return out, nil
}

func clampInt(v, lo, hi int) int {
	return min(max(v, lo), hi)
}

func mergeIntervals(intervals []lineInterval) []lineInterval {
	if len(intervals) == 0 {
		return nil
	}
	sorted := slices.Clone(intervals)
	slices.SortFunc(sorted, func(a, b lineInterval) int {
		if c := cmp.Compare(a.start, b.start); c != 0 {
			return c
		}
		return cmp.Compare(a.end, b.end)
	})
	merged := []lineInterval{sorted[0]}
	for _, iv := range sorted[1:] {
		last := &merged[len(merged)-1]
		if iv.start <= last.end {
			last.end = max(last.end, iv.end)
			continue
		}
		merged = append(merged, iv)
	}
	return merged
}

func capIntervals(intervals []lineInterval, maxLines int) []lineInterval {
	remaining := maxLines
	var out []lineInterval
	for _, iv := range intervals {
		if remaining <= 0 {
			break
		}
		length := iv.end - iv.start
		if length <= remaining {
			out = append(out, iv)
			remaining -= length
			continue
		}
		out = append(out, lineInterval{start: iv.start, end: iv.start + remaining})
		break
	}
	return out
}

func findQuerySpans(line string, w compiledWindow) []byteSpan {
	var spans []byteSpan
	searchFrom := 0
	for searchFrom <= len(line) {
		rel := strings.Index(line[searchFrom:], w.query)
		if rel < 0 {
			break
		}
		start := searchFrom + rel
		end := start + len(w.query)
		spans = append(spans, byteSpan{start: start, end: end})
		searchFrom = end
	}
	return spans
}

func buildLineParts(line string, spans []byteSpan) []LinePart {
	if len(spans) == 0 {
		return nil
	}
	var nonEmpty []byteSpan
	for _, s := range spans {
		start := max(0, min(len(line), s.start))
		end := max(start, min(len(line), s.end))
		for start > 0 && start < len(line) && !utf8.RuneStart(line[start]) {
			start--
		}
		for end < len(line) && !utf8.RuneStart(line[end]) {
			end++
		}
		if end > start {
			nonEmpty = append(nonEmpty, byteSpan{start: start, end: end})
		}
	}
	if len(nonEmpty) == 0 {
		return nil
	}

	slices.SortFunc(nonEmpty, func(a, b byteSpan) int {
		if c := cmp.Compare(a.start, b.start); c != 0 {
			return c
		}
		return cmp.Compare(a.end, b.end)
	})
	merged := []byteSpan{nonEmpty[0]}
	for _, s := range nonEmpty[1:] {
		last := &merged[len(merged)-1]
		if s.start <= last.end {
			last.end = max(last.end, s.end)
			continue
		}
		merged = append(merged, s)
	}

	var parts []LinePart
	cursor := 0
	for _, s := range merged {
		if s.start > cursor {
			parts = append(parts, LinePart{Text: line[cursor:s.start]})
		}
		parts = append(parts, LinePart{Text: line[s.start:s.end], Matched: true})
		cursor = s.end
	}
	if cursor < len(line) {
		parts = append(parts, LinePart{Text: line[cursor:]})
	}
	return parts
}

func extractTopLevelYAMLKeys(yamlBody string) []string {
	keys := make(map[string]struct{})
	for _, line := range workbench.SplitLines(yamlBody) {
		if k, ok := parseTopLevelYAMLKey(line); ok {
			keys[k] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(keys))
}

func parseTopLevelYAMLKey(line string) (string, bool) {
	if line == "" || line[0] == ' ' || line[0] == '-' || line[0] == '#' {
		return "", false
	}
	if strings.HasSuffix(line, ":") {
		return strings.Trim(strings.TrimSuffix(line, ":"), `"`), true
	}
	if idx := findYAMLKeyDelimiter(line); idx > 0 {
		return strings.Trim(line[:idx], `"`), true
	}
	return "", false
}

func findYAMLKeyDelimiter(trimmed string) int {
	if len(trimmed) == 0 {
		return -1
	}
	if trimmed[0] == '"' {
		inEscape := false
		for i := 1; i < len(trimmed); i++ {
			if inEscape {
				inEscape = false
				continue
			}
			if trimmed[i] == '\\' {
				inEscape = true
				continue
			}
			if trimmed[i] == '"' {
				if strings.HasPrefix(trimmed[i+1:], ": ") {
					return i + 1
				}
				return -1
			}
		}
		return -1
	}
	return strings.Index(trimmed, ": ")
}
