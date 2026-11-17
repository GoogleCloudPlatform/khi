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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/proto"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

func TestSliceWorkbenchIntegration(t *testing.T) {
	fx := setupResolverFixture(t)

	t.Run("SliceResourceDiff returns ErrTimelineNotFound for unknown timeline", func(t *testing.T) {
		_, err := SliceResourceDiff(fx.wb, 999999, 0, DiffSliceOptions{Paths: []string{"$.metadata.name"}})
		if !errors.Is(err, workbench.ErrTimelineNotFound) {
			t.Fatalf("SliceResourceDiff() error = %v, want %v", err, workbench.ErrTimelineNotFound)
		}
	})

	t.Run("SliceLogDetail returns ErrLogNotFound for unknown log", func(t *testing.T) {
		_, err := SliceLogDetail(fx.wb, 999999, LogSliceOptions{Windows: []SliceWindow{{Path: "$.message"}}})
		if !errors.Is(err, workbench.ErrLogNotFound) {
			t.Fatalf("SliceLogDetail() error = %v, want %v", err, workbench.ErrLogNotFound)
		}
	})

	t.Run("SliceResourceManifest returns ErrTimelineNotFound for unknown timeline", func(t *testing.T) {
		_, err := SliceResourceManifest(fx.wb, 999999, workbench.RevisionAtIndex(0), ManifestSliceOptions{Paths: []string{"$.metadata.name"}})
		if !errors.Is(err, workbench.ErrTimelineNotFound) {
			t.Fatalf("SliceResourceManifest() error = %v, want %v", err, workbench.ErrTimelineNotFound)
		}
	})
}

