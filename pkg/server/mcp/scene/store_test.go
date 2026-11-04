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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var ignoreMCPUnexported = cmpopts.IgnoreUnexported(
	mcpsdk.CallToolResult{},
	mcpsdk.ImageContent{},
	mcpsdk.TextContent{},
)

func boolRef(v bool) *bool {
	return &v
}

func timeRef(t time.Time) *time.Time {
	return &t
}

func wantErrorResult(code string, bullets ...string) *mcpsdk.CallToolResult {
	res, _, _ := mdtemplate.ErrorResult(code, bullets...)
	return res
}

func TestStore_IssueAndGet(t *testing.T) {
	baseNow := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	currentNow := baseNow
	store := NewStore(30 * time.Minute)
	store.now = func() time.Time { return currentNow }

	stale, err := store.Issue("", Spec{InspectionID: "insp-stale"}, Document{InspectionID: "insp-stale"})
	if err != nil {
		t.Fatalf("Issue(stale) unexpected error = %v", err)
	}

	issueNow := baseNow.Add(31 * time.Minute)
	currentNow = issueNow
	revIdx := 1
	viewStartMs := int64(1759309200000)
	spec := Spec{
		InspectionID: "insp-1",
		TimelineIDs:  []uint32{5, 10},
		Annotations: &Annotations{
			Markers: []MarkerAnnotation{{Label: "OOMKilled", Badge: "1", Style: MarkerStylePin}},
		},
	}
	doc := Document{
		InspectionID:    "insp-1",
		TimelineQuery:   `timelineType == "Pod"`,
		ViewStartTimeMs: &viewStartMs,
		TimelineIDs:     []uint32{5, 10},
		Overlay: &Overlay{
			Markers: []ResolvedMarker{
				{ID: "marker-1", Badge: "1", Label: "OOMKilled", Style: MarkerStylePin, PaletteID: "default", TimeMs: viewStartMs, TimeLabel: "09:00:00Z", TimelineID: 10, RevisionIndex: &revIdx},
			},
			Spans:         []ResolvedSpan{{ID: "span-1", Label: "Window", TimelineIDs: []uint32{5}}},
			RowHighlights: []ResolvedRowHighlight{{ID: "row-1", TimelineIDs: []uint32{10}}},
			Excerpts: []ResolvedExcerptCard{
				{ID: "ex-1", RevisionIndex: &revIdx, Lines: []ExcerptLine{{Kind: ExcerptLineKindAdded, Text: "image: v2", Parts: []LinePart{{Text: "v2", Matched: true}}}}},
			},
			Callouts: []ResolvedCallout{{ID: "cal-1", Label: "Probe failed", RevisionIndex: &revIdx}},
		},
	}

	issued, err := store.Issue("", spec, doc)
	if err != nil {
		t.Fatalf("Issue() unexpected error = %v", err)
	}
	if !strings.HasPrefix(issued.SceneID, "scn_") {
		t.Fatalf("Issue().SceneID = %q, want prefix \"scn_\"", issued.SceneID)
	}

	// Mutating caller inputs and returned Stored values must not mutate stored scene state.
	spec.TimelineIDs[0] = 999
	spec.Annotations.Markers[0].Label = "mutated"
	viewStartMs = -1
	revIdx = 99
	doc.TimelineIDs[0] = 999
	doc.Overlay.Markers[0].Label = "mutated"
	doc.Overlay.Spans[0].TimelineIDs[0] = 999
	doc.Overlay.Excerpts[0].Lines[0].Parts[0].Text = "mutated"
	issued.Spec.TimelineIDs[0] = 888
	issued.Document.TimelineIDs[0] = 888

	wantRevIdx := 1
	wantViewStartMs := int64(1759309200000)
	wantExpiresAt := issueNow.Add(30 * time.Minute)
	wantStored := Stored{
		SceneID:   issued.SceneID,
		ExpiresAt: wantExpiresAt,
		Spec: Spec{
			InspectionID: "insp-1",
			TimelineIDs:  []uint32{5, 10},
			Annotations: &Annotations{
				Markers: []MarkerAnnotation{{Label: "OOMKilled", Badge: "1", Style: MarkerStylePin}},
			},
		},
		Document: Document{
			SceneID:         issued.SceneID,
			ExpiresAt:       wantExpiresAt,
			InspectionID:    "insp-1",
			TimelineQuery:   `timelineType == "Pod"`,
			ViewStartTimeMs: &wantViewStartMs,
			TimelineIDs:     []uint32{5, 10},
			Overlay: &Overlay{
				Markers: []ResolvedMarker{
					{ID: "marker-1", Badge: "1", Label: "OOMKilled", Style: "pin", PaletteID: "default", TimeMs: wantViewStartMs, TimeLabel: "09:00:00Z", TimelineID: 10, RevisionIndex: &wantRevIdx},
				},
				Spans:         []ResolvedSpan{{ID: "span-1", Label: "Window", TimelineIDs: []uint32{5}}},
				RowHighlights: []ResolvedRowHighlight{{ID: "row-1", TimelineIDs: []uint32{10}}},
				Excerpts: []ResolvedExcerptCard{
					{ID: "ex-1", RevisionIndex: &wantRevIdx, Lines: []ExcerptLine{{Kind: ExcerptLineKindAdded, Text: "image: v2", Parts: []LinePart{{Text: "v2", Matched: true}}}}},
				},
				Callouts: []ResolvedCallout{{ID: "cal-1", Label: "Probe failed", RevisionIndex: &wantRevIdx}},
			},
		},
	}

	testCases := []struct {
		name    string
		now     time.Time
		sceneID string
		want    Stored
		wantErr error
	}{
		{
			name:    "returns deep copy of unexpired scene",
			now:     issueNow.Add(10 * time.Minute),
			sceneID: issued.SceneID,
			want:    wantStored,
		},
		{
			name:    "expired entry swept during subsequent Issue returns ErrUnknownScene",
			now:     issueNow,
			sceneID: stale.SceneID,
			wantErr: ErrUnknownScene,
		},
		{
			name:    "unknown scene ID returns ErrUnknownScene",
			now:     issueNow.Add(10 * time.Minute),
			sceneID: "scn_missing",
			wantErr: ErrUnknownScene,
		},
		{
			name:    "expired scene returns ErrExpiredScene",
			now:     issueNow.Add(31 * time.Minute),
			sceneID: issued.SceneID,
			wantErr: ErrExpiredScene,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			currentNow = tc.now
			got, gotErr := store.Get(tc.sceneID)
			if tc.wantErr != nil {
				if !errors.Is(gotErr, tc.wantErr) {
					t.Fatalf("Get(%q) error = %v, want %v", tc.sceneID, gotErr, tc.wantErr)
				}
				return
			}
			if gotErr != nil {
				t.Fatalf("Get(%q) unexpected error = %v", tc.sceneID, gotErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Get(%q) mismatch (-want +got):\n%s", tc.sceneID, diff)
			}
		})
	}
}

