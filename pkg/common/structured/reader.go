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

package structured

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// ErrFieldNotFound is returned when a requested field is not found in the node structure.
var ErrFieldNotFound = errors.New("field not found")

// NodeReaderChildrenIterator is a type that represents an iterator function for navigating
type NodeReaderChildrenIterator = func(func(key NodeChildrenKey, value NodeReader) bool)

// NodeReader provides a convenient way to read values from a node structure.
// It offers type-safe accessor methods and path navigation capabilities.
type NodeReader struct {
	Node
}

// NewNodeReader creates a new NodeReader instance from a given Node.
func NewNodeReader(node Node) *NodeReader {
	return &NodeReader{node}
}

// Children returns an iterator for navigating through readers of the children of this node.
func (n *NodeReader) Children() NodeReaderChildrenIterator {
	return func(callback func(key NodeChildrenKey, value NodeReader) bool) {
		for key, value := range n.Node.Children() {
			if !callback(key, NodeReader{value}) {
				return
			}
		}
	}
}

// WithKeyOrder returns a new NodeReader wrapping the node with the specified key order.
func (n *NodeReader) WithKeyOrder(priorityKeys ...string) *NodeReader {
	return &NodeReader{
		Node: WithKeyOrder(n.Node, priorityKeys...),
	}
}

// Has checks if a field exists at the pre-compiled FieldPath.
func (n *NodeReader) Has(path FieldPath) bool {
	_, err := n.GetNode(path)
	return err == nil
}

