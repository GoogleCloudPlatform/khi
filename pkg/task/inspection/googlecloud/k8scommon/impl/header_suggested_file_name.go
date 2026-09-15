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

package k8scommon_impl

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

	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// HeaderSuggestedFileNameTask is a task to supply the suggested file name of the KHI file generated.
// This name is used in frontend to save the inspection data as a file.
var HeaderSuggestedFileNameTask = inspectiontaskbase.NewInspectionTask(k8scommon.HeaderSuggestedFileNameTaskID, []coretask.Dependency{
	gcpcommon.InputStartTimeTaskID.Ref(),
	gcpcommon.InputEndTimeTaskID.Ref(),
	k8scommon.InputClusterNameTaskID.Ref(),
}, func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (struct{}, error) {
	metadataSet := khictx.MustGetValue(ctx, inspectioncore.InspectionRunMetadata)
	header := typedmap.GetOrDefault(metadataSet, inspectionmetadata.HeaderMetadataKey, &inspectionmetadata.HeaderMetadata{})

	clusterName := coretask.GetTaskResult(ctx, k8scommon.InputClusterNameTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, gcpcommon.InputEndTimeTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref())
	header.SuggestedFileName = getSuggestedFileName(clusterName, startTime, endTime)

	return struct{}{}, nil
}, coretask.NewRequiredTaskLabel())

func getSuggestedFileName(clusterName string, startTime, endTime time.Time) string {
	return fmt.Sprintf("%s-%s-%s.khi", clusterName, startTime.Format("2006_01_02_1504"), endTime.Format("2006_01_02_1504"))
}
