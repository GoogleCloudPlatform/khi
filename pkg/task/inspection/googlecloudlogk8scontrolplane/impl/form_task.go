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

package googlecloudlogk8scontrolplane_impl

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
)

const priorityForControlPlaneGroup = googlecloudcommon_contract.FormBasePriority + 30000

var inputControlPlaneComponentNameAliasMap map[string][]string = map[string][]string{}

// InputControlPlaneComponentNameFilterTask is a form task for filtering control plane component names.
var InputControlPlaneComponentNameFilterTask = formtask.NewTextFormTaskBuilder(
	googlecloudlogk8scontrolplane_contract.InputControlPlaneComponentNameFilterTaskID,
	priorityForControlPlaneGroup+1000,
	"Control plane component names",
).
	WithDefaultValueConstant("@any", true).
	WithSuggestionsConstant([]string{
		"apiserver",
		"controller-manager",
		"scheduler",
	}).
	WithDescription("Control plane component names to query(e.g. apiserver, controller-manager...etc)").
	WithValidator(func(ctx context.Context, value string) (string, error) {
		result, err := gcpqueryutil.ParseSetFilter(value, inputControlPlaneComponentNameAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string) (*gcpqueryutil.SetFilterParseResult, error) {
		result, err := gcpqueryutil.ParseSetFilter(value, inputControlPlaneComponentNameAliasMap, true, true, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()
