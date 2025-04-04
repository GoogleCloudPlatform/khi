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

package inspectiontype

import (
	composer_inspection_type "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/inspectiontype"
	baremetal "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gdcv-for-baremetal"
	vmware "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gdcv-for-vmware"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke"
	aws "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke-on-aws"
	azure "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke-on-azure"
)

// GCPK8sClusterInspectionTypes is the list of inspection types of k8s clusters from Google Cloud.
var GCPK8sClusterInspectionTypes = []string{
	gke.InspectionTypeId, composer_inspection_type.InspectionTypeId, vmware.InspectionTypeId, baremetal.InspectionTypeId, aws.InspectionTypeId, azure.InspectionTypeId,
}

// GKEBasedClusterInspectionTypes is the list of inspection types of GKE.
var GKEBasedClusterInspectionTypes = []string{
	gke.InspectionTypeId, composer_inspection_type.InspectionTypeId,
}

// GKEMultiCloudClusterInspectionTypes is the list of inspection types of GKE multicloud.
var GKEMultiCloudClusterInspectionTypes = []string{
	aws.InspectionTypeId, azure.InspectionTypeId,
}

// GDCClusterInspectionTypes is the list of inspection types of GDC clusters.
var GDCClusterInspectionTypes = []string{
	baremetal.InspectionTypeId, vmware.InspectionTypeId,
}
