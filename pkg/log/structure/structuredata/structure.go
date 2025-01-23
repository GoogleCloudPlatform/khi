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
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

type StructuredDataFieldType string

const (
	StructuredTypeMap     StructuredDataFieldType = "map"
	StructuredTypeArray   StructuredDataFieldType = "array"
	StructuredTypeScalar  StructuredDataFieldType = "scalar"
	StructuredTypeInvalid StructuredDataFieldType = "invalid"
)

// Represents a structured data. For example, the data represented as YAML string.
// This provides low level API for reading the structure data in KHI.
type StructureData interface {
	Type() (StructuredDataFieldType, error)
	Keys() ([]string, error)
	Value(fieldName string) (any, error)
}

type YamlNodeStructuredData struct {
	Node *yaml.Node
}

// DataFromYaml generates the StructureData from string YAML data.
func DataFromYaml(yamlString string) (StructureData, error) {
	var rawNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlString), &rawNode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse yaml\nYAML:%s\nERROR:%s", yamlString, err)
	}
	return DataFromYamlNode(&rawNode)
}

// DataFromYamlNode generates the StructureData from the raw yaml.Node pointer.
func DataFromYamlNode(node *yaml.Node) (StructureData, error) {
	if node.Kind == 0 { // Errornous YAML maybe parsed from empty string
		// Emptry string must be mapped to an empty map
		return &YamlNodeStructuredData{
			Node: &yaml.Node{
				Kind:    yaml.MappingNode,
				Content: make([]*yaml.Node, 0),
			}}, nil
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) > 1 {
			return nil, fmt.Errorf("structured data is not supporting multiple document")
		}
		node = node.Content[0]
	}
	return &YamlNodeStructuredData{Node: node}, nil
}

// Keys implements StrucureData.
func (y *YamlNodeStructuredData) Keys() ([]string, error) {
	t, err := y.Type()
	if err != nil {
		return nil, err
	}
	switch t {
	case StructuredTypeMap:
		if len(y.Node.Content)%2 == 1 {
			return nil, fmt.Errorf("map node content count must be even")
		}
		keys := []string{}
		for i := 0; i < len(y.Node.Content); i += 2 {
			keyValue := y.Node.Content[i]
			keys = append(keys, keyValue.Value)
		}
		return keys, nil
	case StructuredTypeArray:
		return stringSequence(len(y.Node.Content)), nil
	default:
		return []string{""}, nil
	}
}

// Type implements StrucureData.
func (y *YamlNodeStructuredData) Type() (StructuredDataFieldType, error) {
	switch y.Node.Kind {
	case yaml.MappingNode:
		return StructuredTypeMap, nil
	case yaml.SequenceNode:
		return StructuredTypeArray, nil
	case yaml.ScalarNode:
		return StructuredTypeScalar, nil
	default:
		return StructuredTypeInvalid, fmt.Errorf("node kind %d is not a supported type", y.Node.Kind)
	}
}

// Value implements StrucureData.
func (y *YamlNodeStructuredData) Value(fieldName string) (any, error) {
	t, err := y.Type()
	if err != nil {
		return nil, err
	}
	switch t {
	case StructuredTypeScalar:
		if fieldName == "" {
			var data any
			err := y.Node.Decode(&data)
			if err != nil {
				return nil, err
			}
			return data, nil
		} else {
			return nil, fmt.Errorf("attempted to read a field %s from a scalar", fieldName)
		}
	case StructuredTypeMap:
		if fieldName == "" {
			return y, nil
		}
		if len(y.Node.Content)%2 == 1 {
			return nil, fmt.Errorf("map node content count must be even")
		}
		for i := 0; i < len(y.Node.Content); i += 2 {
			keyValue := y.Node.Content[i]
			if keyValue.Value == fieldName {
				return &YamlNodeStructuredData{Node: y.Node.Content[i+1]}, nil
			}
		}
		return nil, fmt.Errorf("field not found:%s", fieldName)
	case StructuredTypeArray:
		if fieldName == "" {
			return y, nil
		}
		index, err := strconv.Atoi(fieldName)
		if err != nil {
			return nil, fmt.Errorf("index can't be parsed as an integer")
		}
		if index < 0 {
			return nil, fmt.Errorf("index must not be negative")
		}
		if index >= len(y.Node.Content) {
			return nil, fmt.Errorf("index out of bounds")
		}
		return &YamlNodeStructuredData{Node: y.Node.Content[index]}, nil
	default:
		return nil, fmt.Errorf("unsupported strcuture type %s", t)
	}
}

var _ StructureData = (*YamlNodeStructuredData)(nil)

func stringSequence(length int) []string {
	result := []string{}
	for i := 0; i < length; i++ {
		result = append(result, strconv.Itoa(i))
	}
	return result
}
