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
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/md"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/RoaringBitmap/roaring/v2"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type pendingExcerpt struct {
	kind          ExcerptKind
	itemIndex     int
	fieldPath     string
	explicitBadge bool
	card          ResolvedExcerptCard
}

func resolveExcerpts(
	wb *workbench.Workbench,
	inspectionID string,
	spec *Spec,
	applied workbench.AppliedFilter,
	paletteReg *paletteRegistry,
	autoInclude *roaring.Bitmap,
	pendingMarkers []pendingMarker,
	overlay *Overlay,
) (*mcpsdk.CallToolResult, error) {
	ann := spec.Annotations
	var pending []pendingExcerpt

	for i := range ann.DiffExcerpts {
		de := &ann.DiffExcerpts[i]
		fieldPath := fmt.Sprintf("annotations.diffExcerpts[%d]", i)
		if utf8.RuneCountInString(de.Note) > maxAnnotationRunes {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.note` exceeds maximum length of %d characters.", fieldPath, maxAnnotationRunes))
			return res, nil
		}
		paletteID, errRes := validateExplicitPaletteID(paletteReg, fieldPath, de.PaletteID)
		if errRes != nil {
			return errRes, nil
		}
		if de.Revision.TimelineID == 0 {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.revision.timelineId` is required.", fieldPath))
			return res, nil
		}

		sliced, err := SliceResourceDiff(wb, de.Revision.TimelineID, de.Revision.RevisionIndex, DiffSliceOptions{
			Windows: de.Windows,
		})
		if err != nil {
			if errRes, handled := sliceErrorResult(err, inspectionID, de.Revision.TimelineID, 0, fieldPath); handled {
				return errRes, nil
			}
			return nil, fmt.Errorf("failed to slice resource diff for %s: %w", fieldPath, err)
		}
		if errRes := checkViewportRange(sliced.Revision.Time, applied, spec.Filter, fieldPath+".revision"); errRes != nil {
			return errRes, nil
		}

		autoInclude.Add(de.Revision.TimelineID)
		revIdx := de.Revision.RevisionIndex
		title := formatRevisionExcerptTitle(sliced.Revision.Verb, sliced.Revision.State)
		subtitle := fmt.Sprintf("+%d / -%d · %d diff lines", sliced.Slice.Changes.Added, sliced.Slice.Changes.Deleted, sliced.Slice.TotalDiffLines)
		logTimestamp, insertID := resolveCloudLoggingMetadata(wb, sliced.Revision.LogID, sliced.Revision.Time)
		pending = append(pending, pendingExcerpt{
			kind:          ExcerptKindDiff,
			itemIndex:     i,
			fieldPath:     fieldPath,
			explicitBadge: de.Badge != "",
			card: ResolvedExcerptCard{
				Kind:          ExcerptKindDiff,
				Badge:         de.Badge,
				PaletteID:     paletteID,
				TimeMs:        sliced.Revision.Time.UnixMilli(),
				TimeLabel:     formatShiftedClock(sliced.Revision.Time, spec.TimezoneShiftHours),
				TimelineID:    de.Revision.TimelineID,
				ResourcePath:  compactResourcePath(sliced.Segments),
				Title:         title,
				Subtitle:      subtitle,
				Lines:         sliced.Slice.Lines,
				Note:          de.Note,
				LogTimestamp:  logTimestamp,
				InsertID:      insertID,
				LogID:         sliced.Revision.LogID,
				RevisionIndex: &revIdx,
			},
		})
	}

	idx := wb.SearchIndex()
	for i := range ann.ManifestExcerpts {
		me := &ann.ManifestExcerpts[i]
		fieldPath := fmt.Sprintf("annotations.manifestExcerpts[%d]", i)
		if utf8.RuneCountInString(me.Note) > maxAnnotationRunes {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.note` exceeds maximum length of %d characters.", fieldPath, maxAnnotationRunes))
			return res, nil
		}
		paletteID, errRes := validateExplicitPaletteID(paletteReg, fieldPath, me.PaletteID)
		if errRes != nil {
			return errRes, nil
		}
		if me.TimelineID == 0 {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.timelineId` is required.", fieldPath))
			return res, nil
		}
		hasRev := me.RevisionIndex != nil
		hasTime := me.Time != nil && !me.Time.IsZero()
		if (hasRev && me.Time != nil) || (!hasRev && !hasTime) {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Specify exactly one of `revisionIndex` or `time` in `%s`.", fieldPath))
			return res, nil
		}

		var selector workbench.RevisionSelector
		if hasTime {
			if idx == nil || idx.TimelineMap[me.TimelineID] == nil {
				res, _, _ := timelineNotFoundResult(inspectionID, me.TimelineID)
				return res, nil
			}
			queryTime := me.Time.UTC()
			if errRes := checkViewportRange(queryTime, applied, spec.Filter, fieldPath+".time"); errRes != nil {
				return errRes, nil
			}
			selector = workbench.RevisionAtTime(queryTime)
		} else {
			selector = workbench.RevisionAtIndex(*me.RevisionIndex)
		}

		sliced, err := SliceResourceManifest(wb, me.TimelineID, selector, ManifestSliceOptions{
			Windows: me.Windows,
		})
		if err != nil {
			if errRes, handled := sliceErrorResult(err, inspectionID, me.TimelineID, 0, fieldPath); handled {
				return errRes, nil
			}
			return nil, fmt.Errorf("failed to slice resource manifest for %s: %w", fieldPath, err)
		}

		var pinTime time.Time
		if hasTime {
			pinTime = me.Time.UTC()
		} else {
			var vpErrRes *mcpsdk.CallToolResult
			pinTime, vpErrRes = resolveManifestRevisionViewportPinTime(
				sliced.Revision.Time,
				sliced.NextRevisionTime,
				applied,
				spec.Filter,
				fieldPath+".revisionIndex",
			)
			if vpErrRes != nil {
				return vpErrRes, nil
			}
		}

		autoInclude.Add(me.TimelineID)
		revIdx := sliced.Revision.Index
		title := formatManifestExcerptTitle(sliced.Revision.State)
		subtitle := formatLineCountSubtitle(sliced.Slice.TotalLines)
		logTimestamp, insertID := resolveCloudLoggingMetadata(wb, sliced.Revision.LogID, sliced.Revision.Time)
		pending = append(pending, pendingExcerpt{
			kind:          ExcerptKindManifest,
			itemIndex:     i,
			fieldPath:     fieldPath,
			explicitBadge: me.Badge != "",
			card: ResolvedExcerptCard{
				Kind:          ExcerptKindManifest,
				Badge:         me.Badge,
				PaletteID:     paletteID,
				TimeMs:        pinTime.UnixMilli(),
				TimeLabel:     formatShiftedClock(pinTime, spec.TimezoneShiftHours),
				TimelineID:    me.TimelineID,
				ResourcePath:  compactResourcePath(sliced.Segments),
				Title:         title,
				Subtitle:      subtitle,
				Lines:         sliced.Slice.Lines,
				Note:          me.Note,
				LogTimestamp:  logTimestamp,
				InsertID:      insertID,
				LogID:         sliced.Revision.LogID,
				RevisionIndex: &revIdx,
			},
		})
	}

	for i := range ann.LogExcerpts {
		le := &ann.LogExcerpts[i]
		fieldPath := fmt.Sprintf("annotations.logExcerpts[%d]", i)
		if utf8.RuneCountInString(le.Note) > maxAnnotationRunes {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.note` exceeds maximum length of %d characters.", fieldPath, maxAnnotationRunes))
			return res, nil
		}
		paletteID, errRes := validateExplicitPaletteID(paletteReg, fieldPath, le.PaletteID)
		if errRes != nil {
			return errRes, nil
		}
		if le.LogID == 0 {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.logId` is required.", fieldPath))
			return res, nil
		}

		sliced, err := SliceLogDetail(wb, le.LogID, LogSliceOptions{
			Windows: le.Windows,
		})
		if err != nil {
			if errRes, handled := sliceErrorResult(err, inspectionID, 0, le.LogID, fieldPath); handled {
				return errRes, nil
			}
			return nil, fmt.Errorf("failed to slice log detail for %s: %w", fieldPath, err)
		}
		if errRes := checkViewportRange(sliced.Time, applied, spec.Filter, fieldPath+".logId"); errRes != nil {
			return errRes, nil
		}

		var tid uint32
		if le.TimelineID != 0 {
			if idx == nil || idx.TimelineMap[le.TimelineID] == nil {
				res, _, _ := timelineNotFoundResult(inspectionID, le.TimelineID)
				return res, nil
			}
			tid = le.TimelineID
		} else if idx != nil {
			tid = selectPrimaryTimelineForLog(idx, le.LogID)
		}
		if tid != 0 {
			autoInclude.Add(tid)
		}

		severityLabel := "INFO"
		if sliced.Severity != nil && sliced.Severity.GetLabel() != "" {
			severityLabel = sliced.Severity.GetLabel()
		}

		var segments []workbench.TimelineSegment
		if tid != 0 && idx != nil {
			segments, _ = idx.TimelineSegments(tid)
		}
		title := fmt.Sprintf("%s · %s", defaultString(sliced.LogType, "log"), severityLabel)
		subtitle := formatLineCountSubtitle(sliced.Slice.TotalLines)
		logTimestamp, insertID := resolveCloudLoggingMetadata(wb, le.LogID, sliced.Time)
		pending = append(pending, pendingExcerpt{
			kind:          ExcerptKindLog,
			itemIndex:     i,
			fieldPath:     fieldPath,
			explicitBadge: le.Badge != "",
			card: ResolvedExcerptCard{
				Kind:         ExcerptKindLog,
				Badge:        le.Badge,
				PaletteID:    paletteID,
				TimeMs:       sliced.Time.UnixMilli(),
				TimeLabel:    formatShiftedClock(sliced.Time, spec.TimezoneShiftHours),
				TimelineID:   tid,
				ResourcePath: compactResourcePath(segments),
				Title:        title,
				Subtitle:     subtitle,
				Lines:        sliced.Slice.Lines,
				Note:         le.Note,
				LogTimestamp: logTimestamp,
				InsertID:     insertID,
				LogID:        le.LogID,
			},
		})
	}

	slices.SortFunc(pending, func(a, b pendingExcerpt) int {
		if c := cmp.Compare(a.card.TimeMs, b.card.TimeMs); c != 0 {
			return c
		}
		if c := cmp.Compare(excerptKindPriority(a.kind), excerptKindPriority(b.kind)); c != 0 {
			return c
		}
		return cmp.Compare(a.itemIndex, b.itemIndex)
	})

	usedBadges := make(map[string]string, len(pending))
	prevSeqNum := 0
	prevSeqBadge := ""
	prevSeqField := ""
	prevSeqTimeLabel := ""
	for idx := range pending {
		item := &pending[idx]
		if !item.explicitBadge {
			continue
		}
		badge := item.card.Badge
		if prevField, dup := usedBadges[badge]; dup {
			res, _, _ := mdtemplate.ErrorResult(
				"INVALID_ARGUMENT",
				fmt.Sprintf("Duplicate excerpt badge `%s` in `%s.badge` (already used by `%s.badge`).", badge, item.fieldPath, prevField),
				"Specify unique `badge` values across `diffExcerpts`, `manifestExcerpts`, and `logExcerpts`, or omit `badge` for automatic chronological numbering (`E1`..`En`).",
			)
			return res, nil
		}
		usedBadges[badge] = item.fieldPath
		if seqNum, ok := parseExcerptSeqBadge(badge); ok {
			if prevSeqNum > 0 && seqNum < prevSeqNum {
				res, _, _ := mdtemplate.ErrorResult(
					"INVALID_ARGUMENT",
					fmt.Sprintf("Out-of-order manual excerpt badge `%s` in `%s.badge` (%s) appears chronologically after `%s` in `%s.badge` (%s).",
						badge, item.fieldPath, item.card.TimeLabel, prevSeqBadge, prevSeqField, prevSeqTimeLabel),
					"Ensure `E<n>` excerpt badges follow chronological card order, or omit `badge` for automatic chronological numbering (`E1`..`En`).",
				)
				return res, nil
			}
			prevSeqNum = seqNum
			prevSeqBadge = badge
			prevSeqField = item.fieldPath
			prevSeqTimeLabel = item.card.TimeLabel
		}
	}

	// Inherit an explicit marker badge onto a co-anchored excerpt that omitted badge.
	for idx := range pending {
		item := &pending[idx]
		if item.explicitBadge {
			continue
		}
		for _, pm := range pendingMarkers {
			if !pm.explicitBadge || !isSameMarkerExcerptAnchor(pm.marker, item.card) {
				continue
			}
			if usedBadges[pm.marker.Badge] == "" {
				item.card.Badge = pm.marker.Badge
				usedBadges[pm.marker.Badge] = item.fieldPath
				setAnnotationExcerptBadge(ann, item.kind, item.itemIndex, pm.marker.Badge)
			}
			break
		}
	}

	nextSeq := 1
	for idx := range pending {
		item := &pending[idx]
		if item.card.Badge == "" {
			for usedBadges[fmt.Sprintf("E%d", nextSeq)] != "" {
				nextSeq++
			}
			badge := fmt.Sprintf("E%d", nextSeq)
			usedBadges[badge] = item.fieldPath
			item.card.Badge = badge
			setAnnotationExcerptBadge(ann, item.kind, item.itemIndex, badge)
		}
		item.card.ID = fmt.Sprintf("excerpt-%d", idx+1)
		overlay.Excerpts = append(overlay.Excerpts, item.card)
	}
	return nil, nil
}

