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

package googlecloudlogk8saudit_impl

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AuditLogNEGDiscoveryTask is the discovery task that extracts NEG to BackendService mappings from Kubernetes Audit logs.
var AuditLogNEGDiscoveryTask = googlecloudk8scommon_contract.NEGToBackendServiceInventoryBuilder.DiscoveryTask(
	googlecloudlogk8saudit_contract.NEGToBackendServiceDiscoveryTaskID,
	[]taskid.UntypedTaskReference{commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref()},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (googlecloudk8scommon_contract.NEGToBackendServiceMap, error) {
		if taskMode != inspectioncore_contract.TaskModeRun {
			return nil, nil
		}

		groups := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref())
		result := make(googlecloudk8scommon_contract.NEGToBackendServiceMap)

		for _, group := range groups {
			if group.Resource == nil || group.Resource.Kind != "Pod" {
				continue
			}
			for _, mLog := range group.Logs {
				if mLog.ResourceBodyReader == nil {
					continue
				}
				conditionsReader, err := mLog.ResourceBodyReader.GetReader("status.conditions")
				if err == nil {
					conditionsReader.Children()(func(_ structured.NodeChildrenKey, conditionReader structured.NodeReader) bool {
						message := conditionReader.ReadStringOrDefault("message", "")
						neg, bs := googlecloudk8scommon_contract.ExtractNEGToBackendService(message)
						if neg != "" && bs != "" {
							result[neg] = bs
						}
						return true
					})
				}
			}
		}
		return result, nil
	},
)
