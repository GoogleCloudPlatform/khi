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

import "fmt"

func EqualStructureData(a StructureData, b StructureData) (bool, error) {
	aType, err := a.Type()
	if err != nil {
		return false, err
	}
	bType, err := b.Type()
	if err != nil {
		return false, err
	}
	if aType != bType {
		return false, nil
	}
	aKeys, err := a.Keys()
	if err != nil {
		return false, err
	}
	bKeys, err := b.Keys()
	if err != nil {
		return false, err
	}
	if len(aKeys) != len(bKeys) {
		return false, nil
	}
	aKeysInStruct := map[string]struct{}{}
	for _, aKey := range aKeys {
		aKeysInStruct[aKey] = struct{}{}
	}
	for _, bKey := range bKeys {
		if _, exist := aKeysInStruct[bKey]; !exist {
			return false, nil
		}
	}
	if aType == StructuredTypeScalar {
		aValue, err := a.Value("")
		if err != nil {
			return false, err
		}
		bValue, err := b.Value("")
		if err != nil {
			return false, err
		}
		return aValue == bValue, nil
	} else {
		for _, aKey := range aKeys {
			aValue, err := a.Value(aKey)
			if err != nil {
				return false, err
			}
			if aValueStructured, convertible := aValue.(StructureData); !convertible {
				return false, fmt.Errorf("failed to convert result to StructureData")
			} else {
				bValue, err := b.Value(aKey)
				if err != nil {
					return false, err
				}
				if bValueStructured, convertible := bValue.(StructureData); !convertible {
					return false, fmt.Errorf("failed to convert result to StructureData")
				} else {
					eq, err := EqualStructureData(aValueStructured, bValueStructured)
					if err != nil {
						return false, err
					}
					if !eq {
						return false, nil
					}
				}
			}
		}
		return true, nil
	}

}
