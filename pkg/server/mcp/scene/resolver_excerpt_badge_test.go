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

type badgeSyncOutcome struct {
	Overlay        *Overlay
	MarkerBadges   []string
	DiffBadges     []string
	ManifestBadges []string
	LogBadges      []string
}

func collectBadgeSyncOutcome(gotOverlay *Overlay, ann *Annotations) badgeSyncOutcome {
	got := badgeSyncOutcome{Overlay: gotOverlay}
	if ann == nil {
		return got
	}
	for _, m := range ann.Markers {
		got.MarkerBadges = append(got.MarkerBadges, m.Badge)
	}
	for _, d := range ann.DiffExcerpts {
		got.DiffBadges = append(got.DiffBadges, d.Badge)
	}
	for _, me := range ann.ManifestExcerpts {
		got.ManifestBadges = append(got.ManifestBadges, me.Badge)
	}
	for _, l := range ann.LogExcerpts {
		got.LogBadges = append(got.LogBadges, l.Badge)
	}
	return got
}

func TestResolve_BadgeSyncAndOrdering(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()

	nodeRev1Bounds, err := fx.wb.ResolveRevisionBounds(fx.nodeTimelineID, 1)
	if err != nil {
		t.Fatalf("ResolveRevisionBounds(node, 1) error = %v", err)
	}
	podRev0Bounds, err := fx.wb.ResolveRevisionBounds(fx.podTimelineID, 0)
	if err != nil {
		t.Fatalf("ResolveRevisionBounds(pod, 0) error = %v", err)
	}

	revZero := 0
	revOne := 1
	earlyRefTime := fx.baseTime.Add(21 * time.Minute)
	lateRefTime := fx.baseTime.Add(36 * time.Minute)

	testCases := []struct {
		name string
		spec Spec
		want badgeSyncOutcome
	}{
		{
			name: "synchronizes co-anchored badges in both directions, numbers unanchored markers chronologically, and orders tied diff before manifest before log",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{
						{Anchor: Anchor{Time: &lateRefTime}, Label: "Post-incident check", Style: MarkerStyleReference},
						{Anchor: Anchor{TimelineID: fx.nodeTimelineID, RevisionIndex: &revOne}, Label: "Node NotReady transition"},
						{Anchor: Anchor{LogID: fx.oomEventLogID}, Label: "App OOMKilled", Badge: "OOM"},
						{Anchor: Anchor{Time: &earlyRefTime}, Label: "Traffic surge started", Style: MarkerStyleReference},
					},
					ManifestExcerpts: []ManifestExcerpt{
						{TimelineID: fx.podTimelineID, RevisionIndex: &revZero, Paths: []string{"$.status.phase"}},
						{TimelineID: fx.nodeTimelineID, RevisionIndex: &revOne, Paths: []string{"$.metadata.name"}},
					},
					DiffExcerpts: []DiffExcerpt{
						{Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1}, Paths: []string{"$.status.reason"}},
					},
					LogExcerpts: []LogExcerpt{
						{LogID: fx.oomEventLogID},
					},
				},
			},
			want: badgeSyncOutcome{
				Overlay: &Overlay{
					AutoIncludeTimelineIDs: []uint32{fx.nodeTimelineID, fx.podTimelineID, fx.appTimelineID},
					Palette: []ResolvedPaletteColors{
						{ID: "default", Main: "#1976d2", OnMain: "#fafafa", Soft: "rgba(25, 118, 210, 0.13)", Surface: "#e8f1fb"},
						{ID: "default-reference", Main: "#5e35b1", OnMain: "#fafafa", Soft: "rgba(94, 53, 177, 0.13)", Surface: "#efebf7"},
					},
					Markers: []ResolvedMarker{
						{ID: "marker-1", Badge: "1", Label: "Traffic surge started", Style: MarkerStyleReference, PaletteID: "default-reference", TimeMs: earlyRefTime.UnixMilli(), TimeLabel: "09:21:00Z"},
						{ID: "marker-2", Badge: "E2", Label: "Node NotReady transition", Style: MarkerStylePin, PaletteID: "default", TimeMs: fx.baseTime.Add(24 * time.Minute).UnixMilli(), TimeLabel: "09:24:00Z", TimelineID: fx.nodeTimelineID, LogID: nodeRev1Bounds.Revision.LogID, RevisionIndex: &revOne},
						{ID: "marker-3", Badge: "OOM", Label: "App OOMKilled", Style: MarkerStylePin, PaletteID: "default", TimeMs: fx.baseTime.Add(25 * time.Minute).UnixMilli(), TimeLabel: "09:25:00Z", TimelineID: fx.appTimelineID, LogID: fx.oomEventLogID},
						{ID: "marker-4", Badge: "2", Label: "Post-incident check", Style: MarkerStyleReference, PaletteID: "default-reference", TimeMs: lateRefTime.UnixMilli(), TimeLabel: "09:36:00Z"},
					},
					Excerpts: []ResolvedExcerptCard{
						{
							ID:           "excerpt-1",
							Kind:         ExcerptKindManifest,
							Badge:        "E1",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(12 * time.Minute).UnixMilli(),
							TimeLabel:    "09:12:00Z",
							TimelineID:   fx.podTimelineID,
							ResourcePath: "pod · default/web-1",
							Title:        "Manifest · Resource exists",
							Subtitle:     "6 lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 lines omitted"},
								{Kind: ExcerptLineKindContext, NewLineNum: 6, Text: "  phase: Running", Parts: []LinePart{{Text: "  "}, {Text: "phase: Running", Matched: true}}},
							},
							LogTimestamp:  "2026-10-01T09:12:00Z",
							InsertID:      "audit-pod-web-1",
							LogID:         podRev0Bounds.Revision.LogID,
							RevisionIndex: &revZero,
						},
						{
							ID:           "excerpt-2",
							Kind:         ExcerptKindDiff,
							Badge:        "E2",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(24 * time.Minute).UnixMilli(),
							TimeLabel:    "09:24:00Z",
							TimelineID:   fx.nodeTimelineID,
							ResourcePath: "node · gke-node-1",
							Title:        "Patch · Resource exists",
							Subtitle:     "+2 / -2 · 7 diff lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 diff lines omitted"},
								{Kind: ExcerptLineKindDeleted, OldLineNum: 7, Text: "  reason: KubeletReady", Parts: []LinePart{{Text: "  "}, {Text: "reason: KubeletReady", Matched: true}}},
								{Kind: ExcerptLineKindAdded, NewLineNum: 7, Text: "  reason: KubeletNotReady", Parts: []LinePart{{Text: "  "}, {Text: "reason: KubeletNotReady", Matched: true}}},
							},
							LogTimestamp:  "2026-10-01T09:24:00Z",
							LogID:         nodeRev1Bounds.Revision.LogID,
							RevisionIndex: &revOne,
						},
						{
							ID:           "excerpt-3",
							Kind:         ExcerptKindManifest,
							Badge:        "E3",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(24 * time.Minute).UnixMilli(),
							TimeLabel:    "09:24:00Z",
							TimelineID:   fx.nodeTimelineID,
							ResourcePath: "node · gke-node-1",
							Title:        "Manifest · Resource exists",
							Subtitle:     "7 lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 3, Text: "⋮ 3 lines omitted"},
								{Kind: ExcerptLineKindContext, NewLineNum: 4, Text: "  name: gke-node-1", Parts: []LinePart{{Text: "  "}, {Text: "name: gke-node-1", Matched: true}}},
								{Kind: ExcerptLineKindOmitted, Omitted: 3, Text: "⋮ 3 lines omitted"},
							},
							LogTimestamp:  "2026-10-01T09:24:00Z",
							LogID:         nodeRev1Bounds.Revision.LogID,
							RevisionIndex: &revOne,
						},
						{
							ID:           "excerpt-4",
							Kind:         ExcerptKindLog,
							Badge:        "OOM",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(25 * time.Minute).UnixMilli(),
							TimeLabel:    "09:25:00Z",
							TimelineID:   fx.appTimelineID,
							ResourcePath: "web-1/app · container",
							Title:        "k8s event · ERROR",
							Subtitle:     "4 lines",
							Lines:        []ExcerptLine{{Kind: ExcerptLineKindSummary, Text: "Container app OOMKilled"}},
							LogTimestamp: "2026-10-01T09:25:00Z",
							InsertID:     "evt-oom",
							LogID:        fx.oomEventLogID,
						},
					},
				},
				MarkerBadges:   []string{"2", "E2", "OOM", "1"},
				DiffBadges:     []string{"E2"},
				ManifestBadges: []string{"E1", "E3"},
				LogBadges:      []string{"OOM"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			specCopy := tc.spec.Clone()
			applied := fx.wb.ResolveFilter(specCopy.Filter)
			gotOverlay, errRes, err := Resolve(ctx, fx.wb, fx.inspectionID, &specCopy, applied)
			if err != nil {
				t.Fatalf("Resolve() unexpected error = %v", err)
			}
			if errRes != nil {
				t.Fatalf("Resolve() unexpected tool error = %+v", errRes)
			}
			got := collectBadgeSyncOutcome(gotOverlay, specCopy.Annotations)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSortPendingExcerpts_AndBadgeSync(t *testing.T) {
	t.Run("sorts tied timestamps diff < manifest < log and breaks kind ties by itemIndex", func(t *testing.T) {
		pending := []pendingExcerpt{
			{kind: ExcerptKindLog, itemIndex: 0, card: ResolvedExcerptCard{TimeMs: 100, Title: "log-0"}},
			{kind: ExcerptKindManifest, itemIndex: 1, card: ResolvedExcerptCard{TimeMs: 100, Title: "manifest-1"}},
			{kind: ExcerptKindManifest, itemIndex: 0, card: ResolvedExcerptCard{TimeMs: 100, Title: "manifest-0"}},
			{kind: ExcerptKindDiff, itemIndex: 0, card: ResolvedExcerptCard{TimeMs: 100, Title: "diff-0"}},
		}
		sortPendingExcerpts(pending)
		var gotTitles []string
		for _, p := range pending {
			gotTitles = append(gotTitles, p.card.Title)
		}
		wantTitles := []string{"diff-0", "manifest-0", "manifest-1", "log-0"}
		if diff := cmp.Diff(wantTitles, gotTitles); diff != "" {
			t.Errorf("sortPendingExcerpts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("accepts increasing, equal sequential, and trailing custom badges while inheriting explicit marker E1 and skipping E1", func(t *testing.T) {
		ann := &Annotations{
			DiffExcerpts:     []DiffExcerpt{{Badge: "E2"}, {Badge: "E02"}, {Badge: "CUSTOM"}},
			ManifestExcerpts: []ManifestExcerpt{{}},
			LogExcerpts:      []LogExcerpt{{}},
		}
		pending := []pendingExcerpt{
			{kind: ExcerptKindDiff, itemIndex: 0, fieldPath: "annotations.diffExcerpts[0]", explicitBadge: true, card: ResolvedExcerptCard{Badge: "E2", TimeMs: 100, TimeLabel: "09:00:00Z"}},
			{kind: ExcerptKindDiff, itemIndex: 1, fieldPath: "annotations.diffExcerpts[1]", explicitBadge: true, card: ResolvedExcerptCard{Badge: "E02", TimeMs: 200, TimeLabel: "09:01:00Z"}},
			{kind: ExcerptKindDiff, itemIndex: 2, fieldPath: "annotations.diffExcerpts[2]", explicitBadge: true, card: ResolvedExcerptCard{Badge: "CUSTOM", TimeMs: 250, TimeLabel: "09:01:30Z"}},
			{kind: ExcerptKindManifest, itemIndex: 0, fieldPath: "annotations.manifestExcerpts[0]", explicitBadge: false, card: ResolvedExcerptCard{TimeMs: 300, TimelineID: 10, LogID: 55}},
			{kind: ExcerptKindLog, itemIndex: 0, fieldPath: "annotations.logExcerpts[0]", explicitBadge: false, card: ResolvedExcerptCard{TimeMs: 400, TimelineID: 20, LogID: 99}},
		}
		markers := []pendingMarker{
			{explicitBadge: false, marker: ResolvedMarker{TimeMs: 300}},
			{explicitBadge: true, marker: ResolvedMarker{Badge: "E1", TimeMs: 300, TimelineID: 10, LogID: 55}},
		}
		overlay := &Overlay{}
		if errRes := validateAndSyncExcerptBadges(ann, pending, markers, overlay); errRes != nil {
			t.Fatalf("validateAndSyncExcerptBadges() unexpected error = %+v", errRes)
		}
		gotBadges := []string{overlay.Excerpts[0].Badge, overlay.Excerpts[1].Badge, overlay.Excerpts[2].Badge, overlay.Excerpts[3].Badge, overlay.Excerpts[4].Badge}
		if diff := cmp.Diff([]string{"E2", "E02", "CUSTOM", "E1", "E3"}, gotBadges); diff != "" {
			t.Errorf("assigned badges mismatch (-want +got):\n%s", diff)
		}
		if ann.ManifestExcerpts[0].Badge != "E1" || ann.LogExcerpts[0].Badge != "E3" {
			t.Errorf("back-filled badges = (%q, %q), want (E1, E3)", ann.ManifestExcerpts[0].Badge, ann.LogExcerpts[0].Badge)
		}
	})

	t.Run("rejects duplicate and out-of-order manual excerpt badges with detailed context", func(t *testing.T) {
		dupPending := []pendingExcerpt{
			{fieldPath: "annotations.diffExcerpts[0]", explicitBadge: true, card: ResolvedExcerptCard{Badge: "DUP", TimeLabel: "09:00:00Z"}},
			{fieldPath: "annotations.logExcerpts[0]", explicitBadge: true, card: ResolvedExcerptCard{Badge: "DUP", TimeLabel: "09:01:00Z"}},
		}
		dupErr := validateAndSyncExcerptBadges(&Annotations{}, dupPending, nil, &Overlay{})
		if dupErr == nil || !strings.Contains(dupErr.Content[0].(*mcpsdk.TextContent).Text, "Duplicate excerpt badge `DUP`") {
			t.Errorf("unexpected duplicate error result: %+v", dupErr)
		}

		orderPending := []pendingExcerpt{
			{fieldPath: "annotations.diffExcerpts[0]", explicitBadge: true, card: ResolvedExcerptCard{Badge: "E4", TimeLabel: "09:24:00Z"}},
			{fieldPath: "annotations.logExcerpts[0]", explicitBadge: true, card: ResolvedExcerptCard{Badge: "E1", TimeLabel: "09:25:00Z"}},
		}
		orderErr := validateAndSyncExcerptBadges(&Annotations{}, orderPending, nil, &Overlay{})
		if orderErr == nil {
			t.Fatal("expected out-of-order error, got nil")
		}
		gotText := orderErr.Content[0].(*mcpsdk.TextContent).Text
		for _, sub := range []string{"`E1`", "`annotations.logExcerpts[0].badge`", "09:25:00Z", "`E4`", "`annotations.diffExcerpts[0].badge`", "09:24:00Z"} {
			if !strings.Contains(gotText, sub) {
				t.Errorf("out-of-order text = %q, want substring %q", gotText, sub)
			}
		}
	})
}

func TestParseExcerptSeqBadge(t *testing.T) {
	testCases := []struct {
		badge  string
		wantN  int
		wantOK bool
	}{
		{badge: "E1", wantN: 1, wantOK: true},
		{badge: "E9", wantN: 9, wantOK: true},
		{badge: "E10", wantN: 10, wantOK: true},
		{badge: "", wantOK: false},
		{badge: "E", wantOK: false},
		{badge: "X1", wantOK: false},
		{badge: "E0", wantOK: false},
		{badge: "E+1", wantOK: false},
		{badge: "E/", wantOK: false},
		{badge: "E:", wantOK: false},
		{badge: "EA", wantOK: false},
		{badge: "E99999999999999999999999", wantOK: false},
	}

	for _, tc := range testCases {
		t.Run("badge_"+tc.badge, func(t *testing.T) {
			gotN, gotOK := parseExcerptSeqBadge(tc.badge)
			if gotN != tc.wantN || gotOK != tc.wantOK {
				t.Errorf("parseExcerptSeqBadge(%q) = (%d, %v), want (%d, %v)", tc.badge, gotN, gotOK, tc.wantN, tc.wantOK)
			}
		})
	}
}

func TestIsSameMarkerExcerptAnchor(t *testing.T) {
	rev0 := 0
	rev1 := 1

	testCases := []struct {
		name string
		m    ResolvedMarker
		ex   ResolvedExcerptCard
		want bool
	}{
		{
			name: "mismatched timestamps return false",
			m:    ResolvedMarker{TimeMs: 100, TimelineID: 1, LogID: 10},
			ex:   ResolvedExcerptCard{TimeMs: 200, TimelineID: 1, LogID: 10},
			want: false,
		},
		{
			name: "matching revision anchor returns true",
			m:    ResolvedMarker{TimeMs: 100, TimelineID: 1, RevisionIndex: &rev1},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 1, RevisionIndex: &rev1},
			want: true,
		},
		{
			name: "different revisionIndex returns false",
			m:    ResolvedMarker{TimeMs: 100, TimelineID: 1, RevisionIndex: &rev0},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 1, RevisionIndex: &rev1},
			want: false,
		},
		{
			name: "different timelineId with same revisionIndex returns false",
			m:    ResolvedMarker{TimeMs: 100, TimelineID: 1, RevisionIndex: &rev1},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 2, RevisionIndex: &rev1},
			want: false,
		},
		{
			name: "zero marker timelineId with revisionIndex does not match revision check",
			m:    ResolvedMarker{TimeMs: 100, TimelineID: 0, RevisionIndex: &rev1},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 0, RevisionIndex: &rev1},
			want: false,
		},
		{
			name: "marker with revisionIndex falls back to logId when excerpt has nil revisionIndex",
			m:    ResolvedMarker{TimeMs: 100, TimelineID: 1, LogID: 10, RevisionIndex: &rev1},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 1, LogID: 10},
			want: true,
		},
		{
			name: "marker without revisionIndex falls back to logId when excerpt has revisionIndex",
			m:    ResolvedMarker{TimeMs: 100, TimelineID: 1, LogID: 10},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 1, LogID: 10, RevisionIndex: &rev1},
			want: true,
		},
		{
			name: "matching logId with zero marker timelineId returns true",
			m:    ResolvedMarker{TimeMs: 100, TimelineID: 0, LogID: 10},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 1, LogID: 10},
			want: true,
		},
		{
			name: "matching logId with zero excerpt timelineId returns true",
			m:    ResolvedMarker{TimeMs: 100, TimelineID: 1, LogID: 10},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 0, LogID: 10},
			want: true,
		},
		{
			name: "matching logId on different non-zero timelines returns false",
			m:    ResolvedMarker{TimeMs: 100, TimelineID: 1, LogID: 10},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 2, LogID: 10},
			want: false,
		},
		{
			name: "different non-zero logIds on same timeline return false",
			m:    ResolvedMarker{TimeMs: 100, Style: MarkerStylePin, TimelineID: 1, LogID: 10},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 1, LogID: 20},
			want: false,
		},
		{
			name: "reference marker without log or revision does not match by timeline",
			m:    ResolvedMarker{TimeMs: 100, Style: MarkerStyleReference, TimelineID: 1},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 1},
			want: false,
		},
		{
			name: "pin marker with timelineId and no log or revision matches excerpt on same timeline",
			m:    ResolvedMarker{TimeMs: 100, Style: MarkerStylePin, TimelineID: 1},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 1},
			want: true,
		},
		{
			name: "pin marker with timelineId and no log or revision rejects different timeline",
			m:    ResolvedMarker{TimeMs: 100, Style: MarkerStylePin, TimelineID: 1},
			ex:   ResolvedExcerptCard{TimeMs: 100, TimelineID: 2},
			want: false,
		},
		{
			name: "pin marker without timeline, log, or revision returns false",
			m:    ResolvedMarker{TimeMs: 100, Style: MarkerStylePin},
			ex:   ResolvedExcerptCard{TimeMs: 100},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSameMarkerExcerptAnchor(tc.m, tc.ex); got != tc.want {
				t.Errorf("isSameMarkerExcerptAnchor() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolve_ExplicitDiffBadgeAndErrors(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()
	revOne := 1

	t.Run("explicit diffExcerpt badge is preserved even when co-anchored marker has explicit badge", func(t *testing.T) {
		spec := Spec{
			InspectionID: fx.inspectionID,
			Annotations: &Annotations{
				Markers: []MarkerAnnotation{
					{Anchor: Anchor{TimelineID: fx.nodeTimelineID, RevisionIndex: &revOne}, Label: "Node NotReady", Badge: "M1"},
				},
				DiffExcerpts: []DiffExcerpt{
					{Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1}, Paths: []string{"$.status.reason"}, Badge: "DIFF"},
				},
			},
		}
		applied := fx.wb.ResolveFilter(spec.Filter)
		gotOverlay, errRes, err := Resolve(ctx, fx.wb, fx.inspectionID, &spec, applied)
		if err != nil {
			t.Fatalf("Resolve() unexpected error = %v", err)
		}
		if errRes != nil {
			t.Fatalf("Resolve() unexpected tool error = %+v", errRes)
		}
		if len(gotOverlay.Excerpts) != 1 || gotOverlay.Excerpts[0].Badge != "DIFF" {
			t.Errorf("overlay Excerpts = %+v, want single card with Badge DIFF", gotOverlay.Excerpts)
		}
		if spec.Annotations.DiffExcerpts[0].Badge != "DIFF" {
			t.Errorf("DiffExcerpts[0].Badge = %q, want %q", spec.Annotations.DiffExcerpts[0].Badge, "DIFF")
		}
	})

	t.Run("duplicate excerpt badge returns INVALID_ARGUMENT through Resolve", func(t *testing.T) {
		spec := Spec{
			InspectionID: fx.inspectionID,
			Annotations: &Annotations{
				DiffExcerpts: []DiffExcerpt{
					{Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1}, Paths: []string{"$.status.reason"}, Badge: "DUP"},
					{Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 2}, Paths: []string{"$.status.reason"}, Badge: "DUP"},
				},
			},
		}
		applied := fx.wb.ResolveFilter(spec.Filter)
		_, errRes, err := Resolve(ctx, fx.wb, fx.inspectionID, &spec, applied)
		if err != nil {
			t.Fatalf("Resolve() unexpected error = %v", err)
		}
		if errRes == nil || !errRes.IsError || len(errRes.Content) == 0 {
			t.Fatalf("Resolve() errRes = %+v, want INVALID_ARGUMENT error", errRes)
		}
		txt := errRes.Content[0].(*mcpsdk.TextContent).Text
		if !strings.HasPrefix(txt, "Error: INVALID_ARGUMENT") {
			t.Errorf("error text = %q, want prefix Error: INVALID_ARGUMENT", txt)
		}
	})
}
