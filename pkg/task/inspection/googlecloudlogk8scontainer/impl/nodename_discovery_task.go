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

package googlecloudlogk8scontainer_impl

import (
	"context"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// NodeNameDiscoveryTask extracts node names from Kubernetes Container log labels and registers them to NodeNameInventoryTask.
var NodeNameDiscoveryTask = commonlogk8saudit_contract.NodeNameInventoryBuilder.DiscoveryTask(
	googlecloudlogk8scontainer_contract.NodeNameDiscoveryTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudlogk8scontainer_contract.FieldSetReaderTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]string, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}

		foundNodeNames := map[string]struct{}{}
		logs := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.FieldSetReaderTaskID.Ref())
		for _, l := range logs {
			fs, err := log.GetFieldSet(l, &googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{})
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
)