func TestStore_MergePatchAndZeroCostBranching(t *testing.T) {
	baseNow := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	store := NewStore(30 * time.Minute)
	store.now = func() time.Time { return baseNow }

	tStart := time.Date(2026, 10, 1, 9, 18, 0, 0, time.UTC)
	tEnd := time.Date(2026, 10, 1, 9, 36, 0, 0, time.UTC)
	tZoomEnd := time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC)

	parentSpec := Spec{
		InspectionID: "insp-oom",
		Filter: workbench.Filter{
			TimelineQuery: `timelineType == "Pod"`,
			StartTime:     timeRef(tStart),
			EndTime:       timeRef(tEnd),
		},
		Width:    1600,
		MaxNodes: 50,
		Annotations: &Annotations{
			DimUnlistedRows: boolRef(true),
			Palette: []PaletteEntry{
				{ID: "trigger", Color: "amber"},
				{ID: "victim", Color: "red"},
			},
			Markers: []MarkerAnnotation{
				{Badge: "ALERT", Label: "External alert fired", Anchor: Anchor{TimelineID: 1, LogID: 1}},
				{Badge: "2", Label: "Pod OOMKilled", Anchor: Anchor{TimelineID: 1, LogID: 2}, PaletteID: "victim"},
			},
			DiffExcerpts: []DiffExcerpt{
				{Badge: "E1", Revision: RevisionRef{TimelineID: 5, RevisionIndex: 1}, Note: "Node condition transition", PaletteID: "trigger"},
			},
		},
	}
	parentScene, err := store.Issue("", parentSpec, Document{InspectionID: "insp-oom"})
	if err != nil {
		t.Fatalf("Issue(parent) unexpected error = %v", err)
	}

	expiredStore := NewStore(5 * time.Minute)
	expiredStore.now = func() time.Time { return baseNow }
	expiredParent, err := expiredStore.Issue("", parentSpec, Document{InspectionID: "insp-oom"})
	if err != nil {
		t.Fatalf("Issue(expiredParent) unexpected error = %v", err)
	}
	expiredStore.now = func() time.Time { return baseNow.Add(10 * time.Minute) }

	testCases := []struct {
		name          string
		targetStore   *Store
		parentSceneID string
		patch         Spec
		wantSpec      Spec
		wantErrResult *mcpsdk.CallToolResult
	}{
		{
			name:        "standalone scene normalizes directive-only annotations to nil",
			targetStore: store,
			patch: Spec{
				InspectionID: "insp-empty-ann",
				Annotations:  &Annotations{RemoveIDs: []string{"unused"}},
			},
			wantSpec: Spec{InspectionID: "insp-empty-ann"},
		},
		{
			name:          "branch A clears diffExcerpts and removes sole diffExcerpt and marker 2 without mutating parent",
			targetStore:   store,
			parentSceneID: parentScene.SceneID,
			patch: Spec{
				Filter: workbench.Filter{EndTime: timeRef(tZoomEnd)},
				Annotations: &Annotations{
					RemoveIDs: []string{"2", "E1", " trigger "},
					Spans: []SpanAnnotation{
						{Label: "Customer impact window", Start: Anchor{TimelineID: 1, LogID: 1}, End: Anchor{TimelineID: 1, LogID: 2}, PaletteID: "victim"},
					},
				},
			},
			wantSpec: Spec{
				InspectionID: "insp-oom",
				Filter: workbench.Filter{
					TimelineQuery: `timelineType == "Pod"`,
					StartTime:     timeRef(tStart),
					EndTime:       timeRef(tZoomEnd),
				},
				Width:    1600,
				MaxNodes: 50,
				Annotations: &Annotations{
					DimUnlistedRows: boolRef(true),
					Palette:         []PaletteEntry{{ID: "victim", Color: "red"}},
					Markers:         []MarkerAnnotation{{Badge: "ALERT", Label: "External alert fired", Anchor: Anchor{TimelineID: 1, LogID: 1}}},
					Spans:           []SpanAnnotation{{Label: "Customer impact window", Start: Anchor{TimelineID: 1, LogID: 1}, End: Anchor{TimelineID: 1, LogID: 2}, PaletteID: "victim"}},
				},
			},
		},
		{
			name:          "branch B upserts trimmed palette entry in-place and appends new entry without mutating parent",
			targetStore:   store,
			parentSceneID: parentScene.SceneID,
			patch: Spec{
				Width: 2200,
				Annotations: &Annotations{
					Palette: []PaletteEntry{
						{ID: " trigger ", Color: "teal"},
						{ID: "recovery", Color: "green"},
					},
					LogExcerpts: []LogExcerpt{{Badge: "E2", LogID: 20, Note: "Container exitCode=137", PaletteID: "victim"}},
				},
			},
			wantSpec: Spec{
				InspectionID: "insp-oom",
				Filter: workbench.Filter{
					TimelineQuery: `timelineType == "Pod"`,
					StartTime:     timeRef(tStart),
					EndTime:       timeRef(tEnd),
				},
				Width:    2200,
				MaxNodes: 50,
				Annotations: &Annotations{
					DimUnlistedRows: boolRef(true),
					Palette: []PaletteEntry{
						{ID: "trigger", Color: "teal"},
						{ID: "victim", Color: "red"},
						{ID: "recovery", Color: "green"},
					},
					Markers: []MarkerAnnotation{
						{Badge: "ALERT", Label: "External alert fired", Anchor: Anchor{TimelineID: 1, LogID: 1}},
						{Badge: "2", Label: "Pod OOMKilled", Anchor: Anchor{TimelineID: 1, LogID: 2}, PaletteID: "victim"},
					},
					DiffExcerpts: []DiffExcerpt{
						{Badge: "E1", Revision: RevisionRef{TimelineID: 5, RevisionIndex: 1}, Note: "Node condition transition", PaletteID: "trigger"},
					},
					LogExcerpts: []LogExcerpt{
						{Badge: "E2", LogID: 20, Note: "Container exitCode=137", PaletteID: "victim"},
					},
				},
			},
		},
		{
			name:          "clearLayers all normalizes empty annotations to nil",
			targetStore:   store,
			parentSceneID: parentScene.SceneID,
			patch:         Spec{Annotations: &Annotations{ClearLayers: []AnnotationLayer{AnnotationLayerAll}}},
			wantSpec: Spec{
				InspectionID: "insp-oom",
				Filter: workbench.Filter{
					TimelineQuery: `timelineType == "Pod"`,
					StartTime:     timeRef(tStart),
					EndTime:       timeRef(tEnd),
				},
				Width:    1600,
				MaxNodes: 50,
			},
		},
		{
			name:          "missing parentSceneId returns SCENE_NOT_FOUND",
			targetStore:   store,
			parentSceneID: "scn_nonexistent",
			wantErrResult: wantErrorResult(
				"SCENE_NOT_FOUND",
				"No scene with ID `scn_nonexistent` exists.",
				"Create a new scene by specifying `inspectionId` and `filter`.",
			),
		},
		{
			name:          "expired parentSceneId returns SCENE_EXPIRED",
			targetStore:   expiredStore,
			parentSceneID: expiredParent.SceneID,
			wantErrResult: wantErrorResult(
				"SCENE_EXPIRED",
				"Parent scene `"+expiredParent.SceneID+"` has expired.",
				"Create a new scene by specifying `inspectionId` and `filter`.",
			),
		},
		{
			name:          "mismatched inspectionId returns INVALID_ARGUMENT",
			targetStore:   store,
			parentSceneID: parentScene.SceneID,
			patch:         Spec{InspectionID: "insp-other"},
			wantErrResult: wantErrorResult(
				"INVALID_ARGUMENT",
				"inspectionId `insp-other` does not match parentSceneId `"+parentScene.SceneID+"` (`insp-oom`).",
				"Omit `inspectionId` when patching a parent scene, or omit `parentSceneId` to create a new scene.",
			),
		},
		{
			name:          "unknown clearLayers value returns INVALID_ARGUMENT",
			targetStore:   store,
			parentSceneID: parentScene.SceneID,
			patch:         Spec{Annotations: &Annotations{ClearLayers: []AnnotationLayer{"unknownLayer"}}},
			wantErrResult: wantErrorResult(
				"INVALID_ARGUMENT",
				"Unknown annotation layer `unknownLayer` in `annotations.clearLayers`.",
				"Supported layers: `palette`, `markers`, `spans`, `rowHighlights`, `diffExcerpts`, `logExcerpts`, `connectors`, `callouts`, `all`.",
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotSpec, gotErrRes := tc.targetStore.MergePatch(tc.parentSceneID, tc.patch)
			if tc.wantErrResult != nil {
				if diff := cmp.Diff(tc.wantErrResult, gotErrRes, ignoreMCPUnexported); diff != "" {
					t.Errorf("MergePatch() error result mismatch (-want +got):\n%s", diff)
				}
				return
			}
			if gotErrRes != nil {
				t.Fatalf("MergePatch() unexpected error result = %+v", gotErrRes)
			}
			if diff := cmp.Diff(tc.wantSpec, gotSpec); diff != "" {
				t.Errorf("MergePatch() spec mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStore_ServeSceneAndConcurrentAccess(t *testing.T) {
	baseNow := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	currentNow := baseNow
	store := NewStore(30 * time.Minute)
	store.now = func() time.Time { return currentNow }

	issued, err := store.Issue("", Spec{InspectionID: "insp-http"}, Document{
		InspectionID:  "insp-http",
		TimelineQuery: `timelineType == "Node"`,
	})
	if err != nil {
		t.Fatalf("Issue() unexpected error = %v", err)
	}

	type httpOutcome struct {
		StatusCode  int
		ContentType string
		Nosniff     string
		Doc         *Document
		ErrMessage  string
	}

	wantDoc := issued.Document
	testCases := []struct {
		name    string
		now     time.Time
		method  string
		sceneID string
		want    httpOutcome
	}{
		{
			name:    "GET returns 200 and JSON Document",
			now:     baseNow.Add(5 * time.Minute),
			method:  http.MethodGet,
			sceneID: issued.SceneID,
			want:    httpOutcome{StatusCode: http.StatusOK, ContentType: "application/json; charset=utf-8", Nosniff: "nosniff", Doc: &wantDoc},
		},
		{
			name:    "HEAD returns 200 with empty body",
			now:     baseNow.Add(5 * time.Minute),
			method:  http.MethodHead,
			sceneID: issued.SceneID,
			want:    httpOutcome{StatusCode: http.StatusOK, ContentType: "application/json; charset=utf-8", Nosniff: "nosniff"},
		},
		{
			name:    "unknown scene GET returns 404 JSON error",
			now:     baseNow.Add(5 * time.Minute),
			method:  http.MethodGet,
			sceneID: "scn_unknown",
			want:    httpOutcome{StatusCode: http.StatusNotFound, ContentType: "application/json", Nosniff: "nosniff", ErrMessage: ErrUnknownScene.Error()},
		},
		{
			name:    "unknown scene HEAD returns 404 with empty body",
			now:     baseNow.Add(5 * time.Minute),
			method:  http.MethodHead,
			sceneID: "scn_unknown",
			want:    httpOutcome{StatusCode: http.StatusNotFound, ContentType: "application/json", Nosniff: "nosniff"},
		},
		{
			name:    "expired scene GET returns 410 Gone",
			now:     baseNow.Add(35 * time.Minute),
			method:  http.MethodGet,
			sceneID: issued.SceneID,
			want:    httpOutcome{StatusCode: http.StatusGone, ContentType: "application/json", Nosniff: "nosniff", ErrMessage: ErrExpiredScene.Error()},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			currentNow = tc.now
			req := httptest.NewRequest(tc.method, "/api/v1/mcp-scenes/"+tc.sceneID, nil)
			rec := httptest.NewRecorder()
			store.ServeScene(rec, req, tc.sceneID)

			got := httpOutcome{
				StatusCode:  rec.Code,
				ContentType: rec.Header().Get("Content-Type"),
				Nosniff:     rec.Header().Get("X-Content-Type-Options"),
			}
			if rec.Body.Len() > 0 {
				if rec.Code == http.StatusOK {
					var doc Document
					if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
						t.Fatalf("json.Unmarshal(ServeScene body) error = %v", err)
					}
					got.Doc = &doc
				} else {
					var errPayload errorResponse
					if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
						t.Fatalf("json.Unmarshal(ServeScene error body) error = %v", err)
					}
					got.ErrMessage = errPayload.Error
				}
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ServeScene() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	concurrentStore := NewStore(30 * time.Minute)
	root, err := concurrentStore.Issue("", Spec{InspectionID: "insp-race"}, Document{InspectionID: "insp-race"})
	if err != nil {
		t.Fatalf("Issue(root) unexpected error = %v", err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			_, _ = concurrentStore.Issue(root.SceneID, Spec{InspectionID: "insp-race"}, Document{InspectionID: "insp-race"})
			_, _ = concurrentStore.Get(root.SceneID)
			_, _ = concurrentStore.MergePatch(root.SceneID, Spec{Width: 1800})
			rec := httptest.NewRecorder()
			concurrentStore.ServeScene(rec, httptest.NewRequest(http.MethodGet, "/api/v1/mcp-scenes/"+root.SceneID, nil), root.SceneID)
		})
	}
	wg.Wait()
}
