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
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/k8saudittask"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var Task = inspection_task.NewInspectionTask(k8saudittask.LogConvertTaskID, []taskid.UntypedTaskReference{
	inspection_task.BuilderGeneratorTask.ID(),
	k8saudittask.K8sAuditQueryTaskID,
}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, tp *progress.TaskProgress) (any, error) {
	if taskMode == inspection_task_interface.TaskModeDryRun {
		return struct{}{}, nil
	}
	builder := task.GetTaskResult(ctx, inspection_task.BuilderGeneratorTaskID.GetTaskReference())
	logs := task.GetTaskResult(ctx, k8saudittask.K8sAuditQueryTaskID.GetTaskReference())

	processedCount := atomic.Int32{}
	updator := progress.NewProgressUpdator(tp, time.Second, func(tp *progress.TaskProgress) {
		current := processedCount.Load()
		tp.Percentage = float32(current) / float32(len(logs))
		tp.Message = fmt.Sprintf("%d/%d", current, len(logs))
	})
	err := updator.Start(ctx)
	if err != nil {
		return nil, err
	}
	defer updator.Done()
	err = builder.PrepareParseLogs(ctx, logs, func() {
		processedCount.Add(1)
	})
	if err != nil {
		return nil, err
	}
	return struct{}{}, nil
})
