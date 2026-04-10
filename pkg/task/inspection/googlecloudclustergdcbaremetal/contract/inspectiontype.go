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
	Id:   InspectionTypeID,
	Name: "GDCV for Baremetal (GKE on Baremetal, Anthos on Baremetal)",
	Description: `Visualize logs generated from GDCV for baremetal cluster (including user, admin, hybrid, or standalone clusters).
Supporting K8s audit log, K8s event log, K8s node log, K8s container log and OnPrem API audit log.

This type can also be used for GCDE or GDCH.`,
	Icon:     "assets/icons/anthos.png",
	Priority: math.MaxInt - 3,
	Labels: map[string]string{
		inspectioncore_contract.InspectionTypeLabelKeyLogSource:         "cloud_logging",
		inspectioncore_contract.InspectionTypeLabelKeyEnvironment:       "googlecloud",
		googlecloudcommon_contract.InspectionTypeLabelKeyClusterType:    "gdc",
		googlecloudcommon_contract.InspectionTypeLabelKeyClusterSubType: "baremetal",
		inspectioncore_contract.InspectionTypeLabelKeyBasePlatform:      "kubernetes",
	},
}
