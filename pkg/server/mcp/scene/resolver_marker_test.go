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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResolveMarkerAnchors_Errors(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()

	testCases := []struct {
		name       string
		spec       Spec
		wantCode   string
		wantSubstr string
	}{
		{
			name: "blank marker label returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{{Label: "   ", Anchor: Anchor{LogID: fx.oomEventLogID}}},
				},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "`annotations.markers[0].label` is required.",
		},
		{
			name: "marker label exceeding maxAnnotationRunes returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{{Label: strings.Repeat("a", maxAnnotationRunes+1), Anchor: Anchor{LogID: fx.oomEventLogID}}},
				},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "`annotations.markers[0].label` exceeds maximum length of 120 characters.",
		},
		{
			name: "unsupported marker style returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{{Label: "ok", Style: "dashed", Anchor: Anchor{LogID: fx.oomEventLogID}}},
				},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "Unsupported `annotations.markers[0].style` `dashed`",
		},
		{
			name: "unknown marker paletteId returns UNKNOWN_PALETTE_ID",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{{Label: "bad palette", PaletteID: "missing", Anchor: Anchor{LogID: fx.oomEventLogID}}},
				},
			},
			wantCode: "UNKNOWN_PALETTE_ID",
		},
		{
			name: "anchor with neither logId nor revision nor time returns INVALID_ANCHOR",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{{Label: "bad"}},
				},
			},
			wantCode:   "INVALID_ANCHOR",
			wantSubstr: "`annotations.markers[0].anchor`",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertResolveError(t, ctx, fx, tc.spec, tc.wantCode, tc.wantSubstr)
		})
	}
}

func TestFinalizeMarkers(t *testing.T) {
	type finalizeOutcome struct {
		OverlayMarkers []ResolvedMarker
		AnnBadges      []string
	}

	ann := &Annotations{
		Markers: []MarkerAnnotation{
			{Badge: "1", Label: "Explicit numeric"},
			{Badge: "PIN", Label: "Explicit co-anchored"},
			{Label: "Inherit 3"},
			{Label: "Tied timestamp A"},
			{Label: "Tied timestamp B"},
		},
	}
	pending := []pendingMarker{
		{itemIndex: 0, explicitBadge: true, marker: ResolvedMarker{Badge: "1", Label: "Explicit numeric", TimeMs: 100}},
		{itemIndex: 1, explicitBadge: true, marker: ResolvedMarker{Badge: "PIN", Label: "Explicit co-anchored", TimeMs: 200, TimelineID: 10, LogID: 1}},
		{itemIndex: 2, explicitBadge: false, marker: ResolvedMarker{Label: "Inherit 3", TimeMs: 300, TimelineID: 20, LogID: 2}},
		{itemIndex: 3, explicitBadge: false, marker: ResolvedMarker{Label: "Tied timestamp A", TimeMs: 400}},
		{itemIndex: 4, explicitBadge: false, marker: ResolvedMarker{Label: "Tied timestamp B", TimeMs: 400}},
	}
	overlay := &Overlay{
		Excerpts: []ResolvedExcerptCard{
			{Badge: "E1", TimeMs: 200, TimelineID: 10, LogID: 1},
			{Badge: "3", TimeMs: 300, TimelineID: 20, LogID: 2},
		},
	}

	finalizeMarkers(ann, pending, overlay)

	got := finalizeOutcome{
		OverlayMarkers: overlay.Markers,
		AnnBadges:      make([]string, len(ann.Markers)),
	}
	for i, m := range ann.Markers {
		got.AnnBadges[i] = m.Badge
	}
	want := finalizeOutcome{
		OverlayMarkers: []ResolvedMarker{
			{ID: "marker-1", Badge: "1", Label: "Explicit numeric", TimeMs: 100},
			{ID: "marker-2", Badge: "PIN", Label: "Explicit co-anchored", TimeMs: 200, TimelineID: 10, LogID: 1},
			{ID: "marker-3", Badge: "3", Label: "Inherit 3", TimeMs: 300, TimelineID: 20, LogID: 2},
			{ID: "marker-4", Badge: "2", Label: "Tied timestamp A", TimeMs: 400},
			{ID: "marker-5", Badge: "4", Label: "Tied timestamp B", TimeMs: 400},
		},
		AnnBadges: []string{"1", "PIN", "3", "2", "4"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("finalizeMarkers() mismatch (-want +got):\n%s", diff)
	}
}
