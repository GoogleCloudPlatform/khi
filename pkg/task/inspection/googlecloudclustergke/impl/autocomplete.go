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

package googlecloudclustergke_impl

import (
	"context"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustergke_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergke/contract"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AutocompleteMetricsK8sContainerTask returns the metrics type used for autocomplete cluster names in GKE.
// The metrics type "kubernetes.io/container/uptime" is used for GKE instead of the default "kubernetes.io/anthos/container/uptime".
var AutocompleteMetricsK8sContainerTask = coretask.NewTask(googlecloudclustergke_contract.AutocompleteMetricsK8sContainerTaskIDForGKE, []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	return "kubernetes.io/container/uptime", nil
}, coretask.WithSelectionPriority(1000), inspectioncore_contract.InspectionTypeLabel(googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes...))

// AutocompleteMetricsK8sNodeTask returns the metrics type used for autocomplete node names in GKE.
// The metrics type "kubernetes.io/node/cpu/total_cores" is used for GKE instead of the default "kubernetes.io/anthos/up".
var AutocompleteMetricsK8sNodeTask = coretask.NewTask(googlecloudclustergke_contract.AutocompleteMetricsK8sNodeTaskIDForGKE, []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	return "kubernetes.io/node/cpu/total_cores", nil
}, coretask.WithSelectionPriority(1000), inspectioncore_contract.InspectionTypeLabel(googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes...))
