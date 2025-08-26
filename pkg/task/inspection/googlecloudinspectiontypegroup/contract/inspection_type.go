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

package googlecloudinspectiontypegroup_contract

import (
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudclustergdcbaremetal_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergdcbaremetal/contract"
	googlecloudclustergdcvmware_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergdcvmware/contract"
	googlecloudclustergke_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergke/contract"
	googlecloudclustergkeonaws_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonaws/contract"
	googlecloudclustergkeonazure_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonazure/contract"
)

// GCPK8sClusterInspectionTypes is the list of inspection types of k8s clusters from Google Cloud.
var GCPK8sClusterInspectionTypes = []string{
	googlecloudclustergke_contract.InspectionTypeId, googlecloudclustercomposer_contract.InspectionTypeId, googlecloudclustergdcvmware_contract.InspectionTypeId, googlecloudclustergdcbaremetal_contract.InspectionTypeId, googlecloudclustergkeonaws_contract.InspectionTypeId, googlecloudclustergkeonazure_contract.InspectionTypeId,
}

// GKEBasedClusterInspectionTypes is the list of inspection types of GKE.
var GKEBasedClusterInspectionTypes = []string{
	googlecloudclustergke_contract.InspectionTypeId, googlecloudclustercomposer_contract.InspectionTypeId,
}

// GKEMultiCloudClusterInspectionTypes is the list of inspection types of GKE multicloud.
var GKEMultiCloudClusterInspectionTypes = []string{
	googlecloudclustergkeonaws_contract.InspectionTypeId, googlecloudclustergkeonazure_contract.InspectionTypeId,
}

// GDCClusterInspectionTypes is the list of inspection types of GDC clusters.
var GDCClusterInspectionTypes = []string{
	googlecloudclustergdcbaremetal_contract.InspectionTypeId, googlecloudclustergdcvmware_contract.InspectionTypeId,
}
