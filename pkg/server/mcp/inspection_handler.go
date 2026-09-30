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
	"cmp"
	"context"
	"embed"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/summary"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed templates/*.md.tmpl
var templateFS embed.FS

// InspectionHandler exposes inspection types, inspection listing, inspection summaries, and inspection lifecycle tools over MCP.
type InspectionHandler struct {
	server       *coreinspection.InspectionTaskServer
	templates    *mdtemplate.Set
	summaryCache map[string]string
	cacheMu      sync.Mutex
}

var _ DomainHandler = (*InspectionHandler)(nil)

// NewInspectionHandler creates a new InspectionHandler using the provided InspectionTaskServer.
func NewInspectionHandler(server *coreinspection.InspectionTaskServer) *InspectionHandler {
	return &InspectionHandler{
		server:       server,
		templates:    mdtemplate.MustParse(templateFS, "templates/*.md.tmpl"),
		summaryCache: make(map[string]string),
	}
}

// Register registers the inspection resources, resource templates, and tools to the MCP server.
func (h *InspectionHandler) Register(srv *mcpsdk.Server) {
	srv.AddResource(&mcpsdk.Resource{
		URI:         "khi://inspection-types",
		Name:        "inspection-types",
		Description: "List all inspection types and their availability over MCP.",
		MIMEType:    "text/markdown",
	}, h.handleListInspectionTypes)

	srv.AddResource(&mcpsdk.Resource{
		URI:         "khi://inspections",
		Name:        "inspections",
		Description: "List all inspections. Read this resource first to find available inspection IDs.",
		MIMEType:    "text/markdown",
	}, h.handleListInspections)

	srv.AddResourceTemplate(&mcpsdk.ResourceTemplate{
		URITemplate: "khi://inspections/{id}",
		Name:        "inspection",
		Description: "Inspection summary or current progress status.",
		MIMEType:    "text/markdown",
	}, h.handleInspectionResource)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "create_inspection",
		Description: "Create a new inspection session of the specified type. Returns the inspection ID and selectable features. Check available types from khi://inspection-types first.",
	}, h.handleCreateInspection)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "update_inspection_features",
		Description: "Update the enabled features for an inspection session. Specify all feature IDs that should be enabled; this replaces the current feature selection. Call dry_run_inspection next.",
	}, h.handleUpdateInspectionFeatures)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "dry_run_inspection",
		Description: "Dry-run the inspection task DAG with parameters to validate inputs and view planned log queries. Must always be run before run_inspection until all errors are resolved. All times are UTC RFC3339. Warnings do not block run_inspection. Wider time ranges increase execution time and memory usage.",
	}, h.handleDryRunInspection)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "run_inspection",
		Description: "Start the inspection with the same parameters as dry_run_inspection. Always call dry_run_inspection first and fix all errors; warnings do not block the run. All times are UTC RFC3339. Wider time ranges increase execution time and memory usage. Returns immediately; call wait_inspection next.",
	}, h.handleRunInspection)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "wait_inspection",
		Description: "Wait until the inspection finishes or timeoutSeconds (default 60, max 300) elapses, then return the status and the progress of running tasks. A timeout is not a failure; call again to keep waiting. This is the only way to read progress.",
	}, h.handleWaitInspection)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "cancel_inspection",
		Description: "Cancel a running inspection.",
	}, h.handleCancelInspection)
}

// mcpUnavailableReason returns a non-empty explanation if the inspection type cannot be used via MCP.
// TODO(#1047): Remove this check when file uploads are supported via MCP.
func mcpUnavailableReason(t *coreinspection.InspectionType) string {
	if t.Labels[inspectioncore.InspectionTypeLabelKeyLogSource] == "file" {
		return "requires file uploads, which are not supported via MCP yet."
	}
	return ""
}

type inspectionTypeRow struct {
	ID          string
	Name        string
	Description string
	Available   string
}

type inspectionTypesData struct {
	Types []inspectionTypeRow
}

func (h *InspectionHandler) handleListInspectionTypes(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
	types := h.server.GetAllInspectionTypes()
	slices.SortFunc(types, func(a, b *coreinspection.InspectionType) int {
		if n := cmp.Compare(b.Priority, a.Priority); n != 0 {
			return n
		}
		return cmp.Compare(a.Id, b.Id)
	})

	rows := make([]inspectionTypeRow, 0, len(types))
	for _, t := range types {
		available := "yes"
		if reason := mcpUnavailableReason(t); reason != "" {
			available = fmt.Sprintf("no: %s Use the KHI Web UI.", reason)
		}
		rows = append(rows, inspectionTypeRow{
			ID:          t.Id,
			Name:        t.Name,
			Description: t.Description,
			Available:   available,
		})
	}

	return h.templates.ResourceResult("khi://inspection-types", "inspection_types.md.tmpl", inspectionTypesData{Types: rows})
}

type inspectionListRow struct {
	ID          string
	Name        string
	Type        string
	Status      string
	TimeRange   string
	Labels      string
	InspectTime int64
}

type inspectionListData struct {
	Rows []inspectionListRow
}

