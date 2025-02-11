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

package k8s_container

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/form"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const priorityForContainerGroup = gcp_task.FormBasePriority + 20000

const InputContainerQueryNamespacesTaskID = gcp_task.GCPPrefix + "input/container-query-namespaces"

var inputNamespacesAliasMap queryutil.SetFilterAliasToItemsMap = map[string][]string{
	"managed": {"kube-system", "gke-system", "istio-system", "asm-system", "gmp-system", "gke-mcs", "configconnector-operator-system", "cnrm-system"},
}
var InputContainerQueryNamespaceFilterTask = form.NewInputFormDefinitionBuilder(InputContainerQueryNamespacesTaskID, priorityForContainerGroup+1000, "Namespaces(Container logs)").
	WithDefaultValueConstant("@managed", true).
	WithUIDescription(`Container logs tend to be a lot and take very long time to query.
Specify the space splitted namespace lists to query container logs only in the specific namespaces.`).
	WithDocumentDescription("The namespace of Pods to gather container logs. Specify `@managed` to gather logs of system components.").
	WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
		result, err := queryutil.ParseSetFilter(value, inputNamespacesAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string, variables *task.VariableSet) (any, error) {
		result, err := queryutil.ParseSetFilter(value, inputNamespacesAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result, nil
	}).
	Build()

func GetInputContainerQueryNamespacesFilterFromTaskVariable(tv *task.VariableSet) (*queryutil.SetFilterParseResult, error) {
	return task.GetTypedVariableFromTaskVariable[*queryutil.SetFilterParseResult](tv, InputContainerQueryNamespacesTaskID, nil)
}

const InputContainerQueryPodNamesTaskID = gcp_task.GCPPrefix + "input/container-query-podnames"

var inputPodNamesAliasMap queryutil.SetFilterAliasToItemsMap = map[string][]string{}
var InputContainerQueryPodNamesFilterMask = form.NewInputFormDefinitionBuilder(InputContainerQueryPodNamesTaskID, priorityForContainerGroup+2000, "Pod names(Container logs)").
	WithDefaultValueConstant("@any", true).
	WithUIDescription(`Container logs tend to be a lot and take very long time to query.
	Specify the space splitted pod names lists to query container logs only in the specific pods.
	This parameter is evaluated as the partial match not the perfect match. You can use the prefix of the pod names.`).
	WithDocumentDescription("The substring of Pod name to gather container logs. Specify `@any` to gather logs of all pods.").
	WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
		result, err := queryutil.ParseSetFilter(value, inputPodNamesAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string, variables *task.VariableSet) (any, error) {
		result, err := queryutil.ParseSetFilter(value, inputPodNamesAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result, nil
	}).
	Build()

func GetInputContainerQueryPodNamesFilterFromTaskVariable(tv *task.VariableSet) (*queryutil.SetFilterParseResult, error) {
	return task.GetTypedVariableFromTaskVariable[*queryutil.SetFilterParseResult](tv, InputContainerQueryPodNamesTaskID, nil)
}
