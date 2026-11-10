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

// LogSliceOptions configures multi-window slicing of a structured log YAML body.
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

// SliceLogBody expands multiline string scalars in yamlBody and slices matching windows with context folding.
func SliceLogBody(yamlBody, summary string, opts LogSliceOptions) (*LogSliceResult, error) {
	maxLines := normalizeMaxLines(opts.MaxLines)
	includeSummary := opts.IncludeSummary == nil || *opts.IncludeSummary
	windows, err := compileSliceWindows(opts.Windows)
	if err != nil {
		return nil, err
	}

	expanded := expandMultilineYAMLLines(yamlBody)
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
		if totalLines == 0 {
			return &LogSliceResult{}, nil
		}
		keptEnd := min(totalLines, maxLines)
		return &LogSliceResult{
			TotalLines: totalLines,
			Lines:      renderLogIntervals(expanded, []lineInterval{{start: 0, end: keptEnd}}, nil, ""),
		}, nil
	}

	matchSpansByLine := make(map[int][]byteSpan)
	var rawIntervals []lineInterval

	for _, w := range windows {
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

func expandMultilineYAMLLines(yamlBody string) []string {
	rawLines := workbench.SplitLines(yamlBody)
	if len(rawLines) == 0 {
		return nil
	}
	out := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		prefix, quoted, indent, ok := splitYAMLQuotedScalarLine(line)
		if !ok || !strings.Contains(quoted, `\n`) {
			out = append(out, line)
			continue
		}
		unescaped, ok := unescapeYAMLQuotedString(quoted)
		if !ok || !strings.Contains(unescaped, "\n") {
			out = append(out, line)
			continue
		}
		out = append(out, prefix+"|")
		childIndent := indent + "  "
		subLines := strings.Split(strings.TrimSuffix(unescaped, "\n"), "\n")
		for _, sub := range subLines {
			out = append(out, childIndent+sub)
		}
	}
	return out
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
