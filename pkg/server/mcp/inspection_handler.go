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
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed templates/*.md.tmpl
var templateFS embed.FS

// InspectionHandler exposes inspection list and summary resources over MCP.
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

// Register registers the inspection resources and resource templates to the MCP server.
func (h *InspectionHandler) Register(srv *mcpsdk.Server) {
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
