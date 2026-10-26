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
	"time"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/inspectionserver"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var ignoreMCPUnexported = cmpopts.IgnoreUnexported(
	mcpsdk.CallToolResult{},
	mcpsdk.ImageContent{},
	mcpsdk.TextContent{},
)

func newTestWorkbenchManager(t *testing.T) (*workbench.WorkbenchManager, string) {
	t.Helper()
	server, inspectionID := inspectionserver.NewServerWithInspection(t, inspectionserver.FeatureTask())
	indexMgr := workbench.NewInspectionIndexManager(server, t.TempDir())
	mgr := workbench.NewWorkbenchManager(server, indexMgr, 5)
	t.Cleanup(indexMgr.Wait)
	return mgr, inspectionID
}

func closeWorkbench(t *testing.T, wb *workbench.Workbench) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if state, _, _, _ := wb.IndexStatus(); state == workbench.IndexStateBuilding {
			_ = wb.AwaitIndex(cleanupCtx)
		}
		wb.Close()
	})
}

func wantTextResult(text string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}
}

func wantErrorResult(code string, bullets ...string) *mcpsdk.CallToolResult {
	res, _, _ := mdtemplate.ErrorResult(code, bullets...)
	return res
}

func TestWorkbenchHandler_Golden(t *testing.T) {
	templates := mdtemplate.MustParse(templateFS, "templates/*.md.tmpl")
	res, _, err := templates.ErrorToolResult("workbench_loading.md.tmpl", workbenchLoadingData{
		ID:            "2026-09-24-0130-a1b2",
		Stage:         "BUILDING_INDEX",
		ProgressRatio: 0.65,
	})
	if err != nil {
		t.Fatalf("ErrorToolResult() unexpected error = %v", err)
	}
	want := &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: readGolden(t, "testdata/workbench_loading.golden.md")},
		},
	}
	if diff := cmp.Diff(want, res, ignoreMCPUnexported); diff != "" {
		t.Errorf("workbench_loading.md.tmpl mismatch (-want +got):\n%s", diff)
	}
}

func TestWorkbenchHandler_AcquireWorkbench(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "shares workbench opened by browser without reloading",
			run: func(t *testing.T) {
				mgr, inspectionID := newTestWorkbenchManager(t)
				handler := NewWorkbenchHandler(mgr, nil)

				ctx := context.Background()
				browserWB, err := mgr.Open(ctx, inspectionID, workbench.AccessorBrowser, func(apiv1.OpenWorkbenchResponse_Stage, float64, string) error { return nil })
				if err != nil {
					t.Fatalf("mgr.Open() unexpected error = %v", err)
				}
				closeWorkbench(t, browserWB)
				if err := browserWB.AwaitIndex(ctx); err != nil {
					t.Fatalf("browserWB.AwaitIndex() unexpected error = %v", err)
				}

				gotWB, toolRes, err := handler.acquireWorkbench(ctx, inspectionID)
				if err != nil {
					t.Fatalf("acquireWorkbench() unexpected error = %v", err)
				}
				if toolRes != nil {
					t.Fatalf("acquireWorkbench() toolRes = %v, want nil", toolRes)
				}
				if gotWB != browserWB {
					t.Errorf("gotWB = %p, want %p", gotWB, browserWB)
				}
			},
		},
		{
			name: "returns inspection not found for unknown inspection ID",
			run: func(t *testing.T) {
				mgr, _ := newTestWorkbenchManager(t)
				handler := NewWorkbenchHandler(mgr, nil)

				gotWB, toolRes, err := handler.acquireWorkbench(context.Background(), "unknown-id")
				if err != nil {
					t.Fatalf("acquireWorkbench() unexpected error = %v", err)
				}
				if gotWB != nil {
					t.Errorf("gotWB = %v, want nil", gotWB)
				}
				wantRes := wantErrorResult("INSPECTION_NOT_FOUND",
					"Inspection `unknown-id` not found.",
					"Read `khi://inspections` to see existing inspections.")
				if diff := cmp.Diff(wantRes, toolRes, ignoreMCPUnexported); diff != "" {
					t.Errorf("acquireWorkbench() toolRes mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "returns WORKBENCH_LOADING when wait timeout expires and succeeds on retry",
			run: func(t *testing.T) {
				mgr, inspectionID := newTestWorkbenchManager(t)
				handler := NewWorkbenchHandler(mgr, nil)
				handler.waitTimeout = 0

				ctx := context.Background()
				gotWB, toolRes, err := handler.acquireWorkbench(ctx, inspectionID)
				if err != nil {
					t.Fatalf("first acquireWorkbench() unexpected error = %v", err)
				}
				if gotWB != nil {
					t.Errorf("first acquireWorkbench() gotWB = %v, want nil", gotWB)
				}
				gotText, isErr := toolResultText(t, "acquireWorkbench", toolRes)
				if !isErr {
					t.Errorf("first toolRes.IsError = false, want true")
				}
				if !strings.HasPrefix(gotText, "Error: WORKBENCH_LOADING") {
					t.Errorf("gotText = %q, want prefix 'Error: WORKBENCH_LOADING'", gotText)
				}
				if !strings.Contains(gotText, inspectionID) {
					t.Errorf("gotText = %q, want containing inspection ID %q", gotText, inspectionID)
				}

				handler.waitTimeout = 10 * time.Second
				retryWB, retryToolRes, err := handler.acquireWorkbench(ctx, inspectionID)
				if err != nil {
					t.Fatalf("retry acquireWorkbench() unexpected error = %v", err)
				}
				if retryToolRes != nil {
					t.Fatalf("retry acquireWorkbench() toolRes = %v, want nil", retryToolRes)
				}
				if retryWB == nil {
					t.Fatalf("retry acquireWorkbench() gotWB = nil, want non-nil")
				}
				closeWorkbench(t, retryWB)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}
