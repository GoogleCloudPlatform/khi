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

package composercluster_impl

import (
	"context"
	"slices"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

// inputComposerEnvironmentNameTask inputs the composer environment name.
var inputComposerEnvironmentNameTask = formtask.DefineTextForm(
	composercluster.InputComposerEnvironmentNameTaskID,
	gcpcommon.PriorityForResourceIdentifierGroup+4400,
	"Composer Environment Name",
	"",
	func(b *coretask.Binder) formtask.TextFormSpec[string] {
		environmentsInput := coretask.Use(b, composercluster.AutocompleteComposerEnvironmentIdentityTaskID.Ref())
		return formtask.TextFormSpec[string]{
			DefaultValue: func(ctx context.Context, previousValues []string) (string, error) {
				environments := environmentsInput.Get(ctx)
				if len(previousValues) > 0 && slices.ContainsFunc(environments.Values, func(env composercluster.ComposerEnvironmentIdentity) bool {
					return env.EnvironmentName == previousValues[0]
				}) {
					return previousValues[0], nil
				}
				if len(environments.Values) == 0 {
					return "", nil
				}
				return environments.Values[0].EnvironmentName, nil
			},
			Suggestions: func(ctx context.Context, value string, previousValues []string) ([]string, error) {
				environments := environmentsInput.Get(ctx)
				if environments.Error != "" {
					return []string{}, nil
				}
				environmentNames := make([]string, len(environments.Values))
				for i, env := range environments.Values {
					environmentNames[i] = env.EnvironmentName
				}
				return common.SortForAutocomplete(value, environmentNames), nil
			},
		}
	},
)
