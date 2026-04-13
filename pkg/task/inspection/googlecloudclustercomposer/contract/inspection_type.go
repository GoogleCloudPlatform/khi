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

package googlecloudclustercomposer_contract

import (
	"math"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InspectionTypeID is the inspection type id for google cloud composer.
const InspectionTypeID = "gcp-composer"

// ComposerInspectionType is the inspection type for google cloud composer.
var ComposerInspectionType = coreinspection.InspectionType{
	Id:   InspectionTypeID,
	Name: "Cloud Composer",
	Description: `Visualize logs related to Cloud Composer environment.
Supports all GKE related logs(Cloud Composer v2) and Airflow logs(Airflow 2.0.0 or higher in any Cloud Composer version(v1-v2, partical v3))`,
	Icon:     "assets/icons/composer.webp",
	Priority: math.MaxInt - 10,
	Labels: map[string]string{
		inspectioncore_contract.InspectionTypeLabelKeyLogSource:      "cloud_logging",
		inspectioncore_contract.InspectionTypeLabelKeyEnvironment:    "googlecloud",
		inspectioncore_contract.InspectionTypeLabelKeyBasePlatform:   "kubernetes",
		googlecloudcommon_contract.InspectionTypeLabelKeyClusterType: "gke",
		googlecloudcommon_contract.InspectionTypeLabelKeyProduct:     "composer",
	},
}
