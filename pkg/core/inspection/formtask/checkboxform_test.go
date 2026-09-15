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
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type testCheckboxFormConfigurator = func(builder *CheckboxFormTaskBuilder)

func TestCheckboxFormDefinitionBuilder(t *testing.T) {
	testCases := []struct {
		name              string
		formConfigurator  testCheckboxFormConfigurator
		requestValue      any
		hasRequestValue   bool
		expectedFormField inspectionmetadata.ParameterFormField
		expectedValue     bool
		expectedError     string
	}{
		{
			name:             "checkbox form with given boolean parameter",
			formConfigurator: func(builder *CheckboxFormTaskBuilder) {},
			requestValue:     true,
			hasRequestValue:  true,
			expectedValue:    true,
			expectedFormField: inspectionmetadata.CheckboxParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly: false,
				Default:  false,
			},
		},
		{
			name:             "checkbox form with string true parameter",
			formConfigurator: func(builder *CheckboxFormTaskBuilder) {},
			requestValue:     "true",
			hasRequestValue:  true,
			expectedValue:    true,
			expectedFormField: inspectionmetadata.CheckboxParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					HintType: inspectionmetadata.None,
				},
				Readonly: false,
				Default:  false,
			},
		},
		{
			name:             "checkbox form with string false parameter",
			formConfigurator: func(builder *CheckboxFormTaskBuilder) {},
			requestValue:     "false",
			hasRequestValue:  true,
			expectedValue:    false,
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
			formConfigurator: func(builder *CheckboxFormTaskBuilder) {
				builder.WithDefaultValue(true)
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
			formConfigurator: func(builder *CheckboxFormTaskBuilder) {
				builder.WithValidator(func(ctx context.Context, value bool) (string, error) {
					if value {
						return "cannot be enabled", nil
					}
					return "", nil
				})
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
			formConfigurator: func(builder *CheckboxFormTaskBuilder) {
				builder.WithReadonly(true).WithDefaultValue(false)
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
			formConfigurator: func(builder *CheckboxFormTaskBuilder) {
				builder.WithHintFunc(func(ctx context.Context, value bool) (string, inspectionmetadata.ParameterHintType, error) {
					return "checkbox hint", inspectionmetadata.Info, nil
				})
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
			name:             "checkbox form with invalid string parameter",
			formConfigurator: func(builder *CheckboxFormTaskBuilder) {},
			requestValue:     "not-a-bool",
			hasRequestValue:  true,
			expectedError:    "request parameter `foo` was not a valid boolean in task foo#default",
		},
		{
			name:             "checkbox form with invalid parameter type",
			formConfigurator: func(builder *CheckboxFormTaskBuilder) {},
			requestValue:     123,
			hasRequestValue:  true,
			expectedError:    "request parameter `foo` was not given as boolean or boolean string in task foo#default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewCheckboxFormTaskBuilder(taskid.NewDefaultImplementationID[bool]("foo"), 1, "foo label")
			builder.WithDescription("foo description")
			tc.formConfigurator(builder)
			taskDef := builder.Build()

			inputs := map[string]any{}
			if tc.hasRequestValue {
				inputs["foo"] = tc.requestValue
			}

			// DryRun mode execution
			dryRunCtx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			_, _, dryRunErr := inspectiontest.RunInspectionTask(dryRunCtx, taskDef, inspectioncore.TaskModeDryRun, inputs)

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

			metadata := khictx.MustGetValue(dryRunCtx, inspectioncore.InspectionRunMetadata)
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
			runResult, _, runErr := inspectiontest.RunInspectionTask(runCtx, taskDef, inspectioncore.TaskModeRun, inputs)

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
