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

package typedmap

// Filter returns a new slice containing only the elements for which the predicate function returns true.
func Filter[T any](items []T, predicate func(item T) bool) []T {
	var result []T
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// WhereFieldEquals returns a predicate function that checks if the value of a specific key in a TypedMap equals the expected value.
func WhereFieldEquals[TItem any, TValue comparable](
	getMap func(TItem) *ReadonlyTypedMap,
	key TypedKey[TValue],
	expectedValue TValue,
	includeIfValueNotFound bool,
) func(TItem) bool {
	return func(item TItem) bool {
		val, ok := Get(getMap(item), key)
		if !ok {
			return includeIfValueNotFound
		}
		return ok && val == expectedValue
	}
}

// WhereFieldContainsElement returns a predicate function that checks if a string slice in a TypedMap contains a specific element.
func WhereFieldContainsElement[TItem any](
	getMap func(TItem) *ReadonlyTypedMap,
	key TypedKey[[]string],
	element string,
	includeIfValueNotFound bool,
) func(TItem) bool {
	return func(item TItem) bool {
		values, ok := Get(getMap(item), key)
		if !ok {
			return includeIfValueNotFound
		}
		for _, v := range values {
			if v == element {
				return true
			}
		}
		return false
	}
}

// WhereFieldIsEnabled returns a predicate function that checks if a boolean flag in a TypedMap is enabled (true).
func WhereFieldIsEnabled[TItem any](
	getMap func(TItem) *ReadonlyTypedMap,
	key TypedKey[bool],
	includeIfValueNotFound bool,
) func(TItem) bool {
	return func(item TItem) bool {
		enabled, ok := Get(getMap(item), key)
		if !ok {
			return includeIfValueNotFound
		}
		return ok && enabled
	}
}
