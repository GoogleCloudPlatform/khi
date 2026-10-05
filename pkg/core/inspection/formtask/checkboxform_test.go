// Copyright 2026 Google LLC
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
	"strings"
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

func TestDefineCheckboxForm_FormField(t *testing.T) {
	testCases := []struct {
		name              string
		spec              CheckboxFormSpec
		requestValue      any
		hasRequestValue   bool
		expectedFormField inspectionmetadata.ParameterFormField
		expectedValue     bool
		expectedError     string
	}{
		{
			name:            "checkbox form with given boolean parameter",
			requestValue:    true,
			hasRequestValue: true,
			expectedValue:   true,
			expectedFormField: inspectionmetadata.CheckboxParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly: false,
				Default:  false,
			},
		},
		{
			name:            "checkbox form with string true parameter",
			requestValue:    "true",
			hasRequestValue: true,
			expectedValue:   true,
			expectedFormField: inspectionmetadata.CheckboxParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly: false,
				Default:  false,
			},
		},
		{
			name:            "checkbox form with string false parameter",
			requestValue:    "false",
			hasRequestValue: true,
			expectedValue:   false,
			expectedFormField: inspectionmetadata.CheckboxParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly: false,
				Default:  false,
			},
		},
		{
			name: "checkbox form with default parameter true",
			spec: CheckboxFormSpec{
				DefaultValue: func(ctx context.Context) (bool, error) { return true, nil },
			},
			hasRequestValue: false,
			expectedValue:   true,
			expectedFormField: inspectionmetadata.CheckboxParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly: false,
				Default:  true,
			},
		},
		{
			name: "checkbox form with validator error",
			spec: CheckboxFormSpec{
				Validator: func(ctx context.Context, value bool) (string, error) {
					if value {
						return "cannot be enabled", nil
					}
					return "", nil
				},
			},
			requestValue:    true,
			hasRequestValue: true,
			expectedValue:   false, // Reverts to default on validation error during DryRun
			expectedFormField: inspectionmetadata.CheckboxParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.Error,
					Hint:     "cannot be enabled",
				},
				Readonly: false,
				Default:  false,
			},
		},
		{
			name: "checkbox form with readonly ignoring request value",
			spec: CheckboxFormSpec{
				Readonly:     func(ctx context.Context) (bool, error) { return true, nil },
				DefaultValue: func(ctx context.Context) (bool, error) { return false, nil },
			},
			requestValue:    true,
			hasRequestValue: true,
			expectedValue:   false,
			expectedFormField: inspectionmetadata.CheckboxParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly: true,
				Default:  false,
			},
		},
		{
			name: "checkbox form with hint",
			spec: CheckboxFormSpec{
				Hint: func(ctx context.Context, value bool) (string, inspectionmetadata.ParameterHintType, error) {
					return "checkbox hint", inspectionmetadata.Info, nil
				},
			},
			hasRequestValue: false,
			expectedValue:   false,
			expectedFormField: inspectionmetadata.CheckboxParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.Info,
					Hint:     "checkbox hint",
				},
				Readonly: false,
				Default:  false,
			},
		},
		{
			name:            "checkbox form with invalid string parameter",
			requestValue:    "not-a-bool",
			hasRequestValue: true,
			expectedError:   "request parameter `foo` was not a valid boolean in task foo#default",
		},
		{
			name:            "checkbox form with invalid parameter type",
			requestValue:    123,
			hasRequestValue: true,
			expectedError:   "request parameter `foo` was not given as boolean or boolean string in task foo#default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskDef := DefineCheckboxForm(taskid.NewDefaultImplementationID[bool]("foo"), 1, "foo label", "foo description", func(b *coretask.Binder) CheckboxFormSpec {
				return tc.spec
			})

			inputs := map[string]any{}
			if tc.hasRequestValue {
				inputs["foo"] = tc.requestValue
			}

			// DryRun mode execution
			dryRunCtx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			_, _, dryRunErr := inspectiontest.Run(t, dryRunCtx, taskDef, inspectioncore.TaskModeDryRun, inputs)

			if tc.expectedError != "" {
				if dryRunErr == nil {
					t.Fatalf("expected error containing %q, got nil", tc.expectedError)
				}
				if !strings.Contains(dryRunErr.Error(), tc.expectedError) {
					t.Fatalf("expected error containing %q, got %q", tc.expectedError, dryRunErr.Error())
				}
				return
			}

			if dryRunErr != nil {
				t.Fatalf("dry run unexpected error: %v", dryRunErr)
			}

			metadata := khictx.MustGetValue(dryRunCtx, inspectionmetadata.MapContextKey)
			fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
			if !found {
				t.Fatal("form field set metadata not found")
			}
			field := fields.DangerouslyGetField("foo")
			checkboxField, ok := field.(inspectionmetadata.CheckboxParameterFormField)
			if !ok {
				t.Fatalf("field type is %T, want CheckboxParameterFormField", field)
			}

			if checkboxField.ID != "foo" {
				t.Errorf("field.ID = %q, want %q", checkboxField.ID, "foo")
			}
			if checkboxField.Label != "foo label" {
				t.Errorf("field.Label = %q, want %q", checkboxField.Label, "foo label")
			}
			if checkboxField.Description != "foo description" {
				t.Errorf("field.Description = %q, want %q", checkboxField.Description, "foo description")
			}
			if checkboxField.Type != inspectionmetadata.Checkbox {
				t.Errorf("field.Type = %q, want %q", checkboxField.Type, inspectionmetadata.Checkbox)
			}

			if diff := cmp.Diff(tc.expectedFormField, checkboxField, cmpopts.IgnoreFields(inspectionmetadata.CheckboxParameterFormField{}, "ID", "Priority", "Type", "Label", "Description")); diff != "" {
				t.Errorf("form field mismatch (-want +got):\n%s", diff)
			}

			// Run mode execution
			runCtx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			runResult, _, runErr := inspectiontest.Run(t, runCtx, taskDef, inspectioncore.TaskModeRun, inputs)

			if checkboxField.HintType == inspectionmetadata.Error {
				if runErr == nil {
					t.Errorf("expected validation error in Run mode, got nil")
				}
			} else {
				if runErr != nil {
					t.Fatalf("run mode unexpected error: %v", runErr)
				}
				if runResult != tc.expectedValue {
					t.Errorf("run mode result = %v, want %v", runResult, tc.expectedValue)
				}
			}
		})
	}
}