func TestResolve_Excerpts(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()

	nodeRev0Bounds, err := fx.wb.ResolveRevisionBounds(fx.nodeTimelineID, 0)
	if err != nil {
		t.Fatalf("ResolveRevisionBounds(node, 0) error = %v", err)
	}
	nodeRev1Bounds, err := fx.wb.ResolveRevisionBounds(fx.nodeTimelineID, 1)
	if err != nil {
		t.Fatalf("ResolveRevisionBounds(node, 1) error = %v", err)
	}
	podBounds, err := fx.wb.ResolveRevisionBounds(fx.podTimelineID, 0)
	if err != nil {
		t.Fatalf("ResolveRevisionBounds(pod, 0) error = %v", err)
	}

	revZero := 0
	revOne := 1
	zoomStart := fx.baseTime.Add(20 * time.Minute)
	zoomEnd := fx.baseTime.Add(40 * time.Minute)
	steadyPointTime := fx.baseTime.Add(28 * time.Minute)
	maxNote := strings.Repeat("n", maxAnnotationRunes)

	type excerptOutcome struct {
		Overlay        *Overlay
		DiffBadges     []string
		ManifestBadges []string
		LogBadges      []string
	}

	testCases := []struct {
		name string
		spec Spec
		want excerptOutcome
	}{
		{
			name: "diff excerpt resolves audit log insertId and timestamp without internal rev prefixes",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					DiffExcerpts: []DiffExcerpt{
						{
							Revision: RevisionRef{TimelineID: fx.podTimelineID, RevisionIndex: 0},
							Paths:    []string{"$.metadata", "$.status"},
							Note:     maxNote,
						},
					},
				},
			},
			want: excerptOutcome{
				Overlay: &Overlay{
					AutoIncludeTimelineIDs: []uint32{fx.podTimelineID},
					Palette: []ResolvedPaletteColors{
						{ID: "default", Main: "#1976d2", OnMain: "#fafafa", Soft: "rgba(25, 118, 210, 0.13)", Surface: "#e8f1fb"},
					},
					Excerpts: []ResolvedExcerptCard{
						{
							ID:           "excerpt-1",
							Kind:         ExcerptKindDiff,
							Badge:        "E1",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(12 * time.Minute).UnixMilli(),
							TimeLabel:    "09:12:00Z",
							TimelineID:   fx.podTimelineID,
							ResourcePath: "pod · default/web-1",
							Title:        "Create · Resource exists",
							Subtitle:     "+6 / -0 · 6 diff lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 2, Text: "⋮ 2 diff lines omitted"},
								{Kind: ExcerptLineKindAdded, NewLineNum: 3, Text: "metadata:", Parts: []LinePart{{Text: "metadata:", Matched: true}}},
								{Kind: ExcerptLineKindAdded, NewLineNum: 4, Text: "  name: web-1"},
								{Kind: ExcerptLineKindAdded, NewLineNum: 5, Text: "status:", Parts: []LinePart{{Text: "status:", Matched: true}}},
								{Kind: ExcerptLineKindAdded, NewLineNum: 6, Text: "  phase: Running"},
							},
							Note:          maxNote,
							LogTimestamp:  "2026-10-01T09:12:00Z",
							InsertID:      "audit-pod-web-1",
							LogID:         podBounds.Revision.LogID,
							RevisionIndex: &revZero,
						},
					},
				},
				DiffBadges: []string{"E1"},
			},
		},
		{
			name: "clamps revisionIndex manifest excerpt active across viewStart and anchors time-queried manifest excerpt at exact point",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter: workbench.Filter{
					StartTime: &zoomStart,
					EndTime:   &zoomEnd,
				},
				Annotations: &Annotations{
					Palette: []PaletteEntry{
						{ID: "before", Color: "slate"},
						{ID: "incident", Color: "amber"},
					},
					ManifestExcerpts: []ManifestExcerpt{
						{
							TimelineID:    fx.nodeTimelineID,
							RevisionIndex: &revZero,
							Badge:         "BEFORE",
							PaletteID:     "before",
							Paths:         []string{"$.status.ready", "$.status.reason"},
							Note:          "Steady Ready=True prior to incident",
						},
						{
							TimelineID: fx.nodeTimelineID,
							Time:       &steadyPointTime,
							Badge:      "DURING",
							PaletteID:  "incident",
							Paths:      []string{"$.status.ready", "$.status.reason"},
							Note:       "Point-in-time state at 09:28Z",
						},
					},
				},
			},
			want: excerptOutcome{
				Overlay: &Overlay{
					AutoIncludeTimelineIDs: []uint32{fx.nodeTimelineID},
					Palette: []ResolvedPaletteColors{
						{ID: "before", Main: "#546e7a", OnMain: "#fafafa", Soft: "rgba(84, 110, 122, 0.13)", Surface: "#eef1f2"},
						{ID: "incident", Main: "#b45309", OnMain: "#fafafa", Soft: "rgba(180, 83, 9, 0.13)", Surface: "#f8eee6"},
					},
					Excerpts: []ResolvedExcerptCard{
						{
							ID:           "excerpt-1",
							Kind:         ExcerptKindManifest,
							Badge:        "BEFORE",
							PaletteID:    "before",
							TimeMs:       zoomStart.UnixMilli(),
							TimeLabel:    "09:20:00Z",
							TimelineID:   fx.nodeTimelineID,
							ResourcePath: "node · gke-node-1",
							Title:        "Manifest · Resource exists",
							Subtitle:     "7 lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 lines omitted"},
								{Kind: ExcerptLineKindContext, NewLineNum: 6, Text: `  ready: "True"`, Parts: []LinePart{{Text: "  "}, {Text: `ready: "True"`, Matched: true}}},
								{Kind: ExcerptLineKindContext, NewLineNum: 7, Text: "  reason: KubeletReady", Parts: []LinePart{{Text: "  "}, {Text: "reason: KubeletReady", Matched: true}}},
							},
							Note:          "Steady Ready=True prior to incident",
							LogTimestamp:  "2026-10-01T09:10:00Z",
							LogID:         nodeRev0Bounds.Revision.LogID,
							RevisionIndex: &revZero,
						},
						{
							ID:           "excerpt-2",
							Kind:         ExcerptKindManifest,
							Badge:        "DURING",
							PaletteID:    "incident",
							TimeMs:       steadyPointTime.UnixMilli(),
							TimeLabel:    "09:28:00Z",
							TimelineID:   fx.nodeTimelineID,
							ResourcePath: "node · gke-node-1",
							Title:        "Manifest · Resource exists",
							Subtitle:     "7 lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 lines omitted"},
								{Kind: ExcerptLineKindContext, NewLineNum: 6, Text: `  ready: "False"`, Parts: []LinePart{{Text: "  "}, {Text: `ready: "False"`, Matched: true}}},
								{Kind: ExcerptLineKindContext, NewLineNum: 7, Text: "  reason: KubeletNotReady", Parts: []LinePart{{Text: "  "}, {Text: "reason: KubeletNotReady", Matched: true}}},
							},
							Note:          "Point-in-time state at 09:28Z",
							LogTimestamp:  "2026-10-01T09:24:00Z",
							LogID:         nodeRev1Bounds.Revision.LogID,
							RevisionIndex: &revOne,
						},
					},
				},
				ManifestBadges: []string{"BEFORE", "DURING"},
			},
		},
		{
			name: "resolves explicit log timelineId and unlinked log with default severity",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					LogExcerpts: []LogExcerpt{
						{
							LogID:      fx.oomEventLogID,
							TimelineID: fx.podTimelineID,
							Note:       maxNote,
						},
						{
							LogID: fx.unlinkedLogID,
						},
					},
				},
			},
			want: excerptOutcome{
				Overlay: &Overlay{
					AutoIncludeTimelineIDs: []uint32{fx.podTimelineID},
					Palette: []ResolvedPaletteColors{
						{ID: "default", Main: "#1976d2", OnMain: "#fafafa", Soft: "rgba(25, 118, 210, 0.13)", Surface: "#e8f1fb"},
					},
					Excerpts: []ResolvedExcerptCard{
						{
							ID:           "excerpt-1",
							Kind:         ExcerptKindLog,
							Badge:        "E1",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(25 * time.Minute).UnixMilli(),
							TimeLabel:    "09:25:00Z",
							TimelineID:   fx.podTimelineID,
							ResourcePath: "pod · default/web-1",
							Title:        "k8s event · ERROR",
							Subtitle:     "4 lines",
							Lines:        []ExcerptLine{{Kind: ExcerptLineKindSummary, Text: "Container app OOMKilled"}},
							Note:         maxNote,
							LogTimestamp: "2026-10-01T09:25:00Z",
							InsertID:     "evt-oom",
							LogID:        fx.oomEventLogID,
						},
						{
							ID:           "excerpt-2",
							Kind:         ExcerptKindLog,
							Badge:        "E2",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(26 * time.Minute).UnixMilli(),
							TimeLabel:    "09:26:00Z",
							Title:        "k8s event · INFO",
							Subtitle:     "1 line",
							Lines:        []ExcerptLine{{Kind: ExcerptLineKindSummary, Text: "Unlinked log entry"}},
							LogTimestamp: "2026-10-01T09:26:00Z",
							LogID:        fx.unlinkedLogID,
						},
					},
				},
				LogBadges: []string{"E1", "E2"},
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
			got := excerptOutcome{Overlay: gotOverlay}
			if specCopy.Annotations != nil {
				for _, d := range specCopy.Annotations.DiffExcerpts {
					got.DiffBadges = append(got.DiffBadges, d.Badge)
				}
				for _, me := range specCopy.Annotations.ManifestExcerpts {
					got.ManifestBadges = append(got.ManifestBadges, me.Badge)
				}
				for _, l := range specCopy.Annotations.LogExcerpts {
					got.LogBadges = append(got.LogBadges, l.Badge)
				}
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveExcerpts_Errors(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()

	narrowStart := fx.baseTime.Add(30 * time.Minute)
	narrowEnd := fx.baseTime.Add(40 * time.Minute)
	overNote := strings.Repeat("n", maxAnnotationRunes+1)

	testCases := []struct {
		name       string
		spec       Spec
		wantCode   string
		wantSubstr string
	}{
		{
			name: "diffExcerpt note exceeding maxAnnotationRunes returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{DiffExcerpts: []DiffExcerpt{{Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1}, Paths: []string{"$.status"}, Note: overNote}}},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "`annotations.diffExcerpts[0].note` exceeds maximum length of 120 characters.",
		},
		{
			name: "diffExcerpt unknown paletteId returns UNKNOWN_PALETTE_ID",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{DiffExcerpts: []DiffExcerpt{{Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1}, Paths: []string{"$.status"}, PaletteID: "missing"}}},
			},
			wantCode: "UNKNOWN_PALETTE_ID",
		},
		{
			name: "diffExcerpt missing timelineId returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{DiffExcerpts: []DiffExcerpt{{Paths: []string{"$.status"}}}},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "`annotations.diffExcerpts[0].revision.timelineId` is required.",
		},
		{
			name: "omitted diffExcerpt paths returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{DiffExcerpts: []DiffExcerpt{{Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1}}}},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "invalid JSONPath syntax in diffExcerpt returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{DiffExcerpts: []DiffExcerpt{{Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1}, Paths: []string{"$.status["}}}},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "unmatched diff JSONPath returns SLICE_QUERY_NOT_FOUND with guidance",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{DiffExcerpts: []DiffExcerpt{{Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1}, Paths: []string{"$.status.noSuchCondition"}}}},
			},
			wantCode: "SLICE_QUERY_NOT_FOUND",
		},
		{
			name: "diffExcerpt outside viewport reports revision fieldPath",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter:       workbench.Filter{StartTime: &narrowStart, EndTime: &narrowEnd},
				Annotations:  &Annotations{DiffExcerpts: []DiffExcerpt{{Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1}, Paths: []string{"$.status"}}}},
			},
			wantCode:   "ANCHOR_OUT_OF_VIEWPORT",
			wantSubstr: "`annotations.diffExcerpts[0].revision`",
		},
		{
			name: "manifestExcerpt note exceeding maxAnnotationRunes returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{TimelineID: fx.nodeTimelineID, RevisionIndex: intRef(0), Paths: []string{"$.status"}, Note: overNote}}},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "`annotations.manifestExcerpts[0].note` exceeds maximum length of 120 characters.",
		},
		{
			name: "manifestExcerpt unknown paletteId returns UNKNOWN_PALETTE_ID",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{TimelineID: fx.nodeTimelineID, RevisionIndex: intRef(0), Paths: []string{"$.status"}, PaletteID: "missing"}}},
			},
			wantCode: "UNKNOWN_PALETTE_ID",
		},
		{
			name: "manifestExcerpt missing timelineId returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{RevisionIndex: intRef(0), Paths: []string{"$.status"}}}},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "`annotations.manifestExcerpts[0].timelineId` is required.",
		},
		{
			name: "manifestExcerpt with neither revisionIndex nor time returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{TimelineID: fx.nodeTimelineID, Paths: []string{"$.status"}}}},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "manifestExcerpt with omitted paths returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{TimelineID: fx.nodeTimelineID, RevisionIndex: intRef(0)}}},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "manifestExcerpt with both revisionIndex and time returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{TimelineID: fx.nodeTimelineID, RevisionIndex: intRef(0), Time: &narrowStart, Paths: []string{"$.status"}}}},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "manifestExcerpt time on unknown timelineId returns TIMELINE_NOT_FOUND",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{TimelineID: 999999, Time: timeRef(fx.baseTime.Add(25 * time.Minute)), Paths: []string{"$.status"}}}},
			},
			wantCode: "TIMELINE_NOT_FOUND",
		},
		{
			name: "manifestExcerpt time outside viewport returns ANCHOR_OUT_OF_VIEWPORT with time fieldPath",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter:       workbench.Filter{StartTime: &narrowStart, EndTime: &narrowEnd},
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{TimelineID: fx.nodeTimelineID, Time: timeRef(fx.baseTime.Add(25 * time.Minute)), Paths: []string{"$.status"}}}},
			},
			wantCode:   "ANCHOR_OUT_OF_VIEWPORT",
			wantSubstr: "`annotations.manifestExcerpts[0].time`",
		},
		{
			name: "manifestExcerpt revisionIndex ended before viewStart returns ANCHOR_OUT_OF_VIEWPORT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter:       workbench.Filter{StartTime: &narrowStart, EndTime: &narrowEnd},
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{TimelineID: fx.nodeTimelineID, RevisionIndex: intRef(0), Paths: []string{"$.status"}}}},
			},
			wantCode:   "ANCHOR_OUT_OF_VIEWPORT",
			wantSubstr: "`annotations.manifestExcerpts[0].revisionIndex`",
		},
		{
			name: "manifestExcerpt revisionIndex starting after viewEnd returns ANCHOR_OUT_OF_VIEWPORT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter:       workbench.Filter{StartTime: timeRef(fx.baseTime.Add(15 * time.Minute)), EndTime: timeRef(fx.baseTime.Add(30 * time.Minute))},
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{TimelineID: fx.nodeTimelineID, RevisionIndex: intRef(2), Paths: []string{"$.status"}}}},
			},
			wantCode:   "ANCHOR_OUT_OF_VIEWPORT",
			wantSubstr: "`annotations.manifestExcerpts[0].revisionIndex`",
		},
		{
			name: "unmatched manifest JSONPath returns SLICE_QUERY_NOT_FOUND",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{ManifestExcerpts: []ManifestExcerpt{{TimelineID: fx.nodeTimelineID, RevisionIndex: intRef(0), Paths: []string{"$.status.noSuchManifestField"}}}},
			},
			wantCode: "SLICE_QUERY_NOT_FOUND",
		},
		{
			name: "logExcerpt note exceeding maxAnnotationRunes returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{LogExcerpts: []LogExcerpt{{LogID: fx.oomEventLogID, Note: overNote}}},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "`annotations.logExcerpts[0].note` exceeds maximum length of 120 characters.",
		},
		{
			name: "logExcerpt unknown paletteId returns UNKNOWN_PALETTE_ID",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{LogExcerpts: []LogExcerpt{{LogID: fx.oomEventLogID, PaletteID: "missing"}}},
			},
			wantCode: "UNKNOWN_PALETTE_ID",
		},
		{
			name: "logExcerpt missing logId returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{LogExcerpts: []LogExcerpt{{}}},
			},
			wantCode:   "INVALID_ARGUMENT",
			wantSubstr: "`annotations.logExcerpts[0].logId` is required.",
		},
		{
			name: "unmatched log slice query returns SLICE_QUERY_NOT_FOUND",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{LogExcerpts: []LogExcerpt{{LogID: fx.oomEventLogID, Windows: []SliceWindow{{Query: "NoSuchField"}}}}},
			},
			wantCode: "SLICE_QUERY_NOT_FOUND",
		},
		{
			name: "empty slice window returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{LogExcerpts: []LogExcerpt{{LogID: fx.oomEventLogID, Windows: []SliceWindow{{}}}}},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "unknown logId in logExcerpt returns LOG_NOT_FOUND",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{LogExcerpts: []LogExcerpt{{LogID: 999999}}},
			},
			wantCode: "LOG_NOT_FOUND",
		},
		{
			name: "logExcerpt outside viewport reports logId fieldPath",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter:       workbench.Filter{StartTime: &narrowStart, EndTime: &narrowEnd},
				Annotations:  &Annotations{LogExcerpts: []LogExcerpt{{LogID: fx.oomEventLogID}}},
			},
			wantCode:   "ANCHOR_OUT_OF_VIEWPORT",
			wantSubstr: "`annotations.logExcerpts[0].logId`",
		},
		{
			name: "logExcerpt unknown explicit timelineId returns TIMELINE_NOT_FOUND",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{LogExcerpts: []LogExcerpt{{LogID: fx.oomEventLogID, TimelineID: 999999}}},
			},
			wantCode: "TIMELINE_NOT_FOUND",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertResolveError(t, ctx, fx, tc.spec, tc.wantCode, tc.wantSubstr)
		})
	}
}

