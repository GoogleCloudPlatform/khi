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
	"testing"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

func newTestTask(id string, labelOpts ...coretask.LabelOpt) coretask.Task[any] {
	return coretask.DefineConstant[any](
		taskid.NewDefaultImplementationID[any](id),
		nil,
		labelOpts...,
	)
}

func newTestTaskWithBind(id string, bind func(b *coretask.Binder), labelOpts ...coretask.LabelOpt) coretask.Task[any] {
	return coretask.Define(
		taskid.NewDefaultImplementationID[any](id),
		func(b *coretask.Binder) func(ctx context.Context) (any, error) {
			if bind != nil {
				bind(b)
			}
			return func(ctx context.Context) (any, error) {
				return nil, nil
			}
		},
		labelOpts...,
	)
}

func newTestEdge(sourceID, targetID string) taskid.TaskEdge {
	return taskid.TaskEdge{
		SourceImplID: sourceID,
		TargetImplID: targetID,
		Cardinality:  taskid.CardinalityPointToPoint,
		Scope:        taskid.ScopeAll,
	}
}

func TestSectionResolver_ExampleDAGFromIssue1039(t *testing.T) {
	tagNodeNames := coretask.NewTag[any]("test-node-names")
	tagEmpty := coretask.NewTag[any]("test-empty-inventory")

	formTask := newTestTask("form-cluster-name", inspectioncore.NewFormTaskLabelOpt("Cluster Name", "Form field"))
	listNodeTask := newTestTask("list-node-names", coretask.ProvidesTag(tagNodeNames))
	inventoryTask := inspectiontaskbase.DefineInventoryTask(
		taskid.NewDefaultImplementationID[any]("inventory-node-names"),
		tagNodeNames,
		func(results []any) (any, error) { return results, nil },
	)
	emptyInventoryTask := inspectiontaskbase.DefineInventoryTask(
		taskid.NewDefaultImplementationID[any]("inventory-empty"),
		tagEmpty,
		func(results []any) (any, error) { return results, nil },
	)

	listAuditTask := newTestTaskWithBind("list-audit-logs", func(b *coretask.Binder) {
		coretask.Use(b, formTask.ID().Ref())
	})
	auditMapperTask := newTestTaskWithBind("audit-log-mapper", func(b *coretask.Binder) {
		coretask.Use(b, listAuditTask.ID().Ref())
		coretask.Use(b, inventoryTask.ID().Ref())
		coretask.Use(b, emptyInventoryTask.ID().Ref())
	})
	featureAuditTask := newTestTaskWithBind("feature-audit", func(b *coretask.Binder) {
		coretask.Use(b, auditMapperTask.ID().Ref())
	}, inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", "", 1, true))

	listEventTask := newTestTaskWithBind("list-event-logs", func(b *coretask.Binder) {
		coretask.Use(b, formTask.ID().Ref())
	})
	eventMapperTask := newTestTaskWithBind("event-log-mapper", func(b *coretask.Binder) {
		coretask.Use(b, listEventTask.ID().Ref())
		coretask.Use(b, inventoryTask.ID().Ref())
	})
	featureEventTask := newTestTaskWithBind("feature-event", func(b *coretask.Binder) {
		coretask.Use(b, eventMapperTask.ID().Ref())
	}, inspectioncore.FeatureTaskLabel("Kubernetes Event Logs", "", 2, true))

	available := []coretask.UntypedTask{
		formTask,
		listNodeTask,
		inventoryTask,
		emptyInventoryTask,
		listAuditTask,
		auditMapperTask,
		featureAuditTask,
		listEventTask,
		eventMapperTask,
		featureEventTask,
	}
	taskGraph, err := coretask.ResolveGraph([]coretask.UntypedTask{featureAuditTask, featureEventTask}, available, nil)
	if err != nil {
		t.Fatalf("ResolveGraph() unexpected error: %v", err)
	}

	resolver := newSectionResolver(taskGraph)

	testCases := []struct {
		name   string
		taskID string
		want   destination
	}{
		{
			name:   "form task routes to Common Properties",
			taskID: formTask.UntypedID().String(),
			want: destination{
				kind: destinationForm,
			},
		},
		{
			name:   "list audit logs routes to Kubernetes Audit Logs",
			taskID: listAuditTask.UntypedID().String(),
			want: destination{
				kind:       destinationFeatureMember,
				featureIDs: []string{featureAuditTask.UntypedID().String()},
			},
		},
		{
			name:   "audit log mapper routes to Kubernetes Audit Logs",
			taskID: auditMapperTask.UntypedID().String(),
			want: destination{
				kind:       destinationFeatureMember,
				featureIDs: []string{featureAuditTask.UntypedID().String()},
			},
		},
		{
			name:   "feature audit routes to its own feature section",
			taskID: featureAuditTask.UntypedID().String(),
			want: destination{
				kind:       destinationFeature,
				featureIDs: []string{featureAuditTask.UntypedID().String()},
			},
		},
		{
			name:   "list event logs routes to Kubernetes Event Logs",
			taskID: listEventTask.UntypedID().String(),
			want: destination{
				kind:       destinationFeatureMember,
				featureIDs: []string{featureEventTask.UntypedID().String()},
			},
		},
		{
			name:   "event log mapper routes to Kubernetes Event Logs",
			taskID: eventMapperTask.UntypedID().String(),
			want: destination{
				kind:       destinationFeatureMember,
				featureIDs: []string{featureEventTask.UntypedID().String()},
			},
		},
		{
			name:   "feature event routes to its own feature section",
			taskID: featureEventTask.UntypedID().String(),
			want: destination{
				kind:       destinationFeature,
				featureIDs: []string{featureEventTask.UntypedID().String()},
			},
		},
		{
			name:   "fan-in inventory task consumed by mappers routes to Shared & Filter Context",
			taskID: inventoryTask.UntypedID().String(),
			want: destination{
				kind: destinationShared,
			},
		},
		{
			name:   "fan-in inventory task with zero active producers still routes to Shared & Filter Context",
			taskID: emptyInventoryTask.UntypedID().String(),
			want: destination{
				kind: destinationShared,
			},
		},
		{
			name:   "ungated discovery task feeding inventory routes to Shared & Filter Context",
			taskID: listNodeTask.UntypedID().String(),
			want: destination{
				kind: destinationShared,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolver.destinationOf(tc.taskID)
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(destination{})); diff != "" {
				t.Errorf("destinationOf(%q) mismatch (-want +got):\n%s", tc.taskID, diff)
			}
		})
	}
}

