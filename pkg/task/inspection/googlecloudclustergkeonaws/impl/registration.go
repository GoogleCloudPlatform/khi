// Copyright 2024 Google LLC
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

package googlecloudclustergkeonaws_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudclustergkeonaws_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonaws/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// Register registers all googlecloudclustergkeonaws inspection tasks to the registry.
func Register(registry coreinspection.InspectionTaskRegistry) error {
	if err := registry.AddInspectionType(googlecloudclustergkeonaws_contract.AnthosOnAWSInspectionType); err != nil {
		return err
	}

	scoped := coreinspection.NewScopedRegistry(
		registry,
		inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{
			inspectioncore_contract.InspectionTypeLabelKeyEnvironment:       "googlecloud",
			googlecloudcommon_contract.InspectionTypeLabelKeyClusterType:    "gke_multicloud",
			googlecloudcommon_contract.InspectionTypeLabelKeyClusterSubType: "aws",
			inspectioncore_contract.InspectionTypeLabelKeyBasePlatform:      "kubernetes",
		}),
	)

	return coretask.RegisterTasks(scoped,
		AnthosOnAWSClusterNamePrefixTask,
	)
}
