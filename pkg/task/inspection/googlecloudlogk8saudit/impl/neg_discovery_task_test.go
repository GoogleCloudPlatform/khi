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

package googlecloudlogk8saudit_impl

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func TestAuditLogNEGDiscoveryTask(t *testing.T) {
	createNode := func(t *testing.T, message string) *structured.NodeReader {
		node, err := structured.FromGoValue(map[string]any{
			"status": map[string]any{
				"conditions": []any{
					map[string]any{
						"message": message,
					},
				},
			},
		}, &structured.AlphabeticalGoMapKeyOrderProvider{})
		if err != nil {
			t.Fatalf("failed to create node: %v", err)
		}
		return structured.NewNodeReader(node)
	}

	tests := []struct {
		name     string
		groupMap commonlogk8sauditv2_contract.ResourceManifestLogGroupMap
		taskMode inspectioncore_contract.InspectionTaskModeType
		want     googlecloudk8scommon_contract.NEGToBackendServiceMap
	}{
		{
			name: "valid audit log",
			groupMap: commonlogk8sauditv2_contract.ResourceManifestLogGroupMap{
				"group1": {
					Resource: &commonlogk8sauditv2_contract.ResourceIdentity{
						Kind: "Pod",
					},
					Logs: []*commonlogk8sauditv2_contract.ResourceManifestLog{
						{
							ResourceBodyReader: createNode(t, `Pod has become Healthy in NEG "Key{\"k8s1-audit\", zone: \"asia-northeast1-b\"}" attached to BackendService "Key{\"bs-audit\"}". Marking condition "cloud.google.com/load-balancer-neg-ready" to True.`),
						},
					},
				},
			},
			taskMode: inspectioncore_contract.TaskModeRun,
			want: googlecloudk8scommon_contract.NEGToBackendServiceMap{
				"k8s1-audit": "bs-audit",
			},
		},
		{
			name: "not a pod",
			groupMap: commonlogk8sauditv2_contract.ResourceManifestLogGroupMap{
				"group1": {
					Resource: &commonlogk8sauditv2_contract.ResourceIdentity{
						Kind: "Deployment",
					},
					Logs: []*commonlogk8sauditv2_contract.ResourceManifestLog{
						{
							ResourceBodyReader: createNode(t, `Pod has become Healthy in NEG "Key{\"k8s1-audit\", zone: \"asia-northeast1-b\"}" attached to BackendService "Key{\"bs-audit\"}". Marking condition "cloud.google.com/load-balancer-neg-ready" to True.`),
						},
					},
				},
			},
			taskMode: inspectioncore_contract.TaskModeRun,
			want:     googlecloudk8scommon_contract.NEGToBackendServiceMap{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			result, _, err := inspectiontest.RunInspectionTask(ctx, AuditLogNEGDiscoveryTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref(), tc.groupMap),
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
