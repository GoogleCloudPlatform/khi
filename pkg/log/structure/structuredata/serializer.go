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

package structuredata

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

func ToYaml(s StructureData) (string, error) {
	serializable, err := toSerializableObject(s)
	if err != nil {
		return "", err
	}
	marshalled, err := yaml.Marshal(serializable)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(marshalled), fieldCommaEscape, ","), nil
}

func ToJson(s StructureData) (string, error) {
	serializable, err := toSerializableObject(s)
	if err != nil {
		return "", err
	}
	marshalled, err := json.Marshal(serializable)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(marshalled), fieldCommaEscape, ","), nil

}

func toSerializableObject(s StructureData) (any, error) {
	ty, err := s.Type()
	if err != nil {
		return "", err
	}
	var yamlSourceObject any
	switch ty {
	case StructuredTypeArray:
		value, err := s.Value("")
		if err != nil {
			return nil, err
		}
		yamlSourceObject = value
	case StructuredTypeMap:
		root := newSortedMap()
		keys, err := s.Keys()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			err = storeValuesInContainerRecursive(s, &mapValueContainer{target: root}, key)
			if err != nil {
				return nil, err
			}
		}
		yamlSourceObject = root
	case StructuredTypeScalar:
		value, err := s.Value("")
		if err != nil {
			return nil, err
		}
		yamlSourceObject = value
	default:
		return nil, fmt.Errorf("unsupported root object")
	}
	return yamlSourceObject, nil
}

func storeValuesInContainerRecursive(s StructureData, addTo valueContainer, key string) error {
	nr, err := s.Value(key)
	if err != nil {
		return err
	}
	nrStructure, convertible := nr.(StructureData)
	if !convertible {
		return fmt.Errorf("unreachable. value result is not a structuredata")
	}
	ty, err := nrStructure.Type()
	if err != nil {
		return err
	}
	if ty == StructuredTypeScalar {
		value, err := nrStructure.Value("")
		if err != nil {
			return err
		}
		return addTo.Store(key, value)
	}
	if ty == StructuredTypeMap {
		current := newSortedMap()
		keys, err := nrStructure.Keys()
		if err != nil {
			return err
		}
		for _, key := range keys {
			err = storeValuesInContainerRecursive(nrStructure, &mapValueContainer{target: current}, key)
			if err != nil {
				return err
			}
		}
		return addTo.Store(key, current)
	}
	if ty == StructuredTypeArray {
		keys, err := nrStructure.Keys()
		if err != nil {
			return err
		}
		current := make([]any, len(keys))
		for _, key := range keys {
			err = storeValuesInContainerRecursive(nrStructure, &arrayValueContainer{target: current}, key)
			if err != nil {
				return err
			}
		}
		return addTo.Store(key, current)
	}
	return fmt.Errorf("unsupported")
}

type valueContainer interface {
	Store(key string, value any) error
}

var _ valueContainer = (*mapValueContainer)(nil)
var _ valueContainer = (*arrayValueContainer)(nil)

type mapValueContainer struct {
	target *sortedMap
}

// Store implements valueContainer.
func (c *mapValueContainer) Store(key string, value any) error {
	c.target.AddNextField(key, value)
	return nil
}

type arrayValueContainer struct {
	target []any
}

// Store implements valueContainer.
func (c *arrayValueContainer) Store(key string, value any) error {
	index, err := strconv.Atoi(key)
	if err != nil {
		return err
	}
	c.target[index] = value
	return nil
}
