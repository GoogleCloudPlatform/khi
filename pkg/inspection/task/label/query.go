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
