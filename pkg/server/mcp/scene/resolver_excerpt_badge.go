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

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
)

var excerptKindOrder = map[ExcerptKind]int{
	ExcerptKindDiff:     0,
	ExcerptKindManifest: 1,
	ExcerptKindLog:      2,
}

func sortPendingExcerpts(pending []pendingExcerpt) {
	slices.SortFunc(pending, func(a, b pendingExcerpt) int {
		if c := cmp.Compare(a.card.TimeMs, b.card.TimeMs); c != 0 {
			return c
		}
		if c := cmp.Compare(excerptKindOrder[a.kind], excerptKindOrder[b.kind]); c != 0 {
			return c
		}
		return cmp.Compare(a.itemIndex, b.itemIndex)
	})
}

func validateAndSyncExcerptBadges(
	ann *Annotations,
	pending []pendingExcerpt,
	pendingMarkers []pendingMarker,
	overlay *Overlay,
) *mcpsdk.CallToolResult {
	usedBadges, errRes := validateExplicitExcerptBadges(pending)
	if errRes != nil {
		return errRes
	}
	inheritExplicitMarkerBadges(ann, pending, pendingMarkers, usedBadges)
	assignAutoExcerptBadges(ann, pending, usedBadges, overlay)
	return nil
}

func validateExplicitExcerptBadges(pending []pendingExcerpt) (map[string]string, *mcpsdk.CallToolResult) {
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
			return nil, res
		}
		usedBadges[badge] = item.fieldPath
		if seqNum, ok := parseExcerptSeqBadge(badge); ok {
			if seqNum < prevSeqNum {
				res, _, _ := mdtemplate.ErrorResult(
					"INVALID_ARGUMENT",
					fmt.Sprintf("Out-of-order manual excerpt badge `%s` in `%s.badge` (%s) appears chronologically after `%s` in `%s.badge` (%s).",
						badge, item.fieldPath, item.card.TimeLabel, prevSeqBadge, prevSeqField, prevSeqTimeLabel),
					"Ensure `E<n>` excerpt badges follow chronological card order, or omit `badge` for automatic chronological numbering (`E1`..`En`).",
				)
				return nil, res
			}
			prevSeqNum = seqNum
			prevSeqBadge = badge
			prevSeqField = item.fieldPath
			prevSeqTimeLabel = item.card.TimeLabel
		}
	}
	return usedBadges, nil
}

func inheritExplicitMarkerBadges(
	ann *Annotations,
	pending []pendingExcerpt,
	pendingMarkers []pendingMarker,
	usedBadges map[string]string,
) {
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
}

func assignAutoExcerptBadges(
	ann *Annotations,
	pending []pendingExcerpt,
	usedBadges map[string]string,
	overlay *Overlay,
) {
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
	if m.LogID != 0 && m.LogID == ex.LogID {
		return m.TimelineID == 0 || ex.TimelineID == 0 || m.TimelineID == ex.TimelineID
	}
	if m.Style != MarkerStyleReference && m.TimelineID != 0 && m.LogID == 0 && m.RevisionIndex == nil {
		return m.TimelineID == ex.TimelineID
	}
	return false
}
