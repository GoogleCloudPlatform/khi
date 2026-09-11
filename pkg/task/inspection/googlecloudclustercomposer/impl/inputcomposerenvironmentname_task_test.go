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

package googlecloudclustercomposer_impl

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	form_task_test "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask/test"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func TestInputComposerEnvironmentNameTask(t *testing.T) {
	mockAutocompleteEnvironments := coretask.NewTask(googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID, []coretask.Dependency{}, func(ctx context.Context) (*inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity], error) {
		return &inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
			Values: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{
				{
					EnvironmentName: "composer-env-1",
					Location:        "us-central1",
					ProjectID:       "sample-project",
				},
				{
					EnvironmentName: "composer-env-2",
					Location:        "asia-northeast1",
					ProjectID:       "sample-project",
				},
			},
		}, nil
	})

	mockAutocompleteEmptyEnvironments := coretask.NewTask(googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID, []coretask.Dependency{}, func(ctx context.Context) (*inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity], error) {
		return &inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
			Values: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{},
		}, nil
	})

	mockAutocompleteErrorEnvironments := coretask.NewTask(googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID, []coretask.Dependency{}, func(ctx context.Context) (*inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity], error) {
		return &inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
			Values: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{},
			Error:  "failed to list environments",
		}, nil
	})

	form_task_test.TestTextForms(t, "composer environment name", InputComposerEnvironmentNameTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "with environment suggestions available",
			Input:         "composer-env-1",
			ExpectedValue: "composer-env-1",
			Dependencies:  []coretask.UntypedTask{mockAutocompleteEnvironments},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       googlecloudclustercomposer_contract.GoogleCloudComposerTaskIDPrefix + "input/composer/environment_name",
					Type:     "Text",
					Label:    "Composer Environment Name",
					HintType: inspectionmetadata.None,
				},
				Suggestions:      []string{"composer-env-1", "composer-env-2"},
				Default:          "composer-env-1",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "without environment suggestions",
			Input:         "",
			ExpectedValue: "",
			Dependencies:  []coretask.UntypedTask{mockAutocompleteEmptyEnvironments},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       googlecloudclustercomposer_contract.GoogleCloudComposerTaskIDPrefix + "input/composer/environment_name",
					Type:     "Text",
					Label:    "Composer Environment Name",
					HintType: inspectionmetadata.None,
				},
				Suggestions:      []string{},
				Default:          "",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
		{
			Name:          "when autocomplete returns error",
			Input:         "",
			ExpectedValue: "",
			Dependencies:  []coretask.UntypedTask{mockAutocompleteErrorEnvironments},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       googlecloudclustercomposer_contract.GoogleCloudComposerTaskIDPrefix + "input/composer/environment_name",
					Type:     "Text",
					Label:    "Composer Environment Name",
					HintType: inspectionmetadata.None,
				},
				Suggestions:      []string{},
				Default:          "",
				ValidationTiming: inspectionmetadata.Change,
			},
		},
	})
}

func TestInputComposerEnvironmentNameTask_PreviousValues(t *testing.T) {
	testCases := []struct {
		name                  string
		previousValue         string
		secondRunEnvironments []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity
		wantDefault           string
	}{
		{
			name:          "previous value retained when present in suggestions",
			previousValue: "composer-env-2",
			secondRunEnvironments: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{
				{EnvironmentName: "composer-env-1"},
				{EnvironmentName: "composer-env-2"},
			},
			wantDefault: "composer-env-2",
		},
		{
			name:          "fallback to first suggestion when previous value not in suggestions",
			previousValue: "old-env",
			secondRunEnvironments: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{
				{EnvironmentName: "composer-env-1"},
				{EnvironmentName: "composer-env-2"},
			},
			wantDefault: "composer-env-1",
		},
		{
			name:                  "empty default when no suggestions available",
			previousValue:         "old-env",
			secondRunEnvironments: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{},
			wantDefault:           "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx1 := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			firstMock := tasktest.StubTaskFromReferenceID(
				googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID.Ref(),
				&inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
					Values: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{
						{EnvironmentName: tc.previousValue},
					},
				},
				nil,
			)

			_, _, err := inspectiontest.RunInspectionTaskWithDependency(
				ctx1,
				InputComposerEnvironmentNameTask,
				[]coretask.UntypedTask{firstMock},
				inspectioncore_contract.TaskModeRun,
				map[string]any{
					InputComposerEnvironmentNameTask.ID().ReferenceIDString(): tc.previousValue,
				},
			)
			if err != nil {
				t.Fatalf("first run failed: %v", err)
			}

			ctx2 := inspectiontest.NextRunTaskContext(context.Background(), ctx1)
			secondMock := tasktest.StubTaskFromReferenceID(
				googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID.Ref(),
				&inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
					Values: tc.secondRunEnvironments,
				},
				nil,
			)

			_, metadata, err := inspectiontest.RunInspectionTaskWithDependency(
				ctx2,
				InputComposerEnvironmentNameTask,
				[]coretask.UntypedTask{secondMock},
				inspectioncore_contract.TaskModeDryRun,
				map[string]any{},
			)
			if err != nil {
				t.Fatalf("second run failed: %v", err)
			}

			formFields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
			if !found {
				t.Fatalf("form field metadata not found")
			}
			field := formFields.DangerouslyGetField(InputComposerEnvironmentNameTask.UntypedID().GetUntypedReference().String())
			textField, ok := field.(inspectionmetadata.TextParameterFormField)
			if !ok {
				t.Fatalf("field is not TextParameterFormField")
			}
			if textField.Default != tc.wantDefault {
				t.Errorf("default value mismatch: got %q, want %q", textField.Default, tc.wantDefault)
			}
		})
	}
}
