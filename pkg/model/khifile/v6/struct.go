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
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FieldPathSeparator is the separator used for field paths in InternedStruct.
// \x00 is hardly included in YAML fields. So \x00 is a good separator.
const fieldPathSeparator = "\x00"

// ToInternedStruct converts a structured.Node to an InternedStruct.
// The input node must be a MapNodeType. It flattens nested maps by joining keys with a null character (\x00).
func ToInternedStruct(node structured.Node, pool *InternPool) (*pb.InternedStruct, error) {
	if node.Type() != structured.MapNodeType {
		return nil, fmt.Errorf("expected map node, got %v", node.Type())
	}

	var flattenedKeys []string
	var flattenedValues []structured.Node
	err := flattenNode(node, "", true, &flattenedKeys, &flattenedValues)
	if err != nil {
		return nil, err
	}

	fieldSetRef := pool.InternFieldSet(flattenedKeys)

	var values []*pb.InternedValue
	for _, valNode := range flattenedValues {
		val, err := ToInternedValue(valNode, pool)
		if err != nil {
			return nil, err
		}
		values = append(values, val)
	}

	id := fieldSetRef.id
	return &pb.InternedStruct{
		FieldPathSetId: &id,
		Values:         values,
	}, nil
}

// flattenNode is a helper function to recursively flatten map nodes.
// It flattens nested maps but preserves empty maps as leaves.
// Note: This function assumes that there are no circular references in the node tree.
// Circular references are not expected as structured.Node represents parsed tree data.
func flattenNode(node structured.Node, prefix string, isRoot bool, keys *[]string, values *[]structured.Node) error {
	if node.Type() != structured.MapNodeType {
		return fmt.Errorf("expected map node in flattenNode, got %v", node.Type())
	}

	for key, child := range node.Children() {
		fullKey := key.Key
		if !isRoot {
			fullKey = prefix + fieldPathSeparator + key.Key
		}

		if child.Type() == structured.MapNodeType {
			if child.Len() == 0 {
				*keys = append(*keys, fullKey)
				*values = append(*values, child)
			} else {
				err := flattenNode(child, fullKey, false, keys, values)
				if err != nil {
					return err
				}
			}
		} else {
			*keys = append(*keys, fullKey)
			*values = append(*values, child)
		}
	}
	return nil
}

// ToInternedValue converts a structured.Node to an InternedValue.
func ToInternedValue(node structured.Node, pool *InternPool) (*pb.InternedValue, error) {
	switch node.Type() {
	case structured.ScalarNodeType:
		return scalarToInternedValue(node, pool)
	case structured.SequenceNodeType:
		return sequenceToInternedValue(node, pool)
	case structured.MapNodeType:
		return mapToInternedValue(node, pool)
	default:
		return nil, fmt.Errorf("unknown node type: %v", node.Type())
	}
}

func scalarToInternedValue(node structured.Node, pool *InternPool) (*pb.InternedValue, error) {
	val, err := node.NodeScalarValue()
	if err != nil {
		return nil, err
	}
	if val == nil {
		return &pb.InternedValue{
			Kind: &pb.InternedValue_NullValue{
				NullValue: structpb.NullValue_NULL_VALUE,
			},
		}, nil
	}
	switch v := val.(type) {
	case bool:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_BoolValue{
				BoolValue: v,
			},
		}, nil
	case string:
		strRef := pool.InternString(v)
		return &pb.InternedValue{
			Kind: &pb.InternedValue_StringValue{
				StringValue: strRef.id,
			},
		}, nil
	case int:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_Int64Value{
				Int64Value: int64(v),
			},
		}, nil
	case float64:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_DoubleValue{
				DoubleValue: v,
			},
		}, nil
	case time.Time:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_TimestampValue{
				TimestampValue: timestamppb.New(v),
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported scalar type: %T", v)
	}
}

func sequenceToInternedValue(node structured.Node, pool *InternPool) (*pb.InternedValue, error) {
	listValues := make([]*pb.InternedValue, 0, node.Len())
	for _, child := range node.Children() {
		val, err := ToInternedValue(child, pool)
		if err != nil {
			return nil, err
		}
		listValues = append(listValues, val)
	}
	return &pb.InternedValue{
		Kind: &pb.InternedValue_ListValue{
			ListValue: &pb.InternedListValue{
				Values: listValues,
			},
		},
	}, nil
}

func mapToInternedValue(node structured.Node, pool *InternPool) (*pb.InternedValue, error) {
	s, err := ToInternedStruct(node, pool)
	if err != nil {
		return nil, err
	}
	return &pb.InternedValue{
		Kind: &pb.InternedValue_StructValue{
			StructValue: s,
		},
	}, nil
}

// FromInternedStruct converts an InternedStruct back to a structured.Node.
func FromInternedStruct(s *pb.InternedStruct, pool *InternPool) (structured.Node, error) {
	if s == nil {
		return nil, fmt.Errorf("InternedStruct is nil")
	}
	if s.FieldPathSetId == nil {
		return nil, fmt.Errorf("FieldPathSetId is nil")
	}

	fieldSetRef := &FieldPathSetRef{pool: pool, id: *s.FieldPathSetId}
	keys := fieldSetRef.Resolve()

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
func FromInternedValue(v *pb.InternedValue, pool *InternPool) (structured.Node, error) {
	if v == nil {
		return nil, fmt.Errorf("InternedValue is nil")
	}
	switch kind := v.Kind.(type) {
	case *pb.InternedValue_NullValue:
		return structured.NewStandardScalarNode[any](nil), nil
	case *pb.InternedValue_BoolValue:
		return structured.NewStandardScalarNode(kind.BoolValue), nil
	case *pb.InternedValue_StringValue:
		return structured.NewStandardScalarNode(pool.resolveStringFromID(kind.StringValue)), nil
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
	case *pb.InternedValue_StructValue:
		return FromInternedStruct(kind.StructValue, pool)
	default:
		return nil, fmt.Errorf("unknown InternedValue kind: %T", kind)
	}
}
