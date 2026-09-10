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

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func mergeNEGNames(results []googlecloudk8scommon_contract.NEGNameToResourceIdentityMap) (googlecloudk8scommon_contract.NEGNameToResourceIdentityMap, error) {
	result := map[string]commonlogk8saudit_contract.ResourceIdentity{}
	for _, r := range results {
		for negName, identity := range r {
			result[negName] = identity
		}
	}
	return result, nil
}

var NEGNamesInventoryTask = inspectiontaskbase.NewInventoryTask(
	googlecloudk8scommon_contract.NEGNamesInventoryTaskID,
	googlecloudk8scommon_contract.TagNEGNamesDiscovery,
	mergeNEGNames,
)

var NEGNamesDiscoveryTask = inspectiontaskbase.NewInspectionTask(
	googlecloudk8scommon_contract.NEGNamesDiscoveryTaskID,
	[]coretask.Dependency{
		commonlogk8saudit_contract.ManifestGeneratorTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (googlecloudk8scommon_contract.NEGNameToResourceIdentityMap, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}
		result := googlecloudk8scommon_contract.NEGNameToResourceIdentityMap{}
		resourceLogs := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ManifestGeneratorTaskID.Ref())
		for _, group := range resourceLogs {
			if group.Resource.Type() != commonlogk8saudit_contract.Resource {
				continue
			}
			if group.Resource.APIVersion != "networking.gke.io/v1beta1" || group.Resource.Kind != "servicenetworkendpointgroup" {
				continue
			}
			result[group.Resource.Name] = *group.Resource
		}
		return result, nil
	},
	coretask.ProvidesTag(googlecloudk8scommon_contract.TagNEGNamesDiscovery),
)
