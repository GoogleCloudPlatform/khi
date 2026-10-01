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

type testFormConfigurator = func(builder *TextFormTaskBuilder[string])

func TestTextFormDefinitionBuilder(t *testing.T) {
	testCases := []struct {
		Name              string
		FormConfigurator  testFormConfigurator
		RequestValue      string
		ExpectedFormField inspectionmetadata.ParameterFormField
		ExpectedValue     any
		ExpectedError     string
	}{
		{
			Name:             "A text form with given parameter",
			FormConfigurator: func(builder *TextFormTaskBuilder[string]) {},
			RequestValue:     "bar",
			ExpectedValue:    "bar",
			ExpectedError:    "",
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
			FormConfigurator: func(builder *TextFormTaskBuilder[string]) {
				builder.WithDefaultValueConstant("foo-default", true)
			},
			RequestValue:  "",
			ExpectedValue: "foo-default",
			ExpectedError: "",
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
			FormConfigurator: func(builder *TextFormTaskBuilder[string]) {
				builder.WithValidator(func(ctx context.Context, value string) (string, error) {
					return "foo validation error", nil
				})
			},
			RequestValue:  "",
			ExpectedValue: "foo-default",
			ExpectedError: "",
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
			FormConfigurator: func(builder *TextFormTaskBuilder[string]) {
				builder.WithReadonlyFunc(func(ctx context.Context) (bool, error) {
					return true, nil
				})
			},
			RequestValue:  "",
			ExpectedValue: "",
			ExpectedError: "",
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
			FormConfigurator: func(builder *TextFormTaskBuilder[string]) {
				builder.WithReadonlyFunc(func(ctx context.Context) (bool, error) {
					return true, nil
				}).WithDefaultValueConstant("foo-from-default", true)
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "foo-from-default",
			ExpectedError: "",
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
			FormConfigurator: func(builder *TextFormTaskBuilder[string]) {
				builder.WithHintFunc(func(ctx context.Context, value string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
					return "foo-hint", inspectionmetadata.Info, nil
				})
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "bar-from-request",
			ExpectedError: "",
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
			FormConfigurator: func(builder *TextFormTaskBuilder[string]) {
				builder.WithReadonlyFunc(func(ctx context.Context) (bool, error) {
					return true, nil
				}).WithDefaultValueConstant("foo-from-default", true)
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "bar-from-request",
			ExpectedError: "",
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
			FormConfigurator: func(builder *TextFormTaskBuilder[string]) {
				builder.WithSuggestionsConstant([]string{
					"foo-suggest1",
					"foo-suggest2",
					"foo-suggest3",
				})
			},
			RequestValue:  "bar-from-request",
			ExpectedValue: "bar-from-request",
			ExpectedError: "",
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
			originalBuilder := NewTextFormTaskBuilder(taskid.NewDefaultImplementationID[string]("foo"), 1, "foo label")
			testCase.FormConfigurator(originalBuilder)
			taskDef := originalBuilder.Build()
			formFields := []inspectionmetadata.ParameterFormField{}

			// Execute task as DryRun mode
			taskCtx := context.Background()
			taskCtx = inspectiontest.WithDefaultTestInspectionTaskContext(taskCtx)

			_, _, err := inspectiontest.RunInspectionTask(taskCtx, taskDef, inspectioncore.TaskModeDryRun, map[string]any{
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
				metadata := khictx.MustGetValue(taskCtx, inspectionmetadata.MapContextKey)

				fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
				if !found {
					t.Fatal("FormFieldSet not found on metadata")
				}
				field := fields.DangerouslyGetField("foo")
				formFields = append(formFields, field)
			}

			// Execute task as Run mode
			if testCase.ExpectedError != "" {
				taskCtx := context.Background()
				taskCtx = inspectiontest.WithDefaultTestInspectionTaskContext(taskCtx)
				result, _, err := inspectiontest.RunInspectionTask(taskCtx, taskDef, inspectioncore.TaskModeRun, map[string]any{
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
					metadata := khictx.MustGetValue(taskCtx, inspectionmetadata.MapContextKey)

					fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
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
			if diff := cmp.Diff(formFields[0], testCase.ExpectedFormField, cmpopts.IgnoreFields(inspectionmetadata.TextParameterFormField{}, "ID", "Priority", "Type", "Label")); diff != "" {
				t.Errorf("the generated form field is different from the expected\n%s", diff)
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

func TestTextFormTaskBuilder_Define(t *testing.T) {
	sourceTaskID := taskid.NewDefaultImplementationID[string]("source")
	formID := taskid.NewDefaultImplementationID[string]("text-form")

	makeTask := func(customizeBuilder func(b *TextFormTaskBuilder[string])) coretask.DefinedTask[string] {
		builder := NewTextFormTaskBuilder(formID, 1, "test text form")
		if customizeBuilder != nil {
			customizeBuilder(builder)
		}
		return builder.Define(func(b *coretask.Binder) TextFormFuncs[string] {
			source := coretask.Use(b, sourceTaskID.Ref())
			return TextFormFuncs[string]{
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
			},
		},
		{
			name: "nil fields keep the builder callbacks",
			task: makeTask(func(b *TextFormTaskBuilder[string]) {
				b.WithValidator(func(ctx context.Context, value string) (string, error) {
					if value == "bad" {
						return "bad value", nil
					}
					return "", nil
				})
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
			task: makeTask(func(b *TextFormTaskBuilder[string]) {
				b.WithValidator(func(ctx context.Context, value string) (string, error) {
					if value == "bad" {
						return "bad value", nil
					}
					return "", nil
				})
			}),
			taskMode:     inspectioncore.TaskModeRun,
			requestValue: map[string]any{formID.ReferenceIDString(): "bad"},
			sourceValue:  "src",
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
}
