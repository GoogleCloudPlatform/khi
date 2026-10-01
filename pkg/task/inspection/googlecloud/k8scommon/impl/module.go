// Copyright 2026 Google LLC
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

package k8scommon_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// Module declares the tasks shared by inspection types for Kubernetes clusters on Google Cloud, such as the cluster identity, the cluster name and resource filter forms and their autocompletes.
var Module = coreinspection.Module{
	Name: "googlecloud/k8scommon",
	Scope: coreinspection.Scope{
		inspectioncore.InspectionTypeLabelKeyEnvironment:  "googlecloud",
		inspectioncore.InspectionTypeLabelKeyBasePlatform: "kubernetes",
	},
	Tasks: []coretask.UntypedTask{
		headerSuggestedFileNameTask,
		autocompleteMetricsK8sContainerTask,
		autocompleteMetricsK8sNodeTask,
		autocompleteClusterIdentityTask,
		autocompleteLocationForClusterTask,
		autocompleteNamespacesTask,
		autocompleteNodeNamesTask,
		autocompletePodNamesTask,
		clusterIdentityTask,
		inputClusterNameTask,
		inputKindFilterTask,
		inputNamespaceFilterTask,
		inputNodeNameFilterTask,
		negNamesInventoryTask,
		negNamesDiscoveryTask,
		negToBackendServiceInventoryTask,
	},
}
