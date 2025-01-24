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

package testlog

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/parser/yaml/yamlutil"
	"gopkg.in/yaml.v3"
)

func StringField(fieldPath string, value string) TestLogOpt {
	return func(original *yaml.Node) (*yaml.Node, error) {
		node, err := ensureNodeAt(original, fieldPath)
		if err != nil {
			return nil, err
		}
		node.Value = value
		return original, nil
	}
}

func ensureNodeAt(node *yaml.Node, path string) (*yaml.Node, error) {
	path = strings.TrimPrefix(path, ".")
	pathSplitted := strings.Split(path, ".")
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) != 1 {
			return nil, fmt.Errorf("multi root node is not supported")
		}
		return ensureNodeAt(node.Content[0], path)
	}
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("unsupported node kind found: %s(%d)", pathSplitted[0], node.Kind)
	}
	nextPath := strings.TrimPrefix(path, pathSplitted[0])
	nextPath = strings.TrimPrefix(nextPath, ".")
	mapChild, err := yamlutil.GetMapElement(node, pathSplitted[0])

	if errors.Is(err, yamlutil.NodeNotFoundError) {
		if nextPath == "" {
			// This node must be a scalar
			child := yamlutil.NewMapElementWithScalarValue(pathSplitted[0], "")
			node.Content = append(node.Content, child...)
			return child[1], nil
		} else {
			child := yamlutil.NewEmptyMapNode()
			key := yamlutil.NewScalarNode(pathSplitted[0])
			node.Content = append(node.Content, key)
			node.Content = append(node.Content, child)
			return ensureNodeAt(child, nextPath)
		}
	} else {
		if nextPath == "" {
			return mapChild, nil
		}
		return ensureNodeAt(mapChild, nextPath)
	}
}
