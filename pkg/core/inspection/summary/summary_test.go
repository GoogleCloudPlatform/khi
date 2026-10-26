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
	"testing"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

func TestCollector_SnapshotOrdering(t *testing.T) {
	featC := newTestTask("feature-c", inspectioncore.FeatureTaskLabel("Feature C", "", 1, false))
	featA := newTestTask("feature-a", inspectioncore.FeatureTaskLabel("Feature A", "", 1, false))
	featB := newTestTask("feature-b", inspectioncore.FeatureTaskLabel("Feature B", "", 2, false))
	featEmpty := newTestTask("feature-empty", inspectioncore.FeatureTaskLabel("Feature Empty", "", 0, false))

	memberA2 := newTestTask("task-a-2", coretask.WithTitle("Task A 2"))
	memberA1 := newTestTask("task-a-1", coretask.WithTitle("Task A 1"))

	tasks := []coretask.UntypedTask{
		featC, featA, featB, featEmpty,
		memberA2, memberA1,
	}

	edges := []taskid.TaskEdge{
		newTestEdge(memberA2.UntypedID().String(), featA.UntypedID().String()),
		newTestEdge(memberA1.UntypedID().String(), featA.UntypedID().String()),
	}

	taskGraph := coretask.NewResolvedTaskSet(tasks, edges, nil)

	testCases := []struct {
		name  string
		setup func(c *Collector)
		want  Snapshot
	}{
		{
			name: "orders features by order then ID, task reports by ID, rows by key, and places shared last while omitting empty sections",
			setup: func(c *Collector) {
				c.coreLabels["cluster"] = "my-cluster"
				c.coreLabels["project"] = "my-project"

				c.common.set("zone", "us-central1-a")
				c.common.set("region", "us-central1")

				// Write to feature B (order 2).
				secB := c.getOrCreateSection(featB.UntypedID().String())
				secB.properties.set("metric", "100")

				// Write to feature C (order 1).
				secC := c.getOrCreateSection(featC.UntypedID().String())
				secC.properties.set("metric", "50")

				// Write to feature A (order 1) via members.
				secA := c.getOrCreateSection(featA.UntypedID().String())
				secA.properties.set("status", "ok")
				tr2 := secA.getOrCreateTaskReport(memberA2.UntypedID().String())
				tr2.properties.set("count", "20")
				tr1 := secA.getOrCreateTaskReport(memberA1.UntypedID().String())
				tr1.properties.set("count", "10")

				// Write to shared.
				secShared := c.getOrCreateShared()
				secShared.properties.set("nodeCount", "3")
			},
			want: Snapshot{
				CoreLabels: []KeyValue{
					{Key: "cluster", Value: "my-cluster"},
					{Key: "project", Value: "my-project"},
				},
				CommonProperties: []KeyValue{
					{Key: "region", Value: "us-central1"},
					{Key: "zone", Value: "us-central1-a"},
				},
				Sections: []Section{
					{
						Title: "Feature A",
						Properties: []KeyValue{
							{Key: "status", Value: "ok"},
						},
						TaskReports: []TaskReport{
							{
								Title: "Task A 1",
								Properties: []KeyValue{
									{Key: "count", Value: "10"},
								},
							},
							{
								Title: "Task A 2",
								Properties: []KeyValue{
									{Key: "count", Value: "20"},
								},
							},
						},
					},
					{
						Title: "Feature C",
						Properties: []KeyValue{
							{Key: "metric", Value: "50"},
						},
					},
					{
						Title: "Feature B",
						Properties: []KeyValue{
							{Key: "metric", Value: "100"},
						},
					},
					{
						Title: "Shared & Filter Context",
						Properties: []KeyValue{
							{Key: "nodeCount", Value: "3"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCollector(taskGraph)
			tc.setup(c)
			got := c.Snapshot()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Snapshot() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
