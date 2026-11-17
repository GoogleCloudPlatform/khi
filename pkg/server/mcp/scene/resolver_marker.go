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
	"fmt"
	"slices"
	"strconv"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
)

var validMarkerStyles = []MarkerStyle{MarkerStylePin, MarkerStyleReference}

type pendingMarker struct {
	itemIndex     int
	explicitBadge bool
	marker        ResolvedMarker
}

func (rc *sceneContext) resolveMarkerAnchors() ([]pendingMarker, *mcpsdk.CallToolResult, error) {
	markers := rc.spec.Annotations.Markers
	pending := make([]pendingMarker, 0, len(markers))
	for i := range markers {
		pm, errRes, err := rc.resolveMarker(i, markers[i])
		if err != nil || errRes != nil {
			return nil, errRes, err
		}
		pending = append(pending, pm)
	}
	return pending, nil, nil
}

func (rc *sceneContext) resolveMarker(index int, m MarkerAnnotation) (pendingMarker, *mcpsdk.CallToolResult, error) {
	fieldPath := fmt.Sprintf("annotations.markers[%d]", index)
	if strings.TrimSpace(m.Label) == "" {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.label` is required.", fieldPath))
		return pendingMarker{}, res, nil
	}
	if errRes := validateOptionalAnnotationText(m.Label, fieldPath, "label", maxAnnotationRunes); errRes != nil {
		return pendingMarker{}, errRes, nil
	}
	style := m.Style
	if style == "" {
		style = MarkerStylePin
	}
	if !slices.Contains(validMarkerStyles, style) {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Unsupported `%s.style` `%s` (use `pin` or `reference`).", fieldPath, style))
		return pendingMarker{}, res, nil
	}
	paletteID, errRes := validateExplicitPaletteID(rc.paletteReg, fieldPath, m.PaletteID)
	if errRes != nil {
		return pendingMarker{}, errRes, nil
	}
	pt, errRes, err := resolveAnchor(rc.wb, rc.inspectionID, rc.applied, rc.spec.Filter, fieldPath+".anchor", m.Anchor)
	if err != nil || errRes != nil {
		return pendingMarker{}, errRes, err
	}
	if pt.timelineID != 0 {
		rc.autoInclude.Add(pt.timelineID)
	}
	return pendingMarker{
		itemIndex:     index,
		explicitBadge: m.Badge != "",
		marker: ResolvedMarker{
			Badge:         m.Badge,
			Label:         m.Label,
			Style:         style,
			PaletteID:     paletteID,
			TimeMs:        pt.time.UnixMilli(),
			TimeLabel:     formatShiftedClock(pt.time, rc.spec.TimezoneShiftHours),
			TimelineID:    pt.timelineID,
			LogID:         pt.logID,
			RevisionIndex: pt.revisionIndex,
		},
	}, nil, nil
}

func finalizeMarkers(ann *Annotations, pending []pendingMarker, overlay *Overlay) {
	if len(pending) == 0 {
		return
	}
	usedBadges := syncCoAnchoredMarkerBadges(ann, pending, overlay.Excerpts)

	// Sort markers chronologically so auto-assigned 1..N badges read left-to-right across the top band.
	slices.SortFunc(pending, func(a, b pendingMarker) int {
		if c := cmp.Compare(a.marker.TimeMs, b.marker.TimeMs); c != 0 {
			return c
		}
		return cmp.Compare(a.itemIndex, b.itemIndex)
	})

	assignAutoMarkerBadges(ann, pending, overlay, usedBadges)
}

func syncCoAnchoredMarkerBadges(ann *Annotations, pending []pendingMarker, excerpts []ResolvedExcerptCard) map[string]bool {
	usedBadges := make(map[string]bool, len(pending)+len(excerpts))
	for i := range pending {
		if pending[i].explicitBadge {
			usedBadges[pending[i].marker.Badge] = true
			continue
		}
		// Inherit co-anchored excerpt badges onto markers that omitted badge so top and bottom pins match.
		for _, ex := range excerpts {
			if isSameMarkerExcerptAnchor(pending[i].marker, ex) {
				pending[i].marker.Badge = ex.Badge
				ann.Markers[pending[i].itemIndex].Badge = ex.Badge
				usedBadges[ex.Badge] = true
				break
			}
		}
	}
	return usedBadges
}

func assignAutoMarkerBadges(ann *Annotations, pending []pendingMarker, overlay *Overlay, usedBadges map[string]bool) {
	nextSeq := 1
	for idx := range pending {
		pm := &pending[idx]
		if pm.marker.Badge == "" {
			for usedBadges[strconv.Itoa(nextSeq)] {
				nextSeq++
			}
			badge := strconv.Itoa(nextSeq)
			usedBadges[badge] = true
			pm.marker.Badge = badge
			ann.Markers[pm.itemIndex].Badge = badge
		}
		pm.marker.ID = fmt.Sprintf("marker-%d", idx+1)
		overlay.Markers = append(overlay.Markers, pm.marker)
	}
}
