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
	"slices"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// NodeNameInventoryTask provides list of node name found in this inspection for later task usage.
var NodeNameInventoryTask = inspectiontaskbase.NewInventoryTask(
	k8saudit.NodeNameInventoryTaskID,
	k8saudit.TagNodeNameDiscovery,
	mergeNodeNames,
)

func mergeNodeNames(results [][]string) ([]string, error) {
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
	slices.Sort(ret)
	return ret, nil
}

// NodeNameDiscoveryTask extracts node name from audit logs and node names are registered on NodeNameInventoryTask.
var NodeNameDiscoveryTask = inspectiontaskbase.NewInspectionTask(
	k8saudit.NodeNameDiscoveryTaskID,
	[]coretask.Dependency{k8saudit.ManifestGeneratorTaskID.Ref()},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]string, error) {
		if taskMode == inspectioncore.TaskModeDryRun {
			return nil, nil
		}

		foundNodeNames := map[string]struct{}{}
		resourceLogs := coretask.GetTaskResult(ctx, k8saudit.ManifestGeneratorTaskID.Ref())
		for _, group := range resourceLogs {
			if group.Resource.Type() != k8saudit.Resource {
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
	coretask.ProvidesTag(k8saudit.TagNodeNameDiscovery),
)
