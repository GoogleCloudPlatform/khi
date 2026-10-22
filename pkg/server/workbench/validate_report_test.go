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

func (r *stubValidationRule) ID() string          { return r.id }
func (r *stubValidationRule) Description() string { return r.description }
func (r *stubValidationRule) Validate(_ context.Context, _ string, _ *Workbench) ([]ValidationFinding, error) {
	return r.findings, nil
}

func TestValidateReport_MultipleRules(t *testing.T) {
	wb := setupValidateReportTestWorkbench(t)

	rules := []ValidationRule{
		NewLogExistenceRule(),
		&stubValidationRule{
			id:          "custom-rule",
			description: "Custom validation rule.",
			findings: []ValidationFinding{
				{
					RuleID:  "custom-rule",
					Line:    1,
					Target:  "line-1-target",
					Message: "Custom finding on line 1.",
				},
			},
		},
	}

	res, err := wb.ValidateReport(context.Background(), "Line 1 `frontend-pod-7f8b9c`\nLine 2 `missing-pod`", rules...)
	if err != nil {
		t.Fatalf("ValidateReport() unexpected error = %v", err)
	}
	want := []ValidationFinding{
		{
			RuleID:  "custom-rule",
			Line:    1,
			Target:  "line-1-target",
			Message: "Custom finding on line 1.",
		},
		{
			RuleID:  RuleIDLogExistence,
			Line:    2,
			Target:  "missing-pod",
			Message: "Not found in inspection logs.",
		},
	}
	if diff := cmp.Diff(want, res.Findings); diff != "" {
		t.Errorf("ValidateReport() Findings mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateReport_ClosedWorkbench(t *testing.T) {
	wb := setupValidateReportTestWorkbench(t)
	wb.Close()

	_, err := wb.ValidateReport(context.Background(), "`frontend-pod-7f8b9c`")
	if !errors.Is(err, ErrWorkbenchClosed) {
		t.Errorf("ValidateReport() error = %v, want %v", err, ErrWorkbenchClosed)
	}
}
