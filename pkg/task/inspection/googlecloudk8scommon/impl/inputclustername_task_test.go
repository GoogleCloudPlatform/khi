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

package googlecloudk8scommon_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	form_task_test "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask/test"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func TestClusterNameInput(t *testing.T) {
	wantDescription := "The cluster name to gather logs."
	testClusterNamePrefix := tasktest.StubTaskFromReferenceID(googlecloudk8scommon_contract.ClusterNamePrefixTaskRef, "", nil)
	mockClusterNamesTask1 := tasktest.StubTaskFromReferenceID(googlecloudk8scommon_contract.AutocompleteClusterIdentityTaskID.Ref(), &inspectioncore_contract.AutocompleteResult[googlecloudk8scommon_contract.GoogleCloudClusterIdentity]{
		Values: []googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
			{
				ClusterName: "foo-cluster",
			},
			{
				ClusterName: "bar-cluster",
			},
		},
		Error: "",
	}, nil)
	form_task_test.TestTextForms(t, "cluster name", InputClusterNameTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "with valid cluster name",
			Input:         "foo-cluster",
			ExpectedValue: "foo-cluster",
			Dependencies:  []coretask.UntypedTask{mockClusterNamesTask1, testClusterNamePrefix},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudk8scommon_contract.GoogleCloudCommonK8STaskIDPrefix + "input-cluster-name",
					Type:        "Text",
					Label:       "Cluster name",
					HintType:    inspectionmetadata.None,
					Description: wantDescription,
				},
				Suggestions:      []string{"foo-cluster", "bar-cluster"},
				Default:          "foo-cluster",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "spaces around cluster name must be trimmed",
			Input:         "  foo-cluster   ",
			ExpectedValue: "foo-cluster",
			Dependencies:  []coretask.UntypedTask{mockClusterNamesTask1, testClusterNamePrefix},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudk8scommon_contract.GoogleCloudCommonK8STaskIDPrefix + "input-cluster-name",
					Type:        "Text",
					Label:       "Cluster name",
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
				Suggestions:      []string{"foo-cluster", "bar-cluster"},
				Default:          "foo-cluster",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "invalid cluster name",
			Input:         "An invalid cluster name",
			ExpectedValue: "foo-cluster",
			Dependencies:  []coretask.UntypedTask{mockClusterNamesTask1, testClusterNamePrefix},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudk8scommon_contract.GoogleCloudCommonK8STaskIDPrefix + "input-cluster-name",
					Type:        "Text",
					Label:       "Cluster name",
					Description: wantDescription,
					HintType:    inspectionmetadata.Error,
					Hint:        "Cluster name must match `^[0-9a-z:\\-]+$`",
				},
				Suggestions:      common.SortForAutocomplete("An invalid cluster name", []string{"foo-cluster", "bar-cluster"}),
				Default:          "foo-cluster",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "non existing cluster should show a hint",
			Input:         "nonexisting-cluster",
			ExpectedValue: "nonexisting-cluster",
			Dependencies:  []coretask.UntypedTask{mockClusterNamesTask1, testClusterNamePrefix},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudk8scommon_contract.GoogleCloudCommonK8STaskIDPrefix + "input-cluster-name",
					Type:        "Text",
					Label:       "Cluster name",
					Description: wantDescription,
					Hint: `Cluster 'nonexisting-cluster' was not found in the specified project at this time. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.
Available cluster names:
* foo-cluster
* bar-cluster
`,
					HintType: inspectionmetadata.Warning,
				},
				Suggestions:      []string{"foo-cluster", "bar-cluster"},
				Default:          "foo-cluster",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
	})
}
