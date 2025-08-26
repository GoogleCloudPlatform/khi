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

package googlecloudcommon_impl

import (
	"testing"

	form_task_test "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask/test"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

func TestProjectIdInput(t *testing.T) {
	wantDescription := "The project ID containing logs of the cluster to query"
	form_task_test.TestTextForms(t, "gcp-project-id", InputProjectIdTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "With valid project ID",
			Input:         "foo-project",
			ExpectedValue: "foo-project",

			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudcommon_contract.GoogleCloudCommonTaskIDPrefix + "input/project-id",
					Type:        inspectionmetadata.Text,
					Label:       "Project ID",
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
			},
		},
		{
			Name:          "With fixed project ID from environment variable",
			Input:         "foo-project",
			ExpectedValue: "bar-project",

			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudcommon_contract.GoogleCloudCommonTaskIDPrefix + "input/project-id",
					Type:        inspectionmetadata.Text,
					Label:       "Project ID",
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
				Readonly: true,
				Default:  "bar-project",
			},
			Before: func() {
				expectedFixedProjectId := "bar-project"
				parameters.Auth.FixedProjectID = &expectedFixedProjectId
			},
			After: func() {
				parameters.Auth.FixedProjectID = nil
			},
		},
		{
			Name:          "With invalid project ID",
			Input:         "A invalid project ID",
			ExpectedValue: "",

			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudcommon_contract.GoogleCloudCommonTaskIDPrefix + "input/project-id",
					Type:        inspectionmetadata.Text,
					Label:       "Project ID",
					Description: wantDescription,
					HintType:    inspectionmetadata.Error,
					Hint:        "Project ID must match `^*[0-9a-z\\.:\\-]+$`",
				},
			},
		},
		{
			Name:          "Spaces around project ID must be trimmed",
			Input:         "  project-foo   ",
			ExpectedValue: "project-foo",

			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudcommon_contract.GoogleCloudCommonTaskIDPrefix + "input/project-id",
					Type:        inspectionmetadata.Text,
					Label:       "Project ID",
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
			},
		},
		{
			Name:          "With valid old style project ID",
			Input:         "  deprecated.com:but-still-usable-project-id   ",
			ExpectedValue: "deprecated.com:but-still-usable-project-id",

			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudcommon_contract.GoogleCloudCommonTaskIDPrefix + "input/project-id",
					Description: wantDescription,
					Type:        "Text",
					Label:       "Project ID",
					HintType:    inspectionmetadata.None,
				},
			},
		},
	})
}
