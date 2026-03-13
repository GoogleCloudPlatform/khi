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

package googlecloudk8scommon_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// Register registers all googlecloudk8scommon inspection tasks to the registry.
func Register(registry coreinspection.InspectionTaskRegistry) error {
	for _, timelineType := range googlecloudk8scommon_contract.TimelineTypes {
		if err := registry.AddTimelineType(timelineType); err != nil {
			return err
		}
	}
	return coretask.RegisterTasks(registry,
		HeaderSuggestedFileNameTask,
		AutocompleteMetricsK8sContainerTask,
		AutocompleteMetricsK8sNodeTask,
		AutocompleteClusterIdentityTask,
		AutocompleteLocationForClusterTask,
		AutocompleteNamespacesTask,
		AutocompleteNodeNamesTask,
		AutocompletePodNamesTask,
		DefaultK8sResourceMergeConfigTask,
		ClusterIdentityTask,
		InputClusterNameTask,
		InputKindFilterTask,
		InputNamespaceFilterTask,
		InputNodeNameFilterTask,
		NEGNamesInventoryTask,
		NEGNamesDiscoveryTask,
	)
}
