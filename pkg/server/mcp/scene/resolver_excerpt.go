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
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

type pendingExcerpt struct {
	kind          ExcerptKind
	itemIndex     int
	fieldPath     string
	explicitBadge bool
	card          ResolvedExcerptCard
}

func (rc *sceneContext) resolveExcerpts(pendingMarkers []pendingMarker) (*mcpsdk.CallToolResult, error) {
	ann := rc.spec.Annotations
	var pending []pendingExcerpt

	for i, de := range ann.DiffExcerpts {
		item, errRes, err := rc.resolveDiffExcerpt(i, de)
		if err != nil || errRes != nil {
			return errRes, err
		}
		pending = append(pending, item)
	}
	for i, me := range ann.ManifestExcerpts {
		item, errRes, err := rc.resolveManifestExcerpt(i, me)
		if err != nil || errRes != nil {
			return errRes, err
		}
		pending = append(pending, item)
	}
	for i, le := range ann.LogExcerpts {
		item, errRes, err := rc.resolveLogExcerpt(i, le)
		if err != nil || errRes != nil {
			return errRes, err
		}
		pending = append(pending, item)
	}

	sortPendingExcerpts(pending)
	if errRes := validateAndSyncExcerptBadges(ann, pending, pendingMarkers, rc.overlay); errRes != nil {
		return errRes, nil
	}
	return nil, nil
}

func (rc *sceneContext) resolveDiffExcerpt(index int, de DiffExcerpt) (pendingExcerpt, *mcpsdk.CallToolResult, error) {
	fieldPath := fmt.Sprintf("annotations.diffExcerpts[%d]", index)
	if errRes := validateOptionalAnnotationText(de.Note, fieldPath, "note", maxAnnotationRunes); errRes != nil {
		return pendingExcerpt{}, errRes, nil
	}
	paletteID, errRes := validateExplicitPaletteID(rc.paletteReg, fieldPath, de.PaletteID)
	if errRes != nil {
		return pendingExcerpt{}, errRes, nil
	}
	if de.Revision.TimelineID == 0 {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.revision.timelineId` is required.", fieldPath))
		return pendingExcerpt{}, res, nil
	}
	if len(de.Paths) == 0 {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.paths` is required and must contain at least one JSONPath selector.", fieldPath))
		return pendingExcerpt{}, res, nil
	}

	sliced, err := SliceResourceDiff(rc.wb, de.Revision.TimelineID, de.Revision.RevisionIndex, DiffSliceOptions{
		Paths: de.Paths,
	})
	if err != nil {
		if errRes, handled := sliceErrorResult(err, rc.inspectionID, de.Revision.TimelineID, 0, fieldPath); handled {
			return pendingExcerpt{}, errRes, nil
		}
		return pendingExcerpt{}, nil, fmt.Errorf("failed to slice resource diff for %s: %w", fieldPath, err)
	}
	if errRes := checkViewportRange(sliced.Revision.Time, rc.applied, rc.spec.Filter, fieldPath+".revision"); errRes != nil {
		return pendingExcerpt{}, errRes, nil
	}

	rc.autoInclude.Add(de.Revision.TimelineID)
	revIdx := de.Revision.RevisionIndex
	subtitle := fmt.Sprintf("+%d / -%d · %d diff lines", sliced.Slice.Changes.Added, sliced.Slice.Changes.Deleted, sliced.Slice.TotalDiffLines)
	logTimestamp, insertID := resolveCloudLoggingMetadata(rc.wb, sliced.Revision.LogID, sliced.Revision.Time)
	return pendingExcerpt{
		kind:          ExcerptKindDiff,
		itemIndex:     index,
		fieldPath:     fieldPath,
		explicitBadge: de.Badge != "",
		card: ResolvedExcerptCard{
			Kind:          ExcerptKindDiff,
			Badge:         de.Badge,
			PaletteID:     paletteID,
			TimeMs:        sliced.Revision.Time.UnixMilli(),
			TimeLabel:     formatShiftedClock(sliced.Revision.Time, rc.spec.TimezoneShiftHours),
			TimelineID:    de.Revision.TimelineID,
			ResourcePath:  compactResourcePath(sliced.Segments),
			Title:         formatRevisionExcerptTitle(sliced.Revision.Verb, sliced.Revision.State),
			Subtitle:      subtitle,
			Lines:         sliced.Slice.Lines,
			Note:          de.Note,
			LogTimestamp:  logTimestamp,
			InsertID:      insertID,
			LogID:         sliced.Revision.LogID,
			RevisionIndex: &revIdx,
		},
	}, nil, nil
}

