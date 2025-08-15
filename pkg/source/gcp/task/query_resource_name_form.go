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

package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	gcp_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/taskid"
	gcp_types "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/types"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
)

var resourceNamesInputKey = typedmap.NewTypedKey[*gcp_types.ResourceNamesInput]("query-resource-names")

var QueryResourceNameInputTask = inspectiontaskbase.NewInspectionTask(gcp_taskid.LoggingFilterResourceNameInputTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspection_contract.InspectionTaskModeType) (*gcp_types.ResourceNamesInput, error) {
	sharedMap := khictx.MustGetValue(ctx, inspection_contract.InspectionSharedMap)
	resourceNamesInput := typedmap.GetOrSetFunc(sharedMap, resourceNamesInputKey, gcp_types.NewResourceNamesInput)

	metadata := khictx.MustGetValue(ctx, inspection_contract.InspectionRunMetadata)
	formFields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
	if !found {
		return nil, fmt.Errorf("failed to get form fields from run metadata")
	}

	requestInput := khictx.MustGetValue(ctx, inspection_contract.InspectionTaskInput)

	queryForms := []inspectionmetadata.ParameterFormField{}
	for _, form := range resourceNamesInput.GetQueryResourceNamePairs() {
		defaultValue := strings.Join(form.DefaultResourceNames, " ")
		formFieldBase := inspectionmetadata.ParameterFormFieldBase{
			Priority:    0,
			ID:          form.GetInputID(),
			Type:        inspectionmetadata.Text,
			Label:       form.QueryID,
			Description: "",
			HintType:    inspectionmetadata.None,
			Hint:        "",
		}
		// This task validates the inputs only.
		formInput, found := requestInput[form.GetInputID()]
		if found {
			resourceNamesFromInput := strings.Split(formInput.(string), " ")
			for i, resourceNameFromInput := range resourceNamesFromInput {
				resourceNameWithoutSurroundingSpace := strings.TrimSpace(resourceNameFromInput)
				err := api.ValidateResourceNameOnLogEntriesList(resourceNameWithoutSurroundingSpace)
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
		})
	}

	groupForm := inspectionmetadata.GroupParameterFormField{
		ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
			Priority:    -1000000,
			ID:          gcp_taskid.LoggingFilterResourceNameInputTaskID.ReferenceIDString(),
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
