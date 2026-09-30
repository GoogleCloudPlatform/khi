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
	"slices"
	"strings"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateInspectionInput defines the input parameters for the create_inspection tool.
type CreateInspectionInput struct {
	InspectionTypeID string `json:"inspectionTypeId" jsonschema:"The ID of the inspection type to create (for example 'gcp-gke')."`
	Name             string `json:"name,omitempty" jsonschema:"Optional display name for the inspection."`
}

// UpdateInspectionFeaturesInput defines the input parameters for the update_inspection_features tool.
type UpdateInspectionFeaturesInput struct {
	InspectionID      string   `json:"inspectionId" jsonschema:"The unique identifier of the inspection."`
	EnabledFeatureIDs []string `json:"enabledFeatureIds" jsonschema:"The complete list of all feature IDs to enable for this inspection."`
}

type featureRow struct {
	ID          string
	Name        string
	Description string
	Enabled     string
}

type inspectionFeaturesData struct {
	ID       string
	Name     string
	TypeName string
	TypeID   string
	Features []featureRow
}

func inspectionNotFoundResult(id string) (*mcpsdk.CallToolResult, any, error) {
	return mdtemplate.ErrorResult("INSPECTION_NOT_FOUND",
		fmt.Sprintf("Inspection %s not found.", mdtemplate.Code(id)),
		"Read `khi://inspections` to see existing inspections.")
}

func inspectionAlreadyStartedResult(id string) (*mcpsdk.CallToolResult, any, error) {
	return mdtemplate.ErrorResult("INSPECTION_ALREADY_STARTED",
		fmt.Sprintf("Inspection %s has already been started.", mdtemplate.Code(id)),
		"Create a new inspection with `create_inspection`.")
}

// inspectionFeaturesResult renders the inspection header and the current enabled state of every feature.
// create_inspection and update_inspection_features share it so both return the same template.
func (h *InspectionHandler) inspectionFeaturesResult(id string, runner *coreinspection.InspectionTaskRunner) (*mcpsdk.CallToolResult, any, error) {
	features, err := runner.FeatureList()
	if err != nil {
		return nil, nil, err
	}

	name, _ := h.server.InspectionNameRegistry().NameOf(id)

	typeID := runner.InspectionTypeID()
	typeName := typeID
	if inspType := h.server.GetInspectionType(typeID); inspType != nil {
		typeName = inspType.Name
	}

	featureRows := make([]featureRow, 0, len(features))
	for _, f := range features {
		enabled := "no"
		if f.Enabled {
			enabled = "yes"
		}
		featureRows = append(featureRows, featureRow{
			ID:          f.Id,
			Name:        f.Label,
			Description: f.Description,
			Enabled:     enabled,
		})
	}

	return h.templates.ToolResult("inspection_features.md.tmpl", inspectionFeaturesData{
		ID:       id,
		Name:     name,
		TypeName: typeName,
		TypeID:   typeID,
		Features: featureRows,
	})
}

func (h *InspectionHandler) handleCreateInspection(ctx context.Context, req *mcpsdk.CallToolRequest, in CreateInspectionInput) (*mcpsdk.CallToolResult, any, error) {
	inspType := h.server.GetInspectionType(in.InspectionTypeID)
	if inspType == nil {
		return mdtemplate.ErrorResult("UNKNOWN_INSPECTION_TYPE",
			fmt.Sprintf("Unknown inspection type ID: %s", mdtemplate.Code(in.InspectionTypeID)),
			"Read `khi://inspection-types` to see available types.")
	}

	if reason := mcpUnavailableReason(inspType); reason != "" {
		return mdtemplate.ErrorResult("INSPECTION_TYPE_NOT_AVAILABLE",
			fmt.Sprintf("Inspection type %s %s", mdtemplate.Code(inspType.Id), reason),
			"Ask the user to create this inspection in the KHI Web UI.")
	}

	customName := strings.TrimSpace(in.Name)
	if customName != "" {
		// Checking before CreateInspection avoids leaving a runner behind on a duplicate name.
		if !h.server.InspectionNameRegistry().IsNameAvailable("", customName) {
			return mdtemplate.ErrorResult("INVALID_INSPECTION_NAME",
				fmt.Sprintf("Inspection name %q is already in use. Choose another name.", customName))
		}
	}

	id, err := h.server.CreateInspection(in.InspectionTypeID)
	if err != nil {
		return nil, nil, err
	}

	if customName != "" {
		if err := h.server.InspectionNameRegistry().ReserveName(id, customName); err != nil {
			return mdtemplate.ErrorResult("INVALID_INSPECTION_NAME",
				fmt.Sprintf("Inspection name %q cannot be reserved: %v", customName, err))
		}
	} else {
		h.server.InspectionNameRegistry().ReserveUniqueName(id, inspType.Name)
	}

	runner := h.server.GetInspection(id)
	return h.inspectionFeaturesResult(id, runner)
}

func (h *InspectionHandler) handleUpdateInspectionFeatures(ctx context.Context, req *mcpsdk.CallToolRequest, in UpdateInspectionFeaturesInput) (*mcpsdk.CallToolResult, any, error) {
	runner := h.server.GetInspection(in.InspectionID)
	if runner == nil {
		return inspectionNotFoundResult(in.InspectionID)
	}

	if runner.Started() {
		return inspectionAlreadyStartedResult(in.InspectionID)
	}

	if len(in.EnabledFeatureIDs) == 0 {
		return mdtemplate.ErrorResult("NO_FEATURES",
			"Specify at least one feature ID to enable.",
			"Use the IDs returned by `create_inspection`.")
	}

	featureIDs := slices.Clone(in.EnabledFeatureIDs)
	slices.Sort(featureIDs)
	featureIDs = slices.Compact(featureIDs)

	features, err := runner.FeatureList()
	if err != nil {
		return nil, nil, err
	}

	validMap := make(map[string]bool, len(features))
	for _, f := range features {
		validMap[f.Id] = true
	}

	var unknown []string
	for _, fid := range featureIDs {
		if !validMap[fid] {
			unknown = append(unknown, mdtemplate.Code(fid))
		}
	}
	if len(unknown) > 0 {
		return mdtemplate.ErrorResult("UNKNOWN_FEATURE",
			fmt.Sprintf("Unknown feature IDs: %s", strings.Join(unknown, ", ")),
			"Use the IDs returned by `create_inspection`.")
	}

	if err := runner.SetFeatureList(featureIDs); err != nil {
		return nil, nil, err
	}

	return h.inspectionFeaturesResult(in.InspectionID, runner)
}
