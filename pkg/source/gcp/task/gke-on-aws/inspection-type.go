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

package aws

import (
	"math"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
)

var InspectionTypeId = "gcp-gke-on-aws"

var AnthosOnAWSInspectionType = inspection.InspectionType{
	Id:   InspectionTypeId,
	Name: "GKE on AWS(Anthos on AWS)",
	Description: `Visualize logs generated from GKE on AWS cluster. 
Supporting K8s audit log, k8s event log,k8s node log, k8s container log and MultiCloud API audit log.`,
	Icon:     "assets/icons/anthos.png",
	Priority: math.MaxInt - 2,
}
