// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package googlecloudlogcomputeapiaudit_impl

import (
	"testing"

	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateComputeAPIQuery(t *testing.T) {
	testCases := []struct {
		Name      string
		TaskMode  inspectioncore_contract.InspectionTaskModeType
		NodeNames []string
		Expected  []string
	}{
		{
			Name:      "DryRun mode",
			TaskMode:  inspectioncore_contract.TaskModeDryRun,
			NodeNames: []string{}, // No nodes specified for dry run
			Expected: []string{`resource.type="gce_instance"
-protoPayload.methodName:("list" OR "get" OR "watch")
-- instance name filters to be determined after audit log query
`},
		},
		{
			Name:      "Run mode with a few nodes",
			TaskMode:  inspectioncore_contract.TaskModeRun,
			NodeNames: []string{"node1", "node2"},
			Expected: []string{`resource.type="gce_instance"
-protoPayload.methodName:("list" OR "get" OR "watch")
protoPayload.resourceName:(instances/node1 OR instances/node2)
`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			act := GenerateComputeAPIQuery(tc.TaskMode, tc.NodeNames)
			if diff := cmp.Diff(tc.Expected, act); diff != "" {
				t.Errorf("The generated result is not matching with the expected\n%s", diff)
			}
		})
	}
}

func TestGenerateComputeAPIQueryIsValid(t *testing.T) {
	testCases := []struct {
		Name      string
		TaskMode  inspectioncore_contract.InspectionTaskModeType
		NodeNames []string
	}{
		{
			Name:      "Valid Query in DryRun mode",
			TaskMode:  inspectioncore_contract.TaskModeDryRun,
			NodeNames: []string{}, // No nodes specified for dry run
		},
		{
			Name:      "Valid Query in Run mode",
			TaskMode:  inspectioncore_contract.TaskModeRun,
			NodeNames: []string{"gke-test-cluster-node-1"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			queries := GenerateComputeAPIQuery(tc.TaskMode, tc.NodeNames)
			for _, query := range queries {
				err := gcp_test.IsValidLogQuery(t, query)
				if err != nil {
					t.Errorf("%s", err.Error())
				}
			}
		})
	}
}
