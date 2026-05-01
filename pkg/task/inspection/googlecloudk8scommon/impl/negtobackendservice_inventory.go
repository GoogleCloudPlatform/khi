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
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// NEGToBackendServiceMergeStrategy is the merge strategy for the NEG to BackendService inventory.
type NEGToBackendServiceMergeStrategy struct{}

var _ inspectiontaskbase.InventoryMergerStrategy[googlecloudk8scommon_contract.NEGToBackendServiceMap] = (*NEGToBackendServiceMergeStrategy)(nil)

func (s *NEGToBackendServiceMergeStrategy) Merge(maps []googlecloudk8scommon_contract.NEGToBackendServiceMap) (googlecloudk8scommon_contract.NEGToBackendServiceMap, error) {
	result := make(googlecloudk8scommon_contract.NEGToBackendServiceMap)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result, nil
}

// NEGToBackendServiceInventoryTask is the inventory task that provides aggregated NEG to BackendService mappings.
var NEGToBackendServiceInventoryTask = googlecloudk8scommon_contract.NEGToBackendServiceInventoryBuilder.InventoryTask(
	&NEGToBackendServiceMergeStrategy{},
)
