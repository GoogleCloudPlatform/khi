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

package googlecloudcommon_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestInputLoggingFilterResourceNameTask(t *testing.T) {
	defaultNames := []string{"projects/foo"}
	testCases := []struct {
		desc       string
		taskMode   inspectioncore_contract.InspectionTaskModeType
		inputValue string
		wantForm   inspectionmetadata.GroupParameterFormField
	}{
		{
			desc:       "basic input",
			taskMode:   inspectioncore_contract.TaskModeDryRun,
			inputValue: "projects/foo",
			wantForm: inspectionmetadata.GroupParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Priority:    -1000000,
					ID:          googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID.ReferenceIDString(),
					Type:        inspectionmetadata.Group,
					Label:       "Logging filter resource names (advanced)",
					Description: "Override these parameters when your logs are not on the same project of the cluster, or customize the log filter target resources.",
					HintType:    inspectionmetadata.None,
					Hint:        "",
				},
				Children: []inspectionmetadata.ParameterFormField{
					&inspectionmetadata.TextParameterFormField{
						ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
							ID:       "cloud.google.com/common/input-query-resource-names/test",
							Type:     "text",
							Label:    "test",
							HintType: "none",
						},
						Default:          "projects/foo",
						Suggestions:      []string{"projects/foo"},
						ValidationTiming: "change",
					},
				},
				Collapsible:        true,
				CollapsedByDefault: true,
			},
		},
		{
			desc:       "invalid input",
			taskMode:   inspectioncore_contract.TaskModeDryRun,
			inputValue: "invalid-resource-name",
			wantForm: inspectionmetadata.GroupParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Priority:    -1000000,
					ID:          googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID.ReferenceIDString(),
					Type:        inspectionmetadata.Group,
					Label:       "Logging filter resource names (advanced)",
					Description: "Override these parameters when your logs are not on the same project of the cluster, or customize the log filter target resources.",
					HintType:    inspectionmetadata.None,
					Hint:        "",
				},
				Children: []inspectionmetadata.ParameterFormField{
					&inspectionmetadata.TextParameterFormField{
						ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
							ID:       "cloud.google.com/common/input-query-resource-names/test",
							Type:     "text",
							Label:    "test",
							HintType: "error",
							Hint:     "0: resource name must begin with one of the following: [projects organizations folders billingAccounts]",
						},
						Default:          "projects/foo",
						Suggestions:      []string{"projects/foo"},
						ValidationTiming: "change",
					},
				},
				Collapsible:        true,
				CollapsedByDefault: true,
			},
		},
		{
			desc:       "basic input for run mode",
			taskMode:   inspectioncore_contract.TaskModeRun,
			inputValue: "projects/foo",
			wantForm: inspectionmetadata.GroupParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Priority:    -1000000,
					ID:          googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID.ReferenceIDString(),
					Type:        inspectionmetadata.Group,
					Label:       "Logging filter resource names (advanced)",
					Description: "Override these parameters when your logs are not on the same project of the cluster, or customize the log filter target resources.",
					HintType:    inspectionmetadata.None,
					Hint:        "",
				},
				Children: []inspectionmetadata.ParameterFormField{
					&inspectionmetadata.TextParameterFormField{
						ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
							ID:       "cloud.google.com/common/input-query-resource-names/test",
							Type:     "text",
							Label:    "test",
							HintType: "none",
						},
						Default:          "projects/foo",
						Suggestions:      []string{"projects/foo"},
						ValidationTiming: "change",
					},
				},
				Collapsible:        true,
				CollapsedByDefault: true,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			resourceNames, _, err := inspectiontest.RunInspectionTask(ctx, InputLoggingFilterResourceNameTask, inspectioncore_contract.TaskModeDryRun, map[string]any{})
			if err != nil {
				t.Fatalf("Failed to call InputLoggingFilterResourceNameTask at 1st time:%v", err)
			}
			resourceName := googlecloudcommon_contract.QueryResourceNames{
				QueryID: "test",
			}
			newCtx := inspectiontest.NextRunTaskContext(t.Context(), ctx)
			resourceNames.UpdateDefaultResourceNamesForQuery("test", defaultNames)
			_, metadata, err := inspectiontest.RunInspectionTask(newCtx, InputLoggingFilterResourceNameTask, tc.taskMode, map[string]any{
				resourceName.GetInputID(): tc.inputValue,
			})
			if err != nil {
				t.Fatalf("Failed to call InputLoggingFilterResourceNameTask at 2nd time:%v", err)
			}
			formFieldSet, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
			if !found {
				t.Fatalf("formFieldSet is not set in the metadata")
			}
			gotForm := formFieldSet.DangerouslyGetField(googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID.ReferenceIDString())
			if diff := cmp.Diff(tc.wantForm, gotForm); diff != "" {
				t.Errorf("InputLoggingFilterResourceNameTask saved group form mismatch (-want,+got):\n%s", diff)
			}
		})
	}
}
