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

package googlecloudk8scommon_contract

import (
	queryutil "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
)

// GoogleCloudCommonK8STaskIDPrefix is the prefix for common task used for K8s on Google Cloud related tasks  IDs.
var GoogleCloudCommonK8STaskIDPrefix = "cloud.google.com/k8s/"

// AutocompleteClusterNamesTaskID is the task ID reference for returning cluster name candidates as AutocompleteClusterNameList.
// Each cluster types implement their own implementation for this task reference.
var AutocompleteClusterNamesTaskID = taskid.NewTaskReference[*AutocompleteClusterNameList](GoogleCloudCommonK8STaskIDPrefix + "autocomplete/cluster-names")

// HeaderSuggestedFileNameTaskID is the task ID for the suggested file name of the inspection file included in the header metadata. This name is used for the default name of downloaded file.
var HeaderSuggestedFileNameTaskID = taskid.NewDefaultImplementationID[struct{}](GoogleCloudCommonK8STaskIDPrefix + "header-suggested-file-name")

// K8sResourceMergeConfigTaskID is the task ID for generating merge config used for merging patch requst logs to generate the manifest at the time.
var K8sResourceMergeConfigTaskID = taskid.NewDefaultImplementationID[*k8s.K8sManifestMergeConfigRegistry](GoogleCloudCommonK8STaskIDPrefix + "merge-config")

// ClusterNamePrefixTaskID is the task ID for generating the cluster name prefix used in query.
// For GKE, it's just a task to return "" always.
// For Anthos on AWS, it should return "awsClusters/" because the `resource.labels.cluster_name` field would be `awsClusters/<cluster-name>`
// For Anthos on Azure, it will be "azureClusters/"
var ClusterNamePrefixTaskID = taskid.NewTaskReference[string](GoogleCloudCommonK8STaskIDPrefix + "cluster-name-prefix")

// InputClusterNameTaskID is the task ID for the cluster name.
var InputClusterNameTaskID = taskid.NewDefaultImplementationID[string](GoogleCloudCommonK8STaskIDPrefix + "input-cluster-name")

// InputKindFilterTaskID is the task ID for the kind filter.
var InputKindFilterTaskID = taskid.NewDefaultImplementationID[*queryutil.SetFilterParseResult](GoogleCloudCommonK8STaskIDPrefix + "input-kinds")

// InputNamespaceFilterTaskID is the task ID for the namespace filter.
var InputNamespaceFilterTaskID = taskid.NewDefaultImplementationID[*queryutil.SetFilterParseResult](GoogleCloudCommonK8STaskIDPrefix + "input-namespaces")

// InputNodeNameFilterTaskID receives space splitted node names to filter node specific logs.
var InputNodeNameFilterTaskID = taskid.NewDefaultImplementationID[[]string](GoogleCloudCommonK8STaskIDPrefix + "input/node-name-filter")

// NEGNamesDiscoveryTaskID is the task ID for extracting NEG names from audit logs.
var NEGNamesDiscoveryTaskID = taskid.NewDefaultImplementationID[NEGNameToResourceIdentityMap](GoogleCloudCommonK8STaskIDPrefix + "neg-names-discovery")
