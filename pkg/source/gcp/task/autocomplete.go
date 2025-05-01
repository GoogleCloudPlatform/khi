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

package task

import (
	"context"
	"fmt"

	inspection_cached_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/cached_task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var AutocompleteClusterNamesTaskID = taskid.NewTaskReference[*AutocompleteClusterNameList](GCPPrefix + "autocomplete/cluster-names")

type AutocompleteClusterNameList struct {
	ClusterNames []string
	Error        string
}

var AutocompleteLocationTaskID taskid.TaskImplementationID[[]string] = taskid.NewDefaultImplementationID[[]string](GCPPrefix + "autocomplete/location")

// default implementation for "Location" field
var AutocompleteLocationTask = inspection_cached_task.NewCachedTask(AutocompleteLocationTaskID,
	[]taskid.UntypedTaskReference{
		InputProjectIdTaskID.Ref(), // for API restriction
	},
	func(ctx context.Context, prevValue inspection_cached_task.PreviousTaskResult[[]string]) (inspection_cached_task.PreviousTaskResult[[]string], error) {
		client, err := api.DefaultGCPClientFactory.NewClient()
		if err != nil {
			return inspection_cached_task.PreviousTaskResult[[]string]{}, err
		}
		projectID := task.GetTaskResult(ctx, InputProjectIdTaskID.Ref())
		dependencyDigest := fmt.Sprintf("location-%s", projectID)

		if prevValue.DependencyDigest == dependencyDigest {
			return prevValue, nil
		}

		defaultResult := inspection_cached_task.PreviousTaskResult[[]string]{
			DependencyDigest: dependencyDigest,
			Value:            []string{},
		}

		if projectID == "" {
			return defaultResult, nil
		}

		regions, err := client.ListRegions(ctx, projectID)
		if err != nil {
			return defaultResult, nil
		}
		result := defaultResult
		result.Value = regions
		return result, nil
	})
