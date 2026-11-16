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
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/RoaringBitmap/roaring/v2"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

const (
	maxSceneMarkers    = 12
	maxSceneSpans      = 6
	maxSceneExcerpts   = 6
	maxSceneConnectors = 8
	maxSceneCallouts   = 8
	maxAnnotationRunes = 120
	maxRoleLabelRunes  = 40
)

var (
	validMarkerStyles    = []MarkerStyle{MarkerStylePin, MarkerStyleReference}
	validSpanPatterns    = []SpanPattern{SpanPatternSolid, SpanPatternHatched, SpanPatternOutline}
	validConnectorStyles = []ConnectorStyle{ConnectorStyleCurve, ConnectorStyleStep, ConnectorStyleCaliper}
	validCalloutSides    = []CalloutPlacement{CalloutPlacementAuto, CalloutPlacementLeft, CalloutPlacementRight, CalloutPlacementAbove, CalloutPlacementBelow}
)

// Resolve validates and resolves all annotations in spec against wb and applied, returning the frontend overlay payload.
// Auto-assigned badges for markers and excerpt cards are back-filled on spec.Annotations.
func Resolve(
	ctx context.Context,
	wb *workbench.Workbench,
	inspectionID string,
	spec *Spec,
	applied workbench.AppliedFilter,
) (*Overlay, *mcpsdk.CallToolResult, error) {
	if spec == nil || spec.Annotations == nil || isAnnotationsEmpty(spec.Annotations) {
		return nil, nil, nil
	}
	ann := spec.Annotations
	paletteReg, errRes := buildPaletteRegistry(ann.Palette)
	if errRes != nil {
		return nil, errRes, nil
	}
	if errRes := validateAnnotationCounts(ann); errRes != nil {
		return nil, errRes, nil
	}

	autoInclude := roaring.NewBitmap()
	overlay := &Overlay{
		DimUnlistedRows: ann.DimUnlistedRows != nil && *ann.DimUnlistedRows,
	}

	if errRes, err := resolveMarkers(wb, inspectionID, spec, applied, paletteReg, autoInclude, overlay); err != nil || errRes != nil {
		return nil, errRes, err
	}
	if errRes, err := resolveSpans(ctx, wb, inspectionID, spec, applied, paletteReg, autoInclude, overlay); err != nil || errRes != nil {
		return nil, errRes, err
	}
	if errRes, err := resolveRowHighlights(ctx, wb, inspectionID, spec, paletteReg, autoInclude, overlay); err != nil || errRes != nil {
		return nil, errRes, err
	}
	if errRes, err := resolveExcerpts(wb, inspectionID, spec, applied, paletteReg, autoInclude, overlay); err != nil || errRes != nil {
		return nil, errRes, err
	}
	if errRes, err := resolveConnectors(wb, inspectionID, spec, applied, paletteReg, autoInclude, overlay); err != nil || errRes != nil {
		return nil, errRes, err
	}
	if errRes, err := resolveCallouts(wb, inspectionID, spec, applied, paletteReg, autoInclude, overlay); err != nil || errRes != nil {
		return nil, errRes, err
	}

	assignInheritedPaletteIDs(overlay, paletteReg)

	if !autoInclude.IsEmpty() {
		overlay.AutoIncludeTimelineIDs = autoInclude.ToArray()
	}
	return overlay, nil, nil
}

