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

package kwaymerge

import (
	"container/heap"
)

// mergeCursor tracks the current position of an element in a specific slice.
type mergeCursor[T any] struct {
	sliceIndex int
	slice      []T
	elemIndex  int
}

// mergeHeap implements heap.Interface to maintain the minimum element across active cursors.
type mergeHeap[T any] struct {
	cursors []*mergeCursor[T]
	cmp     func(a, b T) int
}

var _ heap.Interface = (*mergeHeap[int])(nil)

// Len returns the number of active cursors in the heap.
func (h *mergeHeap[T]) Len() int {
	return len(h.cursors)
}

// Less reports whether the element at cursor i should sort before the element at cursor j.
// It breaks ties deterministically using sliceIndex to preserve stable ordering.
func (h *mergeHeap[T]) Less(i, j int) bool {
	c := h.cmp(h.cursors[i].slice[h.cursors[i].elemIndex], h.cursors[j].slice[h.cursors[j].elemIndex])
	if c != 0 {
		return c < 0
	}
	return h.cursors[i].sliceIndex < h.cursors[j].sliceIndex
}

// Swap swaps the cursors at indexes i and j.
func (h *mergeHeap[T]) Swap(i, j int) {
	h.cursors[i], h.cursors[j] = h.cursors[j], h.cursors[i]
}

// Push adds a cursor to the heap.
func (h *mergeHeap[T]) Push(x any) {
	h.cursors = append(h.cursors, x.(*mergeCursor[T]))
}

// Pop removes and returns the minimum cursor from the heap.
func (h *mergeHeap[T]) Pop() any {
	old := h.cursors
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	h.cursors = old[0 : n-1]
	return x
}

// Merge merges multiple pre-sorted slices into a single sorted slice based on cmp.
// Each input slice must already be sorted in ascending order according to cmp.
// The comparison function cmp should return a negative integer if a < b, zero if a == b, and a positive integer if a > b.
// When elements have equal keys (cmp returns 0), tie-breaking preserves the order of the slice in which they appeared.
func Merge[T any](slices [][]T, cmp func(a, b T) int) []T {
	totalCount := 0
	nonEmptyCount := 0
	var singleNonEmptySlice []T
	for _, s := range slices {
		l := len(s)
		if l > 0 {
			totalCount += l
			nonEmptyCount++
			singleNonEmptySlice = s
		}
	}
	if nonEmptyCount == 0 {
		return []T{}
	}
	if nonEmptyCount == 1 {
		result := make([]T, len(singleNonEmptySlice))
		copy(result, singleNonEmptySlice)
		return result
	}

	result := make([]T, 0, totalCount)
	h := &mergeHeap[T]{
		cursors: make([]*mergeCursor[T], 0, nonEmptyCount),
		cmp:     cmp,
	}

	for i, s := range slices {
		if len(s) > 0 {
			h.cursors = append(h.cursors, &mergeCursor[T]{
				sliceIndex: i,
				slice:      s,
				elemIndex:  0,
			})
		}
	}
	heap.Init(h)

	for len(h.cursors) > 0 {
		top := h.cursors[0]
		result = append(result, top.slice[top.elemIndex])
		top.elemIndex++
		if top.elemIndex < len(top.slice) {
			heap.Fix(h, 0)
		} else {
			heap.Pop(h)
		}
	}

	return result
}
