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

package ai

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
	auditMapperTask := newTestTask("audit-log-mapper")
	featureAuditTask := newTestTask("feature-audit", inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", "", 1, true))

	listEventTask := newTestTask("list-event-logs")
	eventMapperTask := newTestTask("event-log-mapper")
	featureEventTask := newTestTask("feature-event", inspectioncore.FeatureTaskLabel("Kubernetes Event Logs", "", 2, true))

	inventoryTask := newTestTask("inventory-node-names", inspectioncore.InventoryTaskLabel())
	listNodesTask := newTestTask("list-node-names")

	tasks := []coretask.UntypedTask{
		formTask,
		listAuditTask,
		auditMapperTask,
		featureAuditTask,
		listEventTask,
		eventMapperTask,
		featureEventTask,
		inventoryTask,
		listNodesTask,
	}

	edges := []taskid.TaskEdge{
		newTestEdge(formTask.UntypedID().String(), listAuditTask.UntypedID().String()),
		newTestEdge(formTask.UntypedID().String(), listEventTask.UntypedID().String()),
		newTestEdge(listAuditTask.UntypedID().String(), auditMapperTask.UntypedID().String()),
		newTestEdge(auditMapperTask.UntypedID().String(), featureAuditTask.UntypedID().String()),
		newTestEdge(listEventTask.UntypedID().String(), eventMapperTask.UntypedID().String()),
		newTestEdge(eventMapperTask.UntypedID().String(), featureEventTask.UntypedID().String()),
		newTestEdge(inventoryTask.UntypedID().String(), auditMapperTask.UntypedID().String()),
		newTestEdge(inventoryTask.UntypedID().String(), eventMapperTask.UntypedID().String()),
		newTestEdge(listNodesTask.UntypedID().String(), inventoryTask.UntypedID().String()),
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
			name:   "list audit logs is member of audit feature",
			taskID: listAuditTask.UntypedID().String(),
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
			name:   "inventory node names is shared",
			taskID: inventoryTask.UntypedID().String(),
			want: destination{
				kind: destinationShared,
			},
		},
		{
			name:   "list node names is shared because inventory task blocks traversal",
			taskID: listNodesTask.UntypedID().String(),
			want: destination{
				kind: destinationShared,
			},
		},
		{
			name:   "unknown task ID defaults to shared",
			taskID: "unknown-task#default",
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

func TestFormatTaskTitle(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard task ID with default hash",
			input: "audit-log-mapper#default",
			want:  "Audit Log Mapper",
		},
		{
			name:  "task ID with package path and impl hash",
			input: "khi.google.com/inspection/list-audit-logs#impl",
			want:  "List Audit Logs",
		},
		{
			name:  "task ID with underscores and spaces",
			input: "foo_bar baz",
			want:  "Foo Bar Baz",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatTaskTitle(tc.input)
			if got != tc.want {
				t.Errorf("formatTaskTitle(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
