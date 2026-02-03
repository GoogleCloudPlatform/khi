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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

var clusterNameValidator = regexp.MustCompile(`^\s*[0-9a-z\-]+\s*$`)

// InputClusterNameTask is a form task receving cluster name from the user.
// This task return the cluster name with the prefixes defined from the cluster type. For example, a cluster named foo-cluster is `foo-cluster` in GKE but `awsCluster/foo-cluster` in GKE on AWS.
// This input also supports autocomplete cluster names from some task having ID for googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID.
var InputClusterNameTask = formtask.NewTextFormTaskBuilder(googlecloudk8scommon_contract.InputClusterNameTaskID, googlecloudcommon_contract.PriorityForResourceIdentifierGroup+4000, "Cluster name").
	WithDependencies([]taskid.UntypedTaskReference{googlecloudk8scommon_contract.AutocompleteClusterIdentityTaskID.Ref(), googlecloudk8scommon_contract.ClusterNamePrefixTaskRef}).
	WithDescription("The cluster name to gather logs.").
	WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		clusters := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteClusterIdentityTaskID.Ref())
		// If the previous value is included in the list of cluster names, the name is used as the default value.
		if len(previousValues) > 0 && hasClusterNameInAutocomplete(clusters.Values, previousValues[0]) {
			return previousValues[0], nil
		}
		if len(clusters.Values) == 0 {
			return "", nil
		}
		return clusters.Values[0].ClusterName, nil
	}).
	WithSuggestionsFunc(func(ctx context.Context, value string, previousValues []string) ([]string, error) {
		clusters := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteClusterIdentityTaskID.Ref())
		return common.SortForAutocomplete(value, dedupeClusterName(clusters.Values)), nil
	}).
	WithHintFunc(func(ctx context.Context, value string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
		clusters := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteClusterIdentityTaskID.Ref())
		// on failure of getting the list of clusters
		if clusters.Error != "" {
			return fmt.Sprintf("Failed to obtain the cluster list due to the error '%s'.\n The suggestion list won't popup", clusters.Error), inspectionmetadata.Warning, nil
		}
		if clusters.Hint != "" {
			return clusters.Hint, inspectionmetadata.Info, nil
		}
		for _, suggestedCluster := range clusters.Values {
			if suggestedCluster.NameWithClusterTypePrefix() == convertedValue.(string) {
				return "", inspectionmetadata.Info, nil
			}
		}
		availableClusterNameStr := ""
		for _, cluster := range dedupeClusterName(clusters.Values) {
			availableClusterNameStr += fmt.Sprintf("* %s\n", cluster)
		}
		return fmt.Sprintf("Cluster '%s' was not found in the specified project at this time. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.\nAvailable cluster names:\n%s", value, availableClusterNameStr), inspectionmetadata.Warning, nil
	}).
	WithValidator(func(ctx context.Context, value string) (string, error) {
		if !clusterNameValidator.Match([]byte(value)) {
			return "Cluster name must match `^[0-9a-z:\\-]+$`", nil
		}
		return "", nil
	}).
	WithConverter(func(ctx context.Context, value string) (string, error) {
		prefix := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterNamePrefixTaskRef)
		return prefix + strings.TrimSpace(value), nil
	}).
	Build()

func hasClusterNameInAutocomplete(autocmpleteList []googlecloudk8scommon_contract.GoogleCloudClusterIdentity, clusterName string) bool {
	for _, cluster := range autocmpleteList {
		if cluster.ClusterName == clusterName {
			return true
		}
	}
	return false
}

func dedupeClusterName(clusters []googlecloudk8scommon_contract.GoogleCloudClusterIdentity) []string {
	clusterNameMap := make(map[string]bool)
	for _, cluster := range clusters {
		clusterNameMap[cluster.ClusterName] = true
	}
	result := []string{}
	for clusterName := range clusterNameMap {
		result = append(result, clusterName)
	}
	return result
}
