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

package common

import (
	"slices"
	"time"
)

// TimeSeriesEntry represents a value associated with a specific timestamp.
type TimeSeriesEntry[V comparable] struct {
	T   time.Time
	Val V
}

// TimeSeries tracks the value of a single variable over time.
// It is useful for cases where a value can change at different points in time.
type TimeSeries[V comparable] struct {
	Entries []TimeSeriesEntry[V]
}

// NewTimeSeries creates a new TimeSeries.
func NewTimeSeries[V comparable]() *TimeSeries[V] {
	return &TimeSeries[V]{}
}

// Set records that the value is associated starting from time t.
// It preserves chronological order using binary search insertion.
func (s *TimeSeries[V]) Set(t time.Time, val V) {
	if len(s.Entries) == 0 {
		s.Entries = []TimeSeriesEntry[V]{{T: t, Val: val}}
		return
	}
	lastIdx := len(s.Entries) - 1
	// Optimization: if entries are added chronologically (the most common case),
	// we can just append without binary searching or inserting.
	if t.After(s.Entries[lastIdx].T) {
		s.Entries = append(s.Entries, TimeSeriesEntry[V]{T: t, Val: val})
		return
	}
	if t.Equal(s.Entries[lastIdx].T) {
		s.Entries[lastIdx].Val = val
		return
	}

	idx, found := s.binarySearch(t)
	if found {
		s.Entries[idx].Val = val
		return
	}
	// Insert preserving sorted order
	s.Entries = slices.Insert(s.Entries, idx, TimeSeriesEntry[V]{T: t, Val: val})
}

// GetLastBeforeOrEqual returns the value of the last entry whose timestamp is <= t.
func (s *TimeSeries[V]) GetLastBeforeOrEqual(t time.Time) (V, bool) {
	if len(s.Entries) == 0 {
		var zero V
		return zero, false
	}
	idx, found := s.binarySearch(t)
	if found {
		return s.Entries[idx].Val, true
	}
	if idx > 0 {
		return s.Entries[idx-1].Val, true
	}
	var zero V
	return zero, false
}

// GetFirstAfterOrEqual returns the value of the first entry whose timestamp is >= t.
func (s *TimeSeries[V]) GetFirstAfterOrEqual(t time.Time) (V, bool) {
	if len(s.Entries) == 0 {
		var zero V
		return zero, false
	}
	idx, found := s.binarySearch(t)
	if found {
		return s.Entries[idx].Val, true
	}
	if idx < len(s.Entries) {
		return s.Entries[idx].Val, true
	}
	var zero V
	return zero, false
}

// Get returns the value associated at time t.
// It tries to find the value active at t (using GetLastBeforeOrEqual).
// If no such entry exists (t is before all entries), it falls back to the earliest value seen (GetFirstAfterOrEqual).
func (s *TimeSeries[V]) Get(t time.Time) (V, bool) {
	val, ok := s.GetLastBeforeOrEqual(t)
	if ok {
		return val, true
	}
	return s.GetFirstAfterOrEqual(t)
}

func (s *TimeSeries[V]) binarySearch(t time.Time) (int, bool) {
	return slices.BinarySearchFunc(s.Entries, t, func(e TimeSeriesEntry[V], target time.Time) int {
		if e.T.Before(target) {
			return -1
		}
		if e.T.After(target) {
			return 1
		}
		return 0
	})
}
