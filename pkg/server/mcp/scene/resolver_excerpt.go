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
	isLog     bool
	itemIndex int
	card      ResolvedExcerptCard
}

func resolveExcerpts(
	wb *workbench.Workbench,
	inspectionID string,
	spec *Spec,
	applied workbench.AppliedFilter,
	paletteReg *paletteRegistry,
	autoInclude *roaring.Bitmap,
	overlay *Overlay,
) (*mcpsdk.CallToolResult, error) {
	ann := spec.Annotations
	var pending []pendingExcerpt
	usedBadges := make(map[string]bool, len(ann.DiffExcerpts)+len(ann.LogExcerpts))
	for _, de := range ann.DiffExcerpts {
		if de.Badge != "" {
			usedBadges[de.Badge] = true
		}
	}
	for _, le := range ann.LogExcerpts {
		if le.Badge != "" {
			usedBadges[le.Badge] = true
		}
	}

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
			isLog:     false,
			itemIndex: i,
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
			if idx := wb.SearchIndex(); idx == nil || idx.TimelineMap[le.TimelineID] == nil {
				res, _, _ := timelineNotFoundResult(inspectionID, le.TimelineID)
				return res, nil
			}
			tid = le.TimelineID
		} else {
			tid = selectPrimaryTimelineForLog(wb.SearchIndex(), le.LogID)
		}
		if tid != 0 {
			autoInclude.Add(tid)
		}

		severityLabel := "INFO"
		if sliced.Severity != nil && sliced.Severity.GetLabel() != "" {
			severityLabel = sliced.Severity.GetLabel()
		}

		var segments []workbench.TimelineSegment
		if tid != 0 && wb.SearchIndex() != nil {
			segments, _ = wb.SearchIndex().TimelineSegments(tid)
		}
		title := fmt.Sprintf("%s · %s", defaultString(sliced.LogType, "log"), severityLabel)
		subtitle := formatLineCountSubtitle(sliced.Slice.TotalLines)
		logTimestamp, insertID := resolveCloudLoggingMetadata(wb, le.LogID, sliced.Time)
		pending = append(pending, pendingExcerpt{
			isLog:     true,
			itemIndex: i,
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
		if a.isLog != b.isLog {
			if !a.isLog {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.itemIndex, b.itemIndex)
	})

	nextSeq := 1
	for idx := range pending {
		item := &pending[idx]
		badge := item.card.Badge
		if badge == "" {
			for usedBadges[fmt.Sprintf("E%d", nextSeq)] {
				nextSeq++
			}
			badge = fmt.Sprintf("E%d", nextSeq)
			usedBadges[badge] = true
			item.card.Badge = badge
			if item.isLog {
				ann.LogExcerpts[item.itemIndex].Badge = badge
			} else {
				ann.DiffExcerpts[item.itemIndex].Badge = badge
			}
		}
		item.card.ID = fmt.Sprintf("excerpt-%d", idx+1)
		overlay.Excerpts = append(overlay.Excerpts, item.card)
	}
	return nil, nil
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
			bullets = append(bullets, fmt.Sprintf("Changed hunk preview:\n%s", md.Fence("diff", notFoundErr.Preview)))
		}
		bullets = append(bullets, "Inspect the full content with `get_resource_diff` / `get_log` or omit `windows` to use the default excerpt.")
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
