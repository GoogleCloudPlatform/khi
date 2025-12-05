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
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var resourceNamesInputKey = typedmap.NewTypedKey[*googlecloudcommon_contract.ResourceNamesInput]("query-resource-names")

// InputLoggingFilterResourceNameTask defines an inspection task that creates a form group
// for overriding log filter resource names for advanced users.
var InputLoggingFilterResourceNameTask = inspectiontaskbase.NewInspectionTask(googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (*googlecloudcommon_contract.ResourceNamesInput, error) {
	// Tasks requiring active resource names can change, so we always retrieve current tasks that need resource names from the task graph.
	taskRunner := khictx.MustGetValue(ctx, inspectioncore_contract.TaskRunner)
	currentActiveResourceNameInputRequests := getCurrentActiveQueryIDsForResourceName(taskRunner)
	// Since the default resource names registered by the tasks actually used are not known until those tasks are executed, we store them in sharedMap and have the actual tasks update them.
	sharedMap := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionSharedMap)
	resourceNamesInput := typedmap.GetOrSetFunc(sharedMap, resourceNamesInputKey, googlecloudcommon_contract.NewResourceNamesInput)

	metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
	formFields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
	if !found {
		return nil, fmt.Errorf("failed to get form fields from run metadata")
	}

	requestInput := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInput)

	queryForms := []inspectionmetadata.ParameterFormField{}
	for _, request := range currentActiveResourceNameInputRequests {
		queryInfo := resourceNamesInput.GetResourceNamesForQuery(ctx, request)
		defaultValue := strings.Join(queryInfo.DefaultResourceNames, " ")
		formFieldBase := inspectionmetadata.ParameterFormFieldBase{
			Priority:    0,
			ID:          queryInfo.GetInputID(),
			Type:        inspectionmetadata.Text,
			Label:       queryInfo.QueryID,
			Description: "",
			HintType:    inspectionmetadata.None,
			Hint:        "",
		}
		// This task validates the inputs only.
		formInput, found := requestInput[queryInfo.GetInputID()]
		if found {
			resourceNamesFromInput := strings.Split(formInput.(string), " ")
			for i, resourceNameFromInput := range resourceNamesFromInput {
				resourceNameWithoutSurroundingSpace := strings.TrimSpace(resourceNameFromInput)
				err := googlecloud.ValidateResourceNameOnLogEntriesList(resourceNameWithoutSurroundingSpace)
				if err != nil {
					formFieldBase.HintType = inspectionmetadata.Error
					formFieldBase.Hint = fmt.Sprintf("%d: %s", i, err.Error())
					break
				}
			}
		}
		queryForms = append(queryForms, &inspectionmetadata.TextParameterFormField{
			ParameterFormFieldBase: formFieldBase,
			Default:                defaultValue,
			Suggestions:            queryInfo.DefaultResourceNames,
			ValidationTiming:       inspectionmetadata.Change,
		})
	}

	groupForm := inspectionmetadata.GroupParameterFormField{
		ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
			Priority:    -1000000,
			ID:          googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID.ReferenceIDString(),
			Type:        inspectionmetadata.Group,
			Label:       "Logging filter resource names (advanced)",
			Description: "Override these parameters when your logs are not on the same project of the cluster, or customize the log filter target resources.",
			HintType:    inspectionmetadata.None,
			Hint:        "",
		},
		Children:           queryForms,
		Collapsible:        true,
		CollapsedByDefault: true,
	}
	err := formFields.SetField(groupForm)
	if err != nil {
		return nil, err
	}

	return resourceNamesInput, nil
})

// getCurrentActiveQueryIDsForResourceName returns the query IDs that are currently active with retrieving them from the current task graph.
func getCurrentActiveQueryIDsForResourceName(runner coretask.TaskRunner) []string {
	tasks := runner.Tasks()
	resultMap := map[string]struct{}{}
	for _, t := range tasks {
		requestInput, found := typedmap.Get(t.Labels(), googlecloudcommon_contract.RequestOptionalInputResourceNameTaskLabel)
		if !found {
			continue
		}
		resultMap[requestInput] = struct{}{}
	}
	result := []string{}
	for k := range resultMap {
		result = append(result, k)
	}
	slices.Sort(result)
	return result
}
