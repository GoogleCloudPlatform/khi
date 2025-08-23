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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

var inputKindNameAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{
	"default": strings.Split("pods replicasets daemonsets nodes deployments namespaces statefulsets services servicenetworkendpointgroups ingresses poddisruptionbudgets jobs cronjobs endpointslices persistentvolumes persistentvolumeclaims storageclasses horizontalpodautoscalers verticalpodautoscalers multidimpodautoscalers", " "),
}

// InputKindFilterTask is a form task for inputting the kind filter.
var InputKindFilterTask = formtask.NewTextFormTaskBuilder(googlecloudk8scommon_contract.InputKindFilterTaskID, googlecloudcommon_contract.PriorityForK8sResourceFilterGroup+5000, "Kind").
	WithDefaultValueConstant("@default", true).
	WithDescription("The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources").
	WithValidator(func(ctx context.Context, value string) (string, error) {
		if value == "" {
			return "kind filter can't be empty", nil
		}
		result, err := gcpqueryutil.ParseSetFilter(value, inputKindNameAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string) (*gcpqueryutil.SetFilterParseResult, error) {
		result, err := gcpqueryutil.ParseSetFilter(value, inputKindNameAliasMap, true, true, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()
