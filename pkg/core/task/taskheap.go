// Copyright 2026 Google LLC
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

package coretask

import (
	"container/heap"
)

// taskMinHeap implements heap.Interface for UntypedTask, sorted deterministically by task implementation ID string.
type taskMinHeap []UntypedTask

var _ heap.Interface = (*taskMinHeap)(nil)

// Len returns the number of tasks in the heap.
func (h taskMinHeap) Len() int { return len(h) }

// Less reports whether the task with index i must sort before the task with index j.
func (h taskMinHeap) Less(i, j int) bool {
	return h[i].UntypedID().String() < h[j].UntypedID().String()
}

// Swap swaps the elements with indexes i and j.
func (h taskMinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

// Push appends an UntypedTask element to the heap.
func (h *taskMinHeap) Push(x any) { *h = append(*h, x.(UntypedTask)) }

// Pop removes and returns the last element in the heap slice.
func (h *taskMinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*h = old[0 : n-1]
	return x
}
