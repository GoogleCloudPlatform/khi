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
	"slices"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func NewLogSorterByTimeTask(taskID taskid.TaskImplementationID[[]*log.Log], logSource taskid.TaskReference[[]*log.Log]) coretask.Task[[]*log.Log] {
	return NewProgressReportableInspectionTask(taskID, []taskid.UntypedTaskReference{logSource}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
		if taskMode != inspectioncore_contract.TaskModeRun {
			return []*log.Log{}, nil
		}
		progress.MarkIndeterminate()
		logs := coretask.GetTaskResult(ctx, logSource)
		logs = slices.Clone(logs)
		slices.SortFunc(logs, func(a, b *log.Log) int {
			aFieldSet := log.MustGetFieldSet(a, &log.CommonFieldSet{})
			bFieldSet := log.MustGetFieldSet(b, &log.CommonFieldSet{})
			return aFieldSet.Timestamp.Compare(bFieldSet.Timestamp)
		})
		return logs, nil
	})
}
