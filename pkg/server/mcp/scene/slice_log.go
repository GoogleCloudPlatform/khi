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
	"fmt"
	"strconv"
	"strings"
	"time"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

// LogSliceOptions configures JSONPath and/or substring window slicing of a structured log YAML body.
type LogSliceOptions struct {
	Windows        []SliceWindow
	IncludeSummary *bool
	MaxLines       int
}

// LogSliceResult holds the sliced lines and total expanded line count for a log entry.
type LogSliceResult struct {
	TotalLines int
	Lines      []ExcerptLine
}

// LogExcerptSlice combines log metadata and linked timelines with sliced log lines.
type LogExcerptSlice struct {
	LogID           uint32
	Time            time.Time
	Severity        *khifilev6.Severity
	LogType         string
	Summary         string
	LinkedTimelines []workbench.LinkedTimeline
	Slice           LogSliceResult
}

// SliceLogDetail fetches the untruncated YAML body for logID from wb and slices it according to opts.
func SliceLogDetail(wb *workbench.Workbench, logID uint32, opts LogSliceOptions) (*LogExcerptSlice, error) {
	raw, err := wb.GetRawLogDetail(logID)
	if err != nil {
		return nil, fmt.Errorf("failed to read raw log detail: %w", err)
	}
	sliced, err := SliceLogBody(raw.YAMLBody, raw.Summary, opts)
	if err != nil {
		return nil, err
	}
	return &LogExcerptSlice{
		LogID:           raw.LogID,
		Time:            raw.Time,
		Severity:        raw.Severity,
		LogType:         raw.LogType,
		Summary:         raw.Summary,
		LinkedTimelines: raw.LinkedTimelines,
		Slice:           *sliced,
	}, nil
}