func (rc *sceneContext) resolveManifestExcerpt(index int, me ManifestExcerpt) (pendingExcerpt, *mcpsdk.CallToolResult, error) {
	fieldPath := fmt.Sprintf("annotations.manifestExcerpts[%d]", index)
	paletteID, selector, errRes := rc.validateManifestExcerpt(me, fieldPath)
	if errRes != nil {
		return pendingExcerpt{}, errRes, nil
	}

	sliced, err := SliceResourceManifest(rc.wb, me.TimelineID, selector, ManifestSliceOptions{
		Paths: me.Paths,
	})
	if err != nil {
		if errRes, handled := sliceErrorResult(err, rc.inspectionID, me.TimelineID, 0, fieldPath); handled {
			return pendingExcerpt{}, errRes, nil
		}
		return pendingExcerpt{}, nil, fmt.Errorf("failed to slice resource manifest for %s: %w", fieldPath, err)
	}

	pinTime, errRes := rc.resolveManifestPinTime(me, sliced, fieldPath)
	if errRes != nil {
		return pendingExcerpt{}, errRes, nil
	}

	rc.autoInclude.Add(me.TimelineID)
	revIdx := sliced.Revision.Index
	logTimestamp, insertID := resolveCloudLoggingMetadata(rc.wb, sliced.Revision.LogID, sliced.Revision.Time)
	return pendingExcerpt{
		kind:          ExcerptKindManifest,
		itemIndex:     index,
		fieldPath:     fieldPath,
		explicitBadge: me.Badge != "",
		card: ResolvedExcerptCard{
			Kind:          ExcerptKindManifest,
			Badge:         me.Badge,
			PaletteID:     paletteID,
			TimeMs:        pinTime.UnixMilli(),
			TimeLabel:     formatShiftedClock(pinTime, rc.spec.TimezoneShiftHours),
			TimelineID:    me.TimelineID,
			ResourcePath:  compactResourcePath(sliced.Segments),
			Title:         formatManifestExcerptTitle(sliced.Revision.State),
			Subtitle:      formatLineCountSubtitle(sliced.Slice.TotalLines),
			Lines:         sliced.Slice.Lines,
			Note:          me.Note,
			LogTimestamp:  logTimestamp,
			InsertID:      insertID,
			LogID:         sliced.Revision.LogID,
			RevisionIndex: &revIdx,
		},
	}, nil, nil
}

func (rc *sceneContext) validateManifestExcerpt(me ManifestExcerpt, fieldPath string) (string, workbench.RevisionSelector, *mcpsdk.CallToolResult) {
	if errRes := validateOptionalAnnotationText(me.Note, fieldPath, "note", maxAnnotationRunes); errRes != nil {
		return "", workbench.RevisionSelector{}, errRes
	}
	paletteID, errRes := validateExplicitPaletteID(rc.paletteReg, fieldPath, me.PaletteID)
	if errRes != nil {
		return "", workbench.RevisionSelector{}, errRes
	}
	if me.TimelineID == 0 {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.timelineId` is required.", fieldPath))
		return "", workbench.RevisionSelector{}, res
	}
	if len(me.Paths) == 0 {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.paths` is required and must contain at least one JSONPath selector.", fieldPath))
		return "", workbench.RevisionSelector{}, res
	}
	selector, errRes := rc.buildManifestSelector(me, fieldPath)
	if errRes != nil {
		return "", workbench.RevisionSelector{}, errRes
	}
	return paletteID, selector, nil
}

func (rc *sceneContext) buildManifestSelector(me ManifestExcerpt, fieldPath string) (workbench.RevisionSelector, *mcpsdk.CallToolResult) {
	hasRev := me.RevisionIndex != nil
	hasTime := me.Time != nil && !me.Time.IsZero()
	if (hasRev && me.Time != nil) || (!hasRev && !hasTime) {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Specify exactly one of `revisionIndex` or `time` in `%s`.", fieldPath))
		return workbench.RevisionSelector{}, res
	}
	if !hasTime {
		return workbench.RevisionAtIndex(*me.RevisionIndex), nil
	}
	idx := rc.wb.SearchIndex()
	if idx.TimelineMap[me.TimelineID] == nil {
		res, _, _ := timelineNotFoundResult(rc.inspectionID, me.TimelineID)
		return workbench.RevisionSelector{}, res
	}
	queryTime := me.Time.UTC()
	if errRes := checkViewportRange(queryTime, rc.applied, rc.spec.Filter, fieldPath+".time"); errRes != nil {
		return workbench.RevisionSelector{}, errRes
	}
	return workbench.RevisionAtTime(queryTime), nil
}

func (rc *sceneContext) resolveManifestPinTime(
	me ManifestExcerpt,
	sliced *ResourceManifestSlice,
	fieldPath string,
) (time.Time, *mcpsdk.CallToolResult) {
	if me.Time != nil {
		return me.Time.UTC(), nil
	}
	return resolveManifestRevisionViewportPinTime(
		sliced.Revision.Time,
		sliced.NextRevisionTime,
		rc.applied,
		rc.spec.Filter,
		fieldPath+".revisionIndex",
	)
}

