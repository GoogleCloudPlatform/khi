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
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// Register registers all googlecloudclustercomposer inspection tasks to the registry.
func Register(registry coreinspection.InspectionTaskRegistry) error {
	if err := registry.AddInspectionType(googlecloudclustercomposer_contract.ComposerInspectionType); err != nil {
		return err
	}

	scopedCloudLogging := coreinspection.NewScopedRegistry(
		registry,
		inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{
			inspectioncore_contract.InspectionTypeLabelKeyLogSource:      "cloud_logging",
			googlecloudcommon_contract.InspectionTypeLabelKeyProduct:     "composer",
			inspectioncore_contract.InspectionTypeLabelKeyEnvironment:    "googlecloud",
			googlecloudcommon_contract.InspectionTypeLabelKeyClusterType: "gke",
			inspectioncore_contract.InspectionTypeLabelKeyBasePlatform:   "kubernetes",
		}),
	)

	if err := coretask.RegisterTasks(scopedCloudLogging, ComposerLogsQueryTask); err != nil {
		return err
	}

	scopedAll := coreinspection.NewScopedRegistry(
		registry,
		inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{
			googlecloudcommon_contract.InspectionTypeLabelKeyProduct:     "composer",
			inspectioncore_contract.InspectionTypeLabelKeyEnvironment:    "googlecloud",
			googlecloudcommon_contract.InspectionTypeLabelKeyClusterType: "gke",
			inspectioncore_contract.InspectionTypeLabelKeyBasePlatform:   "kubernetes",
		}),
	)

	return coretask.RegisterTasks(scopedAll,
		ClusterIdentityAliasTask,

		ComposerEnvironmentListFetcherTask,
		ComposerEnvironmentClusterFinderTask,

		AutocompleteComposerClusterNamesTask,

		AutocompleteComposerEnvironmentIdentityTask,
		AutocompleteLocationForComposerEnvironmentTask,
		InputComposerEnvironmentNameTask,

		AutocompleteComposerComponentsTask,
		InputComposerComponentsTask,

		AirflowSchedulerLogFilterTask,
		AirflowSchedulerLogGrouperTask,
		AirflowSchedulerLogIngesterTask,
		AirflowSchedulerLogToTimelineMapperTask,

		AirflowWorkerLogFilterTask,
		AirflowWorkerLogGrouperTask,
		AirflowWorkerLogIngesterTask,
		AirflowWorkerLogToTimelineMapperTask,

		AirflowOtherLogFilterTask,
		AirflowOtherLogGrouperTask,
		AirflowOtherLogIngesterTask,
		AirflowOtherLogToTimelineMapperTask,

		AirflowDagProcessorManagerLogFilterTask,
		AirflowDagProcessorManagerLogSorterTask,
		AirflowDagProcessorManagerLogGrouperTask,
		AirflowDagProcessorManagerLogIngesterTask,
		AirflowDagProcessorManagerLogToTimelineMapperTask,

		ComposerLogsFieldSetReadTask,
		ComposerLogsTailTask,
	)
}
