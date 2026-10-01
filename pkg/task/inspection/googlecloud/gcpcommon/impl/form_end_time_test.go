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

package gcpcommon_impl

import (
	"testing"
	"time"

	form_task_test "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask/test"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func TestInputEndtime(t *testing.T) {
	expectedDescription := "The endtime of query. Please input it in the format of RFC3339\n(example: 2006-01-02T15:04:05-07:00)"
	expectedLabel := "End time"
	expectedValue1, err := time.Parse(time.RFC3339, "2025-01-01T01:01:01Z")
	if err != nil {
		t.Errorf("unexpected error\n%s", err)
	}
	expectedValue2, err := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	if err != nil {
		t.Errorf("unexpected error\n%s", err)
	}

	form_task_test.TestTextForms(t, "endtime", inputEndTimeTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "with empty",
			Input:         "",
			ExpectedValue: expectedValue1,
			TaskInputs: []tasktest.InputValue{
				tasktest.Given(inspectioncore.TimeZoneShiftInputTaskID.Ref(), time.UTC),
			},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					Hint:        "invalid time format. Please specify in the format of `2006-01-02T15:04:05-07:00`(RFC3339)",
					HintType:    inspectionmetadata.Error,
				},
				Default:          "2025-01-01T01:01:01Z",
				Suggestions:      []string{},
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "with valid timestamp and UTC timezone",
			Input:         "2020-01-02T00:00:00Z",
			ExpectedValue: expectedValue2,
			TaskInputs: []tasktest.InputValue{
				tasktest.Given(inspectioncore.TimeZoneShiftInputTaskID.Ref(), time.UTC),
			},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					HintType:    inspectionmetadata.None,
				},
				Suggestions:      []string{},
				Default:          "2025-01-01T01:01:01Z",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "with valid timestamp and non UTC timezone",
			Input:         "2020-01-02T00:00:00Z",
			ExpectedValue: expectedValue2,
			TaskInputs: []tasktest.InputValue{
				tasktest.Given(inspectioncore.TimeZoneShiftInputTaskID.Ref(), time.FixedZone("", 9*3600)),
			},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					HintType:    inspectionmetadata.None,
				},
				Suggestions:      []string{},
				Default:          "2025-01-01T10:01:01+09:00",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
	})
}
