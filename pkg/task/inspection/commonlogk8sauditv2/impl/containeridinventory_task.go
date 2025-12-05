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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var ContainerIDInventoryTask = commonlogk8sauditv2_contract.ContainerIDInventoryBuilder.InventoryTask(&containerIDMergeStrategy{})

type containerIDMergeStrategy struct{}

// Merge implements inspectiontaskbase.InventoryMergerStrategy.
func (c *containerIDMergeStrategy) Merge(results []commonlogk8sauditv2_contract.ContainerIDToContainerIdentity) (commonlogk8sauditv2_contract.ContainerIDToContainerIdentity, error) {
	result := map[string]*commonlogk8sauditv2_contract.ContainerIdentity{}
	for _, r := range results {
		for cid, s := range r {
			if current, ok := result[cid]; ok {
				result[cid] = current.Merge(s)
			} else {
				result[cid] = s
			}
		}
	}
	return result, nil
}

var _ inspectiontaskbase.InventoryMergerStrategy[commonlogk8sauditv2_contract.ContainerIDToContainerIdentity] = (*containerIDMergeStrategy)(nil)

var ContainerIDPatternFinderTask = inspectiontaskbase.NewProgressReportableInspectionTask(
	commonlogk8sauditv2_contract.ContainerIDPatternFinderTaskID,
	[]taskid.UntypedTaskReference{
		commonlogk8sauditv2_contract.ContainerIDInventoryTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (patternfinder.PatternFinder[*commonlogk8sauditv2_contract.ContainerIdentity], error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}

		cidMap := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ContainerIDInventoryTaskID.Ref())
		finder := patternfinder.NewTriePatternFinder[*commonlogk8sauditv2_contract.ContainerIdentity]()
		for cid, v := range cidMap {
			finder.AddPattern(cid, v)
		}
		return finder, nil
	},
)

var ContainerIDDiscoveryTask = commonlogk8sauditv2_contract.ContainerIDInventoryBuilder.DiscoveryTask(
	commonlogk8sauditv2_contract.ContainerIDDiscoveryTaskID,
	[]taskid.UntypedTaskReference{
		commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (commonlogk8sauditv2_contract.ContainerIDToContainerIdentity, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}

		result := commonlogk8sauditv2_contract.ContainerIDToContainerIdentity{}
		resourceLogs := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref())
		for _, group := range resourceLogs {
			if group.Resource.Type() != commonlogk8sauditv2_contract.Resource {
				continue
			}
			if group.Resource.APIVersion != "core/v1" || group.Resource.Kind != "pod" {
				continue
			}

			for _, log := range group.Logs {
				if log.ResourceBodyReader == nil {
					continue
				}
				extractContainerIDs(log.ResourceBodyReader, "status.containerStatuses", result)
				extractContainerIDs(log.ResourceBodyReader, "status.initContainerStatuses", result)
				extractContainerIDs(log.ResourceBodyReader, "status.ephemeralContainerStatuses", result)
			}
		}
		return result, nil
	},
)

func extractContainerIDs(reader *structured.NodeReader, fieldPath string, result commonlogk8sauditv2_contract.ContainerIDToContainerIdentity) {
	statusesReader, err := reader.GetReader(fieldPath)
	if err != nil {
		return
	}
	statusesReader.Children()(func(key structured.NodeChildrenKey, value structured.NodeReader) bool {
		containerID, err := value.ReadString("containerID")
		if err != nil || containerID == "" {
			return true
		}
		containerID = strings.TrimPrefix(containerID, "containerd://")
		name, _ := value.ReadString("name")
		result[containerID] = &commonlogk8sauditv2_contract.ContainerIdentity{
			ContainerID:   containerID,
			ContainerName: name,
		}
		return true
	})
}