func resolveMarkers(
	wb *workbench.Workbench,
	inspectionID string,
	spec *Spec,
	applied workbench.AppliedFilter,
	paletteReg *paletteRegistry,
	autoInclude *roaring.Bitmap,
	overlay *Overlay,
) (*mcpsdk.CallToolResult, error) {
	ann := spec.Annotations
	usedBadges := make(map[string]bool, len(ann.Markers))
	for _, m := range ann.Markers {
		if m.Badge != "" {
			usedBadges[m.Badge] = true
		}
	}
	nextSeq := 1
	for i := range ann.Markers {
		m := &ann.Markers[i]
		fieldPath := fmt.Sprintf("annotations.markers[%d]", i)
		if strings.TrimSpace(m.Label) == "" {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.label` is required.", fieldPath))
			return res, nil
		}
		if utf8.RuneCountInString(m.Label) > maxAnnotationRunes {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.label` exceeds maximum length of %d characters.", fieldPath, maxAnnotationRunes))
			return res, nil
		}
		style := m.Style
		if style == "" {
			style = MarkerStylePin
		}
		if !slices.Contains(validMarkerStyles, style) {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Unsupported `%s.style` `%s` (use `pin` or `reference`).", fieldPath, style))
			return res, nil
		}
		paletteID, errRes := validateExplicitPaletteID(paletteReg, fieldPath, m.PaletteID)
		if errRes != nil {
			return errRes, nil
		}

		pt, errRes, err := resolveAnchor(wb, inspectionID, applied, spec.Filter, fieldPath+".anchor", m.Anchor)
		if err != nil || errRes != nil {
			return errRes, err
		}
		if pt.timelineID != 0 {
			autoInclude.Add(pt.timelineID)
		}
		badge := m.Badge
		if badge == "" {
			for usedBadges[strconv.Itoa(nextSeq)] {
				nextSeq++
			}
			badge = strconv.Itoa(nextSeq)
			usedBadges[badge] = true
			m.Badge = badge
		}
		timeLabel := formatShiftedClock(pt.time, spec.TimezoneShiftHours)
		overlay.Markers = append(overlay.Markers, ResolvedMarker{
			ID:            fmt.Sprintf("marker-%d", i+1),
			Badge:         badge,
			Label:         m.Label,
			Style:         style,
			PaletteID:     paletteID,
			TimeMs:        pt.time.UnixMilli(),
			TimeLabel:     timeLabel,
			TimelineID:    pt.timelineID,
			LogID:         pt.logID,
			RevisionIndex: pt.revisionIndex,
		})
	}
	return nil, nil
}

func resolveSpans(
	ctx context.Context,
	wb *workbench.Workbench,
	inspectionID string,
	spec *Spec,
	applied workbench.AppliedFilter,
	paletteReg *paletteRegistry,
	autoInclude *roaring.Bitmap,
	overlay *Overlay,
) (*mcpsdk.CallToolResult, error) {
	ann := spec.Annotations
	for i := range ann.Spans {
		s := &ann.Spans[i]
		fieldPath := fmt.Sprintf("annotations.spans[%d]", i)
		if strings.TrimSpace(s.Label) == "" {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.label` is required.", fieldPath))
			return res, nil
		}
		if utf8.RuneCountInString(s.Label) > maxAnnotationRunes {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.label` exceeds maximum length of %d characters.", fieldPath, maxAnnotationRunes))
			return res, nil
		}
		pattern := s.Pattern
		if pattern == "" {
			pattern = SpanPatternSolid
		}
		if !slices.Contains(validSpanPatterns, pattern) {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Unsupported `%s.pattern` `%s` (use `solid`, `hatched`, or `outline`).", fieldPath, pattern))
			return res, nil
		}
		paletteID, errRes := validateExplicitPaletteID(paletteReg, fieldPath, s.PaletteID)
		if errRes != nil {
			return errRes, nil
		}

		startPt, errRes, err := resolveAnchor(wb, inspectionID, applied, spec.Filter, fieldPath+".start", s.Start)
		if err != nil || errRes != nil {
			return errRes, err
		}
		endPt, errRes, err := resolveAnchor(wb, inspectionID, applied, spec.Filter, fieldPath+".end", s.End)
		if err != nil || errRes != nil {
			return errRes, err
		}
		if endPt.time.Before(startPt.time) {
			res, _, _ := mdtemplate.ErrorResult(
				"INVALID_ANCHOR",
				fmt.Sprintf("`%s.end` (%s) must not be before `%s.start` (%s).", fieldPath, mdtemplate.FormatTime(endPt.time), fieldPath, mdtemplate.FormatTime(startPt.time)),
			)
			return res, nil
		}
		if startPt.timelineID != 0 {
			autoInclude.Add(startPt.timelineID)
		}
		if endPt.timelineID != 0 {
			autoInclude.Add(endPt.timelineID)
		}

		var scopedIDs []uint32
		if s.TimelineQuery != "" || len(s.TimelineIDs) > 0 {
			includeSubtree := s.IncludeSubtree == nil || *s.IncludeSubtree
			scopedIDs, errRes, err = resolveScopedTimelineIDs(ctx, wb, inspectionID, fieldPath, s.TimelineQuery, s.TimelineIDs, includeSubtree, autoInclude)
			if err != nil || errRes != nil {
				return errRes, err
			}
		}

		duration := endPt.time.Sub(startPt.time)
		durationLabel := formatElapsedDelta(duration, false)
		overlay.Spans = append(overlay.Spans, ResolvedSpan{
			ID:            fmt.Sprintf("span-%d", i+1),
			Label:         s.Label,
			StartTimeMs:   startPt.time.UnixMilli(),
			EndTimeMs:     endPt.time.UnixMilli(),
			DurationLabel: durationLabel,
			TimelineIDs:   scopedIDs,
			Pattern:       pattern,
			PaletteID:     paletteID,
		})
	}
	return nil, nil
}

