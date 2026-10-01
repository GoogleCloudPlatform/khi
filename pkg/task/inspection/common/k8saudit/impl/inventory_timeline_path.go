// Copyright 2026 Google LLC
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

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// TimelinePathInventoryTask aggregates timeline paths discovered from audit logs.
var TimelinePathInventoryTask = inspectiontaskbase.NewInventoryTask(
	k8saudit.TimelinePathInventoryTaskID,
	k8saudit.TagTimelinePathDiscovery,
	mergeTimelinePaths,
)

func mergeTimelinePaths(results []k8saudit.TimelinePathSet) (k8saudit.TimelinePathSet, error) {
	result := k8saudit.TimelinePathSet{}
	for _, r := range results {
		for path := range r {
			result[path] = struct{}{}
		}
	}
	return result, nil
}

// TimelinePathDiscoveryTask extracts timeline paths written by audit logs and registers them to TimelinePathInventoryTask.
var TimelinePathDiscoveryTask = inspectiontaskbase.NewInspectionTask(
	k8saudit.TimelinePathDiscoveryTaskID,
	[]coretask.Dependency{
		k8saudit.ManifestGeneratorTaskID.Ref(),
		k8saudit.K8sAuditLogExtractorRef.Ref(coretask.FromActiveGraph),
	},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8saudit.TimelinePathSet, error) {
		if taskMode == inspectioncore.TaskModeDryRun {
			return k8saudit.TimelinePathSet{}, nil
		}
		result := k8saudit.TimelinePathSet{}
		resourceLogs := coretask.GetTaskResult(ctx, k8saudit.ManifestGeneratorTaskID.Ref())
		for _, group := range resourceLogs {
			if group.Resource.Type() == k8saudit.Namespace {
				continue
			}
			for _, l := range group.Logs {
				k8sFieldSet, err := k8saudit.ExtractK8sAuditLog(ctx, l.Log.NodeReader)
				if err != nil || k8sFieldSet == nil || k8sFieldSet.IsDryRun {
					continue
				}
				targetPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, group.Resource)
				result[targetPath] = struct{}{}

				if group.Resource.Type() == k8saudit.Resource && group.Resource.APIVersion == "core/v1" && group.Resource.Kind == "pod" && l.ResourceBodyReader != nil {
					uid, foundUID := GetUID(l.ResourceBodyReader)
					nodeName, foundNode := GetNodeNameOfPod(l.ResourceBodyReader)
					if foundUID && uid != "" && foundNode && nodeName != "" {
						podPhasePath := MustPodPhaseTimelinePath(ctx, k8sFieldSet.ClusterName, nodeName, group.Resource.Namespace, group.Resource.Name, uid)
						result[podPhasePath] = struct{}{}
					}
				}
			}
		}
		return result, nil
	},
	coretask.ProvidesTag(k8saudit.TagTimelinePathDiscovery),
	coretask.WithFeatureGate(k8saudit.K8sAuditLogParserTailRef),
)
