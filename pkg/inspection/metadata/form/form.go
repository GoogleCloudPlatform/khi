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
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const FormFieldSetMetadataKey = "form"

// ParameterInputType represents the type of parameter form field.
type ParameterInputType string

const (
	Group ParameterInputType = "group"
	Text  ParameterInputType = "text"
	File  ParameterInputType = "file"
)

// ParameterHintType represents the types of hint message shown at the bottom of parameter forms.
type ParameterHintType string

const (
	None    ParameterHintType = "none"
	Error   ParameterHintType = "error"
	Warning ParameterHintType = "warning"
	Info    ParameterHintType = "info"
)

type ParameterFormField interface{}

// ParameterFormFieldBase is the base type of parameter form fields.
type ParameterFormFieldBase struct {
	Priority    int                `json:"-"`
	ID          string             `json:"id"`
	Type        ParameterInputType `json:"type"`
	Label       string             `json:"label"`
	Description string             `json:"description"`
	HintType    ParameterHintType  `json:"hintType"`
	Hint        string             `json:"hint"`
}

// GroupParameterFormField represents Group type parameter specific data.
type GroupParameterFormField struct {
	ParameterFormFieldBase
	Children []ParameterFormField `json:"children"`
}

// TextParameterFormField represents Text type parameter specific data.
type TextParameterFormField struct {
	ParameterFormFieldBase
	Readonly    bool     `json:"readonly"`
	Default     string   `json:"default"`
	Suggestions []string `json:"suggestions"`
}

// UploadStatus represents the types of UploadStatus given from the backend.
type UploadStatus int

const (
	Waiting   UploadStatus = 0
	Uploading UploadStatus = 1
	Verifying UploadStatus = 2
	Done      UploadStatus = 3
)

// FileParameterFormField represents File type parameter specific data.
type FileParameterFormField struct {
	ParameterFormFieldBase
	Token  upload.UploadToken `json:"token"`
	Status UploadStatus       `json:"status"`
}

// FormFieldSet is a metadata type used in frontend to generate the form fields.
type FormFieldSet struct {
	fieldsLock sync.RWMutex
	fields     []ParameterFormField
}

var _ metadata.Metadata = (*FormFieldSet)(nil)

// Labels implements Metadata.
func (*FormFieldSet) Labels() *task.LabelSet {
	return task.NewLabelSet(metadata.IncludeInDryRunResult())
}

func (f *FormFieldSet) ToSerializable() interface{} {
	return f.fields
}

func (f *FormFieldSet) SetField(newField ParameterFormField) error {
	f.fieldsLock.Lock()
	defer f.fieldsLock.Unlock()
	newFieldBase := GetParameterFormFieldBase(newField)
	if newFieldBase.ID == "" {
		return fmt.Errorf("id must not be empty")
	}
	for _, field := range f.fields {
		fieldBase := GetParameterFormFieldBase(field)
		if fieldBase.ID == newFieldBase.ID {
			return fmt.Errorf("id %s is already used", newFieldBase.ID)
		}
	}
	f.fields = append(f.fields, newField)
	slices.SortFunc(f.fields, func(a, b ParameterFormField) int {
		return GetParameterFormFieldBase(b).Priority - GetParameterFormFieldBase(a).Priority
	})
	return nil
}

// DangerouslyGetField shouldn't be used in non testing code. Because a field shouldn't depend on the other field metadata.
// This is only for testing purpose.
func (f *FormFieldSet) DangerouslyGetField(id string) ParameterFormField {
	f.fieldsLock.RLock()
	defer f.fieldsLock.RUnlock()
	for _, field := range f.fields {
		if GetParameterFormFieldBase(field).ID == id {
			return field
		}
	}
	return ParameterFormFieldBase{}
}

// GetParameterFormFieldBase returns the ParameterFormFieldBase from the given ParameterFormField.
func GetParameterFormFieldBase(parameter ParameterFormField) ParameterFormFieldBase {
	switch v := parameter.(type) {
	case GroupParameterFormField:
		return v.ParameterFormFieldBase
	case TextParameterFormField:
		return v.ParameterFormFieldBase
	case FileParameterFormField:
		return v.ParameterFormFieldBase
	default:
		return ParameterFormFieldBase{}
	}
}

type FormFieldSetMetadataFactory struct{}

// Instanciate implements metadata.MetadataFactory.
func (f *FormFieldSetMetadataFactory) Instanciate() metadata.Metadata {
	return &FormFieldSet{
		fields: make([]ParameterFormField, 0),
	}
}

// FormFieldSetMetadataFactory implements metadata.MetadataFactory
var _ (metadata.MetadataFactory) = (*FormFieldSetMetadataFactory)(nil)
