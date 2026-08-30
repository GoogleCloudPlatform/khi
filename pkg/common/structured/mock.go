// Copyright 2026 Google LLC
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
	"reflect"
)

// MockNode is a specialized Node for unit tests that stores typed mock values.
type MockNode struct {
	values map[reflect.Type]any
}

var _ Node = (*MockNode)(nil)

// NewMockNode instantiates a MockNode containing the given mock values.
func NewMockNode(mockValues ...any) *MockNode {
	m := &MockNode{
		values: make(map[reflect.Type]any, len(mockValues)),
	}
	for _, val := range mockValues {
		if val == nil {
			continue
		}
		t := reflect.TypeOf(val)
		m.values[t] = val
	}
	return m
}

// Type implements Node.
func (m *MockNode) Type() NodeType {
	return MapNodeType
}

// NodeScalarValue implements Node.
func (m *MockNode) NodeScalarValue() (any, error) {
	return nil, ErrNonScalarNode
}

// Children implements Node.
func (m *MockNode) Children() NodeChildrenIterator {
	return func(f func(key NodeChildrenKey, value Node) bool) {}
}

// Len implements Node.
func (m *MockNode) Len() int {
	return len(m.values)
}

// Get retrieves a mock value of the target type (supports both value and pointer registrations).
func (m *MockNode) Get(targetType reflect.Type) (any, bool) {
	if val, ok := m.values[targetType]; ok {
		return val, true
	}
	// Fallback check if pointer was registered but value was requested, or vice versa.
	if targetType.Kind() == reflect.Pointer {
		if val, ok := m.values[targetType.Elem()]; ok {
			// val is value type T, return pointer to it.
			valCopy := reflect.New(targetType.Elem())
			valCopy.Elem().Set(reflect.ValueOf(val))
			return valCopy.Interface(), true
		}
	} else {
		ptrType := reflect.PointerTo(targetType)
		if val, ok := m.values[ptrType]; ok {
			// val is pointer type *T, return dereferenced value T.
			elem := reflect.ValueOf(val).Elem().Interface()
			return elem, true
		}
	}
	return nil, false
}

// GetMock retrieves a mocked value of type T from the NodeReader if it wraps a MockNode.
func GetMock[T any](reader *NodeReader) (T, bool) {
	if reader == nil || reader.Node == nil {
		return *new(T), false
	}
	mockNode, ok := reader.Node.(*MockNode)
	if !ok {
		return *new(T), false
	}
	val, found := mockNode.Get(reflect.TypeFor[T]())
	if !found {
		return *new(T), false
	}
	typedVal, ok := val.(T)
	return typedVal, ok
}
