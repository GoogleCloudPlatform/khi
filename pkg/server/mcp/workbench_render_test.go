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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeTimelineImageRenderer struct {
	gotReq *TimelineRenderRequest
	result *TimelineRenderResult
	err    error
}

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
	server, inspectionID := setupTestInspectionServer(t)
	indexMgr := workbench.NewInspectionIndexManager(server, t.TempDir())
	mgr := workbench.NewWorkbenchManager(server, indexMgr, 5)

	t.Cleanup(func() {
		indexMgr.Wait()
	})

	fakePNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	startTime := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	startMs := startTime.UnixMilli()
	endMs := endTime.UnixMilli()

	_, invalidCELErr := workbench.ValidateFilter(workbench.Filter{
		TimelineQuery: "invalid &&& syntax",
	})
	wantInvalidCELResult, _, _ := mdtemplate.CELErrorResult("filter.timelineQuery", "invalid &&& syntax", invalidCELErr)

	testCases := []struct {
		name       string
		renderer   *fakeTimelineImageRenderer
		input      RenderTimelinesInput
		wantResult *mcpsdk.CallToolResult
		wantReq    *TimelineRenderRequest
	}{
		{
			name: "valid render_timelines returns PNG ImageContent and Markdown TextContent",
			renderer: &fakeTimelineImageRenderer{
				result: &TimelineRenderResult{
					PNGBytes:        fakePNG,
					Width:           1600,
					Height:          460,
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
			wantResult: &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.ImageContent{
						MIMEType: "image/png",
						Data:     fakePNG,
					},
					&mcpsdk.TextContent{
						Text: "# Rendered timeline view\n\nRendered 8 of 8 matched timelines (1600x460 px, 2026-10-01T09:00:00Z to 2026-10-01T10:00:00Z).\n\n## Applied filter\nEnter these values in the KHI Web UI filter to see the same data.\n\n```yaml\ntimelineQuery: \"\"\ntimelineExclusionQuery: \"\"\nlogQuery: \"\"\nstartTime: \"\"\nendTime: \"\"\nexcludeTimelinesWithoutLogs: false\n```",
					},
				},
			},
			wantReq: &TimelineRenderRequest{
				InspectionID: inspectionID,
				MaxNodes:     defaultRenderMaxNodes,
				Width:        defaultRenderWidth,
			},
		},
		{
			name: "render_timelines with hasMore includes narrowing hint and forwards filter parameters",
			renderer: &fakeTimelineImageRenderer{
				result: &TimelineRenderResult{
					PNGBytes:        fakePNG,
					Width:           1200,
					Height:          800,
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
				Width:              1200,
				TimezoneShiftHours: 9,
			},
			wantResult: &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.ImageContent{
						MIMEType: "image/png",
						Data:     fakePNG,
					},
					&mcpsdk.TextContent{
						Text: "# Rendered timeline view\n\nRendered 20 of 75 matched timelines (1200x800 px, 2026-10-01T09:00:00Z to 2026-10-01T10:00:00Z).\nNarrow with `filter.timelineQuery`, `filter.logQuery`, `timelineIds`, or increase `maxNodes` (up to 150) to render omitted timelines.\n\n## Applied filter\nEnter these values in the KHI Web UI filter to see the same data.\n\n```yaml\ntimelineQuery: |-\n  timelineType == \"Pod\"\ntimelineExclusionQuery: \"\"\nlogQuery: |-\n  severity >= WARNING\nstartTime: 2026-10-01T09:00:00Z\nendTime: 2026-10-01T10:00:00Z\nexcludeTimelinesWithoutLogs: true\n```",
					},
				},
			},
			wantReq: &TimelineRenderRequest{
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
				Width:              1200,
				TimezoneShiftHours: 9,
			},
		},
		{
			name:     "missing inspectionId returns INVALID_ARGUMENT",
			renderer: &fakeTimelineImageRenderer{},
			input:    RenderTimelinesInput{},
			wantResult: &mcpsdk.CallToolResult{
				IsError: true,
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{
						Text: mdtemplate.FormatError("INVALID_ARGUMENT", "inspectionId is required."),
					},
				},
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
			wantResult: wantInvalidCELResult,
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
			wantResult: &mcpsdk.CallToolResult{
				IsError: true,
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{
						Text: mdtemplate.FormatError("INVALID_ARGUMENT", "filter.endTime must not be before filter.startTime."),
					},
				},
			},
		},
		{
			name:     "non-numeric timelineIds entry returns INVALID_ARGUMENT",
			renderer: &fakeTimelineImageRenderer{},
			input: RenderTimelinesInput{
				InspectionID: inspectionID,
				TimelineIDs:  []string{"pod-1"},
			},
			wantResult: &mcpsdk.CallToolResult{
				IsError: true,
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{
						Text: mdtemplate.FormatError("INVALID_ARGUMENT", "timelineIds must be a positive integer ID, got `pod-1`."),
					},
				},
			},
		},
		{
			name:     "unknown timelineIds entry returns TIMELINE_NOT_FOUND",
			renderer: &fakeTimelineImageRenderer{},
			input: RenderTimelinesInput{
				InspectionID: inspectionID,
				TimelineIDs:  []string{"999999"},
			},
			wantResult: &mcpsdk.CallToolResult{
				IsError: true,
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{
						Text: mdtemplate.FormatError("TIMELINE_NOT_FOUND",
							fmt.Sprintf("No timeline with ID `999999` exists in inspection `%s`.", inspectionID),
							"Use a timeline ID from `search_timelines` or `search_logs`."),
					},
				},
			},
		},
		{
			name: "renderer failure returns RENDER_FAILED",
			renderer: &fakeTimelineImageRenderer{
				err: errors.New("chrome process crashed"),
			},
			input: RenderTimelinesInput{InspectionID: inspectionID},
			wantResult: &mcpsdk.CallToolResult{
				IsError: true,
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{
						Text: mdtemplate.FormatError("RENDER_FAILED", "Failed to render timelines: chrome process crashed."),
					},
				},
			},
			wantReq: &TimelineRenderRequest{
				InspectionID: inspectionID,
				MaxNodes:     defaultRenderMaxNodes,
				Width:        defaultRenderWidth,
			},
		},
	}

	cmpOpt := cmpopts.IgnoreUnexported(
		mcpsdk.CallToolResult{},
		mcpsdk.ImageContent{},
		mcpsdk.TextContent{},
	)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewWorkbenchHandler(mgr, tc.renderer)

			res, _, err := handler.handleRenderTimelines(context.Background(), nil, tc.input)
			if err != nil {
				t.Fatalf("handleRenderTimelines() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.wantResult, res, cmpOpt); diff != "" {
				t.Errorf("CallToolResult mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantReq, tc.renderer.gotReq); diff != "" {
				t.Errorf("TimelineRenderRequest mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
