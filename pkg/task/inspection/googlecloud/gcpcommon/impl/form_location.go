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
	"slices"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

// InputLocationsTask defines a form task for inputting the resource location.
var InputLocationsTask = formtask.NewTextFormTaskBuilder(gcpcommon.InputLocationsTaskID, gcpcommon.PriorityForResourceIdentifierGroup+3000, "Location").
	WithDependencies([]coretask.Dependency{gcpcommon.AutocompleteLocationTaskID.Ref()}).
	WithDescription(
		"The location(region) to specify the resource exist(s|ed)",
	).
	WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		locations := coretask.GetTaskResult(ctx, gcpcommon.AutocompleteLocationTaskID.Ref())
		if len(previousValues) > 0 && slices.Contains(locations.Values, previousValues[0]) {
			return previousValues[0], nil
		}
		if len(locations.Values) == 0 {
			return "", nil
		}
		return locations.Values[0], nil
	}).
	WithSuggestionsFunc(func(ctx context.Context, value string, previousValues []string) ([]string, error) {
		regions := coretask.GetTaskResult(ctx, gcpcommon.AutocompleteLocationTaskID.Ref())
		return common.SortForAutocomplete(value, regions.Values), nil
	}).
	WithValidator(func(ctx context.Context, value string) (string, error) {
		if value == "" {
			return "location is required", nil
		}
		return "", nil
	}).
	Build()
