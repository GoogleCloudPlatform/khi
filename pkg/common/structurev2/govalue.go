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
	"slices"
	"unique"

	"golang.org/x/exp/maps"
)

var defaultFieldPathCapacity = 8

// GoMapKeyOrderProvider decides the order of keys from the given map.
type GoMapKeyOrderProvider interface {
	// GetOrderedKeys returns the keys of map in the order to store keys. This interface is necessary because the order of map keys are not stable in Go.
	GetOrderedKeys(fieldPath []string, mapToDecideOrder map[string]any) ([]string, error)
}

// AlphabeticalGoMapKeyOrderProvider implements GoMapKeyOrderProvider with sorting map keys by alphabetical order.
type AlphabeticalGoMapKeyOrderProvider struct {
}

func (a *AlphabeticalGoMapKeyOrderProvider) GetOrderedKeys(sourcePath []string, sourceMap map[string]any) ([]string, error) {
	keys := maps.Keys(sourceMap)
	slices.Sort(keys)
	return keys, nil
}

// FromGoValue instanciate the Node interface from given Go map, slice or scalars.
func FromGoValue(source any, mapKeyOrderProvider GoMapKeyOrderProvider) (Node, error) {
	return fromGoValue(make([]string, 0, defaultFieldPathCapacity), source, mapKeyOrderProvider)
}

func fromGoValue(path []string, source any, mapKeyOrderProvider GoMapKeyOrderProvider) (Node, error) {
	switch v := source.(type) {
	case map[string]any:
		return fromGoMap(path, v, mapKeyOrderProvider)
	case []any:
		return fromGoSlice(path, v, mapKeyOrderProvider)
	default:
		return fromGoScalar(v)
	}
}

func fromGoMap(path []string, source map[string]any, mapKeyOrderProvider GoMapKeyOrderProvider) (Node, error) {
	keys, err := mapKeyOrderProvider.GetOrderedKeys(path, source)
	if err != nil {
		return nil, err
	}
	internedKeys := make([]unique.Handle[string], 0, len(keys))
	for _, key := range keys {
		internedKeys = append(internedKeys, unique.Make(key))
	}
	children := make([]Node, 0, len(keys))
	for _, key := range keys {
		path = append(path, key)
		child, err := fromGoValue(path, source[key], mapKeyOrderProvider)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
		path = path[:len(path)-1]
	}
	return &StandardMapNode{keys: internedKeys, values: children}, nil

}

func fromGoSlice(path []string, source []any, mapKeyOrderProvider GoMapKeyOrderProvider) (Node, error) {
	children := make([]Node, 0, len(source))
	path = append(path, "[]")
	for _, v := range source {
		child, err := fromGoValue(path, v, mapKeyOrderProvider)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	return &StandardSequenceNode{value: children}, nil
}

func fromGoScalar(source any) (Node, error) {
	return NewStandardScalarNode(source), nil
}
