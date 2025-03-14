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

package filter

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// TypedMapFilter is an interface for filtering objects that have a TypedMap
// Type parameter T represents the type of the value to filter on
type TypedMapFilter[T any] interface {
	// KeyToFilter returns the key to evaluate
	KeyToFilter() typedmap.TypedKey[T]

	// ShouldInclude determines if an item should be included in filtered results
	// found: whether the key exists on the item
	// value: the value if it exists (zero value if found=false)
	ShouldInclude(found bool, value T) bool
}

// FilterTypedMapCollection filters a collection of items using a type-safe filter
// TItem: the type of items being filtered
// TValue: the type of value being filtered on
func FilterTypedMapCollection[TItem any, TValue any](
	items []TItem,
	getMap func(TItem) *typedmap.ReadonlyTypedMap,
	filter TypedMapFilter[TValue],
) []TItem {
	var result []TItem

	for _, item := range items {
		typedMap := getMap(item)
		key := filter.KeyToFilter()
		value, exist := typedmap.Get(typedMap, key)

		if filter.ShouldInclude(exist, value) {
			result = append(result, item)
		}
	}

	return result
}

// EqualFilter matches items with values exactly equal to the specified value
type EqualFilter[T comparable] struct {
	key              typedmap.TypedKey[T]
	expectedValue    T
	includeUndefined bool
}

// KeyToFilter implements TypedMapFilter interface
func (f *EqualFilter[T]) KeyToFilter() typedmap.TypedKey[T] {
	return f.key
}

// ShouldInclude implements TypedMapFilter interface
func (f *EqualFilter[T]) ShouldInclude(found bool, value T) bool {
	if found {
		return value == f.expectedValue
	}
	return f.includeUndefined
}

// NewEqualFilter creates a new filter that matches exact values
func NewEqualFilter[T comparable](key typedmap.TypedKey[T], value T, includeUndefined bool) TypedMapFilter[T] {
	return &EqualFilter[T]{
		key:              key,
		expectedValue:    value,
		includeUndefined: includeUndefined,
	}
}

// ContainsElementFilter checks if a string array contains a specific element
type ContainsElementFilter struct {
	key              typedmap.TypedKey[[]string]
	element          string
	includeUndefined bool
}

// KeyToFilter implements TypedMapFilter interface
func (f *ContainsElementFilter) KeyToFilter() typedmap.TypedKey[[]string] {
	return f.key
}

// ShouldInclude implements TypedMapFilter interface
func (f *ContainsElementFilter) ShouldInclude(found bool, values []string) bool {
	if !found {
		return f.includeUndefined
	}

	for _, v := range values {
		if v == f.element {
			return true
		}
	}
	return false
}

// NewContainsElementFilter creates a filter that checks if an array contains an element
func NewContainsElementFilter(key typedmap.TypedKey[[]string], element string, includeUndefined bool) TypedMapFilter[[]string] {
	return &ContainsElementFilter{
		key:              key,
		element:          element,
		includeUndefined: includeUndefined,
	}
}

// EnabledFilter checks if a boolean value is true
type EnabledFilter struct {
	key              typedmap.TypedKey[bool]
	includeUndefined bool
}

// KeyToFilter implements TypedMapFilter interface
func (f *EnabledFilter) KeyToFilter() typedmap.TypedKey[bool] {
	return f.key
}

// ShouldInclude implements TypedMapFilter interface
func (f *EnabledFilter) ShouldInclude(found bool, value bool) bool {
	if !found {
		return f.includeUndefined
	}
	return value
}

// NewEnabledFilter creates a filter that checks if a boolean value is true
func NewEnabledFilter(key typedmap.TypedKey[bool], includeUndefined bool) TypedMapFilter[bool] {
	return &EnabledFilter{
		key:              key,
		includeUndefined: includeUndefined,
	}
}
