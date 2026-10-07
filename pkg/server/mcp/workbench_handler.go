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
	"strings"
	"time"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultWorkbenchWaitTimeout = 30 * time.Second

// WorkbenchHandler exposes workbench operations and tools over MCP.
type WorkbenchHandler struct {
	manager     *workbench.WorkbenchManager
	templates   *mdtemplate.Set
	waitTimeout time.Duration
}

var _ DomainHandler = (*WorkbenchHandler)(nil)

// NewWorkbenchHandler creates a new WorkbenchHandler.
func NewWorkbenchHandler(manager *workbench.WorkbenchManager) *WorkbenchHandler {
	return &WorkbenchHandler{
		manager:     manager,
		templates:   mdtemplate.MustParse(templateFS, "templates/*.md.tmpl"),
		waitTimeout: defaultWorkbenchWaitTimeout,
	}
}

// Register registers workbench tools to the MCP server.
func (h *WorkbenchHandler) Register(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "search_timelines",
		Description: `Search the resource timeline tree with a filter and return matching nodes with event and revision counts and per-severity log counts including descendants. Timeline CEL uses path["kind"], path["namespace"], path["resource"], path["subresource"], name, timelineType, match(), revision_body(), minSeverity(WARNING), hasSeverity(ERROR). Log CEL uses severity >= WARNING, logType, body("regex"), body("field.path", "regex").`,
	}, h.handleSearchTimelines)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "search_logs",
		Description: `Search logs across all timelines with a filter and return severity counts, top linked timelines, and evenly spaced sample logs. Timeline CEL uses path["kind"], path["namespace"], path["resource"], path["subresource"], name, timelineType, match(), revision_body(), minSeverity(WARNING), hasSeverity(ERROR). Log CEL uses severity >= WARNING, logType, body("regex"), body("field.path", "regex").`,
	}, h.handleSearchLogs)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_timeline_logs",
		Description: `List logs on one timeline, oldest first, with an optional log CEL and time range filter, and count the other timelines linked to the matching logs. Log CEL uses severity >= WARNING, logType, body("regex"), body("field.path", "regex"). Results are paginated; pass the pageToken from the last line to read the next page.`,
	}, h.handleGetTimelineLogs)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_log",
		Description: `Get one log with its metadata, linked timelines, and full body in YAML. Long bodies are truncated; pass the byteOffset from the last line to read the rest.`,
	}, h.handleGetLog)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_resource_revisions",
		Description: `List the manifest revisions of a resource timeline, oldest first, with verb, state, principal, source log, and added/deleted line counts against the previous revision. Results are paginated; pass the pageToken from the last line to read the next page.`,
	}, h.handleGetResourceRevisions)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_resource_manifest",
		Description: `Get the manifest of a resource at a revision index or at a point in time. Long manifests are truncated; pass the byteOffset from the last line together with the revisionIndex shown in the response to read the rest.`,
	}, h.handleGetResourceManifest)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_resource_diff",
		Description: `Get the unified diff between a resource revision and the revision right before it. Long diffs are truncated; pass the byteOffset from the last line to read the rest.`,
	}, h.handleGetResourceDiff)
}

type workbenchLoadingData struct {
	ID            string
	Stage         string
	ProgressRatio float64
}

// acquireWorkbench loads or retrieves a workbench for the inspection, waiting up to waitTimeout.
// If the workbench or index is still loading when the timeout expires, it returns a WORKBENCH_LOADING tool result.
func (h *WorkbenchHandler) acquireWorkbench(ctx context.Context, inspectionID string) (*workbench.Workbench, *mcpsdk.CallToolResult, error) {
	waitCtx, cancel := context.WithTimeout(ctx, h.waitTimeout)
	defer cancel()

	handle := h.manager.Load(inspectionID, workbench.AccessorMCP)
	wb, err := handle.Wait(waitCtx, func(apiv1.OpenWorkbenchResponse_Stage, float64, string) error { return nil })
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		if errors.Is(err, context.DeadlineExceeded) {
			p := handle.Progress()
			return h.workbenchLoadingResult(inspectionID, loadStageName(p.Stage), p.Percentage/100.0)
		}
		if errors.Is(err, workbench.ErrInspectionNotFound) {
			res, _, err := inspectionNotFoundResult(inspectionID)
			return nil, res, err
		}
		return nil, nil, err
	}

	err = wb.AwaitIndex(waitCtx)
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		if errors.Is(err, context.DeadlineExceeded) {
			_, pct, _, _ := wb.IndexStatus()
			return h.workbenchLoadingResult(inspectionID, "BUILDING_INDEX", pct/100.0)
		}
		return nil, nil, err
	}

	return wb, nil, nil
}

func loadStageName(stage apiv1.OpenWorkbenchResponse_Stage) string {
	if stage == apiv1.OpenWorkbenchResponse_STAGE_UNSPECIFIED {
		return "INITIALIZING"
	}
	return strings.TrimPrefix(stage.String(), "STAGE_")
}

func (h *WorkbenchHandler) workbenchLoadingResult(inspectionID, stage string, progressRatio float64) (*workbench.Workbench, *mcpsdk.CallToolResult, error) {
	res, _, err := h.templates.ErrorToolResult("workbench_loading.md.tmpl", workbenchLoadingData{
		ID:            inspectionID,
		Stage:         stage,
		ProgressRatio: progressRatio,
	})
	return nil, res, err
}
