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

package grouper

// BasicGrouper implements Grouper
type BasicGrouper[T any, K comparable] struct {
	GroupingFunc func(input T) K
}

func (b *BasicGrouper[T, K]) Group(input []T) map[K][]T {
	result := map[K][]T{}
	for _, v := range input {
		key := b.GroupingFunc(v)
		if _, found := result[key]; !found {
			result[key] = []T{}
		}
		result[key] = append(result[key], v)
	}
	return result
}

func NewBasicGrouper[T any, K comparable](groupingFunction func(input T) K) *BasicGrouper[T, K] {
	return &BasicGrouper[T, K]{
		GroupingFunc: groupingFunction,
	}
}
