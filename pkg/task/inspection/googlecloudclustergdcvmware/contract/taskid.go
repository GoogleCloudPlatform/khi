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

package googlecloudclustergdcvmware_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// ClusterGDCVMWareCommonTaskPrefix is the common task id prefix for GDC Vmware cluster related tasks.
var ClusterGDCVMWareCommonTaskPrefix = googlecloudk8scommon_contract.GoogleCloudCommonK8STaskIDPrefix + "gdc-vmware"

// ClusterNamePrefixTaskIDForGDCVForVMWare is the task ID for the GDCV for VMWare cluster name prefix.
var ClusterNamePrefixTaskIDForGDCVForVMWare = taskid.NewImplementationID(googlecloudk8scommon_contract.ClusterNamePrefixTaskID, "gdcv-for-vmware")
