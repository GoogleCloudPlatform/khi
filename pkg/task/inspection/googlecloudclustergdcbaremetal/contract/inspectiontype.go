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

package googlecloudclustergdcbaremetal_contract

import (
	"math"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InspectionTypeID is the unique identifier for the GDCV for Baremetal inspection type.
const InspectionTypeID = "gcp-gdcv-for-baremetal"

// GDCVForBaremetalInspectionType defines the inspection type for GDCV for Baremetal.
var GDCVForBaremetalInspectionType = coreinspection.InspectionType{
	Id:          InspectionTypeID,
	Name:        "GDCV for Baremetal (GKE on Baremetal, Anthos on Baremetal)",
	Description: `Gather and parse Google Distributed Cloud Virtual (GDCV) for Baremetal cluster logs (including user, admin, hybrid, or standalone clusters; Kubernetes audit, event, node, container, and On-Premises API audit logs) to visualize cluster operations on timelines. This type also supports GCDE and GDCH clusters.`,
	Icon:        "assets/icons/anthos.png",
	Priority:    math.MaxInt - 3,
	Labels: map[string]string{
		inspectioncore_contract.InspectionTypeLabelKeyLogSource:         "cloud_logging",
		inspectioncore_contract.InspectionTypeLabelKeyEnvironment:       "googlecloud",
		inspectioncore_contract.InspectionTypeLabelKeyBasePlatform:      "kubernetes",
		googlecloudcommon_contract.InspectionTypeLabelKeyClusterType:    "gdc",
		googlecloudcommon_contract.InspectionTypeLabelKeyClusterSubType: "baremetal",
	},
}
