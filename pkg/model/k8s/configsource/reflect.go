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

package configsource

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/merger"
)

const MAXIMUM_STRUCTURE_DEPTH = 100

func FromResourceTypeReflection(resourceType interface{}) (*merger.MergeConfigResolver, error) {
	result := &merger.MergeConfigResolver{
		MergeStrategies: make(map[string]merger.MergeArrayStrategy),
		MergeKeys:       map[string]string{},
	}
	refType := reflect.TypeOf(resourceType)
	err := resolveTypeRecursive("", refType, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func resolveTypeRecursive(path string, current reflect.Type, resolver *merger.MergeConfigResolver) error {
	if strings.Count(path, ".") > MAXIMUM_STRUCTURE_DEPTH {
		return fmt.Errorf("maximum structure depth reached. is this a recursive structure?")
	}
	kind := current.Kind()
	switch kind {
	case reflect.Struct:
		fieldCount := current.NumField()
		for i := 0; i < fieldCount; i++ {
			field := current.Field(i)
			fieldKind := field.Type.Kind()
			json, ok := field.Tag.Lookup("json")
			if !ok {
				continue
			}
			jsonSegments := strings.Split(json, ",")
			if slices.Contains[[]string](jsonSegments, "inline") {
				err := resolveTypeRecursive(path, field.Type, resolver)
				if err != nil {
					return err
				}
			}
			jsonName := jsonSegments[0]
			jsonFieldPath := fmt.Sprintf("%s.%s", path, jsonName)
			if fieldKind == reflect.Slice || fieldKind == reflect.Array {
				patchStrategy, ok := field.Tag.Lookup("patchStrategy")
				if !ok || patchStrategy != "merge" {
					resolver.MergeStrategies[jsonFieldPath] = merger.MergeStrategyReplace
				} else {
					resolver.MergeStrategies[jsonFieldPath] = merger.MergeStrategyMerge
					patchMergeKey, ok := field.Tag.Lookup("patchMergeKey")
					if ok {
						resolver.MergeKeys[jsonFieldPath] = patchMergeKey
					} else {
						resolver.MergeKeys[jsonFieldPath] = ""
					}
				}
				err := resolveTypeRecursive(jsonFieldPath+"[]", field.Type.Elem(), resolver)
				if err != nil {
					return err
				}
			} else if fieldKind == reflect.Struct || fieldKind == reflect.Ptr {
				fieldType := field.Type
				if fieldKind == reflect.Ptr {
					fieldType = field.Type.Elem()
				}
				err := resolveTypeRecursive(jsonFieldPath, fieldType, resolver)
				if err != nil {
					return err
				}
			}
		}
		return nil
	case reflect.Chan:
		return fmt.Errorf("unsupported kind %s", current.Kind())
	case reflect.Ptr:
		return fmt.Errorf("unsupported kind %s", current.Kind())
	default:
		return nil
	}
}
