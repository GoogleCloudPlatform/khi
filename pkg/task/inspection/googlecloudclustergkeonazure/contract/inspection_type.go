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

package googlecloudclustergkeonazure_contract

import (
	"math"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// InspectionTypeID is the unique identifier for the GKE on Azure inspection type.
const InspectionTypeID = "gcp-gke-on-azure"

// AnthosOnAzureInspectionType defines the inspection type for GKE on Azure.
var AnthosOnAzureInspectionType = coreinspection.InspectionType{
	Id:          InspectionTypeID,
	Name:        "GKE on Azure (Anthos on Azure)",
	Description: `Gather and parse GKE on Azure cluster logs (Kubernetes audit, event, node, container logs, and Multi-Cloud API audit logs) to visualize cluster operations on timelines.`,
	Icon:        "assets/icons/anthos.png",
	Priority:    math.MaxInt - 3,
	Labels: map[string]string{
		inspectioncore.InspectionTypeLabelKeyLogSource:    "cloud_logging",
		inspectioncore.InspectionTypeLabelKeyEnvironment:  "googlecloud",
		inspectioncore.InspectionTypeLabelKeyBasePlatform: "kubernetes",
		gcpcommon.InspectionTypeLabelKeyClusterType:       "gke_multicloud",
		gcpcommon.InspectionTypeLabelKeyClusterSubType:    "azure",
	},
}
