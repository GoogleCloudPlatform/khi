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

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func mergeNEGNames(results []k8scommon.NEGNameToResourceIdentityMap) (k8scommon.NEGNameToResourceIdentityMap, error) {
	result := map[string]k8saudit.ResourceIdentity{}
	for _, r := range results {
		for negName, identity := range r {
			result[negName] = identity
		}
	}
	return result, nil
}

var NEGNamesInventoryTask = inspectiontaskbase.NewInventoryTask(
	k8scommon.NEGNamesInventoryTaskID,
	k8scommon.TagNEGNamesDiscovery,
	mergeNEGNames,
)

var NEGNamesDiscoveryTask = inspectiontaskbase.NewInspectionTask(
	k8scommon.NEGNamesDiscoveryTaskID,
	[]coretask.Dependency{
		k8saudit.ManifestGeneratorTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8scommon.NEGNameToResourceIdentityMap, error) {
		if taskMode == inspectioncore.TaskModeDryRun {
			return nil, nil
		}
		result := k8scommon.NEGNameToResourceIdentityMap{}
		resourceLogs := coretask.GetTaskResult(ctx, k8saudit.ManifestGeneratorTaskID.Ref())
		for _, group := range resourceLogs {
			if group.Resource.Type() != k8saudit.Resource {
				continue
			}
			if group.Resource.APIVersion != "networking.gke.io/v1beta1" || group.Resource.Kind != "servicenetworkendpointgroup" {
				continue
			}
			result[group.Resource.Name] = *group.Resource
		}
		return result, nil
	},
	coretask.ProvidesTag(k8scommon.TagNEGNamesDiscovery),
)
