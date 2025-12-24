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

package googlecloudk8scommon_impl

import (
	"context"
	"fmt"
	"regexp"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

const maxNodeNameFilterOptions = 500

var nodeNameSubstringValidator = regexp.MustCompile("^[-a-z0-9]*$")

// InputNodeNameFilterTask is a task to collect list of substrings of node names. This input value is used in querying k8s_node or serialport logs.
var InputNodeNameFilterTask = formtask.NewSetFormTaskBuilder(googlecloudk8scommon_contract.InputNodeNameFilterTaskID, googlecloudcommon_contract.PriorityForK8sResourceFilterGroup+3000, "Node names").
	WithDependencies([]taskid.UntypedTaskReference{googlecloudk8scommon_contract.AutocompleteNodeNamesTaskID.Ref()}).
	WithDefaultValueConstant([]string{}, true).
	WithDescription("A space-separated list of node name substrings used to collect node-related logs. If left blank, KHI gathers logs from all nodes in the cluster.").
	WithAllowAddAll(false).
	WithAllowRemoveAll(false).
	WithAllowCustomValue(true).
	WithValidator(func(ctx context.Context, value []string) (string, error) {
		for _, v := range value {
			if !nodeNameSubstringValidator.MatchString(v) {
				return fmt.Sprintf("invalid node name substring: %s", v), nil
			}
		}
		return "", nil
	}).
	WithOptionsFunc(func(ctx context.Context, prevValue []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
		result := []inspectionmetadata.SetParameterFormFieldOptionItem{}
		nodeNames := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteNodeNamesTaskID.Ref())
		for i, v := range nodeNames.Values {
			if i >= maxNodeNameFilterOptions {
				break
			}
			result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{ID: v})
		}
		return result, nil
	}).
	WithHintFunc(func(ctx context.Context, value []string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
		nodeNames := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteNodeNamesTaskID.Ref())
		if len(nodeNames.Values) > maxNodeNameFilterOptions {
			return fmt.Sprintf("Some node names are not shown on the suggestion list because the number of node names is %d, which is more than %d.", len(nodeNames.Values), maxNodeNameFilterOptions), inspectionmetadata.Warning, nil
		}
		return "", inspectionmetadata.None, nil
	}).
	Build()
