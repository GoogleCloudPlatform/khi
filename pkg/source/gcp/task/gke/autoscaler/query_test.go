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

package autoscaler

import (
	"testing"

	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
)

func TestGenerateAutoscalerQuery(t *testing.T) {
	testCases := []struct {
		projectId     string
		clusterName   string
		excludeStatus bool
		expected      string
	}{
		{
			projectId:     "my-project",
			clusterName:   "my-cluster",
			excludeStatus: false,
			expected: `resource.type="k8s_cluster"
resource.labels.project_id="my-project"
resource.labels.cluster_name="my-cluster"
-- include query for status log
logName="projects/my-project/logs/container.googleapis.com%2Fcluster-autoscaler-visibility"`,
		},
		{
			projectId:     "my-project",
			clusterName:   "my-cluster",
			excludeStatus: true,
			expected: `resource.type="k8s_cluster"
resource.labels.project_id="my-project"
resource.labels.cluster_name="my-cluster"
-jsonPayload.status: ""
logName="projects/my-project/logs/container.googleapis.com%2Fcluster-autoscaler-visibility"`,
		},
	}

	for _, tc := range testCases {
		result := GenerateAutoscalerQuery(tc.projectId, tc.clusterName, tc.excludeStatus)
		if result != tc.expected {
			t.Errorf("Expected query:\n%s\nGot:\n%s", tc.expected, result)
		}
	}
}

func TestGeneratedAutoscalerQueryIsValid(t *testing.T) {
	testCases := []struct {
		Name          string
		ProjectId     string
		ClusterName   string
		ExcludeStatus bool
	}{
		{
			Name:          "Valid Query",
			ProjectId:     "gcp-project-id",
			ClusterName:   "gcp-cluster-name",
			ExcludeStatus: false,
		},
		{
			Name:          "Valid Query with Exclude Status",
			ProjectId:     "gcp-project-id",
			ClusterName:   "gcp-cluster-name",
			ExcludeStatus: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := GenerateAutoscalerQuery(tc.ProjectId, tc.ClusterName, tc.ExcludeStatus)
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf("%s", err.Error())
			}
		})
	}
}
