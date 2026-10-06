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

package composerairflow_impl

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

// inputComposerComponentsTask allows the user to select which Composer V3 components to fetch logs from.
var inputComposerComponentsTask = formtask.DefineSetForm(
	composerairflow.InputComposerComponentsTaskID,
	gcpcommon.FormBasePriority+3000,
	"Composer Components",
	"Select which Composer V3 components to fetch logs from.",
	func(b *coretask.Binder) formtask.SetFormSpec[[]string] {
		components := coretask.Use(b, composerairflow.AutocompleteComposerComponentsTaskID.Ref())
		return formtask.SetFormSpec[[]string]{
			DefaultValue:     formtask.PreviousOrConstantDefaultValue([]string{"@any"}),
			DisableAddAll:    formtask.ConstantBool(true),
			DisableRemoveAll: formtask.ConstantBool(true),
			AllowCustomValue: formtask.ConstantBool(false),
			Options: func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
				autocompleteResult := components.Get(ctx)

				var options []inspectionmetadata.SetParameterFormFieldOptionItem
				options = append(options, inspectionmetadata.SetParameterFormFieldOptionItem{
					ID: "@any",
				})
				if autocompleteResult != nil {
					for _, comp := range autocompleteResult.Values {
						options = append(options, inspectionmetadata.SetParameterFormFieldOptionItem{
							ID: comp,
						})
					}
				}
				return options, nil
			},
			Hint: func(ctx context.Context, value []string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
				autocompleteResult := components.Get(ctx)
				if autocompleteResult != nil {
					if autocompleteResult.Error != "" {
						return autocompleteResult.Error, inspectionmetadata.Error, nil
					}
					if autocompleteResult.Hint != "" {
						return autocompleteResult.Hint, inspectionmetadata.Info, nil
					}
				}
				return "", inspectionmetadata.None, nil
			},
			Converter: func(ctx context.Context, value []string) ([]string, error) {
				return value, nil
			},
		}
	},
)
