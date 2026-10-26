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
	"net/http"
	"net/http/httptest"
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
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestTask(id string, labelOpts ...coretask.LabelOpt) coretask.UntypedTask {
	return coretask.DefineConstant[any](
		taskid.NewDefaultImplementationID[any](id),
		nil,
		labelOpts...,
	)
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
		{SourceImplID: formTask.UntypedID().String(), TargetImplID: listAuditTask.UntypedID().String()},
		{SourceImplID: listAuditTask.UntypedID().String(), TargetImplID: auditMapperTask.UntypedID().String()},
		{SourceImplID: auditMapperTask.UntypedID().String(), TargetImplID: featureAuditTask.UntypedID().String()},
	}

	taskGraph := coretask.NewResolvedTaskSet(tasks, edges, nil)
	c := summary.NewCollector(taskGraph)

	ctxForm := taskContext(c, formTask.UntypedID())
	summary.SetCoreLabel(ctxForm, "clusterName", "prod-cluster-1")
	summary.SetCoreLabel(ctxForm, "projectId", "my-gcp-project")
	summary.SetProperty(ctxForm, "duration", "1h")

	ctxListAudit := taskContext(c, listAuditTask.UntypedID())
	summary.RecordQuery(ctxListAudit, "k8s-audit", "K8s audit logs", "resource.type=\"k8s_cluster\"\nresource.labels.cluster_name=\"prod-cluster-1\"")
	summary.RecordQueryLogCount(ctxListAudit, "k8s-audit", 1520)
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

	vocab := inspectionmetadata.NewFilterVocabularyMetadata()
	vocab.Set(inspectionmetadata.FilterVocabulary{
		TimelineTypes: []inspectionmetadata.TimelineTypeVocabulary{
			{
				Label:        "apiVersion",
				NameCount:    1,
				ExampleNames: []string{"core/v1"},
			},
			{
				Label:        "kind",
				NameCount:    6,
				ExampleNames: []string{"node", "pod", "service", "replicaset", "deployment"},
			},
			{
				Label:        "namespace",
				NameCount:    2,
				ExampleNames: []string{"kube-system", "default"},
			},
		},
		LogTypes: []string{"k8s audit", "k8s event"},
	})

	md := typedmap.NewTypedMap()
	typedmap.Set(md, inspectionmetadata.HeaderMetadataKey, header)
	typedmap.Set(md, summary.MetadataKey, c)
	typedmap.Set(md, inspectionmetadata.FilterVocabularyMetadataKey, vocab)

	got, err := NewInspectionHandler(nil, nil).renderSummary("2026-09-24-0130-a1b2", md.AsReadonly())
	if err != nil {
		t.Fatalf("renderSummary() error: %v", err)
	}

	want := readGolden(t, "testdata/inspection_summary.golden.md")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("inspection summary mismatch (-want +got):\n%s", diff)
	}

	t.Run("WithoutFilterVocabulary", func(t *testing.T) {
		mdWithoutVocab := typedmap.NewTypedMap()
		typedmap.Set(mdWithoutVocab, inspectionmetadata.HeaderMetadataKey, header)
		typedmap.Set(mdWithoutVocab, summary.MetadataKey, c)
		gotWithoutVocab, err := NewInspectionHandler(nil, nil).renderSummary("2026-09-24-0130-a1b2", mdWithoutVocab.AsReadonly())
		if err != nil {
			t.Fatalf("renderSummary() without filter vocabulary error: %v", err)
		}
		wantNotice := "This inspection file does not record a filter vocabulary in its metadata. Call `search_timelines` or `search_logs` to see the available `path` keys, `timelineType` values, and `logType` values in the response."
		if !strings.Contains(gotWithoutVocab, wantNotice) {
			t.Errorf("expected summary without filter vocabulary to contain %q, got:\n%s", wantNotice, gotWithoutVocab)
		}
	})
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
	}); err != nil {
		t.Fatalf("failed to add oss-kubernetes-from-files inspection type: %v", err)
	}

	if err := server.AddTask(newTestTask("k8s-audit-log", inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", "", 1, true))); err != nil {
		t.Fatalf("failed to add k8s-audit-log task: %v", err)
	}
	if err := server.AddTask(newTestTask("k8s-event-log", inspectioncore.FeatureTaskLabel("Kubernetes Event Logs", "", 2, true))); err != nil {
		t.Fatalf("failed to add k8s-event-log task: %v", err)
	}

	// This test does not call request_file_upload.
	handler := NewInspectionHandler(server, nil)
	mcpServer := NewServer(handler)

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpServer.HTTPHandler())

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	session := connectClient(t, ctx, &mcpsdk.StreamableClientTransport{Endpoint: ts.URL + "/mcp"})

	t.Run("ListInspectionTypesResource", func(t *testing.T) {
		got := strings.TrimRight(readResourceText(t, ctx, session, "khi://inspection-types"), "\r\n")
		if diff := cmp.Diff(readGolden(t, "testdata/inspection_types.golden.md"), got); diff != "" {
			t.Errorf("inspection types resource mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("EmptyInspections", func(t *testing.T) {
		got := readResourceText(t, ctx, session, "khi://inspections")
		if diff := cmp.Diff("# Inspections\n\nNo inspections yet.", got); diff != "" {
			t.Errorf("empty inspections list mismatch (-want +got):\n%s", diff)
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

	vocab1 := inspectionmetadata.NewFilterVocabularyMetadata()
	vocab1.Set(inspectionmetadata.FilterVocabulary{
		TimelineTypes: []inspectionmetadata.TimelineTypeVocabulary{
			{Label: "kind", NameCount: 2, ExampleNames: []string{"node", "pod"}},
		},
		LogTypes: []string{"k8s audit"},
	})
	typedmap.Set(md1, inspectionmetadata.FilterVocabularyMetadataKey, vocab1)

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
		got := readResourceText(t, ctx, session, "khi://inspections")
		want := "# Inspections\n\n" +
			"3 inspections, newest first. Read `khi://inspections/{id}` for the summary of each inspection.\n\n" +
			"| ID | Name | Type | Status | Time range | Labels |\n" +
			"| --- | --- | --- | --- | --- | --- |\n" +
			"| `2026-09-24-0200-c3d4` | staging-cluster investigation | Google Kubernetes Engine | RUNNING |  |  |\n" +
			"| `2026-09-24-0130-a1b2` | prod-cluster-1 restart investigation | Google Kubernetes Engine | DONE | 2026-09-24T01:00:00Z - 2026-09-24T02:00:00Z | clusterName=prod-cluster-1, location=us-central1, projectId=my-gcp-project |\n" +
			"| `2026-09-24-0010-e5f6` | failed investigation | Google Kubernetes Engine | ERROR |  |  |"
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ListInspections mismatch (-want +got):\n%s", diff)
		}
	})

	wantDoneSummary := "# Inspection Summary: prod-cluster-1 restart investigation\n\n" +
		"- **ID**: `2026-09-24-0130-a1b2`\n" +
		"- **Type**: Google Kubernetes Engine\n" +
		"- **Time Range**: 2026-09-24T01:00:00Z - 2026-09-24T02:00:00Z\n\n" +
		"## Core Labels\n\n" +
		"| Key | Value |\n" +
		"| --- | --- |\n" +
		"| clusterName | prod-cluster-1 |\n" +
		"| location | us-central1 |\n" +
		"| projectId | my-gcp-project |\n\n" +
		"---\n\n" +
		"## Filter vocabulary\n" +
		"These are the values that filter CEL can match in this inspection.\n" +
		"- Timeline names by timelineType, matched with `path[\"<key>\"]`. Each line shows up to 5 example names and the number of distinct names.\n" +
		"  - `kind` → `path[\"kind\"]`: `node`, `pod` (all 2 names)\n" +
		"- logType: `k8s audit`\n" +
		"- severity: `UNKNOWN`, `INFO`, `WARNING`, `ERROR`, `FATAL` (reassigned by KHI per logType for investigation importance; does not match Cloud Logging severity — query `body(\"severity\", \"ERROR\")` for the raw Cloud Logging field)"

	readCases := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		{
			name: "ReadResourceDONE",
			uri:  "khi://inspections/2026-09-24-0130-a1b2",
			want: wantDoneSummary,
		},
		{
			name: "ReadResourceRUNNING",
			uri:  "khi://inspections/2026-09-24-0200-c3d4",
			want: "# Inspection `2026-09-24-0200-c3d4`\n\nStatus: RUNNING. The summary is available after the inspection finishes. Call `wait_inspection` to wait for it.",
		},
		{
			name: "ReadResourceERROR",
			uri:  "khi://inspections/2026-09-24-0010-e5f6",
			want: "# Inspection `2026-09-24-0010-e5f6`\n\nStatus: ERROR. No summary is available. Call `wait_inspection` to see which task failed.",
		},
		{
			name:    "ReadResourceNotFound",
			uri:     "khi://inspections/unknown-id",
			wantErr: true,
		},
	}

	for _, tc := range readCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: tc.uri})
			if (err != nil) != tc.wantErr {
				t.Fatalf("ReadResource(%q) error = %v, wantErr %v", tc.uri, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if len(res.Contents) != 1 {
				t.Fatalf("len(res.Contents) = %d, want 1", len(res.Contents))
			}
			if diff := cmp.Diff(tc.want, res.Contents[0].Text); diff != "" {
				t.Errorf("ReadResource(%q) mismatch (-want +got):\n%s", tc.uri, diff)
			}
		})
	}

	t.Run("ReadResourceDONECached", func(t *testing.T) {
		summary.SetCoreLabel(ctxTask1, "clusterName", "mutated-after-cache")
		got := readResourceText(t, ctx, session, "khi://inspections/2026-09-24-0130-a1b2")
		if diff := cmp.Diff(wantDoneSummary, got); diff != "" {
			t.Errorf("cached ReadResource mismatch (-want +got):\n%s", diff)
		}
	})
}
