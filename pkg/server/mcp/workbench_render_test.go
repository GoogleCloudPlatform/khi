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

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/scene"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

const testServerBaseURL = "http://127.0.0.1:8080"

func TestRenderTimelines_Golden(t *testing.T) {
	startTime := time.Date(2026, 9, 24, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC)
	verifyTemplateGolden(t, "render_timelines.md.tmpl", renderTimelinesTemplateData{
		RenderedCount:   8,
		TotalMatched:    8,
		HasMore:         false,
		MaxAllowedNodes: maxRenderMaxNodes,
		ViewStartTime:   startTime,
		ViewEndTime:     endTime,
		OverlaySummary:  "1 marker [1], 1 deck excerpt [E1]",
		SceneID:         "scn_abcdef1234567890",
		ParentSceneID:   "scn_parent00000000",
		ImageURL:        "http://127.0.0.1:8080/api/v1/mcp-assets/ABCDEF.png",
		InteractiveURL:  "http://127.0.0.1:8080/render/mcp-timeline?sceneId=scn_abcdef1234567890",
		ExpiresAt:       time.Date(2026, 9, 24, 2, 30, 0, 0, time.UTC),
		Applied: workbench.AppliedFilter{
			TimelineQuery:               `timelineType == "Pod"`,
			LogQuery:                    `severity >= WARNING`,
			StartTime:                   startTime,
			EndTime:                     endTime,
			ExcludeTimelinesWithoutLogs: true,
		},
	}, "testdata/render_timelines.golden.md")
}

type fakeTimelineImageRenderer struct {
	gotReq *TimelineRenderRequest
	result *TimelineRenderResult
	err    error
}

var _ TimelineImageRenderer = (*fakeTimelineImageRenderer)(nil)

func (f *fakeTimelineImageRenderer) RenderTimeline(_ context.Context, req TimelineRenderRequest) (*TimelineRenderResult, error) {
	f.gotReq = &req
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeTimelineImageRenderer) Close() error {
	return nil
}

