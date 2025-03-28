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

package form

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	form_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	inspection_task_test "github.com/GoogleCloudPlatform/khi/pkg/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type testFormConfigurator = func(builder *TextFormDefinitionBuilder[string])

func TestTextFormDefinitionBuilder(t *testing.T) {
	testCases := []struct {
		Name              string
		FormConfigurator  testFormConfigurator
		RequestValue      string
		ExpectedFormField form_metadata.FormField
		ExpectedValue     any
		ExpectedError     string
	}{
		{
			Name:             "A text form with given parameter",
			FormConfigurator: func(builder *TextFormDefinitionBuilder[string]) {},
			RequestValue:     "bar",
			ExpectedValue:    "bar",
			ExpectedError:    "",
			ExpectedFormField: form_metadata.FormField{
				AllowEdit: true,
				HintType:  form_metadata.HintTypeInfo,
			},
		},
		{
			Name: "A text form with default parameter",
			FormConfigurator: func(builder *TextFormDefinitionBuilder[string]) {
				builder.WithDefaultValueConstant("foo-default", true)
			},
			RequestValue:  "",
			ExpectedValue: "foo-default",
			ExpectedError: "",
			ExpectedFormField: form_metadata.FormField{
				AllowEdit: true,
				Default:   "foo-default",
				HintType:  form_metadata.HintTypeInfo,
			},
		},
		{
			Name: "A text form with validator",
			FormConfigurator: func(builder *TextFormDefinitionBuilder[string]) {
				builder.WithValidator(func(ctx context.Context, value string) (string, error) {
					return "foo validation error", nil
				})
			},
			RequestValue:  "",
			ExpectedValue: "foo-default",
			ExpectedError: "",
			ExpectedFormField: form_metadata.FormField{
				AllowEdit:       true,
				ValidationError: "foo validation error",
				HintType:        form_metadata.HintTypeInfo,
			},
		},
		{
			Name: "A text form with allow edit hand",
			FormConfigurator: func(builder *TextFormDefinitionBuilder[string]) {
				builder.WithAllowEditFunc(func(ctx context.Context) (bool, error) {
					return false, nil
				})
			},
			RequestValue:  "",
			ExpectedValue: "",
			ExpectedError: "",
			ExpectedFormField: form_metadata.FormField{
				AllowEdit: false,
				HintType:  form_metadata.HintTypeInfo,
			},
		},
		{
			Name: "A text form with non allow edit hand but with parameter",
			FormConfigurator: func(builder *TextFormDefinitionBuilder[string]) {
				builder.WithAllowEditFunc(func(ctx context.Context) (bool, error) {
					return false, nil
				}).WithDefaultValueConstant("foo-from-default", true)
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "foo-from-default",
			ExpectedError: "",
			ExpectedFormField: form_metadata.FormField{
				AllowEdit: false,
				Default:   "foo-from-default",
				HintType:  form_metadata.HintTypeInfo,
			},
		},
		{
			Name: "A text form with hint",
			FormConfigurator: func(builder *TextFormDefinitionBuilder[string]) {
				builder.WithHintFunc(func(ctx context.Context, value string, convertedValue any) (string, form_metadata.FormFieldHintType, error) {
					return "foo-hint", form_metadata.HintTypeInfo, nil
				})
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "bar-from-request",
			ExpectedError: "",
			ExpectedFormField: form_metadata.FormField{
				AllowEdit: true,
				Hint:      "foo-hint",
				HintType:  form_metadata.HintTypeInfo,
			},
		},
		{
			Name: "A text form with allow edit but with parameter",
			FormConfigurator: func(builder *TextFormDefinitionBuilder[string]) {
				builder.WithAllowEditFunc(func(ctx context.Context) (bool, error) {
					return true, nil
				}).WithDefaultValueConstant("foo-from-default", true)
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "bar-from-request",
			ExpectedError: "",
			ExpectedFormField: form_metadata.FormField{
				AllowEdit: true,
				Default:   "foo-from-default",
				HintType:  form_metadata.HintTypeInfo,
			},
		},
		{
			Name: "A text form with suggestions",
			FormConfigurator: func(builder *TextFormDefinitionBuilder[string]) {
				builder.WithSuggestionsConstant([]string{
					"foo-suggest1",
					"foo-suggest2",
					"foo-suggest3",
				})
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "bar-from-request",
			ExpectedError: "",
			ExpectedFormField: form_metadata.FormField{
				AllowEdit: true,
				Suggestions: []string{
					"foo-suggest1",
					"foo-suggest2",
					"foo-suggest3",
				},
				HintType: form_metadata.HintTypeInfo,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			originalBuilder := NewInputFormDefinitionBuilder(taskid.NewDefaultImplementationID[string]("foo"), 1, "foo label")
			testCase.FormConfigurator(originalBuilder)
			taskDef := originalBuilder.Build()
			formFields := []form_metadata.FormField{}

			// Execute task as DryRun mode
			taskCtx := context.Background()
			taskCtx = inspection_task_test.WithDefaultTestInspectionTaskContext(taskCtx)

			_, _, err := inspection_task_test.RunInspectionTask(taskCtx, taskDef, inspection_task_interface.TaskModeDryRun, map[string]any{
				"foo": testCase.RequestValue,
			})
			if testCase.ExpectedError != "" {
				if err == nil {
					t.Errorf("task was expected to be end with an error. But the task finished without an error")
				}
				if err.Error() != testCase.ExpectedError {
					t.Errorf("task was expected to be end with an error. But the expected error is different.\n expected:%s\nactual:%s", testCase.ExpectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("task was ended with unexpected error\n%s", err)
				}
				metadata := khictx.MustGetValue(taskCtx, inspection_task_contextkey.InspectionRunMetadata)

				fields, found := typedmap.Get(metadata, form_metadata.FormFieldSetMetadataKey)
				if !found {
					t.Fatal("FormFieldSet not found on metadata")
				}
				field := fields.DangerouslyGetField("foo")
				formFields = append(formFields, field)
			}

			// Execute task as Run mode
			if testCase.ExpectedError != "" {
				taskCtx := context.Background()
				taskCtx = inspection_task_test.WithDefaultTestInspectionTaskContext(taskCtx)
				result, _, err := inspection_task_test.RunInspectionTask(taskCtx, taskDef, inspection_task_interface.TaskModeRun, map[string]any{
					"foo": testCase.RequestValue,
				})

				if testCase.ExpectedError != "" {
					if err == nil {
						t.Errorf("task was expected to be end with an error. But the task finished without an error")
					}
					if err.Error() != testCase.ExpectedError {
						t.Errorf("task was expected to be end with an error. But the expected error is different.\n expected:%s\nactual:%s", testCase.ExpectedError, err.Error())
					}
				} else {
					if err != nil {
						t.Errorf("task was ended with unexpected error\n%s", err)
					}
					if result != testCase.RequestValue {
						t.Errorf("the result is not matching with the expected value\nexpected:%s\nactual:%s", testCase.RequestValue, result)
					}
					metadata := khictx.MustGetValue(taskCtx, inspection_task_contextkey.InspectionRunMetadata)

					fields, found := typedmap.Get(metadata, form_metadata.FormFieldSetMetadataKey)
					if !found {
						t.Fatal("FormFieldSet not found on metadata")
					}
					field := fields.DangerouslyGetField("foo")
					formFields = append(formFields, field)
				}

				if diff := cmp.Diff(formFields[0], formFields[1]); diff != "" {
					t.Errorf("form field is different between DryRun mode and Run mode with same parameter.\n%s", diff)
				}
			}
			if diff := cmp.Diff(formFields[0], testCase.ExpectedFormField, cmpopts.IgnoreFields(form_metadata.FormField{}, "Id", "Priority", "Type", "Label")); diff != "" {
				t.Errorf("the generated form field is different from the expected\n%s", diff)
			}
		})
	}
}
