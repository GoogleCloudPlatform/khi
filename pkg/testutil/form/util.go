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

package form_test

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type FormTestCase struct {
	Name              string
	Input             string
	ExpectedValue     any
	ExpectedFormField *form.FormField
	Dependencies      []task.Definition
	Before            func()
	After             func()
}

func TestTextForms(t *testing.T, label string, formVariable task.Definition, testCases []*FormTestCase, cmpOptions ...cmp.Option) {
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			if testCase.Before != nil {
				testCase.Before()
			}
			if testCase.Dependencies == nil {
				testCase.Dependencies = make([]task.Definition, 0)
			}
			availableSet, err := task.NewSet(testCase.Dependencies)
			if err != nil {
				t.Errorf("unexpected error\n%v", err)
			}
			formTaskSet, err := task.NewSet([]task.Definition{formVariable})
			if err != nil {
				t.Errorf("unexpected error\n%v", err)
			}
			resolved, err := formTaskSet.ResolveTask(availableSet)
			if err != nil {
				t.Errorf("unexpected error during resolving the task\n%v", err)
			}

			runner, err := task.NewLocalRunner(resolved)
			if err != nil {
				t.Errorf("unexpected error during creating an instance of LocalRunner\n%v", err)
			}
			md := metadata.NewSet()
			err = runner.Run(context.Background(), inspection_task.TaskModeDryRun, map[string]any{
				inspection_task.MetadataVariableName: md,
				inspection_task.InspectionRequestVariableName: &inspection_task.InspectionRequest{
					Values: map[string]any{
						formVariable.ID().ReferenceId().String(): testCase.Input,
					},
				},
			})
			if err != nil {
				t.Errorf("failed to start the task graph.\n%v", err)
			}
			<-runner.Wait()
			vs, err := runner.Result()
			if err != nil {
				t.Errorf("task graph was ended with a failure result.\n%v", err)
			}
			result, err := vs.Get(formVariable.ID().ReferenceId().String())
			if err != nil {
				t.Errorf("failed to get the form variable\n%v", err)
			}
			if diff := cmp.Diff(testCase.ExpectedValue, result, cmpOptions...); diff != "" {
				t.Errorf("the form task didn't generate the expected output\n%s", diff)
			}

			formFields := md.LoadOrStore(form.FormFieldSetMetadataKey, &form.FormFieldSetMetadataFactory{}).(*form.FormFieldSet)
			field := formFields.DangerouslyGetField(formVariable.ID().ReferenceId().String())
			if field.Type != "Text" {
				t.Errorf("the generated form has type %s and it's not Text", field.Type)
			}
			if field.Id == "" {
				t.Errorf("the generated form had the empty Id")
			}
			if diff := cmp.Diff(testCase.ExpectedFormField, field, cmpopts.IgnoreFields(form.FormField{}, "Priority", "Id", "Type")); diff != "" {
				t.Errorf("the form task didn't generate the expected form field metadata\n%s", diff)
			}
			if testCase.After != nil {
				testCase.After()
			}
		})
	}
}
