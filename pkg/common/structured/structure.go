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

import "errors"

const (
	// InvalidNodeType represents an uninitialized or unrecognized node type.
	InvalidNodeType = 0
	// ScalarNodeType is a NodeType for scalar values.
	// The scalar type currently supported is nil,bool,string,int,float and time.Time.
	ScalarNodeType = 1
	// SequenceNodeType is a NodeType for sequence (array, slice) values.
	SequenceNodeType = 2
	// MapNodeType is a NodeType for map(dictionary) values. It needs to retain the order of keys.
	MapNodeType = 3
)

// ErrNonScalarNode is returned when NodeScalarValue is called on a non-scalar node.
var ErrNonScalarNode = errors.New("this is not a scalar node but called NodeScalarValue method")

// NodeType identifies the structural kind of a Node (scalar, sequence, or map).
type NodeType int

// NodeChildrenIterator is a type to represent the iterator returned from the Children method of Node interface.
type NodeChildrenIterator = func(func(key NodeChildrenKey, value Node) bool)

// Node interface is a recursive data structure representing structured data.
type Node interface {
	// Type returns the NodeType of this node.
	Type() NodeType
	// NodeScalarValue returns the underlying scalar value, or ErrNonScalarNode if the node is not a scalar.
	NodeScalarValue() (any, error)
	// Children returns an iterator over the child entries of this node.
	Children() NodeChildrenIterator
	// Len returns the number of child entries in this node.
	Len() int
	// GetChildByKey returns the child node matching the key.
	// For MapNodeType, implementations should perform direct lookup without iterating all children.
	GetChildByKey(key string) (Node, bool)
}

// NodeChildrenKey identifies a child entry yielded by Node.Children.
type NodeChildrenKey struct {
	// Index is the zero-based position of the child.
	Index int
	// Key is the key of the children in the map.
	// This value is empty when the Node is a sequence and not a map.
	Key string
}
