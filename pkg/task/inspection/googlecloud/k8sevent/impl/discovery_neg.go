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

package k8sevent_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8sevent"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// EventLogNEGDiscoveryTask is the discovery task that extracts NEG to BackendService mappings from Kubernetes Event logs.
var EventLogNEGDiscoveryTask = inspectiontaskbase.NewInspectionTask(
	k8sevent.NEGToBackendServiceDiscoveryTaskID,
	[]coretask.Dependency{k8sevent.ListLogEntriesTaskID.Ref()},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8scommon.NEGToBackendServiceMap, error) {
		if taskMode != inspectioncore.TaskModeRun {
			return nil, nil
		}

		logs := coretask.GetTaskResult(ctx, k8sevent.ListLogEntriesTaskID.Ref())
		result := make(k8scommon.NEGToBackendServiceMap)

		for _, l := range logs {
			fs, err := k8sevent.ExtractKubernetesEvent(l.NodeReader)
			if err == nil {
				neg, bs := k8scommon.ExtractNEGToBackendService(fs.Message)
				if neg != "" && bs != "" {
					result[neg] = bs
				}
			}
		}
		return result, nil
	},
	coretask.ProvidesTag(k8scommon.TagNEGToBackendServiceDiscovery),
)
