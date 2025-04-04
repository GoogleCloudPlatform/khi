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

package taskfilter

// ContainsElement returns a function that represents a condition to filter only tasks that have the specified element in the specified label value.
func ContainsElement[T comparable](comparedWith T) func(taskLabelValueAny any) bool {
	return func(v any) bool {
		taskLabelValue := v.([]T)
		for _, element := range taskLabelValue {
			if element == comparedWith {
				return true
			}
		}
		return false
	}
}

// HasTrue is a function that represents a condition to filter only tasks with true value.
func HasTrue(taskLabelValueAny any) bool {
	return taskLabelValueAny.(bool)
}
