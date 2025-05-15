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
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrFieldNotFound is returned when a requested field is not found in the node structure.
var ErrFieldNotFound = errors.New("field not found")

// NodeReaderChildrenIterator is a type that represents an iterator function for navigating
type NodeReaderChildrenIterator = func(func(key NodeChildrenKey, value NodeReader) bool)

// NodeReader provides a convenient way to read values from a node structure.
// It offers type-safe accessor methods and path navigation capabilities.
type NodeReader struct {
	node Node
}

// NewNodeReader creates a new NodeReader instance from a given Node.
func NewNodeReader(node Node) *NodeReader {
	return &NodeReader{node: node}
}

// Has checks if a field exists at the specified path in the node structure.
// Returns true if the field exists, false otherwise.
func (n *NodeReader) Has(fieldPath string) bool {
	_, err := n.getNode(fieldPath)
	return err == nil
}

// GetReader obtains the NodeReader from the specified field path.
func (n *NodeReader) GetReader(fieldPath string) (*NodeReader, error) {
	node, err := n.getNode(fieldPath)
	if err != nil {
		return nil, err
	}
	return &NodeReader{node: node}, nil
}

// Serialize serializes the structured data with the given NodeSerializer.
func (n *NodeReader) Serialize(serializer NodeSerializer) ([]byte, error) {
	return serializer.Serialize(n.node)
}

// Children returns an iterator for navigating through readers of the children of this node.
func (n *NodeReader) Children() NodeReaderChildrenIterator {
	return func(callback func(key NodeChildrenKey, value NodeReader) bool) {
		for key, value := range n.node.Children() {
			if !callback(key, NodeReader{node: value}) {
				return
			}
		}
	}
}

// ReadBool retrieves a boolean value from the specified field path.
// Returns an error if the field doesn't exist or cannot be cast to a boolean.
func (n *NodeReader) ReadBool(fieldPath string) (bool, error) {
	return getScalarValueAt[bool](fieldPath, n)
}

// ReadString retrieves a string value from the specified field path.
// Returns an error if the field doesn't exist or cannot be cast to a string.
func (n *NodeReader) ReadString(fieldPath string) (string, error) {
	return getScalarValueAt[string](fieldPath, n)
}

// ReadInt retrieves an integer value from the specified field path.
// Returns an error if the field doesn't exist or cannot be cast to an integer.
func (n *NodeReader) ReadInt(fieldPath string) (int, error) {
	return getScalarValueAt[int](fieldPath, n)
}

// ReadFloat retrieves a floating-point value from the specified field path.
// Returns an error if the field doesn't exist or cannot be cast to a float64.
func (n *NodeReader) ReadFloat(fieldPath string) (float64, error) {
	return getScalarValueAt[float64](fieldPath, n)
}

// ReadTimestamp retrieves a timestamp value from the specified field path.
// Returns an error if the field doesn't exist or cannot be cast to a time.Time.
func (n *NodeReader) ReadTimestamp(fieldPath string) (time.Time, error) {
	return getScalarValueAt[time.Time](fieldPath, n)
}

// ReadStringOrDefault retrieves a string value from the specified field path.
// Returns the provided default value if the field doesn't exist or an error occurs.
func (n *NodeReader) ReadStringOrDefault(fieldPath string, defaultValue string) string {
	return getScalarValueOrDefaultAt(fieldPath, defaultValue, n)
}

// ReadIntOrDefault retrieves an integer value from the specified field path.
// Returns the provided default value if the field doesn't exist or an error occurs.
func (n *NodeReader) ReadIntOrDefault(fieldPath string, defaultValue int) int {
	return getScalarValueOrDefaultAt(fieldPath, defaultValue, n)
}

// ReadFloatOrDefault retrieves a floating-point value from the specified field path.
// Returns the provided default value if the field doesn't exist or an error occurs.
func (n *NodeReader) ReadFloatOrDefault(fieldPath string, defaultValue float64) float64 {
	return getScalarValueOrDefaultAt(fieldPath, defaultValue, n)
}

// ReadTimestampOrDefault retrieves a timestamp value from the specified field path.
// Returns the provided default value if the field doesn't exist or an error occurs.
func (n *NodeReader) ReadTimestampOrDefault(fieldPath string, defaultValue time.Time) time.Time {
	return getScalarValueOrDefaultAt(fieldPath, defaultValue, n)
}

// ReadBoolOrDefault retrieves a boolean value from the specified field path.
// Returns the provided default value if the field doesn't exist or an error occurs.
func (n *NodeReader) ReadBoolOrDefault(fieldPath string, defaultValue bool) bool {
	return getScalarValueOrDefaultAt(fieldPath, defaultValue, n)
}

func (n *NodeReader) getNode(fieldPath string) (Node, error) {
	if fieldPath == "" {
		return n.node, nil
	}
	pathSegments := parseFieldPath(fieldPath)
	currentNode := n.node
	for pathCursor := 0; pathCursor < len(pathSegments); pathCursor++ {
		found := false
		for key, value := range currentNode.Children() {
			if key.Key == pathSegments[pathCursor] {
				currentNode = value
				found = true
				break
			}
		}
		if !found {
			return nil, ErrFieldNotFound
		}
	}
	return currentNode, nil
}

// parseFieldPath splits a field path string according to specified rules.
// It uses '.' as a delimiter, but '\.' is treated as an escaped literal dot.
func parseFieldPath(s string) []string {
	var result []string
	var currentSegment strings.Builder
	isEscaped := false

	for _, r := range s {
		if isEscaped {
			if r == '.' {
				currentSegment.WriteRune('.') // '\.' is treated as a literal '.' and added to the current segment
			} else {
				// If '\' is followed by something other than '.', treat '\' as a literal character too
				currentSegment.WriteRune('\\')
				currentSegment.WriteRune(r)
			}
			isEscaped = false
		} else {
			if r == '\\' {
				isEscaped = true
			} else if r == '.' {
				result = append(result, currentSegment.String())
				currentSegment.Reset() // Reset the current segment
			} else {
				currentSegment.WriteRune(r)
			}
		}
	}

	if isEscaped {
		// If the string ends with '\', treat it as a literal '\'
		currentSegment.WriteRune('\\')
	}
	result = append(result, currentSegment.String())

	return result
}

func getScalarValueOrDefaultAt[T any](fieldPath string, defaultValue T, nodeReader *NodeReader) T {
	value, err := getScalarValueAt[T](fieldPath, nodeReader)
	if err != nil {
		return defaultValue
	}
	return value

}

func getScalarValueAt[T any](fieldPath string, nodeReader *NodeReader) (T, error) {
	holderNode, err := nodeReader.getNode(fieldPath)
	if err != nil {
		return *new(T), err
	}
	return getScalarAs[T](holderNode)
}

func getScalarAs[T any](scalarNode Node) (T, error) {
	anyValue, err := scalarNode.NodeScalarValue()
	if err != nil {
		return *new(T), err
	}
	if anyValue == nil {
		return *new(T), nil
	}
	if value, ok := anyValue.(T); ok {
		return value, nil
	}
	return *new(T), fmt.Errorf("failed to cast value %v to type %T", anyValue, *new(T))
}
