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
	"time"

	form_task_test "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask/test"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	inspectioncore_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/impl"
)

func TestInputEndtime(t *testing.T) {
	expectedDescription := "The endtime of query. Please input it in the format of RFC3339\n(example: 2006-01-02T15:04:05-07:00)"
	expectedLabel := "End time"
	expectedValue1, err := time.Parse(time.RFC3339, "2020-01-02T03:04:05Z")
	if err != nil {
		t.Errorf("unexpected error\n%s", err)
	}
	expectedValue2, err := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	timezoneTaskUTC := tasktest.StubTask(inspectioncore_impl.TimeZoneShiftInputTask, time.UTC, nil)
	timezoneTaskJST := tasktest.StubTask(inspectioncore_impl.TimeZoneShiftInputTask, time.FixedZone("", 9*3600), nil)

	if err != nil {
		t.Errorf("unexpected error\n%s", err)
	}
	form_task_test.TestTextForms(t, "endtime", InputEndTimeTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "with empty",
			Input:         "",
			ExpectedValue: expectedValue1,
			Dependencies:  []coretask.UntypedTask{inspectioncore_impl.TestInspectionTimeTaskProducer("2020-01-02T03:04:05Z"), timezoneTaskUTC},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					Hint:        "invalid time format. Please specify in the format of `2006-01-02T15:04:05-07:00`(RFC3339)",
					HintType:    inspectionmetadata.Error,
				},
				Default:     "2020-01-02T03:04:05Z",
				Suggestions: []string{},
			},
		},
		{
			Name:          "with valid timestamp and UTC timezone",
			Input:         "2020-01-02T00:00:00Z",
			ExpectedValue: expectedValue2,
			Dependencies:  []coretask.UntypedTask{inspectioncore_impl.TestInspectionTimeTaskProducer("2020-01-02T03:04:05Z"), timezoneTaskUTC},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					HintType:    inspectionmetadata.None,
				},
				Suggestions: []string{},
				Default:     "2020-01-02T03:04:05Z",
			},
		},
		{
			Name:          "with valid timestamp and non UTC timezone",
			Input:         "2020-01-02T00:00:00Z",
			ExpectedValue: expectedValue2,
			Dependencies:  []coretask.UntypedTask{inspectioncore_impl.TestInspectionTimeTaskProducer("2020-01-02T03:04:05Z"), timezoneTaskJST},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					HintType:    inspectionmetadata.None,
				},
				Suggestions: []string{},
				Default:     "2020-01-02T12:04:05+09:00",
			},
		},
	})
}
