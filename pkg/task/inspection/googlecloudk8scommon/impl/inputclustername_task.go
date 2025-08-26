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
	"slices"
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
	WithDependencies([]taskid.UntypedTaskReference{googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID, googlecloudk8scommon_contract.ClusterNamePrefixTaskID}).
	WithDescription("The cluster name to gather logs.").
	WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		clusters := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID)
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
		clusters := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID)
		return common.SortForAutocomplete(value, clusters.ClusterNames), nil
	}).
	WithHintFunc(func(ctx context.Context, value string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
		clusters := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID)
		prefix := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterNamePrefixTaskID)

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
		prefix := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterNamePrefixTaskID)
		return prefix + strings.TrimSpace(value), nil
	}).
	Build()
