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

package structuredata

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
)

// There is no way to escape `,` in yaml tag to use it in a key.
// Replace it with hardly unused string and replace at the end.
// User of sortedMap needs to convert it back to `,`
var fieldCommaEscape = "@@COMMA@@"

type sortedMap struct {
	keys   []string
	values map[string]any
}

// MarshalJSON implements json.Marshaler.
func (s *sortedMap) MarshalJSON() ([]byte, error) {
	structValue, err := s.toReflectStructWithOrderedFields()
	if err != nil {
		return nil, err
	}
	return json.Marshal(structValue)
}

func newSortedMap() *sortedMap {
	return &sortedMap{
		keys:   make([]string, 0),
		values: map[string]any{},
	}
}

// MarshalYAML implements yaml.Marshaler.
func (s *sortedMap) MarshalYAML() (interface{}, error) {
	return s.toReflectStructWithOrderedFields()
}

func (s *sortedMap) toReflectStructWithOrderedFields() (interface{}, error) {
	if len(s.keys) == 0 {
		return map[string]any{}, nil
	}

	fields := []reflect.StructField{}
	for i, key := range s.keys {
		var t reflect.Type
		if s.values[key] == nil {
			t = reflect.ValueOf((*struct{})(nil)).Type()
		} else {
			t = reflect.ValueOf(s.values[key]).Type()
		}
		escapedKey := escapeKey(key)
		structTag := fmt.Sprintf(`yaml:"%s" json:"%s"`, escapedKey, escapedKey)
		fields = append(fields, reflect.StructField{
			Name: genFieldName(i),
			Type: t,
			Tag:  reflect.StructTag(structTag),
		})
	}

	structType := reflect.StructOf(fields)
	structInstance := reflect.New(structType).Elem()

	for i, key := range s.keys {
		structFieldValue := structInstance.FieldByName(genFieldName(i))
		var value reflect.Value
		if s.values[key] == nil {
			value = reflect.ValueOf((*struct{})(nil))
		} else {
			value = reflect.ValueOf(s.values[key])
		}
		structFieldValue.Set(value)
	}

	return structInstance.Interface(), nil
}

var _ yaml.Marshaler = (*sortedMap)(nil)
var _ json.Marshaler = (*sortedMap)(nil)

func (s *sortedMap) AddNextField(key string, value any) {
	s.keys = append(s.keys, key)
	s.values[key] = value
}

func genFieldName(fieldIndex int) string {
	return fmt.Sprintf("Field_%d", fieldIndex)
}

func escapeKey(key string) string {
	return strings.ReplaceAll(strings.ReplaceAll(key, "\"", "\\\""), ",", fieldCommaEscape)
}
