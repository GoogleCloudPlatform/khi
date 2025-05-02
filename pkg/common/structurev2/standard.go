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
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// standard.go contains types to hold structured data on memory as its field and implements Node interface.

// StandardScalarNode is a leaf of structured data implemting Node interface.
type StandardScalarNode[T any] struct {
	value T
}

func (n *StandardScalarNode[T]) Type() NodeType {
	return ScalarNodeType
}

func (n *StandardScalarNode[T]) NodeScalarValue() (any, error) {
	return n.value, nil
}

func (n *StandardScalarNode[T]) Children() NodeChildrenIterator {
	return func(func(key NodeChildrenKey, value Node) bool) {}
}

// MarshalJSON implements json.Marshaler.
func (n *StandardScalarNode[T]) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	value, err := json.Marshal(n.value)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(value)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalYAML implements yaml.Marshaler.
func (n *StandardScalarNode[T]) MarshalYAML() (interface{}, error) {
	yamlNode := &yaml.Node{
		Kind: yaml.ScalarNode,
	}
	anyValue, err := n.NodeScalarValue()
	if err != nil {
		return nil, err
	}
	if anyValue == nil {
		yamlNode.Tag = "!!null"
		yamlNode.Value = "null"
	} else {
		switch value := anyValue.(type) {
		case string:
			yamlNode.Tag = "!!str"
			yamlNode.Value = value
		case int:
			yamlNode.Tag = "!!int"
			yamlNode.Value = fmt.Sprintf("%d", value)
		case bool:
			yamlNode.Tag = "!!bool"
			yamlNode.Value = fmt.Sprintf("%t", value)
		case float64:
			yamlNode.Tag = "!!float"
			yamlNode.Value = fmt.Sprintf("%f", value)
		case time.Time:
			yamlNode.Tag = "!!timestamp"
			yamlNode.Value = value.Format(time.RFC3339)
		default:
			return nil, fmt.Errorf("unsupported scalar type: %T", value)
		}
	}
	return yamlNode, nil
}

var _ Node = (*StandardScalarNode[any])(nil)
var _ json.Marshaler = (*StandardScalarNode[any])(nil)
var _ yaml.Marshaler = (*StandardScalarNode[any])(nil)

// StandardScalarNode is a sequence field of a structured data implementing Node interface.
type StandardSequenceNode struct {
	value []Node
}

func (n *StandardSequenceNode) Type() NodeType {
	return SequenceNodeType
}

func (n *StandardSequenceNode) NodeScalarValue() (any, error) {
	return nil, ErrNonScalarNode
}

func (n *StandardSequenceNode) Children() NodeChildrenIterator {
	return func(f func(key NodeChildrenKey, value Node) bool) {
		for i, v := range n.value {
			if !f(NodeChildrenKey{Index: i}, v) {
				return
			}
		}
	}
}

// MarshalYAML implements yaml.Marshaler.
func (n *StandardSequenceNode) MarshalYAML() (interface{}, error) {
	sequenceNode := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: []*yaml.Node{},
	}
	for _, child := range n.Children() {
		marshaller, ok := child.(yaml.Marshaler)
		if !ok {
			return nil, fmt.Errorf("sequence node child is not implementing yaml.Marshaller")
		}
		node, err := marshaller.MarshalYAML()
		if err != nil {
			return nil, err
		}
		yamlNode, ok := node.(*yaml.Node)
		if !ok {
			return nil, fmt.Errorf("sequence node child returned non *yaml.Node as its result")
		}
		sequenceNode.Content = append(sequenceNode.Content, yamlNode)
	}
	return sequenceNode, nil
}

// MarshalJSON implements json.Marshaler.
func (n *StandardSequenceNode) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("[")
	for i, child := range n.Children() {
		if i.Index > 0 {
			buf.WriteString(",")
		}
		marshaller, ok := child.(json.Marshaler)
		if !ok {
			return nil, fmt.Errorf("sequence node child is not implementing json.Marshaller")
		}
		marshalled, err := marshaller.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(marshalled)
	}
	buf.WriteString("]")

	return buf.Bytes(), nil
}

var _ Node = (*StandardSequenceNode)(nil)
var _ yaml.Marshaler = (*StandardSequenceNode)(nil)
var _ json.Marshaler = (*StandardSequenceNode)(nil)

