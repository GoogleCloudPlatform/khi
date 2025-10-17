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
	"context"
	"testing"

	form_task_test "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask/test"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

func TestLocationInput(t *testing.T) {
	mockAutocompleteLocationsTask := coretask.NewTask(googlecloudcommon_contract.AutocompleteLocationTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) ([]string, error) {
		return []string{"asia-northeast1", "us-central1"}, nil
	})
	form_task_test.TestTextForms(t, "gcp-location", InputLocationsTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "With valid location",
			Input:         "asia-northeast1",
			ExpectedValue: "asia-northeast1",
			Dependencies:  []coretask.UntypedTask{mockAutocompleteLocationsTask, InputProjectIdTask},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudcommon_contract.GoogleCloudCommonTaskIDPrefix + "input-location",
					Type:        "Text",
					Label:       "Location",
					Description: "The location(region) to specify the resource exist(s|ed)",
					HintType:    inspectionmetadata.None,
				},
				Suggestions: []string{
					"asia-northeast1", "us-central1",
				},
				Readonly:         false,
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "Location suggestion is sorted by the distance from the input",
			Input:         "us",
			ExpectedValue: "us",
			Dependencies:  []coretask.UntypedTask{mockAutocompleteLocationsTask, InputProjectIdTask},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          googlecloudcommon_contract.GoogleCloudCommonTaskIDPrefix + "input-location",
					Type:        "Text",
					Label:       "Location",
					Description: "The location(region) to specify the resource exist(s|ed)",
					HintType:    inspectionmetadata.None,
				},
				Suggestions: []string{
					"us-central1", "asia-northeast1",
				},
				Readonly:         false,
				ValidationTiming: inspectionmetadata.Change,
			},
		},
	})
}
