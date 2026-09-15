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

package k8scontrolplane_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// Register registers all googlecloudlogk8scontrolplane inspection tasks to the registry.
/*
flowchart TD
    ListLogEntriesTask --> LogIngesterTask
    ListLogEntriesTask --> SchedulerLogFilterTask --> SchedulerGrouperTask --> SchedulerLogToTimelineMapperTask --> TailTask
    ListLogEntriesTask --> ControllerManagerLogFilterTask --> ControllerManagerGrouperTask --> ControllerManagerLogToTimelineMapperTask --> TailTask
    ListLogEntriesTask --> HpaControllerLogFilterTask --> HpaControllerGrouperTask --> HpaControllerLogToTimelineMapperTask --> TailTask
    ListLogEntriesTask --> OtherLogFilterTask --> OtherGrouperTask --> OtherLogToTimelineMapperTask --> TailTask
    LogIngesterTask --> SchedulerLogToTimelineMapperTask
    LogIngesterTask --> ControllerManagerLogToTimelineMapperTask
    LogIngesterTask --> HpaControllerLogToTimelineMapperTask
    LogIngesterTask --> OtherLogToTimelineMapperTask
*/
func Register(registry coreinspection.InspectionTaskRegistry) error {
	scopedWithLogSource := coreinspection.NewScopedRegistry(
		registry,
		inspectioncore.InspectionTypeLabelSelector(map[string]string{
			inspectioncore.InspectionTypeLabelKeyLogSource:    "cloud_logging",
			inspectioncore.InspectionTypeLabelKeyEnvironment:  "googlecloud",
			inspectioncore.InspectionTypeLabelKeyBasePlatform: "kubernetes",
			gcpcommon.InspectionTypeLabelKeyClusterType:       "gke",
		}),
	)
	if err := coretask.RegisterTasks(scopedWithLogSource, ListLogEntriesTask); err != nil {
		return err
	}

	scoped := coreinspection.NewScopedRegistry(
		registry,
		inspectioncore.InspectionTypeLabelSelector(map[string]string{
			inspectioncore.InspectionTypeLabelKeyEnvironment:  "googlecloud",
			inspectioncore.InspectionTypeLabelKeyBasePlatform: "kubernetes",
			gcpcommon.InspectionTypeLabelKeyClusterType:       "gke",
		}),
	)
	return coretask.RegisterTasks(scoped,
		ClusterIdentityAliasTask,

		InputControlPlaneComponentNameFilterTask,
		LogIngesterTask,
		SchedulerLogFilterTask,
		SchedulerGrouperTask,
		SchedulerLogToTimelineMapperTask,
		ControllerManagerFilterTask,
		ControllerManagerGrouperTask,
		ControllerManagerLogToTimelineMapperTask,
		HpaControllerLogFilterTask,
		HpaControllerGrouperTask,
		HpaControllerLogToTimelineMapperTask,
		OtherLogFilterTask,
		OtherGrouperTask,
		OtherLogToTimelineMapperTask,
		TailTask,
	)
}
