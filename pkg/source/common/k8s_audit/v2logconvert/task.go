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

package v2logconvert

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	common_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/taskid"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
)

var Task = inspectiontaskbase.NewProgressReportableInspectionTask(common_k8saudit_taskid.LogConvertTaskID, []taskid.UntypedTaskReference{
	common_k8saudit_taskid.CommonAuitLogSource,
}, func(ctx context.Context, taskMode inspection_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) (struct{}, error) {
	if taskMode == inspection_contract.TaskModeDryRun {
		return struct{}{}, nil
	}
	builder := khictx.MustGetValue(ctx, inspection_contract.CurrentHistoryBuilder)
	logs := coretask.GetTaskResult(ctx, common_k8saudit_taskid.CommonAuitLogSource)

	processedCount := atomic.Int32{}
	updator := progressutil.NewProgressUpdator(tp, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
		current := processedCount.Load()
		tp.Percentage = float32(current) / float32(len(logs.Logs))
		tp.Message = fmt.Sprintf("%d/%d", current, len(logs.Logs))
	})
	err := updator.Start(ctx)
	if err != nil {
		return struct{}{}, err
	}
	defer updator.Done()
	err = builder.PrepareParseLogs(ctx, logs.Logs, func() {
		processedCount.Add(1)
	})
	if err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
})
