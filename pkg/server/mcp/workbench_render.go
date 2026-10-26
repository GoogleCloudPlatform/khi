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
	"fmt"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultRenderMaxNodes = 50
	maxRenderMaxNodes     = 150
	defaultRenderWidth    = 1600
	minRenderWidth        = 800
	maxRenderWidth        = 3840
)

// RenderTimelinesInput defines the input parameters for the render_timelines MCP tool.
type RenderTimelinesInput struct {
	InspectionID       string           `json:"inspectionId" jsonschema:"The target inspection ID."`
	Filter             workbench.Filter `json:"filter,omitempty" jsonschema:"CEL and time range filter parameters. When startTime and endTime are omitted, the entire inspection time range is rendered. excludeTimelinesWithoutLogs defaults to true when logQuery is non-empty and false otherwise."`
	TimelineIDs        []string         `json:"timelineIds,omitempty" jsonschema:"Optional list of timeline IDs (from search_timelines or search_logs) to render along with their ancestor hierarchy nodes."`
	MaxDepth           int              `json:"maxDepth,omitempty" jsonschema:"Maximum tree depth from root timelines (0-based). 0 means unlimited."`
	MaxNodes           int              `json:"maxNodes,omitempty" jsonschema:"Maximum number of timeline rows to render (default 50, max 150)."`
	Width              int              `json:"width,omitempty" jsonschema:"Rendered image width in pixels (default 1600, min 800, max 3840)."`
	TimezoneShiftHours float64          `json:"timezoneShiftHours,omitempty" jsonschema:"Timezone offset in hours from UTC for the timeline ruler (default 0)."`
}

type renderTimelinesTemplateData struct {
	RenderedCount   int
	TotalMatched    int
	HasMore         bool
	MaxAllowedNodes int
	Width           int
	Height          int
	ViewStartTime   time.Time
	ViewEndTime     time.Time
	Applied         workbench.AppliedFilter
}

func (h *WorkbenchHandler) handleRenderTimelines(ctx context.Context, _ *mcpsdk.CallToolRequest, input RenderTimelinesInput) (*mcpsdk.CallToolResult, any, error) {
	if input.InspectionID == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "inspectionId is required.")
	}
	if field, err := workbench.ValidateFilter(input.Filter); err != nil {
		return mdtemplate.CELErrorResult(field, filterExpressionByField(input.Filter, field), err)
	}
	if input.Filter.StartTime != nil && input.Filter.EndTime != nil && input.Filter.EndTime.Before(*input.Filter.StartTime) {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "filter.endTime must not be before filter.startTime.")
	}

	parsedTimelineIDs, errRes, err := parseTimelineIDArguments(input.TimelineIDs)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	wb, errRes, err := h.acquireWorkbench(ctx, input.InspectionID)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	if errRes, ok := verifyTimelinesExist(wb, input.InspectionID, input.TimelineIDs, parsedTimelineIDs); !ok {
		return errRes, nil, nil
	}

	applied := wb.ResolveFilter(input.Filter)
	renderReq := buildTimelineRenderRequest(input, applied, parsedTimelineIDs)

	renderRes, err := h.renderer.RenderTimeline(ctx, renderReq)
	if err != nil {
		return mdtemplate.ErrorResult("RENDER_FAILED", fmt.Sprintf("Failed to render timelines: %s.", err.Error()))
	}

	summaryText, err := h.templates.Render("render_timelines.md.tmpl", renderTimelinesTemplateData{
		RenderedCount:   renderRes.RenderedCount,
		TotalMatched:    renderRes.TotalMatched,
		HasMore:         renderRes.HasMore,
		MaxAllowedNodes: maxRenderMaxNodes,
		Width:           renderRes.Width,
		Height:          renderRes.Height,
		ViewStartTime:   time.UnixMilli(renderRes.ViewStartTimeMs).UTC(),
		ViewEndTime:     time.UnixMilli(renderRes.ViewEndTimeMs).UTC(),
		Applied:         applied,
	})
	if err != nil {
		return nil, nil, err
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.ImageContent{
				MIMEType: "image/png",
				Data:     renderRes.PNGBytes,
			},
			&mcpsdk.TextContent{
				Text: summaryText,
			},
		},
	}, nil, nil
}

func parseTimelineIDArguments(rawIDs []string) ([]uint32, *mcpsdk.CallToolResult, error) {
	if len(rawIDs) == 0 {
		return nil, nil, nil
	}
	parsed := make([]uint32, len(rawIDs))
	for i, rawID := range rawIDs {
		id, ok := parseIDArgument(rawID)
		if !ok {
			res, _, err := invalidIDResult("timelineIds", rawID)
			return nil, res, err
		}
		parsed[i] = id
	}
	return parsed, nil, nil
}

func verifyTimelinesExist(wb *workbench.Workbench, inspectionID string, rawIDs []string, parsedIDs []uint32) (*mcpsdk.CallToolResult, bool) {
	if len(parsedIDs) == 0 {
		return nil, true
	}
	idx := wb.SearchIndex()
	for i, id := range parsedIDs {
		if idx == nil || idx.TimelineMap[id] == nil {
			res, _, _ := timelineNotFoundResult(inspectionID, rawIDs[i])
			return res, false
		}
	}
	return nil, true
}

func buildTimelineRenderRequest(input RenderTimelinesInput, applied workbench.AppliedFilter, timelineIDs []uint32) TimelineRenderRequest {
	maxNodes := input.MaxNodes
	if maxNodes <= 0 {
		maxNodes = defaultRenderMaxNodes
	}
	maxNodes = min(maxNodes, maxRenderMaxNodes)

	width := input.Width
	if width <= 0 {
		width = defaultRenderWidth
	}
	width = max(minRenderWidth, min(width, maxRenderWidth))

	req := TimelineRenderRequest{
		InspectionID:           input.InspectionID,
		TimelineQuery:          applied.TimelineQuery,
		TimelineExclusionQuery: applied.TimelineExclusionQuery,
		LogQuery:               applied.LogQuery,
		ExcludeNoLogs:          applied.ExcludeTimelinesWithoutLogs,
		TimelineIDs:            timelineIDs,
		MaxDepth:               max(0, input.MaxDepth),
		MaxNodes:               maxNodes,
		Width:                  width,
		TimezoneShiftHours:     input.TimezoneShiftHours,
	}

	if !applied.StartTime.IsZero() {
		startMs := applied.StartTime.UnixMilli()
		req.ViewStartTimeMs = &startMs
		req.FilterStartTimeNs = strconv.FormatInt(applied.StartTime.UnixNano(), 10)
	}
	if !applied.EndTime.IsZero() {
		endMs := applied.EndTime.UnixMilli()
		req.ViewEndTimeMs = &endMs
		req.FilterEndTimeNs = strconv.FormatInt(applied.EndTime.UnixNano(), 10)
	}
	return req
}
