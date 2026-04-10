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

package googlecloudclustergdcvmware_contract

import (
	"math"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InspectionTypeID is the unique identifier for the GDCV for VMWare inspection type.
const InspectionTypeID = "gcp-gdcv-for-vmware"

// GDCVForVMWareInspectionType defines the inspection type for GDCV for VMWare.
var GDCVForVMWareInspectionType = coreinspection.InspectionType{
	Id:   InspectionTypeID,
	Name: "GDCV for VMWare (GKE on VMWare, Anthos on VMWare)",
	Description: `Visualize logs generated from GDCV for VMWare cluster (including admin cluster/user cluster).
Supporting K8s audit log, K8s event log, K8s node log, K8s container log and OnPrem API audit log.`,
	Icon:     "assets/icons/anthos.png",
	Priority: math.MaxInt - 4,
	Labels: map[string]string{
		inspectioncore_contract.InspectionTypeLabelKeyLogSource:         "cloud_logging",
		inspectioncore_contract.InspectionTypeLabelKeyEnvironment:       "googlecloud",
		googlecloudcommon_contract.InspectionTypeLabelKeyClusterType:    "gdc",
		googlecloudcommon_contract.InspectionTypeLabelKeyClusterSubType: "vmware",
		inspectioncore_contract.InspectionTypeLabelKeyBasePlatform:      "kubernetes",
	},
}
