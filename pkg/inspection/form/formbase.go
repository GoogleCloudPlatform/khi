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
)

// FormTaskBuilderBase provides common functionality for form task builders
type FormTaskBuilderBase struct {
	id           string
	label        string
	priority     int
	dependencies []string
	description  string
}

// NewFormTaskBuilderBase creates a new instance of the base builder
func NewFormTaskBuilderBase(id string, priority int, label string) FormTaskBuilderBase {
	return FormTaskBuilderBase{
		id:           id,
		priority:     priority,
		label:        label,
		dependencies: []string{},
	}
}

// WithDescription sets the description for the form field
func (b *FormTaskBuilderBase) WithDescription(description string) *FormTaskBuilderBase {
	b.description = description
	return b
}

// WithDependencies sets the task dependencies
func (b *FormTaskBuilderBase) WithDependencies(dependencies []string) *FormTaskBuilderBase {
	b.dependencies = dependencies
	return b
}

// SetupBaseFormField configures common form field properties
func (b *FormTaskBuilderBase) SetupBaseFormField(field *form_metadata.ParameterFormFieldBase) {
	field.ID = b.id
	field.Label = b.label
	field.Priority = b.priority
	field.Description = b.description
}
