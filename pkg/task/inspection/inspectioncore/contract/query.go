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

package inspectioncore_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

var (
	TaskLabelKeyIsQueryTask          = coretask.NewTaskLabelKey[bool](InspectionTaskPrefix + "is-query-task")
	TaskLabelKeyQueryTaskSampleQuery = coretask.NewTaskLabelKey[string](InspectionTaskPrefix + "query-task-sample-query")
)

type QueryTaskLabelOpt struct {
	TargetLogType *pb.LogType
	SampleQuery   string
}

// Write implements task.LabelOpt.
func (q *QueryTaskLabelOpt) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, TaskLabelKeyIsQueryTask, true)
	typedmap.Set(label, TaskLabelKeyQueryTaskSampleQuery, q.SampleQuery)
}

var _ (coretask.LabelOpt) = (*QueryTaskLabelOpt)(nil)

// NewQueryTaskLabelOpt constructs a new instance of task.LabelOpt for query related tasks.
func NewQueryTaskLabelOpt(sampleQuery string) *QueryTaskLabelOpt {
	return &QueryTaskLabelOpt{
		SampleQuery: sampleQuery,
	}
}
