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

package task

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

type VariableSet struct {
	variables *typedmap.TypedMap
}

func NewVariableSet(initialVariables map[string]any) *VariableSet {
	vs := &VariableSet{
		variables: &typedmap.TypedMap{},
	}
	for variableKey, data := range initialVariables {
		vs.Set(variableKey, data)
	}
	return vs
}

// TODO: define a new type safe function
func (s *VariableSet) Set(key string, value any) error {
	typedmap.Set(s.variables, typedmap.NewTypedKey[any](key), value)
	return nil
}

func (s *VariableSet) DeleteItems(selector func(key string) bool) {
	keys := map[string]struct{}{}
	for _, key := range s.variables.Keys() {
		if selector(key) {
			keys[key] = struct{}{}
		}
	}
	for k := range keys {
		typedmap.Delete(s.variables, typedmap.NewTypedKey[any](k))
	}
}

// TODO: define a new type safe function
// GetTypedVariableFromTaskVariable returns the specified variable from given variable set with type cast.
func GetTypedVariableFromTaskVariable[T any](tv *VariableSet, variableId string, defaultValue T) (T, error) {
	value, found := typedmap.Get(tv.variables, typedmap.NewTypedKey[T](variableId))
	if !found {
		return value, nil
	}
	return value, nil
}
