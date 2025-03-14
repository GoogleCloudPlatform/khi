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

package task

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/filter"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// Subset returns a new DefinitionSet filtered using the provided type-safe filter
func Subset[T any](definitionSet *DefinitionSet, mapFilter filter.TypedMapFilter[T]) *DefinitionSet {
	getMap := func(d Definition) *typedmap.ReadonlyTypedMap {
		return d.Labels()
	}

	filteredTasks := filter.FilterTypedMapCollection(definitionSet.GetAll(), getMap, mapFilter)
	result, _ := NewSet(filteredTasks)
	return result
}

// NewEqualFilter creates a new filter that matches exact label values
func NewEqualFilter[T comparable](labelKey TaskLabelKey[T], value T, includeUndefined bool) filter.TypedMapFilter[T] {
	return filter.NewEqualFilter(labelKey, value, includeUndefined)
}
