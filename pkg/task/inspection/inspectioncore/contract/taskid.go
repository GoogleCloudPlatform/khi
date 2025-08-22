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

package inspectioncore_contract

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

const (
	InspectionMainSubgraphName = InspectionTaskPrefix + "inspection-main"
)

var InspectionMainSubgraphInitTaskID = taskid.NewDefaultImplementationID[any](InspectionMainSubgraphName + "-init")
var InspectionMainSubgraphDoneTaskID = taskid.NewDefaultImplementationID[any](InspectionMainSubgraphName + "-done")

var InspectionTimeTaskID = taskid.NewDefaultImplementationID[time.Time](InspectionTaskPrefix + "task/time")
var TimeZoneShiftInputTaskID = taskid.NewDefaultImplementationID[*time.Location](InspectionTaskPrefix + "input-timezone-shift")
var SerializerTaskID = taskid.NewDefaultImplementationID[*FileSystemStore](InspectionTaskPrefix + "serialize")
