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

package task

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
)

const FormBasePriority = 100000
const PriorityForQueryTimeGroup = FormBasePriority + 50000
const PriorityForResourceIdentifierGroup = FormBasePriority + 40000
const PriorityForK8sResourceFilterGroup = FormBasePriority + 30000

var InputClusterNameTaskID = taskid.NewDefaultImplementationID[string](GCPPrefix + "input/cluster-name")

var clusterNameValidator = regexp.MustCompile(`^\s*[0-9a-z\-]+\s*$`)

var InputClusterNameTask = formtask.NewTextFormTaskBuilder(InputClusterNameTaskID, PriorityForResourceIdentifierGroup+4000, "Cluster name").
	WithDependencies([]taskid.UntypedTaskReference{AutocompleteClusterNamesTaskID, ClusterNamePrefixTaskID}).
	WithDescription("The cluster name to gather logs.").
	WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		clusters := coretask.GetTaskResult(ctx, AutocompleteClusterNamesTaskID)
		// If the previous value is included in the list of cluster names, the name is used as the default value.
		if len(previousValues) > 0 && slices.Index(clusters.ClusterNames, previousValues[0]) > -1 {
			return previousValues[0], nil
		}
		if len(clusters.ClusterNames) == 0 {
			return "", nil
		}
		return clusters.ClusterNames[0], nil
	}).
	WithSuggestionsFunc(func(ctx context.Context, value string, previousValues []string) ([]string, error) {
		clusters := coretask.GetTaskResult(ctx, AutocompleteClusterNamesTaskID)
		return common.SortForAutocomplete(value, clusters.ClusterNames), nil
	}).
	WithHintFunc(func(ctx context.Context, value string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
		clusters := coretask.GetTaskResult(ctx, AutocompleteClusterNamesTaskID)
		prefix := coretask.GetTaskResult(ctx, ClusterNamePrefixTaskID)

		// on failure of getting the list of clusters
		if clusters.Error != "" {
			return fmt.Sprintf("Failed to obtain the cluster list due to the error '%s'.\n The suggestion list won't popup", clusters.Error), inspectionmetadata.Warning, nil
		}
		convertedWithoutPrefix := strings.TrimPrefix(convertedValue.(string), prefix)
		for _, suggestedCluster := range clusters.ClusterNames {
			if suggestedCluster == convertedWithoutPrefix {
				return "", inspectionmetadata.Info, nil
			}
		}
		return fmt.Sprintf("Cluster `%s` was not found in the specified project at this time. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.", value), inspectionmetadata.Warning, nil
	}).
	WithValidator(func(ctx context.Context, value string) (string, error) {
		if !clusterNameValidator.Match([]byte(value)) {
			return "Cluster name must match `^[0-9a-z:\\-]+$`", nil
		}
		return "", nil
	}).
	WithConverter(func(ctx context.Context, value string) (string, error) {
		prefix := coretask.GetTaskResult(ctx, ClusterNamePrefixTaskID)

		return prefix + strings.TrimSpace(value), nil
	}).
	Build()

var InputKindFilterTaskID = taskid.NewDefaultImplementationID[*queryutil.SetFilterParseResult](GCPPrefix + "input/kinds")

