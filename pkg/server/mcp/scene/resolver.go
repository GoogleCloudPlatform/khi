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
	validConnectorStyles = []ConnectorStyle{ConnectorStyleCurve, ConnectorStyleStep, ConnectorStyleCaliper}
	validCalloutSides    = []CalloutPlacement{CalloutPlacementAuto, CalloutPlacementLeft, CalloutPlacementRight, CalloutPlacementAbove, CalloutPlacementBelow}
)

type sceneContext struct {
	wb           *workbench.Workbench
	inspectionID string
	spec         *Spec
	applied      workbench.AppliedFilter
	paletteReg   *paletteRegistry
	autoInclude  *roaring.Bitmap
	overlay      *Overlay
}

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
	rc := &sceneContext{
		wb:           wb,
		inspectionID: inspectionID,
		spec:         spec,
		applied:      applied,
		paletteReg:   paletteReg,
		autoInclude:  autoInclude,
		overlay:      overlay,
	}

	pendingMarkers, errRes, err := rc.resolveMarkerAnchors()
	if err != nil || errRes != nil {
		return nil, errRes, err
	}
	if errRes, err := rc.resolveSpans(ctx); err != nil || errRes != nil {
		return nil, errRes, err
	}
	if errRes, err := resolveRowHighlights(ctx, wb, inspectionID, spec, paletteReg, autoInclude, overlay); err != nil || errRes != nil {
		return nil, errRes, err
	}
	if errRes, err := rc.resolveExcerpts(pendingMarkers); err != nil || errRes != nil {
		return nil, errRes, err
	}
	finalizeMarkers(ann, pendingMarkers, overlay)
	if errRes, err := resolveConnectors(rc.wb, rc.inspectionID, rc.spec, rc.applied, rc.paletteReg, rc.autoInclude, rc.overlay); err != nil || errRes != nil {
		return nil, errRes, err
	}
	if errRes, err := resolveCallouts(rc.wb, rc.inspectionID, rc.spec, rc.applied, rc.paletteReg, rc.autoInclude, rc.overlay); err != nil || errRes != nil {
		return nil, errRes, err
	}

	assignInheritedPaletteIDs(overlay, paletteReg)

	if !autoInclude.IsEmpty() {
		overlay.AutoIncludeTimelineIDs = autoInclude.ToArray()
	}
	return overlay, nil, nil
}

func validateOptionalAnnotationText(text, fieldPath, fieldName string, maxRunes int) *mcpsdk.CallToolResult {
	if utf8.RuneCountInString(text) > maxRunes {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.%s` exceeds maximum length of %d characters.", fieldPath, fieldName, maxRunes))
		return res
	}
	return nil
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
	if totalExcerpts := len(ann.DiffExcerpts) + len(ann.ManifestExcerpts) + len(ann.LogExcerpts); totalExcerpts > maxSceneExcerpts {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Total `diffExcerpts` + `manifestExcerpts` + `logExcerpts` exceeds maximum of %d cards (got %d).", maxSceneExcerpts, totalExcerpts))
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