func resolveRowHighlights(
	ctx context.Context,
	wb *workbench.Workbench,
	inspectionID string,
	spec *Spec,
	paletteReg *paletteRegistry,
	autoInclude *roaring.Bitmap,
	overlay *Overlay,
) (*mcpsdk.CallToolResult, error) {
	ann := spec.Annotations
	for i := range ann.RowHighlights {
		rh := &ann.RowHighlights[i]
		fieldPath := fmt.Sprintf("annotations.rowHighlights[%d]", i)
		if rh.TimelineQuery == "" && len(rh.TimelineIDs) == 0 {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Specify at least one of `timelineQuery` or `timelineIds` in `%s`.", fieldPath))
			return res, nil
		}
		roleLabel := strings.TrimSpace(rh.RoleLabel)
		if utf8.RuneCountInString(roleLabel) > maxRoleLabelRunes {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.roleLabel` exceeds maximum length of %d characters.", fieldPath, maxRoleLabelRunes))
			return res, nil
		}
		paletteID, errRes := validateExplicitPaletteID(paletteReg, fieldPath, rh.PaletteID)
		if errRes != nil {
			return errRes, nil
		}

		includeSubtree := rh.IncludeSubtree == nil || *rh.IncludeSubtree
		timelineIDs, errRes, err := resolveScopedTimelineIDs(ctx, wb, inspectionID, fieldPath, rh.TimelineQuery, rh.TimelineIDs, includeSubtree, autoInclude)
		if err != nil || errRes != nil {
			return errRes, err
		}

		overlay.RowHighlights = append(overlay.RowHighlights, ResolvedRowHighlight{
			ID:          fmt.Sprintf("row-highlight-%d", i+1),
			RoleLabel:   roleLabel,
			TimelineIDs: timelineIDs,
			PaletteID:   paletteID,
		})
	}
	return nil, nil
}

func resolveConnectors(
	wb *workbench.Workbench,
	inspectionID string,
	spec *Spec,
	applied workbench.AppliedFilter,
	paletteReg *paletteRegistry,
	autoInclude *roaring.Bitmap,
	overlay *Overlay,
) (*mcpsdk.CallToolResult, error) {
	ann := spec.Annotations
	for i := range ann.Connectors {
		c := &ann.Connectors[i]
		fieldPath := fmt.Sprintf("annotations.connectors[%d]", i)
		if utf8.RuneCountInString(c.Label) > maxAnnotationRunes {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.label` exceeds maximum length of %d characters.", fieldPath, maxAnnotationRunes))
			return res, nil
		}
		style := c.Style
		if style == "" {
			style = ConnectorStyleCurve
		}
		if !slices.Contains(validConnectorStyles, style) {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Unsupported `%s.style` `%s` (use `curve`, `step`, or `caliper`).", fieldPath, style))
			return res, nil
		}
		paletteID, errRes := validateExplicitPaletteID(paletteReg, fieldPath, c.PaletteID)
		if errRes != nil {
			return errRes, nil
		}

		fromPt, errRes, err := resolveAnchor(wb, inspectionID, applied, spec.Filter, fieldPath+".from", c.From)
		if err != nil || errRes != nil {
			return errRes, err
		}
		toPt, errRes, err := resolveAnchor(wb, inspectionID, applied, spec.Filter, fieldPath+".to", c.To)
		if err != nil || errRes != nil {
			return errRes, err
		}
		if fromPt.timelineID != 0 {
			autoInclude.Add(fromPt.timelineID)
		}
		if toPt.timelineID != 0 {
			autoInclude.Add(toPt.timelineID)
		}

		label := c.Label
		if label == "" {
			label = formatElapsedDelta(toPt.time.Sub(fromPt.time), style != ConnectorStyleCaliper)
		}
		overlay.Connectors = append(overlay.Connectors, ResolvedConnector{
			ID:             fmt.Sprintf("connector-%d", i+1),
			Badge:          c.Badge,
			Label:          label,
			Style:          style,
			PaletteID:      paletteID,
			FromTimeMs:     fromPt.time.UnixMilli(),
			FromTimelineID: fromPt.timelineID,
			ToTimeMs:       toPt.time.UnixMilli(),
			ToTimelineID:   toPt.timelineID,
		})
	}
	return nil, nil
}

