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
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// Subset returns a new TaskSet filtered using the provided predicate function.
func Subset(taskSet *TaskSet, predicate func(UntypedTask) bool) *TaskSet {
	filteredTasks := typedmap.Filter(taskSet.GetAll(), predicate)
	result, _ := NewTaskSet(filteredTasks)
	return result
}

// WhereLabelEquals returns a predicate function that checks if a task's label
// matches the expected value.
func WhereLabelEquals[T comparable](labelKey TaskLabelKey[T], value T) func(UntypedTask) bool {
	getMap := func(d UntypedTask) *typedmap.ReadonlyTypedMap {
		return d.Labels()
	}
	return typedmap.WhereFieldEquals(getMap, labelKey, value)
}

// WhereLabelContainsElement returns a predicate function that checks if a task's label
// contains the expected element.
func WhereLabelContainsElement(labelKey TaskLabelKey[[]string], element string) func(UntypedTask) bool {
	getMap := func(d UntypedTask) *typedmap.ReadonlyTypedMap {
		return d.Labels()
	}
	return typedmap.WhereFieldContainsElement(getMap, labelKey, element)
}

// WhereLabelIsEnabled returns a predicate function that checks if a task's label
// is enabled.
func WhereLabelIsEnabled(labelKey TaskLabelKey[bool]) func(UntypedTask) bool {
	getMap := func(d UntypedTask) *typedmap.ReadonlyTypedMap {
		return d.Labels()
	}
	return typedmap.WhereFieldIsEnabled(getMap, labelKey)
}
