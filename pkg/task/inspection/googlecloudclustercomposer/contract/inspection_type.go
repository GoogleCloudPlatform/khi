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
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// InspectionTypeID is the inspection type id for google cloud composer.
const InspectionTypeID = "gcp-composer"

// ComposerInspectionType is the inspection type for google cloud composer.
var ComposerInspectionType = coreinspection.InspectionType{
	Id:          InspectionTypeID,
	Name:        "Managed Airflow",
	Description: "Gather and parse Managed Airflow logs to visualize environment operations on timelines.",
	Icon:        "assets/icons/composer.webp",
	Priority:    math.MaxInt - 10,
	Labels: map[string]string{
		inspectioncore.InspectionTypeLabelKeyLogSource:    "cloud_logging",
		inspectioncore.InspectionTypeLabelKeyEnvironment:  "googlecloud",
		inspectioncore.InspectionTypeLabelKeyBasePlatform: "kubernetes",
		gcpcommon.InspectionTypeLabelKeyClusterType:       "gke",
		gcpcommon.InspectionTypeLabelKeyProduct:           "composer",
	},
}
