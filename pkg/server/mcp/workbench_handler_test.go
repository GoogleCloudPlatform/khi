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

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestInspectionServer(t *testing.T) (*coreinspection.InspectionTaskServer, string) {
	t.Helper()
	ioConfig, err := inspectioncore.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("failed to create test IOConfig: %v", err)
	}
	tempDir := t.TempDir()
	ioConfig.DataDestination = tempDir
	ioConfig.TemporaryFolder = tempDir
	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("failed to create inspection server: %v", err)
	}

	inspectionType := coreinspection.InspectionType{
		Id:   "test-type",
		Name: "Test Type",
	}
	if err := server.AddInspectionType(inspectionType); err != nil {
		t.Fatalf("failed to add inspection type: %v", err)
	}

	dummyTaskID := taskid.NewDefaultImplementationID[any]("dummy-task")
	dummyTask := coretask.NewTask(
		dummyTaskID,
		nil,
		func(ctx context.Context) (any, error) {
			return "success", nil
		},
		coretask.WithLabelValue(inspectioncore.LabelKeyInspectionDefaultFeatureFlag, true),
		coretask.WithLabelValue(inspectioncore.LabelKeyInspectionFeatureFlag, true),
	)
	if err := server.AddTask(dummyTask); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	inspectionID, err := server.CreateInspection("test-type")
	if err != nil {
		t.Fatalf("failed to create inspection: %v", err)
	}
	runner := server.GetInspection(inspectionID)
	if err := runner.Run(context.Background(), &inspectioncore.InspectionRequest{Values: map[string]any{}}); err != nil {
		t.Fatalf("failed to run inspection: %v", err)
	}
	<-runner.Wait()
	return server, inspectionID
}

func extractToolResultText(t *testing.T, res *mcpsdk.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("len(res.Content) = %d, want 1", len(res.Content))
	}
	tc, ok := res.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("res.Content[0] type = %T, want *mcpsdk.TextContent", res.Content[0])
	}
	return tc.Text
}

func TestWorkbenchHandler_Golden(t *testing.T) {
	testCases := []struct {
		name       string
		template   string
		goldenFile string
		data       workbenchLoadingData
	}{
		{
			name:       "WorkbenchLoading",
			template:   "workbench_loading.md.tmpl",
			goldenFile: "testdata/workbench_loading.golden.md",
			data: workbenchLoadingData{
				ID:            "2026-09-24-0130-a1b2",
				Stage:         "BUILDING_INDEX",
				ProgressRatio: 0.65,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			templates := mdtemplate.MustParse(templateFS, "templates/*.md.tmpl")
			res, _, err := templates.ErrorToolResult(tc.template, tc.data)
			if err != nil {
				t.Fatalf("ErrorToolResult() unexpected error = %v", err)
			}
			if !res.IsError {
				t.Errorf("res.IsError = %v, want true", res.IsError)
			}
			got := extractToolResultText(t, res)
			want := readGolden(t, tc.goldenFile)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", tc.template, diff)
			}
		})
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
				server, inspectionID := setupTestInspectionServer(t)
				indexMgr := workbench.NewInspectionIndexManager(server, t.TempDir())
				mgr := workbench.NewWorkbenchManager(server, indexMgr, 5)
				handler := NewWorkbenchHandler(mgr)

				ctx := context.Background()
				browserWB, err := mgr.Open(ctx, inspectionID, workbench.AccessorBrowser, func(apiv1.OpenWorkbenchResponse_Stage, float64, string) error { return nil })
				if err != nil {
					t.Fatalf("mgr.Open() unexpected error = %v", err)
				}
				if err := browserWB.AwaitIndex(ctx); err != nil {
					t.Fatalf("browserWB.AwaitIndex() unexpected error = %v", err)
				}

				t.Cleanup(func() {
					cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					if state, _, _, _ := browserWB.IndexStatus(); state == workbench.IndexStateBuilding {
						_ = browserWB.AwaitIndex(cleanupCtx)
					}
					browserWB.Close()
					indexMgr.Wait()
				})

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
				server, _ := setupTestInspectionServer(t)
				indexMgr := workbench.NewInspectionIndexManager(server, t.TempDir())
				mgr := workbench.NewWorkbenchManager(server, indexMgr, 5)
				handler := NewWorkbenchHandler(mgr)

				t.Cleanup(func() {
					indexMgr.Wait()
				})

				ctx := context.Background()
				gotWB, toolRes, err := handler.acquireWorkbench(ctx, "unknown-id")
				if err != nil {
					t.Fatalf("acquireWorkbench() unexpected error = %v", err)
				}
				if gotWB != nil {
					t.Errorf("gotWB = %v, want nil", gotWB)
				}
				if toolRes == nil {
					t.Fatalf("toolRes = nil, want non-nil")
				}
				if !toolRes.IsError {
					t.Errorf("toolRes.IsError = %v, want true", toolRes.IsError)
				}
				gotText := extractToolResultText(t, toolRes)
				wantText := mdtemplate.FormatError("INSPECTION_NOT_FOUND",
					"Inspection `unknown-id` not found.",
					"Read `khi://inspections` to see existing inspections.")
				if diff := cmp.Diff(wantText, gotText); diff != "" {
					t.Errorf("acquireWorkbench() error text mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "returns WORKBENCH_LOADING when wait timeout expires and succeeds on retry",
			run: func(t *testing.T) {
				server, inspectionID := setupTestInspectionServer(t)
				indexMgr := workbench.NewInspectionIndexManager(server, t.TempDir())
				mgr := workbench.NewWorkbenchManager(server, indexMgr, 5)
				handler := NewWorkbenchHandler(mgr)
				handler.waitTimeout = 0

				var loadedWB *workbench.Workbench
				t.Cleanup(func() {
					cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					if loadedWB != nil {
						if state, _, _, _ := loadedWB.IndexStatus(); state == workbench.IndexStateBuilding {
							_ = loadedWB.AwaitIndex(cleanupCtx)
						}
						loadedWB.Close()
					}
					indexMgr.Wait()
				})

				ctx := context.Background()
				gotWB, toolRes, err := handler.acquireWorkbench(ctx, inspectionID)
				if err != nil {
					t.Fatalf("first acquireWorkbench() unexpected error = %v", err)
				}
				if gotWB != nil {
					t.Errorf("first acquireWorkbench() gotWB = %v, want nil", gotWB)
				}
				if toolRes == nil {
					t.Fatalf("first acquireWorkbench() toolRes = nil, want non-nil")
				}
				if !toolRes.IsError {
					t.Errorf("first toolRes.IsError = %v, want true", toolRes.IsError)
				}
				gotText := extractToolResultText(t, toolRes)
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
				loadedWB = retryWB
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}
