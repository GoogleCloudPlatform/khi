package label

import (
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const (
	TaskLabelKeyIsFormTask           = inspection_task.InspectionTaskPrefix + "is-form-task"
	TaskLabelKeyFormFieldLabel       = inspection_task.InspectionTaskPrefix + "form-field-label"
	TaskLabelKeyFormFieldDescription = inspection_task.InspectionTaskPrefix + "form-field-description"
)

type FormTaskLabelOpt struct {
	description string
	label       string
}

// Write implements task.LabelOpt.
func (f *FormTaskLabelOpt) Write(label *task.LabelSet) {
	label.Set(TaskLabelKeyIsFormTask, true)
	label.Set(TaskLabelKeyFormFieldLabel, f.label)
	label.Set(TaskLabelKeyFormFieldDescription, f.description)

}

// NewFormTaskLabelOpt constucts a new instance of task.LabelOpt for form related tasks.
func NewFormTaskLabelOpt(label, description string) *FormTaskLabelOpt {
	return &FormTaskLabelOpt{
		label:       label,
		description: description,
	}
}

var _ (task.LabelOpt) = (*FormTaskLabelOpt)(nil)
