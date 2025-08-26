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
	"fmt"

	googlecloudapi "github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

// AutocompleteComposerEnvironmentNamesTask is the task that autocompletes composer environment names.
var AutocompleteComposerEnvironmentNamesTask = inspectiontaskbase.NewCachedTask(googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentNamesTaskID, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputLocationsTaskID.Ref(),
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.PreviousTaskResult[[]string]) (inspectiontaskbase.PreviousTaskResult[[]string], error) {
	client, err := googlecloudapi.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return inspectiontaskbase.PreviousTaskResult[[]string]{}, err
	}
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	location := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputLocationsTaskID.Ref())
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
