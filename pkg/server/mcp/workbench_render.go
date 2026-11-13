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
	"net/url"
	"strconv"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/scene"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

const (
	defaultRenderMaxNodes = 50
	maxRenderMaxNodes     = 150
)

// RenderTimelinesInput defines the input parameters for the render_timelines MCP tool.
type RenderTimelinesInput struct {
	// InspectionID is the unique identifier of the target inspection when creating a new scene.
	InspectionID string `json:"inspectionId,omitempty" jsonschema:"The target inspection ID. Required when creating a new scene; inherited automatically when parentSceneId is specified."`
	// ParentSceneID references an existing scene whose settings and annotations are cloned before applying patches.
	ParentSceneID string `json:"parentSceneId,omitempty" jsonschema:"Optional parent scene ID from a previous render_timelines call. Clones its inspectionId, filter, timelineIds, and annotations into a new immutable scene before applying overrides."`
	// Filter specifies CEL and time range filters for the rendered timelines and logs.
	Filter workbench.Filter `json:"filter,omitempty" jsonschema:"CEL and time range filter parameters. Non-empty fields override the inherited filter when parentSceneId is set. When startTime and endTime are omitted, the entire inspection time range is rendered. excludeTimelinesWithoutLogs defaults to true when logQuery is non-empty and false otherwise."`
	// TimelineIDs restricts rendering to the specified timeline IDs and their ancestor hierarchy nodes.
	TimelineIDs []string `json:"timelineIds,omitempty" jsonschema:"Optional list of timeline IDs (from search_timelines or search_logs) to render along with their ancestor hierarchy nodes. Timelines referenced by annotations are also included automatically."`
	// MaxDepth limits the rendered timeline tree depth (0 means unlimited).
	MaxDepth int `json:"maxDepth,omitempty" jsonschema:"Maximum tree depth from root timelines (0-based). 0 means unlimited."`
	// MaxNodes caps the number of timeline rows rendered.
	MaxNodes int `json:"maxNodes,omitempty" jsonschema:"Maximum number of timeline rows to render (default 50, max 150)."`
	// TimezoneShiftHours specifies the UTC offset in hours for the timeline ruler.
	TimezoneShiftHours float64 `json:"timezoneShiftHours,omitempty" jsonschema:"Timezone offset in hours from UTC for the timeline ruler (default 0)."`
	// Annotations specifies visual overlays, evidence deck cards, and patch directives for the scene.
	Annotations *scene.Annotations `json:"annotations,omitempty" jsonschema:"Optional named palette, visual overlays (markers, spans, rowHighlights, connectors, callouts), and bottom-deck evidence cards (diffExcerpts, logExcerpts), plus clearLayers and removeIds patch directives."`
}

type renderTimelinesTemplateData struct {
	RenderedCount   int
	TotalMatched    int
	HasMore         bool
	MaxAllowedNodes int
	ViewStartTime   time.Time
	ViewEndTime     time.Time
	OverlaySummary  string
	SceneID         string
	ParentSceneID   string
	ImageURL        string
	InteractiveURL  string
	ExpiresAt       time.Time
	Applied         workbench.AppliedFilter
}

