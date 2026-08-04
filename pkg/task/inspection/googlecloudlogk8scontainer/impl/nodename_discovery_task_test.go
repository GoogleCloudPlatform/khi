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

package googlecloudlogk8scontainer_impl

import (
	"sort"
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestNodeNameDiscoveryTask(t *testing.T) {
	tests := []struct {
		name     string
		logs     []*log.Log
		taskMode inspectioncore_contract.InspectionTaskModeType
		want     []string
	}{
		{
			name: "valid container logs with node name labels",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
					NodeName: "gke-test-cluster-default-pool-node-1",
				}),
				log.NewLogWithFieldSetsForTest(&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
					NodeName: "gke-test-cluster-default-pool-node-2",
				}),
				log.NewLogWithFieldSetsForTest(&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
					NodeName: "gke-test-cluster-default-pool-node-1",
				}),
			},
			taskMode: inspectioncore_contract.TaskModeRun,
			want: []string{
				"gke-test-cluster-default-pool-node-1",
				"gke-test-cluster-default-pool-node-2",
			},
		},
		{
			name: "empty node name label is ignored",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
					NodeName: "",
				}),
				log.NewLogWithFieldSetsForTest(&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
					NodeName: "gke-test-cluster-default-pool-node-1",
				}),
			},
			taskMode: inspectioncore_contract.TaskModeRun,
			want: []string{
				"gke-test-cluster-default-pool-node-1",
			},
		},
		{
			name: "dry run returns nil",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
					NodeName: "gke-test-cluster-default-pool-node-1",
				}),
			},
			taskMode: inspectioncore_contract.TaskModeDryRun,
			want:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			result, _, err := inspectiontest.RunInspectionTask(ctx, NodeNameDiscoveryTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(googlecloudlogk8scontainer_contract.FieldSetReaderTaskID.Ref(), tc.logs),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sort.Strings(result)
			sort.Strings(tc.want)
			if diff := cmp.Diff(tc.want, result); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
