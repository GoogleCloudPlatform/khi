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
)

// InspectionTypeId is the unique identifier for the GKE on Azure inspection type.
var InspectionTypeId = "gcp-gke-on-azure"

// AnthosOnAzureInspectionType defines the inspection type for GKE on Azure.
var AnthosOnAzureInspectionType = coreinspection.InspectionType{
	Id:   InspectionTypeId,
	Name: "GKE on Azure(Anthos on Azure)",
	Description: `Visualize logs generated from GKE on Azure cluster. 
Supporting K8s audit log, k8s event log,k8s node log, k8s container log and MultiCloud API audit log.`,
	Icon:     "assets/icons/anthos.png",
	Priority: math.MaxInt - 3,
}
