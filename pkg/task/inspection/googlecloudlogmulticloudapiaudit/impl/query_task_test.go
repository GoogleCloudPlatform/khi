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

package googlecloudlogmulticloudapiaudit_impl

import (
	"testing"

	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateMultiCloudAPIQuery(t *testing.T) {
	testCases := []struct {
		name    string
		cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		want    string
	}{
		{
			name: "standard input",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:         "test-project",
				ClusterName:       "test-cluster",
				ClusterTypePrefix: "awsClusters/",
				Location:          "asia-northeast1",
			},
			want: `
log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access")
resource.type="audited_resource"
resource.labels.service="gkemulticloud.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete")
protoPayload.resourceName:"projects/test-project/locations/asia-northeast1/"
protoPayload.resourceName:"awsClusters/test-cluster"
`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := generateQuery(testCase.cluster)
			if diff := cmp.Diff(testCase.want, actual); diff != "" {
				t.Errorf("The generated result is not matching with the expected\n%s", diff)
			}
		})
	}
}

func TestGenerateMultiCloudAPIQueryIsValid(t *testing.T) {
	testCases := []struct {
		name    string
		cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity
	}{
		{
			name: "Valid Query",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:         "test-project",
				ClusterName:       "test-cluster",
				ClusterTypePrefix: "awsClusters/",
				Location:          "asia-northeast1",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := generateQuery(tc.cluster)
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf("%s", err.Error())
			}
		})
	}
}
