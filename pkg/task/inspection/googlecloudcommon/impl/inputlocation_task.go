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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

// InputLocationsTask defines a form task for inputting the resource location.
var InputLocationsTask = formtask.NewTextFormTaskBuilder(googlecloudcommon_contract.InputLocationsTaskID, googlecloudcommon_contract.PriorityForResourceIdentifierGroup+4500, "Location").
	WithDependencies([]taskid.UntypedTaskReference{googlecloudcommon_contract.AutocompleteLocationTaskID.Ref()}).
	WithDescription(
		"The location(region) to specify the resource exist(s|ed)",
	).
	WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		if len(previousValues) > 0 {
			return previousValues[0], nil
		}
		return "", nil
	}).
	WithSuggestionsFunc(func(ctx context.Context, value string, previousValues []string) ([]string, error) {
		if len(previousValues) > 0 { // no need to call twice; should be the same
			return previousValues, nil
		}
		regions := coretask.GetTaskResult(ctx, googlecloudcommon_contract.AutocompleteLocationTaskID.Ref())
		return regions, nil
	}).
	Build()
