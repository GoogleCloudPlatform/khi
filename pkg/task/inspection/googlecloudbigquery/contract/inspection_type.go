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

package googlecloudbigquery_contract

import (
	"math"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
)

var InspectionTypeId = "gcp-bigquery"

var BigQueryInspectionType = coreinspection.InspectionType{
	Id:          InspectionTypeId,
	Name:        "BigQuery",
	Description: `Visualize BigQuery Job. This inspection allows you to see all BigQuery jobs whitin the specified project.`,
	Icon:        "assets/icons/composer.webp",
	Priority:    math.MaxInt - 1001,
}
