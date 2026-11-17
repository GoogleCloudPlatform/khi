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
	"time"

	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestResolve_SpanLabelPlacementAndHideDuration(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()

	beforeStart := fx.baseTime.Add(15 * time.Minute)
	beforeEnd := fx.baseTime.Add(23 * time.Minute)
	afterStart := fx.baseTime.Add(24 * time.Minute)
	afterEnd := fx.baseTime.Add(34 * time.Minute)
	revOne := 1
	maxLabel := strings.Repeat("a", maxAnnotationRunes)

	type spanResolveOutcome struct {
		AutoIncludeTimelineIDs []uint32
		Spans                  []ResolvedSpan
	}

	testCases := []struct {
		name  string
		spans []SpanAnnotation
		want  spanResolveOutcome
	}{
		{
			name: "propagates explicit above and inside placements with default solid pattern and hideDuration",
			spans: []SpanAnnotation{
				{
					Label:          "averageUtilization: 20 ~ 27%",
					Start:          Anchor{Time: &beforeStart},
					End:            Anchor{Time: &beforeEnd},
					TimelineIDs:    []uint32{fx.podTimelineID},
					IncludeSubtree: boolRef(false),
					Pattern:        SpanPatternOutline,
					LabelPlacement: SpanLabelPlacementAbove,
					HideDuration:   true,
				},
				{
					Label:          "averageUtilization: 1 ~ 2%",
					Start:          Anchor{Time: &afterStart},
					End:            Anchor{Time: &afterEnd},
					TimelineIDs:    []uint32{fx.podTimelineID},
					IncludeSubtree: boolRef(false),
					LabelPlacement: SpanLabelPlacementInside,
					HideDuration:   false,
				},
			},
			want: spanResolveOutcome{
				AutoIncludeTimelineIDs: []uint32{fx.podTimelineID},
				Spans: []ResolvedSpan{
					{
						ID:             "span-1",
						Label:          "averageUtilization: 20 ~ 27%",
						StartTimeMs:    beforeStart.UnixMilli(),
						EndTimeMs:      beforeEnd.UnixMilli(),
						DurationLabel:  "",
						TimelineIDs:    []uint32{fx.podTimelineID},
						Pattern:        SpanPatternOutline,
						LabelPlacement: SpanLabelPlacementAbove,
						HideDuration:   true,
						PaletteID:      "default",
					},
					{
						ID:             "span-2",
						Label:          "averageUtilization: 1 ~ 2%",
						StartTimeMs:    afterStart.UnixMilli(),
						EndTimeMs:      afterEnd.UnixMilli(),
						DurationLabel:  "10m",
						TimelineIDs:    []uint32{fx.podTimelineID},
						Pattern:        SpanPatternSolid,
						LabelPlacement: SpanLabelPlacementInside,
						HideDuration:   false,
						PaletteID:      "default",
					},
				},
			},
		},
		{
			name: "auto-includes distinct start and end timelines and accepts max-length label",
			spans: []SpanAnnotation{
				{
					Label:        maxLabel,
					Start:        Anchor{TimelineID: fx.nodeTimelineID, RevisionIndex: &revOne},
					End:          Anchor{LogID: fx.oomEventLogID},
					Pattern:      SpanPatternSolid,
					HideDuration: true,
				},
			},
			want: spanResolveOutcome{
				AutoIncludeTimelineIDs: []uint32{fx.nodeTimelineID, fx.appTimelineID},
				Spans: []ResolvedSpan{
					{
						ID:             "span-1",
						Label:          maxLabel,
						StartTimeMs:    afterStart.UnixMilli(),
						EndTimeMs:      fx.baseTime.Add(25 * time.Minute).UnixMilli(),
						DurationLabel:  "",
						Pattern:        SpanPatternSolid,
						LabelPlacement: SpanLabelPlacementAuto,
						HideDuration:   true,
						PaletteID:      "default",
					},
				},
			},
		},
		{
			name: "scopes span via timelineQuery with default subtree inclusion",
			spans: []SpanAnnotation{
				{
					Label:          "Pod subtree window",
					Start:          Anchor{Time: &beforeStart},
					End:            Anchor{Time: &beforeEnd},
					TimelineQuery:  `name == "web-1"`,
					Pattern:        SpanPatternHatched,
					LabelPlacement: SpanLabelPlacementAbove,
				},
			},
			want: spanResolveOutcome{
				AutoIncludeTimelineIDs: fx.allPodTreeIDs,
				Spans: []ResolvedSpan{
					{
						ID:             "span-1",
						Label:          "Pod subtree window",
						StartTimeMs:    beforeStart.UnixMilli(),
						EndTimeMs:      beforeEnd.UnixMilli(),
						DurationLabel:  "8m",
						TimelineIDs:    fx.allPodTreeIDs,
						Pattern:        SpanPatternHatched,
						LabelPlacement: SpanLabelPlacementAbove,
						PaletteID:      "default",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spec := Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Spans: tc.spans,
				},
			}
			specCopy := spec.Clone()
			applied := fx.wb.ResolveFilter(specCopy.Filter)
			gotOverlay, errRes, err := Resolve(ctx, fx.wb, fx.inspectionID, &specCopy, applied)
			if err != nil {
				t.Fatalf("Resolve() unexpected error = %v", err)
			}
			if errRes != nil {
				t.Fatalf("Resolve() unexpected tool error = %+v", errRes)
			}
			got := spanResolveOutcome{
				AutoIncludeTimelineIDs: gotOverlay.AutoIncludeTimelineIDs,
				Spans:                  gotOverlay.Spans,
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Resolve() span outcome mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolve_SpanErrors(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()

	validStart := fx.baseTime.Add(30 * time.Minute)
	validEnd := fx.baseTime.Add(40 * time.Minute)

	testCases := []struct {
		name       string
		span       SpanAnnotation
		wantCode   string
		wantSubstr string
	}{
		{
			name: "empty label returns INVALID_ARGUMENT",
			span: SpanAnnotation{
				Label: "   ",
				Start: Anchor{Time: &validStart},
				End:   Anchor{Time: &validEnd},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "`annotations.spans[0].label` is required.",
		},
		{
			name: "label exceeding max runes returns INVALID_ARGUMENT",
			span: SpanAnnotation{
				Label: strings.Repeat("a", maxAnnotationRunes+1),
				Start: Anchor{Time: &validStart},
				End:   Anchor{Time: &validEnd},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "exceeds maximum length of 120 characters",
		},
		{
			name: "unsupported pattern returns INVALID_ARGUMENT",
			span: SpanAnnotation{
				Label:   "bad pattern",
				Pattern: SpanPattern("dotted"),
				Start:   Anchor{Time: &validStart},
				End:     Anchor{Time: &validEnd},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "Unsupported `annotations.spans[0].pattern` `dotted`",
		},
		{
			name: "unsupported labelPlacement returns INVALID_ARGUMENT",
			span: SpanAnnotation{
				Label:          "bad placement",
				LabelPlacement: SpanLabelPlacement("below"),
				Start:          Anchor{Time: &validStart},
				End:            Anchor{Time: &validEnd},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "Unsupported `annotations.spans[0].labelPlacement` `below`",
		},
		{
			name: "unregistered paletteId returns UNKNOWN_PALETTE_ID",
			span: SpanAnnotation{
				Label:     "bad palette",
				PaletteID: "missing-id",
				Start:     Anchor{Time: &validStart},
				End:       Anchor{Time: &validEnd},
			},
			wantCode:   "UNKNOWN_PALETTE_ID",
			wantSubstr: "references unregistered palette ID `missing-id`",
		},
		{
			name: "invalid start anchor returns INVALID_ANCHOR",
			span: SpanAnnotation{
				Label: "bad start",
				Start: Anchor{},
				End:   Anchor{Time: &validEnd},
			},
			wantCode:   "INVALID_ANCHOR",
			wantSubstr: "annotations.spans[0].start",
		},
		{
			name: "unknown end logId returns LOG_NOT_FOUND",
			span: SpanAnnotation{
				Label: "bad end",
				Start: Anchor{Time: &validStart},
				End:   Anchor{LogID: 999999},
			},
			wantCode:   "LOG_NOT_FOUND",
			wantSubstr: "No log with ID 999999",
		},
		{
			name: "end before start returns INVALID_ANCHOR",
			span: SpanAnnotation{
				Label: "reversed",
				Start: Anchor{Time: &validEnd},
				End:   Anchor{Time: &validStart},
			},
			wantCode:   "INVALID_ANCHOR",
			wantSubstr: "must not be before",
		},
		{
			name: "unknown scoped timelineId returns TIMELINE_NOT_FOUND",
			span: SpanAnnotation{
				Label:       "bad scope",
				Start:       Anchor{Time: &validStart},
				End:         Anchor{Time: &validEnd},
				TimelineIDs: []uint32{999999},
			},
			wantCode:   "TIMELINE_NOT_FOUND",
			wantSubstr: "No timeline with ID 999999",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spec := Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Spans: []SpanAnnotation{tc.span},
				},
			}
			specCopy := spec.Clone()
			applied := fx.wb.ResolveFilter(specCopy.Filter)
			_, errRes, err := Resolve(ctx, fx.wb, fx.inspectionID, &specCopy, applied)
			if err != nil {
				t.Fatalf("Resolve() unexpected error = %v", err)
			}
			if errRes == nil || !errRes.IsError || len(errRes.Content) == 0 {
				t.Fatalf("Resolve() errRes = %+v, want error result with code %s", errRes, tc.wantCode)
			}
			textContent, ok := errRes.Content[0].(*mcpsdk.TextContent)
			if !ok {
				t.Fatalf("errRes.Content[0] = %T, want *mcpsdk.TextContent", errRes.Content[0])
			}
			wantHeader := "Error: " + tc.wantCode
			if !strings.HasPrefix(textContent.Text, wantHeader) {
				t.Errorf("error text = %q, want prefix %q", textContent.Text, wantHeader)
			}
			if !strings.Contains(textContent.Text, tc.wantSubstr) {
				t.Errorf("error text = %q, want substring %q", textContent.Text, tc.wantSubstr)
			}
		})
	}
}
