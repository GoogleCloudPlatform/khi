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

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
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

func TestSectionResolver(t *testing.T) {
	formTask := newTestTask("form-cluster-name", inspectioncore.NewFormTaskLabelOpt("Cluster Name", "Form field"))
	listAuditTask := newTestTask("list-audit-logs")
	auditExtractorTask := newTestTask("audit-log-extractor")
	auditMapperTask := newTestTask("audit-log-mapper")
	featureAuditTask := newTestTask("feature-audit", inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", "", 2, true))

	listEventTask := newTestTask("list-event-logs")
	eventMapperTask := newTestTask("event-log-mapper")
	featureEventTask := newTestTask("feature-event", inspectioncore.FeatureTaskLabel("Kubernetes Event Logs", "", 1, true))

	nodeDiscoveryTask := newTestTask("discovery-node-names", coretask.WithFeatureGate(featureAuditTask.UntypedID().GetUntypedReference()))
	inventoryTask := newTestTask("inventory-node-names")
	serializerTask := newTestTask("serializer")

	tasks := []coretask.UntypedTask{
		formTask,
		listAuditTask,
		auditExtractorTask,
		auditMapperTask,
		featureAuditTask,
		listEventTask,
		eventMapperTask,
		featureEventTask,
		nodeDiscoveryTask,
		inventoryTask,
		serializerTask,
	}

	edges := []taskid.TaskEdge{
		newTestEdge(formTask.UntypedID().String(), listAuditTask.UntypedID().String()),
		newTestEdge(formTask.UntypedID().String(), listEventTask.UntypedID().String()),
		newTestEdge(listAuditTask.UntypedID().String(), auditExtractorTask.UntypedID().String()),
		newTestEdge(auditExtractorTask.UntypedID().String(), auditMapperTask.UntypedID().String()),
		newTestEdge(auditMapperTask.UntypedID().String(), featureAuditTask.UntypedID().String()),
		newTestEdge(listEventTask.UntypedID().String(), eventMapperTask.UntypedID().String()),
		newTestEdge(eventMapperTask.UntypedID().String(), featureEventTask.UntypedID().String()),
		newTestEdge(listAuditTask.UntypedID().String(), nodeDiscoveryTask.UntypedID().String()),
		newTestEdge(nodeDiscoveryTask.UntypedID().String(), inventoryTask.UntypedID().String()),
		newTestEdge(inventoryTask.UntypedID().String(), featureEventTask.UntypedID().String()),
		newTestEdge(featureAuditTask.UntypedID().String(), serializerTask.UntypedID().String()),
	}

	taskGraph := coretask.NewResolvedTaskSet(tasks, edges, nil)
	resolver := newSectionResolver(taskGraph)

	testCases := []struct {
		name   string
		taskID string
		want   destination
	}{
		{
			name:   "form task",
			taskID: formTask.UntypedID().String(),
			want: destination{
				kind: destinationForm,
			},
		},
		{
			name:   "list audit logs is member of audit feature and does not traverse gated discovery task",
			taskID: listAuditTask.UntypedID().String(),
			want: destination{
				kind:      destinationFeatureMember,
				featureID: featureAuditTask.UntypedID().String(),
			},
		},
		{
			name:   "audit log extractor is member of audit feature",
			taskID: auditExtractorTask.UntypedID().String(),
			want: destination{
				kind:      destinationFeatureMember,
				featureID: featureAuditTask.UntypedID().String(),
			},
		},
		{
			name:   "audit log mapper is member of audit feature",
			taskID: auditMapperTask.UntypedID().String(),
			want: destination{
				kind:      destinationFeatureMember,
				featureID: featureAuditTask.UntypedID().String(),
			},
		},
		{
			name:   "node discovery task is member of audit feature via feature gate",
			taskID: nodeDiscoveryTask.UntypedID().String(),
			want: destination{
				kind:      destinationFeatureMember,
				featureID: featureAuditTask.UntypedID().String(),
			},
		},
		{
			name:   "feature audit is feature itself",
			taskID: featureAuditTask.UntypedID().String(),
			want: destination{
				kind:      destinationFeature,
				featureID: featureAuditTask.UntypedID().String(),
			},
		},
		{
			name:   "list event logs is member of event feature",
			taskID: listEventTask.UntypedID().String(),
			want: destination{
				kind:      destinationFeatureMember,
				featureID: featureEventTask.UntypedID().String(),
			},
		},
		{
			name:   "event log mapper is member of event feature",
			taskID: eventMapperTask.UntypedID().String(),
			want: destination{
				kind:      destinationFeatureMember,
				featureID: featureEventTask.UntypedID().String(),
			},
		},
		{
			name:   "feature event is feature itself",
			taskID: featureEventTask.UntypedID().String(),
			want: destination{
				kind:      destinationFeature,
				featureID: featureEventTask.UntypedID().String(),
			},
		},
		{
			name:   "serializer task downstream of features is shared",
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
				t.Errorf("destinationOf() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSectionResolverTieBreak(t *testing.T) {
	feature1 := newTestTask("feature-1", inspectioncore.FeatureTaskLabel("Feature One", "", 10, false))
	feature2 := newTestTask("feature-2", inspectioncore.FeatureTaskLabel("Feature Two", "", 5, false))
	taskSharedChild := newTestTask("task-split")

	tasks := []coretask.UntypedTask{feature1, feature2, taskSharedChild}
	edges := []taskid.TaskEdge{
		newTestEdge(taskSharedChild.UntypedID().String(), feature1.UntypedID().String()),
		newTestEdge(taskSharedChild.UntypedID().String(), feature2.UntypedID().String()),
	}

	taskGraph := coretask.NewResolvedTaskSet(tasks, edges, nil)
	resolver := newSectionResolver(taskGraph)

	testCases := []struct {
		name string
		want destination
	}{
		{
			name: "picks feature with lower order at equal distance",
			want: destination{
				kind:      destinationFeatureMember,
				featureID: feature2.UntypedID().String(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolver.destinationOf(taskSharedChild.UntypedID().String())
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(destination{})); diff != "" {
				t.Errorf("destinationOf() tie-break mismatch (-want +got):\n%s", diff)
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
