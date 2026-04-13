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

package ossclusterk8s_contract

import (
	"math"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InspectionTypeID is the unique identifier for the OSS Kubernetes log files inspection type.
const InspectionTypeID = "oss-kubernetes-from-files"

// OSSKubernetesLogFilesInspectionType defines the inspection type for OSS Kubernetes logs.
var OSSKubernetesLogFilesInspectionType = coreinspection.InspectionType{
	Id:          InspectionTypeID,
	Name:        "OSS Kubernetes Log Files",
	Description: "Visualize OSS Kubernetes logs through the uploaded files",
	Icon:        "assets/icons/k8s.png",
	Priority:    math.MaxInt - 1000,
	Labels: map[string]string{
		inspectioncore_contract.InspectionTypeLabelKeyLogSource:    "file",
		inspectioncore_contract.InspectionTypeLabelKeyEnvironment:  "oss",
		inspectioncore_contract.InspectionTypeLabelKeyBasePlatform: "kubernetes",
	},
}
