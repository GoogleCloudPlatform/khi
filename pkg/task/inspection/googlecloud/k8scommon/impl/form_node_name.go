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

package k8scommon_impl

import (
	"context"
	"fmt"
	"regexp"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

const maxNodeNameFilterOptions = 500

var nodeNameSubstringValidator = regexp.MustCompile("^[-a-z0-9]*$")

// inputNodeNameFilterTask is a task to collect list of substrings of node names. This input value is used in querying k8s_node or serialport logs.
var inputNodeNameFilterTask = formtask.DefineSetForm(
	k8scommon.InputNodeNameFilterTaskID,
	gcpcommon.PriorityForK8sResourceFilterGroup+3000,
	"Node names",
	"A space-separated list of node name substrings used to collect node-related logs. If left blank, KHI gathers logs from all nodes in the cluster.",
	func(b *coretask.Binder) formtask.SetFormSpec[[]string] {
		nodeNames := coretask.Use(b, k8scommon.AutocompleteNodeNamesTaskID.Ref())
		return formtask.SetFormSpec[[]string]{
			DefaultValue:     formtask.PreviousOrConstantDefaultValue([]string{}),
			AllowCustomValue: formtask.ConstantBool(true),
			DisableAddAll:    formtask.ConstantBool(true),
			DisableRemoveAll: formtask.ConstantBool(true),
			Validator:        validateNodeNameFilter,
			Options: func(ctx context.Context, prevValue []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
				result := []inspectionmetadata.SetParameterFormFieldOptionItem{}
				names := nodeNames.Get(ctx)
				for i, v := range names.Values {
					if i >= maxNodeNameFilterOptions {
						break
					}
					result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{ID: v})
				}
				return result, nil
			},
			Hint: func(ctx context.Context, value []string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
				names := nodeNames.Get(ctx)
				if len(names.Values) > maxNodeNameFilterOptions {
					return fmt.Sprintf("Some node names are not shown on the suggestion list because the number of node names is %d, which is more than %d.", len(names.Values), maxNodeNameFilterOptions), inspectionmetadata.Warning, nil
				}
				return "", inspectionmetadata.None, nil
			},
		}
	},
)

// validateNodeNameFilter validates that each node name substring contains only allowed characters.
func validateNodeNameFilter(ctx context.Context, value []string) (string, error) {
	for _, v := range value {
		if !nodeNameSubstringValidator.MatchString(v) {
			return fmt.Sprintf("invalid node name substring: %s", v), nil
		}
	}
	return "", nil
}
