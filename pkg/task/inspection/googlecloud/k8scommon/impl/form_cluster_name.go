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
	"regexp"
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

var clusterNameValidator = regexp.MustCompile(`^\s*[0-9a-z\-]+\s*$`)

// inputClusterNameTask receives the cluster name from the user.
// This task returns the raw cluster name without the prefix defined from the cluster type.
// This input also supports autocomplete cluster names from some task having ID for k8scommon.AutocompleteClusterIdentityTaskID.
var inputClusterNameTask = formtask.DefineTextForm(
	k8scommon.InputClusterNameTaskID,
	gcpcommon.PriorityForResourceIdentifierGroup+4000,
	"Cluster name",
	"The cluster name to gather logs.",
	func(b *coretask.Binder) formtask.TextFormSpec[string] {
		clusters := coretask.Use(b, k8scommon.AutocompleteClusterIdentityTaskID.Ref())
		return formtask.TextFormSpec[string]{
			DefaultValue: func(ctx context.Context, previousValues []string) (string, error) {
				c := clusters.Get(ctx)
				// If the previous value is included in the list of cluster names, the name is used as the default value.
				if len(previousValues) > 0 && hasClusterNameInAutocomplete(c.Values, previousValues[0]) {
					return previousValues[0], nil
				}
				if len(c.Values) == 0 {
					return "", nil
				}
				return c.Values[0].ClusterName, nil
			},
			Suggestions: func(ctx context.Context, value string, previousValues []string) ([]string, error) {
				c := clusters.Get(ctx)
				return common.SortForAutocomplete(value, dedupeClusterName(c.Values)), nil
			},
			Validator: validateClusterName,
			Converter: convertClusterName,
			Hint: func(ctx context.Context, value string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
				c := clusters.Get(ctx)
				// on failure of getting the list of clusters
				if c.Error != "" {
					return fmt.Sprintf("Failed to obtain the cluster list due to the error '%s'.\n The suggestion list won't popup", c.Error), inspectionmetadata.Warning, nil
				}
				if c.Hint != "" {
					return c.Hint, inspectionmetadata.Info, nil
				}
				for _, suggestedCluster := range c.Values {
					if suggestedCluster.ClusterName == convertedValue.(string) {
						return "", inspectionmetadata.Info, nil
					}
				}

				availableClusterNameStr := ""
				for _, cluster := range dedupeClusterName(c.Values) {
					availableClusterNameStr += fmt.Sprintf("* %s\n", cluster)
				}
				return fmt.Sprintf("Cluster '%s' was not found in the specified project at this time. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.\nAvailable cluster names:\n%s", value, availableClusterNameStr), inspectionmetadata.Warning, nil
			},
		}
	},
)

func hasClusterNameInAutocomplete(autocmpleteList []k8scommon.GoogleCloudClusterIdentity, clusterName string) bool {
	for _, cluster := range autocmpleteList {
		if cluster.ClusterName == clusterName {
			return true
		}
	}
	return false
}

func dedupeClusterName(clusters []k8scommon.GoogleCloudClusterIdentity) []string {
	clusterNameMap := make(map[string]bool)
	for _, cluster := range clusters {
		clusterNameMap[cluster.ClusterName] = true
	}
	result := []string{}
	for clusterName := range clusterNameMap {
		result = append(result, clusterName)
	}
	sort.Strings(result)
	return result
}

func validateClusterName(ctx context.Context, value string) (string, error) {
	if !clusterNameValidator.Match([]byte(value)) {
		return "Cluster name must consist of alphanumeric characters and hyphens only.", nil
	}
	return "", nil
}

func convertClusterName(ctx context.Context, value string) (string, error) {
	return strings.TrimSpace(value), nil
}
