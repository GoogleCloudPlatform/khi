// Copyright 2025 Google LLC
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

package googlecloudlogk8sevent_impl

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8sevent_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func TestEventLogNEGDiscoveryTask(t *testing.T) {
	tests := []struct {
		name     string
		logs     []*log.Log
		taskMode inspectioncore_contract.InspectionTaskModeType
		want     googlecloudk8scommon_contract.NEGToBackendServiceMap
	}{
		{
			name: "valid event log",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: time.Now()}, &googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
					Message: `Pod has become Healthy in NEG "Key{\"k8s1-event\", zone: \"asia-northeast1-b\"}" attached to BackendService "Key{\"bs-event\"}". Marking condition "cloud.google.com/load-balancer-neg-ready" to True.`,
				}),
				log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: time.Now()}), // no fieldset
			},
			taskMode: inspectioncore_contract.TaskModeRun,
			want: googlecloudk8scommon_contract.NEGToBackendServiceMap{
				"k8s1-event": "bs-event",
			},
		},
		{
			name: "dry run mode",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: time.Now()}, &googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
					Message: `Pod has become Healthy in NEG "Key{\"k8s1-event\", zone: \"asia-northeast1-b\"}" attached to BackendService "Key{\"bs-event\"}". Marking condition "cloud.google.com/load-balancer-neg-ready" to True.`,
				}),
			},
			taskMode: inspectioncore_contract.TaskModeDryRun,
			want:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			result, _, err := inspectiontest.RunInspectionTask(ctx, EventLogNEGDiscoveryTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(googlecloudlogk8sevent_contract.FieldSetReaderTaskID.Ref(), tc.logs),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, result); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
