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

package inspectioncore_impl

import (
	"context"
	"time"

	common_task "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InspectionTimeProducer is a provider of inspection time.
// Tasks shouldn't use time.Now() directly to make test easier.
var InspectionTimeProducer common_task.Task[time.Time] = common_task.NewTask(inspectioncore_contract.InspectionTimeTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) (time.Time, error) {
	return time.Now(), nil
})

// TestInspectionTimeTaskProducer is a function to generate a fake InspectionTimeProducer task with the given time string.
var TestInspectionTimeTaskProducer func(timeStr string) common_task.Task[time.Time] = func(timeStr string) common_task.Task[time.Time] {
	return common_task.NewTask(inspectioncore_contract.InspectionTimeTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) (time.Time, error) {
		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return time.Time{}, err
		}
		return t, nil
	})
}
