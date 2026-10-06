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
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDefineSetForm_FormField(t *testing.T) {
	testCases := []struct {
		Name              string
		Spec              SetFormSpec[[]string]
		RequestValue      interface{} // Change to interface{} to allow passing []string or []interface{}
		ExpectedFormField inspectionmetadata.ParameterFormField
		ExpectedValue     any
		ExpectedError     string
	}{
		{
			Name:          "A set form with given parameter",
			RequestValue:  []string{"bar"},
			ExpectedValue: []string{"bar"},
			ExpectedError: "",
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
			Spec: SetFormSpec[[]string]{
				DefaultValue: PreviousOrConstantDefaultValue([]string{"foo-default"}),
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
			Spec: SetFormSpec[[]string]{
				Options: ConstantOptions(
					inspectionmetadata.SetParameterFormFieldOptionItem{ID: "opt1"},
					inspectionmetadata.SetParameterFormFieldOptionItem{ID: "opt2"},
				),
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
			Spec: SetFormSpec[[]string]{
				AllowCustomValue: ConstantBool(true),
				DisableAddAll:    ConstantBool(true),
				DisableRemoveAll: ConstantBool(true),
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
			taskDef := DefineSetForm(taskid.NewDefaultImplementationID[[]string]("foo-set"), 1, "foo label", "", func(b *coretask.Binder) SetFormSpec[[]string] {
				return testCase.Spec
			})
			formFields := []inspectionmetadata.ParameterFormField{}

			// Execute task as DryRun mode
			taskCtx := context.Background()
			taskCtx = inspectiontest.WithDefaultTestInspectionTaskContext(taskCtx)

			inputMap := map[string]any{}
			if testCase.RequestValue != nil {
				inputMap["foo-set"] = testCase.RequestValue
			}

			_, _, err := inspectiontest.Run(t, taskCtx, taskDef, inspectioncore.TaskModeDryRun, inputMap)
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
				metadata := khictx.MustGetValue(taskCtx, inspectionmetadata.MapContextKey)

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
				result, _, err := inspectiontest.Run(t, taskCtx, taskDef, inspectioncore.TaskModeRun, inputMap)

				if err != nil {
					t.Errorf("task was ended with unexpected error\n%s", err)
				}
				if diff := cmp.Diff(testCase.ExpectedValue, result); diff != "" {
					t.Errorf("the result is not matching with the expected value\n%s", diff)
				}
				metadata := khictx.MustGetValue(taskCtx, inspectionmetadata.MapContextKey)

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

func TestDefineSetForm(t *testing.T) {
	sourceTaskID := taskid.NewDefaultImplementationID[[]string]("source")
	formID := taskid.NewDefaultImplementationID[[]string]("set-form")

	makeTask := func(customizeSpec func(source coretask.Input[[]string]) SetFormSpec[[]string]) coretask.Task[[]string] {
		return DefineSetForm(formID, 1, "test set form", "test description", func(b *coretask.Binder) SetFormSpec[[]string] {
			source := coretask.Use(b, sourceTaskID.Ref())
			if customizeSpec != nil {
				return customizeSpec(source)
			}
			return SetFormSpec[[]string]{
				DefaultValue: func(ctx context.Context, previousValues []string) ([]string, error) {
					return source.Get(ctx), nil
				},
				Options: func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
					opts := make([]inspectionmetadata.SetParameterFormFieldOptionItem, len(source.Get(ctx)))
					for i, opt := range source.Get(ctx) {
						opts[i] = inspectionmetadata.SetParameterFormFieldOptionItem{ID: opt}
					}
					return opts, nil
				},
			}
		})
	}

	testCases := []struct {
		name         string
		task         coretask.Task[[]string]
		taskMode     inspectioncore.InspectionTaskModeType
		requestValue map[string]any
		sourceValue  []string
		wantValue    []string
		wantErr      bool
		checkField   func(t *testing.T, field inspectionmetadata.SetParameterFormField)
	}{
		{
			name:        "callbacks read the declared input",
			task:        makeTask(nil),
			taskMode:    inspectioncore.TaskModeDryRun,
			sourceValue: []string{"opt1", "opt2"},
			wantValue:   []string{"opt1", "opt2"},
			checkField: func(t *testing.T, field inspectionmetadata.SetParameterFormField) {
				if diff := cmp.Diff([]string{"opt1", "opt2"}, field.Default); diff != "" {
					t.Errorf("field.Default mismatch (-want +got):\n%s", diff)
				}
				wantOpts := []inspectionmetadata.SetParameterFormFieldOptionItem{
					{ID: "opt1"},
					{ID: "opt2"},
				}
				if diff := cmp.Diff(wantOpts, field.Options); diff != "" {
					t.Errorf("field.Options mismatch (-want +got):\n%s", diff)
				}
				if field.Label != "test set form" {
					t.Errorf("field.Label = %q, want %q", field.Label, "test set form")
				}
				if field.Description != "test description" {
					t.Errorf("field.Description = %q, want %q", field.Description, "test description")
				}
				if field.Priority != 1 {
					t.Errorf("field.Priority = %d, want %d", field.Priority, 1)
				}
			},
		},
		{
			name: "nil fields use the defaults",
			task: makeTask(func(coretask.Input[[]string]) SetFormSpec[[]string] {
				return SetFormSpec[[]string]{}
			}),
			taskMode:     inspectioncore.TaskModeDryRun,
			requestValue: map[string]any{formID.ReferenceIDString(): []any{"abc"}},
			sourceValue:  []string{"opt1"},
			wantValue:    []string{"abc"},
			checkField: func(t *testing.T, field inspectionmetadata.SetParameterFormField) {
				if field.AllowCustomValue != false {
					t.Errorf("field.AllowCustomValue = %v, want false", field.AllowCustomValue)
				}
				if field.AllowAddAll != true {
					t.Errorf("field.AllowAddAll = %v, want true", field.AllowAddAll)
				}
				if field.AllowRemoveAll != true {
					t.Errorf("field.AllowRemoveAll = %v, want true", field.AllowRemoveAll)
				}
				if diff := cmp.Diff([]inspectionmetadata.SetParameterFormFieldOptionItem{}, field.Options); diff != "" {
					t.Errorf("field.Options mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "validation error falls back to the default in dry run",
			task: makeTask(func(source coretask.Input[[]string]) SetFormSpec[[]string] {
				return SetFormSpec[[]string]{
					DefaultValue: func(ctx context.Context, previousValues []string) ([]string, error) {
						return source.Get(ctx), nil
					},
					Validator: func(ctx context.Context, value []string) (string, error) {
						for _, v := range value {
							if v == "bad" {
								return "bad value", nil
							}
						}
						return "", nil
					},
				}
			}),
			taskMode:     inspectioncore.TaskModeDryRun,
			requestValue: map[string]any{formID.ReferenceIDString(): []any{"bad"}},
			sourceValue:  []string{"opt1"},
			wantValue:    []string{"opt1"},
			checkField: func(t *testing.T, field inspectionmetadata.SetParameterFormField) {
				if field.HintType != inspectionmetadata.Error {
					t.Errorf("field.HintType = %v, want %v", field.HintType, inspectionmetadata.Error)
				}
				if field.Hint != "bad value" {
					t.Errorf("field.Hint = %q, want %q", field.Hint, "bad value")
				}
			},
		},
		{
			name: "run mode rejects invalid value",
			task: makeTask(func(source coretask.Input[[]string]) SetFormSpec[[]string] {
				return SetFormSpec[[]string]{
					DefaultValue: func(ctx context.Context, previousValues []string) ([]string, error) {
						return source.Get(ctx), nil
					},
					Validator: func(ctx context.Context, value []string) (string, error) {
						for _, v := range value {
							if v == "bad" {
								return "bad value", nil
							}
						}
						return "", nil
					},
				}
			}),
			taskMode:     inspectioncore.TaskModeRun,
			requestValue: map[string]any{formID.ReferenceIDString(): []string{"bad"}},
			sourceValue:  []string{"opt1"},
			wantErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			req := tc.requestValue
			if req == nil {
				req = map[string]any{}
			}
			got, metadata, err := inspectiontest.Run(t, ctx, tc.task, tc.taskMode, req,
				tasktest.Given(sourceTaskID.Ref(), tc.sourceValue),
			)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantValue, got); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
			if tc.checkField != nil {
				fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
				if !found {
					t.Fatalf("form field set not found on metadata")
				}
				rawField := fields.DangerouslyGetField(formID.ReferenceIDString())
				setField, ok := rawField.(inspectionmetadata.SetParameterFormField)
				if !ok {
					t.Fatalf("field is not SetParameterFormField: %T", rawField)
				}
				tc.checkField(t, setField)
			}
		})
	}

	t.Run("declares required input on source reference", func(t *testing.T) {
		task := makeTask(nil)
		wantInputs := []string{"required source"}
		if diff := cmp.Diff(wantInputs, describeInputs(task.Inputs())); diff != "" {
			t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("has form task label", func(t *testing.T) {
		task := makeTask(nil)
		isFormTask, found := typedmap.Get(task.Labels(), inspectioncore.TaskLabelKeyIsFormTask)
		if !found || !isFormTask {
			t.Errorf("task label %v = %v, want true", inspectioncore.TaskLabelKeyIsFormTask, isFormTask)
		}
	})
}

// TestPreviousOrConstantDefaultValue checks the default value resolution based on previous values.
func TestPreviousOrConstantDefaultValue(t *testing.T) {
	testCases := []struct {
		name           string
		values         []string
		previousValues []string
		want           []string
		wantNonNil     bool
	}{
		{
			name:           "previous values exist",
			values:         []string{"fallback"},
			previousValues: []string{"prev"},
			want:           []string{"prev"},
		},
		{
			name:           "previous nil",
			values:         []string{"fallback"},
			previousValues: nil,
			want:           []string{"fallback"},
		},
		{
			name:           "previous empty",
			values:         []string{"fallback"},
			previousValues: []string{},
			want:           []string{"fallback"},
		},
		{
			name:           "values is an empty non-nil slice and no previous",
			values:         []string{},
			previousValues: nil,
			want:           []string{},
			wantNonNil:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generator := PreviousOrConstantDefaultValue(tc.values)
			got, err := generator(t.Context(), tc.previousValues)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNonNil && got == nil {
				t.Errorf("PreviousOrConstantDefaultValue() returned nil, want non-nil slice")
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("PreviousOrConstantDefaultValue() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
