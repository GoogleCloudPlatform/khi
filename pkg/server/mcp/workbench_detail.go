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
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// defaultDetailPageSize is the page size of get_timeline_logs and get_resource_revisions when pageSize is omitted.
	defaultDetailPageSize = 50
	// maxDetailPageSize caps pageSize so that a single response stays readable.
	maxDetailPageSize = 500
)

// GetTimelineLogsInput defines the input parameters for the get_timeline_logs MCP tool.
type GetTimelineLogsInput struct {
	InspectionID string                      `json:"inspectionId" jsonschema:"The target inspection ID."`
	TimelineID   string                      `json:"timelineId" jsonschema:"The timeline ID from search_timelines or search_logs."`
	Filter       workbench.TimelineLogFilter `json:"filter,omitempty" jsonschema:"Log CEL and time range filter parameters."`
	PageSize     int                         `json:"pageSize,omitempty" jsonschema:"Maximum number of logs per page (default 50, max 500)."`
	PageToken    string                      `json:"pageToken,omitempty" jsonschema:"The pageToken value from the last line of the previous response."`
}

// GetLogInput defines the input parameters for the get_log MCP tool.
type GetLogInput struct {
	InspectionID string `json:"inspectionId" jsonschema:"The target inspection ID."`
	LogID        string `json:"logId" jsonschema:"The log ID from search_logs or get_timeline_logs."`
	ByteOffset   int    `json:"byteOffset,omitempty" jsonschema:"The byteOffset value from the last line of the previous response to read the rest of a truncated body."`
}

// GetResourceRevisionsInput defines the input parameters for the get_resource_revisions MCP tool.
type GetResourceRevisionsInput struct {
	InspectionID string     `json:"inspectionId" jsonschema:"The target inspection ID."`
	TimelineID   string     `json:"timelineId" jsonschema:"The timeline ID from search_timelines or search_logs."`
	StartTime    *time.Time `json:"startTime,omitempty" jsonschema:"Inclusive start of the revision time range in RFC3339 format."`
	EndTime      *time.Time `json:"endTime,omitempty" jsonschema:"Inclusive end of the revision time range in RFC3339 format."`
	PageSize     int        `json:"pageSize,omitempty" jsonschema:"Maximum number of revisions per page (default 50, max 500)."`
	PageToken    string     `json:"pageToken,omitempty" jsonschema:"The pageToken value from the last line of the previous response."`
}

// GetResourceManifestInput defines the input parameters for the get_resource_manifest MCP tool.
type GetResourceManifestInput struct {
	InspectionID  string     `json:"inspectionId" jsonschema:"The target inspection ID."`
	TimelineID    string     `json:"timelineId" jsonschema:"The timeline ID from search_timelines or search_logs."`
	RevisionIndex *int       `json:"revisionIndex,omitempty" jsonschema:"The revision index from get_resource_revisions. Specify either revisionIndex or time."`
	Time          *time.Time `json:"time,omitempty" jsonschema:"Selects the revision effective at this time in RFC3339 format. Specify either revisionIndex or time."`
	ByteOffset    int        `json:"byteOffset,omitempty" jsonschema:"The byteOffset value from the last line of the previous response to read the rest of a truncated body. Pass it with the revisionIndex of that response."`
}

// GetResourceDiffInput defines the input parameters for the get_resource_diff MCP tool.
type GetResourceDiffInput struct {
	InspectionID  string `json:"inspectionId" jsonschema:"The target inspection ID."`
	TimelineID    string `json:"timelineId" jsonschema:"The timeline ID from search_timelines or search_logs."`
	RevisionIndex int    `json:"revisionIndex" jsonschema:"The revision index from get_resource_revisions. The diff is taken against the revision right before it."`
	ByteOffset    int    `json:"byteOffset,omitempty" jsonschema:"The byteOffset value from the last line of the previous response to read the rest of a truncated diff."`
}

