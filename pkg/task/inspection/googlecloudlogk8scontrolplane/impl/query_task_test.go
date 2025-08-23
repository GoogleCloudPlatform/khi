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
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
)

func TestGenerateK8sControlPlaneQuery(t *testing.T) {
	testCases := []struct {
		ExpectedQuery                        string
		InputClusterName                     string
		InputProjectName                     string
		InputControlplaneComponentNameFilter *gcpqueryutil.SetFilterParseResult
	}{
		{
			InputClusterName:                     "foo-cluster",
			InputProjectName:                     "foo-project",
			InputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true},
			ExpectedQuery: `resource.type="k8s_control_plane_component"
resource.labels.cluster_name="foo-cluster"
resource.labels.project_id="foo-project"
-sourceLocation.file="httplog.go"
-- No component name filter`,
		},
		{
			InputClusterName:                     "foo-cluster",
			InputProjectName:                     "foo-project",
			InputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{"apiserver", "autoscaler"}},
			ExpectedQuery: `resource.type="k8s_control_plane_component"
resource.labels.cluster_name="foo-cluster"
resource.labels.project_id="foo-project"
-sourceLocation.file="httplog.go"
-resource.labels.component_name:("apiserver" OR "autoscaler")`,
		},
		{
			InputClusterName:                     "foo-cluster",
			InputProjectName:                     "foo-project",
			InputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: false, Additives: []string{"apiserver"}},
			ExpectedQuery: `resource.type="k8s_control_plane_component"
resource.labels.cluster_name="foo-cluster"
resource.labels.project_id="foo-project"
-sourceLocation.file="httplog.go"
resource.labels.component_name:("apiserver")`,
		},
		{
			InputClusterName:                     "foo-cluster",
			InputProjectName:                     "foo-project",
			InputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: false, Additives: []string{}},
			ExpectedQuery: `resource.type="k8s_control_plane_component"
resource.labels.cluster_name="foo-cluster"
resource.labels.project_id="foo-project"
-sourceLocation.file="httplog.go"
-- Invalid: none of the controlplane component will be selected. Ignoreing component name filter.`,
		},
		{
			InputClusterName:                     "foo-cluster",
			InputProjectName:                     "foo-project",
			InputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{ValidationError: "test error"},
			ExpectedQuery: `resource.type="k8s_control_plane_component"
resource.labels.cluster_name="foo-cluster"
resource.labels.project_id="foo-project"
-sourceLocation.file="httplog.go"
-- Failed to generate component name filter due to the validation error "test error"`,
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%d-%s", i, testCase.ExpectedQuery), func(t *testing.T) {
			result := GenerateK8sControlPlaneQuery(testCase.InputClusterName, testCase.InputProjectName, testCase.InputControlplaneComponentNameFilter)
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
