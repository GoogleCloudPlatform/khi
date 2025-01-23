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

package yamlutil

import (
	"errors"

	"gopkg.in/yaml.v3"
)

var NodeNotFoundError = errors.New("Node not found")

func NewEmptyMapNode() *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: make([]*yaml.Node, 0),
	}
}

func NewMapElementWithScalarValue(key string, value string) []*yaml.Node {
	return []*yaml.Node{
		{
			Kind:    yaml.ScalarNode,
			Content: make([]*yaml.Node, 0),
			Value:   key,
		},
		{
			Kind:    yaml.ScalarNode,
			Content: make([]*yaml.Node, 0),
			Value:   value,
		},
	}
}

func NewScalarNode(value string) *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.ScalarNode,
		Content: make([]*yaml.Node, 0),
		Value:   value,
	}
}

func DecomposeMapElement(mapNode *yaml.Node, index int) (string, *yaml.Node) {
	mapIndex := index * 2
	key := mapNode.Content[mapIndex]
	value := mapNode.Content[mapIndex+1]
	return key.Value, value
}

func GetMapElement(mapNode *yaml.Node, key string) (*yaml.Node, error) {
	for i := 0; i < GetMapLength(mapNode); i++ {
		fieldName, node := DecomposeMapElement(mapNode, i)
		if key == fieldName {
			return node, nil
		}
	}
	return nil, NodeNotFoundError
}

func GetMapLength(mapNode *yaml.Node) int {
	return len(mapNode.Content) / 2
}
