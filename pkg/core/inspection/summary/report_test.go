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

package summary

import (
	"context"
	"sync"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

func buildTestGraph() (*coretask.TaskSet, map[string]coretask.UntypedTask) {
	formTask := newTestTask("form-cluster-name", inspectioncore.NewFormTaskLabelOpt("Cluster Name", "Form field"))
	listAuditTask := newTestTask("list-audit-logs", coretask.WithTitle("List Audit Logs"))
	auditMapperTask := newTestTask("audit-log-mapper", coretask.WithTitle("Audit Log Mapper"))
	featureAuditTask := newTestTask("feature-audit", inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", "", 1, true))

	listEventTask := newTestTask("list-event-logs")
	eventMapperTask := newTestTask("event-log-mapper")
	featureEventTask := newTestTask("feature-event", inspectioncore.FeatureTaskLabel("Kubernetes Event Logs", "", 2, true))

	inventoryTask := newTestTask("inventory-node-names")

	tasks := []coretask.UntypedTask{
		formTask,
		listAuditTask,
		auditMapperTask,
		featureAuditTask,
		listEventTask,
		eventMapperTask,
		featureEventTask,
		inventoryTask,
	}

	taskMap := make(map[string]coretask.UntypedTask)
	for _, t := range tasks {
		taskMap[t.UntypedID().String()] = t
	}

	edges := []taskid.TaskEdge{
		newTestEdge(formTask.UntypedID().String(), listAuditTask.UntypedID().String()),
		newTestEdge(formTask.UntypedID().String(), listEventTask.UntypedID().String()),
		newTestEdge(listAuditTask.UntypedID().String(), auditMapperTask.UntypedID().String()),
		newTestEdge(auditMapperTask.UntypedID().String(), featureAuditTask.UntypedID().String()),
		newTestEdge(listEventTask.UntypedID().String(), eventMapperTask.UntypedID().String()),
		newTestEdge(eventMapperTask.UntypedID().String(), featureEventTask.UntypedID().String()),
	}

	return coretask.NewResolvedTaskSet(tasks, edges, nil), taskMap
}

func buildTaskContext(collector *Collector, task coretask.UntypedTask) context.Context {
	metaMap := typedmap.NewTypedMap()
	typedmap.Set(metaMap, MetadataKey, collector)
	ctx := khictx.WithValue(context.Background(), inspectionmetadata.MapContextKey, metaMap.AsReadonly())
	if task != nil {
		ctx = khictx.WithValue(ctx, core_contract.TaskImplementationIDContextKey, task.UntypedID())
	}
	return ctx
}

func TestReportAPI(t *testing.T) {
	taskGraph, tasks := buildTestGraph()

	testCases := []struct {
		name string
		run  func(c *Collector)
		want Snapshot
	}{
		{
			name: "SetCoreLabel records core labels",
			run: func(c *Collector) {
				ctx := buildTaskContext(c, tasks["form-cluster-name#default"])
				SetCoreLabel(ctx, "cluster", "production-cluster")
				SetCoreLabel(ctx, "environment", "gcp")
			},
			want: Snapshot{
				CoreLabels: []KeyValue{
					{Key: "cluster", Value: "production-cluster"},
					{Key: "environment", Value: "gcp"},
				},
			},
		},
		{
			name: "SetProperty from form task writes to common properties",
			run: func(c *Collector) {
				ctx := buildTaskContext(c, tasks["form-cluster-name#default"])
				SetProperty(ctx, "cluster_id", "c-12345")
			},
			want: Snapshot{
				CommonProperties: []KeyValue{
					{Key: "cluster_id", Value: "c-12345"},
				},
			},
		},
		{
			name: "SetProperty from feature task writes to feature section properties",
			run: func(c *Collector) {
				ctx := buildTaskContext(c, tasks["feature-audit#default"])
				SetProperty(ctx, "status", "enabled")
			},
			want: Snapshot{
				Sections: []Section{
					{
						Title: "Kubernetes Audit Logs",
						Properties: []KeyValue{
							{Key: "status", Value: "enabled"},
						},
					},
				},
			},
		},
		{
			name: "SetProperty from featureMember writes to task report",
			run: func(c *Collector) {
				ctx := buildTaskContext(c, tasks["audit-log-mapper#default"])
				SetProperty(ctx, "mapped_count", "42")
			},
			want: Snapshot{
				Sections: []Section{
					{
						Title: "Kubernetes Audit Logs",
						TaskReports: []TaskReport{
							{
								Title: "Audit Log Mapper",
								Properties: []KeyValue{
									{Key: "mapped_count", Value: "42"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "SetProperty from shared/inventory writes to shared section properties",
			run: func(c *Collector) {
				ctx := buildTaskContext(c, tasks["inventory-node-names#default"])
				SetProperty(ctx, "nodeCount", "10")
			},
			want: Snapshot{
				Sections: []Section{
					{
						Title: "Shared & Filter Context",
						Properties: []KeyValue{
							{Key: "nodeCount", Value: "10"},
						},
					},
				},
			},
		},
		{
			name: "AddFeatureIntProperty writes to enclosing feature section properties",
			run: func(c *Collector) {
				ctx := buildTaskContext(c, tasks["audit-log-mapper#default"])
				AddFeatureIntProperty(ctx, "totalLogs", 15)
				AddFeatureIntProperty(ctx, "totalLogs", 25)
			},
			want: Snapshot{
				Sections: []Section{
					{
						Title: "Kubernetes Audit Logs",
						Properties: []KeyValue{
							{Key: "totalLogs", Value: "40"},
						},
					},
				},
			},
		},
		{
			name: "AddFeatureIntProperty from form writes to common, from shared writes to shared",
			run: func(c *Collector) {
				ctxForm := buildTaskContext(c, tasks["form-cluster-name#default"])
				AddFeatureIntProperty(ctxForm, "formDelta", 5)

				ctxShared := buildTaskContext(c, tasks["inventory-node-names#default"])
				AddFeatureIntProperty(ctxShared, "sharedDelta", 8)
			},
			want: Snapshot{
				CommonProperties: []KeyValue{
					{Key: "formDelta", Value: "5"},
				},
				Sections: []Section{
					{
						Title: "Shared & Filter Context",
						Properties: []KeyValue{
							{Key: "sharedDelta", Value: "8"},
						},
					},
				},
			},
		},
		{
			name: "AppendMarkdown routing",
			run: func(c *Collector) {
				// Feature task writes to section insights.
				ctxFeat := buildTaskContext(c, tasks["feature-audit#default"])
				AppendMarkdown(ctxFeat, "Feature insight 1")
				AppendMarkdown(ctxFeat, "Feature insight 2")

				// Member task writes to task report markdown.
				ctxMember := buildTaskContext(c, tasks["audit-log-mapper#default"])
				AppendMarkdown(ctxMember, "Task report line 1")
				AppendMarkdown(ctxMember, "Task report line 2")

				// Shared task writes to shared section insights.
				ctxShared := buildTaskContext(c, tasks["inventory-node-names#default"])
				AppendMarkdown(ctxShared, "Shared insight")
			},
			want: Snapshot{
				Sections: []Section{
					{
						Title: "Kubernetes Audit Logs",
						TaskReports: []TaskReport{
							{
								Title:    "Audit Log Mapper",
								Markdown: "Task report line 1\n\nTask report line 2",
							},
						},
						Insights: "Feature insight 1\n\nFeature insight 2",
					},
					{
						Title:    "Shared & Filter Context",
						Insights: "Shared insight",
					},
				},
			},
		},
		{
			name: "RecordQuery routing",
			run: func(c *Collector) {
				ctxFeat := buildTaskContext(c, tasks["feature-audit#default"])
				RecordQuery(ctxFeat, "q2", "Query Two", "text two")

				ctxMember := buildTaskContext(c, tasks["audit-log-mapper#default"])
				RecordQuery(ctxMember, "q1", "Query One", "text one")

				ctxShared := buildTaskContext(c, tasks["inventory-node-names#default"])
				RecordQuery(ctxShared, "qShared", "Shared Query", "text shared")
			},
			want: Snapshot{
				Sections: []Section{
					{
						Title: "Kubernetes Audit Logs",
						Queries: []Query{
							{Name: "Query One", Text: "text one"},
							{Name: "Query Two", Text: "text two"},
						},
					},
					{
						Title: "Shared & Filter Context",
						Queries: []Query{
							{Name: "Shared Query", Text: "text shared"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCollector(taskGraph)
			tc.run(c)
			got := c.Snapshot()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Snapshot() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReportAPI_PanicsOnInvalidContext(t *testing.T) {
	taskGraph, _ := buildTestGraph()

	testCases := []struct {
		name string
		ctx  func() context.Context
	}{
		{
			name: "missing metadata map in context",
			ctx:  context.Background,
		},
		{
			name: "missing collector in metadata map",
			ctx: func() context.Context {
				metaMap := typedmap.NewTypedMap().AsReadonly()
				return khictx.WithValue(context.Background(), inspectionmetadata.MapContextKey, metaMap)
			},
		},
		{
			name: "missing task implementation ID in context",
			ctx: func() context.Context {
				collector := NewCollector(taskGraph)
				metaMap := typedmap.NewTypedMap()
				typedmap.Set(metaMap, MetadataKey, collector)
				return khictx.WithValue(context.Background(), inspectionmetadata.MapContextKey, metaMap.AsReadonly())
			},
		},
		{
			name: "unknown task implementation ID not in task graph",
			ctx: func() context.Context {
				collector := NewCollector(taskGraph)
				metaMap := typedmap.NewTypedMap()
				typedmap.Set(metaMap, MetadataKey, collector)
				ctx := khictx.WithValue(context.Background(), inspectionmetadata.MapContextKey, metaMap.AsReadonly())
				unknownID := taskid.NewDefaultImplementationID[any]("unknown-task")
				return khictx.WithValue(ctx, core_contract.TaskImplementationIDContextKey, unknownID.(taskid.UntypedTaskImplementationID))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic, got nil")
				}
			}()
			ctx := tc.ctx()
			SetProperty(ctx, "key", "val")
		})
	}
}

func TestReportAPI_ConcurrentAddIntProperty(t *testing.T) {
	taskGraph, tasks := buildTestGraph()
	c := NewCollector(taskGraph)
	taskCtx := buildTaskContext(c, tasks["audit-log-mapper#default"])

	var wg sync.WaitGroup
	const goroutines = 100
	const callsPerGoroutine = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				AddIntProperty(taskCtx, "counter", 1)
			}
		}()
	}
	wg.Wait()

	snapshot := c.Snapshot()
	if len(snapshot.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(snapshot.Sections))
	}
	if len(snapshot.Sections[0].TaskReports) != 1 {
		t.Fatalf("expected 1 task report, got %d", len(snapshot.Sections[0].TaskReports))
	}
	props := snapshot.Sections[0].TaskReports[0].Properties
	if len(props) != 1 {
		t.Fatalf("expected 1 property, got %d", len(props))
	}
	gotVal := props[0].Value
	wantVal := "10000"
	if gotVal != wantVal {
		t.Errorf("counter property = %q, want %q", gotVal, wantVal)
	}
}

func TestReportAPI_AddToSetProperty(t *testing.T) {
	taskGraph, tasks := buildTestGraph()
	c := NewCollector(taskGraph)
	taskCtx := buildTaskContext(c, tasks["audit-log-mapper#default"])

	testCases := []struct {
		name   string
		writes []string
		want   string
	}{
		{
			name:   "deduplicates and sorts set elements",
			writes: []string{"banana", "apple", "banana", "cherry", "apple"},
			want:   "apple, banana, cherry",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, val := range tc.writes {
				AddToSetProperty(taskCtx, "fruits", val)
			}
			snapshot := c.Snapshot()
			props := snapshot.Sections[0].TaskReports[0].Properties
			if len(props) != 1 {
				t.Fatalf("expected 1 property, got %d", len(props))
			}
			got := props[0].Value
			if got != tc.want {
				t.Errorf("AddToSetProperty() = %q, want %q", got, tc.want)
			}
		})
	}
}
