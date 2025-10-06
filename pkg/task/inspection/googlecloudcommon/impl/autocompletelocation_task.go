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

	googlecloudapi "github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

// AutocompleteLocationTask is a task that provides a list of available locations for autocomplete.
var AutocompleteLocationTask = inspectiontaskbase.NewCachedTask(googlecloudcommon_contract.AutocompleteLocationTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(), // for API restriction
	},
	func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[[]string]) (inspectiontaskbase.CacheableTaskResult[[]string], error) {
		client, err := googlecloudapi.DefaultGCPClientFactory.NewClient()
		if err != nil {
			return inspectiontaskbase.CacheableTaskResult[[]string]{}, err
		}
		projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
		dependencyDigest := fmt.Sprintf("location-%s", projectID)

		if prevValue.DependencyDigest == dependencyDigest {
			return prevValue, nil
		}

		defaultResult := inspectiontaskbase.CacheableTaskResult[[]string]{
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
