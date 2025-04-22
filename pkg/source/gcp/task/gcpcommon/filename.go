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

package gcpcommon

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/inspectiontype"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var HeaderSuggestedFileNameTaskID = taskid.NewDefaultImplementationID[struct{}]("header-suggested-file-name")

// HeaderSuggestedFileNameTask is a task to supply the suggested file name of the KHI file generated.
// This name is used in frontend to save the inspection data as a file.
var HeaderSuggestedFileNameTask = inspection_task.NewInspectionTask(HeaderSuggestedFileNameTaskID, []taskid.UntypedTaskReference{
	gcp_task.InputStartTimeTaskID,
	gcp_task.InputEndTimeTaskID,
	gcp_task.InputClusterNameTaskID,
}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode) (struct{}, error) {
	metadataSet := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionRunMetadata)
	header := typedmap.GetOrDefault(metadataSet, header.HeaderMetadataKey, &header.Header{})

	clusterName := task.GetTaskResult(ctx, gcp_task.InputClusterNameTaskID.GetTaskReference())
	endTime := task.GetTaskResult(ctx, gcp_task.InputEndTimeTaskID.GetTaskReference())
	startTime := task.GetTaskResult(ctx, gcp_task.InputStartTimeTaskID.GetTaskReference())

	header.SuggestedFileName = getSuggestedFileName(clusterName, startTime, endTime)

	return struct{}{}, nil
}, inspection_task.NewRequiredTaskLabel(), inspection_task.InspectionTypeLabel(inspectiontype.GCPK8sClusterInspectionTypes...))

func getSuggestedFileName(clusterName string, startTime, endTime time.Time) string {
	return fmt.Sprintf("%s-%s-%s.khi", clusterName, startTime.Format("2006_01_02_1504"), endTime.Format("2006_01_02_1504"))
}
