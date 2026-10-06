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

package formtask

import (
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// formTaskBase holds the field settings shared by every form task.
type formTaskBase[T any] struct {
	id          taskid.TaskImplementationID[T]
	label       string
	priority    int
	description string
}

// newFormTaskBase creates the shared field settings of a form task.
func newFormTaskBase[T any](id taskid.TaskImplementationID[T], priority int, label string, description string) formTaskBase[T] {
	return formTaskBase[T]{
		id:          id,
		priority:    priority,
		label:       label,
		description: description,
	}
}

// setupBaseFormField configures the form field properties shared by every form type.
func (b *formTaskBase[T]) setupBaseFormField(field *inspectionmetadata.ParameterFormFieldBase) {
	field.ID = b.id.ReferenceIDString()
	field.Label = b.label
	field.Priority = b.priority
	field.Description = b.description
}

// formLabelOpts appends the form task label to the given label options.
func (b *formTaskBase[T]) formLabelOpts(labelOpts []coretask.LabelOpt) []coretask.LabelOpt {
	return append(labelOpts, inspectioncore.NewFormTaskLabelOpt(b.label, b.description))
}