func excerptKindPriority(kind ExcerptKind) int {
	switch kind {
	case ExcerptKindDiff:
		return 0
	case ExcerptKindManifest:
		return 1
	case ExcerptKindLog:
		return 2
	default:
		return 3
	}
}

func parseExcerptSeqBadge(badge string) (int, bool) {
	if len(badge) < 2 || badge[0] != 'E' {
		return 0, false
	}
	for i := 1; i < len(badge); i++ {
		if badge[i] < '0' || badge[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(badge[1:])
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func setAnnotationExcerptBadge(ann *Annotations, kind ExcerptKind, itemIndex int, badge string) {
	switch kind {
	case ExcerptKindDiff:
		ann.DiffExcerpts[itemIndex].Badge = badge
	case ExcerptKindManifest:
		ann.ManifestExcerpts[itemIndex].Badge = badge
	case ExcerptKindLog:
		ann.LogExcerpts[itemIndex].Badge = badge
	}
}

func isSameMarkerExcerptAnchor(m ResolvedMarker, ex ResolvedExcerptCard) bool {
	if m.TimeMs != ex.TimeMs {
		return false
	}
	if m.TimelineID != 0 && m.RevisionIndex != nil && ex.RevisionIndex != nil {
		return m.TimelineID == ex.TimelineID && *m.RevisionIndex == *ex.RevisionIndex
	}
	if m.LogID != 0 && ex.LogID != 0 && m.LogID == ex.LogID {
		return m.TimelineID == 0 || ex.TimelineID == 0 || m.TimelineID == ex.TimelineID
	}
	if m.Style != MarkerStyleReference && m.TimelineID != 0 && m.LogID == 0 && m.RevisionIndex == nil {
		return m.TimelineID == ex.TimelineID
	}
	return false
}

func resolveManifestRevisionViewportPinTime(
	revTime time.Time,
	nextRevTime *time.Time,
	applied workbench.AppliedFilter,
	rawFilter workbench.Filter,
	fieldPath string,
) (time.Time, *mcpsdk.CallToolResult) {
	if revTime.IsZero() {
		return revTime, nil
	}
	minBound := applied.StartTime
	if !minBound.IsZero() && rawFilter.StartTime == nil {
		minBound = minBound.Add(-defaultHeaderSlack)
	}
	maxBound := applied.EndTime
	if !maxBound.IsZero() && rawFilter.EndTime == nil {
		maxBound = maxBound.Add(defaultHeaderSlack)
	}

	if !maxBound.IsZero() && revTime.After(maxBound) {
		return time.Time{}, checkViewportRange(revTime, applied, rawFilter, fieldPath)
	}
	if !minBound.IsZero() && revTime.Before(minBound) {
		if nextRevTime != nil && !nextRevTime.After(minBound) {
			return time.Time{}, checkViewportRange(revTime, applied, rawFilter, fieldPath)
		}
	}

	pinTime := revTime
	if !applied.StartTime.IsZero() && pinTime.Before(applied.StartTime) {
		pinTime = applied.StartTime
	}
	return pinTime, nil
}

func sliceErrorResult(err error, inspectionID string, timelineID, logID uint32, fieldPath string) (*mcpsdk.CallToolResult, bool) {
	if timelineID != 0 {
		if res, ok := revisionLookupErrorResult(err, inspectionID, timelineID); ok {
			return res, true
		}
	}
	if logID != 0 && errors.Is(err, workbench.ErrLogNotFound) {
		res, _, _ := mdtemplate.ErrorResult(
			"LOG_NOT_FOUND",
			fmt.Sprintf("No log with ID %d exists in inspection %s.", logID, md.Code(inspectionID)),
			"Use a log ID from `search_logs` or `get_timeline_logs`.",
		)
		return res, true
	}

	var winErr *InvalidSliceWindowError
	if errors.As(err, &winErr) {
		res, _, _ := mdtemplate.ErrorResult(
			"INVALID_ARGUMENT",
			fmt.Sprintf("Invalid `%s.windows[%d]`: %s.", fieldPath, winErr.WindowIndex, winErr.Reason),
		)
		return res, true
	}

	var notFoundErr *SliceQueryNotFoundError
	if errors.As(err, &notFoundErr) {
		bullets := []string{
			fmt.Sprintf("Query %s in `%s.windows[%d]` matched 0 lines.", md.Code(notFoundErr.Query), fieldPath, notFoundErr.WindowIndex),
		}
		if len(notFoundErr.TopLevelKeys) > 0 {
			bullets = append(bullets, fmt.Sprintf("Available top-level keys: `%s`.", strings.Join(notFoundErr.TopLevelKeys, "`, `")))
		}
		if notFoundErr.Summary != "" {
			bullets = append(bullets, fmt.Sprintf("Log summary: %s.", md.Code(notFoundErr.Summary)))
		}
		if notFoundErr.Preview != "" {
			if strings.HasPrefix(fieldPath, "annotations.manifestExcerpts") {
				bullets = append(bullets, fmt.Sprintf("Manifest preview:\n%s", md.Fence("yaml", notFoundErr.Preview)))
			} else {
				bullets = append(bullets, fmt.Sprintf("Changed hunk preview:\n%s", md.Fence("diff", notFoundErr.Preview)))
			}
		}
		bullets = append(bullets, "Inspect the full content with `get_resource_diff` / `get_resource_manifest` / `get_log` or omit `windows` to use the default excerpt.")
		res, _, _ := mdtemplate.ErrorResult("SLICE_QUERY_NOT_FOUND", bullets...)
		return res, true
	}

	return nil, false
}

func compactResourcePath(segments []workbench.TimelineSegment) string {
	if len(segments) == 0 {
		return ""
	}
	var kindName, nsName, resName string
	for _, seg := range segments {
		switch strings.ToLower(seg.Type) {
		case "kind":
			kindName = seg.Name
		case "namespace":
			nsName = seg.Name
		case "resource":
			resName = seg.Name
		}
	}
	last := segments[len(segments)-1]
	if strings.EqualFold(last.Type, "resource") && kindName != "" {
		if nsName != "" && nsName != "cluster-scope" {
			return fmt.Sprintf("%s · %s/%s", kindName, nsName, last.Name)
		}
		return fmt.Sprintf("%s · %s", kindName, last.Name)
	}
	if resName != "" && resName != last.Name {
		return fmt.Sprintf("%s/%s · %s", resName, last.Name, last.Type)
	}
	if len(segments) == 1 {
		return fmt.Sprintf("%s / %s", last.Type, last.Name)
	}
	prev := segments[len(segments)-2]
	return fmt.Sprintf("%s/%s · %s", prev.Name, last.Name, last.Type)
}

func formatRevisionExcerptTitle(verb, state string) string {
	v := strings.TrimSpace(verb)
	s := strings.TrimSpace(state)
	switch {
	case v != "" && s != "":
		return fmt.Sprintf("%s · %s", v, s)
	case v != "":
		return v
	case s != "":
		return s
	default:
		return "Resource diff"
	}
}

func formatManifestExcerptTitle(state string) string {
	s := strings.TrimSpace(state)
	if s != "" {
		return fmt.Sprintf("Manifest · %s", s)
	}
	return "Manifest"
}

func formatLineCountSubtitle(totalLines int) string {
	if totalLines == 1 {
		return "1 line"
	}
	return fmt.Sprintf("%d lines", totalLines)
}

func defaultString(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

func resolveCloudLoggingMetadata(wb *workbench.Workbench, logID uint32, fallbackTime time.Time) (logTimestamp string, insertID string) {
	if !fallbackTime.IsZero() {
		logTimestamp = fallbackTime.UTC().Format(time.RFC3339Nano)
	}
	if meta, found := wb.GetLogQueryMetadata(logID); found {
		insertID = meta.InsertID
		if !meta.Time.IsZero() {
			logTimestamp = meta.Time.UTC().Format(time.RFC3339Nano)
		}
	}
	return logTimestamp, insertID
}
