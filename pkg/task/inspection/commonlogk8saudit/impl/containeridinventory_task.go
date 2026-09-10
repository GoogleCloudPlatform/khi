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

package commonlogk8saudit_impl

import (
	"context"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var ContainerIDInventoryTask = inspectiontaskbase.NewInventoryTask(
	commonlogk8saudit_contract.ContainerIDInventoryTaskID,
	commonlogk8saudit_contract.TagContainerIDDiscovery,
	mergeContainerIDs,
)

func mergeContainerIDs(results []commonlogk8saudit_contract.ContainerIDToContainerIdentity) (commonlogk8saudit_contract.ContainerIDToContainerIdentity, error) {
	result := map[string]*commonlogk8saudit_contract.ContainerIdentity{}
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

var ContainerIDPatternFinderTask = inspectiontaskbase.NewProgressReportableInspectionTask(
	commonlogk8saudit_contract.ContainerIDPatternFinderTaskID,
	[]coretask.Dependency{
		commonlogk8saudit_contract.ContainerIDInventoryTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (patternfinder.PatternFinder[*commonlogk8saudit_contract.ContainerIdentity], error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}

		cidMap := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ContainerIDInventoryTaskID.Ref())
		finder := patternfinder.NewRadixPatternFinder[*commonlogk8saudit_contract.ContainerIdentity]()
		for cid, v := range cidMap {
			finder.AddPattern(cid, v)
		}
		return finder, nil
	},
)

var (
	pathContainerStatuses          = structured.CompileFieldPath("status.containerStatuses")
	pathInitContainerStatuses      = structured.CompileFieldPath("status.initContainerStatuses")
	pathEphemeralContainerStatuses = structured.CompileFieldPath("status.ephemeralContainerStatuses")
	pathContainerID                = structured.CompileFieldPath("containerID")
	pathContainerName              = structured.CompileFieldPath("name")
)

var ContainerIDDiscoveryTask = inspectiontaskbase.NewInspectionTask(
	commonlogk8saudit_contract.ContainerIDDiscoveryTaskID,
	[]coretask.Dependency{
		commonlogk8saudit_contract.ManifestGeneratorTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (commonlogk8saudit_contract.ContainerIDToContainerIdentity, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}

		result := commonlogk8saudit_contract.ContainerIDToContainerIdentity{}
		resourceLogs := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ManifestGeneratorTaskID.Ref())
		for _, group := range resourceLogs {
			if group.Resource.Type() != commonlogk8saudit_contract.Resource {
				continue
			}
			if group.Resource.APIVersion != "core/v1" || group.Resource.Kind != "pod" {
				continue
			}

			for _, log := range group.Logs {
				if log.ResourceBodyReader == nil {
					continue
				}
				extractContainerIDs(log.ResourceBodyReader, pathContainerStatuses, result)
				extractContainerIDs(log.ResourceBodyReader, pathInitContainerStatuses, result)
				extractContainerIDs(log.ResourceBodyReader, pathEphemeralContainerStatuses, result)
			}
		}
		return result, nil
	},
	coretask.ProvidesTag(commonlogk8saudit_contract.TagContainerIDDiscovery),
)

func extractContainerIDs(reader *structured.NodeReader, fieldPath structured.FieldPath, result commonlogk8saudit_contract.ContainerIDToContainerIdentity) {
	statusesReader, err := reader.GetReader(fieldPath)
	if err != nil {
		return
	}
	statusesReader.Children()(func(key structured.NodeChildrenKey, value structured.NodeReader) bool {
		containerID, err := value.ReadString(pathContainerID)
		if err != nil || containerID == "" {
			return true
		}
		containerID = strings.TrimPrefix(containerID, "containerd://")
		name, _ := value.ReadString(pathContainerName)
		result[containerID] = &commonlogk8saudit_contract.ContainerIdentity{
			ContainerID:   containerID,
			ContainerName: name,
		}
		return true
	})
}
