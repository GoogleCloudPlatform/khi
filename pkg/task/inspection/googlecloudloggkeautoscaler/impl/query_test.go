// Copyright 2024 Google LLC
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

package googlecloudloggkeautoscaler_impl

import (
	"testing"

	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
)

func TestGenerateAutoscalerQuery(t *testing.T) {
	testCases := []struct {
		cluster       googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		excludeStatus bool
		expected      string
	}{
		{
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "my-project",
				Location:    "my-location",
				ClusterName: "my-cluster",
			},
			excludeStatus: false,
			expected: `resource.type="k8s_cluster"
resource.labels.project_id="my-project"
resource.labels.location="my-location"
resource.labels.cluster_name="my-cluster"
-- include query for status log
logName="projects/my-project/logs/container.googleapis.com%2Fcluster-autoscaler-visibility"`,
		},
		{
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "my-project",
				Location:    "my-location",
				ClusterName: "my-cluster",
			},
			excludeStatus: true,
			expected: `resource.type="k8s_cluster"
resource.labels.project_id="my-project"
resource.labels.location="my-location"
resource.labels.cluster_name="my-cluster"
-jsonPayload.status: ""
logName="projects/my-project/logs/container.googleapis.com%2Fcluster-autoscaler-visibility"`,
		},
	}

	for _, tc := range testCases {
		result := generateAutoscalerQuery(tc.cluster, tc.excludeStatus)
		if result != tc.expected {
			t.Errorf("Expected query:\n%s\nGot:\n%s", tc.expected, result)
		}
	}
}

func TestGeneratedAutoscalerQueryIsValid(t *testing.T) {
	testCases := []struct {
		Name          string
		Cluster       googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		ExcludeStatus bool
	}{
		{
			Name: "Valid Query",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "gcp-project-id",
				Location:    "gcp-location",
				ClusterName: "gcp-cluster-name",
			},
			ExcludeStatus: false,
		},
		{
			Name: "Valid Query with Exclude Status",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "gcp-project-id",
				Location:    "gcp-location",
				ClusterName: "gcp-cluster-name",
			},
			ExcludeStatus: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := generateAutoscalerQuery(tc.Cluster, tc.ExcludeStatus)
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf("%s", err.Error())
			}
		})
	}
}
