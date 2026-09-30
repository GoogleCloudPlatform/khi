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
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/summary"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestTask(id string, labelOpts ...coretask.LabelOpt) coretask.UntypedTask {
	return coretask.NewTask(
		taskid.NewDefaultImplementationID[any](id),
		nil,
		func(ctx context.Context) (any, error) { return nil, nil },
		labelOpts...,
	)
}

func newTestEdge(sourceID, targetID string) taskid.TaskEdge {
	return taskid.TaskEdge{
		SourceImplID: sourceID,
		TargetImplID: targetID,
	}
}

func taskContext(c *summary.Collector, taskImplID taskid.UntypedTaskImplementationID) context.Context {
	ctx := context.Background()
	m := typedmap.NewTypedMap()
	typedmap.Set(m, summary.MetadataKey, c)
	ctx = khictx.WithValue(ctx, inspectionmetadata.MapContextKey, m.AsReadonly())
	ctx = khictx.WithValue(ctx, core_contract.TaskImplementationIDContextKey, taskImplID)
	return ctx
}

func TestInspectionHandler_SummaryGolden(t *testing.T) {
	formTask := newTestTask("form-cluster-name", inspectioncore.NewFormTaskLabelOpt("Cluster Name", "Form field"))
	listAuditTask := newTestTask("list-audit-logs", coretask.WithTitle("List Audit Logs"))
	auditMapperTask := newTestTask("audit-log-mapper", coretask.WithTitle("Audit Log Mapper"))
	featureAuditTask := newTestTask("feature-audit", inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", "", 1, true))
	inventoryTask := newTestTask("inventory-node-names")

	tasks := []coretask.UntypedTask{
		formTask,
		listAuditTask,
		auditMapperTask,
		featureAuditTask,
		inventoryTask,
	}

	edges := []taskid.TaskEdge{
		newTestEdge(formTask.UntypedID().String(), listAuditTask.UntypedID().String()),
		newTestEdge(listAuditTask.UntypedID().String(), auditMapperTask.UntypedID().String()),
		newTestEdge(auditMapperTask.UntypedID().String(), featureAuditTask.UntypedID().String()),
	}

	taskGraph := coretask.NewResolvedTaskSet(tasks, edges, nil)
	c := summary.NewCollector(taskGraph)

	ctxForm := taskContext(c, formTask.UntypedID())
	summary.SetCoreLabel(ctxForm, "clusterName", "prod-cluster-1")
	summary.SetCoreLabel(ctxForm, "projectId", "my-gcp-project")
	summary.SetProperty(ctxForm, "duration", "1h")

	ctxListAudit := taskContext(c, listAuditTask.UntypedID())
	summary.RecordQuery(ctxListAudit, "k8s-audit", "K8s audit logs", "resource.type=\"k8s_cluster\"\nresource.labels.cluster_name=\"prod-cluster-1\"")
	summary.AddIntProperty(ctxListAudit, "fetchedLogs", 1520)
	summary.AppendMarkdown(ctxListAudit, "Split the query into 4 time ranges because of the log volume.")

	ctxAuditMapper := taskContext(c, auditMapperTask.UntypedID())
	for i := 0; i < 1520; i++ {
		summary.AddIntProperty(ctxAuditMapper, "mappedLogs", 1)
		summary.AddFeatureIntProperty(ctxAuditMapper, "totalLogs", 1)
	}
	for _, v := range []string{"create", "delete", "patch", "update", "patch", "create"} {
		summary.AddToSetProperty(ctxAuditMapper, "verbs", v)
	}

	ctxFeatureAudit := taskContext(c, featureAuditTask.UntypedID())
	summary.SetProperty(ctxFeatureAudit, "timelineCount", "342")
	summary.AppendMarkdown(ctxFeatureAudit, "- 3 audit logs failed to parse and were skipped.")

	ctxInventory := taskContext(c, inventoryTask.UntypedID())
	summary.SetProperty(ctxInventory, "nodeCount", "3")

	header := &inspectionmetadata.HeaderMetadata{
		InspectionName:         "prod-cluster-1 restart investigation",
		InspectionType:         "Google Kubernetes Engine",
		StartTimeUnixSeconds:   time.Date(2026, 9, 24, 1, 0, 0, 0, time.UTC).Unix(),
		EndTimeUnixSeconds:     time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC).Unix(),
		InspectTimeUnixSeconds: time.Date(2026, 9, 24, 1, 30, 0, 0, time.UTC).Unix(),
	}

	server, err := coreinspection.NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	handler := NewInspectionHandler(server)

	startTime := time.Unix(header.StartTimeUnixSeconds, 0)
	endTime := time.Unix(header.EndTimeUnixSeconds, 0)
	data := inspectionSummaryData{
		ID:        "2026-09-24-0130-a1b2",
		Header:    header,
		TimeRange: fmt.Sprintf("%s - %s", mdtemplate.FormatTime(startTime), mdtemplate.FormatTime(endTime)),
		Summary:   c.Snapshot(),
	}

	got, err := handler.templates.Render("inspection_summary.md.tmpl", data)
	if err != nil {
		t.Fatalf("failed to render inspection summary template: %v", err)
	}

	goldenBytes, err := os.ReadFile("testdata/inspection_summary.golden.md")
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}
	want := strings.TrimRight(string(goldenBytes), "\r\n")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("inspection summary mismatch (-want +got):\n%s", diff)
	}
}

