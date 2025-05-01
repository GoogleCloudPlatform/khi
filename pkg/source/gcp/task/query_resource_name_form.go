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
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	form_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	gcp_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/taskid"
	gcp_types "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/types"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var resourceNamesInputKey = typedmap.NewTypedKey[*gcp_types.ResourceNamesInput]("query-resource-names")

var QueryResourceNameInputTask = inspection_task.NewInspectionTask(gcp_taskid.LoggingFilterResourceNameInputTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode) (*gcp_types.ResourceNamesInput, error) {
	sharedMap := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionSharedMap)
	resourceNamesInput := typedmap.GetOrSetFunc(sharedMap, resourceNamesInputKey, func() *gcp_types.ResourceNamesInput {
		return gcp_types.NewResourceNamesInput()
	})

	metadata := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionRunMetadata)
	formFields, found := typedmap.Get(metadata, form_metadata.FormFieldSetMetadataKey)
	if !found {
		return nil, fmt.Errorf("failed to get form fields from run metadata")
	}

	requestInput := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionTaskInput)

	queryForms := []form_metadata.ParameterFormField{}
	for _, form := range resourceNamesInput.GetQueryResourceNamePairs() {
		defaultValue := strings.Join(form.DefaultResourceNames, " ")
		formFieldBase := form_metadata.ParameterFormFieldBase{
			Priority:    0,
			ID:          form.GetInputID(),
			Type:        form_metadata.Text,
			Label:       form.QueryID,
			Description: "",
			HintType:    form_metadata.None,
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
					formFieldBase.HintType = form_metadata.Error
					formFieldBase.Hint = fmt.Sprintf("%d: %s", i, err.Error())
					break
				}
			}
		}
		queryForms = append(queryForms, &form_metadata.TextParameterFormField{
			ParameterFormFieldBase: formFieldBase,
			Default:                defaultValue,
		})
	}

	groupForm := form_metadata.GroupParameterFormField{
		ParameterFormFieldBase: form_metadata.ParameterFormFieldBase{
			Priority:    -1000000,
			ID:          gcp_taskid.LoggingFilterResourceNameInputTaskID.ReferenceIDString(),
			Type:        form_metadata.Group,
			Label:       "Logging filter resource names (advanced)",
			Description: "Override these parameters when your logs are not on the same project of the cluster, or customize the log filter target resources.",
			HintType:    form_metadata.None,
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
