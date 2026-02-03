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

package googlecloudlogk8scontrolplane_impl

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
)

func TestGenerateK8sControlPlaneQuery(t *testing.T) {
	testCases := []struct {
		ExpectedQuery                        string
		Cluster                              googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		InputControlplaneComponentNameFilter *gcpqueryutil.SetFilterParseResult
	}{
		{
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			InputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true},
			ExpectedQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.
-- No component name filter`,
		},
		{
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			InputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{"apiserver", "autoscaler"}},
			ExpectedQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.
-resource.labels.component_name:("apiserver" OR "autoscaler")`,
		},
		{
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			InputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: false, Additives: []string{"apiserver"}},
			ExpectedQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.
resource.labels.component_name:("apiserver")`,
		},
		{
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			InputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: false, Additives: []string{}},
			ExpectedQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.
-- Invalid: none of the controlplane component will be selected. Ignoreing component name filter.`,
		},
		{
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			InputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{ValidationError: "test error"},
			ExpectedQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.
-- Failed to generate component name filter due to the validation error "test error"`,
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%d-%s", i, testCase.ExpectedQuery), func(t *testing.T) {
			result := GenerateK8sControlPlaneQuery(testCase.Cluster, testCase.InputControlplaneComponentNameFilter)
			if result != testCase.ExpectedQuery {
				t.Errorf("the result query is not valid:\nInput:\n%v\nActual:\n%s\nExpected:\n%s", testCase, result, testCase.ExpectedQuery)
			}
			t.Run("generated query must be valid in Cloud Logging", func(t *testing.T) {
				err := gcp_test.IsValidLogQuery(t, result)
				if err != nil {
					t.Errorf("%s", err.Error())
				}
			})
		})
	}
}
