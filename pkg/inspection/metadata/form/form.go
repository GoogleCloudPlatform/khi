// Copyright 2024 Google LLC
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
	"fmt"
	"slices"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const FormFieldSetMetadataKey = "form"

type FormFieldHintType string

const (
	HintTypeWarning = "warning"
	HintTypeInfo    = "info"
)

type FormField struct {
	Priority        int               `json:"-"`
	Id              string            `json:"id"`
	Type            string            `json:"type"`
	Label           string            `json:"label"`
	Description     string            `json:"description"`
	Hint            string            `json:"hint"`
	HintType        FormFieldHintType `json:"hintType"`
	Default         string            `json:"default"`
	AllowEdit       bool              `json:"allowEdit"`
	Suggestions     []string          `json:"suggestions"`
	ValidationError string            `json:"validationError"`
}

// FormFieldSet is a metadata type used in frontend to generate the form fields.
type FormFieldSet struct {
	fields []*FormField
}

var _ metadata.Metadata = (*FormFieldSet)(nil)

// Labels implements Metadata.
func (*FormFieldSet) Labels() *task.LabelSet {
	return task.NewLabelSet(metadata.IncludeInDryRunResult())
}

func (f *FormFieldSet) ToSerializable() interface{} {
	return f.fields
}

func (f *FormFieldSet) SetField(newField *FormField) error {
	if newField.Id == "" {
		return fmt.Errorf("id must not be empty")
	}
	for _, field := range f.fields {
		if field.Id == newField.Id {
			return fmt.Errorf("id %s is already used", newField.Id)
		}
	}
	f.fields = append(f.fields, newField)
	slices.SortFunc(f.fields, func(a, b *FormField) int {
		return b.Priority - a.Priority
	})
	return nil
}

// DangerouslyGetField shouldn't be used in non testing code. Because a field shouldn't depend on the other field metadata.
// This is only for testing purpose.
func (f *FormFieldSet) DangerouslyGetField(id string) *FormField {
	for _, field := range f.fields {
		if field.Id == id {
			return field
		}
	}
	return nil
}

type FormFieldSetMetadataFactory struct{}

// Instanciate implements metadata.MetadataFactory.
func (f *FormFieldSetMetadataFactory) Instanciate() metadata.Metadata {
	return &FormFieldSet{
		fields: make([]*FormField, 0),
	}
}

// FormFieldSetMetadataFactory implements metadata.MetadataFactory
var _ (metadata.MetadataFactory) = (*FormFieldSetMetadataFactory)(nil)
