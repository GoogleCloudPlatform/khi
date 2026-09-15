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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	commonk8saudit "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8saudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

var (
	pathStatusConditions = structured.CompileFieldPath("status.conditions")
	pathMessage          = structured.CompileFieldPath("message")
)

// AuditLogNEGDiscoveryTask is the discovery task that extracts NEG to BackendService mappings from Kubernetes Audit logs.
var AuditLogNEGDiscoveryTask = inspectiontaskbase.NewInspectionTask(
	googlecloudlogk8saudit_contract.NEGToBackendServiceDiscoveryTaskID,
	[]coretask.Dependency{commonk8saudit.ManifestGeneratorTaskID.Ref()},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (googlecloudk8scommon_contract.NEGToBackendServiceMap, error) {
		if taskMode != inspectioncore.TaskModeRun {
			return nil, nil
		}

		groups := coretask.GetTaskResult(ctx, commonk8saudit.ManifestGeneratorTaskID.Ref())
		result := make(googlecloudk8scommon_contract.NEGToBackendServiceMap)

		for _, group := range groups {
			if group.Resource == nil || strings.ToLower(group.Resource.Kind) != "pod" {
				continue
			}
			for _, mLog := range group.Logs {
				if mLog.ResourceBodyReader == nil {
					continue
				}
				conditionsReader, err := mLog.ResourceBodyReader.GetReader(pathStatusConditions)
				if err == nil {
					conditionsReader.Children()(func(_ structured.NodeChildrenKey, conditionReader structured.NodeReader) bool {
						message := conditionReader.ReadStringOrDefault(pathMessage, "")
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
	coretask.ProvidesTag(googlecloudk8scommon_contract.TagNEGToBackendServiceDiscovery),
)
