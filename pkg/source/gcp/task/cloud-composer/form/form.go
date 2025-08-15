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

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	composer_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/taskid"
)

var AutocompleteComposerEnvironmentNames = inspectiontaskbase.NewCachedTask(composer_taskid.AutocompleteComposerEnvironmentNamesTaskID, []taskid.UntypedTaskReference{
	gcp_task.InputLocationsTaskID.Ref(),
	gcp_task.InputProjectIdTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.PreviousTaskResult[[]string]) (inspectiontaskbase.PreviousTaskResult[[]string], error) {
	client, err := api.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return inspectiontaskbase.PreviousTaskResult[[]string]{}, err
	}
	projectID := coretask.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.Ref())
	location := coretask.GetTaskResult(ctx, gcp_task.InputLocationsTaskID.Ref())
	dependencyDigest := fmt.Sprintf("%s-%s", projectID, location)

	if prevValue.DependencyDigest == dependencyDigest {
		return prevValue, nil
	}

	if projectID != "" && location != "" {
		clusterNames, err := client.GetComposerEnvironmentNames(ctx, projectID, location)
		if err != nil {
			// Failed to read the composer environments in the (project,location)
			return inspectiontaskbase.PreviousTaskResult[[]string]{
				DependencyDigest: dependencyDigest,
				Value:            []string{},
			}, nil
		}
		return inspectiontaskbase.PreviousTaskResult[[]string]{
			DependencyDigest: dependencyDigest,
			Value:            clusterNames,
		}, nil
	}
	return inspectiontaskbase.PreviousTaskResult[[]string]{
		DependencyDigest: dependencyDigest,
		Value:            []string{},
	}, nil
})

var InputComposerEnvironmentNameTask = formtask.NewTextFormTaskBuilder(composer_taskid.InputComposerEnvironmentTaskID, gcp_task.PriorityForResourceIdentifierGroup+4400, "Composer Environment Name").WithDependencies(
	[]taskid.UntypedTaskReference{composer_taskid.AutocompleteComposerEnvironmentNamesTaskID.Ref()},
).WithSuggestionsFunc(func(ctx context.Context, value string, previousValues []string) ([]string, error) {
	environments := coretask.GetTaskResult(ctx, composer_taskid.AutocompleteComposerEnvironmentNamesTaskID.Ref())
	return common.SortForAutocomplete(value, environments), nil
}).Build()
