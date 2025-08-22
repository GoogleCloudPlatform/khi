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

func TestDurationInput(t *testing.T) {
	expectedDescription := "The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)"
	expectedLabel := "Duration"
	expectedSuggestions := []string{"1m", "10m", "1h", "3h", "12h", "24h"}
	timezoneTaskUTC := tasktest.StubTask(inspectioncore_impl.TimeZoneShiftInputTask, time.UTC, nil)
	timezoneTaskJST := tasktest.StubTask(inspectioncore_impl.TimeZoneShiftInputTask, time.FixedZone("", 9*3600), nil)
	currentTimeTask1 := tasktest.StubTask(inspectioncore_impl.InspectionTimeProducer, time.Date(2023, time.April, 5, 12, 0, 0, 0, time.UTC), nil)
	endTimeTask := tasktest.StubTask(InputEndTimeTask, time.Date(2023, time.April, 1, 12, 0, 0, 0, time.UTC), nil)

	form_task_test.TestTextForms(t, "duration", InputDurationTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "With valid time duration",
			Input:         "10m",
			ExpectedValue: time.Duration(time.Minute) * 10,
			Dependencies:  []coretask.UntypedTask{endTimeTask, currentTimeTask1, timezoneTaskUTC},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					HintType:    inspectionmetadata.Info,
					Hint: `Query range:
2023-04-01T11:50:00Z ~ 2023-04-01T12:00:00Z
(UTC: 2023-04-01T11:50:00 ~ 2023-04-01T12:00:00)
(PDT: 2023-04-01T04:50:00 ~ 2023-04-01T05:00:00)`,
				},
				Suggestions: expectedSuggestions,
				Default:     "1h",
			},
		},
		{
			Name:          "With invalid time duration",
			Input:         "foo",
			ExpectedValue: time.Hour,
			Dependencies:  []coretask.UntypedTask{endTimeTask, currentTimeTask1, timezoneTaskUTC},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					Hint:        "time: invalid duration \"foo\"",
					HintType:    inspectionmetadata.Error,
				},
				Default:     "1h",
				Suggestions: expectedSuggestions,
			},
		},
		{
			Name:          "With invalid time duration(negative)",
			Input:         "-10m",
			ExpectedValue: time.Hour,
			Dependencies:  []coretask.UntypedTask{endTimeTask, currentTimeTask1, timezoneTaskUTC},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					Hint:        "duration must be positive",
					HintType:    inspectionmetadata.Error,
				},
				Suggestions: expectedSuggestions,
				Default:     "1h",
			},
		},
		{
			Name:          "with longer duration starting before than 30 days",
			Input:         "672h", // starting time will be 30 days before the inspection time
			ExpectedValue: time.Hour * 672,
			Dependencies:  []coretask.UntypedTask{endTimeTask, currentTimeTask1, timezoneTaskUTC},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Type:        "Text",
					Label:       expectedLabel,
					Description: expectedDescription,
					Hint: `Specified time range starts from over than 30 days ago, maybe some logs are missing and the generated result could be incomplete.
This duration can be too long for big clusters and lead OOM. Please retry with shorter duration when your machine crashed.
Query range:
2023-03-04T12:00:00Z ~ 2023-04-01T12:00:00Z
(UTC: 2023-03-04T12:00:00 ~ 2023-04-01T12:00:00)
(PDT: 2023-03-04T05:00:00 ~ 2023-04-01T05:00:00)`,
					HintType: inspectionmetadata.Info,
				},
				Suggestions: expectedSuggestions,
				Default:     "1h",
			},
		},
		{
			Name:          "With non UTC timezone",
			Input:         "1h",
			ExpectedValue: time.Hour,
			Dependencies:  []coretask.UntypedTask{endTimeTask, currentTimeTask1, timezoneTaskJST},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Type:        "Text",
					Label:       expectedLabel,
					Description: expectedDescription,
					Hint: `Query range:
2023-04-01T20:00:00+09:00 ~ 2023-04-01T21:00:00+09:00
(UTC: 2023-04-01T11:00:00 ~ 2023-04-01T12:00:00)
(PDT: 2023-04-01T04:00:00 ~ 2023-04-01T05:00:00)`,
					HintType: inspectionmetadata.Info,
				},
				Suggestions: expectedSuggestions,
				Default:     "1h",
			},
		},
	})
}
