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

package googlecloudclustergkeonaws_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// ClusterGKEOnAWSTaskCommonPrefix is the task id prefix defined in googlecloudclustergkeonaws.
var ClusterGKEOnAWSTaskCommonPrefix = googlecloudk8scommon_contract.GoogleCloudCommonK8STaskIDPrefix + "cluster/gke-on-aws/"

// AutocompleteGKEOnAWSClusterNamesTaskID is the task ID for listing up GKE on AWS cluster names.
var AutocompleteGKEOnAWSClusterNamesTaskID = taskid.NewImplementationID(googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID, "anthos-on-aws")

// ClusterNamePrefixTaskID is the task ID for the GKE on AWS cluster name prefix.
var ClusterNamePrefixTaskID = taskid.NewImplementationID(googlecloudk8scommon_contract.ClusterNamePrefixTaskID, "gke-on-aws")

// ClusterListFetcherTaskID is the task ID for getting ClusterListFetcher interface.
var ClusterListFetcherTaskID = taskid.NewDefaultImplementationID[ClusterListFetcher](ClusterGKEOnAWSTaskCommonPrefix + "cluster-list-fetcher")
