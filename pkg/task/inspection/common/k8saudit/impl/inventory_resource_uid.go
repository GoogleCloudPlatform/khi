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

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// resourceUIDInventoryTask merges resource UID to resource identity mappings discovered across tasks.
var resourceUIDInventoryTask = inspectiontaskbase.DefineInventoryTask(
	k8saudit.ResourceUIDInventoryTaskID,
	k8saudit.TagResourceUIDDiscovery,
	mergeResourceUIDs,
)

func mergeResourceUIDs(results []k8saudit.UIDToResourceIdentity) (k8saudit.UIDToResourceIdentity, error) {
	result := map[string]*k8saudit.ResourceIdentity{}
	for _, r := range results {
		for uid, s := range r {
			result[uid] = s
		}
	}
	return result, nil
}

// resourceUIDDiscoveryTask extracts resource UIDs from resource manifests in audit logs.
var resourceUIDDiscoveryTask = inspectiontaskbase.DefineInspectionTask(
	k8saudit.ResourceUIDDiscoveryTaskID,
	func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[k8saudit.UIDToResourceIdentity] {
		resourceLogsInput := coretask.Use(b, k8saudit.ManifestGeneratorTaskID.Ref())
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8saudit.UIDToResourceIdentity, error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return k8saudit.UIDToResourceIdentity{}, nil
			}
			result := k8saudit.UIDToResourceIdentity{}
			resourceLogs := resourceLogsInput.Get(ctx)
			for _, group := range resourceLogs {
				if group.Resource.Type() != k8saudit.Resource {
					continue
				}
				for _, log := range group.Logs {
					if log.ResourceBodyReader == nil {
						continue
					}
					uid, found := GetUID(log.ResourceBodyReader)
					if !found {
						continue
					}
					result[uid] = group.Resource
				}
			}
			return result, nil
		}
	},
	coretask.ProvidesTag(k8saudit.TagResourceUIDDiscovery),
	coretask.WithFeatureGate(k8saudit.K8sAuditLogParserTailRef),
)

// uidPatternFinderTask builds a pattern finder for resource UIDs from the resource UID inventory.
var uidPatternFinderTask = inspectiontaskbase.DefineInspectionTask(
	k8saudit.ResourceUIDPatternFinderTaskID,
	func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[patternfinder.PatternFinder[*k8saudit.ResourceIdentity]] {
		uidMapInput := coretask.Use(b, k8saudit.ResourceUIDInventoryTaskID.Ref())
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (patternfinder.PatternFinder[*k8saudit.ResourceIdentity], error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return nil, nil
			}
			uidMap := uidMapInput.Get(ctx)
			finder := patternfinder.NewRadixPatternFinder[*k8saudit.ResourceIdentity]()
			for uid, resource := range uidMap {
				err := finder.AddPattern(uid, resource)
				if err != nil {
					return nil, err
				}
			}
			return finder, nil
		}
	},
)
