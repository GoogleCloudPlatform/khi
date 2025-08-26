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
	"time"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// LogFilterFunc defines the function signature for filtering logs. It returns true if the log should be kept.
type LogFilterFunc = func(ctx context.Context, log *log.Log) bool

// NewLogFilterTask creates a task that consumes a list of logs and returns a new list
// containing only the logs that satisfy the filter function.
func NewLogFilterTask(tid taskid.TaskImplementationID[[]*log.Log], sourceLogs taskid.TaskReference[[]*log.Log], logFilter LogFilterFunc) coretask.Task[[]*log.Log] {
	return NewProgressReportableInspectionTask(tid, []taskid.UntypedTaskReference{sourceLogs}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
		if taskMode != inspectioncore_contract.TaskModeRun {
			return []*log.Log{}, nil
		}

		logs := coretask.GetTaskResult(ctx, sourceLogs)
		completed := 0
		filteredLogs := []*log.Log{}

		progressUpdator := progressutil.NewProgressUpdator(progress, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			tp.Percentage = float32(completed) / float32(len(logs))
			tp.Message = fmt.Sprintf("%d/%d", completed, len(logs))
		})
		progressUpdator.Start(ctx)

		for _, l := range logs {
			if logFilter(ctx, l) {
				filteredLogs = append(filteredLogs, l)
			}
			completed++
		}

		progressUpdator.Done()
		return filteredLogs, nil
	})
}
