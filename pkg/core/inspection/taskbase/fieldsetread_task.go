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

package inspectiontaskbase

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
)

// NewFieldSetReadTask create a task consumes the list of logs and set the given field set readers for all of the logs concurrently.
func NewFieldSetReadTask(taskId taskid.TaskImplementationID[struct{}], logTask taskid.TaskReference[[]*log.Log], fieldSetReaders []log.FieldSetReader) coretask.Task[struct{}] {
	return NewProgressReportableInspectionTask(taskId, []taskid.UntypedTaskReference{
		logTask,
	}, func(ctx context.Context, taskMode inspection_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (struct{}, error) {
		if taskMode != inspection_contract.TaskModeRun {
			return struct{}{}, nil
		}

		logs := coretask.GetTaskResult(ctx, logTask)
		concurrency := 16
		pool := worker.NewPool(concurrency)
		completed := atomic.Uint64{}

		progressUpdator := progressutil.NewProgressUpdator(progress, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := completed.Load()
			tp.Percentage = float32(current) / float32(len(logs))
			tp.Message = fmt.Sprintf("%d/%d", current, len(logs))
		})
		progressUpdator.Start(ctx)

		for c := 0; c < concurrency; c++ {
			pool.Run(func() {
				for i := c; i < len(logs); i += concurrency {
					l := logs[i]
					for _, fieldSetReader := range fieldSetReaders {
						err := l.SetFieldSetReader(fieldSetReader)
						if err != nil {
							slog.WarnContext(ctx, fmt.Sprintf("failed to run fieldSetReader(%s) for log id=%s\nError: %v", fieldSetReader.FieldSetKind(), l.ID, err.Error()))
						}
					}
					completed.Add(1)
				}
			})
		}

		pool.Wait()
		progressUpdator.Done()

		return struct{}{}, nil
	})
}