func (h *WorkbenchHandler) handleRenderTimelines(ctx context.Context, _ *mcpsdk.CallToolRequest, input RenderTimelinesInput) (*mcpsdk.CallToolResult, any, error) {
	if input.ParentSceneID == "" && input.InspectionID == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "Specify inspectionId to create a new scene, or parentSceneId to patch an existing scene.")
	}

	parsedInputIDs, errRes, err := parseTimelineIDArguments(input.TimelineIDs)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	mergedSpec, errRes := h.sceneStore.MergePatch(input.ParentSceneID, scene.Spec{
		InspectionID:       input.InspectionID,
		Filter:             input.Filter,
		TimelineIDs:        parsedInputIDs,
		MaxDepth:           input.MaxDepth,
		MaxNodes:           input.MaxNodes,
		TimezoneShiftHours: input.TimezoneShiftHours,
		Annotations:        input.Annotations,
	})
	if errRes != nil {
		return errRes, nil, nil
	}

	if field, err := workbench.ValidateFilter(mergedSpec.Filter); err != nil {
		return mdtemplate.CELErrorResult(field, filterExpressionByField(mergedSpec.Filter, field), err)
	}
	if mergedSpec.Filter.StartTime != nil && mergedSpec.Filter.EndTime != nil && mergedSpec.Filter.EndTime.Before(*mergedSpec.Filter.StartTime) {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "filter.endTime must not be before filter.startTime.")
	}

	wb, errRes, err := h.acquireWorkbench(ctx, mergedSpec.InspectionID)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	if errRes, ok := verifyParsedTimelinesExist(wb, mergedSpec.InspectionID, mergedSpec.TimelineIDs); !ok {
		return errRes, nil, nil
	}

	applied := wb.ResolveFilter(mergedSpec.Filter)
	overlay, errRes, err := scene.Resolve(ctx, wb, mergedSpec.InspectionID, &mergedSpec, applied)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	renderReq := buildTimelineRenderRequest(input.ParentSceneID, mergedSpec, applied, overlay)
	renderRes, err := h.renderer.RenderTimeline(ctx, renderReq)
	if err != nil {
		return mdtemplate.ErrorResult("RENDER_FAILED", fmt.Sprintf("Failed to render timelines: %s.", err.Error()))
	}

	grant := h.assetStore.Issue(renderRes.PNGBytes, "image/png", ".png")
	storedScene, err := h.sceneStore.Issue(input.ParentSceneID, mergedSpec, toSceneDocument(renderReq))
	if err != nil {
		return mdtemplate.ErrorResult("INTERNAL_ERROR", fmt.Sprintf("Failed to issue timeline scene: %s.", err.Error()))
	}

	expiresAt := grant.ExpiresAt
	if storedScene.ExpiresAt.Before(expiresAt) {
		expiresAt = storedScene.ExpiresAt
	}
	interactiveURL := ""
	if h.baseURL != "" {
		interactiveURL = fmt.Sprintf("%s/render/mcp-timeline?sceneId=%s", h.baseURL, url.QueryEscape(storedScene.SceneID))
	}

	summaryText, err := h.templates.Render("render_timelines.md.tmpl", renderTimelinesTemplateData{
		RenderedCount:   renderRes.RenderedCount,
		TotalMatched:    renderRes.TotalMatched,
		HasMore:         renderRes.HasMore,
		MaxAllowedNodes: maxRenderMaxNodes,
		ViewStartTime:   time.UnixMilli(renderRes.ViewStartTimeMs).UTC(),
		ViewEndTime:     time.UnixMilli(renderRes.ViewEndTimeMs).UTC(),
		OverlaySummary:  formatOverlaySummary(overlay),
		SceneID:         storedScene.SceneID,
		ParentSceneID:   storedScene.ParentSceneID,
		ImageURL:        grant.URL,
		InteractiveURL:  interactiveURL,
		ExpiresAt:       expiresAt,
		Applied:         applied,
	})
	if err != nil {
		return mdtemplate.ErrorResult("INTERNAL_ERROR", fmt.Sprintf("Failed to render render_timelines template: %s.", err.Error()))
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

func verifyParsedTimelinesExist(wb *workbench.Workbench, inspectionID string, parsedIDs []uint32) (*mcpsdk.CallToolResult, bool) {
	if len(parsedIDs) == 0 {
		return nil, true
	}
	idx := wb.SearchIndex()
	for _, id := range parsedIDs {
		if idx == nil || idx.TimelineMap[id] == nil {
			res, _, _ := timelineNotFoundResult(inspectionID, strconv.FormatUint(uint64(id), 10))
			return res, false
		}
	}
	return nil, true
}

func buildTimelineRenderRequest(
	parentSceneID string,
	spec scene.Spec,
	applied workbench.AppliedFilter,
	overlay *scene.Overlay,
) TimelineRenderRequest {
	maxNodes := spec.MaxNodes
	if maxNodes <= 0 {
		maxNodes = defaultRenderMaxNodes
	}
	maxNodes = min(maxNodes, maxRenderMaxNodes)

	req := TimelineRenderRequest{
		ParentSceneID:          parentSceneID,
		InspectionID:           spec.InspectionID,
		TimelineQuery:          applied.TimelineQuery,
		TimelineExclusionQuery: applied.TimelineExclusionQuery,
		LogQuery:               applied.LogQuery,
		ExcludeNoLogs:          applied.ExcludeTimelinesWithoutLogs,
		TimelineIDs:            spec.TimelineIDs,
		MaxDepth:               max(0, spec.MaxDepth),
		MaxNodes:               maxNodes,
		TimezoneShiftHours:     spec.TimezoneShiftHours,
		Overlay:                overlay,
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

func toSceneDocument(req TimelineRenderRequest) scene.Document {
	return scene.Document{
		InspectionID:           req.InspectionID,
		TimelineQuery:          req.TimelineQuery,
		TimelineExclusionQuery: req.TimelineExclusionQuery,
		LogQuery:               req.LogQuery,
		ExcludeNoLogs:          req.ExcludeNoLogs,
		FilterStartTimeNs:      req.FilterStartTimeNs,
		FilterEndTimeNs:        req.FilterEndTimeNs,
		ViewStartTimeMs:        req.ViewStartTimeMs,
		ViewEndTimeMs:          req.ViewEndTimeMs,
		TimelineIDs:            req.TimelineIDs,
		MaxDepth:               req.MaxDepth,
		MaxNodes:               req.MaxNodes,
		TimezoneShiftHours:     req.TimezoneShiftHours,
		Overlay:                req.Overlay,
	}
}

func formatOverlaySummary(overlay *scene.Overlay) string {
	if overlay == nil {
		return ""
	}
	var parts []string
	if n := len(overlay.Markers); n > 0 {
		badges := make([]string, n)
		for i, m := range overlay.Markers {
			badges[i] = m.Badge
		}
		parts = append(parts, fmt.Sprintf("%s [%s]", pluralize(n, "marker", "markers"), strings.Join(badges, ", ")))
	}
	if n := len(overlay.Spans); n > 0 {
		parts = append(parts, pluralize(n, "span", "spans"))
	}
	if n := len(overlay.RowHighlights); n > 0 {
		parts = append(parts, pluralize(n, "row highlight", "row highlights"))
	}
	if n := len(overlay.Excerpts); n > 0 {
		badges := make([]string, n)
		for i, ex := range overlay.Excerpts {
			badges[i] = ex.Badge
		}
		parts = append(parts, fmt.Sprintf("%s [%s]", pluralize(n, "deck excerpt", "deck excerpts"), strings.Join(badges, ", ")))
	}
	if n := len(overlay.Connectors); n > 0 {
		parts = append(parts, pluralize(n, "connector", "connectors"))
	}
	if n := len(overlay.Callouts); n > 0 {
		parts = append(parts, pluralize(n, "callout", "callouts"))
	}
	if overlay.DimUnlistedRows {
		parts = append(parts, "unlisted rows dimmed")
	}
	return strings.Join(parts, ", ")
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}
