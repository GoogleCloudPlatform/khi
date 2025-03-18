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

package composer_inspection_type

import (
	"math"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
)

var InspectionTypeId = "gcp-composer"

var ComposerInspectionType = inspection.InspectionType{
	Id:   InspectionTypeId,
	Name: "Cloud Composer",
	Description: `Visualize logs related to Cloud Composer environment.
Supports all GKE related logs(Cloud Composer v2 or v1) and Airflow logs(Airflow 2.0.0 or higher in any Cloud Composer version(v1-v3))`,
	Icon:     "assets/icons/composer.webp",
	Priority: math.MaxInt - 1,
}
