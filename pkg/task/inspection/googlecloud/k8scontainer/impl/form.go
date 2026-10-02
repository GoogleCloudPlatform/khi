// Copyright 2024 Google LLC
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

package k8scontainer_impl

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
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontainer"
)

const priorityForContainerGroup = gcpcommon.FormBasePriority + 20000

const maxNamespaceFilterOptions = 500
const maxPodNameFilterOptions = 500

var inputNamespacesAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{
	"managed": {"kube-system", "gke-system", "istio-system", "asm-system", "gmp-system", "gke-mcs", "configconnector-operator-system", "cnrm-system"},
}

// inputContainerQueryNamespaceFilterTask is a form task that allows users to specify which namespaces to query for container logs.
var inputContainerQueryNamespaceFilterTask = formtask.DefineSetForm(
	k8scontainer.InputContainerQueryNamespacesTaskID,
	priorityForContainerGroup+1000,
	"Namespaces(Container logs)",
	`Container logs tend to be a lot and take very long time to query.
Specify the space splitted namespace lists to query container logs only in the specific namespaces.`,
	func(b *coretask.Binder) formtask.SetFormSpec[*gcpqueryutil.SetFilterParseResult] {
		namespaces := coretask.Use(b, k8scommon.AutocompleteNamespacesTaskID.Ref())
		return formtask.SetFormSpec[*gcpqueryutil.SetFilterParseResult]{
			DefaultValue:     formtask.PreviousOrConstantDefaultValue([]string{"@managed"}),
			AllowCustomValue: formtask.ConstantBool(true),
			AllowAddAll:      formtask.ConstantBool(false),
			AllowRemoveAll:   formtask.ConstantBool(false),
			Options: func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
				result := []inspectionmetadata.SetParameterFormFieldOptionItem{
					{
						ID:          "@managed",
						Description: "[Alias] An alias matches the managed namespaces(e.g kube-system,gke-system,...etc).",
					},
					{
						ID:          "@any",
						Description: "[Alias] An alias matches any pod namespaces.",
					},
				}
				ns := namespaces.Get(ctx)
				for i, namespace := range ns.Values {
					if i >= maxNamespaceFilterOptions {
						break
					}
					result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{
						ID: namespace,
					})
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
			Validator: func(ctx context.Context, value []string) (string, error) {
				setFilterStr := strings.Join(value, " ")
				result, err := gcpqueryutil.ParseSetFilter(setFilterStr, inputNamespacesAliasMap, true, true, true)
				if err != nil {
					return "", err
				}
				return result.ValidationError, nil
			},
			Converter: func(ctx context.Context, value []string) (*gcpqueryutil.SetFilterParseResult, error) {
				setFilterStr := strings.Join(value, " ")
				result, err := gcpqueryutil.ParseSetFilter(setFilterStr, inputNamespacesAliasMap, true, true, true)
				if err != nil {
					return nil, err
				}
				return result, nil
			},
		}
	},
)

var inputPodNamesAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{}

// inputContainerQueryPodNamesFilterTask is a form task that allows users to specify which pod names to query for container logs.
var inputContainerQueryPodNamesFilterTask = formtask.DefineSetForm(
	k8scontainer.InputContainerQueryPodNamesTaskID,
	priorityForContainerGroup+2000,
	"Pod names(Container logs)",
	`Container logs tend to be a lot and take very long time to query.
	Specify the space splitted pod names lists to query container logs only in the specific pods.
	This parameter is evaluated as the partial match not the perfect match. You can use the prefix of the pod names.`,
	func(b *coretask.Binder) formtask.SetFormSpec[*gcpqueryutil.SetFilterParseResult] {
		podNames := coretask.Use(b, k8scommon.AutocompletePodNamesTaskID.Ref())
		return formtask.SetFormSpec[*gcpqueryutil.SetFilterParseResult]{
			DefaultValue:     formtask.PreviousOrConstantDefaultValue([]string{"@any"}),
			AllowCustomValue: formtask.ConstantBool(true),
			AllowAddAll:      formtask.ConstantBool(false),
			AllowRemoveAll:   formtask.ConstantBool(false),
			Options: func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
				result := []inspectionmetadata.SetParameterFormFieldOptionItem{
					{
						ID:          "@any",
						Description: "[Alias] An alias matches any pod names.",
					},
				}
				names := podNames.Get(ctx)
				for i, podName := range names.Values {
					if i >= maxPodNameFilterOptions {
						break
					}
					result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{
						ID: podName,
					})
				}
				return result, nil
			},
			Hint: func(ctx context.Context, value []string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
				names := podNames.Get(ctx)
				if len(names.Values) > maxPodNameFilterOptions {
					return fmt.Sprintf("Some pod names are not shown on the suggestion list because the number of pod names is %d, which is more than %d.", len(names.Values), maxPodNameFilterOptions), inspectionmetadata.Warning, nil
				}
				return "", inspectionmetadata.None, nil
			},
			Validator: func(ctx context.Context, value []string) (string, error) {
				setFilterStr := strings.Join(value, " ")
				result, err := gcpqueryutil.ParseSetFilter(setFilterStr, inputPodNamesAliasMap, true, true, true)
				if err != nil {
					return "", err
				}
				return result.ValidationError, nil
			},
			Converter: func(ctx context.Context, value []string) (*gcpqueryutil.SetFilterParseResult, error) {
				setFilterStr := strings.Join(value, " ")
				result, err := gcpqueryutil.ParseSetFilter(setFilterStr, inputPodNamesAliasMap, true, true, true)
				if err != nil {
					return nil, err
				}
				return result, nil
			},
		}
	},
)
