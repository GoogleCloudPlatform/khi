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

package baremetal

import (
	"math"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
)

var InspectionTypeId = "gcp-gdcv-for-baremetal"

var AnthosOnBaremetalInspectionType = inspection.InspectionType{
	Id:   InspectionTypeId,
	Name: "GDCV for Baremetal(GKE on Baremetal, Anthos on Baremetal)",
	Description: `Visualize logs generated from GDCV for baremetal cluster(including user cluster/admin cluster/hybrid cluster or standalone cluster).
Supporting K8s audit log, k8s event log,k8s node log, k8s container log and OnPream API audit log.

This type can also be used for GCDE or GDCH.`,
	Icon:     "assets/icons/anthos.png",
	Priority: math.MaxInt - 3,
}
