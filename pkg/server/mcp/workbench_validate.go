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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ValidateReportInput defines the input parameters for the validate_report MCP tool.
type ValidateReportInput struct {
	InspectionID string `json:"inspectionId" jsonschema:"The target inspection ID."`
	Report       string `json:"report" jsonschema:"The Markdown report or writeup to validate against the inspection data."`
}

func (h *WorkbenchHandler) handleValidateReport(ctx context.Context, _ *mcpsdk.CallToolRequest, input ValidateReportInput) (*mcpsdk.CallToolResult, any, error) {
	if input.InspectionID == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "inspectionId is required.")
	}
	if strings.TrimSpace(input.Report) == "" {
		return mdtemplate.ErrorResult("INVALID_ARGUMENT", "report is required.")
	}

	wb, errRes, err := h.acquireWorkbench(ctx, input.InspectionID)
	if err != nil || errRes != nil {
		return errRes, nil, err
	}

	res, err := wb.ValidateReport(ctx, input.Report)
	if err != nil {
		return nil, nil, err
	}

	return h.templates.ToolResult("validate_report.md.tmpl", res)
}
