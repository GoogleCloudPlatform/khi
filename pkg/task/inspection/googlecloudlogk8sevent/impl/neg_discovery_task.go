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

package googlecloudlogk8sevent_impl

import (
	"context"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8sevent_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// EventLogNEGDiscoveryTask is the discovery task that extracts NEG to BackendService mappings from Kubernetes Event logs.
var EventLogNEGDiscoveryTask = googlecloudk8scommon_contract.NEGToBackendServiceInventoryBuilder.DiscoveryTask(
	googlecloudlogk8sevent_contract.NEGToBackendServiceDiscoveryTaskID,
	[]taskid.UntypedTaskReference{googlecloudlogk8sevent_contract.FieldSetReaderTaskID.Ref()},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (googlecloudk8scommon_contract.NEGToBackendServiceMap, error) {
		if taskMode != inspectioncore_contract.TaskModeRun {
			return nil, nil
		}

		logs := coretask.GetTaskResult(ctx, googlecloudlogk8sevent_contract.FieldSetReaderTaskID.Ref())
		result := make(googlecloudk8scommon_contract.NEGToBackendServiceMap)

		for _, l := range logs {
			fs, err := log.GetFieldSet(l, &googlecloudlogk8sevent_contract.KubernetesEventFieldSet{})
			if err == nil {
				neg, bs := googlecloudk8scommon_contract.ExtractNEGToBackendService(fs.Message)
				if neg != "" && bs != "" {
					result[neg] = bs
				}
			}
		}
		return result, nil
	},
)
