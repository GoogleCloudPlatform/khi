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

package form

import (
	form_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// FormTaskBuilderBase provides common functionality for form task builders
type FormTaskBuilderBase[T any] struct {
	id           taskid.TaskImplementationID[T]
	label        string
	priority     int
	dependencies []taskid.UntypedTaskReference
	description  string
}

// NewFormTaskBuilderBase creates a new instance of the base builder
func NewFormTaskBuilderBase[T any](id taskid.TaskImplementationID[T], priority int, label string) FormTaskBuilderBase[T] {
	return FormTaskBuilderBase[T]{
		id:           id,
		priority:     priority,
		label:        label,
		dependencies: []taskid.UntypedTaskReference{},
	}
}

// WithDescription sets the description for the form field
func (b *FormTaskBuilderBase[T]) WithDescription(description string) *FormTaskBuilderBase[T] {
	b.description = description
	return b
}

// WithDependencies sets the task dependencies
func (b *FormTaskBuilderBase[T]) WithDependencies(dependencies []taskid.UntypedTaskReference) *FormTaskBuilderBase[T] {
	b.dependencies = dependencies
	return b
}

// SetupBaseFormField configures common form field properties
func (b *FormTaskBuilderBase[T]) SetupBaseFormField(field *form_metadata.ParameterFormFieldBase) {
	field.ID = b.id.ReferenceIDString()
	field.Label = b.label
	field.Priority = b.priority
	field.Description = b.description
}
