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
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

const maxSlicePreviewLines = 6

type diffHunkRange struct {
	opStart int
	opEnd   int
}

// DiffSliceOptions configures multi-window slicing of a resource revision diff.
type DiffSliceOptions struct {
	Windows  []SliceWindow
	MaxLines int
}

// DiffSliceResult holds the sliced lines and summary counts for a unified diff.
type DiffSliceResult struct {
	Changes        workbench.LineChangeCount
	TotalDiffLines int
	Lines          []ExcerptLine
}

// ResourceDiffSlice combines revision metadata with a sliced managedFields-clean unified diff.
type ResourceDiffSlice struct {
	TimelineID       uint32
	Segments         []workbench.TimelineSegment
	Revision         workbench.ResourceRevision
	Previous         *workbench.ResourceRevision
	NextRevisionTime *time.Time
	Slice            DiffSliceResult
}

// SliceResourceDiff fetches the managedFields-stripped revision pair from wb and slices its unified diff according to opts.
func SliceResourceDiff(wb *workbench.Workbench, timelineID uint32, revisionIndex int, opts DiffSliceOptions) (*ResourceDiffSlice, error) {
	pair, err := wb.GetRevisionPair(timelineID, revisionIndex, false)
	if err != nil {
		return nil, fmt.Errorf("failed to read revision pair: %w", err)
	}
	sliced, err := SliceUnifiedDiff(pair.PreviousBody, pair.Body, revisionIndex == 0, opts)
	if err != nil {
		return nil, err
	}
	return &ResourceDiffSlice{
		TimelineID:       pair.TimelineID,
		Segments:         pair.Segments,
		Revision:         pair.Revision,
		Previous:         pair.Previous,
		NextRevisionTime: pair.NextRevisionTime,
		Slice:            *sliced,
	}, nil
}

// SliceUnifiedDiff slices changed hunks between oldText and newText using multi-window search and context folding.
func SliceUnifiedDiff(oldText, newText string, isInitialRevision bool, opts DiffSliceOptions) (*DiffSliceResult, error) {
	if isInitialRevision {
		oldText = ""
	}
	maxLines := normalizeMaxLines(opts.MaxLines)
	preservePair := true
	windows, err := compileSliceWindows(opts.Windows)
	if err != nil {
		return nil, err
	}

	ops := workbench.DiffLines(workbench.SplitLines(oldText), workbench.SplitLines(newText))
	changes := workbench.CountDiffOps(ops)
	hunks := collectDiffHunks(ops)
	totalDiffLines := 0
	for _, h := range hunks {
		totalDiffLines += h.opEnd - h.opStart
	}

	if len(windows) == 0 {
		if len(hunks) == 0 {
			return &DiffSliceResult{Changes: changes}, nil
		}
		kept := selectLeadingDiffIntervals(hunks, maxLines)
		return &DiffSliceResult{
			Changes:        changes,
			TotalDiffLines: totalDiffLines,
			Lines:          renderDiffIntervals(ops, hunks, kept, nil),
		}, nil
	}

	if len(hunks) == 0 {
		return nil, &SliceQueryNotFoundError{
			WindowIndex:  windows[0].index,
			Query:        windows[0].query,
			TopLevelKeys: extractTopLevelYAMLKeys(newText),
			Preview:      "(no manifest changes outside metadata.managedFields)",
		}
	}

	pairs := collectReplacementPairs(ops, hunks)
	matchSpansByOp := make(map[int][]byteSpan)
	var rawIntervals []lineInterval

	for _, w := range windows {
		matchedCount := 0
		for _, h := range hunks {
			for opIdx := h.opStart; opIdx < h.opEnd; opIdx++ {
				spans := findQuerySpans(ops[opIdx].Line, w)
				if len(spans) == 0 {
					continue
				}
				matchSpansByOp[opIdx] = append(matchSpansByOp[opIdx], spans...)
				rawIntervals = append(rawIntervals, expandDiffMatchInterval(opIdx, h, pairs, w.before, w.after, preservePair, maxLines))
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
				Query:        w.query,
				TopLevelKeys: extractChangedTopLevelYAMLKeys(ops),
				Preview:      buildDiffPreview(ops, hunks),
			}
		}
	}

	merged := capIntervals(mergeIntervals(rawIntervals), maxLines)
	return &DiffSliceResult{
		Changes:        changes,
		TotalDiffLines: totalDiffLines,
		Lines:          renderDiffIntervals(ops, hunks, merged, matchSpansByOp),
	}, nil
}

