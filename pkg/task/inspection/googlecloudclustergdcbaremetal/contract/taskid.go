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

package googlecloudclustergdcbaremetal_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// AutocompleteGDCVForBaremetalClusterNamesTaskID is the task ID for listing up GDCV for Baremetal cluster names on the project.
var AutocompleteGDCVForBaremetalClusterNamesTaskID = taskid.NewImplementationID(googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID, "anthos-on-baremetal")

// ClusterNamePrefixTaskIDForGDCVForBaremetal is the task ID for the GDCV for Baremetal cluster name prefix.
var ClusterNamePrefixTaskIDForGDCVForBaremetal = taskid.NewImplementationID(googlecloudk8scommon_contract.ClusterNamePrefixTaskID, "gdcv-for-baremetal")