// SliceLogBody expands multiline string scalars in yamlBody and slices matching JSONPath and/or query windows with context folding.
func SliceLogBody(yamlBody, summary string, opts LogSliceOptions) (*LogSliceResult, error) {
	maxLines := normalizeMaxLines(opts.MaxLines)
	includeSummary := opts.IncludeSummary == nil || *opts.IncludeSummary
	windows, err := compileSliceWindows(opts.Windows)
	if err != nil {
		return nil, err
	}

	expanded, rawToExpStart, rawToExpEnd := expandMultilineYAMLLinesWithMap(yamlBody)
	totalLines := len(expanded)

	if len(windows) == 0 {
		if summary != "" {
			return &LogSliceResult{
				TotalLines: totalLines,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindSummary, Text: summary},
				},
			}, nil
		}
		return nil, &InvalidSliceWindowError{
			WindowIndex: -1,
			Reason:      "windows must contain at least one path or query selector when log summary is empty",
		}
	}

	var parsedDoc *indexedYAMLDocument
	ensureParsedDoc := func() (*indexedYAMLDocument, error) {
		if parsedDoc != nil {
			return parsedDoc, nil
		}
		doc, parseErr := parseIndexedYAMLDocument(yamlBody)
		if parseErr != nil {
			return nil, parseErr
		}
		parsedDoc = doc
		return parsedDoc, nil
	}

	matchSpansByLine := make(map[int][]byteSpan)
	var rawIntervals []lineInterval

	for _, w := range windows {
		if w.path != nil {
			doc, parseErr := ensureParsedDoc()
			if parseErr != nil {
				return nil, parseErr
			}
			selections := evalCompiledJSONPath(doc, *w.path)
			if len(selections) == 0 {
				matchedPrefix, availableKeys := diagnoseUnmatchedJSONPath(*w.path, doc)
				return nil, &SliceQueryNotFoundError{
					WindowIndex:   w.index,
					Path:          w.path.raw,
					MatchedPrefix: matchedPrefix,
					AvailableKeys: availableKeys,
					TopLevelKeys:  extractTopLevelYAMLKeys(yamlBody),
					Summary:       summary,
				}
			}

			var scopedIntervals []lineInterval
			for _, sel := range selections {
				expStart := rawToExpStart[sel.startLine-1]
				expEnd := rawToExpEnd[sel.endLine-1]
				scopedIntervals = append(scopedIntervals, lineInterval{start: expStart, end: expEnd})
				if w.query == "" && sel.highlightLine > 0 {
					expHL := rawToExpStart[sel.highlightLine-1]
					_, hlSpan := trimmedContentSpanOnLine(expanded, expHL+1)
					if hlSpan.end > hlSpan.start {
						matchSpansByLine[expHL] = append(matchSpansByLine[expHL], hlSpan)
					}
				}
			}

			if w.query == "" {
				rawIntervals = append(rawIntervals, scopedIntervals...)
				continue
			}

			// Search w.query strictly within the YAML subtrees selected by w.path.
			matchedCount := 0
			for _, scope := range mergeIntervals(scopedIntervals) {
				for lineIdx := scope.start; lineIdx < scope.end; lineIdx++ {
					spans := findQuerySpans(expanded[lineIdx], w)
					if len(spans) == 0 {
						continue
					}
					matchSpansByLine[lineIdx] = append(matchSpansByLine[lineIdx], spans...)
					rawIntervals = append(rawIntervals, lineInterval{
						start: max(scope.start, lineIdx-w.before),
						end:   min(scope.end, lineIdx+w.after+1),
					})
					matchedCount++
					if matchedCount >= w.maxMatches {
						break
					}
				}
				if matchedCount >= w.maxMatches {
					break
				}
			}
			if matchedCount == 0 {
				return nil, &SliceQueryNotFoundError{
					WindowIndex:  w.index,
					Path:         w.path.raw,
					Query:        w.query,
					TopLevelKeys: extractTopLevelYAMLKeys(yamlBody),
					Summary:      summary,
				}
			}
			continue
		}

		matchedCount := 0
		for lineIdx, line := range expanded {
			spans := findQuerySpans(line, w)
			if len(spans) == 0 {
				continue
			}
			matchSpansByLine[lineIdx] = append(matchSpansByLine[lineIdx], spans...)
			rawIntervals = append(rawIntervals, lineInterval{
				start: max(0, lineIdx-w.before),
				end:   min(totalLines, lineIdx+w.after+1),
			})
			matchedCount++
			if matchedCount >= w.maxMatches {
				break
			}
		}
		if matchedCount == 0 {
			return nil, &SliceQueryNotFoundError{
				WindowIndex:  w.index,
				Query:        w.query,
				TopLevelKeys: extractTopLevelYAMLKeys(yamlBody),
				Summary:      summary,
			}
		}
	}

	prependedSummary := ""
	if includeSummary && summary != "" {
		prependedSummary = summary
	}
	merged := capIntervals(mergeIntervals(rawIntervals), maxLines)
	return &LogSliceResult{
		TotalLines: totalLines,
		Lines:      renderLogIntervals(expanded, merged, matchSpansByLine, prependedSummary),
	}, nil
}

func renderLogIntervals(
	expanded []string,
	intervals []lineInterval,
	matchSpansByLine map[int][]byteSpan,
	prependedSummary string,
) []ExcerptLine {
	var out []ExcerptLine
	if prependedSummary != "" {
		out = append(out, ExcerptLine{
			Kind: ExcerptLineKindSummary,
			Text: prependedSummary,
		})
	}
	if len(intervals) == 0 {
		return out
	}

	if intervals[0].start > 0 {
		out = append(out, makeLogOmittedLine(intervals[0].start))
	}
	for idx, iv := range intervals {
		if idx > 0 {
			if gap := iv.start - intervals[idx-1].end; gap > 0 {
				out = append(out, makeLogOmittedLine(gap))
			}
		}
		for lineIdx := iv.start; lineIdx < iv.end; lineIdx++ {
			text := expanded[lineIdx]
			out = append(out, ExcerptLine{
				Kind:       ExcerptLineKindContext,
				NewLineNum: lineIdx + 1,
				Text:       text,
				Parts:      buildLineParts(text, matchSpansByLine[lineIdx]),
			})
		}
	}
	lastEnd := intervals[len(intervals)-1].end
	if trailing := len(expanded) - lastEnd; trailing > 0 {
		out = append(out, makeLogOmittedLine(trailing))
	}
	return out
}

