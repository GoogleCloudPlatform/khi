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

package k8saudit_impl

import (
	"context"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// containerIDInventoryTask merges container ID to container identity mappings discovered across tasks.
var containerIDInventoryTask = inspectiontaskbase.DefineInventoryTask(
	k8saudit.ContainerIDInventoryTaskID,
	k8saudit.TagContainerIDDiscovery,
	mergeContainerIDs,
)

func mergeContainerIDs(results []k8saudit.ContainerIDToContainerIdentity) (k8saudit.ContainerIDToContainerIdentity, error) {
	result := map[string]*k8saudit.ContainerIdentity{}
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

// containerIDPatternFinderTask builds a pattern finder for container IDs from the container ID inventory.
var containerIDPatternFinderTask = inspectiontaskbase.DefineInspectionTask(
	k8saudit.ContainerIDPatternFinderTaskID,
	func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[patternfinder.PatternFinder[*k8saudit.ContainerIdentity]] {
		cidMapInput := coretask.Use(b, k8saudit.ContainerIDInventoryTaskID.Ref())
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (patternfinder.PatternFinder[*k8saudit.ContainerIdentity], error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return nil, nil
			}

			cidMap := cidMapInput.Get(ctx)
			finder := patternfinder.NewRadixPatternFinder[*k8saudit.ContainerIdentity]()
			for cid, v := range cidMap {
				finder.AddPattern(cid, v)
			}
			return finder, nil
		}
	},
)

var (
	pathContainerStatuses          = structured.CompileFieldPath("status.containerStatuses")
	pathInitContainerStatuses      = structured.CompileFieldPath("status.initContainerStatuses")
	pathEphemeralContainerStatuses = structured.CompileFieldPath("status.ephemeralContainerStatuses")
	pathContainerID                = structured.CompileFieldPath("containerID")
	pathContainerName              = structured.CompileFieldPath("name")
)

// containerIDDiscoveryTask extracts container IDs from pod manifests in audit logs.
var containerIDDiscoveryTask = inspectiontaskbase.DefineInspectionTask(
	k8saudit.ContainerIDDiscoveryTaskID,
	func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[k8saudit.ContainerIDToContainerIdentity] {
		resourceLogsInput := coretask.Use(b, k8saudit.ManifestGeneratorTaskID.Ref())
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8saudit.ContainerIDToContainerIdentity, error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return nil, nil
			}

			result := k8saudit.ContainerIDToContainerIdentity{}
			resourceLogs := resourceLogsInput.Get(ctx)
			for _, group := range resourceLogs {
				if group.Resource.Type() != k8saudit.Resource {
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
		}
	},
	coretask.ProvidesTag(k8saudit.TagContainerIDDiscovery),
	coretask.WithFeatureGate(k8saudit.K8sAuditLogParserTailRef),
)

func extractContainerIDs(reader *structured.NodeReader, fieldPath structured.FieldPath, result k8saudit.ContainerIDToContainerIdentity) {
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
		result[containerID] = &k8saudit.ContainerIdentity{
			ContainerID:   containerID,
			ContainerName: name,
		}
		return true
	})
}
