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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gopkg.in/yaml.v3"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestNewEmptyNode(t *testing.T) {
	expected := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: make([]*yaml.Node, 0),
	}

	actual := NewEmptyMapNode()

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("(-expected,+actual)%s", diff)
	}
}

func TestNewMapWithScalarValue(t *testing.T) {
	cases := []struct {
		description string
		key         string
		value       string
		expected    []*yaml.Node
	}{
		{
			description: "key=foo,value=bar",
			key:         "foo",
			value:       "bar",
			expected: []*yaml.Node{
				{
					Kind:    yaml.ScalarNode,
					Content: make([]*yaml.Node, 0),
					Value:   "foo",
				},
				{
					Kind:    yaml.ScalarNode,
					Content: make([]*yaml.Node, 0),
					Value:   "bar",
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			actual := NewMapElementWithScalarValue(c.key, c.value)
			if diff := cmp.Diff(c.expected, actual); diff != "" {
				t.Errorf("(-expected,+actual)%s", diff)
			}
		})
	}
}

func TestNewScalarNode(t *testing.T) {
	cases := []struct {
		description string
		value       string
		expected    *yaml.Node
	}{
		{
			description: "value=foo",
			value:       "foo",
			expected: &yaml.Node{
				Kind:    yaml.ScalarNode,
				Content: make([]*yaml.Node, 0),
				Value:   "foo",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			actual := NewScalarNode(c.value)

			if diff := cmp.Diff(c.expected, actual); diff != "" {
				t.Errorf("(-expected,+actual)%s", diff)
			}
		})
	}
}

func TestDecomposeMapElement(t *testing.T) {
	mapNode := NewEmptyMapNode()
	mapNode.Content = append(mapNode.Content, NewMapElementWithScalarValue("foo", "foo-value")...)
	mapNode.Content = append(mapNode.Content, NewMapElementWithScalarValue("bar", "bar-value")...)
	mapNode.Content = append(mapNode.Content, NewMapElementWithScalarValue("qux", "qux-value")...)

	cases := []struct {
		description string
		mapNode     *yaml.Node
		index       int
		expected    struct {
			Key   string
			Value string
		}
	}{
		{
			description: "first map element",
			mapNode:     mapNode,
			index:       0,
			expected: struct {
				Key   string
				Value string
			}{
				Key:   "foo",
				Value: "foo-value",
			},
		},
		{
			description: "last map element",
			mapNode:     mapNode,
			index:       2,
			expected: struct {
				Key   string
				Value string
			}{
				Key:   "qux",
				Value: "qux-value",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			actualKey, actualValue := DecomposeMapElement(c.mapNode, c.index)
			actual := struct {
				Key   string
				Value string
			}{
				Key:   actualKey,
				Value: actualValue.Value,
			}

			if diff := cmp.Diff(c.expected, actual); diff != "" {
				t.Errorf("(-expected,+actual)%s", diff)
			}
		})
	}
}

func TestGetMapElement(t *testing.T) {
	mapNode := NewEmptyMapNode()
	mapNode.Content = append(mapNode.Content, NewMapElementWithScalarValue("foo", "foo-value")...)
	mapNode.Content = append(mapNode.Content, NewMapElementWithScalarValue("bar", "bar-value")...)
	mapNode.Content = append(mapNode.Content, NewMapElementWithScalarValue("qux", "qux-value")...)

	cases := []struct {
		description string
		key         string
		mapNode     *yaml.Node
		expected    struct {
			Result *yaml.Node
			Error  error
		}
	}{
		{
			description: "An existing element",
			key:         "bar",
			mapNode:     mapNode,
			expected: struct {
				Result *yaml.Node
				Error  error
			}{
				Result: mapNode.Content[3],
				Error:  nil,
			},
		},
		{
			description: "Non existing element",
			key:         "quux",
			mapNode:     mapNode,
			expected: struct {
				Result *yaml.Node
				Error  error
			}{
				Result: nil,
				Error:  NodeNotFoundError,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			actualResult, actualError := GetMapElement(c.mapNode, c.key)
			actual := struct {
				Result *yaml.Node
				Error  error
			}{
				Result: actualResult,
				Error:  actualError,
			}

			if diff := cmp.Diff(c.expected, actual, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("(-expected,+actual)%s", diff)
			}
		})
	}
}
