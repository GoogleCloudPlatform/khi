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

package googlecloudclustergke_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

// ClusterGKETaskCommonPrefix is the task id prefix originally defined in googlecloudclustergke.
var ClusterGKETaskCommonPrefix = gcpcommon.GoogleCloudCommonTaskIDPrefix + "cluster/gke/"

// ClusterNamePrefixTaskIDForGKE is the task ID for the GKE cluster name prefix(it's "" for GKE)
var ClusterNamePrefixTaskIDForGKE = taskid.NewImplementationID(k8scommon.ClusterNamePrefixTaskRef, "gke")

// AutocompleteMetricsK8sContainerTaskIDForGKE is the task ID for the metrics type used for autocomplete cluster names in GKE.
var AutocompleteMetricsK8sContainerTaskIDForGKE = taskid.NewImplementationID(k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref(), "gke")

// AutocompleteMetricsK8sNodeTaskIDForGKE is the task ID for the metrics type used for autocomplete cluster names in GKE.
var AutocompleteMetricsK8sNodeTaskIDForGKE = taskid.NewImplementationID(k8scommon.AutocompleteMetricsK8sNodeTaskID.Ref(), "gke")
