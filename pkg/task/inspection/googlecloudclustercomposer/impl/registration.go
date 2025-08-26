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

package googlecloudclustercomposer_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
)

// Register registers all googlecloudclustercomposer inspection tasks to the registry.
func Register(registry coreinspection.InspectionTaskRegistry) error {
	err := registry.AddInspectionType(googlecloudclustercomposer_contract.ComposerInspectionType)
	if err != nil {
		return err
	}
	return coretask.RegisterTasks(registry,
		AutocompleteComposerClusterNamesTask,
		ComposerClusterNamePrefixTask,

		AutocompleteComposerEnvironmentNamesTask,
		InputComposerEnvironmentNameTask,

		ComposerSchedulerLogQueryTask,
		ComposerDagProcessorManagerLogQueryTask,
		ComposerMonitoringLogQueryTask,
		ComposerWorkerLogQueryTask,

		AirflowSchedulerLogParseTask,
		AirflowWorkerLogParseTask,
		AirflowDagProcessorLogParseTask,
	)
}
