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

package gcpcommon_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// AutocompleteLocationTask is a task that provides a list of available locations for autocomplete.
// This serves as the default fallback implementation for generic Google Cloud environments.
// When an inspection type specific implementation is available (e.g. Kubernetes cluster or Cloud Composer),
// this implementation is shadowed by the higher-priority task.
var AutocompleteLocationTask = inspectiontaskbase.NewGlobalCachedTask(gcpcommon.AutocompleteLocationTaskID,
	[]coretask.Dependency{
		gcpcommon.InputProjectIdTaskID.Ref(), // for API restriction
		gcpcommon.LocationFetcherTaskID.Ref(),
	},
	func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]], error) {
		projectID := coretask.GetTaskResult(ctx, gcpcommon.InputProjectIdTaskID.Ref())
		dependencyDigest := fmt.Sprintf("location-%s", projectID)

		if prevValue.DependencyDigest == dependencyDigest {
			return prevValue, nil
		}

		defaultResult := inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
			DependencyDigest: dependencyDigest,
			Value: &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "",
			},
		}

		if projectID == "" {
			return defaultResult, nil
		}

		locationFetcher := coretask.GetTaskResult(ctx, gcpcommon.LocationFetcherTaskID.Ref())
		regions, err := locationFetcher.FetchRegions(ctx, projectID)
		if err != nil {
			return defaultResult, nil
		}
		result := defaultResult
		result.Value.Values = regions
		return result, nil
	})