func TestSectionResolver_BoundariesAndMultiFeatureFanOut(t *testing.T) {
	tagNodeNames := coretask.NewTag[any]("test-gated-node-names")

	formHelperTask := newTestTask("form-autocomplete-helper")
	formTask := newTestTaskWithBind("form-cluster-name", func(b *coretask.Binder) {
		coretask.Use(b, formHelperTask.ID().Ref())
	}, inspectioncore.NewFormTaskLabelOpt("Cluster Name", "Form field"))

	inventoryTask := inspectiontaskbase.DefineInventoryTask(
		taskid.NewDefaultImplementationID[any]("inventory-node-names"),
		tagNodeNames,
		func(results []any) (any, error) { return results, nil },
	)

	// Shared downstream consumer of inventory, required by both features at unequal hop distances (2 hops to audit, 1 hop to event).
	sharedPatternFinderTask := newTestTaskWithBind("resource-uid-pattern-finder", func(b *coretask.Binder) {
		coretask.Use(b, inventoryTask.ID().Ref())
	})

	listAuditTask := newTestTaskWithBind("list-audit-logs", func(b *coretask.Binder) {
		coretask.Use(b, formTask.ID().Ref())
	})
	auditExtractorTask := newTestTaskWithBind("audit-log-extractor", func(b *coretask.Binder) {
		coretask.Use(b, listAuditTask.ID().Ref())
	})
	auditMapperTask := newTestTaskWithBind("audit-log-mapper", func(b *coretask.Binder) {
		coretask.Use(b, auditExtractorTask.ID().Ref())
		coretask.Use(b, sharedPatternFinderTask.ID().Ref())
	})
	// Task required by feature-audit, but only optionally referenced by event-log-mapper.
	auditOnlyProducerTask := newTestTask("audit-manifest-table")
	featureAuditConsumerTask := newTestTaskWithBind("audit-manifest-consumer", func(b *coretask.Binder) {
		coretask.Use(b, auditOnlyProducerTask.ID().Ref())
	})
	featureAuditTask := newTestTaskWithBind("feature-audit", func(b *coretask.Binder) {
		coretask.Use(b, auditMapperTask.ID().Ref())
		coretask.Use(b, featureAuditConsumerTask.ID().Ref())
	}, inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", "", 10, true))

	gatedDiscoveryTask := newTestTaskWithBind("discovery-node-names", func(b *coretask.Binder) {
		coretask.Use(b, listAuditTask.ID().Ref())
	}, coretask.ProvidesTag(tagNodeNames), coretask.WithFeatureGate(featureAuditTask.ID().Ref()))

	listEventTask := newTestTaskWithBind("list-event-logs", func(b *coretask.Binder) {
		coretask.Use(b, formTask.ID().Ref())
	})
	eventMapperTask := newTestTaskWithBind("event-log-mapper", func(b *coretask.Binder) {
		coretask.Use(b, listEventTask.ID().Ref())
		coretask.UseOptional(b, auditOnlyProducerTask.ID().Ref(coretask.FromActiveGraph))
	})
	featureEventTask := newTestTaskWithBind("feature-event", func(b *coretask.Binder) {
		coretask.Use(b, eventMapperTask.ID().Ref())
		coretask.Use(b, sharedPatternFinderTask.ID().Ref())
	}, inspectioncore.FeatureTaskLabel("Kubernetes Event Logs", "", 1, true))

	serializerTask := newTestTaskWithBind("serializer", func(b *coretask.Binder) {
		coretask.UseOptional(b, featureAuditTask.ID().Ref(coretask.FromActiveGraph))
		coretask.UseOptional(b, featureEventTask.ID().Ref(coretask.FromActiveGraph))
	}, coretask.NewRequiredTaskLabel())

	// Also test a helper upstream of an ungated discovery task: stops at fan-in boundary -> Shared.
	ungatedDiscoveryHelperTask := newTestTask("ungated-discovery-helper")
	ungatedDiscoveryTask := newTestTaskWithBind("ungated-discovery", func(b *coretask.Binder) {
		coretask.Use(b, ungatedDiscoveryHelperTask.ID().Ref())
	}, coretask.ProvidesTag(tagNodeNames))

	available := []coretask.UntypedTask{
		formHelperTask,
		formTask,
		inventoryTask,
		sharedPatternFinderTask,
		listAuditTask,
		auditExtractorTask,
		auditMapperTask,
		auditOnlyProducerTask,
		featureAuditConsumerTask,
		featureAuditTask,
		gatedDiscoveryTask,
		listEventTask,
		eventMapperTask,
		featureEventTask,
		serializerTask,
		ungatedDiscoveryHelperTask,
		ungatedDiscoveryTask,
	}
	taskGraph, err := coretask.ResolveGraph([]coretask.UntypedTask{featureAuditTask, featureEventTask}, available, nil)
	if err != nil {
		t.Fatalf("ResolveGraph() unexpected error: %v", err)
	}

	resolver := newSectionResolver(taskGraph)

	testCases := []struct {
		name   string
		taskID string
		want   destination
	}{
		{
			name:   "helper upstream of form task stops at form boundary and routes to shared",
			taskID: formHelperTask.UntypedID().String(),
			want: destination{
				kind: destinationShared,
			},
		},
		{
			name:   "feature-gated discovery task belongs to its gated feature",
			taskID: gatedDiscoveryTask.UntypedID().String(),
			want: destination{
				kind:       destinationFeatureMember,
				featureIDs: []string{featureAuditTask.UntypedID().String()},
			},
		},
		{
			name:   "list audit logs stays exclusively in audit feature without leaking through gated discovery",
			taskID: listAuditTask.UntypedID().String(),
			want: destination{
				kind:       destinationFeatureMember,
				featureIDs: []string{featureAuditTask.UntypedID().String()},
			},
		},
		{
			name:   "UseOptional does not pull audit producer into event feature",
			taskID: auditOnlyProducerTask.UntypedID().String(),
			want: destination{
				kind:       destinationFeatureMember,
				featureIDs: []string{featureAuditTask.UntypedID().String()},
			},
		},
		{
			name:   "task required by multiple features at unequal hop distances fans out to all reachable features",
			taskID: sharedPatternFinderTask.UntypedID().String(),
			want: destination{
				kind: destinationFeatureMember,
				featureIDs: []string{
					featureAuditTask.UntypedID().String(),
					featureEventTask.UntypedID().String(),
				},
			},
		},
		{
			name:   "helper upstream of ungated discovery routes to shared",
			taskID: ungatedDiscoveryHelperTask.UntypedID().String(),
			want: destination{
				kind: destinationShared,
			},
		},
		{
			name:   "serializer downstream of features routes to shared",
			taskID: serializerTask.UntypedID().String(),
			want: destination{
				kind: destinationShared,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolver.destinationOf(tc.taskID)
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(destination{})); diff != "" {
				t.Errorf("destinationOf(%q) mismatch (-want +got):\n%s", tc.taskID, diff)
			}
		})
	}
}

