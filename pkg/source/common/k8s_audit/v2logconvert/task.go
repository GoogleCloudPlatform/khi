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

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	common_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var Task = inspection_task.NewProgressReportableInspectionTask(common_k8saudit_taskid.LogConvertTaskID, []taskid.UntypedTaskReference{
	inspection_task.BuilderGeneratorTaskID.Ref(),
	common_k8saudit_taskid.CommonAuitLogSource,
}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, tp *progress.TaskProgress) (struct{}, error) {
	if taskMode == inspection_task_interface.TaskModeDryRun {
		return struct{}{}, nil
	}
	builder := task.GetTaskResult(ctx, inspection_task.BuilderGeneratorTaskID.Ref())
	logs := task.GetTaskResult(ctx, common_k8saudit_taskid.CommonAuitLogSource)

	processedCount := atomic.Int32{}
	updator := progress.NewProgressUpdator(tp, time.Second, func(tp *progress.TaskProgress) {
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