func resolveCallouts(
	wb *workbench.Workbench,
	inspectionID string,
	spec *Spec,
	applied workbench.AppliedFilter,
	paletteReg *paletteRegistry,
	autoInclude *roaring.Bitmap,
	overlay *Overlay,
) (*mcpsdk.CallToolResult, error) {
	ann := spec.Annotations
	for i := range ann.Callouts {
		ca := &ann.Callouts[i]
		fieldPath := fmt.Sprintf("annotations.callouts[%d]", i)
		if strings.TrimSpace(ca.Label) == "" {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.label` is required.", fieldPath))
			return res, nil
		}
		if utf8.RuneCountInString(ca.Label) > maxAnnotationRunes {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.label` exceeds maximum length of %d characters.", fieldPath, maxAnnotationRunes))
			return res, nil
		}
		placement := ca.Placement
		if placement == "" {
			placement = CalloutPlacementAuto
		}
		if !slices.Contains(validCalloutSides, placement) {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Unsupported `%s.placement` `%s` (use `auto`, `left`, `right`, `above`, or `below`).", fieldPath, placement))
			return res, nil
		}
		paletteID, errRes := validateExplicitPaletteID(paletteReg, fieldPath, ca.PaletteID)
		if errRes != nil {
			return errRes, nil
		}

		pt, errRes, err := resolveAnchor(wb, inspectionID, applied, spec.Filter, fieldPath+".anchor", ca.Anchor)
		if err != nil || errRes != nil {
			return errRes, err
		}
		if pt.timelineID != 0 {
			autoInclude.Add(pt.timelineID)
		}

		overlay.Callouts = append(overlay.Callouts, ResolvedCallout{
			ID:            fmt.Sprintf("callout-%d", i+1),
			Label:         ca.Label,
			Placement:     placement,
			Dashed:        ca.Dashed,
			PaletteID:     paletteID,
			TimeMs:        pt.time.UnixMilli(),
			TimelineID:    pt.timelineID,
			LogID:         pt.logID,
			RevisionIndex: pt.revisionIndex,
		})
	}
	return nil, nil
}

func validateAnnotationCounts(ann *Annotations) *mcpsdk.CallToolResult {
	if len(ann.Markers) > maxSceneMarkers {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`annotations.markers` exceeds maximum of %d items (got %d).", maxSceneMarkers, len(ann.Markers)))
		return res
	}
	if len(ann.Spans) > maxSceneSpans {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`annotations.spans` exceeds maximum of %d items (got %d).", maxSceneSpans, len(ann.Spans)))
		return res
	}
	if totalExcerpts := len(ann.DiffExcerpts) + len(ann.LogExcerpts); totalExcerpts > maxSceneExcerpts {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Total `diffExcerpts` + `logExcerpts` exceeds maximum of %d cards (got %d).", maxSceneExcerpts, totalExcerpts))
		return res
	}
	if len(ann.Connectors) > maxSceneConnectors {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`annotations.connectors` exceeds maximum of %d items (got %d).", maxSceneConnectors, len(ann.Connectors)))
		return res
	}
	if len(ann.Callouts) > maxSceneCallouts {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`annotations.callouts` exceeds maximum of %d items (got %d).", maxSceneCallouts, len(ann.Callouts)))
		return res
	}
	return nil
}
