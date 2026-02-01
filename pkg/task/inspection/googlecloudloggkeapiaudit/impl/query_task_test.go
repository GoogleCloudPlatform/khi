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

package googlecloudloggkeapiaudit_impl

import (
	"testing"

	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateGKEAuditQuery(t *testing.T) {
	testCases := []struct {
		clusterIdentity googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		expected        string
	}{
		{
			clusterIdentity: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				ClusterName: "test-cluster",
				Location:    "asia-northeast1",
			},
			expected: `log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access")
resource.type=("gke_cluster" OR "gke_nodepool")
resource.labels.project_id="test-project"
resource.labels.location="asia-northeast1"
resource.labels.cluster_name="test-cluster"
protoPayload.serviceName="container.googleapis.com"
`,
		},
	}

	for _, tc := range testCases {
		result := GenerateGKEAuditQuery(tc.clusterIdentity)
		if diff := cmp.Diff(result, tc.expected); diff != "" {
			t.Errorf("GenerateGKEAuditQuery() mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestGeneratedGKEAuditQueryIsValid(t *testing.T) {
	testCases := []struct {
		name            string
		clusterIdentity googlecloudk8scommon_contract.GoogleCloudClusterIdentity
	}{
		{
			name: "Valid Query",
			clusterIdentity: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				ClusterName: "test-cluster",
				Location:    "asia-northeast1",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := GenerateGKEAuditQuery(tc.clusterIdentity)
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf("%s", err.Error())
			}
		})
	}
}
