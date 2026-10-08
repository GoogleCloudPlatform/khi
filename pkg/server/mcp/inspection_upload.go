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

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
)

// RequestFileUploadInput defines the input parameters for the request_file_upload tool.
type RequestFileUploadInput struct {
	InspectionID string `json:"inspectionId" jsonschema:"The unique identifier of the inspection."`
	FieldID      string `json:"fieldId" jsonschema:"The ID of a field with Type: file in the dry_run_inspection result."`
}

func (h *InspectionHandler) handleRequestFileUpload(ctx context.Context, req *mcpsdk.CallToolRequest, in RequestFileUploadInput) (*mcpsdk.CallToolResult, any, error) {
	runner := h.server.GetInspection(in.InspectionID)
	if runner == nil {
		return inspectionNotFoundResult(in.InspectionID)
	}

	if runner.Started() {
		return inspectionAlreadyStartedResult(in.InspectionID)
	}

	// The dry run registers the upload token of each file field, so the URL is bound to a token that the run reads.
	data, _, errRes, err := h.dryRun(ctx, in.InspectionID, runner, nil, "request_file_upload")
	if errRes != nil || err != nil {
		return errRes, nil, err
	}

	uploadTokenID, found := findUploadTokenID(data.Groups, in.FieldID)
	if !found {
		return mdtemplate.ErrorResult("FILE_FIELD_NOT_FOUND",
			fmt.Sprintf("Inspection %s has no field %s with `Type: file`.", mdtemplate.Code(in.InspectionID), mdtemplate.Code(in.FieldID)),
			"Call `dry_run_inspection` and use the ID of a field with `Type: file`.")
	}

	return h.templates.ToolResult("request_file_upload.md.tmpl", h.uploadURLs.Issue(uploadTokenID, in.FieldID))
}

// findUploadTokenID returns the upload token ID of the file field fieldID in groups.
// It returns false when no field has the ID or the field is not a file field.
func findUploadTokenID(groups []formGroupData, fieldID string) (string, bool) {
	for _, group := range groups {
		for _, field := range group.Fields {
			if field.ID == fieldID && field.uploadTokenID != "" {
				return field.uploadTokenID, true
			}
		}
	}
	return "", false
}
