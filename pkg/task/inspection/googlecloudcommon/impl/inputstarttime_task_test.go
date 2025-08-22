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

package googlecloudcommon_impl

import (
	"context"
	"testing"
	"time"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func TestInputStartTime(t *testing.T) {
	duration, err := time.ParseDuration("1h30m")
	if err != nil {
		t.Fatal(err)
	}
	endTime, err := time.Parse(time.RFC3339, "2023-01-02T15:45:00Z")
	if err != nil {
		t.Fatal(err)
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
	startTime, _, err := inspectiontest.RunInspectionTask(ctx, InputStartTimeTask, inspectioncore_contract.TaskModeDryRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputDurationTaskID.Ref(), duration),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputEndTimeTaskID.Ref(), endTime),
		tasktest.NewTaskDependencyValuePair(inspectioncore_contract.TimeZoneShiftInputTaskID.Ref(), time.UTC),
	)
	if err != nil {
		t.Errorf("unexpected error\n%v", err)
	}
	expectedTime, err := time.Parse(time.RFC3339, "2023-01-02T14:15:00Z")
	if err != nil {
		t.Errorf("unexpected error\n%v", err)
	}

	if startTime.String() != expectedTime.String() {
		t.Errorf("returned time is not matching with the expected value\n%s", startTime)
	}
}
