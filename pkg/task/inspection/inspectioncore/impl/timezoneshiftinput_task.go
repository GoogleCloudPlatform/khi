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

package inspectioncore_impl

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var TimeZoneShiftInputTask = inspectiontaskbase.NewInspectionTask(inspectioncore_contract.TimeZoneShiftInputTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (*time.Location, error) {
	req := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInput)
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
