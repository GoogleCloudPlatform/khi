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

// DiffSliceOptions configures JSONPath slicing of a resource revision diff.
type DiffSliceOptions struct {
	// Paths specifies non-empty JSONPath selectors evaluated against both old and new revisions.
	Paths []string
	// MaxLines caps the total number of diff lines returned across all selected paths.
	MaxLines int
}

// DiffSliceResult holds the sliced lines and summary counts for a unified diff.
type DiffSliceResult struct {
	// Changes holds the total added and deleted line counts across the full diff.
	Changes workbench.LineChangeCount
	// TotalDiffLines is the total number of lines across all unified diff hunks.
	TotalDiffLines int
	// Lines contains the sliced excerpt lines including omitted-gap markers.
	Lines []ExcerptLine
}

// ResourceDiffSlice combines revision metadata with a sliced managedFields-clean unified diff.
type ResourceDiffSlice struct {
	// TimelineID is the resource timeline ID containing the revision.
	TimelineID uint32
	// Segments holds the timeline path segments identifying the resource.
	Segments []workbench.TimelineSegment
	// Revision holds metadata for the target resource revision.
	Revision workbench.ResourceRevision
	// Previous holds metadata for the preceding revision, or nil for the initial revision.
	Previous *workbench.ResourceRevision
	// NextRevisionTime is the timestamp of the following revision, or nil if this is the latest revision.
	NextRevisionTime *time.Time
	// Slice contains the sliced unified diff lines and change counts.
	Slice DiffSliceResult
}

type diffHunkRange struct {
	opStart int
	opEnd   int
}

type diffSlicer struct {
	ops           []workbench.DiffOp
	hunks         []diffHunkRange
	oldDoc        *indexedYAMLDocument
	newDoc        *indexedYAMLDocument
	selectedOp    []bool
	oldHighlights map[int][]byteSpan
	newHighlights map[int][]byteSpan
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
	if len(ops) == 0 {
		return nil, &SliceQueryNotFoundError{
			WindowIndex: compiledPaths[0].index,
			Path:        compiledPaths[0].raw,
			Preview:     "(empty resource diff)",
		}
	}

	changes := workbench.CountDiffOps(ops)
	hunks := collectDiffHunks(ops)
	slicer := &diffSlicer{
		ops:           ops,
		hunks:         hunks,
		oldDoc:        oldDoc,
		newDoc:        newDoc,
		selectedOp:    make([]bool, len(ops)),
		oldHighlights: make(map[int][]byteSpan),
		newHighlights: make(map[int][]byteSpan),
	}
	if err := slicer.selectPaths(compiledPaths, newText); err != nil {
		return nil, err
	}

	orderedOps, orderedSelected := groupSelectedReplacementPairs(ops, slicer.selectedOp)
	rawIntervals := collectSelectedRuns(orderedSelected, 0, len(orderedSelected))
	matchSpansByOp := slicer.collectSelectedMatchSpans(orderedOps, orderedSelected)
	merged := capIntervals(mergeIntervals(rawIntervals), maxLines)
	return &DiffSliceResult{
		Changes:        changes,
		TotalDiffLines: countTotalHunkLines(hunks),
		Lines:          renderDiffIntervals(orderedOps, hunks, merged, matchSpansByOp),
	}, nil
}

func (s *diffSlicer) selectPaths(paths []compiledJSONPath, newText string) error {
	for _, p := range paths {
		oldSels := evalCompiledJSONPath(s.oldDoc, p)
		newSels := evalCompiledJSONPath(s.newDoc, p)
		if len(oldSels) == 0 && len(newSels) == 0 {
			return s.unmatchedPathError(p, newText)
		}
		oldLineSet := buildSelectedLineBitmap(len(s.oldDoc.lines), oldSels, s.oldHighlights)
		newLineSet := buildSelectedLineBitmap(len(s.newDoc.lines), newSels, s.newHighlights)
		markMatchingDiffOps(s.ops, oldLineSet, newLineSet, s.selectedOp)
	}
	return nil
}

func (s *diffSlicer) unmatchedPathError(p compiledJSONPath, newText string) error {
	matchedPrefix, availableKeys := diagnoseUnmatchedJSONPath(p, s.newDoc, s.oldDoc)
	topKeys := extractChangedTopLevelYAMLKeys(s.ops)
	if len(topKeys) == 0 {
		topKeys = extractTopLevelYAMLKeys(newText)
	}
	preview := buildDiffPreview(s.ops, s.hunks)
	if preview == "" {
		preview = "(no manifest changes outside metadata.managedFields)"
	}
	return &SliceQueryNotFoundError{
		WindowIndex:   p.index,
		Path:          p.raw,
		MatchedPrefix: matchedPrefix,
		AvailableKeys: availableKeys,
		TopLevelKeys:  topKeys,
		Preview:       preview,
	}
}

func (s *diffSlicer) collectSelectedMatchSpans(orderedOps []workbench.DiffOp, orderedSelected []bool) map[int][]byteSpan {
	matchSpansByOp := make(map[int][]byteSpan)
	for i, sel := range orderedSelected {
		if !sel {
			continue
		}
		op := orderedOps[i]
		switch op.Kind {
		case workbench.DiffOpDelete:
			matchSpansByOp[i] = append(matchSpansByOp[i], s.oldHighlights[op.OldPos]...)
		case workbench.DiffOpInsert:
			matchSpansByOp[i] = append(matchSpansByOp[i], s.newHighlights[op.NewPos]...)
		default:
			spans := s.newHighlights[op.NewPos]
			if len(spans) == 0 {
				spans = s.oldHighlights[op.OldPos]
			}
			matchSpansByOp[i] = append(matchSpansByOp[i], spans...)
		}
	}
	return matchSpansByOp
}

func countTotalHunkLines(hunks []diffHunkRange) int {
	total := 0
	for _, h := range hunks {
		total += h.opEnd - h.opStart
	}
	return total
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

func changedHunkOverlap(ops []workbench.DiffOp, h diffHunkRange, start, end int) (int, int, bool) {
	lo := max(h.opStart, start)
	hi := min(h.opEnd, end)
	return lo, hi, gapHasChanges(ops, lo, hi)
}

func countSkippedDiffLines(ops []workbench.DiffOp, hunks []diffHunkRange, gapStart, gapEnd int) int {
	total := 0
	for _, h := range hunks {
		if lo, hi, ok := changedHunkOverlap(ops, h, gapStart, gapEnd); ok {
			total += hi - lo
		}
	}
	return total
}

func renderGap(ops []workbench.DiffOp, hunks []diffHunkRange, gapStart, gapEnd int) []ExcerptLine {
	var out []ExcerptLine
	cursor := gapStart
	for _, h := range hunks {
		if lo, hi, ok := changedHunkOverlap(ops, h, cursor, gapEnd); ok {
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
		Text:    fmt.Sprintf("%d %s omitted", count, unit),
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
