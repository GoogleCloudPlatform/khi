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

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var ResourceUIDInventoryTask = commonlogk8sauditv2_contract.ResourceUIDInventoryBuilder.InventoryTask(&resourceUIDMergeStrategy{})

type resourceUIDMergeStrategy struct{}

var _ inspectiontaskbase.InventoryMergerStrategy[commonlogk8sauditv2_contract.UIDToResourceIdentity] = (*resourceUIDMergeStrategy)(nil)

// Merge implements inspectiontaskbase.InventoryMergerStrategy.
func (r *resourceUIDMergeStrategy) Merge(results []commonlogk8sauditv2_contract.UIDToResourceIdentity) (commonlogk8sauditv2_contract.UIDToResourceIdentity, error) {
	result := map[string]*commonlogk8sauditv2_contract.ResourceIdentity{}
	for _, r := range results {
		for uid, s := range r {
			result[uid] = s
		}
	}
	return result, nil
}

var ResourceUIDDiscoveryTask = commonlogk8sauditv2_contract.ResourceUIDInventoryBuilder.DiscoveryTask(
	commonlogk8sauditv2_contract.ResourceUIDDiscoveryTaskID,
	[]taskid.UntypedTaskReference{commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref()},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (commonlogk8sauditv2_contract.UIDToResourceIdentity, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return commonlogk8sauditv2_contract.UIDToResourceIdentity{}, nil
		}
		result := commonlogk8sauditv2_contract.UIDToResourceIdentity{}
		resourceLogs := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref())
		for _, group := range resourceLogs {
			if group.Resource.Type() != commonlogk8sauditv2_contract.Resource {
				continue
			}
			for _, log := range group.Logs {
				if log.ResourceBodyReader == nil {
					continue
				}
				uid, err := log.ResourceBodyReader.ReadString("metadata.uid")
				if err != nil {
					continue
				}
				result[uid] = group.Resource
			}
		}
		return result, nil
	},
)

var UIDPatternFinderTask = inspectiontaskbase.NewProgressReportableInspectionTask(
	commonlogk8sauditv2_contract.ResourceUIDPatternFinderTaskID,
	[]taskid.UntypedTaskReference{commonlogk8sauditv2_contract.ResourceUIDInventoryTaskID.Ref()},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (patternfinder.PatternFinder[*commonlogk8sauditv2_contract.ResourceIdentity], error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}
		uidMap := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ResourceUIDInventoryTaskID.Ref())
		finder := patternfinder.NewTriePatternFinder[*commonlogk8sauditv2_contract.ResourceIdentity]()
		for uid, resource := range uidMap {
			err := finder.AddPattern(uid, resource)
			if err != nil {
				return nil, err
			}
		}
		return finder, nil
	},
)
