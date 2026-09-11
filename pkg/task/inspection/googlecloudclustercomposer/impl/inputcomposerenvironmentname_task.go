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
	"slices"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

// InputComposerEnvironmentNameTask is the task that inputs composer environment name.
var InputComposerEnvironmentNameTask = formtask.NewTextFormTaskBuilder(googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID, googlecloudcommon_contract.PriorityForResourceIdentifierGroup+4400, "Composer Environment Name").WithDependencies(
	[]taskid.UntypedTaskReference{googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID.Ref()},
).WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
	environments := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID.Ref())
	if len(previousValues) > 0 && slices.ContainsFunc(environments.Values, func(env googlecloudclustercomposer_contract.ComposerEnvironmentIdentity) bool {
		return env.EnvironmentName == previousValues[0]
	}) {
		return previousValues[0], nil
	}
	if len(environments.Values) == 0 {
		return "", nil
	}
	return environments.Values[0].EnvironmentName, nil
}).WithSuggestionsFunc(func(ctx context.Context, value string, previousValues []string) ([]string, error) {
	environments := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID.Ref())
	if environments.Error != "" {
		return []string{}, nil
	}
	environmentNames := make([]string, len(environments.Values))
	for i, env := range environments.Values {
		environmentNames[i] = env.EnvironmentName
	}
	return common.SortForAutocomplete(value, environmentNames), nil
}).Build()
