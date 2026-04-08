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

package googlecloudlogk8scontrolplane_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// Register registers all googlecloudlogk8scontrolplane inspection tasks to the registry.
/*
flowchart TD
    ListLogEntriesTask --> CommonFieldSetReadTask
    ListLogEntriesTask --> LogIngesterTask
    CommonFieldSetReadTask --> SchedulerLogFilterTask -->SchedulerFieldSetReaderTask --> SchedulerGroupterTask --> SchedulerLogToTimelineMapperTask --> TailTask
    CommonFieldSetReadTask --> ControllerManagerLogFilterTask --> ControllerManagerFieldSetReaderTask --> ControllerManagerGrouperTask --> ControllerManagerLogToTimelineMapperTask --> TailTask
    CommonFieldSetReadTask --> OtherLogFilterTask --> OtherFieldSetReaderTask --> OtherGrouperTask --> OtherLogToTimelineMapperTask --> TailTask
    LogIngesterTask --> SchedulerLogToTimelineMapperTask
    LogIngesterTask --> ControllerManagerLogToTimelineMapperTask
    LogIngesterTask --> OtherLogToTimelineMapperTask
```
*/
func Register(registry coreinspection.InspectionTaskRegistry) error {
	scopedWithLogSource := coreinspection.NewScopedRegistry(
		registry,
		inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{
			inspectioncore_contract.InspectionTypeLabelKeyLogSource:      "cloud_logging",
			inspectioncore_contract.InspectionTypeLabelKeyEnvironment:    "googlecloud",
			inspectioncore_contract.InspectionTypeLabelKeyBasePlatform:   "kubernetes",
			googlecloudcommon_contract.InspectionTypeLabelKeyClusterType: "gke",
		}),
	)
	if err := coretask.RegisterTasks(scopedWithLogSource, ListLogEntriesTask); err != nil {
		return err
	}

	scoped := coreinspection.NewScopedRegistry(
		registry,
		inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{
			inspectioncore_contract.InspectionTypeLabelKeyEnvironment:    "googlecloud",
			inspectioncore_contract.InspectionTypeLabelKeyBasePlatform:   "kubernetes",
			googlecloudcommon_contract.InspectionTypeLabelKeyClusterType: "gke",
		}),
	)
	return coretask.RegisterTasks(scoped,
		ClusterIdentityAliasTask,

		InputControlPlaneComponentNameFilterTask,
		LogIngesterTask,
		CommonFieldSetReaderTask,
		SchedulerLogFilterTask,
		SchedulerLogFieldSetReaderTask,
		SchedulerGrouperTask,
		SchedulerLogToTimelineMapperTask,
		ControllerManagerFilterTask,
		ControllerManagerLogFieldSetReaderTask,
		ControllerManagerGrouperTask,
		ControllerManagerLogToTimelineMapperTask,
		OtherLogFilterTask,
		OtherLogFieldSetReaderTask,
		OtherGrouperTask,
		OtherLogToTimelineMapperTask,
		TailTask,
	)
}