func TestResolveManifestRevisionViewportPinTime(t *testing.T) {
	base := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	viewStart := base.Add(10 * time.Minute)
	viewEnd := base.Add(20 * time.Minute)
	applied := workbench.AppliedFilter{StartTime: viewStart, EndTime: viewEnd}

	testCases := []struct {
		name        string
		revTime     time.Time
		nextRevTime *time.Time
		applied     workbench.AppliedFilter
		rawFilter   workbench.Filter
		wantTime    time.Time
		wantErrCode string
	}{
		{
			name:     "zero revTime returns zero time without error",
			revTime:  time.Time{},
			applied:  applied,
			wantTime: time.Time{},
		},
		{
			name:        "header slack allows revTime 500ms before applied start when rawFilter.StartTime is nil",
			revTime:     viewStart.Add(-500 * time.Millisecond),
			nextRevTime: timeRef(viewStart),
			applied:     applied,
			rawFilter:   workbench.Filter{},
			wantTime:    viewStart,
		},
		{
			name:        "explicit rawFilter.StartTime rejects revTime 500ms before applied start",
			revTime:     viewStart.Add(-500 * time.Millisecond),
			nextRevTime: timeRef(viewStart),
			applied:     applied,
			rawFilter:   workbench.Filter{StartTime: &viewStart},
			wantErrCode: "ANCHOR_OUT_OF_VIEWPORT",
		},
		{
			name:      "header slack allows revTime 500ms after applied end when rawFilter.EndTime is nil",
			revTime:   viewEnd.Add(500 * time.Millisecond),
			applied:   applied,
			rawFilter: workbench.Filter{},
			wantTime:  viewEnd.Add(500 * time.Millisecond),
		},
		{
			name:        "explicit rawFilter.EndTime rejects revTime 500ms after applied end",
			revTime:     viewEnd.Add(500 * time.Millisecond),
			applied:     applied,
			rawFilter:   workbench.Filter{EndTime: &viewEnd},
			wantErrCode: "ANCHOR_OUT_OF_VIEWPORT",
		},
		{
			name:        "revTime equal to minBound with nextRevTime equal to minBound stays inside viewport",
			revTime:     viewStart,
			nextRevTime: timeRef(viewStart),
			applied:     applied,
			rawFilter:   workbench.Filter{StartTime: &viewStart},
			wantTime:    viewStart,
		},
		{
			name:      "zero applied start does not clamp revTime",
			revTime:   viewStart,
			applied:   workbench.AppliedFilter{},
			rawFilter: workbench.Filter{},
			wantTime:  viewStart,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotTime, errRes := resolveManifestRevisionViewportPinTime(tc.revTime, tc.nextRevTime, tc.applied, tc.rawFilter, "annotations.manifestExcerpts[0].revisionIndex")
			if tc.wantErrCode != "" {
				if errRes == nil || !errRes.IsError || len(errRes.Content) == 0 {
					t.Fatalf("resolveManifestRevisionViewportPinTime() errRes = %+v, want %s", errRes, tc.wantErrCode)
				}
				txt := errRes.Content[0].(*mcpsdk.TextContent).Text
				if !strings.HasPrefix(txt, "Error: "+tc.wantErrCode) {
					t.Errorf("error text = %q, want prefix Error: %s", txt, tc.wantErrCode)
				}
				return
			}
			if errRes != nil {
				t.Fatalf("unexpected error result = %+v", errRes)
			}
			if !gotTime.Equal(tc.wantTime) {
				t.Errorf("pinTime = %v, want %v", gotTime, tc.wantTime)
			}
		})
	}
}

func TestFormatManifestExcerptTitle_AndLogExcerptTitle(t *testing.T) {
	testCases := []struct {
		name  string
		state string
		want  string
	}{
		{name: "blank state falls back to Manifest", state: "   ", want: "Manifest"},
		{name: "non-empty state includes state suffix", state: "Resource exists", want: "Manifest · Resource exists"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatManifestExcerptTitle(tc.state); got != tc.want {
				t.Errorf("formatManifestExcerptTitle(%q) = %q, want %q", tc.state, got, tc.want)
			}
		})
	}

	logTitleCases := []struct {
		name     string
		logType  string
		severity *khifilev6.Severity
		want     string
	}{
		{name: "nil severity and empty logType default to log and INFO", want: "log · INFO"},
		{name: "explicit logType and severity label", logType: "k8s event", severity: &khifilev6.Severity{Label: proto.String("ERROR")}, want: "k8s event · ERROR"},
	}
	for _, tc := range logTitleCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatLogExcerptTitle(tc.logType, tc.severity); got != tc.want {
				t.Errorf("formatLogExcerptTitle(%q, %+v) = %q, want %q", tc.logType, tc.severity, got, tc.want)
			}
		})
	}
}
