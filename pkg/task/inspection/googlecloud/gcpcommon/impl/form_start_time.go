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

package gcpcommon_impl

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// inputStartTimeTask defines an inspection task that calculates the start time of a query
// from the end time and duration.
var inputStartTimeTask = inspectiontaskbase.DefineInspectionTask(gcpcommon.InputStartTimeTaskID, func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[time.Time] {
	endTimeInput := coretask.Use(b, gcpcommon.InputEndTimeTaskID.Ref())
	durationInput := coretask.Use(b, gcpcommon.InputDurationTaskID.Ref())
	return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (time.Time, error) {
		endTime := endTimeInput.Get(ctx)
		duration := durationInput.Get(ctx)
		startTime := endTime.Add(-duration)
		// Add starttime and endtime on the header metadata
		metadataSet := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)

		header, found := typedmap.Get(metadataSet, inspectionmetadata.HeaderMetadataKey)
		if !found {
			return time.Time{}, fmt.Errorf("header metadata not found")
		}

		header.StartTimeUnixSeconds = startTime.Unix()
		header.EndTimeUnixSeconds = endTime.Unix()
		return startTime, nil
	}
})
