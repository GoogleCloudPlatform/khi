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

package log

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structurev2"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

var logInstanceID = atomic.Int32{}

// FieldSet represents set of log fields read from a log.
// FieldSet implementations defines how to read these actual fields from a log for parsers to read without using explicit path of field.
type FieldSet interface {
	// FieldSetKind identifies the set of fields. It must be same if the result type has same fields.
	Kind() string
}

type FieldSetReader interface {
	// Read attempts to read the log fields into its field.
	Read(reader *structurev2.NodeReader) (FieldSet, error)

	// FieldSetKind identifies the set of fields. It must be same if the result type has same fields.
	FieldSetKind() string
}

// Log represents a log handled in KHI.
// It provides direct access to its fields and abstracted cached access via FieldSet interface.
type Log struct {
	*structurev2.NodeReader
	fields  *typedmap.TypedMap
	ID      string
	LogType enum.LogType
}

// NewLog returns a log instance from NodeReader instance.
func NewLog(reader *structurev2.NodeReader) *Log {
	return &Log{
		ID:         strconv.Itoa(int(logInstanceID.Add(1))),
		NodeReader: reader,
		fields:     typedmap.NewTypedMap(),
		LogType:    enum.LogTypeUnknown,
	}
}

// NewLogFromYAMLString instanciate a new Log from the given YAML string.
func NewLogFromYAMLString(yaml string) (*Log, error) {
	node, err := structurev2.FromYAML(yaml)
	if err != nil {
		return nil, err
	}
	return NewLog(structurev2.NewNodeReader(node)), nil
}

// SetFieldSetReader reads set of fields with the FieldSetReader and keep it in the log.
func (l *Log) SetFieldSetReader(reader FieldSetReader) error {
	fieldSet, err := reader.Read(l.NodeReader)
	if err != nil {
		return err
	}
	typedmap.Set(l.fields, typedmap.NewTypedKey[FieldSet](reader.FieldSetKind()), fieldSet)
	return nil
}

// GetFieldSet returns the read FieldSet associated with the log.
// It returns an error when the FieldSet wasn't found on the log.
func GetFieldSet[T FieldSet](l *Log, fieldSet T) (T, error) {
	field, found := typedmap.Get(l.fields, typedmap.NewTypedKey[T](fieldSet.Kind()))
	if !found {
		return *new(T), fmt.Errorf("no fieldset loaded for key %s on this log. available keys are %q", fieldSet.Kind(), l.fields.Keys())
	}
	return field, nil
}

// MustGetFieldSet returns the read FieldSet associated with the log.
// It will go panic when the FieldSet was not found on the log.
func MustGetFieldSet[T FieldSet](l *Log, fieldSet T) T {
	field, err := GetFieldSet(l, fieldSet)
	if err != nil {
		panic(err)
	}
	return field
}
