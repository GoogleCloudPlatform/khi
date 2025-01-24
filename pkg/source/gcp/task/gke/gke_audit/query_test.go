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

package gke_audit

import (
	"testing"

	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
)

func TestGenerateGKEAuditQuery(t *testing.T) {
	testCases := []struct {
		projectName string
		clusterName string
		expected    string
	}{
		{
			projectName: "my-project",
			clusterName: "my-cluster",
			expected: `resource.type=("gke_cluster" OR "gke_nodepool")
logName="projects/my-project/logs/cloudaudit.googleapis.com%2Factivity"
resource.labels.cluster_name="my-cluster"`,
		},
	}

	for _, tc := range testCases {
		result := GenerateGKEAuditQuery(tc.projectName, tc.clusterName)
		if result != tc.expected {
			t.Errorf("Expected query:\n%s\nGot:\n%s", tc.expected, result)
		}
	}
}

func TestGeneratedGKEAuditQueryIsValid(t *testing.T) {
	testCases := []struct {
		Name        string
		ProjectId   string
		ClusterName string
	}{
		{
			Name:        "Valid Query",
			ProjectId:   "gcp-project-id",
			ClusterName: "gcp-cluster-name",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := GenerateGKEAuditQuery(tc.ProjectId, tc.ClusterName)
			err := gcp_test.IsValidLogQuery(query)
			if err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}