func TestDefineCheckboxForm(t *testing.T) {
	sourceTaskID := taskid.NewDefaultImplementationID[bool]("source")
	formID := taskid.NewDefaultImplementationID[bool]("checkbox-form")
	task := DefineCheckboxForm(formID, 1, "test checkbox form", "test description", func(b *coretask.Binder) CheckboxFormSpec {
		source := coretask.Use(b, sourceTaskID.Ref())
		return CheckboxFormSpec{
			DefaultValue: func(ctx context.Context) (bool, error) {
				return source.Get(ctx), nil
			},
			Readonly: func(ctx context.Context) (bool, error) {
				return source.Get(ctx), nil
			},
		}
	})

	testCases := []struct {
		name         string
		sourceValue  bool
		inputs       map[string]any
		wantValue    bool
		wantReadonly bool
	}{
		{
			name:         "callbacks read the declared input",
			sourceValue:  true,
			inputs:       map[string]any{formID.ReferenceIDString(): false},
			wantValue:    true,
			wantReadonly: true,
		},
		{
			name:         "request value is used when the input makes the field editable",
			sourceValue:  false,
			inputs:       map[string]any{formID.ReferenceIDString(): true},
			wantValue:    true,
			wantReadonly: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, metadata, err := inspectiontest.Run(t, ctx, task, inspectioncore.TaskModeDryRun, tc.inputs,
				tasktest.Given(sourceTaskID.Ref(), tc.sourceValue),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantValue {
				t.Errorf("got %v, want %v", got, tc.wantValue)
			}
			fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
			if !found {
				t.Fatalf("form field set not found on metadata")
			}
			rawField := fields.DangerouslyGetField(formID.ReferenceIDString())
			checkboxField, ok := rawField.(inspectionmetadata.CheckboxParameterFormField)
			if !ok {
				t.Fatalf("field is not CheckboxParameterFormField: %T", rawField)
			}
			if checkboxField.Readonly != tc.wantReadonly {
				t.Errorf("field.Readonly = %v, want %v", checkboxField.Readonly, tc.wantReadonly)
			}
		})
	}

	t.Run("declares required input on source reference", func(t *testing.T) {
		wantInputs := []string{"required source"}
		if diff := cmp.Diff(wantInputs, describeInputs(task.Inputs())); diff != "" {
			t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("has form task label", func(t *testing.T) {
		isFormTask, found := typedmap.Get(task.Labels(), inspectioncore.TaskLabelKeyIsFormTask)
		if !found || !isFormTask {
			t.Errorf("task label %v = %v, want true", inspectioncore.TaskLabelKeyIsFormTask, isFormTask)
		}
	})
}