// StandardMapNode is a map field of structured data implementing Node interface.
// This type retain the order of keys.
type StandardMapNode struct {
	// keys is the list of keys in values.
	// Needed because the key order of map is not assured in Go.
	keys   []string
	values []Node
}

func (n *StandardMapNode) Type() NodeType {
	return MapNodeType
}

func (n *StandardMapNode) NodeScalarValue() (any, error) {
	return nil, ErrNonScalarNode
}

func (n *StandardMapNode) Children() NodeChildrenIterator {
	return func(f func(key NodeChildrenKey, value Node) bool) {
		for i, k := range n.keys {
			if !f(NodeChildrenKey{Index: i, Key: k}, n.values[i]) {
				return
			}
		}
	}
}

// MarshalYAML implements yaml.Marshaler.
func (n *StandardMapNode) MarshalYAML() (interface{}, error) {
	mapNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: []*yaml.Node{},
	}
	for k, child := range n.Children() {
		key := k.Key
		marshaller, ok := child.(yaml.Marshaler)
		if !ok {
			return nil, fmt.Errorf("map node child is not implementing yaml.Marshaller")
		}
		node, err := marshaller.MarshalYAML()
		if err != nil {
			return nil, err
		}
		yamlNode, ok := node.(*yaml.Node)
		if !ok {
			return nil, fmt.Errorf("map node child returned non *yaml.Node as its result")
		}
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
			Tag:   "!!str",
		}
		mapNode.Content = append(mapNode.Content, keyNode)
		mapNode.Content = append(mapNode.Content, yamlNode)
	}
	return mapNode, nil
}

// MarshalJSON implements json.Marshaler.
func (n *StandardMapNode) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{")
	for i, child := range n.Children() {
		if i.Index > 0 {
			buf.WriteString(",")
		}
		key := fmt.Sprintf("\"%s\"", i.Key)
		buf.WriteString(key)
		buf.WriteString(":")
		marshaller, ok := child.(json.Marshaler)
		if !ok {
			return nil, fmt.Errorf("map node child is not implementing json.Marshaller")
		}
		marshalled, err := marshaller.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(marshalled)
	}
	buf.WriteString("}")

	return buf.Bytes(), nil

}

var _ Node = (*StandardMapNode)(nil)
var _ yaml.Marshaler = (*StandardMapNode)(nil)
var _ json.Marshaler = (*StandardMapNode)(nil)

// getYAMLMarshaler returns the yaml.Marshaller from Node interface.
func getYAMLMarshaler(node Node) (yaml.Marshaler, error) {
	standardRootNode, err := cloneStandardNodeFromNode(node)
	if err != nil {
		return nil, err
	}
	return standardRootNode.(yaml.Marshaler), nil
}

// getJSONMarshalerFromNode returns the json.Marshaller from Node interface.
func getJSONMarshalerFromNode(node Node) (json.Marshaler, error) {
	standardRootNode, err := cloneStandardNodeFromNode(node)
	if err != nil {
		return nil, err
	}
	return standardRootNode.(json.Marshaler), nil
}

// cloneStandardNodeFromNode clones Node interface into Standard***Node.
// Standard**Node implements json.Marshaller and yaml.Marshaller, it allows serializing any implementing Node interface.
func cloneStandardNodeFromNode(node Node) (Node, error) {
	switch node.Type() {
	case ScalarNodeType:
		scalarValue, err := node.NodeScalarValue()
		if err != nil {
			return nil, err
		}
		return &StandardScalarNode[any]{
			value: scalarValue,
		}, nil
	case SequenceNodeType:
		sequence := StandardSequenceNode{
			value: make([]Node, 0),
		}
		for _, child := range node.Children() {
			child, err := cloneStandardNodeFromNode(child)
			if err != nil {
				return nil, err
			}
			sequence.value = append(sequence.value, child)
		}
		return &sequence, nil
	case MapNodeType:
		mapNode := StandardMapNode{
			keys:   make([]string, 0),
			values: make([]Node, 0),
		}
		for key, child := range node.Children() {
			mapNode.keys = append(mapNode.keys, key.Key)
			child, err := cloneStandardNodeFromNode(child)
			if err != nil {
				return nil, err
			}
			mapNode.values = append(mapNode.values, child)
		}
		return &mapNode, nil
	default:
		return nil, fmt.Errorf("unknown node type: %v", node.Type())
	}
}
