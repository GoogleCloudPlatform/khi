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

package googlecloudk8scommon_impl

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
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/inspectiontype"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// HeaderSuggestedFileNameTask is a task to supply the suggested file name of the KHI file generated.
// This name is used in frontend to save the inspection data as a file.
var HeaderSuggestedFileNameTask = inspectiontaskbase.NewInspectionTask(googlecloudk8scommon_contract.HeaderSuggestedFileNameTaskID, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
	googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
	googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
}, func(ctx context.Context, taskMode inspection_contract.InspectionTaskModeType) (struct{}, error) {
	metadataSet := khictx.MustGetValue(ctx, inspection_contract.InspectionRunMetadata)
	header := typedmap.GetOrDefault(metadataSet, inspectionmetadata.HeaderMetadataKey, &inspectionmetadata.HeaderMetadata{})

	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	header.SuggestedFileName = getSuggestedFileName(clusterName, startTime, endTime)

	return struct{}{}, nil
}, inspection_contract.NewRequiredTaskLabel(), inspection_contract.InspectionTypeLabel(inspectiontype.GCPK8sClusterInspectionTypes...))

func getSuggestedFileName(clusterName string, startTime, endTime time.Time) string {
	return fmt.Sprintf("%s-%s-%s.khi", clusterName, startTime.Format("2006_01_02_1504"), endTime.Format("2006_01_02_1504"))
}