func (h *InspectionHandler) handleListInspections(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
	runners := h.server.GetAllRunners()
	rows := make([]inspectionListRow, 0, len(runners))

	for _, runner := range runners {
		metadata, err := runner.GetCurrentMetadata()
		if err != nil {
			// Runner has not been started yet.
			continue
		}

		id := runner.ID
		var name, inspectionType string
		var inspectTime int64
		var startTime, endTime time.Time

		header, found := typedmap.Get(metadata, inspectionmetadata.HeaderMetadataKey)
		if found && header != nil {
			name = header.InspectionName
			inspectionType = header.InspectionType
			inspectTime = header.InspectTimeUnixSeconds
			if header.StartTimeUnixSeconds != 0 {
				startTime = time.Unix(header.StartTimeUnixSeconds, 0)
			}
			if header.EndTimeUnixSeconds != 0 {
				endTime = time.Unix(header.EndTimeUnixSeconds, 0)
			}
		}

		status := "DONE"
		if progress, found := typedmap.Get(metadata, inspectionmetadata.ProgressMetadataKey); found && progress != nil {
			status = string(progress.Snapshot().Phase)
		}

		timeRange := ""
		if !startTime.IsZero() && !endTime.IsZero() {
			timeRange = fmt.Sprintf("%s - %s", mdtemplate.FormatTime(startTime), mdtemplate.FormatTime(endTime))
		}

		labels := ""
		if s, found := typedmap.Get(metadata, summary.MetadataKey); found && s != nil {
			snap := s.Snapshot()
			if len(snap.CoreLabels) > 0 {
				parts := make([]string, len(snap.CoreLabels))
				for i, label := range snap.CoreLabels {
					parts[i] = fmt.Sprintf("%s=%s", label.Key, label.Value)
				}
				labels = strings.Join(parts, ", ")
			}
		}

		rows = append(rows, inspectionListRow{
			ID:          id,
			Name:        name,
			Type:        inspectionType,
			Status:      status,
			TimeRange:   timeRange,
			Labels:      labels,
			InspectTime: inspectTime,
		})
	}

	slices.SortFunc(rows, func(a, b inspectionListRow) int {
		if a.InspectTime != b.InspectTime {
			if a.InspectTime > b.InspectTime {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})

	return h.templates.ResourceResult("khi://inspections", "inspections.md.tmpl", inspectionListData{Rows: rows})
}

type inspectionStatusData struct {
	ID     string
	Status string
}

type inspectionSummaryData struct {
	ID        string
	Header    *inspectionmetadata.HeaderMetadata
	TimeRange string
	Summary   summary.Snapshot
}

func (h *InspectionHandler) handleInspectionResource(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
	uri := req.Params.URI
	const prefix = "khi://inspections/"
	if !strings.HasPrefix(uri, prefix) {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}

	id := strings.TrimPrefix(uri, prefix)
	if id == "" || strings.Contains(id, "/") {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}

	runner := h.server.GetInspection(id)
	if runner == nil {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}

	metadata, err := runner.GetCurrentMetadata()
	if err != nil {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}

	status := "DONE"
	if progress, found := typedmap.Get(metadata, inspectionmetadata.ProgressMetadataKey); found && progress != nil {
		status = string(progress.Snapshot().Phase)
	}

	if status != "DONE" {
		return h.templates.ResourceResult(uri, "inspection_status.md.tmpl", inspectionStatusData{
			ID:     id,
			Status: status,
		})
	}

	text, err := h.renderSummary(id, metadata)
	if err != nil {
		return nil, err
	}

	return &mcpsdk.ReadResourceResult{
		Contents: []*mcpsdk.ResourceContents{
			{
				URI:      uri,
				MIMEType: "text/markdown",
				Text:     text,
			},
		},
	}, nil
}

func (h *InspectionHandler) renderSummary(id string, metadata *typedmap.ReadonlyTypedMap) (string, error) {
	h.cacheMu.Lock()
	cached, ok := h.summaryCache[id]
	h.cacheMu.Unlock()
	if ok {
		return cached, nil
	}

	header, found := typedmap.Get(metadata, inspectionmetadata.HeaderMetadataKey)
	if !found || header == nil {
		return "", mcpsdk.ResourceNotFoundError(fmt.Sprintf("khi://inspections/%s", id))
	}

	var startTime, endTime time.Time
	if header.StartTimeUnixSeconds != 0 {
		startTime = time.Unix(header.StartTimeUnixSeconds, 0)
	}
	if header.EndTimeUnixSeconds != 0 {
		endTime = time.Unix(header.EndTimeUnixSeconds, 0)
	}

	timeRange := ""
	if !startTime.IsZero() && !endTime.IsZero() {
		timeRange = fmt.Sprintf("%s - %s", mdtemplate.FormatTime(startTime), mdtemplate.FormatTime(endTime))
	}

	var snap summary.Snapshot
	if s, found := typedmap.Get(metadata, summary.MetadataKey); found && s != nil {
		snap = s.Snapshot()
	}

	text, err := h.templates.Render("inspection_summary.md.tmpl", inspectionSummaryData{
		ID:        id,
		Header:    header,
		TimeRange: timeRange,
		Summary:   snap,
	})
	if err != nil {
		return "", err
	}

	h.cacheMu.Lock()
	h.summaryCache[id] = text
	h.cacheMu.Unlock()

	return text, nil
}
