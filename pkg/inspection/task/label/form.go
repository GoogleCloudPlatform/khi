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
