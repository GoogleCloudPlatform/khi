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

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	form_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func generateFakeVariableSet(taskId string, value string) *common_task.VariableSet {
	requestMap := map[string]any{}
	if value != "" {
		requestMap[taskId] = value
	}
	m := metadata.NewSet()
	vs := common_task.NewVariableSet(map[string]any{
		task.MetadataVariableName: m,
		task.InspectionRequestVariableName: &task.InspectionRequest{
			Values: requestMap,
		},
		common_task.TaskCacheTaskID: common_task.NewLocalTaskVariableCache(),
	})

	return vs
}

type testFormConfigurator = func(builder *TextFormDefinitionBuilder)

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
			FormConfigurator: func(builder *TextFormDefinitionBuilder) {},
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
			FormConfigurator: func(builder *TextFormDefinitionBuilder) {
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
			FormConfigurator: func(builder *TextFormDefinitionBuilder) {
				builder.WithValidator(func(ctx context.Context, value string, variables *common_task.VariableSet) (string, error) {
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
			FormConfigurator: func(builder *TextFormDefinitionBuilder) {
				builder.WithAllowEditFunc(func(ctx context.Context, variables *common_task.VariableSet) (bool, error) {
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
			FormConfigurator: func(builder *TextFormDefinitionBuilder) {
				builder.WithAllowEditFunc(func(ctx context.Context, variables *common_task.VariableSet) (bool, error) {
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
			FormConfigurator: func(builder *TextFormDefinitionBuilder) {
				builder.WithHintFunc(func(ctx context.Context, value string, convertedValue any, variables *common_task.VariableSet) (string, form_metadata.FormFieldHintType, error) {
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
			FormConfigurator: func(builder *TextFormDefinitionBuilder) {
				builder.WithAllowEditFunc(func(ctx context.Context, variables *common_task.VariableSet) (bool, error) {
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
			FormConfigurator: func(builder *TextFormDefinitionBuilder) {
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
			originalBuilder := NewInputFormDefinitionBuilder("foo", 1, "foo label")
			testCase.FormConfigurator(originalBuilder)
			taskDef := originalBuilder.Build()
			formFields := []form_metadata.FormField{}

			// Execute task as DryRun mode
			vs := generateFakeVariableSet("foo", testCase.RequestValue)
			err := taskDef.Runnable(task.TaskModeDryRun).Run(context.Background(), vs)
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
				ms, err := task.GetMetadataSetFromVariable(vs)
				if err != nil {
					t.Errorf("unexpected error while getting metadata\n%v", err)
				}
				field := ms.LoadOrStore(form_metadata.FormFieldSetMetadataKey, &form_metadata.FormFieldSetMetadataFactory{}).(*form_metadata.FormFieldSet).DangerouslyGetField("foo")
				formFields = append(formFields, field)
			}

			// Execute task as Run mode
			if testCase.ExpectedError != "" {
				vs = generateFakeVariableSet("foo", testCase.RequestValue)
				err = taskDef.Runnable(task.TaskModeRun).Run(context.Background(), vs)
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
					ms, err := task.GetMetadataSetFromVariable(vs)
					if err != nil {
						t.Errorf("unexpected error while getting metadata\n%v", err)
					}
					field := ms.LoadOrStore(form_metadata.FormFieldSetMetadataKey, &form_metadata.FormFieldSetMetadataFactory{}).(*form_metadata.FormFieldSet).DangerouslyGetField("foo")
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
