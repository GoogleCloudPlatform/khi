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

package inspectioncore

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

var (
	// TaskLabelKeyIsFormTask marks whether a task represents a user input form field.
	TaskLabelKeyIsFormTask = coretask.NewTaskLabelKey[bool](InspectionTaskPrefix + "is-form-task")
	// TaskLabelKeyFormFieldLabel stores the display label for a form field task.
	TaskLabelKeyFormFieldLabel = coretask.NewTaskLabelKey[string](InspectionTaskPrefix + "form-field-label")
	// TaskLabelKeyFormFieldDescription stores the description text for a form field task.
	TaskLabelKeyFormFieldDescription = coretask.NewTaskLabelKey[string](InspectionTaskPrefix + "form-field-description")
)

var _ (coretask.LabelOpt) = (*FormTaskLabelOpt)(nil)

// FormTaskLabelOpt writes form-related labels onto a task label map.
type FormTaskLabelOpt struct {
	description string
	label       string
}

// NewFormTaskLabelOpt constructs a new instance of task.LabelOpt for form related tasks.
func NewFormTaskLabelOpt(label, description string) *FormTaskLabelOpt {
	return &FormTaskLabelOpt{
		label:       label,
		description: description,
	}
}

// Write implements task.LabelOpt.
func (f *FormTaskLabelOpt) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, TaskLabelKeyIsFormTask, true)
	typedmap.Set(label, TaskLabelKeyFormFieldLabel, f.label)
	typedmap.Set(label, TaskLabelKeyFormFieldDescription, f.description)
	typedmap.Set(label, coretask.LabelKeyTaskTitle, f.label)
	typedmap.Set(label, coretask.LabelKeyTaskDescription, f.description)
}
