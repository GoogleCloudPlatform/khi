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

package label

import (
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const (
	TaskLabelKeyIsQueryTask            = inspection_task.InspectionTaskPrefix + "is-query-task"
	TaskLabelKeyQueryTaskTargetLogType = inspection_task.InspectionTaskPrefix + "query-task-target-log-type"
	TaskLabelKeyQueryTaskSampleQuery   = inspection_task.InspectionTaskPrefix + "query-task-sample-query"
)

type QueryTaskLabelOpt struct {
	TargetLogType enum.LogType
	SampleQuery   string
}

// Write implements task.LabelOpt.
func (q *QueryTaskLabelOpt) Write(label *task.LabelSet) {
	label.Set(TaskLabelKeyIsQueryTask, true)
	label.Set(TaskLabelKeyQueryTaskTargetLogType, q.TargetLogType)
	label.Set(TaskLabelKeyQueryTaskSampleQuery, q.SampleQuery)

}

var _ (task.LabelOpt) = (*QueryTaskLabelOpt)(nil)

// NewQueryTaskLabelOpt constucts a new instance of task.LabelOpt for query related tasks.
func NewQueryTaskLabelOpt(targetLogType enum.LogType, sampleQuery string) *QueryTaskLabelOpt {
	return &QueryTaskLabelOpt{
		TargetLogType: targetLogType,
		SampleQuery:   sampleQuery,
	}
}
