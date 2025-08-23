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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
)

const priorityForContainerGroup = googlecloudcommon_contract.FormBasePriority + 20000

var inputNamespacesAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{
	"managed": {"kube-system", "gke-system", "istio-system", "asm-system", "gmp-system", "gke-mcs", "configconnector-operator-system", "cnrm-system"},
}

// InputContainerQueryNamespaceFilterTask is a form task that allows users to specify which namespaces to query for container logs.
var InputContainerQueryNamespaceFilterTask = formtask.NewTextFormTaskBuilder(googlecloudlogk8scontainer_contract.InputContainerQueryNamespacesTaskID, priorityForContainerGroup+1000, "Namespaces(Container logs)").
	WithDefaultValueConstant("@managed", true).
	WithDescription(`Container logs tend to be a lot and take very long time to query.
Specify the space splitted namespace lists to query container logs only in the specific namespaces.`).
	WithValidator(func(ctx context.Context, value string) (string, error) {
		result, err := gcpqueryutil.ParseSetFilter(value, inputNamespacesAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string) (*gcpqueryutil.SetFilterParseResult, error) {
		result, err := gcpqueryutil.ParseSetFilter(value, inputNamespacesAliasMap, true, true, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()

var inputPodNamesAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{}

// InputContainerQueryPodNamesFilterMask is a form task that allows users to specify which pod names to query for container logs.
var InputContainerQueryPodNamesFilterMask = formtask.NewTextFormTaskBuilder(googlecloudlogk8scontainer_contract.InputContainerQueryPodNamesTaskID, priorityForContainerGroup+2000, "Pod names(Container logs)").
	WithDefaultValueConstant("@any", true).
	WithDescription(`Container logs tend to be a lot and take very long time to query.
	Specify the space splitted pod names lists to query container logs only in the specific pods.
	This parameter is evaluated as the partial match not the perfect match. You can use the prefix of the pod names.`).
	WithValidator(func(ctx context.Context, value string) (string, error) {
		result, err := gcpqueryutil.ParseSetFilter(value, inputPodNamesAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string) (*gcpqueryutil.SetFilterParseResult, error) {
		result, err := gcpqueryutil.ParseSetFilter(value, inputPodNamesAliasMap, true, true, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()