func (h *WorkbenchHandler) handleGetTimelineLogs(ctx context.Context, _ *mcpsdk.CallToolRequest, input GetTimelineLogsInput) (*mcpsdk.CallToolResult, any, error) {
	if input.InspectionID == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "inspectionId is required.")
	}
	timelineID, ok := parseIDArgument(input.TimelineID)
	if !ok {
		return invalidIDResult("timelineId", input.TimelineID)
	}
	if field, err := workbench.ValidateFilter(workbench.Filter{LogQuery: input.Filter.LogQuery}); err != nil {
		return mdtemplate.CELErrorResult(field, input.Filter.LogQuery, err)
	}

	wb, errRes, err := h.acquireWorkbench(ctx, input.InspectionID)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	res, err := wb.GetTimelineLogs(ctx, timelineID, input.Filter)
	if errors.Is(err, workbench.ErrTimelineNotFound) {
		return timelineNotFoundResult(input.InspectionID, input.TimelineID)
	}
	if err != nil {
		return nil, nil, err
	}
	page, err := mdtemplate.Paginate(res.Logs, input.PageSize, defaultDetailPageSize, maxDetailPageSize, input.PageToken)
	if err != nil {
		return invalidPageTokenResult(err)
	}

	return h.templates.ToolResult("get_timeline_logs.md.tmpl", buildTimelineLogsTemplateData(res, page))
}

func (h *WorkbenchHandler) handleGetLog(ctx context.Context, _ *mcpsdk.CallToolRequest, input GetLogInput) (*mcpsdk.CallToolResult, any, error) {
	if input.InspectionID == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "inspectionId is required.")
	}
	logID, ok := parseIDArgument(input.LogID)
	if !ok {
		return invalidIDResult("logId", input.LogID)
	}
	if input.ByteOffset < 0 {
		return negativeByteOffsetResult(input.ByteOffset)
	}

	wb, errRes, err := h.acquireWorkbench(ctx, input.InspectionID)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	detail, err := wb.GetLogDetail(logID, input.ByteOffset)
	if errors.Is(err, workbench.ErrLogNotFound) {
		return mdtemplate.ErrorResult("LOG_NOT_FOUND",
			fmt.Sprintf("No log with ID %s exists in inspection %s.", mdtemplate.Code(input.LogID), mdtemplate.Code(input.InspectionID)),
			"Use a log ID from `search_logs` or `get_timeline_logs`.",
		)
	}
	if err != nil {
		return nil, nil, err
	}
	if errRes, ok := byteOffsetBeyondBodyResult(input.ByteOffset, detail.Body.TotalBytes); ok {
		return errRes, nil, nil
	}

	return h.templates.ToolResult("get_log.md.tmpl", buildLogTemplateData(detail))
}

func (h *WorkbenchHandler) handleGetResourceRevisions(ctx context.Context, _ *mcpsdk.CallToolRequest, input GetResourceRevisionsInput) (*mcpsdk.CallToolResult, any, error) {
	if input.InspectionID == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "inspectionId is required.")
	}
	timelineID, ok := parseIDArgument(input.TimelineID)
	if !ok {
		return invalidIDResult("timelineId", input.TimelineID)
	}
	offset, err := mdtemplate.DecodePageToken(input.PageToken)
	if err != nil {
		return invalidPageTokenResult(err)
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = defaultDetailPageSize
	}
	pageSize = min(pageSize, maxDetailPageSize)

	wb, errRes, err := h.acquireWorkbench(ctx, input.InspectionID)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	res, err := wb.GetResourceRevisions(timelineID, workbench.ResourceRevisionsQuery{
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
		Offset:    offset,
		Limit:     pageSize,
	})
	if errors.Is(err, workbench.ErrTimelineNotFound) {
		return timelineNotFoundResult(input.InspectionID, input.TimelineID)
	}
	if err != nil {
		return nil, nil, err
	}

	return h.templates.ToolResult("get_resource_revisions.md.tmpl", buildResourceRevisionsTemplateData(res, offset))
}

func (h *WorkbenchHandler) handleGetResourceManifest(ctx context.Context, _ *mcpsdk.CallToolRequest, input GetResourceManifestInput) (*mcpsdk.CallToolResult, any, error) {
	if input.InspectionID == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "inspectionId is required.")
	}
	timelineID, ok := parseIDArgument(input.TimelineID)
	if !ok {
		return invalidIDResult("timelineId", input.TimelineID)
	}
	if (input.RevisionIndex == nil) == (input.Time == nil) {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "Specify exactly one of revisionIndex or time.")
	}
	if input.ByteOffset < 0 {
		return negativeByteOffsetResult(input.ByteOffset)
	}

	wb, errRes, err := h.acquireWorkbench(ctx, input.InspectionID)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	manifest, err := wb.GetResourceManifest(timelineID, workbench.RevisionSelector{
		Index: input.RevisionIndex,
		Time:  input.Time,
	}, input.ByteOffset)
	if errRes, ok := revisionLookupErrorResult(err, input.InspectionID, input.TimelineID); ok {
		return errRes, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if errRes, ok := byteOffsetBeyondBodyResult(input.ByteOffset, manifest.Body.TotalBytes); ok {
		return errRes, nil, nil
	}

	return h.templates.ToolResult("get_resource_manifest.md.tmpl", buildResourceManifestTemplateData(manifest))
}