// GetNode obtains the Node at the given pre-compiled FieldPath.
func (n *NodeReader) GetNode(path FieldPath) (Node, error) {
	if n == nil || n.Node == nil {
		return nil, ErrFieldNotFound
	}
	if len(path.segments) == 0 {
		return n.Node, nil
	}
	currentNode := n.Node
	for i := 0; i < len(path.segments); i++ {
		// Fast-path: direct handle lookup on StandardMapNode without allocating closures.
		if mapNode, ok := currentNode.(*StandardMapNode); ok {
			child, found := mapNode.GetChildByHandle(path.handles[i])
			if !found {
				return nil, ErrFieldNotFound
			}
			currentNode = child
			continue
		}

		// Fallback for custom Node implementations.
		found := false
		seg := path.segments[i]
		for key, value := range currentNode.Children() {
			if key.Key == seg {
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

// GetReader obtains a NodeReader at the given pre-compiled FieldPath.
func (n *NodeReader) GetReader(path FieldPath) (*NodeReader, error) {
	node, err := n.GetNode(path)
	if err != nil {
		return nil, err
	}
	return &NodeReader{node}, nil
}

// Serialize serializes the structured data at the given pre-compiled FieldPath with the given NodeSerializer.
func (n *NodeReader) Serialize(path FieldPath, serializer NodeSerializer) ([]byte, error) {
	node, err := n.GetNode(path)
	if err != nil {
		return nil, err
	}
	return serializer.Serialize(node)
}

// ReadBool retrieves a boolean value from the pre-compiled FieldPath.
// Returns an error if the field doesn't exist or cannot be cast to a boolean.
func (n *NodeReader) ReadBool(path FieldPath) (bool, error) {
	node, err := n.GetNode(path)
	if err != nil {
		return false, err
	}
	return getScalarAs[bool](node)
}

// ReadBoolOrDefault retrieves a boolean value from the pre-compiled FieldPath.
func (n *NodeReader) ReadBoolOrDefault(path FieldPath, defaultValue bool) bool {
	node, err := n.GetNode(path)
	if err != nil {
		return defaultValue
	}
	val, err := getScalarAs[bool](node)
	if err != nil {
		return defaultValue
	}
	return val
}

// ReadString retrieves a string value from the pre-compiled FieldPath.
// Returns an error if the field doesn't exist or cannot be cast to a string.
func (n *NodeReader) ReadString(path FieldPath) (string, error) {
	node, err := n.GetNode(path)
	if err != nil {
		return "", err
	}
	return getScalarAs[string](node)
}

// ReadStringOrDefault retrieves a string value from the pre-compiled FieldPath.
func (n *NodeReader) ReadStringOrDefault(path FieldPath, defaultValue string) string {
	node, err := n.GetNode(path)
	if err != nil {
		return defaultValue
	}
	val, err := getScalarAs[string](node)
	if err != nil {
		return defaultValue
	}
	return val
}

// ReadInt retrieves an integer value from the pre-compiled FieldPath.
// Returns an error if the field doesn't exist or cannot be cast to an integer.
func (n *NodeReader) ReadInt(path FieldPath) (int, error) {
	node, err := n.GetNode(path)
	if err != nil {
		return 0, err
	}
	return getScalarAs[int](node)
}

// ReadIntOrDefault retrieves an integer value from the pre-compiled FieldPath.
func (n *NodeReader) ReadIntOrDefault(path FieldPath, defaultValue int) int {
	node, err := n.GetNode(path)
	if err != nil {
		return defaultValue
	}
	val, err := getScalarAs[int](node)
	if err != nil {
		return defaultValue
	}
	return val
}

// ReadFloat retrieves a floating-point value from the pre-compiled FieldPath.
// Returns an error if the field doesn't exist or cannot be cast to a float64.
func (n *NodeReader) ReadFloat(path FieldPath) (float64, error) {
	node, err := n.GetNode(path)
	if err != nil {
		return 0, err
	}
	return getScalarAs[float64](node)
}

// ReadFloatOrDefault retrieves a floating-point value from the pre-compiled FieldPath.
func (n *NodeReader) ReadFloatOrDefault(path FieldPath, defaultValue float64) float64 {
	node, err := n.GetNode(path)
	if err != nil {
		return defaultValue
	}
	val, err := getScalarAs[float64](node)
	if err != nil {
		return defaultValue
	}
	return val
}

// ReadTimestamp retrieves a timestamp value from the pre-compiled FieldPath.
// Returns an error if the field doesn't exist or cannot be cast to a time.Time.
func (n *NodeReader) ReadTimestamp(path FieldPath) (time.Time, error) {
	node, err := n.GetNode(path)
	if err != nil {
		return time.Time{}, err
	}
	t, err := getScalarAs[time.Time](node)
	if err != nil {
		tStr, err := getScalarAs[string](node)
		if err != nil {
			return time.Time{}, err
		}
		return common.ParseTime(tStr)
	}
	return t, nil
}

// ReadTimestampOrDefault retrieves a timestamp value from the pre-compiled FieldPath.
func (n *NodeReader) ReadTimestampOrDefault(path FieldPath, defaultValue time.Time) time.Time {
	node, err := n.GetNode(path)
	if err != nil {
		return defaultValue
	}
	t, err := getScalarAs[time.Time](node)
	if err != nil {
		tStr, err := getScalarAs[string](node)
		if err != nil {
			return defaultValue
		}
		t, err = common.ParseTime(tStr)
		if err != nil {
			return defaultValue
		}
	}
	return t
}

// ReadReflect unmarshals the structured data into a given type after the given FieldPath.
// TODO: ReadReflect currently marshals and unmarshals the structured data into the target.
//
//	There should be room to improve this behavior regarding the performance.
func ReadReflect[T any](r *NodeReader, path FieldPath, target T) error {
	rawJSON, err := r.Serialize(path, &JSONNodeSerializer{})
	if err != nil {
		return err
	}
	err = json.Unmarshal(rawJSON, &target)
	if err != nil {
		return err
	}
	return nil
}

// ReadReflectK8sRuntimeObject unmarshals the structured data into a type implementing runtime.Object.
func ReadReflectK8sRuntimeObject[T runtime.Object](r *NodeReader, path FieldPath, target T) error {
	rawJSON, err := r.Serialize(path, &JSONNodeSerializer{})
	if err != nil {
		return err
	}
	scheme := runtime.NewScheme()
	codecFactory := serializer.NewCodecFactory(scheme)
	deserializer := codecFactory.UniversalDeserializer()
	_, _, err = deserializer.Decode(rawJSON, nil, target)
	if err != nil {
		return fmt.Errorf("failed to decode JSON as runtime.Object: \n source: %s\nerror:%s", string(rawJSON), err.Error())
	}
	return nil
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

// getScalarAsString get the scalar node value as string.
func getScalarAsString(scalarNode Node) (string, error) {
	result, err := getScalarAs[string](scalarNode)
	if err == nil {
		return result, nil
	}
	resultInt, err := getScalarAs[int](scalarNode)
	if err == nil {
		return strconv.Itoa(resultInt), nil
	}
	resultBool, err := getScalarAs[bool](scalarNode)
	if err == nil {
		return strconv.FormatBool(resultBool), nil
	}
	resultTime, err := getScalarAs[time.Time](scalarNode)
	if err == nil {
		return resultTime.String(), nil
	}
	resultFloat, err := getScalarAs[float64](scalarNode)
	if err == nil {
		return strconv.FormatFloat(resultFloat, 'f', -1, 64), nil
	}
	return "", fmt.Errorf("failed to cast value %v to type string", scalarNode)
}
