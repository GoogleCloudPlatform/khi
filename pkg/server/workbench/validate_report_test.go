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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
)

func setupValidateReportTestWorkbench(t *testing.T) *Workbench {
	t.Helper()
	wb := NewWorkbench("wb-validate", "insp-validate")
	pool := khifilev6model.NewTestInternPool(id.NewGenerator())

	log1YAML := "insertId: abc123valid\nresource:\n  labels:\n    namespace_name: default\n    pod_name: frontend-pod-7f8b9c\nmetadata:\n  labels:\n    app: nginx\nstatus:\n  exitCode: 137\ntimestamp: \"2026-10-01T10:15:00Z\"\n"
	log2YAML := "insertId: xyz987valid\nkind: Pod\nmessage: Container crashed with OOMKilled\n"

	logs := []cel.LogData{
		{
			ID:              1,
			Timestamp:       1000,
			SummaryStringID: pool.InternString("Pod frontend-pod-7f8b9c started on node-pool-1").ID(),
			BodyStructID:    internTestYAML(t, pool, log1YAML),
		},
		{
			ID:              2,
			Timestamp:       2000,
			SummaryStringID: pool.InternString("Container OOMKilled in namespace kube-system").ID(),
			BodyStructID:    internTestYAML(t, pool, log2YAML),
		},
	}

	readonlyPool := newReadonlyInternPool(pool)
	trigramIdx := cel.NewTrigramIndex()
	err := trigramIdx.BuildFromLogPool(readonlyPool, []cel.LogTrigramItem{
		{ID: 1, SummaryStringID: logs[0].SummaryStringID, BodyStructID: logs[0].BodyStructID},
		{ID: 2, SummaryStringID: logs[1].SummaryStringID, BodyStructID: logs[1].BodyStructID},
	}, nil)
	if err != nil {
		t.Fatalf("BuildFromLogPool() failed: %v", err)
	}

	wb.internPool = readonlyPool
	wb.searchIndex = &SearchIndex{
		Logs:         logs,
		InternPool:   readonlyPool,
		TrigramIndex: trigramIdx,
	}
	return wb
}

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
			name:      "ignores unmatched backticks and whitespace-only code spans",
			report:    "Unmatched `backtick on line 1\nEmpty `   ` span on line 2.",
			wantSpans: nil,
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

func TestValidateReport(t *testing.T) {
	wb := setupValidateReportTestWorkbench(t)

	testCases := []struct {
		name      string
		report    string
		rules     []ValidationRule
		wantCount int
		want      []ValidationFinding
	}{
		{
			name: "passes when all inline code spans match log summaries, YAML bodies, or composite notations",
			report: strings.Join([]string{
				"Pod `frontend-pod-7f8b9c` on `node-pool-1` exited with `137` (`OOMKilled`) at `2026-10-01T10:15:00Z`.",
				"Evidence log insertId is `abc123valid` for `default/frontend-pod-7f8b9c` (`Pod/frontend-pod-7f8b9c`, label `app=nginx`).",
				"```yaml",
				"insertId: `hallucinated-inside-fenced-block-is-ignored`",
				"```",
			}, "\n"),
			wantCount: 9,
			want:      nil,
		},
		{
			name: "flags hallucinated insertId, pod name, and unmatched composite components",
			report: strings.Join([]string{
				"1. Valid pod `frontend-pod-7f8b9c` and hallucinated insertId `hallucinated-insert-id`.",
				"2. Partially hallucinated namespace/pod `default/ghost-pod-999`.",
				"3. Completely hallucinated composite `ghost-ns/ghost-pod`.",
				"4. Partially hallucinated label `app=ghost-app`.",
			}, "\n"),
			wantCount: 5,
			want: []ValidationFinding{
				{
					RuleID:  RuleIDUngroundedInlineCode,
					Line:    1,
					Target:  "hallucinated-insert-id",
					Message: "Not found in inspection logs.",
				},
				{
					RuleID:  RuleIDUngroundedInlineCode,
					Line:    2,
					Target:  "default/ghost-pod-999",
					Message: "Not found in inspection logs (unmatched component: `ghost-pod-999`).",
				},
				{
					RuleID:  RuleIDUngroundedInlineCode,
					Line:    3,
					Target:  "ghost-ns/ghost-pod",
					Message: "Not found in inspection logs (unmatched components: `ghost-ns`, `ghost-pod`).",
				},
				{
					RuleID:  RuleIDUngroundedInlineCode,
					Line:    4,
					Target:  "app=ghost-app",
					Message: "Not found in inspection logs (unmatched component: `ghost-app`).",
				},
			},
		},
		{
			name:   "aggregates and sorts findings across multiple ValidationRules",
			report: "Line 1 `frontend-pod-7f8b9c`\nLine 2 `missing-pod`",
			rules: []ValidationRule{
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
			},
			wantCount: 2,
			want: []ValidationFinding{
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
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := wb.ValidateReport(context.Background(), tc.report, tc.rules...)
			if err != nil {
				t.Fatalf("ValidateReport() unexpected error = %v", err)
			}
			if res.InlineCodeSpanCount != tc.wantCount {
				t.Errorf("InlineCodeSpanCount = %d, want %d", res.InlineCodeSpanCount, tc.wantCount)
			}
			if diff := cmp.Diff(tc.want, res.Findings); diff != "" {
				t.Errorf("ValidateReport() Findings mismatch (-want +got):\n%s", diff)
			}
		})
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
