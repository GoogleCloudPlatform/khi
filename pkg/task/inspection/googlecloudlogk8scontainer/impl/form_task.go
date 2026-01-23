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

package googlecloudlogk8scontainer_impl

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
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
)

const priorityForContainerGroup = googlecloudcommon_contract.FormBasePriority + 20000

const maxNamespaceFilterOptions = 500
const maxPodNameFilterOptions = 500

var inputNamespacesAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{
	"managed": {"kube-system", "gke-system", "istio-system", "asm-system", "gmp-system", "gke-mcs", "configconnector-operator-system", "cnrm-system"},
}

// InputContainerQueryNamespaceFilterTask is a form task that allows users to specify which namespaces to query for container logs.
var InputContainerQueryNamespaceFilterTask = formtask.NewSetFormTaskBuilder(googlecloudlogk8scontainer_contract.InputContainerQueryNamespacesTaskID, priorityForContainerGroup+1000, "Namespaces(Container logs)").
	WithDependencies([]taskid.UntypedTaskReference{googlecloudk8scommon_contract.AutocompleteNamespacesTaskID.Ref()}).
	WithDefaultValueConstant([]string{"@managed"}, true).
	WithAllowAddAll(false).
	WithAllowRemoveAll(false).
	WithAllowCustomValue(true).
	WithDescription(`Container logs tend to be a lot and take very long time to query.
Specify the space splitted namespace lists to query container logs only in the specific namespaces.`).
	WithOptionsFunc(func(ctx context.Context, value []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
		result := []inspectionmetadata.SetParameterFormFieldOptionItem{}
		namespaces := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteNamespacesTaskID.Ref())
		result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{
			ID:          "@managed",
			Description: "[Alias] An alias matches the managed namespaces(e.g kube-system,gke-system,...etc).",
		}, inspectionmetadata.SetParameterFormFieldOptionItem{
			ID:          "@any",
			Description: "[Alias] An alias matches any pod namespaces.",
		})
		for i, namespace := range namespaces.Values {
			if i >= maxNamespaceFilterOptions {
				break
			}
			result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{
				ID: namespace,
			})
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
		setFilterStr := strings.Join(value, " ")
		result, err := gcpqueryutil.ParseSetFilter(setFilterStr, inputNamespacesAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value []string) (*gcpqueryutil.SetFilterParseResult, error) {
		setFilterStr := strings.Join(value, " ")
		result, err := gcpqueryutil.ParseSetFilter(setFilterStr, inputNamespacesAliasMap, true, true, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()

var inputPodNamesAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{}

// InputContainerQueryPodNamesFilterMask is a form task that allows users to specify which pod names to query for container logs.
var InputContainerQueryPodNamesFilterMask = formtask.NewSetFormTaskBuilder(googlecloudlogk8scontainer_contract.InputContainerQueryPodNamesTaskID, priorityForContainerGroup+2000, "Pod names(Container logs)").
	WithDependencies([]taskid.UntypedTaskReference{googlecloudk8scommon_contract.AutocompletePodNamesTaskID.Ref()}).
	WithDefaultValueConstant([]string{"@any"}, true).
	WithAllowAddAll(false).
	WithAllowRemoveAll(false).
	WithAllowCustomValue(true).
	WithDescription(`Container logs tend to be a lot and take very long time to query.
	Specify the space splitted pod names lists to query container logs only in the specific pods.
	This parameter is evaluated as the partial match not the perfect match. You can use the prefix of the pod names.`).
	WithOptionsFunc(func(ctx context.Context, value []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
		result := []inspectionmetadata.SetParameterFormFieldOptionItem{}
		podNames := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompletePodNamesTaskID.Ref())
		result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{
			ID:          "@any",
			Description: "[Alias] An alias matches any pod names.",
		})
		for i, podName := range podNames.Values {
			if i >= maxPodNameFilterOptions {
				break
			}
			result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{
				ID: podName,
			})
		}
		return result, nil
	}).
	WithHintFunc(func(ctx context.Context, value []string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
		podNames := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompletePodNamesTaskID.Ref())
		if len(podNames.Values) > maxPodNameFilterOptions {
			return fmt.Sprintf("Some pod names are not shown on the suggestion list because the number of pod names is %d, which is more than %d.", len(podNames.Values), maxPodNameFilterOptions), inspectionmetadata.Warning, nil
		}
		return "", inspectionmetadata.None, nil
	}).
	WithValidator(func(ctx context.Context, value []string) (string, error) {
		setFilterStr := strings.Join(value, " ")
		result, err := gcpqueryutil.ParseSetFilter(setFilterStr, inputPodNamesAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value []string) (*gcpqueryutil.SetFilterParseResult, error) {
		setFilterStr := strings.Join(value, " ")
		result, err := gcpqueryutil.ParseSetFilter(setFilterStr, inputPodNamesAliasMap, true, true, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()
