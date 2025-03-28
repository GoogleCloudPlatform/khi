// Copyright 2024 Google LLC
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

package task

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var TimeZoneShiftInputTaskID = taskid.NewDefaultImplementationID[*time.Location](GCPPrefix + "input/timezone-shift")

var TimeZoneShiftInputTask = inspection_task.NewInspectionTask(TimeZoneShiftInputTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) (*time.Location, error) {
	req := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionTaskInput)
	if tzShiftAny, found := req["timezoneShift"]; found {
		if tzShiftFloat, convertible := tzShiftAny.(float64); convertible {
			return time.FixedZone("Unknown", int(tzShiftFloat*3600)), nil
		} else {
			return time.UTC, nil
		}
	} else {
		return time.UTC, nil
	}
})