func (rc *sceneContext) resolveLogExcerpt(index int, le LogExcerpt) (pendingExcerpt, *mcpsdk.CallToolResult, error) {
	fieldPath := fmt.Sprintf("annotations.logExcerpts[%d]", index)
	if errRes := validateOptionalAnnotationText(le.Note, fieldPath, "note", maxAnnotationRunes); errRes != nil {
		return pendingExcerpt{}, errRes, nil
	}
	paletteID, errRes := validateExplicitPaletteID(rc.paletteReg, fieldPath, le.PaletteID)
	if errRes != nil {
		return pendingExcerpt{}, errRes, nil
	}
	if le.LogID == 0 {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.logId` is required.", fieldPath))
		return pendingExcerpt{}, res, nil
	}

	sliced, err := SliceLogDetail(rc.wb, le.LogID, LogSliceOptions{
		Windows: le.Windows,
	})
	if err != nil {
		if errRes, handled := sliceErrorResult(err, rc.inspectionID, 0, le.LogID, fieldPath); handled {
			return pendingExcerpt{}, errRes, nil
		}
		return pendingExcerpt{}, nil, fmt.Errorf("failed to slice log detail for %s: %w", fieldPath, err)
	}
	if errRes := checkViewportRange(sliced.Time, rc.applied, rc.spec.Filter, fieldPath+".logId"); errRes != nil {
		return pendingExcerpt{}, errRes, nil
	}

	idx := rc.wb.SearchIndex()
	tid, errRes := rc.resolveLogExcerptTimeline(idx, le)
	if errRes != nil {
		return pendingExcerpt{}, errRes, nil
	}

	var segments []workbench.TimelineSegment
	if tid != 0 {
		rc.autoInclude.Add(tid)
		segments, _ = idx.TimelineSegments(tid)
	}
	logTimestamp, insertID := resolveCloudLoggingMetadata(rc.wb, le.LogID, sliced.Time)
	return pendingExcerpt{
		kind:          ExcerptKindLog,
		itemIndex:     index,
		fieldPath:     fieldPath,
		explicitBadge: le.Badge != "",
		card: ResolvedExcerptCard{
			Kind:         ExcerptKindLog,
			Badge:        le.Badge,
			PaletteID:    paletteID,
			TimeMs:       sliced.Time.UnixMilli(),
			TimeLabel:    formatShiftedClock(sliced.Time, rc.spec.TimezoneShiftHours),
			TimelineID:   tid,
			ResourcePath: compactResourcePath(segments),
			Title:        formatLogExcerptTitle(sliced.LogType, sliced.Severity),
			Subtitle:     formatLineCountSubtitle(sliced.Slice.TotalLines),
			Lines:        sliced.Slice.Lines,
			Note:         le.Note,
			LogTimestamp: logTimestamp,
			InsertID:     insertID,
			LogID:        le.LogID,
		},
	}, nil, nil
}

func (rc *sceneContext) resolveLogExcerptTimeline(idx *workbench.SearchIndex, le LogExcerpt) (uint32, *mcpsdk.CallToolResult) {
	if le.TimelineID != 0 {
		if idx.TimelineMap[le.TimelineID] == nil {
			res, _, _ := timelineNotFoundResult(rc.inspectionID, le.TimelineID)
			return 0, res
		}
		return le.TimelineID, nil
	}
	return selectPrimaryTimelineForLog(idx, le.LogID), nil
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
	minBound, maxBound := excerptViewportBounds(applied, rawFilter)
	if !maxBound.IsZero() && revTime.After(maxBound) {
		return time.Time{}, checkViewportRange(revTime, applied, rawFilter, fieldPath)
	}
	if !minBound.IsZero() && revTime.Before(minBound) && nextRevTime != nil && !nextRevTime.After(minBound) {
		return time.Time{}, checkViewportRange(revTime, applied, rawFilter, fieldPath)
	}

	pinTime := revTime
	if !applied.StartTime.IsZero() && pinTime.Before(applied.StartTime) {
		pinTime = applied.StartTime
	}
	return pinTime, nil
}

func excerptViewportBounds(applied workbench.AppliedFilter, rawFilter workbench.Filter) (time.Time, time.Time) {
	minBound := applied.StartTime
	if !minBound.IsZero() && rawFilter.StartTime == nil {
		minBound = minBound.Add(-defaultHeaderSlack)
	}
	maxBound := applied.EndTime
	if !maxBound.IsZero() && rawFilter.EndTime == nil {
		maxBound = maxBound.Add(defaultHeaderSlack)
	}
	return minBound, maxBound
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

func formatLogExcerptTitle(logType string, severity *khifilev6.Severity) string {
	label := severity.GetLabel()
	if label == "" {
		label = "INFO"
	}
	return fmt.Sprintf("%s · %s", defaultString(logType, "log"), label)
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
