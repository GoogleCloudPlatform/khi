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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// NewLogSerializerTask store its given logs to history to prepare the history type to have ChangeSet associated with the log.
// This must be called before HistoryModifier and Logs must be discarded before this task if it shouldn't be included in the result.
func NewLogSerializerTask(taskID taskid.TaskImplementationID[[]*log.Log], input taskid.TaskReference[[]*log.Log]) coretask.Task[[]*log.Log] {
	return NewProgressReportableInspectionTask(taskID, []taskid.UntypedTaskReference{input}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return []*log.Log{}, nil
		}
		logs := coretask.GetTaskResult(ctx, input)
		builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)

		processedLogCount := 0
		err := builder.SerializeLogs(ctx, logs, func() {
			p := float32(processedLogCount) / float32(len(logs))
			progress.Update(p, fmt.Sprintf("%d/%d", processedLogCount, len(logs)))
			processedLogCount++
		})
		if err != nil {
			return nil, err
		}

		tracingActive, _ := khictx.GetValue(ctx, inspectioncore_contract.TracingActive)
		if tracingActive {
			trace.SpanFromContext(ctx).SetAttributes(
				attribute.String("log_count", fmt.Sprintf("%d", len(logs))),
			)
		}
		return logs, nil
	},
		// Tasks modifying history must be dependent from SerializerTask.
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()))
}
