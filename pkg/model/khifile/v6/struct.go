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

// Package khifilev6 provides models and utilities for KHI file format version 6.
package khifilev6

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
	"unsafe"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
)

// FieldPathSeparator is the separator used for field paths in InternedStruct.
// \x00 is hardly included in YAML fields. So \x00 is a good separator.
const fieldPathSeparator = "\x00"

// ToInternedStruct converts a structured.Node to an InternedStructRef.
// The input node must be a MapNodeType. It flattens nested maps by joining keys with a null character (\x00).
func ToInternedStruct(node structured.Node, pool *InternPool) (*InternStructRef, error) {
	if node.Type() != structured.MapNodeType {
		return nil, fmt.Errorf("expected map node, got %v", node.Type())
	}

	flattenedKeys := make([]string, 0, 16)
	flattenedValues := make([]structured.Node, 0, 16)
	keyBuf := make([]byte, 0, 64)
	err := flattenNodeHelper(node, keyBuf, true, &flattenedKeys, &flattenedValues)
	if err != nil {
		return nil, err
	}

	fieldSetRef := pool.InternFieldSet(flattenedKeys)

	// Fast-path: compute deterministic structKey directly from node values.
	// If the struct is already interned, we can return its reference immediately
	// without allocating []*pb.InternedValue and individual *pb.InternedValue proto messages.
	key, err := structKeyFromNodes(fieldSetRef.id, flattenedValues, pool, keyBuf)
	if err != nil {
		return nil, err
	}

	if id, ok := pool.structToID.Load(key); ok {
		return &InternStructRef{pool: pool, id: id.(uint32)}, nil
	}

	newID := pool.idGen.New(id.Struct)
	if err := pool.flatStructs.StoreFromNodes(newID, fieldSetRef.id, flattenedValues, pool); err != nil {
		return nil, err
	}

	actual, loaded := pool.structToID.LoadOrStore(key, newID)
	if loaded {
		return &InternStructRef{pool: pool, id: actual.(uint32)}, nil
	}

	pool.stageStruct(newID)

	return &InternStructRef{pool: pool, id: newID}, nil
}

// flattenNode is a helper function to recursively flatten map nodes.
// It flattens nested maps but preserves empty maps as leaves.
// Note: This function assumes that there are no circular references in the node tree.
// Circular references are not expected as structured.Node represents parsed tree data.
func flattenNode(node structured.Node, prefix string, isRoot bool, keys *[]string, values *[]structured.Node) error {
	keyBuf := make([]byte, 0, len(prefix)+64)
	if prefix != "" {
		keyBuf = append(keyBuf, prefix...)
	}
	return flattenNodeHelper(node, keyBuf, isRoot, keys, values)
}

func flattenNodeHelper(node structured.Node, keyBuf []byte, isRoot bool, keys *[]string, values *[]structured.Node) error {
	if node.Type() != structured.MapNodeType {
		return fmt.Errorf("expected map node in flattenNode, got %v", node.Type())
	}

	for key, child := range node.Children() {
		origLen := len(keyBuf)
		if !isRoot {
			keyBuf = append(keyBuf, fieldPathSeparator...)
		}
		keyBuf = append(keyBuf, key.Key...)

		if child.Type() == structured.MapNodeType {
			if child.Len() == 0 {
				*keys = append(*keys, string(keyBuf))
				*values = append(*values, child)
			} else {
				err := flattenNodeHelper(child, keyBuf, false, keys, values)
				if err != nil {
					return err
				}
			}
		} else {
			*keys = append(*keys, string(keyBuf))
			*values = append(*values, child)
		}
		keyBuf = keyBuf[:origLen]
	}
	return nil
}

func structKeyFromNodes(fieldPathSetID uint32, nodes []structured.Node, pool *InternPool, buf []byte) (string, error) {
	buf = buf[:0]
	buf = binary.LittleEndian.AppendUint32(buf, fieldPathSetID)
	for _, n := range nodes {
		var err error
		buf, err = appendNodeKey(buf, n, pool)
		if err != nil {
			return "", err
		}
	}
	return unsafe.String(unsafe.SliceData(buf), len(buf)), nil
}

func appendNodeKey(buf []byte, node structured.Node, pool *InternPool) ([]byte, error) {
	if node == nil {
		return append(buf, 0x00), nil
	}
	switch node.Type() {
	case structured.ScalarNodeType:
		val, err := node.NodeScalarValue()
		if err != nil {
			return nil, err
		}
		if val == nil {
			return append(buf, 0x01), nil
		}
		switch v := val.(type) {
		case bool:
			if v {
				return append(buf, 0x02, 0x01), nil
			}
			return append(buf, 0x02, 0x00), nil
		case int:
			buf = append(buf, 0x03)
			return binary.LittleEndian.AppendUint64(buf, uint64(v)), nil
		case float64:
			buf = append(buf, 0x04)
			return binary.LittleEndian.AppendUint64(buf, math.Float64bits(v)), nil
		case string:
			strRef := pool.InternString(v)
			buf = append(buf, 0x05)
			return binary.LittleEndian.AppendUint32(buf, strRef.id), nil
		case time.Time:
			buf = append(buf, 0x07)
			buf = binary.LittleEndian.AppendUint64(buf, uint64(v.Unix()))
			return binary.LittleEndian.AppendUint32(buf, uint32(v.Nanosecond())), nil
		default:
			return nil, fmt.Errorf("unsupported scalar type: %T", v)
		}
	case structured.SequenceNodeType:
		buf = append(buf, 0x08)
		buf = binary.LittleEndian.AppendUint32(buf, uint32(node.Len()))
		for _, child := range node.Children() {
			var err error
			buf, err = appendNodeKey(buf, child, pool)
			if err != nil {
				return nil, err
			}
		}
		return buf, nil
	case structured.MapNodeType:
		s, err := ToInternedStruct(node, pool)
		if err != nil {
			return nil, err
		}
		buf = append(buf, 0x06)
		return binary.LittleEndian.AppendUint32(buf, s.id), nil
	default:
		return append(buf, 0xFF), nil
	}
}

