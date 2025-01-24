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

package gke

import (
	"math"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
)

var InspectionTypeId = "gcp-gke"

var GKEInspectionType = inspection.InspectionType{
	Id:   InspectionTypeId,
	Name: "Google Kubernetes Engine",
	Description: `Visualize logs generated from GKE cluster. 
Supporting K8s audit log, K8s event log,K8s node log, K8s container log, GCE audit log, Networking audit log(NEG attach/detach) and autoscaler log.`,
	Icon:     "assets/icons/gke.png",
	Priority: math.MaxInt,
}
