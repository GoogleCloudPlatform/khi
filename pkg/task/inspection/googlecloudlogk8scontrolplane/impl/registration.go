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
)

// Register registers all googlecloudlogk8scontrolplane inspection tasks to the registry.
/*
flowchart TD
    ListLogEntriesTask --> CommonFieldSetReadTask
    ListLogEntriesTask --> LogSerializerTask
    CommonFieldSetReadTask --> SchedulerLogFilterTask -->SchedulerFieldSetReaderTask --> SchedulerGroupterTask --> SchedulerHistoryModifierTask --> TailTask
    CommonFieldSetReadTask --> ControllerManagerLogFilterTask --> ControllerManagerFieldSetReaderTask --> ControllerManagerGrouperTask --> ControllerManagerHistoryModifierTask --> TailTask
    CommonFieldSetReadTask --> OtherLogFilterTask --> OtherFieldSetReaderTask --> OtherGrouperTask --> OtherHistoryModifierTask --> TailTask
    LogSerializerTask --> SchedulerHistoryModifierTask
    LogSerializerTask --> ControllerManagerHistoryModifierTask
    LogSerializerTask --> OtherHistoryModifierTask
*/
func Register(registry coreinspection.InspectionTaskRegistry) error {
	return coretask.RegisterTasks(registry,
		InputControlPlaneComponentNameFilterTask,
		ListLogEntriesTask,
		LogSerializerTask,
		CommonFieldSetReaderTask,
		SchedulerLogFilterTask,
		SchedulerLogFieldSetReaderTask,
		SchedulerGrouperTask,
		SchedulerHistoryModifierTask,
		ControllerManagerFilterTask,
		ControllerManagerLogFieldSetReaderTask,
		ControllerManagerGrouperTask,
		ControllerManagerHistoryModifierTask,
		OtherLogFilterTask,
		OtherLogFieldSetReaderTask,
		OtherGrouperTask,
		OtherHistoryModifierTask,
		TailTask,
	)
}
