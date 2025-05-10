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

package structurev2

import (
	"gopkg.in/yaml.v3"
)

type NodeSerializer interface {
	Serialize(node Node) ([]byte, error)
}

type YAMLNodeSerializer struct{}

// Serialize implements NodeSerializer.
func (y *YAMLNodeSerializer) Serialize(node Node) ([]byte, error) {
	serializable, err := getYAMLMarshaler(node)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(serializable)
}

var _ NodeSerializer = (*YAMLNodeSerializer)(nil)

type JSONNodeSerializer struct{}

// Serialize implements NodeSerializer.
func (j *JSONNodeSerializer) Serialize(node Node) ([]byte, error) {
	serializable, err := getJSONMarshalerFromNode(node)
	if err != nil {
		return nil, err
	}
	return serializable.MarshalJSON()
}