func collectDiffHunks(ops []workbench.DiffOp) []diffHunkRange {
	var hunks []diffHunkRange
	i := 0
	for i < len(ops) {
		for i < len(ops) && ops[i].Kind == workbench.DiffOpEqual {
			i++
		}
		if i >= len(ops) {
			break
		}
		start := max(0, i-workbench.UnifiedDiffContextLines)
		lastChange := i
		for j := i; j < len(ops); j++ {
			if ops[j].Kind != workbench.DiffOpEqual {
				if j-lastChange > 2*workbench.UnifiedDiffContextLines {
					break
				}
				lastChange = j
			}
		}
		end := min(len(ops), lastChange+workbench.UnifiedDiffContextLines+1)
		hunks = append(hunks, diffHunkRange{opStart: start, opEnd: end})
		i = end
	}
	return hunks
}

func collectReplacementPairs(ops []workbench.DiffOp, hunks []diffHunkRange) []lineInterval {
	var pairs []lineInterval
	for _, h := range hunks {
		i := h.opStart
		for i < h.opEnd {
			if ops[i].Kind != workbench.DiffOpDelete {
				i++
				continue
			}
			delStart := i
			for i < h.opEnd && ops[i].Kind == workbench.DiffOpDelete {
				i++
			}
			insStart := i
			for i < h.opEnd && ops[i].Kind == workbench.DiffOpInsert {
				i++
			}
			if i > insStart {
				pairs = append(pairs, lineInterval{start: delStart, end: i})
			}
		}
	}
	return pairs
}

func findReplacementPair(pairs []lineInterval, opIdx int) (lineInterval, bool) {
	for _, p := range pairs {
		if opIdx >= p.start && opIdx < p.end {
			return p, true
		}
	}
	return lineInterval{}, false
}

func expandDiffMatchInterval(
	opIdx int,
	hunk diffHunkRange,
	pairs []lineInterval,
	before, after int,
	preservePair bool,
	maxLines int,
) lineInterval {
	coreStart := opIdx
	coreEnd := opIdx + 1
	if preservePair {
		if pair, ok := findReplacementPair(pairs, opIdx); ok && pair.end-pair.start <= maxLines {
			coreStart = pair.start
			coreEnd = pair.end
		}
	}

	segStart := max(hunk.opStart, coreStart-before)
	segEnd := min(hunk.opEnd, coreEnd+after)

	if preservePair {
		if pair, ok := findReplacementPair(pairs, segStart); ok {
			candStart := min(segStart, pair.start)
			if segEnd-candStart <= maxLines {
				segStart = max(hunk.opStart, candStart)
			}
		}
		if segEnd > segStart {
			if pair, ok := findReplacementPair(pairs, segEnd-1); ok {
				candEnd := max(segEnd, pair.end)
				if candEnd-segStart <= maxLines {
					segEnd = min(hunk.opEnd, candEnd)
				}
			}
		}
	}
	return lineInterval{start: segStart, end: segEnd}
}

func selectLeadingDiffIntervals(hunks []diffHunkRange, maxLines int) []lineInterval {
	remaining := maxLines
	var out []lineInterval
	for _, h := range hunks {
		if remaining <= 0 {
			break
		}
		take := min(h.opEnd-h.opStart, remaining)
		out = append(out, lineInterval{start: h.opStart, end: h.opStart + take})
		remaining -= take
	}
	return out
}