func TestWorkbenchHandler_HandleRenderTimelines(t *testing.T) {
	mgr, inspectionID := newTestWorkbenchManager(t)

	fakePNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	startTime := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	startMs := startTime.UnixMilli()
	endMs := endTime.UnixMilli()

	_, invalidCELErr := workbench.ValidateFilter(workbench.Filter{
		TimelineQuery: "invalid &&& syntax",
	})
	wantInvalidCELResult, _, _ := mdtemplate.CELErrorResult("filter.timelineQuery", "invalid &&& syntax", invalidCELErr)

	type renderTestOutcome struct {
		Result                *mcpsdk.CallToolResult
		Request               *TimelineRenderRequest
		ServedPNG             []byte
		ServedSceneInspection string
	}

	const (
		placeholderSceneID        = "scn_<SCENE_TOKEN>"
		placeholderImageURL       = testAssetBaseURL + "<TOKEN>.png"
		placeholderInteractiveURL = testServerBaseURL + "/render/mcp-timeline?sceneId=" + placeholderSceneID
	)

	testCases := []struct {
		name     string
		renderer *fakeTimelineImageRenderer
		input    RenderTimelinesInput
		want     renderTestOutcome
	}{
		{
			name: "valid render_timelines returns PNG ImageContent, sceneId, dual URLs, and Markdown TextContent",
			renderer: &fakeTimelineImageRenderer{
				result: &TimelineRenderResult{
					PNGBytes:        fakePNG,
					RenderedCount:   8,
					TotalMatched:    8,
					HasMore:         false,
					ViewStartTimeMs: startMs,
					ViewEndTimeMs:   endMs,
				},
			},
			input: RenderTimelinesInput{
				InspectionID: inspectionID,
			},
			want: renderTestOutcome{
				Result: &mcpsdk.CallToolResult{
					Content: []mcpsdk.Content{
						&mcpsdk.ImageContent{
							MIMEType: "image/png",
							Data:     fakePNG,
						},
						&mcpsdk.TextContent{
							Text: strings.Join([]string{
								"# Rendered timeline view",
								"",
								"Rendered 8 of 8 matched timelines (2026-10-01T09:00:00Z to 2026-10-01T10:00:00Z).",
								"",
								"- Scene ID: `" + placeholderSceneID + "`",
								"- Image URL: `" + placeholderImageURL + "`",
								"- Interactive URL: `" + placeholderInteractiveURL + "`",
								"- Expires at: <EXPIRES_AT>",
								"",
								"Pass `parentSceneId: \"" + placeholderSceneID + "\"` to `render_timelines` to refine or branch this scene, or open the Interactive URL in a browser to jump directly to pinned diffs and logs in KHI.",
								"Embed the static PNG in Markdown (`![Timeline](" + placeholderImageURL + ")`) or download it before it expires:",
								"",
								"```sh",
								"curl --fail-with-body -o timeline.png '" + placeholderImageURL + "'",
								"```",
								"",
								mdtemplate.FormatAppliedFilter(workbench.AppliedFilter{}),
							}, "\n"),
						},
					},
				},
				Request: &TimelineRenderRequest{
					InspectionID: inspectionID,
					MaxNodes:     defaultRenderMaxNodes,
				},
				ServedPNG:             fakePNG,
				ServedSceneInspection: inspectionID,
			},
		},
		{
			name: "render_timelines with hasMore includes narrowing hint and forwards filter parameters",
			renderer: &fakeTimelineImageRenderer{
				result: &TimelineRenderResult{
					PNGBytes:        fakePNG,
					RenderedCount:   20,
					TotalMatched:    75,
					HasMore:         true,
					ViewStartTimeMs: startMs,
					ViewEndTimeMs:   endMs,
				},
			},
			input: RenderTimelinesInput{
				InspectionID: inspectionID,
				Filter: workbench.Filter{
					TimelineQuery: `timelineType == "Pod"`,
					LogQuery:      `severity >= WARNING`,
					StartTime:     &startTime,
					EndTime:       &endTime,
				},
				MaxDepth:           3,
				MaxNodes:           20,
				TimezoneShiftHours: 9,
			},
			want: renderTestOutcome{
				Result: &mcpsdk.CallToolResult{
					Content: []mcpsdk.Content{
						&mcpsdk.ImageContent{
							MIMEType: "image/png",
							Data:     fakePNG,
						},
						&mcpsdk.TextContent{
							Text: strings.Join([]string{
								"# Rendered timeline view",
								"",
								"Rendered 20 of 75 matched timelines (2026-10-01T09:00:00Z to 2026-10-01T10:00:00Z).",
								"Narrow with `filter.timelineQuery`, `filter.logQuery`, `timelineIds`, or increase `maxNodes` (up to 150) to render omitted timelines.",
								"",
								"- Scene ID: `" + placeholderSceneID + "`",
								"- Image URL: `" + placeholderImageURL + "`",
								"- Interactive URL: `" + placeholderInteractiveURL + "`",
								"- Expires at: <EXPIRES_AT>",
								"",
								"Pass `parentSceneId: \"" + placeholderSceneID + "\"` to `render_timelines` to refine or branch this scene, or open the Interactive URL in a browser to jump directly to pinned diffs and logs in KHI.",
								"Embed the static PNG in Markdown (`![Timeline](" + placeholderImageURL + ")`) or download it before it expires:",
								"",
								"```sh",
								"curl --fail-with-body -o timeline.png '" + placeholderImageURL + "'",
								"```",
								"",
								mdtemplate.FormatAppliedFilter(workbench.AppliedFilter{
									TimelineQuery:               `timelineType == "Pod"`,
									LogQuery:                    `severity >= WARNING`,
									StartTime:                   startTime,
									EndTime:                     endTime,
									ExcludeTimelinesWithoutLogs: true,
								}),
							}, "\n"),
						},
					},
				},
				Request: &TimelineRenderRequest{
					InspectionID:       inspectionID,
					TimelineQuery:      `timelineType == "Pod"`,
					LogQuery:           `severity >= WARNING`,
					ExcludeNoLogs:      true,
					FilterStartTimeNs:  "1790845200000000000",
					FilterEndTimeNs:    "1790848800000000000",
					ViewStartTimeMs:    &startMs,
					ViewEndTimeMs:      &endMs,
					MaxDepth:           3,
					MaxNodes:           20,
					TimezoneShiftHours: 9,
				},
				ServedPNG:             fakePNG,
				ServedSceneInspection: inspectionID,
			},
		},
		{
			name:     "missing both inspectionId and parentSceneId returns INVALID_ARGUMENT",
			renderer: &fakeTimelineImageRenderer{},
			input:    RenderTimelinesInput{},
			want: renderTestOutcome{
				Result: wantErrorResult("INVALID_ARGUMENT", "Specify inspectionId to create a new scene, or parentSceneId to patch an existing scene."),
			},
		},
		{
			name:     "unknown parentSceneId returns SCENE_NOT_FOUND",
			renderer: &fakeTimelineImageRenderer{},
			input: RenderTimelinesInput{
				ParentSceneID: "scn_missing",
			},
			want: renderTestOutcome{
				Result: wantErrorResult(
					"SCENE_NOT_FOUND",
					"No scene with ID `scn_missing` exists.",
					"Create a new scene by specifying `inspectionId` and `filter`.",
				),
			},
		},
		{
			name:     "invalid CEL in filter.timelineQuery returns INVALID_CEL",
			renderer: &fakeTimelineImageRenderer{},
			input: RenderTimelinesInput{
				InspectionID: inspectionID,
				Filter: workbench.Filter{
					TimelineQuery: "invalid &&& syntax",
				},
			},
			want: renderTestOutcome{
				Result: wantInvalidCELResult,
			},
		},
		{
			name:     "endTime before startTime returns INVALID_ARGUMENT",
			renderer: &fakeTimelineImageRenderer{},
			input: RenderTimelinesInput{
				InspectionID: inspectionID,
				Filter: workbench.Filter{
					StartTime: &endTime,
					EndTime:   &startTime,
				},
			},
			want: renderTestOutcome{
				Result: wantErrorResult("INVALID_ARGUMENT", "filter.endTime must not be before filter.startTime."),
			},
		},
		{
			name:     "non-numeric timelineIds entry returns INVALID_ARGUMENT",
			renderer: &fakeTimelineImageRenderer{},
			input: RenderTimelinesInput{
				InspectionID: inspectionID,
				TimelineIDs:  []string{"pod-1"},
			},
			want: renderTestOutcome{
				Result: wantErrorResult("INVALID_ARGUMENT", "timelineIds must be a positive integer ID, got `pod-1`."),
			},
		},
		{
			name:     "unknown timelineIds entry returns TIMELINE_NOT_FOUND",
			renderer: &fakeTimelineImageRenderer{},
			input: RenderTimelinesInput{
				InspectionID: inspectionID,
				TimelineIDs:  []string{"999999"},
			},
			want: renderTestOutcome{
				Result: wantErrorResult("TIMELINE_NOT_FOUND",
					fmt.Sprintf("No timeline with ID `999999` exists in inspection `%s`.", inspectionID),
					"Use a timeline ID from `search_timelines` or `search_logs`."),
			},
		},
		{
			name: "renderer failure returns RENDER_FAILED",
			renderer: &fakeTimelineImageRenderer{
				err: errors.New("chrome process crashed"),
			},
			input: RenderTimelinesInput{InspectionID: inspectionID},
			want: renderTestOutcome{
				Result: wantErrorResult("RENDER_FAILED", "Failed to render timelines: chrome process crashed."),
				Request: &TimelineRenderRequest{
					InspectionID: inspectionID,
					MaxNodes:     defaultRenderMaxNodes,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assetStore := NewAssetStore(testAssetBaseURL, DefaultAssetTTL)
			sceneStore := scene.NewStore(scene.DefaultTTL)
			handler := NewWorkbenchHandler(mgr, tc.renderer, assetStore, sceneStore, testServerBaseURL)
			before := time.Now().UTC().Truncate(time.Second)

			res, _, err := handler.handleRenderTimelines(context.Background(), nil, tc.input)
			if err != nil {
				t.Fatalf("handleRenderTimelines() unexpected error = %v", err)
			}

			var servedPNG []byte
			var servedSceneInspection string
			if res != nil && !res.IsError && len(res.Content) >= 2 {
				if textContent, ok := res.Content[1].(*mcpsdk.TextContent); ok {
					issuedSceneID, issuedImageURL, issuedInteractiveURL, expiresAtStr := extractRenderedSceneMetadata(t, textContent.Text)
					expiresAt, parseErr := time.Parse(time.RFC3339, expiresAtStr)
					if parseErr != nil {
						t.Fatalf("Expires at %q failed RFC3339 parse: %v", expiresAtStr, parseErr)
					}
					if expiresAt.Before(before.Add(DefaultAssetTTL)) {
						t.Errorf("Expires at = %v, want at least %v", expiresAt, before.Add(DefaultAssetTTL))
					}
					wantInteractiveURL := testServerBaseURL + "/render/mcp-timeline?sceneId=" + issuedSceneID
					if issuedInteractiveURL != wantInteractiveURL {
						t.Errorf("Interactive URL = %q, want %q", issuedInteractiveURL, wantInteractiveURL)
					}

					fileName := strings.TrimPrefix(issuedImageURL, testAssetBaseURL)
					assetRec := httptest.NewRecorder()
					assetReq := httptest.NewRequest(http.MethodGet, issuedImageURL, nil)
					assetStore.ServeAsset(assetRec, assetReq, fileName)
					if assetRec.Code != http.StatusOK {
						t.Fatalf("GET %s returned %d, want %d", issuedImageURL, assetRec.Code, http.StatusOK)
					}
					servedPNG = assetRec.Body.Bytes()

					sceneRec := httptest.NewRecorder()
					sceneReq := httptest.NewRequest(http.MethodGet, testServerBaseURL+"/api/v1/mcp-scenes/"+issuedSceneID, nil)
					sceneStore.ServeScene(sceneRec, sceneReq, issuedSceneID)
					if sceneRec.Code != http.StatusOK {
						t.Fatalf("GET scene %s returned %d, want %d", issuedSceneID, sceneRec.Code, http.StatusOK)
					}
					var decodedDoc scene.Document
					if err := json.Unmarshal(sceneRec.Body.Bytes(), &decodedDoc); err != nil {
						t.Fatalf("json.Unmarshal(scene doc) failed: %v", err)
					}
					servedSceneInspection = decodedDoc.InspectionID

					normalizedText := strings.ReplaceAll(textContent.Text, issuedSceneID, placeholderSceneID)
					normalizedText = strings.ReplaceAll(normalizedText, issuedImageURL, placeholderImageURL)
					normalizedText = strings.Replace(normalizedText, "- Expires at: "+expiresAtStr, "- Expires at: <EXPIRES_AT>", 1)
					textContent.Text = normalizedText
				}
			}

			got := renderTestOutcome{
				Result:                res,
				Request:               tc.renderer.gotReq,
				ServedPNG:             servedPNG,
				ServedSceneInspection: servedSceneInspection,
			}
			if diff := cmp.Diff(tc.want, got, ignoreMCPUnexported); diff != "" {
				t.Errorf("handleRenderTimelines() outcome mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func extractRenderedSceneMetadata(t *testing.T, markdown string) (sceneID, imageURL, interactiveURL, expiresAt string) {
	t.Helper()

	extractCodeField := func(prefix string) (string, string) {
		_, after, found := strings.Cut(markdown, prefix)
		if !found {
			t.Fatalf("render_timelines markdown missing %q:\n%s", prefix, markdown)
		}
		val, rest, found := strings.Cut(after, "`")
		if !found {
			t.Fatalf("render_timelines markdown missing closing backtick for %q:\n%s", prefix, markdown)
		}
		return val, rest
	}

	sceneID, _ = extractCodeField("- Scene ID: `")
	imageURL, _ = extractCodeField("- Image URL: `")
	interactiveURL, rest := extractCodeField("- Interactive URL: `")

	const expiresPrefix = "\n- Expires at: "
	_, afterExpires, found := strings.Cut(rest, expiresPrefix)
	if !found {
		t.Fatalf("render_timelines markdown missing %q:\n%s", expiresPrefix, markdown)
	}
	expiresAt, _, _ = strings.Cut(afterExpires, "\n")
	return sceneID, imageURL, interactiveURL, expiresAt
}

func TestWorkbenchHandler_HandleRenderTimelines_StatefulPatchAndBranch(t *testing.T) {
	inspectionServer, inspectionID := setupRealisticTimelineTestServer(t)
	indexMgr := workbench.NewInspectionIndexManager(inspectionServer, t.TempDir())
	mgr := workbench.NewWorkbenchManager(inspectionServer, indexMgr, 5)
	t.Cleanup(indexMgr.Wait)

	ctx := context.Background()
	fakePNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	baseTime := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

	renderer := &fakeTimelineImageRenderer{
		result: &TimelineRenderResult{
			PNGBytes:        fakePNG,
			RenderedCount:   12,
			TotalMatched:    12,
			HasMore:         false,
			ViewStartTimeMs: baseTime.UnixMilli(),
			ViewEndTimeMs:   baseTime.Add(1 * time.Hour).UnixMilli(),
		},
	}
	assetStore := NewAssetStore(testAssetBaseURL, DefaultAssetTTL)
	sceneStore := scene.NewStore(scene.DefaultTTL)
	handler := NewWorkbenchHandler(mgr, renderer, assetStore, sceneStore, testServerBaseURL)

	wb, toolRes, err := handler.acquireWorkbench(ctx, inspectionID)
	if err != nil || toolRes != nil {
		t.Fatalf("acquireWorkbench() failed: err=%v toolRes=%v", err, toolRes)
	}

	var nodeReadyTimelineID uint32
	var appTimelineID uint32
	var oomLogID uint32
	for id, tl := range wb.SearchIndex().TimelineMap {
		if tl != nil && tl.Name == "Ready" {
			nodeReadyTimelineID = uint32(id)
		}
		if tl != nil && tl.Name == "app" {
			page, err := wb.GetTimelineLogs(ctx, uint32(id), workbench.TimelineLogFilter{LogQuery: `severity == ERROR`})
			if err == nil && len(page.Logs) > 0 {
				appTimelineID = uint32(id)
				oomLogID = page.Logs[0].LogID
			}
		}
	}
	if nodeReadyTimelineID == 0 || appTimelineID == 0 || oomLogID == 0 {
		t.Fatalf("failed to locate fixture IDs: nodeReadyTimelineID=%d appTimelineID=%d oomLogID=%d", nodeReadyTimelineID, appTimelineID, oomLogID)
	}

	// Step 1: Create initial annotated parent scene.
	dimUnlisted := true
	parentRes, _, err := handler.handleRenderTimelines(ctx, nil, RenderTimelinesInput{
		InspectionID: inspectionID,
		Filter: workbench.Filter{
			TimelineQuery: `path["namespace"] == "default"`,
		},
		Annotations: &scene.Annotations{
			Palette: []scene.PaletteEntry{
				{ID: "trigger", Color: "amber"},
				{ID: "victim", Color: "red"},
			},
			RowHighlights: []scene.RowHighlight{
				{
					TimelineQuery: `name == "frontend-web-6d8f9b-x7k2p"`,
					RoleLabel:     "VICTIM · Pod-1",
					PaletteID:     "victim",
				},
			},
			DimUnlistedRows: &dimUnlisted,
			Markers: []scene.MarkerAnnotation{
				{
					Anchor: scene.Anchor{TimelineID: appTimelineID, LogID: oomLogID},
					Label:  "OOMKilled container app",
				},
			},
		},
	})
	if err != nil || parentRes.IsError {
		t.Fatalf("Step 1 handleRenderTimelines() failed: err=%v res=%v", err, parentRes)
	}
	parentText := parentRes.Content[1].(*mcpsdk.TextContent).Text
	parentSceneID, _, _, _ := extractRenderedSceneMetadata(t, parentText)
	if !strings.Contains(parentText, "Annotations: 1 marker [1], 1 row highlight, unlisted rows dimmed.") {
		t.Errorf("Step 1 summary missing annotation summary:\n%s", parentText)
	}

	// Step 2: Incrementally patch parentSceneId with diffExcerpts + logExcerpts.
	childARes, _, err := handler.handleRenderTimelines(ctx, nil, RenderTimelinesInput{
		ParentSceneID: parentSceneID,
		Annotations: &scene.Annotations{
			DiffExcerpts: []scene.DiffExcerpt{
				{
					Revision: scene.RevisionRef{
						TimelineID:    nodeReadyTimelineID,
						RevisionIndex: 1,
					},
					Windows: []scene.SliceWindow{
						{Query: "MemoryPressure", BeforeLines: 1, AfterLines: 1},
					},
					Note: "Node Ready condition flipped to False",
				},
			},
			LogExcerpts: []scene.LogExcerpt{
				{
					LogID:      oomLogID,
					TimelineID: appTimelineID,
					Note:       "Container OOM termination event",
				},
			},
		},
	})
	if err != nil || childARes.IsError {
		t.Fatalf("Step 2 handleRenderTimelines() failed: err=%v res=%v", err, childARes)
	}
	childAText := childARes.Content[1].(*mcpsdk.TextContent).Text
	childASceneID, _, _, _ := extractRenderedSceneMetadata(t, childAText)
	if !strings.Contains(childAText, "(derived from `"+parentSceneID+"`)") {
		t.Errorf("Step 2 summary missing parent lineage:\n%s", childAText)
	}
	if len(renderer.gotReq.Overlay.Excerpts) != 2 {
		t.Fatalf("Step 2 Excerpts len = %d, want 2", len(renderer.gotReq.Overlay.Excerpts))
	}
	if renderer.gotReq.Overlay.Excerpts[1].PaletteID != "victim" {
		t.Errorf("Step 2 co-anchored log excerpt PaletteID = %q, want \"victim\"", renderer.gotReq.Overlay.Excerpts[1].PaletteID)
	}
	for _, line := range renderer.gotReq.Overlay.Excerpts[0].Lines {
		if strings.Contains(line.Text, "managedFields") {
			t.Errorf("Step 2 diff excerpt leaked managedFields: %q", line.Text)
		}
	}

	// Step 3: Branch a second variant from the same parentSceneId without mutating parent or childA.
	spanStart := baseTime.Add(24 * time.Minute)
	spanEnd := baseTime.Add(29*time.Minute + 30*time.Second)
	childBRes, _, err := handler.handleRenderTimelines(ctx, nil, RenderTimelinesInput{
		ParentSceneID: parentSceneID,
		Annotations: &scene.Annotations{
			ClearLayers: []scene.AnnotationLayer{scene.AnnotationLayerMarkers},
			Spans: []scene.SpanAnnotation{
				{
					Start:     scene.Anchor{Time: &spanStart},
					End:       scene.Anchor{Time: &spanEnd},
					Label:     "Customer impact window",
					Pattern:   scene.SpanPatternHatched,
					PaletteID: "trigger",
				},
			},
		},
	})
	if err != nil || childBRes.IsError {
		t.Fatalf("Step 3 handleRenderTimelines() failed: err=%v res=%v", err, childBRes)
	}
	if len(renderer.gotReq.Overlay.Markers) != 0 || len(renderer.gotReq.Overlay.Spans) != 1 || len(renderer.gotReq.Overlay.Excerpts) != 0 {
		t.Errorf("Step 3 branched overlay mismatch: markers=%d spans=%d excerpts=%d",
			len(renderer.gotReq.Overlay.Markers), len(renderer.gotReq.Overlay.Spans), len(renderer.gotReq.Overlay.Excerpts))
	}

	// Confirm parent and childA remain untouched in sceneStore.
	storedParent, err := sceneStore.Get(parentSceneID)
	if err != nil || len(storedParent.Document.Overlay.Markers) != 1 || len(storedParent.Document.Overlay.Excerpts) != 0 {
		t.Errorf("storedParent mutated unexpectedly: err=%v overlay=%+v", err, storedParent.Document.Overlay)
	}
	storedChildA, err := sceneStore.Get(childASceneID)
	if err != nil || len(storedChildA.Document.Overlay.Excerpts) != 2 {
		t.Errorf("storedChildA mutated unexpectedly: err=%v overlay=%+v", err, storedChildA.Document.Overlay)
	}

	// Step 4: Verify actionable error codes (SLICE_QUERY_NOT_FOUND and ANCHOR_OUT_OF_VIEWPORT).
	errSliceRes, _, err := handler.handleRenderTimelines(ctx, nil, RenderTimelinesInput{
		ParentSceneID: parentSceneID,
		Annotations: &scene.Annotations{
			DiffExcerpts: []scene.DiffExcerpt{
				{
					Revision: scene.RevisionRef{TimelineID: nodeReadyTimelineID, RevisionIndex: 1},
					Windows:  []scene.SliceWindow{{Query: "NonExistentFieldXYZ"}},
				},
			},
		},
	})
	if err != nil || !errSliceRes.IsError {
		t.Fatalf("SLICE_QUERY_NOT_FOUND check failed: err=%v res=%v", err, errSliceRes)
	}
	if gotErrText := errSliceRes.Content[0].(*mcpsdk.TextContent).Text; !strings.HasPrefix(gotErrText, "Error: SLICE_QUERY_NOT_FOUND") {
		t.Errorf("expected SLICE_QUERY_NOT_FOUND, got:\n%s", gotErrText)
	}

	narrowStart := baseTime.Add(40 * time.Minute)
	narrowEnd := baseTime.Add(50 * time.Minute)
	errViewportRes, _, err := handler.handleRenderTimelines(ctx, nil, RenderTimelinesInput{
		ParentSceneID: parentSceneID,
		Filter: workbench.Filter{
			StartTime: &narrowStart,
			EndTime:   &narrowEnd,
		},
	})
	if err != nil || !errViewportRes.IsError {
		t.Fatalf("ANCHOR_OUT_OF_VIEWPORT check failed: err=%v res=%v", err, errViewportRes)
	}
	if gotErrText := errViewportRes.Content[0].(*mcpsdk.TextContent).Text; !strings.HasPrefix(gotErrText, "Error: ANCHOR_OUT_OF_VIEWPORT") {
		t.Errorf("expected ANCHOR_OUT_OF_VIEWPORT, got:\n%s", gotErrText)
	}
}

func TestBuildTimelineRenderRequest(t *testing.T) {
	startTime := time.Unix(100, 123456789).UTC()
	endTime := time.Unix(200, 987654321).UTC()
	startMs, endMs := int64(100123), int64(200987)

	testCases := []struct {
		name          string
		spec          scene.Spec
		applied       workbench.AppliedFilter
		parentSceneID string
		overlay       *scene.Overlay
		want          TimelineRenderRequest
	}{
		{
			name: "zero values use defaults",
			want: TimelineRenderRequest{MaxNodes: 50},
		},
		{
			name: "above maxNodes cap is capped",
			spec: scene.Spec{MaxNodes: 151},
			want: TimelineRenderRequest{MaxNodes: 150},
		},
		{
			name: "negative depth becomes zero",
			spec: scene.Spec{MaxDepth: -1},
			want: TimelineRenderRequest{MaxNodes: 50},
		},
		{
			name:    "only start time is set",
			applied: workbench.AppliedFilter{StartTime: startTime},
			want: TimelineRenderRequest{
				MaxNodes:          50,
				ViewStartTimeMs:   &startMs,
				FilterStartTimeNs: "100123456789",
			},
		},
		{
			name:    "only end time is set",
			applied: workbench.AppliedFilter{EndTime: endTime},
			want: TimelineRenderRequest{
				MaxNodes:        50,
				ViewEndTimeMs:   &endMs,
				FilterEndTimeNs: "200987654321",
			},
		},
		{
			name: "forwards in-range limits, scene parameters, and both time bounds",
			spec: scene.Spec{
				InspectionID:       "inspection-1",
				TimelineIDs:        []uint32{1, 2},
				MaxNodes:           20,
				MaxDepth:           3,
				TimezoneShiftHours: 5.5,
			},
			applied: workbench.AppliedFilter{
				TimelineQuery:               `timelineType == "Pod"`,
				TimelineExclusionQuery:      `timelineType == "Node"`,
				LogQuery:                    "severity >= WARNING",
				ExcludeTimelinesWithoutLogs: true,
				StartTime:                   startTime,
				EndTime:                     endTime,
			},
			parentSceneID: "parent-scene",
			overlay:       &scene.Overlay{DimUnlistedRows: true},
			want: TimelineRenderRequest{
				ParentSceneID:          "parent-scene",
				Overlay:                &scene.Overlay{DimUnlistedRows: true},
				InspectionID:           "inspection-1",
				TimelineQuery:          `timelineType == "Pod"`,
				TimelineExclusionQuery: `timelineType == "Node"`,
				LogQuery:               "severity >= WARNING",
				ExcludeNoLogs:          true,
				TimelineIDs:            []uint32{1, 2},
				MaxNodes:               20,
				MaxDepth:               3,
				TimezoneShiftHours:     5.5,
				ViewStartTimeMs:        &startMs,
				ViewEndTimeMs:          &endMs,
				FilterStartTimeNs:      "100123456789",
				FilterEndTimeNs:        "200987654321",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildTimelineRenderRequest(tc.parentSceneID, tc.spec, tc.applied, tc.overlay)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("buildTimelineRenderRequest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
