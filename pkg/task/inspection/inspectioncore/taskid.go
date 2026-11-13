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

package inspectioncore

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// InspectionTimeTaskID is the task ID for obtaining the inspection execution timestamp.
var InspectionTimeTaskID = taskid.NewDefaultImplementationID[time.Time](InspectionTaskPrefix + "task/time")

// TimeZoneShiftInputTaskID is the task ID for the timezone shift input form task.
var TimeZoneShiftInputTaskID = taskid.NewDefaultImplementationID[*time.Location](InspectionTaskPrefix + "input-timezone-shift")

// InputInspectionNameTaskID is the task ID for the inspection name input form task.
var InputInspectionNameTaskID = taskid.NewDefaultImplementationID[string](InspectionTaskPrefix + "input/inspection-name")

// SerializerTaskID is the task ID for serializing the inspection result into a FileSystemStore.
var SerializerTaskID = taskid.NewDefaultImplementationID[*FileSystemStore](InspectionTaskPrefix + "serialize")
