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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var FormFieldSetMetadataKey = metadata.NewMetadataKey[*FormFieldSet]("form")

// ParameterInputType represents the type of parameter form field.
type ParameterInputType string

const (
	// Group is a type of ParameetrInputType. This contains multiple children fields.
	Group ParameterInputType = "group"
	// Text is a type of ParameterInputType. This represents the text type input field.
	Text ParameterInputType = "text"
	// File is a type of ParameterInputType. This represents the file type input field.
	File ParameterInputType = "file"
)

// ParameterHintType represents the types of hint message shown at the bottom of parameter forms.
type ParameterHintType string

const (
	// None is a type of ParameterHintType. Frontend will supress hint when this type is given.
	None ParameterHintType = "none"
	// Error is a type of ParameterHintType. Frontend will prevent user to click the button to go the next page when there is at least a field with a hint of this type.
	Error ParameterHintType = "error"
	// Warning is a type of ParameterHintType. User may need be cautious about the field, but user can navigate the form to the next step with a field having this type hint.
	Warning ParameterHintType = "warning"
	// Info is a type of ParameterHintType. It's a supplemntal hint just for helping user.
	Info ParameterHintType = "info"
)

type ParameterFormField interface{}

// ParameterFormFieldBase is the base type of parameter form fields.
type ParameterFormFieldBase struct {
	// Priority is a number used in sorting order.
	Priority int `json:"-"`
	// ID is a unique name of the form field.
	ID string `json:"id"`
	// Type is the ParameterInputType of this field.
	Type ParameterInputType `json:"type"`
	// Label is a short human readable title of this field. This is visible on the form.
	Label string `json:"label"`
	// Description is a human readable explaination of this field.
	Description string `json:"description"`
	// HintType is the ParameterHintType of the Hint field of this parameter field.
	HintType ParameterHintType `json:"hintType"`
	// Hint is the message shown under the form field. Assign HintType as well when you assign a value to this field.
	Hint string `json:"hint"`
}

// GroupParameterFormField represents Group type parameter specific data.
type GroupParameterFormField struct {
	ParameterFormFieldBase
	// Children is the children of this field.
	Children []ParameterFormField `json:"children"`
}

// TextParameterFormField represents Text type parameter specific data.
type TextParameterFormField struct {
	ParameterFormFieldBase
	// Readonly limits users to modify the field.
	Readonly bool `json:"readonly"`
	// Default is the default value of this field.
	Default string `json:"default"`
	// Suggestion is the auto complete drop down values.
	Suggestions []string `json:"suggestions"`
}

// FileParameterFormField represents File type parameter specific data.
type FileParameterFormField struct {
	ParameterFormFieldBase
	// Token is the type used for specifying the destination of file. This value is generated from server side and the client will upload the file with the token.
	Token upload.UploadToken `json:"token"`
	// Status is the current status of the file.
	Status upload.UploadStatus `json:"status"`
}

// FormFieldSet is a metadata type used in frontend to generate the form fields.
type FormFieldSet struct {
	fieldsLock sync.RWMutex
	fields     []ParameterFormField
}

var _ metadata.Metadata = (*FormFieldSet)(nil)

// Labels implements Metadata.
func (*FormFieldSet) Labels() *typedmap.ReadonlyTypedMap {
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

func NewFormFieldSet() *FormFieldSet {
	return &FormFieldSet{
		fields: make([]ParameterFormField, 0),
	}
}
