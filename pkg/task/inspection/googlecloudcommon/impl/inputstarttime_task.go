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
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InputStartTimeTask defines an inspection task that calculates the start time of a query
// from the end time and duration.
var InputStartTimeTask = inspectiontaskbase.NewInspectionTask(googlecloudcommon_contract.InputStartTimeTaskID, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
	googlecloudcommon_contract.InputDurationTaskID.Ref(),
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (time.Time, error) {
	endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
	duration := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputDurationTaskID.Ref())
	startTime := endTime.Add(-duration)
	// Add starttime and endtime on the header metadata
	metadataSet := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)

	header, found := typedmap.Get(metadataSet, inspectionmetadata.HeaderMetadataKey)
	if !found {
		return time.Time{}, fmt.Errorf("header metadata not found")
	}

	header.StartTimeUnixSeconds = startTime.Unix()
	header.EndTimeUnixSeconds = endTime.Unix()
	return startTime, nil
})
