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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestWorkbenchHandler_HandleValidateReport(t *testing.T) {
	mgr, inspectionID := newTestWorkbenchManager(t)
	handler := NewWorkbenchHandler(mgr, nil)

	testCases := []struct {
		name  string
		input ValidateReportInput
		want  *mcpsdk.CallToolResult
	}{
		{
			name: "flags ungrounded inline code spans in report",
			input: ValidateReportInput{
				InspectionID: inspectionID,
				Report:       "Pod `hallucinated-pod-xyz` failed.",
			},
			want: wantTextResult(strings.Join([]string{
				"# Report validation",
				"",
				"Found 1 potential grounding issues.",
				"",
				"## Rules checked",
				"- `log-existence`: Inline code spans (`...`) must exist in the inspection logs. (1 findings)",
				"",
				"## Findings",
				"Verify each flagged item with `search_timelines`, `search_logs`, or `get_log`, and remove or correct values that do not exist in the inspection.",
				"",
				"| Line | Rule | Target | Message |",
				"| --- | --- | --- | --- |",
				"| 1 | `log-existence` | `hallucinated-pod-xyz` | Not found in inspection logs. |",
			}, "\n")),
		},
		{
			name: "missing inspectionId returns INVALID_ARGUMENT",
			input: ValidateReportInput{
				InspectionID: "",
				Report:       "Report text",
			},
			want: wantErrorResult("INVALID_ARGUMENT", "inspectionId is required."),
		},
		{
			name: "empty report returns INVALID_ARGUMENT",
			input: ValidateReportInput{
				InspectionID: inspectionID,
				Report:       "   ",
			},
			want: wantErrorResult("INVALID_ARGUMENT", "report is required."),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, _, err := handler.handleValidateReport(context.Background(), nil, tc.input)
			if err != nil {
				t.Fatalf("handleValidateReport() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, res, ignoreMCPUnexported); diff != "" {
				t.Errorf("handleValidateReport() result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWorkbenchValidate_Template(t *testing.T) {
	templates := mdtemplate.MustParse(templateFS, "templates/*.md.tmpl")

	testCases := []struct {
		name string
		data *workbench.ReportValidationResult
		want string
	}{
		{
			name: "zero findings",
			data: &workbench.ReportValidationResult{
				Rules: []workbench.ValidationRuleSummary{
					{
						ID:           workbench.RuleIDLogExistence,
						Description:  "Inline code spans (`...`) must exist in the inspection logs.",
						FindingCount: 0,
					},
				},
			},
			want: strings.Join([]string{
				"# Report validation",
				"",
				"Found 0 issues.",
				"",
				"## Rules checked",
				"- `log-existence`: Inline code spans (`...`) must exist in the inspection logs. (0 findings)",
			}, "\n"),
		},
		{
			name: "with findings",
			data: &workbench.ReportValidationResult{
				Rules: []workbench.ValidationRuleSummary{
					{
						ID:           workbench.RuleIDLogExistence,
						Description:  "Inline code spans (`...`) must exist in the inspection logs.",
						FindingCount: 2,
					},
				},
				Findings: []workbench.ValidationFinding{
					{
						RuleID:  workbench.RuleIDLogExistence,
						Line:    5,
						Target:  "ghost-insert-id",
						Message: "Not found in inspection logs.",
					},
					{
						RuleID:  workbench.RuleIDLogExistence,
						Line:    12,
						Target:  "default/ghost-pod",
						Message: "Not found in inspection logs (unmatched component: `ghost-pod`).",
					},
				},
			},
			want: strings.Join([]string{
				"# Report validation",
				"",
				"Found 2 potential grounding issues.",
				"",
				"## Rules checked",
				"- `log-existence`: Inline code spans (`...`) must exist in the inspection logs. (2 findings)",
				"",
				"## Findings",
				"Verify each flagged item with `search_timelines`, `search_logs`, or `get_log`, and remove or correct values that do not exist in the inspection.",
				"",
				"| Line | Rule | Target | Message |",
				"| --- | --- | --- | --- |",
				"| 5 | `log-existence` | `ghost-insert-id` | Not found in inspection logs. |",
				"| 12 | `log-existence` | `default/ghost-pod` | Not found in inspection logs (unmatched component: `ghost-pod`). |",
			}, "\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := templates.Render("validate_report.md.tmpl", tc.data)
			if err != nil {
				t.Fatalf("templates.Render() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("templates.Render() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