func TestMCPUnavailableReason(t *testing.T) {
	testCases := []struct {
		name string
		in   *coreinspection.InspectionType
		want string
	}{
		{
			name: "file log source",
			in: &coreinspection.InspectionType{
				Labels: map[string]string{inspectioncore.InspectionTypeLabelKeyLogSource: "file"},
			},
			want: "requires file uploads, which are not supported via MCP yet.",
		},
		{
			name: "cloud log source",
			in: &coreinspection.InspectionType{
				Labels: map[string]string{inspectioncore.InspectionTypeLabelKeyLogSource: "cloud"},
			},
			want: "",
		},
		{
			name: "nil labels",
			in: &coreinspection.InspectionType{
				Labels: nil,
			},
			want: "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := mcpUnavailableReason(tc.in)
			if got != tc.want {
				t.Errorf("mcpUnavailableReason() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInspectionHandler_E2E(t *testing.T) {
	server, err := coreinspection.NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create inspection task server: %v", err)
	}

	if err := server.AddInspectionType(coreinspection.InspectionType{
		Id:          "gcp-gke",
		Name:        "Google Kubernetes Engine",
		Description: "Gather and parse Google Kubernetes Engine (GKE) cluster logs ...",
		Priority:    100,
	}); err != nil {
		t.Fatalf("failed to add gcp-gke inspection type: %v", err)
	}

	if err := server.AddInspectionType(coreinspection.InspectionType{
		Id:          "oss-kubernetes-from-files",
		Name:        "OSS Kubernetes Log Files",
		Description: "Parse uploaded OSS Kubernetes log files to visualize cluster operations on timelines.",
		Priority:    50,
		Labels:      map[string]string{inspectioncore.InspectionTypeLabelKeyLogSource: "file"},
	}); err != nil {
		t.Fatalf("failed to add oss-kubernetes-from-files inspection type: %v", err)
	}

	if err := server.AddTask(newTestTask("k8s-audit-log", inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", "", 1, true))); err != nil {
		t.Fatalf("failed to add k8s-audit-log task: %v", err)
	}
	if err := server.AddTask(newTestTask("k8s-event-log", inspectioncore.FeatureTaskLabel("Kubernetes Event Logs", "", 2, true))); err != nil {
		t.Fatalf("failed to add k8s-event-log task: %v", err)
	}

	handler := NewInspectionHandler(server)
	mcpServer := NewServer(handler)

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpServer.HTTPHandler())

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(ctx, &mcpsdk.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("client.Connect() failed: %v", err)
	}
	defer session.Close()

	t.Run("ListTools", func(t *testing.T) {
		toolsRes, err := session.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("session.ListTools() failed: %v", err)
		}
		if len(toolsRes.Tools) != 4 {
			t.Errorf("len(toolsRes.Tools) = %d, want 4", len(toolsRes.Tools))
		}
		toolNames := make(map[string]bool)
		for _, tool := range toolsRes.Tools {
			toolNames[tool.Name] = true
		}
		for _, want := range []string{"create_inspection", "update_inspection_features", "dry_run_inspection", "run_inspection"} {
			if !toolNames[want] {
				t.Errorf("missing tool: %s", want)
			}
		}
	})

	t.Run("ListInspectionTypesResource", func(t *testing.T) {
		res, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "khi://inspection-types"})
		if err != nil {
			t.Fatalf("session.ReadResource(\"khi://inspection-types\") failed: %v", err)
		}
		if len(res.Contents) != 1 {
			t.Fatalf("len(res.Contents) = %d, want 1", len(res.Contents))
		}
		goldenBytes, err := os.ReadFile("testdata/inspection_types.golden.md")
		if err != nil {
			t.Fatalf("failed to read golden file: %v", err)
		}
		want := strings.TrimRight(string(goldenBytes), "\r\n")
		got := strings.TrimRight(res.Contents[0].Text, "\r\n")
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("inspection types resource mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("EmptyInspections", func(t *testing.T) {
		listRes, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "khi://inspections"})
		if err != nil {
			t.Fatalf("session.ReadResource(\"khi://inspections\") failed: %v", err)
		}
		if len(listRes.Contents) != 1 {
			t.Fatalf("len(listRes.Contents) = %d, want 1", len(listRes.Contents))
		}
		wantEmpty := "# Inspections\n\nNo inspections yet."
		if listRes.Contents[0].Text != wantEmpty {
			t.Errorf("empty inspections list = %q, want %q", listRes.Contents[0].Text, wantEmpty)
		}
	})

	// Register completed inspection (DONE)
	md1 := typedmap.NewTypedMap()
	header1 := &inspectionmetadata.HeaderMetadata{
		InspectionName:         "prod-cluster-1 restart investigation",
		InspectionType:         "Google Kubernetes Engine",
		StartTimeUnixSeconds:   time.Date(2026, 9, 24, 1, 0, 0, 0, time.UTC).Unix(),
		EndTimeUnixSeconds:     time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC).Unix(),
		InspectTimeUnixSeconds: 1000,
	}
	typedmap.Set(md1, inspectionmetadata.HeaderMetadataKey, header1)

	prog1 := inspectionmetadata.NewProgress()
	if err := prog1.MarkDone(); err != nil {
		t.Fatalf("prog1.MarkDone() failed: %v", err)
	}
	typedmap.Set(md1, inspectionmetadata.ProgressMetadataKey, prog1)

	taskGraph := coretask.NewResolvedTaskSet(nil, nil, nil)
	summary1 := summary.NewCollector(taskGraph)
	ctxTask1 := taskContext(summary1, taskid.NewDefaultImplementationID[any]("form-task"))
	summary.SetCoreLabel(ctxTask1, "clusterName", "prod-cluster-1")
	summary.SetCoreLabel(ctxTask1, "location", "us-central1")
	summary.SetCoreLabel(ctxTask1, "projectId", "my-gcp-project")
	typedmap.Set(md1, summary.MetadataKey, summary1)

	server.RegisterImportedInspection("2026-09-24-0130-a1b2", nil, md1.AsReadonly())

	// Register running inspection (RUNNING, newer InspectTime)
	md2 := typedmap.NewTypedMap()
	header2 := &inspectionmetadata.HeaderMetadata{
		InspectionName:         "staging-cluster investigation",
		InspectionType:         "Google Kubernetes Engine",
		InspectTimeUnixSeconds: 2000,
	}
	typedmap.Set(md2, inspectionmetadata.HeaderMetadataKey, header2)

	prog2 := inspectionmetadata.NewProgress()
	typedmap.Set(md2, inspectionmetadata.ProgressMetadataKey, prog2)

	server.RegisterImportedInspection("2026-09-24-0200-c3d4", nil, md2.AsReadonly())

	// Register failed inspection (ERROR)
	md3 := typedmap.NewTypedMap()
	header3 := &inspectionmetadata.HeaderMetadata{
		InspectionName:         "failed investigation",
		InspectionType:         "Google Kubernetes Engine",
		InspectTimeUnixSeconds: 500,
	}
	typedmap.Set(md3, inspectionmetadata.HeaderMetadataKey, header3)

	prog3 := inspectionmetadata.NewProgress()
	if err := prog3.MarkError(); err != nil {
		t.Fatalf("prog3.MarkError() failed: %v", err)
	}
	typedmap.Set(md3, inspectionmetadata.ProgressMetadataKey, prog3)

	server.RegisterImportedInspection("2026-09-24-0010-e5f6", nil, md3.AsReadonly())

	t.Run("ListInspections", func(t *testing.T) {
		listRes2, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "khi://inspections"})
		if err != nil {
			t.Fatalf("session.ReadResource(\"khi://inspections\") with data failed: %v", err)
		}
		listText := listRes2.Contents[0].Text

		if !strings.Contains(listText, "3 inspections, newest first.") {
			t.Errorf("listText missing header line: %s", listText)
		}
		idx2 := strings.Index(listText, "2026-09-24-0200-c3d4")
		idx1 := strings.Index(listText, "2026-09-24-0130-a1b2")
		idx3 := strings.Index(listText, "2026-09-24-0010-e5f6")
		if !(idx2 < idx1 && idx1 < idx3) {
			t.Errorf("inspections not sorted newest first: idx2=%d, idx1=%d, idx3=%d", idx2, idx1, idx3)
		}
		if !strings.Contains(listText, "clusterName=prod-cluster-1, location=us-central1, projectId=my-gcp-project") {
			t.Errorf("listText missing labels: %s", listText)
		}
		wantTimeRange := fmt.Sprintf("%s - %s", mdtemplate.FormatTime(time.Unix(header1.StartTimeUnixSeconds, 0)), mdtemplate.FormatTime(time.Unix(header1.EndTimeUnixSeconds, 0)))
		if !strings.Contains(listText, wantTimeRange) {
			t.Errorf("listText missing time range %q: %s", wantTimeRange, listText)
		}
	})

	t.Run("ReadResourceDONE", func(t *testing.T) {
		readDone, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "khi://inspections/2026-09-24-0130-a1b2"})
		if err != nil {
			t.Fatalf("ReadResource for DONE inspection failed: %v", err)
		}
		if !strings.Contains(readDone.Contents[0].Text, "# Inspection Summary: prod-cluster-1 restart investigation") {
			t.Errorf("ReadResource DONE unexpected text: %s", readDone.Contents[0].Text)
		}

		// Read again to verify cached result
		readDoneCached, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "khi://inspections/2026-09-24-0130-a1b2"})
		if err != nil {
			t.Fatalf("ReadResource for cached DONE inspection failed: %v", err)
		}
		if readDoneCached.Contents[0].Text != readDone.Contents[0].Text {
			t.Errorf("cached read mismatch (-want +got):\n%s", cmp.Diff(readDone.Contents[0].Text, readDoneCached.Contents[0].Text))
		}
	})

	t.Run("ReadResourceRUNNING", func(t *testing.T) {
		readRunning, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "khi://inspections/2026-09-24-0200-c3d4"})
		if err != nil {
			t.Fatalf("ReadResource for RUNNING inspection failed: %v", err)
		}
		wantRunning := "# Inspection `2026-09-24-0200-c3d4`\n\nStatus: RUNNING. The summary is available after the inspection finishes. Call `wait_inspection` to wait for it."
		if readRunning.Contents[0].Text != wantRunning {
			t.Errorf("readRunning text = %q, want %q", readRunning.Contents[0].Text, wantRunning)
		}
	})

	t.Run("ReadResourceERROR", func(t *testing.T) {
		readError, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "khi://inspections/2026-09-24-0010-e5f6"})
		if err != nil {
			t.Fatalf("ReadResource for ERROR inspection failed: %v", err)
		}
		wantError := "# Inspection `2026-09-24-0010-e5f6`\n\nStatus: ERROR. No summary is available. Call `wait_inspection` to see which task failed."
		if readError.Contents[0].Text != wantError {
			t.Errorf("readError text = %q, want %q", readError.Contents[0].Text, wantError)
		}
	})

	t.Run("ReadResourceNotFound", func(t *testing.T) {
		_, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "khi://inspections/unknown-id"})
		if err == nil {
			t.Error("expected error for unknown inspection, got nil")
		}
	})
}