// FromInternedStruct converts an InternedStruct back to a structured.Node.
func FromInternedStruct(s *pb.InternedStruct, pool ReadonlyPool) (structured.Node, error) {
	if s == nil {
		return nil, fmt.Errorf("InternedStruct is nil")
	}
	if s.FieldPathSetId == nil {
		return nil, fmt.Errorf("FieldPathSetId is nil")
	}

	fieldSetIDs := pool.ResolveFieldSetFromID(*s.FieldPathSetId)
	keys := make([]string, len(fieldSetIDs))
	for i, id := range fieldSetIDs {
		keys[i] = pool.ResolveStringFromID(id)
	}

	if len(keys) != len(s.Values) {
		return nil, fmt.Errorf("length mismatch: keys=%d, values=%d", len(keys), len(s.Values))
	}

	var values []structured.Node
	for _, val := range s.Values {
		node, err := FromInternedValue(val, pool)
		if err != nil {
			return nil, err
		}
		values = append(values, node)
	}

	return unflattenNodes(keys, values)
}

// unflattenNodes reconstructs a nested map structure from flattened keys and values.
//
// Algorithm:
// It reverses the operation performed by `flattenNode`. Given a list of full paths
// (e.g., ["a\x00b", "a\x00c", "d"]) and their corresponding leaf values, it builds
// the original nested `structured.MapNode`.
//
//  1. First, it iterates through all provided keys and splits them by the separator ('\x00').
//  2. It groups the remaining path components and their corresponding values by the
//     first path component (the top-level key).
//     For example, "a\x00b" and "a\x00c" are both grouped under the top-level key "a".
//  3. It iterates through these grouped top-level keys to reconstruct the child nodes:
//     - If a group contains exactly one path and that path has no sub-components
//     (i.e., it was a direct leaf like "d"), the corresponding value is attached directly.
//     - If the paths have sub-components, it indicates a nested map. It rejoins the
//     remaining sub-components into flattened keys and recursively calls `unflattenNodes`
//     to reconstruct that nested map.
//     - If it detects that a key is being used as both a leaf value and a nested map,
//     it returns a conflict error.
func unflattenNodes(keys []string, values []structured.Node) (structured.Node, error) {
	var uniqueKeys []string
	groupedPaths := make(map[string][][]string)
	groupedValues := make(map[string][]structured.Node)

	for i, key := range keys {
		path := strings.Split(key, "\x00")
		first := path[0]
		if len(groupedPaths[first]) == 0 {
			uniqueKeys = append(uniqueKeys, first)
		}
		groupedPaths[first] = append(groupedPaths[first], path[1:])
		groupedValues[first] = append(groupedValues[first], values[i])
	}

	var childNodes []structured.Node
	for _, first := range uniqueKeys {
		paths := groupedPaths[first]
		vals := groupedValues[first]

		if len(paths) == 1 && len(paths[0]) == 0 {
			// Leaf node
			childNodes = append(childNodes, vals[0])
		} else {
			// Nested map
			for _, p := range paths {
				if len(p) == 0 {
					return nil, fmt.Errorf("conflict at key %s", first)
				}
			}

			var nestedKeys []string
			for _, p := range paths {
				nestedKeys = append(nestedKeys, strings.Join(p, "\x00"))
			}

			childNode, err := unflattenNodes(nestedKeys, vals)
			if err != nil {
				return nil, err
			}
			childNodes = append(childNodes, childNode)
		}
	}

	return structured.NewStandardMap(uniqueKeys, childNodes), nil
}

// FromInternedValue converts an InternedValue back to a structured.Node.
func FromInternedValue(v *pb.InternedValue, pool ReadonlyPool) (structured.Node, error) {
	if v == nil {
		return nil, fmt.Errorf("InternedValue is nil")
	}
	switch kind := v.Kind.(type) {
	case *pb.InternedValue_NullValue:
		return structured.NewStandardScalarNode[any](nil), nil
	case *pb.InternedValue_BoolValue:
		return structured.NewStandardScalarNode(kind.BoolValue), nil
	case *pb.InternedValue_StringValue:
		return structured.NewStandardScalarNode(pool.ResolveStringFromID(kind.StringValue)), nil
	case *pb.InternedValue_Int64Value:
		return structured.NewStandardScalarNode(int(kind.Int64Value)), nil
	case *pb.InternedValue_DoubleValue:
		return structured.NewStandardScalarNode(kind.DoubleValue), nil
	case *pb.InternedValue_TimestampValue:
		return structured.NewStandardScalarNode(kind.TimestampValue.AsTime()), nil
	case *pb.InternedValue_ListValue:
		elements := make([]structured.Node, 0, len(kind.ListValue.GetValues()))
		for _, elem := range kind.ListValue.GetValues() {
			node, err := FromInternedValue(elem, pool)
			if err != nil {
				return nil, err
			}
			elements = append(elements, node)
		}
		return structured.NewStandardSequenceNode(elements), nil
	case *pb.InternedValue_StructId:
		resolved := pool.ResolveStructFromID(kind.StructId)
		if resolved == nil {
			return nil, fmt.Errorf("struct id %d not found in pool", kind.StructId)
		}
		return FromInternedStruct(resolved, pool)
	case *pb.InternedValue_StructValue:
		return FromInternedStruct(kind.StructValue, pool)
	default:
		return nil, fmt.Errorf("unknown InternedValue kind: %T", kind)
	}
}
