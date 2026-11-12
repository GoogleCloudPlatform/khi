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

// DiffSliceOptions configures JSONPath slicing of a resource revision diff.
type DiffSliceOptions struct {
	Paths    []string
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

// SliceUnifiedDiff slices JSONPath-selected fields (both changed and unchanged) between oldText and newText with context folding.
func SliceUnifiedDiff(oldText, newText string, isInitialRevision bool, opts DiffSliceOptions) (*DiffSliceResult, error) {
	if isInitialRevision {
		oldText = ""
	}
	maxLines := normalizeMaxLines(opts.MaxLines)
	compiledPaths, err := compileJSONPaths(opts.Paths)
	if err != nil {
		return nil, err
	}

	oldDoc, err := parseIndexedYAMLDocument(oldText)
	if err != nil {
		return nil, err
	}
	newDoc, err := parseIndexedYAMLDocument(newText)
	if err != nil {
		return nil, err
	}

	ops := workbench.DiffLines(oldDoc.lines, newDoc.lines)
	changes := workbench.CountDiffOps(ops)
	hunks := collectDiffHunks(ops)
	totalDiffLines := 0
	for _, h := range hunks {
		totalDiffLines += h.opEnd - h.opStart
	}

	if len(ops) == 0 {
		return nil, &SliceQueryNotFoundError{
			WindowIndex: compiledPaths[0].index,
			Path:        compiledPaths[0].raw,
			Preview:     "(empty resource diff)",
		}
	}

	selectedOp := make([]bool, len(ops))
	oldHighlights := make(map[int][]byteSpan)
	newHighlights := make(map[int][]byteSpan)

	for _, p := range compiledPaths {
		oldSels := evalCompiledJSONPath(oldDoc, p)
		newSels := evalCompiledJSONPath(newDoc, p)
		if len(oldSels) == 0 && len(newSels) == 0 {
			matchedPrefix, availableKeys := diagnoseUnmatchedJSONPath(p, newDoc, oldDoc)
			topKeys := extractChangedTopLevelYAMLKeys(ops)
			if len(topKeys) == 0 {
				topKeys = extractTopLevelYAMLKeys(newText)
			}
			preview := buildDiffPreview(ops, hunks)
			if preview == "" {
				preview = "(no manifest changes outside metadata.managedFields)"
			}
			return nil, &SliceQueryNotFoundError{
				WindowIndex:   p.index,
				Path:          p.raw,
				MatchedPrefix: matchedPrefix,
				AvailableKeys: availableKeys,
				TopLevelKeys:  topKeys,
				Preview:       preview,
			}
		}

		oldLineSet := buildSelectedLineBitmap(len(oldDoc.lines), oldSels, oldHighlights)
		newLineSet := buildSelectedLineBitmap(len(newDoc.lines), newSels, newHighlights)
		markMatchingDiffOps(ops, oldLineSet, newLineSet, len(oldSels) > 0, len(newSels) > 0, selectedOp)
	}

	orderedOps, orderedSelected := groupSelectedReplacementPairs(ops, selectedOp)
	matchSpansByOp := make(map[int][]byteSpan)
	var rawIntervals []lineInterval
	runStart := -1
	for i, sel := range orderedSelected {
		if sel {
			if runStart < 0 {
				runStart = i
			}
			op := orderedOps[i]
			switch op.Kind {
			case workbench.DiffOpDelete:
				if spans := oldHighlights[op.OldPos+1]; len(spans) > 0 {
					matchSpansByOp[i] = append(matchSpansByOp[i], spans...)
				}
			case workbench.DiffOpInsert, workbench.DiffOpEqual:
				if spans := newHighlights[op.NewPos+1]; len(spans) > 0 {
					matchSpansByOp[i] = append(matchSpansByOp[i], spans...)
				} else if op.Kind == workbench.DiffOpEqual {
					if oldSpans := oldHighlights[op.OldPos+1]; len(oldSpans) > 0 {
						matchSpansByOp[i] = append(matchSpansByOp[i], oldSpans...)
					}
				}
			}
		} else if runStart >= 0 {
			rawIntervals = append(rawIntervals, lineInterval{start: runStart, end: i})
			runStart = -1
		}
	}
	if runStart >= 0 {
		rawIntervals = append(rawIntervals, lineInterval{start: runStart, end: len(orderedSelected)})
	}

	merged := capIntervals(mergeIntervals(rawIntervals), maxLines)
	return &DiffSliceResult{
		Changes:        changes,
		TotalDiffLines: totalDiffLines,
		Lines:          renderDiffIntervals(orderedOps, hunks, merged, matchSpansByOp),
	}, nil
}

func buildSelectedLineBitmap(totalLines int, sels []yamlLineSelection, highlights map[int][]byteSpan) []bool {
	if len(sels) == 0 || totalLines == 0 {
		return nil
	}
	selected := make([]bool, totalLines+1)
	for _, s := range sels {
		lo := clampInt(s.startLine, 1, totalLines)
		hi := clampInt(s.endLine, lo, totalLines)
		for line := lo; line <= hi; line++ {
			selected[line] = true
		}
		if s.highlightLine >= 1 && s.highlightLine <= totalLines {
			highlights[s.highlightLine] = append(highlights[s.highlightLine], s.highlightSpan)
		}
	}
	return selected
}

func markMatchingDiffOps(
	ops []workbench.DiffOp,
	oldLineSet []bool,
	newLineSet []bool,
	hasOld bool,
	hasNew bool,
	selectedOp []bool,
) {
	inOld := func(lineNum int) bool {
		return lineNum >= 1 && lineNum < len(oldLineSet) && oldLineSet[lineNum]
	}
	inNew := func(lineNum int) bool {
		return lineNum >= 1 && lineNum < len(newLineSet) && newLineSet[lineNum]
	}

	for i, op := range ops {
		switch op.Kind {
		case workbench.DiffOpEqual:
			switch {
			case hasOld && hasNew:
				if inOld(op.OldPos+1) && inNew(op.NewPos+1) {
					selectedOp[i] = true
				}
			case hasNew:
				if inNew(op.NewPos + 1) {
					selectedOp[i] = true
				}
			case hasOld:
				if inOld(op.OldPos + 1) {
					selectedOp[i] = true
				}
			}
		case workbench.DiffOpDelete:
			if hasOld && inOld(op.OldPos+1) {
				selectedOp[i] = true
			}
		case workbench.DiffOpInsert:
			if hasNew && inNew(op.NewPos+1) {
				selectedOp[i] = true
			}
		}
	}
}

// groupSelectedReplacementPairs keeps paired selected '-' and '+' sub-runs adjacent within multi-line replacement blocks
// so unselected sibling changes in the same block fold into leading/interior/trailing omitted counts rather than splitting '-/+'.
func groupSelectedReplacementPairs(ops []workbench.DiffOp, selected []bool) ([]workbench.DiffOp, []bool) {
	outOps := make([]workbench.DiffOp, 0, len(ops))
	outSel := make([]bool, 0, len(selected))

	i := 0
	for i < len(ops) {
		if ops[i].Kind != workbench.DiffOpDelete {
			outOps = append(outOps, ops[i])
			outSel = append(outSel, selected[i])
			i++
			continue
		}
		delStart := i
		for i < len(ops) && ops[i].Kind == workbench.DiffOpDelete {
			i++
		}
		insStart := i
		for i < len(ops) && ops[i].Kind == workbench.DiffOpInsert {
			i++
		}
		pairEnd := i
		if insStart == pairEnd {
			outOps = append(outOps, ops[delStart:pairEnd]...)
			outSel = append(outSel, selected[delStart:pairEnd]...)
			continue
		}

		delRuns := collectSelectedRuns(selected, delStart, insStart)
		insRuns := collectSelectedRuns(selected, insStart, pairEnd)
		if len(delRuns) == 0 || len(insRuns) == 0 || len(delRuns) != len(insRuns) {
			outOps = append(outOps, ops[delStart:pairEnd]...)
			outSel = append(outSel, selected[delStart:pairEnd]...)
			continue
		}

		appendRange := func(lo, hi int, sel bool) {
			for idx := lo; idx < hi; idx++ {
				outOps = append(outOps, ops[idx])
				outSel = append(outSel, sel)
			}
		}

		// Leading unselected '-' and '+' ops before the first selected pair.
		appendRange(delStart, delRuns[0].start, false)
		appendRange(insStart, insRuns[0].start, false)

		for k := range delRuns {
			if k > 0 {
				// Interior unselected '-' and '+' ops between pair k-1 and pair k.
				appendRange(delRuns[k-1].end, delRuns[k].start, false)
				appendRange(insRuns[k-1].end, insRuns[k].start, false)
			}
			appendRange(delRuns[k].start, delRuns[k].end, true)
			appendRange(insRuns[k].start, insRuns[k].end, true)
		}

		// Trailing unselected '-' and '+' ops after the last selected pair.
		lastK := len(delRuns) - 1
		appendRange(delRuns[lastK].end, insStart, false)
		appendRange(insRuns[lastK].end, pairEnd, false)
	}

	return outOps, outSel
}

func collectSelectedRuns(selected []bool, lo, hi int) []lineInterval {
	var runs []lineInterval
	runStart := -1
	for i := lo; i < hi; i++ {
		if selected[i] {
			if runStart < 0 {
				runStart = i
			}
		} else if runStart >= 0 {
			runs = append(runs, lineInterval{start: runStart, end: i})
			runStart = -1
		}
	}
	if runStart >= 0 {
		runs = append(runs, lineInterval{start: runStart, end: hi})
	}
	return runs
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
			out = append(out, renderGap(ops, hunks, prevEnd, iv.start)...)
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

func renderGap(ops []workbench.DiffOp, hunks []diffHunkRange, gapStart, gapEnd int) []ExcerptLine {
	var out []ExcerptLine
	cursor := gapStart
	for _, h := range hunks {
		if h.opEnd <= cursor {
			continue
		}
		if h.opStart >= gapEnd {
			break
		}
		lo := max(h.opStart, cursor)
		hi := min(h.opEnd, gapEnd)
		if gapHasChanges(ops, lo, hi) {
			if lo > cursor {
				out = append(out, makeDiffOmittedLine(lo-cursor, false))
			}
			out = append(out, makeDiffOmittedLine(hi-lo, true))
			cursor = hi
		}
	}
	if gapEnd > cursor {
		out = append(out, makeDiffOmittedLine(gapEnd-cursor, false))
	}
	return out
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
