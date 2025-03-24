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

package k8scontrolplanecomponent

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/form"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	k8s_control_plane_component_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_control_plane_component/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const priorityForControlPlaneGroup = gcp_task.FormBasePriority + 30000

var inputControlPlaneComponentNameAliasMap map[string][]string = map[string][]string{}

var InputControlPlaneComponentNameFilterTask = form.NewInputFormDefinitionBuilder(
	k8s_control_plane_component_taskid.InputControlPlaneComponentNameFilterTaskID,
	priorityForControlPlaneGroup+1000,
	"Control plane component names",
).
	WithDefaultValueConstant("@any", true).
	WithSuggestionsConstant([]string{
		"apiserver",
		"controller-manager",
		"scheduler",
	}).
	WithUIDescription("Control plane component names to query(e.g. apiserver, controller-manager...etc)").
	WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
		result, err := queryutil.ParseSetFilter(value, inputControlPlaneComponentNameAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string, variables *task.VariableSet) (*queryutil.SetFilterParseResult, error) {
		result, err := queryutil.ParseSetFilter(value, inputControlPlaneComponentNameAliasMap, true, true, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}).
	Build()

func GetInputControlPlaneComponentNameFilterFromTaskVariable(tv *task.VariableSet) (*queryutil.SetFilterParseResult, error) {
	return task.GetTypedVariableFromTaskVariable[*queryutil.SetFilterParseResult](tv, k8s_control_plane_component_taskid.InputControlPlaneComponentNameFilterTaskID.ReferenceIDString(), nil)
}
