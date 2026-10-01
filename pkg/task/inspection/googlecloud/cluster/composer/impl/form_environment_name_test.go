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

package composercluster_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	form_task_test "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask/test"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func TestInputComposerEnvironmentNameTask(t *testing.T) {
	autocompleteEnvironments := &inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]{
		Values: []composercluster.ComposerEnvironmentIdentity{
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
	}

	autocompleteEmptyEnvironments := &inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]{
		Values: []composercluster.ComposerEnvironmentIdentity{},
	}

	autocompleteErrorEnvironments := &inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]{
		Values: []composercluster.ComposerEnvironmentIdentity{},
		Error:  "failed to list environments",
	}

	form_task_test.TestTextForms(t, "composer environment name", inputComposerEnvironmentNameTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "with environment suggestions available",
			Input:         "composer-env-1",
			ExpectedValue: "composer-env-1",
			TaskInputs:    []tasktest.InputValue{tasktest.Given(composercluster.AutocompleteComposerEnvironmentIdentityTaskID.Ref(), autocompleteEnvironments)},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       composercluster.GoogleCloudComposerTaskIDPrefix + "input/composer/environment_name",
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
			TaskInputs:    []tasktest.InputValue{tasktest.Given(composercluster.AutocompleteComposerEnvironmentIdentityTaskID.Ref(), autocompleteEmptyEnvironments)},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       composercluster.GoogleCloudComposerTaskIDPrefix + "input/composer/environment_name",
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
			TaskInputs:    []tasktest.InputValue{tasktest.Given(composercluster.AutocompleteComposerEnvironmentIdentityTaskID.Ref(), autocompleteErrorEnvironments)},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       composercluster.GoogleCloudComposerTaskIDPrefix + "input/composer/environment_name",
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
		secondRunEnvironments []composercluster.ComposerEnvironmentIdentity
		wantDefault           string
	}{
		{
			name:          "previous value retained when present in suggestions",
			previousValue: "composer-env-2",
			secondRunEnvironments: []composercluster.ComposerEnvironmentIdentity{
				{EnvironmentName: "composer-env-1"},
				{EnvironmentName: "composer-env-2"},
			},
			wantDefault: "composer-env-2",
		},
		{
			name:          "fallback to first suggestion when previous value not in suggestions",
			previousValue: "old-env",
			secondRunEnvironments: []composercluster.ComposerEnvironmentIdentity{
				{EnvironmentName: "composer-env-1"},
				{EnvironmentName: "composer-env-2"},
			},
			wantDefault: "composer-env-1",
		},
		{
			name:                  "empty default when no suggestions available",
			previousValue:         "old-env",
			secondRunEnvironments: []composercluster.ComposerEnvironmentIdentity{},
			wantDefault:           "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx1 := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

			_, _, err := inspectiontest.Run(
				t,
				ctx1,
				inputComposerEnvironmentNameTask,
				inspectioncore.TaskModeRun,
				map[string]any{
					inputComposerEnvironmentNameTask.ID().ReferenceIDString(): tc.previousValue,
				},
				tasktest.Given(
					composercluster.AutocompleteComposerEnvironmentIdentityTaskID.Ref(),
					&inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]{
						Values: []composercluster.ComposerEnvironmentIdentity{
							{EnvironmentName: tc.previousValue},
						},
					},
				),
			)
			if err != nil {
				t.Fatalf("first run failed: %v", err)
			}

			ctx2 := inspectiontest.NextRunTaskContext(t.Context(), ctx1)

			_, metadata, err := inspectiontest.Run(
				t,
				ctx2,
				inputComposerEnvironmentNameTask,
				inspectioncore.TaskModeDryRun,
				map[string]any{},
				tasktest.Given(
					composercluster.AutocompleteComposerEnvironmentIdentityTaskID.Ref(),
					&inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]{
						Values: tc.secondRunEnvironments,
					},
				),
			)
			if err != nil {
				t.Fatalf("second run failed: %v", err)
			}

			formFields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
			if !found {
				t.Fatalf("form field metadata not found")
			}
			field := formFields.DangerouslyGetField(inputComposerEnvironmentNameTask.UntypedID().GetUntypedReference().String())
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
