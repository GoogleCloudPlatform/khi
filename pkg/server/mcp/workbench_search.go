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
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchTimelinesInput defines the input parameters for the search_timelines MCP tool.
type SearchTimelinesInput struct {
	InspectionID string           `json:"inspectionId" jsonschema:"The target inspection ID."`
	Filter       workbench.Filter `json:"filter,omitempty" jsonschema:"CEL and time range filter parameters."`
	MaxDepth     int              `json:"maxDepth,omitempty" jsonschema:"Maximum tree depth from root timelines (0-based). 0 means unlimited."`
	MaxNodes     int              `json:"maxNodes,omitempty" jsonschema:"Maximum number of timeline nodes to return (default 200)."`
}

// SearchLogsInput defines the input parameters for the search_logs MCP tool.
type SearchLogsInput struct {
	InspectionID  string           `json:"inspectionId" jsonschema:"The target inspection ID."`
	Filter        workbench.Filter `json:"filter,omitempty" jsonschema:"CEL and time range filter parameters."`
	MaxTimelines  int              `json:"maxTimelines,omitempty" jsonschema:"Maximum number of linked timelines to return (default 20)."`
	MaxSampleLogs int              `json:"maxSampleLogs,omitempty" jsonschema:"Maximum number of evenly spaced sample logs to return (default 20)."`
}

type searchTimelinesTemplateData struct {
	MatchedTimelineCount int
	ReturnedNodeCount    int
	MaxDepth             int
	MaxNodes             int
	Applied              workbench.AppliedFilter
	TimelineTypes        []workbench.TimelineTypeDescription
	TreeLines            []string
}

type timelineLogGroupTemplateData struct {
	TimelineID      string
	Segments        []workbench.TimelineSegment
	MatchedLogCount int
	SeveritySummary string
	FirstMatchTime  time.Time
	LastMatchTime   time.Time
}

type sampleLogTemplateData struct {
	LogID         string
	Time          time.Time
	Severity      string
	LogType       string
	Summary       string
	TimelinesCell string
}

type searchLogsTemplateData struct {
	MatchedLogCount      int
	MatchedTimelineCount int
	FirstMatchTime       time.Time
	LastMatchTime        time.Time
	SeveritySummary      string
	Applied              workbench.AppliedFilter
	TimelineGroups       []timelineLogGroupTemplateData
	SampleLogs           []sampleLogTemplateData
}

func (h *WorkbenchHandler) handleSearchTimelines(ctx context.Context, _ *mcpsdk.CallToolRequest, input SearchTimelinesInput) (*mcpsdk.CallToolResult, any, error) {
	if input.InspectionID == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "inspectionId is required.")
	}
	if field, err := workbench.ValidateFilter(input.Filter); err != nil {
		return mdtemplate.CELErrorResult(field, filterExpressionByField(input.Filter, field), err)
	}

	wb, errRes, err := h.acquireWorkbench(ctx, input.InspectionID)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	treeRes, err := wb.SearchTimelines(ctx, input.Filter, input.MaxDepth, input.MaxNodes)
	if err != nil {
		return nil, nil, err
	}

	sameDay := isSameUTCDate(treeRes.Applied.StartTime, treeRes.Applied.EndTime)
	treeLines := make([]string, len(treeRes.Nodes))
	for i, node := range treeRes.Nodes {
		treeLines[i] = formatTimelineTreeLine(node, sameDay)
	}

	return h.templates.ToolResult("search_timelines.md.tmpl", searchTimelinesTemplateData{
		MatchedTimelineCount: treeRes.MatchedTimelineCount,
		ReturnedNodeCount:    treeRes.ReturnedNodeCount,
		MaxDepth:             treeRes.MaxDepth,
		MaxNodes:             treeRes.MaxNodes,
		Applied:              treeRes.Applied,
		TimelineTypes:        treeRes.TimelineTypes,
		TreeLines:            treeLines,
	})
}