func renderDiffIntervals(
	ops []workbench.DiffOp,
	hunks []diffHunkRange,
	intervals []lineInterval,
	matchSpansByOp map[int][]byteSpan,
) []ExcerptLine {
	if len(intervals) == 0 {
		return nil
	}
	var out []ExcerptLine

	if leadCount := countSkippedDiffLines(ops, hunks, 0, intervals[0].start); leadCount > 0 {
		out = append(out, makeDiffOmittedLine(leadCount, true))
	}

	for idx, iv := range intervals {
		if idx > 0 {
			prevEnd := intervals[idx-1].end
			if gapHasChanges(ops, prevEnd, iv.start) {
				if diffSkipped := countSkippedDiffLines(ops, hunks, prevEnd, iv.start); diffSkipped > 0 {
					out = append(out, makeDiffOmittedLine(diffSkipped, true))
				}
			} else if unchangedSkipped := iv.start - prevEnd; unchangedSkipped > 0 {
				out = append(out, makeDiffOmittedLine(unchangedSkipped, false))
			}
		}

		for opIdx := iv.start; opIdx < iv.end; opIdx++ {
			op := ops[opIdx]
			line := ExcerptLine{
				Text:  op.Line,
				Parts: buildLineParts(op.Line, matchSpansByOp[opIdx]),
			}
			switch op.Kind {
			case workbench.DiffOpEqual:
				line.Kind = ExcerptLineKindContext
				line.OldLineNum = op.OldPos + 1
				line.NewLineNum = op.NewPos + 1
			case workbench.DiffOpDelete:
				line.Kind = ExcerptLineKindDeleted
				line.OldLineNum = op.OldPos + 1
			case workbench.DiffOpInsert:
				line.Kind = ExcerptLineKindAdded
				line.NewLineNum = op.NewPos + 1
			}
			out = append(out, line)
		}
	}

	lastEnd := intervals[len(intervals)-1].end
	if trailCount := countSkippedDiffLines(ops, hunks, lastEnd, len(ops)); trailCount > 0 {
		out = append(out, makeDiffOmittedLine(trailCount, true))
	}
	return out
}

func gapHasChanges(ops []workbench.DiffOp, start, end int) bool {
	for i := start; i < end; i++ {
		if ops[i].Kind != workbench.DiffOpEqual {
			return true
		}
	}
	return false
}

func countSkippedDiffLines(ops []workbench.DiffOp, hunks []diffHunkRange, gapStart, gapEnd int) int {
	total := 0
	for _, h := range hunks {
		lo := max(h.opStart, gapStart)
		hi := min(h.opEnd, gapEnd)
		if lo >= hi {
			continue
		}
		if gapHasChanges(ops, lo, hi) {
			total += hi - lo
		}
	}
	return total
}

func makeDiffOmittedLine(count int, hasChanges bool) ExcerptLine {
	unit := "unchanged lines"
	if hasChanges {
		if count == 1 {
			unit = "diff line"
		} else {
			unit = "diff lines"
		}
	} else if count == 1 {
		unit = "unchanged line"
	}
	return ExcerptLine{
		Kind:    ExcerptLineKindOmitted,
		Omitted: count,
		Text:    fmt.Sprintf("⋮ %d %s omitted", count, unit),
	}
}

func extractChangedTopLevelYAMLKeys(ops []workbench.DiffOp) []string {
	keys := make(map[string]struct{})
	currentOldTop := ""
	currentNewTop := ""
	for _, op := range ops {
		if k, ok := parseTopLevelYAMLKey(op.Line); ok {
			switch op.Kind {
			case workbench.DiffOpEqual:
				currentOldTop = k
				currentNewTop = k
			case workbench.DiffOpDelete:
				currentOldTop = k
				keys[k] = struct{}{}
			case workbench.DiffOpInsert:
				currentNewTop = k
				keys[k] = struct{}{}
			}
			continue
		}
		switch op.Kind {
		case workbench.DiffOpDelete:
			if currentOldTop != "" {
				keys[currentOldTop] = struct{}{}
			}
		case workbench.DiffOpInsert:
			if currentNewTop != "" {
				keys[currentNewTop] = struct{}{}
			}
		}
	}
	return slices.Sorted(maps.Keys(keys))
}

func buildDiffPreview(ops []workbench.DiffOp, hunks []diffHunkRange) string {
	var lines []string
	for _, h := range hunks {
		for i := h.opStart; i < h.opEnd && len(lines) < maxSlicePreviewLines; i++ {
			switch ops[i].Kind {
			case workbench.DiffOpDelete:
				lines = append(lines, "-"+ops[i].Line)
			case workbench.DiffOpInsert:
				lines = append(lines, "+"+ops[i].Line)
			}
		}
		if len(lines) >= maxSlicePreviewLines {
			break
		}
	}
	return strings.Join(lines, "\n")
}