func TestSectionResolver_PanicsOnUnknownTaskID(t *testing.T) {
	taskGraph := coretask.NewResolvedTaskSet(nil, nil, nil)
	resolver := newSectionResolver(taskGraph)

	testCases := []struct {
		name string
		call func()
	}{
		{
			name: "destinationOf panics on unknown task ID",
			call: func() {
				resolver.destinationOf("unknown-task#default")
			},
		},
		{
			name: "titleOf panics on unknown task ID",
			call: func() {
				resolver.titleOf("unknown-task#default")
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
			tc.call()
		})
	}
}

func TestSectionResolver_TitleOf(t *testing.T) {
	taskWithTitle := newTestTask("task-with-title", coretask.WithTitle("Custom Title"))
	featureTask := newTestTask("feature-task", inspectioncore.FeatureTaskLabel("Feature Title", "desc", 1, true))
	taskWithoutTitle := newTestTask("task-without-title")
	taskWithEmptyTitle := newTestTask("task-with-empty-title", coretask.WithTitle(""))

	taskGraph := coretask.NewResolvedTaskSet([]coretask.UntypedTask{
		taskWithTitle,
		featureTask,
		taskWithoutTitle,
		taskWithEmptyTitle,
	}, nil, nil)
	resolver := newSectionResolver(taskGraph)

	testCases := []struct {
		name   string
		taskID string
		want   string
	}{
		{
			name:   "explicit title label",
			taskID: taskWithTitle.UntypedID().String(),
			want:   "Custom Title",
		},
		{
			name:   "feature task title",
			taskID: featureTask.UntypedID().String(),
			want:   "Feature Title",
		},
		{
			name:   "fallback to task ID when no title is set",
			taskID: taskWithoutTitle.UntypedID().String(),
			want:   taskWithoutTitle.UntypedID().String(),
		},
		{
			name:   "explicit empty title falls back to task ID",
			taskID: taskWithEmptyTitle.UntypedID().String(),
			want:   taskWithEmptyTitle.UntypedID().String(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolver.titleOf(tc.taskID)
			if got != tc.want {
				t.Errorf("titleOf(%q) = %q, want %q", tc.taskID, got, tc.want)
			}
		})
	}
}