func makeLogOmittedLine(count int) ExcerptLine {
	unit := "lines"
	if count == 1 {
		unit = "line"
	}
	return ExcerptLine{
		Kind:    ExcerptLineKindOmitted,
		Omitted: count,
		Text:    fmt.Sprintf("⋮ %d %s omitted", count, unit),
	}
}

func expandMultilineYAMLLinesWithMap(yamlBody string) (expanded []string, rawToExpStart []int, rawToExpEnd []int) {
	rawLines := workbench.SplitLines(yamlBody)
	if len(rawLines) == 0 {
		return nil, nil, nil
	}
	expanded = make([]string, 0, len(rawLines))
	rawToExpStart = make([]int, len(rawLines))
	rawToExpEnd = make([]int, len(rawLines))

	for rawIdx, line := range rawLines {
		rawToExpStart[rawIdx] = len(expanded)
		prefix, quoted, indent, ok := splitYAMLQuotedScalarLine(line)
		if !ok || !strings.Contains(quoted, `\n`) {
			expanded = append(expanded, line)
			rawToExpEnd[rawIdx] = len(expanded)
			continue
		}
		unescaped, ok := unescapeYAMLQuotedString(quoted)
		if !ok || !strings.Contains(unescaped, "\n") {
			expanded = append(expanded, line)
			rawToExpEnd[rawIdx] = len(expanded)
			continue
		}
		expanded = append(expanded, prefix+"|")
		childIndent := indent + "  "
		subLines := strings.Split(strings.TrimSuffix(unescaped, "\n"), "\n")
		for _, sub := range subLines {
			expanded = append(expanded, childIndent+sub)
		}
		rawToExpEnd[rawIdx] = len(expanded)
	}
	return expanded, rawToExpStart, rawToExpEnd
}

func splitYAMLQuotedScalarLine(line string) (prefix string, quoted string, indent string, ok bool) {
	trimmedLeft := strings.TrimLeft(line, " ")
	indent = line[:len(line)-len(trimmedLeft)]
	if strings.HasPrefix(trimmedLeft, `- "`) && strings.HasSuffix(trimmedLeft, `"`) && len(trimmedLeft) >= 4 {
		return indent + "- ", trimmedLeft[2:], indent, true
	}
	colonIdx := findYAMLKeyDelimiter(trimmedLeft)
	if colonIdx < 0 {
		return "", "", "", false
	}
	val := trimmedLeft[colonIdx+2:]
	if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
		return indent + trimmedLeft[:colonIdx+2], val, indent, true
	}
	return "", "", "", false
}

func unescapeYAMLQuotedString(quoted string) (string, bool) {
	if len(quoted) < 2 || quoted[0] != '"' || quoted[len(quoted)-1] != '"' {
		return "", false
	}
	inner := quoted[1 : len(quoted)-1]
	var sb strings.Builder
	sb.Grow(len(inner))
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c != '\\' {
			sb.WriteByte(c)
			continue
		}
		i++
		if i >= len(inner) {
			return "", false
		}
		switch inner[i] {
		case '"':
			sb.WriteByte('"')
		case '\\':
			sb.WriteByte('\\')
		case 'n':
			sb.WriteByte('\n')
		case 'r':
			sb.WriteByte('\r')
		case 't':
			sb.WriteByte('\t')
		case 'x':
			if i+2 >= len(inner) {
				return "", false
			}
			v, err := strconv.ParseUint(inner[i+1:i+3], 16, 8)
			if err != nil {
				return "", false
			}
			sb.WriteByte(byte(v))
			i += 2
		default:
			return "", false
		}
	}
	return sb.String(), true
}
