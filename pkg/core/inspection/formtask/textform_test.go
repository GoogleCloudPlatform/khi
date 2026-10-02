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

package formtask

import (
	"context"
	"fmt"
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

// TestDefineTextForm_FormField checks the form field and value computed from each spec setting.
func TestDefineTextForm_FormField(t *testing.T) {
	getField := func(t *testing.T, ctx context.Context) inspectionmetadata.ParameterFormField {
		t.Helper()
		metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
		fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
		if !found {
			t.Fatal("FormFieldSet not found on metadata")
		}
		return fields.DangerouslyGetField("foo")
	}

	testCases := []struct {
		Name              string
		Spec              TextFormSpec[string]
		RequestValue      string
		ExpectedFormField inspectionmetadata.ParameterFormField
		ExpectedValue     string
		ExpectRunError    bool
	}{
		{
			Name:          "A text form with given parameter",
			Spec:          TextFormSpec[string]{},
			RequestValue:  "bar",
			ExpectedValue: "bar",
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				Readonly: false,
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name: "A text form with default parameter",
			Spec: TextFormSpec[string]{
				DefaultValue: func(ctx context.Context, previousValues []string) (string, error) {
					if len(previousValues) > 0 {
						return previousValues[0], nil
					}
					return "foo-default", nil
				},
			},
			RequestValue:  "",
			ExpectedValue: "foo-default",
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly:         false,
				Default:          "foo-default",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name: "A text form with validator",
			Spec: TextFormSpec[string]{
				Validator: func(ctx context.Context, value string) (string, error) {
					return "foo validation error", nil
				},
			},
			RequestValue:   "",
			ExpectedValue:  "",
			ExpectRunError: true,
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.Error,
					Hint:     "foo validation error",
				},
				Readonly:         false,
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name: "A text form with allow edit hand",
			Spec: TextFormSpec[string]{
				Readonly: func(ctx context.Context) (bool, error) {
					return true, nil
				},
			},
			RequestValue:  "",
			ExpectedValue: "",
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly:         true,
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name: "A text form with non allow edit hand but with parameter",
			Spec: TextFormSpec[string]{
				DefaultValue: func(ctx context.Context, previousValues []string) (string, error) {
					if len(previousValues) > 0 {
						return previousValues[0], nil
					}
					return "foo-from-default", nil
				},
				Readonly: func(ctx context.Context) (bool, error) {
					return true, nil
				},
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "foo-from-default",
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly:         true,
				Default:          "foo-from-default",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name: "A text form with hint",
			Spec: TextFormSpec[string]{
				Hint: func(ctx context.Context, value string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
					return "foo-hint", inspectionmetadata.Info, nil
				},
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "bar-from-request",
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.Info,
					Hint:     "foo-hint",
				},
				Readonly:         false,
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name: "A text form with allow edit but with parameter",
			Spec: TextFormSpec[string]{
				DefaultValue: func(ctx context.Context, previousValues []string) (string, error) {
					if len(previousValues) > 0 {
						return previousValues[0], nil
					}
					return "foo-from-default", nil
				},
				Readonly: func(ctx context.Context) (bool, error) {
					return true, nil
				},
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "foo-from-default",
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly:         true,
				Default:          "foo-from-default",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name: "A text form with suggestions",
			Spec: TextFormSpec[string]{
				Suggestions: ConstantSuggestions(
					"foo-suggest1",
					"foo-suggest2",
					"foo-suggest3",
				),
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "bar-from-request",
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly: false,
				Suggestions: []string{
					"foo-suggest1",
					"foo-suggest2",
					"foo-suggest3",
				},
				ValidationTiming: inspectionmetadata.Change,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			taskDef := DefineTextForm(taskid.NewDefaultImplementationID[string]("foo"), 1, "foo label", "", func(*coretask.Binder) TextFormSpec[string] {
				return testCase.Spec
			})

			// An empty RequestValue means the request does not contain the field.
			request := map[string]any{}
			if testCase.RequestValue != "" {
				request["foo"] = testCase.RequestValue
			}

			// 1. DryRun
			dryRunCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.Run(t, dryRunCtx, taskDef, inspectioncore.TaskModeDryRun, request)
			if err != nil {
				t.Fatalf("DryRun unexpected error: %v", err)
			}
			if got != testCase.ExpectedValue {
				t.Errorf("DryRun value = %q, want %q", got, testCase.ExpectedValue)
			}
			dryRunField := getField(t, dryRunCtx)
			if diff := cmp.Diff(testCase.ExpectedFormField, dryRunField, cmpopts.IgnoreFields(inspectionmetadata.TextParameterFormField{}, "ID", "Priority", "Type", "Label")); diff != "" {
				t.Errorf("DryRun form field mismatch (-want +got):\n%s", diff)
			}

			// 2. Run with a fresh context
			runCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err = inspectiontest.Run(t, runCtx, taskDef, inspectioncore.TaskModeRun, request)
			if testCase.ExpectRunError {
				if err == nil {
					t.Errorf("Run returned no error, want a validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Run unexpected error: %v", err)
			}
			if got != testCase.ExpectedValue {
				t.Errorf("Run value = %q, want %q", got, testCase.ExpectedValue)
			}
			runField := getField(t, runCtx)
			if diff := cmp.Diff(dryRunField, runField); diff != "" {
				t.Errorf("form field differs between DryRun and Run (-dryrun +run):\n%s", diff)
			}
		})
	}
}

// describeInputs converts point-to-point input specs to comparable strings.
func describeInputs(specs []coretask.InputSpec) []string {
	result := make([]string, 0, len(specs))
	for _, spec := range specs {
		result = append(result, fmt.Sprintf("%s %s", spec.Kind, spec.Dependency.(taskid.PointToPointDescriptor).ReferenceID()))
	}
	return result
}

func TestDefineTextForm(t *testing.T) {
	sourceTaskID := taskid.NewDefaultImplementationID[string]("source")
	formID := taskid.NewDefaultImplementationID[string]("text-form")

	makeTask := func(customizeSpec func(source coretask.Input[string]) TextFormSpec[string]) coretask.DefinedTask[string] {
		return DefineTextForm(formID, 1, "test text form", "test description", func(b *coretask.Binder) TextFormSpec[string] {
			source := coretask.Use(b, sourceTaskID.Ref())
			if customizeSpec != nil {
				return customizeSpec(source)
			}
			return TextFormSpec[string]{
				DefaultValue: func(ctx context.Context, previousValues []string) (string, error) {
					return "default-" + source.Get(ctx), nil
				},
				Suggestions: func(ctx context.Context, value string, previousValues []string) ([]string, error) {
					return []string{"suggest-" + source.Get(ctx)}, nil
				},
			}
		})
	}

	testCases := []struct {
		name         string
		task         coretask.DefinedTask[string]
		taskMode     inspectioncore.InspectionTaskModeType
		requestValue map[string]any
		sourceValue  string
		wantValue    string
		wantErr      bool
		checkField   func(t *testing.T, field inspectionmetadata.TextParameterFormField)
	}{
		{
			name:        "callbacks read the declared input",
			task:        makeTask(nil),
			taskMode:    inspectioncore.TaskModeDryRun,
			sourceValue: "src",
			wantValue:   "default-src",
			checkField: func(t *testing.T, field inspectionmetadata.TextParameterFormField) {
				if field.Default != "default-src" {
					t.Errorf("field.Default = %q, want %q", field.Default, "default-src")
				}
				if diff := cmp.Diff([]string{"suggest-src"}, field.Suggestions); diff != "" {
					t.Errorf("field.Suggestions mismatch (-want +got):\n%s", diff)
				}
				if field.Label != "test text form" {
					t.Errorf("field.Label = %q, want %q", field.Label, "test text form")
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
			task: makeTask(func(coretask.Input[string]) TextFormSpec[string] {
				return TextFormSpec[string]{}
			}),
			taskMode:     inspectioncore.TaskModeDryRun,
			requestValue: map[string]any{formID.ReferenceIDString(): "abc"},
			sourceValue:  "src",
			wantValue:    "abc",
			checkField: func(t *testing.T, field inspectionmetadata.TextParameterFormField) {
				if field.Readonly != false {
					t.Errorf("field.Readonly = %v, want false", field.Readonly)
				}
				if field.Suggestions != nil {
					t.Errorf("field.Suggestions = %v, want nil", field.Suggestions)
				}
				if field.ValidationTiming != inspectionmetadata.Change {
					t.Errorf("field.ValidationTiming = %v, want %v", field.ValidationTiming, inspectionmetadata.Change)
				}
				if field.HintType != inspectionmetadata.None {
					t.Errorf("field.HintType = %v, want %v", field.HintType, inspectionmetadata.None)
				}
			},
		},
		{
			name: "validation error falls back to the default in dry run",
			task: makeTask(func(source coretask.Input[string]) TextFormSpec[string] {
				return TextFormSpec[string]{
					DefaultValue: func(ctx context.Context, previousValues []string) (string, error) {
						return "default-" + source.Get(ctx), nil
					},
					Validator: func(ctx context.Context, value string) (string, error) {
						if value == "bad" {
							return "bad value", nil
						}
						return "", nil
					},
				}
			}),
			taskMode:     inspectioncore.TaskModeDryRun,
			requestValue: map[string]any{formID.ReferenceIDString(): "bad"},
			sourceValue:  "src",
			wantValue:    "default-src",
			checkField: func(t *testing.T, field inspectionmetadata.TextParameterFormField) {
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
			task: makeTask(func(source coretask.Input[string]) TextFormSpec[string] {
				return TextFormSpec[string]{
					DefaultValue: func(ctx context.Context, previousValues []string) (string, error) {
						return "default-" + source.Get(ctx), nil
					},
					Validator: func(ctx context.Context, value string) (string, error) {
						if value == "bad" {
							return "bad value", nil
						}
						return "", nil
					},
				}
			}),
			taskMode:     inspectioncore.TaskModeRun,
			requestValue: map[string]any{formID.ReferenceIDString(): "bad"},
			sourceValue:  "src",
			wantErr:      true,
		},
		{
			name: "validation timing is applied",
			task: makeTask(func(coretask.Input[string]) TextFormSpec[string] {
				return TextFormSpec[string]{
					ValidationTiming: inspectionmetadata.Blur,
				}
			}),
			taskMode:    inspectioncore.TaskModeDryRun,
			sourceValue: "src",
			wantValue:   "",
			checkField: func(t *testing.T, field inspectionmetadata.TextParameterFormField) {
				if field.ValidationTiming != inspectionmetadata.Blur {
					t.Errorf("field.ValidationTiming = %v, want %v", field.ValidationTiming, inspectionmetadata.Blur)
				}
			},
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
			if got != tc.wantValue {
				t.Errorf("got %q, want %q", got, tc.wantValue)
			}
			if tc.checkField != nil {
				fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
				if !found {
					t.Fatalf("form field set not found on metadata")
				}
				rawField := fields.DangerouslyGetField(formID.ReferenceIDString())
				textField, ok := rawField.(inspectionmetadata.TextParameterFormField)
				if !ok {
					t.Fatalf("field is not TextParameterFormField: %T", rawField)
				}
				tc.checkField(t, textField)
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

func TestPreviousOrDefaultValue(t *testing.T) {
	testCases := []struct {
		name           string
		previousValues []string
		defaultValue   func(ctx context.Context) (string, error)
		want           string
		wantErr        bool
	}{
		{
			name:           "returns the first previous value without calling defaultValue",
			previousValues: []string{"prev-1", "prev-2"},
			defaultValue: func(ctx context.Context) (string, error) {
				t.Fatal("defaultValue should not be called when previousValues is not empty")
				return "", nil
			},
			want: "prev-1",
		},
		{
			name:           "computes the default value when previousValues is empty",
			previousValues: nil,
			defaultValue: func(ctx context.Context) (string, error) {
				return "computed-default", nil
			},
			want: "computed-default",
		},
		{
			name:           "propagates error from defaultValue",
			previousValues: []string{},
			defaultValue: func(ctx context.Context) (string, error) {
				return "", fmt.Errorf("compute error")
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gen := PreviousOrDefaultValue(tc.defaultValue)
			got, err := gen(t.Context(), tc.previousValues)
			if (err != nil) != tc.wantErr {
				t.Fatalf("PreviousOrDefaultValue() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("PreviousOrDefaultValue() = %q, want %q", got, tc.want)
			}
		})
	}
}
