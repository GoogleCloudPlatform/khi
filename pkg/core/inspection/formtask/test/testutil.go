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

package form_task_test

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TextFormTestCase is the type to represent a test case of an inspection task to generate a text field.
type TextFormTestCase struct {
	Name              string
	Input             string
	ExpectedValue     any
	ExpectedFormField inspectionmetadata.TextParameterFormField
	Dependencies      []coretask.UntypedTask
	Before            func()
	After             func()
}

// TestTextForms tests an inspection task generating a TextForm in the metadata.
func TestTextForms[T any](t *testing.T, label string, formTask coretask.Task[T], testCases []*TextFormTestCase, cmpOptions ...cmp.Option) {
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			if testCase.Before != nil {
				testCase.Before()
			}
			if testCase.After != nil {
				defer testCase.After()
			}

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			_, metadata, err := inspectiontest.RunInspectionTaskWithDependency(ctx, formTask, testCase.Dependencies, inspectioncore_contract.TaskModeDryRun, map[string]any{
				formTask.ID().ReferenceIDString(): testCase.Input,
			})

			if err != nil {
				t.Errorf("form field task returned an error %v", err)
			}

			formFields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
			if !found {
				t.Fatalf("form field metadata not found!")
			}
			field := formFields.DangerouslyGetField(formTask.UntypedID().GetUntypedReference().String())
			textField, convertible := field.(inspectionmetadata.TextParameterFormField)
			if !convertible {
				t.Fatal("the generated form is not a TextParameterFormField")
			}
			if textField.ParameterFormFieldBase.Type != "text" {
				t.Errorf("the generated form has type %s and it's not text", textField.ParameterFormFieldBase.Type)
			}
			if textField.ParameterFormFieldBase.ID == "" {
				t.Errorf("the generated form had the empty Id")
			}
			if diff := cmp.Diff(testCase.ExpectedFormField, field, cmpopts.IgnoreFields(inspectionmetadata.ParameterFormFieldBase{}, "Priority", "ID", "Type")); diff != "" {
				t.Errorf("the form task didn't generate the expected form field metadata\n%s", diff)
			}
		})
	}
}
