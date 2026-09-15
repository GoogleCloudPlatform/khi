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

package k8scontainer_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontainer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// NodeNameDiscoveryTask extracts node names from Kubernetes Container log labels and registers them to NodeNameInventoryTask.
var NodeNameDiscoveryTask = inspectiontaskbase.NewInspectionTask(
	k8scontainer.NodeNameDiscoveryTaskID,
	[]coretask.Dependency{
		k8scontainer.ListLogEntriesTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]string, error) {
		if taskMode == inspectioncore.TaskModeDryRun {
			return nil, nil
		}

		foundNodeNames := map[string]struct{}{}
		logs := coretask.GetTaskResult(ctx, k8scontainer.ListLogEntriesTaskID.Ref())
		for _, l := range logs {
			fs, err := k8scontainer.ExtractGCPContainerLogNodeNameLabel(l.NodeReader)
			if err == nil && fs.NodeName != "" {
				foundNodeNames[fs.NodeName] = struct{}{}
			}
		}

		var result []string
		for k := range foundNodeNames {
			result = append(result, k)
		}
		return result, nil
	},
	coretask.ProvidesTag(k8saudit.TagNodeNameDiscovery),
)
