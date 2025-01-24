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

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var TimeZoneShiftInputTaskID = GCPPrefix + "input/timezone-shift"

var TimeZoneShiftInputTask = inspection_task.NewInspectionProcessor(TimeZoneShiftInputTaskID, []string{}, func(ctx context.Context, taskMode int, v *task.VariableSet, progress *progress.TaskProgress) (any, error) {
	req, err := inspection_task.GetInspectionRequestFromVariable(v)
	if err != nil {
		return nil, err
	}
	if tzShiftAny, found := req.Values["timezoneShift"]; found {
		if tzShiftFloat, convertible := tzShiftAny.(float64); convertible {
			return time.FixedZone("Unknown", int(tzShiftFloat*3600)), nil
		} else {
			return time.UTC, nil
		}
	} else {
		return time.UTC, nil
	}
})

func GetTimezoneShiftInput(tv *task.VariableSet) (*time.Location, error) {
	return task.GetTypedVariableFromTaskVariable[*time.Location](tv, TimeZoneShiftInputTaskID, time.UTC)
}
