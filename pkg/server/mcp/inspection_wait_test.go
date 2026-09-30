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
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestInspectionWait_Golden(t *testing.T) {
	server, err := coreinspection.NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	handler := NewInspectionHandler(server)

	testCases := []struct {
		name       string
		template   string
		goldenFile string
		data       any
	}{
		{
			name:       "WaitInspection_Running",
			template:   "wait_inspection.md.tmpl",
			goldenFile: "testdata/wait_inspection_running.golden.md",
			data: waitInspectionData{
				ID:            "2026-09-24-0130-a1b2",
				Phase:         "RUNNING",
				ProgressRatio: 0.42,
				RunningTasks: []runningTaskRow{
					{
						Label:    "Query K8s audit logs",
						Progress: "60%",
						Message:  "91200 logs fetched",
					},
				},
			},
		},
		{
			name:       "WaitInspection_Done",
			template:   "wait_inspection.md.tmpl",
			goldenFile: "testdata/wait_inspection_done.golden.md",
			data: waitInspectionData{
				ID:    "2026-09-24-0130-a1b2",
				Phase: "DONE",
			},
		},
		{
			name:       "WaitInspection_Error",
			template:   "wait_inspection.md.tmpl",
			goldenFile: "testdata/wait_inspection_error.golden.md",
			data: waitInspectionData{
				ID:    "2026-09-24-0130-a1b2",
				Phase: "ERROR",
				FailedTasks: []failedTaskRow{
					{
						Title:  "Query K8s audit logs",
						TaskID: "k8s-audit-log#default",
						Error:  "PERMISSION_DENIED: logging.logEntries.list on project my-gcp-project",
					},
				},
			},
		},
		{
			name:       "WaitInspection_Cancelled",
			template:   "wait_inspection.md.tmpl",
			goldenFile: "testdata/wait_inspection_cancelled.golden.md",
			data: waitInspectionData{
				ID:    "2026-09-24-0130-a1b2",
				Phase: "CANCELLED",
			},
		},
		{
			name:       "CancelInspection",
			template:   "cancel_inspection.md.tmpl",
			goldenFile: "testdata/cancel_inspection.golden.md",
			data: inspectionIDData{
				ID: "2026-09-24-0130-a1b2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := handler.templates.Render(tc.template, tc.data)
			if err != nil {
				t.Fatalf("failed to render %s: %v", tc.template, err)
			}

			want := readGolden(t, tc.goldenFile)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", tc.template, diff)
			}
		})
	}
}

// waitBlockingTaskStarted waits until the blocking feature task reports progress in run mode.
func (e *toolsTestEnv) waitBlockingTaskStarted(t *testing.T) {
	t.Helper()
	select {
	case <-e.blockingTaskStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for the blocking feature task to start")
	}
}

func TestInspectionTools_WaitInspection(t *testing.T) {
	env := newToolsTestEnv(t)
	id := env.createInspection(t)
	env.runInspection(t, id, map[string]any{"cluster-name": "prod-cluster-1"})
	env.waitBlockingTaskStarted(t)

	t.Run("zero timeout returns the running status immediately", func(t *testing.T) {
		start := time.Now()
		text, isError := callTool(t, env.ctx, env.session, "wait_inspection", map[string]any{
			"inspectionId":   id,
			"timeoutSeconds": 0,
		})
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("wait_inspection with timeoutSeconds 0 took %v, want less than 5s", elapsed)
		}
		if isError {
			t.Fatalf("wait_inspection returned error: %s", text)
		}
		// The total percentage and the other task rows depend on how framework tasks are scheduled,
		// so only the status line and the row of the blocking task are checked.
		wantPrefix := fmt.Sprintf("# Inspection `%s`\n\nStatus: RUNNING, ", id)
		if !strings.HasPrefix(text, wantPrefix) {
			t.Errorf("wait_inspection text = %q, want prefix %q", text, wantPrefix)
		}
		wantRow := "| Query K8s audit logs | 60% | 91200 logs fetched |"
		if !strings.Contains(text, wantRow) {
			t.Errorf("wait_inspection text = %q, want row %q", text, wantRow)
		}
	})

	t.Run("returns before the timeout when finished", func(t *testing.T) {
		type callResult struct {
			res *mcpsdk.CallToolResult
			err error
		}
		resultCh := make(chan callResult, 1)
		go func() {
			res, err := env.session.CallTool(env.ctx, &mcpsdk.CallToolParams{
				Name: "wait_inspection",
				Arguments: map[string]any{
					"inspectionId":   id,
					"timeoutSeconds": 30,
				},
			})
			resultCh <- callResult{res: res, err: err}
		}()

		env.releaseBlockingTask()

		select {
		case r := <-resultCh:
			if r.err != nil {
				t.Fatalf("session.CallTool(wait_inspection) failed: %v", r.err)
			}
			text, isError := toolResultText(t, "wait_inspection", r.res)
			want := fmt.Sprintf("# Inspection `%s`\n\nStatus: DONE. Read `khi://inspections/%s` for the summary.", id, id)
			checkToolResult(t, "wait_inspection", text, isError, want, false)
		case <-time.After(10 * time.Second):
			t.Fatal("wait_inspection did not return within 10s after the inspection finished")
		}
	})
}

