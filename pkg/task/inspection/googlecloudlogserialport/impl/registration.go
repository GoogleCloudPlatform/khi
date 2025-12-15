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

package googlecloudlogserialport_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

// Register registers all googlecloudlogserialport inspection tasks to the registry.
/*
flowchart TD
    LogQueryTask
    LogFilterTask
    FieldSetReadTask
    LogIngesterTask
    LogGrouperTask
    LogToTimelineMapperTask

    LogQueryTask --> FieldSetReadTask
    FieldSetReadTask --> LogFilterTask
    LogFilterTask --> LogIngesterTask
    LogFilterTask --> LogGrouperTask
    LogGrouperTask --> LogToTimelineMapperTask
    LogIngesterTask --> LogToTimelineMapperTask
*/
func Register(registry coreinspection.InspectionTaskRegistry) error {
	return coretask.RegisterTasks(registry,
		LogQueryTask,
		FieldSetReadTask,
		LogIngesterTask,
		LogGrouperTask,
		LogFilterTask,
		LogToTimelineMapperTask,
	)
}
