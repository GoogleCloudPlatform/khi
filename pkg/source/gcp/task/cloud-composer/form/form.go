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

package composer_form

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/form"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	composer_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var AutocompleteComposerEnvironmentNames = task.NewCachedProcessor(composer_taskid.AutocompleteComposerEnvironmentNamesTaskID, []taskid.UntypedTaskReference{
	gcp_task.InputLocationsTaskID,
	gcp_task.InputProjectIdTaskID,
}, func(ctx context.Context, taskMode int, v *task.VariableSet) ([]string, error) {
	client, err := api.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return nil, err
	}
	projectId, err := gcp_task.GetInputProjectIdFromTaskVariable(v)
	if err != nil {
		return nil, err
	}
	location, err := gcp_task.GetInputLocationsFromTaskVariable(v)
	if err != nil {
		return nil, err
	}

	if projectId != "" && location != "" {
		clusterNames, err := client.GetComposerEnvironmentNames(ctx, projectId, location)
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Failed to read the composer environments in the (project,location) (%s, %s) \n%s", projectId, location, err))
			return []string{}, nil
		}
		return clusterNames, nil
	}
	return []string{}, nil
})

func GetAutocompleteComposerEnvironmentNamesTaskVariable(v *task.VariableSet) ([]string, error) {
	return task.GetTypedVariableFromTaskVariable[[]string](v, composer_taskid.AutocompleteComposerEnvironmentNamesTaskID.ReferenceIDString(), nil)
}

var InputComposerEnvironmentNameTask = form.NewInputFormDefinitionBuilder(composer_taskid.InputComposerEnvironmentTaskID, gcp_task.PriorityForResourceIdentifierGroup+5000, "Composer Environment Name").WithDependencies(
	[]taskid.UntypedTaskReference{composer_taskid.AutocompleteComposerEnvironmentNamesTaskID},
).WithSuggestionsFunc(func(ctx context.Context, value string, variables *task.VariableSet, previousValues []string) ([]string, error) {
	environments, err := GetAutocompleteComposerEnvironmentNamesTaskVariable(variables)
	if err != nil {
		return nil, err
	}
	return common.SortForAutocomplete(value, environments), nil
}).Build()

func GetInputComposerEnvironmentVariable(tv *task.VariableSet) (string, error) {
	return task.GetTypedVariableFromTaskVariable[string](tv, InputComposerEnvironmentNameTask.ID().ReferenceIDString(), "<INVALID>")
}