func (h *WorkbenchHandler) handleGetResourceDiff(ctx context.Context, _ *mcpsdk.CallToolRequest, input GetResourceDiffInput) (*mcpsdk.CallToolResult, any, error) {
	if input.InspectionID == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "inspectionId is required.")
	}
	timelineID, ok := parseIDArgument(input.TimelineID)
	if !ok {
		return invalidIDResult("timelineId", input.TimelineID)
	}
	if input.ByteOffset < 0 {
		return negativeByteOffsetResult(input.ByteOffset)
	}

	wb, errRes, err := h.acquireWorkbench(ctx, input.InspectionID)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	diff, err := wb.GetResourceDiff(timelineID, input.RevisionIndex, input.ByteOffset)
	if errRes, ok := revisionLookupErrorResult(err, input.InspectionID, input.TimelineID); ok {
		return errRes, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if errRes, ok := byteOffsetBeyondBodyResult(input.ByteOffset, diff.Diff.TotalBytes); ok {
		return errRes, nil, nil
	}

	return h.templates.ToolResult("get_resource_diff.md.tmpl", buildResourceDiffTemplateData(diff))
}

// revisionLookupErrorResult converts a timeline or revision lookup error into an MCP error result.
// It returns false when err is not a lookup error.
func revisionLookupErrorResult(err error, inspectionID, timelineID string) (*mcpsdk.CallToolResult, bool) {
	switch {
	case errors.Is(err, workbench.ErrTimelineNotFound):
		res, _, _ := timelineNotFoundResult(inspectionID, timelineID)
		return res, true
	case errors.Is(err, workbench.ErrRevisionNotFound):
		res, _, _ := mdtemplate.ErrorResult("REVISION_NOT_FOUND",
			fmt.Sprintf("Timeline %s has no matching revision: %s.", mdtemplate.Code(timelineID), err.Error()),
			"Use a revision index from `get_resource_revisions`.",
		)
		return res, true
	default:
		return nil, false
	}
}

func timelineNotFoundResult(inspectionID, timelineID string) (*mcpsdk.CallToolResult, any, error) {
	return mdtemplate.ErrorResult("TIMELINE_NOT_FOUND",
		fmt.Sprintf("No timeline with ID %s exists in inspection %s.", mdtemplate.Code(timelineID), mdtemplate.Code(inspectionID)),
		"Use a timeline ID from `search_timelines` or `search_logs`.",
	)
}

func invalidIDResult(field, value string) (*mcpsdk.CallToolResult, any, error) {
	return mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("%s must be a positive integer ID, got %s.", field, mdtemplate.Code(value)))
}

func invalidPageTokenResult(err error) (*mcpsdk.CallToolResult, any, error) {
	return mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("pageToken is invalid: %s.", err.Error()), "Pass the pageToken value from the previous response as is.")
}

func negativeByteOffsetResult(byteOffset int) (*mcpsdk.CallToolResult, any, error) {
	return mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("byteOffset must not be negative, got %s.", mdtemplate.Code(strconv.Itoa(byteOffset))), "Pass the byteOffset value from the last line of the previous response as is.")
}

// byteOffsetBeyondBodyResult returns an INVALID_ARGUMENT result when byteOffset points past the end of a body of totalBytes bytes.
// A byteOffset equal to totalBytes is accepted because it reads the empty remainder of the body.
// It returns false when byteOffset is within the body.
func byteOffsetBeyondBodyResult(byteOffset, totalBytes int) (*mcpsdk.CallToolResult, bool) {
	if byteOffset <= totalBytes {
		return nil, false
	}
	res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT",
		fmt.Sprintf("byteOffset %s exceeds the body size of %d bytes.", mdtemplate.Code(strconv.Itoa(byteOffset)), totalBytes),
		"Pass the byteOffset value from the last line of the previous response as is.",
	)
	return res, true
}

// parseIDArgument parses a timeline or log ID passed as a decimal string. IDs start from 1.
func parseIDArgument(value string) (uint32, bool) {
	id, err := strconv.ParseUint(value, 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint32(id), true
}
