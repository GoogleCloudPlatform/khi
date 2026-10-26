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

package workbench

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type stubValidationRule struct {
	id          string
	description string
	findings    []ValidationFinding
}

var _ ValidationRule = (*stubValidationRule)(nil)

func (r *stubValidationRule) ID() string          { return r.id }
func (r *stubValidationRule) Description() string { return r.description }
func (r *stubValidationRule) Validate(_ context.Context, _ string, _ *Workbench) ([]ValidationFinding, error) {
	return r.findings, nil
}

func TestValidateReport(t *testing.T) {
	customFinding := ValidationFinding{
		RuleID:  "custom-rule",
		Line:    1,
		Target:  "line-1-target",
		Message: "Custom finding on line 1.",
	}
	testCases := []struct {
		name          string
		report        string
		rules         []ValidationRule
		modify        func(*Workbench)
		wantFindings  []ValidationFinding
		wantErrTarget error
	}{
		{
			name:   "combines findings across multiple validation rules",
			report: "Line 1 `frontend-pod-7f8b9c`\nLine 2 `missing-pod`",
			rules: []ValidationRule{
				NewLogExistenceRule(),
				&stubValidationRule{
					id:          "custom-rule",
					description: "Custom validation rule.",
					findings:    []ValidationFinding{customFinding},
				},
			},
			wantFindings: []ValidationFinding{
				customFinding,
				{
					RuleID:  RuleIDLogExistence,
					Line:    2,
					Target:  "missing-pod",
					Message: "Not found in inspection logs.",
				},
			},
		},
		{
			name:          "closed workbench returns ErrWorkbenchClosed",
			report:        "`frontend-pod-7f8b9c`",
			modify:        func(wb *Workbench) { wb.Close() },
			wantErrTarget: ErrWorkbenchClosed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupValidateReportTestWorkbench(t)
			if tc.modify != nil {
				tc.modify(wb)
			}
			res, err := wb.ValidateReport(context.Background(), tc.report, tc.rules...)
			if tc.wantErrTarget != nil {
				if !errors.Is(err, tc.wantErrTarget) {
					t.Fatalf("ValidateReport() error = %v, want %v", err, tc.wantErrTarget)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateReport() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.wantFindings, res.Findings); diff != "" {
				t.Errorf("ValidateReport() Findings mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