func TestInspectionTools_CancelInspection(t *testing.T) {
	env := newToolsTestEnv(t)
	id := env.createInspection(t)
	env.runInspection(t, id, map[string]any{"cluster-name": "prod-cluster-1"})
	env.waitBlockingTaskStarted(t)

	steps := []struct {
		name        string
		tool        string
		args        map[string]any
		want        string
		wantIsError bool
	}{
		{
			name:        "cancel running inspection",
			tool:        "cancel_inspection",
			args:        map[string]any{"inspectionId": id},
			want:        fmt.Sprintf("# Cancelled `%s`\n\nStatus: CANCELLED", id),
			wantIsError: false,
		},
		{
			name:        "cancel again",
			tool:        "cancel_inspection",
			args:        map[string]any{"inspectionId": id},
			want:        "Error: INSPECTION_ALREADY_FINISHED\n\n- Call `wait_inspection` to read the final status.",
			wantIsError: true,
		},
		{
			name:        "wait reports cancelled",
			tool:        "wait_inspection",
			args:        map[string]any{"inspectionId": id, "timeoutSeconds": 5},
			want:        fmt.Sprintf("# Inspection `%s`\n\nStatus: CANCELLED.", id),
			wantIsError: false,
		},
	}

	// Steps run in order because each one depends on the state left by the previous step.
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			text, isError := callTool(t, env.ctx, env.session, step.tool, step.args)
			checkToolResult(t, step.tool, text, isError, step.want, step.wantIsError)
		})
	}
}

func TestInspectionTools_TaskError(t *testing.T) {
	env := newToolsTestEnv(t)

	testCases := []struct {
		name       string
		featureIDs []string
		params     map[string]any
		// wantRows are the expected error table rows.
		wantRows []string
	}{
		{
			name:       "task with a title shows the title and its own error",
			featureIDs: []string{failFeatureID},
			params:     map[string]any{},
			wantRows:   []string{"| Fail task | network connection lost |"},
		},
		{
			name:       "task without a title shows the task ID as code",
			featureIDs: []string{untitledFailFeatureID},
			params:     map[string]any{},
			wantRows:   []string{"| `untitled-fail-feature#default` | disk quota exceeded |"},
		},
		{
			name:       "tasks cancelled by another failure are omitted",
			featureIDs: []string{blockingFeatureID, failFeatureID},
			params:     map[string]any{"cluster-name": "prod-cluster-1"},
			wantRows:   []string{"| Fail task | network connection lost |"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := env.createInspection(t, tc.featureIDs...)
			env.runInspection(t, id, tc.params)

			text, isError := callTool(t, env.ctx, env.session, "wait_inspection", map[string]any{
				"inspectionId":   id,
				"timeoutSeconds": 30,
			})
			want := fmt.Sprintf("# Inspection `%s`\n\nStatus: ERROR.\n\n| Task | Error |\n| --- | --- |\n%s", id, strings.Join(tc.wantRows, "\n"))
			checkToolResult(t, "wait_inspection", text, isError, want, false)
		})
	}
}

func TestFailedTaskRows(t *testing.T) {
	failedErr := errors.New("network connection lost")
	testCases := []struct {
		name     string
		statuses map[string]coretask.TaskRunStatus
		want     []failedTaskRow
	}{
		{
			name: "failed tasks are sorted by ID and other phases are skipped",
			statuses: map[string]coretask.TaskRunStatus{
				"b-task#default": {Phase: coretask.TaskRunPhaseError, Error: errors.New("disk quota exceeded")},
				"a-task#default": {Phase: coretask.TaskRunPhaseError, Error: failedErr},
				"c-task#default": {Phase: coretask.TaskRunPhaseDone},
				"d-task#default": {Phase: coretask.TaskRunPhaseWaiting},
			},
			want: []failedTaskRow{
				{TaskID: "a-task#default", Error: "network connection lost"},
				{TaskID: "b-task#default", Error: "disk quota exceeded"},
			},
		},
		{
			name: "tasks cancelled by another failure are omitted",
			statuses: map[string]coretask.TaskRunStatus{
				"a-task#default": {Phase: coretask.TaskRunPhaseError, Error: fmt.Errorf("query aborted: %w", context.Canceled)},
				"b-task#default": {Phase: coretask.TaskRunPhaseError, Error: failedErr},
			},
			want: []failedTaskRow{
				{TaskID: "b-task#default", Error: "network connection lost"},
			},
		},
		{
			name: "cancelled tasks are listed when no task failed for another reason",
			statuses: map[string]coretask.TaskRunStatus{
				"a-task#default": {Phase: coretask.TaskRunPhaseError, Error: context.Canceled},
			},
			want: []failedTaskRow{
				{TaskID: "a-task#default", Error: "context canceled"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := failedTaskRows(&coreinspection.RunTaskGraphSnapshot{TaskRunStatuses: tc.statuses})
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("failedTaskRows() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInspectionTools_WaitImportedInspection(t *testing.T) {
	env := newToolsTestEnv(t)
	const id = "imported-id"
	// Imported inspections carry a header but no progress metadata because they never ran in this process.
	md := typedmap.NewTypedMap()
	typedmap.Set(md, inspectionmetadata.HeaderMetadataKey, &inspectionmetadata.HeaderMetadata{
		InspectionName: "imported inspection",
		InspectionType: "Google Kubernetes Engine",
	})
	env.server.RegisterImportedInspection(id, nil, md.AsReadonly())

	text, isError := callTool(t, env.ctx, env.session, "wait_inspection", map[string]any{
		"inspectionId":   id,
		"timeoutSeconds": 5,
	})
	want := fmt.Sprintf("# Inspection `%s`\n\nStatus: DONE. Read `khi://inspections/%s` for the summary.", id, id)
	checkToolResult(t, "wait_inspection", text, isError, want, false)
}
