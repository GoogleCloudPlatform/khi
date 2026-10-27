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

package k8sevent_impl

import (
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8sevent"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TestNodeNameDiscoveryTask tests nodeNameDiscoveryTask.
func TestNodeNameDiscoveryTask(t *testing.T) {
	tests := []struct {
		name     string
		logs     []*log.Log
		taskMode inspectioncore.InspectionTaskModeType
		want     []string
	}{
		{
			name: "valid event logs with source host",
			logs: []*log.Log{
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					SourceHost: "gke-test-cluster-default-pool-node-1",
				}),
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					SourceHost: "gke-test-cluster-default-pool-node-2",
				}),
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					SourceHost: "gke-test-cluster-default-pool-node-1",
				}),
			},
			taskMode: inspectioncore.TaskModeRun,
			want: []string{
				"gke-test-cluster-default-pool-node-1",
				"gke-test-cluster-default-pool-node-2",
			},
		},
		{
			name: "empty source host is ignored",
			logs: []*log.Log{
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					SourceHost: "",
				}),
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					SourceHost: "gke-test-cluster-default-pool-node-1",
				}),
			},
			taskMode: inspectioncore.TaskModeRun,
			want: []string{
				"gke-test-cluster-default-pool-node-1",
			},
		},
		{
			name: "dry run returns nil",
			logs: []*log.Log{
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					SourceHost: "gke-test-cluster-default-pool-node-1",
				}),
			},
			taskMode: inspectioncore.TaskModeDryRun,
			want:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			result, _, err := inspectiontest.Run(t, ctx, nodeNameDiscoveryTask, tc.taskMode, map[string]any{},
				tasktest.Given(k8sevent.ListLogEntriesTaskID.Ref(), tc.logs),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, result, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