func (h *WorkbenchHandler) handleSearchLogs(ctx context.Context, _ *mcpsdk.CallToolRequest, input SearchLogsInput) (*mcpsdk.CallToolResult, any, error) {
	if input.InspectionID == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "inspectionId is required.")
	}
	if field, err := workbench.ValidateFilter(input.Filter); err != nil {
		return mdtemplate.CELErrorResult(field, filterExpressionByField(input.Filter, field), err)
	}

	wb, errRes, err := h.acquireWorkbench(ctx, input.InspectionID)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	logRes, err := wb.SearchLogs(ctx, input.Filter, input.MaxTimelines, input.MaxSampleLogs)
	if err != nil {
		return nil, nil, err
	}

	groups := make([]timelineLogGroupTemplateData, len(logRes.TimelineGroups))
	for i, g := range logRes.TimelineGroups {
		groups[i] = timelineLogGroupTemplateData{
			TimelineID:      strconv.FormatUint(uint64(g.TimelineID), 10),
			Segments:        g.Segments,
			MatchedLogCount: g.MatchedLogCount,
			SeveritySummary: formatSeveritySummary(g.SeverityCounts),
			FirstMatchTime:  g.FirstMatchTime,
			LastMatchTime:   g.LastMatchTime,
		}
	}

	samples := make([]sampleLogTemplateData, len(logRes.SampleLogs))
	for i, s := range logRes.SampleLogs {
		samples[i] = sampleLogTemplateData{
			LogID:         strconv.FormatUint(uint64(s.LogID), 10),
			Time:          s.Time,
			Severity:      s.Severity.GetLabel(),
			LogType:       s.LogType,
			Summary:       s.Summary,
			TimelinesCell: formatTimelineIDsCell(s.TimelineIDs),
		}
	}

	return h.templates.ToolResult("search_logs.md.tmpl", searchLogsTemplateData{
		MatchedLogCount:      logRes.MatchedLogCount,
		MatchedTimelineCount: logRes.MatchedTimelineCount,
		FirstMatchTime:       logRes.FirstMatchTime,
		LastMatchTime:        logRes.LastMatchTime,
		SeveritySummary:      formatSeveritySummary(logRes.SeverityCounts),
		Applied:              logRes.Applied,
		TimelineGroups:       groups,
		SampleLogs:           samples,
	})
}

func filterExpressionByField(filter workbench.Filter, field string) string {
	switch field {
	case "filter.timelineQuery":
		return filter.TimelineQuery
	case "filter.timelineExclusionQuery":
		return filter.TimelineExclusionQuery
	case "filter.logQuery":
		return filter.LogQuery
	default:
		return ""
	}
}

func isSameUTCDate(start, end time.Time) bool {
	if start.IsZero() || end.IsZero() {
		return false
	}
	su := start.UTC()
	eu := end.UTC()
	return su.Year() == eu.Year() && su.Month() == eu.Month() && su.Day() == eu.Day()
}

func formatTimelineTreeLine(node workbench.TimelineTreeNode, sameDay bool) string {
	indent := strings.Repeat("  ", node.Depth)
	tokens := []string{
		fmt.Sprintf("%s[%s] %s", indent, node.Type, node.Name),
		fmt.Sprintf("id=%d", node.ID),
	}
	if node.OmittedChildren > 0 {
		tokens = append(tokens, fmt.Sprintf("children=%d", node.OmittedChildren))
	}
	if node.EventCount > 0 || node.RevisionCount > 0 {
		tokens = append(tokens, fmt.Sprintf("ev=%d rev=%d", node.EventCount, node.RevisionCount))
	}
	for _, c := range node.SeverityCounts {
		tokens = append(tokens, fmt.Sprintf("%s=%d", strings.ToLower(c.Severity.GetLabel()), c.Count))
	}
	if !node.FirstMatchTime.IsZero() && !node.LastMatchTime.IsZero() {
		layout := time.RFC3339
		if sameDay {
			layout = "15:04:05"
		}
		tokens = append(tokens, fmt.Sprintf("%s..%s",
			node.FirstMatchTime.UTC().Format(layout),
			node.LastMatchTime.UTC().Format(layout),
		))
	}
	return strings.Join(tokens, " ")
}

func formatSeveritySummary(counts []workbench.SeverityCount) string {
	if len(counts) == 0 {
		return ""
	}
	parts := make([]string, len(counts))
	for i, c := range counts {
		parts[i] = fmt.Sprintf("%s %d", c.Severity.GetLabel(), c.Count)
	}
	return strings.Join(parts, ", ")
}

func formatTimelineIDsCell(ids []uint32) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = mdtemplate.Code(strconv.FormatUint(uint64(id), 10))
	}
	return strings.Join(parts, ", ")
}