var inputKindNameAliasMap queryutil.SetFilterAliasToItemsMap = map[string][]string{
	"default": strings.Split("pods replicasets daemonsets nodes deployments namespaces statefulsets services servicenetworkendpointgroups ingresses poddisruptionbudgets jobs cronjobs endpointslices persistentvolumes persistentvolumeclaims storageclasses horizontalpodautoscalers verticalpodautoscalers multidimpodautoscalers", " "),
}
var InputKindFilterTask = formtask.NewTextFormTaskBuilder(InputKindFilterTaskID, PriorityForK8sResourceFilterGroup+5000, "Kind").
	WithDefaultValueConstant("@default", true).
	WithDescription("The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources").
	WithValidator(func(ctx context.Context, value string) (string, error) {
		if value == "" {
			return "kind filter can't be empty", nil
		}
		result, err := queryutil.ParseSetFilter(value, inputKindNameAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string) (*queryutil.SetFilterParseResult, error) {
		result, err := queryutil.ParseSetFilter(value, inputKindNameAliasMap, true, true, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()

var InputNamespaceFilterTaskID = taskid.NewDefaultImplementationID[*queryutil.SetFilterParseResult](GCPPrefix + "input/namespaces")

var inputNamespacesAliasMap queryutil.SetFilterAliasToItemsMap = map[string][]string{
	"all_cluster_scoped": {"#cluster-scoped"},
	"all_namespaced":     {"#namespaced"},
}
var InputNamespaceFilterTask = formtask.NewTextFormTaskBuilder(InputNamespaceFilterTaskID, PriorityForK8sResourceFilterGroup+4000, "Namespaces").
	WithDefaultValueConstant("@all_cluster_scoped @all_namespaced", true).
	WithDescription("The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.").
	WithValidator(func(ctx context.Context, value string) (string, error) {
		if value == "" {
			return "namespace filter can't be empty", nil
		}
		result, err := queryutil.ParseSetFilter(value, inputNamespacesAliasMap, false, false, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string) (*queryutil.SetFilterParseResult, error) {
		result, err := queryutil.ParseSetFilter(value, inputNamespacesAliasMap, false, false, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()

var InputNodeNameFilterTaskID = taskid.NewDefaultImplementationID[[]string](GCPPrefix + "input/node-name-filter")

var nodeNameSubstringValidator = regexp.MustCompile("^[-a-z0-9]*$")

// getNodeNameSubstringsFromRawInput splits input by spaces and returns result in array.
// This removes surround spaces and removes empty string.
func getNodeNameSubstringsFromRawInput(value string) []string {
	result := []string{}
	nodeNameSubstrings := strings.Split(value, " ")
	for _, v := range nodeNameSubstrings {
		nodeNameSubstring := strings.TrimSpace(v)
		if nodeNameSubstring != "" {
			result = append(result, nodeNameSubstring)
		}
	}
	return result
}

// InputNodeNameFilterTask is a task to collect list of substrings of node names. This input value is used in querying k8s_node or serialport logs.
var InputNodeNameFilterTask = formtask.NewTextFormTaskBuilder(InputNodeNameFilterTaskID, PriorityForK8sResourceFilterGroup+3000, "Node names").
	WithDefaultValueConstant("", true).
	WithDescription("A space-separated list of node name substrings used to collect node-related logs. If left blank, KHI gathers logs from all nodes in the cluster.").
	WithValidator(func(ctx context.Context, value string) (string, error) {
		nodeNameSubstrings := getNodeNameSubstringsFromRawInput(value)
		for _, name := range nodeNameSubstrings {
			if !nodeNameSubstringValidator.Match([]byte(name)) {
				return fmt.Sprintf("substring `%s` is not valid as a substring of node name", name), nil
			}
		}
		return "", nil
	}).WithConverter(func(ctx context.Context, value string) ([]string, error) {
	return getNodeNameSubstringsFromRawInput(value), nil
}).Build()

var InputLocationsTaskID = taskid.NewDefaultImplementationID[string](GCPPrefix + "input/location")

var InputLocationsTask = formtask.NewTextFormTaskBuilder(InputLocationsTaskID, PriorityForResourceIdentifierGroup+4500, "Location").
	WithDependencies([]taskid.UntypedTaskReference{AutocompleteLocationTaskID.Ref()}).
	WithDescription(
		"The location(region) to specify the resource exist(s|ed)",
	).
	WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		if len(previousValues) > 0 {
			return previousValues[0], nil
		}
		return "", nil
	}).
	WithSuggestionsFunc(func(ctx context.Context, value string, previousValues []string) ([]string, error) {
		if len(previousValues) > 0 { // no need to call twice; should be the same
			return previousValues, nil
		}
		regions := coretask.GetTaskResult(ctx, AutocompleteLocationTaskID.Ref())
		return regions, nil
	}).
	Build()
