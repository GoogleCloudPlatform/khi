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

	form_task_test "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask/test"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

func TestNodeNameFiltertask(t *testing.T) {
	wantLabelName := "Node names"
	wantDescription := "A space-separated list of node name substrings used to collect node-related logs. If left blank, KHI gathers logs from all nodes in the cluster."
	form_task_test.TestTextForms(t, "node-name", InputNodeNameFilterTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "With an empty input",
			Input:         "",
			ExpectedValue: []string{},
			Dependencies:  []coretask.UntypedTask{},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       wantLabelName,
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
				Readonly:         false,
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "With a single node name substring",
			Input:         "node-name-1",
			ExpectedValue: []string{"node-name-1"},
			Dependencies:  []coretask.UntypedTask{},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       wantLabelName,
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "With multiple node name substrings",
			Input:         "node-name-1 node-name-2 node-name-3",
			ExpectedValue: []string{"node-name-1", "node-name-2", "node-name-3"},
			Dependencies:  []coretask.UntypedTask{},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       wantLabelName,
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "With invalid node name substring",
			Input:         "node-name-1 invalid=node=name node-name-3",
			ExpectedValue: []string{},
			Dependencies:  []coretask.UntypedTask{},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       wantLabelName,
					Description: wantDescription,
					Hint:        "substring `invalid=node=name` is not valid as a substring of node name",
					HintType:    inspectionmetadata.Error,
				},
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "With spaces around node name substring",
			Input:         "  node-name-1  node-name-2  ",
			ExpectedValue: []string{"node-name-1", "node-name-2"},
			Dependencies:  []coretask.UntypedTask{},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       wantLabelName,
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
				ValidationTiming: inspectionmetadata.Change,
			},
		},
	})
}
