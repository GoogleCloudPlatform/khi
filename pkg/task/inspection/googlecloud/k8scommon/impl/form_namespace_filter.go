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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

const maxNamespaceFilterOptions = 500

var inputNamespacesAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{
	"all_cluster_scoped": {"#cluster-scoped"},
	"all_namespaced":     {"#namespaced"},
}

// inputNamespaceFilterTask is a form task for inputting the namespace filter.
var inputNamespaceFilterTask = formtask.DefineSetForm(
	k8scommon.InputNamespaceFilterTaskID,
	gcpcommon.PriorityForK8sResourceFilterGroup+4000,
	"Namespaces",
	"The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.",
	func(b *coretask.Binder) formtask.SetFormSpec[*gcpqueryutil.SetFilterParseResult] {
		namespaces := coretask.Use(b, k8scommon.AutocompleteNamespacesTaskID.Ref())
		return formtask.SetFormSpec[*gcpqueryutil.SetFilterParseResult]{
			DefaultValue:     formtask.PreviousOrConstantDefaultValue([]string{"@all_cluster_scoped", "@all_namespaced"}),
			AllowCustomValue: formtask.ConstantBool(true),
			AllowAddAll:      formtask.ConstantBool(false),
			AllowRemoveAll:   formtask.ConstantBool(false),
			Options: func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
				result := []inspectionmetadata.SetParameterFormFieldOptionItem{
					{ID: "@all_cluster_scoped", Description: "[Alias] An alias matches any of the cluster scoped resources"},
					{ID: "@all_namespaced", Description: "[Alias] An alias matches any of the namespaced resources"},
				}
				ns := namespaces.Get(ctx)
				for index, namespace := range ns.Values {
					if index >= maxNamespaceFilterOptions {
						break
					}
					result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{ID: namespace})
				}
				return result, nil
			},
			Hint: func(ctx context.Context, value []string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
				ns := namespaces.Get(ctx)
				if len(ns.Values) > maxNamespaceFilterOptions {
					return fmt.Sprintf("Some namespaces are not shown on the suggestion list because the number of namespaces is %d, which is more than %d.", len(ns.Values), maxNamespaceFilterOptions), inspectionmetadata.Warning, nil
				}
				return "", inspectionmetadata.None, nil
			},
			Validator: validateNamespaceFilter,
			Converter: convertNamespaceFilter,
		}
	},
)

// validateNamespaceFilter validates that the namespace filter is not empty and contains valid filter expressions.
func validateNamespaceFilter(ctx context.Context, value []string) (string, error) {
	if len(value) == 0 {
		return "namespace filter can't be empty", nil
	}
	namespaceFilterInStr := strings.Join(value, " ")
	result, err := gcpqueryutil.ParseSetFilter(namespaceFilterInStr, inputNamespacesAliasMap, false, false, true)
	if err != nil {
		return "", err
	}
	return result.ValidationError, nil
}

// convertNamespaceFilter parses the namespace filter string slice into a SetFilterParseResult.
func convertNamespaceFilter(ctx context.Context, value []string) (*gcpqueryutil.SetFilterParseResult, error) {
	namespaceFilterInStr := strings.Join(value, " ")
	result, err := gcpqueryutil.ParseSetFilter(namespaceFilterInStr, inputNamespacesAliasMap, false, false, true)
	if err != nil {
		return nil, err
	}
	return result, nil
}
