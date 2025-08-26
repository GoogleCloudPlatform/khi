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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

var inputNamespacesAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{
	"all_cluster_scoped": {"#cluster-scoped"},
	"all_namespaced":     {"#namespaced"},
}

// InputNamespaceFilterTask is a form task for inputting the namespace filter.
var InputNamespaceFilterTask = formtask.NewTextFormTaskBuilder(googlecloudk8scommon_contract.InputNamespaceFilterTaskID, googlecloudcommon_contract.PriorityForK8sResourceFilterGroup+4000, "Namespaces").
	WithDefaultValueConstant("@all_cluster_scoped @all_namespaced", true).
	WithDescription("The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.").
	WithValidator(func(ctx context.Context, value string) (string, error) {
		if value == "" {
			return "namespace filter can't be empty", nil
		}
		result, err := gcpqueryutil.ParseSetFilter(value, inputNamespacesAliasMap, false, false, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string) (*gcpqueryutil.SetFilterParseResult, error) {
		result, err := gcpqueryutil.ParseSetFilter(value, inputNamespacesAliasMap, false, false, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()
