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
	inspection_cached_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/cached_task"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/form"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	composer_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var AutocompleteComposerEnvironmentNames = inspection_cached_task.NewCachedTask(composer_taskid.AutocompleteComposerEnvironmentNamesTaskID, []taskid.UntypedTaskReference{
	gcp_task.InputLocationsTaskID,
	gcp_task.InputProjectIdTaskID,
}, func(ctx context.Context, prevValue inspection_cached_task.PreviousTaskResult[[]string]) (inspection_cached_task.PreviousTaskResult[[]string], error) {
	client, err := api.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return inspection_cached_task.PreviousTaskResult[[]string]{}, err
	}
	projectID := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.GetTaskReference())
	location := task.GetTaskResult(ctx, gcp_task.InputLocationsTaskID.GetTaskReference())
	dependencyDigest := fmt.Sprintf("%s-%s", projectID, location)

	if prevValue.DependencyDigest == dependencyDigest {
		return prevValue, nil
	}

	if projectID != "" && location != "" {
		clusterNames, err := client.GetComposerEnvironmentNames(ctx, projectID, location)
		if err != nil {
			// Failed to read the composer environments in the (project,location)
			return inspection_cached_task.PreviousTaskResult[[]string]{
				DependencyDigest: dependencyDigest,
				Value:            []string{},
			}, nil
		}
		return inspection_cached_task.PreviousTaskResult[[]string]{
			DependencyDigest: dependencyDigest,
			Value:            clusterNames,
		}, nil
	}
	return inspection_cached_task.PreviousTaskResult[[]string]{
		DependencyDigest: dependencyDigest,
		Value:            []string{},
	}, nil
})

var InputComposerEnvironmentNameTask = form.NewTextFormTaskBuilder(composer_taskid.InputComposerEnvironmentTaskID, gcp_task.PriorityForResourceIdentifierGroup+4400, "Composer Environment Name").WithDependencies(
	[]taskid.UntypedTaskReference{composer_taskid.AutocompleteComposerEnvironmentNamesTaskID},
).WithSuggestionsFunc(func(ctx context.Context, value string, previousValues []string) ([]string, error) {
	environments := task.GetTaskResult(ctx, composer_taskid.AutocompleteComposerEnvironmentNamesTaskID.GetTaskReference())
	return common.SortForAutocomplete(value, environments), nil
}).Build()
