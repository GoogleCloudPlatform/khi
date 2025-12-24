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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

const maxNamespaceFilterOptions = 500

var inputNamespacesAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{
	"all_cluster_scoped": {"#cluster-scoped"},
	"all_namespaced":     {"#namespaced"},
}

// InputNamespaceFilterTask is a form task for inputting the namespace filter.
var InputNamespaceFilterTask = formtask.NewSetFormTaskBuilder(googlecloudk8scommon_contract.InputNamespaceFilterTaskID, googlecloudcommon_contract.PriorityForK8sResourceFilterGroup+4000, "Namespaces").
	WithDependencies([]taskid.UntypedTaskReference{googlecloudk8scommon_contract.AutocompleteNamespacesTaskID.Ref()}).
	WithDefaultValueConstant([]string{"@all_cluster_scoped", "@all_namespaced"}, true).
	WithDescription("The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.").
	WithAllowAddAll(false).
	WithAllowRemoveAll(false).
	WithAllowCustomValue(true).
	WithOptionsFunc(func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
		result := []inspectionmetadata.SetParameterFormFieldOptionItem{
			{ID: "@all_cluster_scoped", Description: "[Alias] An alias matches any of the cluster scoped resources"},
			{ID: "@all_namespaced", Description: "[Alias] An alias matches any of the namespaced resources"},
		}
		namespaces := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteNamespacesTaskID.Ref())
		for index, namespace := range namespaces.Values {
			if index >= maxNamespaceFilterOptions {
				break
			}
			result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{ID: namespace})
		}
		return result, nil
	}).
	WithHintFunc(func(ctx context.Context, value []string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
		namespaces := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteNamespacesTaskID.Ref())
		if len(namespaces.Values) > maxNamespaceFilterOptions {
			return fmt.Sprintf("Some namespaces are not shown on the suggestion list because the number of namespaces is %d, which is more than %d.", len(namespaces.Values), maxNamespaceFilterOptions), inspectionmetadata.Warning, nil
		}
		return "", inspectionmetadata.None, nil
	}).
	WithValidator(func(ctx context.Context, value []string) (string, error) {
		if len(value) == 0 {
			return "namespace filter can't be empty", nil
		}
		namespaceFilterInStr := strings.Join(value, " ")
		result, err := gcpqueryutil.ParseSetFilter(namespaceFilterInStr, inputNamespacesAliasMap, false, false, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value []string) (*gcpqueryutil.SetFilterParseResult, error) {
		namespaceFilterInStr := strings.Join(value, " ")
		result, err := gcpqueryutil.ParseSetFilter(namespaceFilterInStr, inputNamespacesAliasMap, false, false, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()
