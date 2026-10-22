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

func TestParseReportDocument(t *testing.T) {
	testCases := []struct {
		name      string
		report    string
		wantSpans []InlineCodeSpan
	}{
		{
			name:   "extracts single and multiple inline code spans with line numbers",
			report: "# Report\nPod `pod-a` failed with `OOMKilled`.\nInsert ID: `abc123`.",
			wantSpans: []InlineCodeSpan{
				{Text: "pod-a", Line: 2},
				{Text: "OOMKilled", Line: 2},
				{Text: "abc123", Line: 3},
			},
		},
		{
			name:   "skips backtick and tilde fenced code blocks including blockquote fences",
			report: "Before `outside-1`\n```yaml\nname: `inside-backtick-fence`\n```\nMiddle `outside-2`\n> ~~~bash\n> echo `inside-tilde-fence`\n> ~~~\nAfter `outside-3`",
			wantSpans: []InlineCodeSpan{
				{Text: "outside-1", Line: 1},
				{Text: "outside-2", Line: 5},
				{Text: "outside-3", Line: 9},
			},
		},
		{
			name:   "supports multi-backtick inline code spans and strips one symmetric space",
			report: "Inline `` `nested` `` and ```triple-inline``` on one line.",
			wantSpans: []InlineCodeSpan{
				{Text: "`nested`", Line: 1},
				{Text: "triple-inline", Line: 1},
			},
		},
		{
			name:      "ignores unmatched backticks, escaped backticks, and whitespace-only code spans",
			report:    "Unmatched `backtick on line 1\nEmpty `   ` span on line 2.\nEscaped \\`not-a-span\\` and \\\\`valid-after-double-backslash` on line 3.",
			wantSpans: []InlineCodeSpan{{Text: "valid-after-double-backslash", Line: 3}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := ParseReportDocument(tc.report)
			if diff := cmp.Diff(tc.wantSpans, doc.InlineCodeSpans); diff != "" {
				t.Errorf("ParseReportDocument() InlineCodeSpans mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type stubValidationRule struct {
	id          string
	description string
	findings    []ValidationFinding
}

func (r *stubValidationRule) ID() string          { return r.id }
func (r *stubValidationRule) Description() string { return r.description }
func (r *stubValidationRule) Validate(_ context.Context, _ *ReportDocument, _ *Workbench) ([]ValidationFinding, error) {
	return r.findings, nil
}

func TestValidateReport_MultipleRules(t *testing.T) {
	wb := setupValidateReportTestWorkbench(t)

	rules := []ValidationRule{
		NewInlineCodeGroundingRule(),
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
	if res.InlineCodeSpanCount != 2 {
		t.Errorf("InlineCodeSpanCount = %d, want 2", res.InlineCodeSpanCount)
	}
	want := []ValidationFinding{
		{
			RuleID:  "custom-rule",
			Line:    1,
			Target:  "line-1-target",
			Message: "Custom finding on line 1.",
		},
		{
			RuleID:  RuleIDUngroundedInlineCode,
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
