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

package formtask

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type setFormConfigurator = func(builder *SetFormTaskBuilder[[]string])

func TestSetFormDefinitionBuilder(t *testing.T) {
	testCases := []struct {
		Name              string
		FormConfigurator  setFormConfigurator
		RequestValue      interface{} // Change to interface{} to allow passing []string or []interface{}
		ExpectedFormField inspectionmetadata.ParameterFormField
		ExpectedValue     any
		ExpectedError     string
	}{
		{
			Name:             "A set form with given parameter",
			FormConfigurator: func(builder *SetFormTaskBuilder[[]string]) {},
			RequestValue:     []string{"bar"},
			ExpectedValue:    []string{"bar"},
			ExpectedError:    "",
			ExpectedFormField: inspectionmetadata.SetParameterFormField{
				AllowCustomValue: false,
				AllowAddAll:      true,
				AllowRemoveAll:   true,
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Options: []inspectionmetadata.SetParameterFormFieldOptionItem{},
			},
		},
		{
			Name: "A set form with default parameter",
			FormConfigurator: func(builder *SetFormTaskBuilder[[]string]) {
				builder.WithDefaultValueConstant([]string{"foo-default"}, true)
			},
			RequestValue:  nil, // Simulate missing input
			ExpectedValue: []string{"foo-default"},
			ExpectedError: "",
			ExpectedFormField: inspectionmetadata.SetParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				AllowCustomValue: false,
				AllowAddAll:      true,
				AllowRemoveAll:   true,
				Default:          []string{"foo-default"},
				Options:          []inspectionmetadata.SetParameterFormFieldOptionItem{},
			},
		},
		{
			Name: "A set form with options",
			FormConfigurator: func(builder *SetFormTaskBuilder[[]string]) {
				builder.WithOptionsSimple([]string{"opt1", "opt2"})
			},
			RequestValue:  []string{"opt1"},
			ExpectedValue: []string{"opt1"},
			ExpectedError: "",
			ExpectedFormField: inspectionmetadata.SetParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				AllowCustomValue: false,
				AllowAddAll:      true,
				AllowRemoveAll:   true,
				Options: []inspectionmetadata.SetParameterFormFieldOptionItem{
					{ID: "opt1"},
					{ID: "opt2"},
				},
			},
		},
		{
			Name: "A set form with custom configuration",
			FormConfigurator: func(builder *SetFormTaskBuilder[[]string]) {
				builder.WithAllowCustomValue(true).WithAllowAddAll(false).WithAllowRemoveAll(false)
			},
			RequestValue:  []string{"custom"},
			ExpectedValue: []string{"custom"},
			ExpectedError: "",
			ExpectedFormField: inspectionmetadata.SetParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				AllowCustomValue: true,
				AllowAddAll:      false,
				AllowRemoveAll:   false,
				Options:          []inspectionmetadata.SetParameterFormFieldOptionItem{},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			originalBuilder := NewSetFormTaskBuilder(taskid.NewDefaultImplementationID[[]string]("foo-set"), 1, "foo label")
			testCase.FormConfigurator(originalBuilder)
			taskDef := originalBuilder.Build()
			formFields := []inspectionmetadata.ParameterFormField{}

			// Execute task as DryRun mode
			taskCtx := context.Background()
			taskCtx = inspectiontest.WithDefaultTestInspectionTaskContext(taskCtx)

			inputMap := map[string]any{}
			if testCase.RequestValue != nil {
				inputMap["foo-set"] = testCase.RequestValue
			}

			_, _, err := inspectiontest.RunInspectionTask(taskCtx, taskDef, inspectioncore_contract.TaskModeDryRun, inputMap)
			if testCase.ExpectedError != "" {
				if err == nil {
					t.Errorf("task was expected to be end with an error. But the task finished without an error")
				} else if err.Error() != testCase.ExpectedError {
					t.Errorf("task was expected to be end with an error. But the expected error is different.\n expected:%s\nactual:%s", testCase.ExpectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("task was ended with unexpected error\n%s", err)
				}
				metadata := khictx.MustGetValue(taskCtx, inspectioncore_contract.InspectionRunMetadata)

				fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
				if !found {
					t.Fatal("FormFieldSet not found on metadata")
				}
				field := fields.DangerouslyGetField("foo-set")
				formFields = append(formFields, field)
			}

			// Execute task as Run mode if dry run succeeded
			if testCase.ExpectedError == "" {
				taskCtx := context.Background()
				taskCtx = inspectiontest.WithDefaultTestInspectionTaskContext(taskCtx)
				result, _, err := inspectiontest.RunInspectionTask(taskCtx, taskDef, inspectioncore_contract.TaskModeRun, inputMap)

				if err != nil {
					t.Errorf("task was ended with unexpected error\n%s", err)
				}
				if diff := cmp.Diff(testCase.ExpectedValue, result); diff != "" {
					t.Errorf("the result is not matching with the expected value\n%s", diff)
				}
				metadata := khictx.MustGetValue(taskCtx, inspectioncore_contract.InspectionRunMetadata)

				fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
				if !found {
					t.Fatal("FormFieldSet not found on metadata")
				}
				field := fields.DangerouslyGetField("foo-set")
				formFields = append(formFields, field)

				if diff := cmp.Diff(formFields[0], formFields[1], cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("form field is different between DryRun mode and Run mode with same parameter.\n%s", diff)
				}
			}

			if len(formFields) > 0 {
				if diff := cmp.Diff(formFields[0], testCase.ExpectedFormField, cmpopts.IgnoreFields(inspectionmetadata.SetParameterFormField{}, "ID", "Priority", "Type", "Label")); diff != "" {
					t.Errorf("the generated form field is different from the expected\n%s", diff)
				}
			}
		})
	}
}
