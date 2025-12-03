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

package commonlogk8sauditv2_impl

import (
	"context"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// NodeNameInventoryTask provides list of node name found in this inspection for later task usage.
var NodeNameInventoryTask = commonlogk8sauditv2_contract.NodeNameInventoryBuilder.InventoryTask(&nodeNameMergeStrategy{})

type nodeNameMergeStrategy struct{}

// Merge implements inspectiontaskbase.InventoryMergerStrategy.
func (n *nodeNameMergeStrategy) Merge(results [][]string) ([]string, error) {
	result := map[string]struct{}{}
	for _, r := range results {
		for _, s := range r {
			result[s] = struct{}{}
		}
	}

	var ret []string
	for k := range result {
		ret = append(ret, k)
	}
	return ret, nil
}

var _ inspectiontaskbase.InventoryMergerStrategy[[]string] = (*nodeNameMergeStrategy)(nil)

// NodeNameDiscoveryTask extracts node name from audit logs and node names are registered on NodeNameInventoryTask.
var NodeNameDiscoveryTask = commonlogk8sauditv2_contract.NodeNameInventoryBuilder.DiscoveryTask(
	commonlogk8sauditv2_contract.NodeNameDiscoveryTaskID,
	[]taskid.UntypedTaskReference{commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref()},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]string, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}

		foundNodeNames := map[string]struct{}{}
		resourceLogs := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref())
		for _, group := range resourceLogs {
			if group.Resource.Type() != commonlogk8sauditv2_contract.Resource {
				continue
			}
			if group.Resource.APIVersion != "core/v1" || group.Resource.Kind != "node" {
				continue
			}
			foundNodeNames[group.Resource.Name] = struct{}{}
		}
		var ret []string
		for k := range foundNodeNames {
			ret = append(ret, k)
		}
		return ret, nil
	},
)
