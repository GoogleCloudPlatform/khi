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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestTimeSeries_Set(t *testing.T) {
	t0 := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Second)
	t2 := t0.Add(2 * time.Second)
	t3 := t0.Add(3 * time.Second)

	t.Run("empty", func(t *testing.T) {
		ts := NewTimeSeries[string]()
		if len(ts.Entries) != 0 {
			t.Errorf("expected empty series, got %d entries", len(ts.Entries))
		}
	})

	t.Run("chronological", func(t *testing.T) {
		ts := NewTimeSeries[string]()
		ts.Set(t1, "A")
		ts.Set(t3, "B")

		wantEntries := []TimeSeriesEntry[string]{
			{T: t1, Val: "A"},
			{T: t3, Val: "B"},
		}
		if diff := cmp.Diff(wantEntries, ts.Entries); diff != "" {
			t.Errorf("Entries mismatch after chronological sets (-want +got):\n%s", diff)
		}
	})

	t.Run("out of order", func(t *testing.T) {
		ts := NewTimeSeries[string]()
		ts.Set(t1, "A")
		ts.Set(t3, "B")
		ts.Set(t2, "C")

		wantEntries := []TimeSeriesEntry[string]{
			{T: t1, Val: "A"},
			{T: t2, Val: "C"},
			{T: t3, Val: "B"},
		}
		if diff := cmp.Diff(wantEntries, ts.Entries); diff != "" {
			t.Errorf("Entries mismatch after out-of-order set (-want +got):\n%s", diff)
		}
	})

	t.Run("overwrite same timestamp", func(t *testing.T) {
		ts := NewTimeSeries[string]()
		ts.Set(t1, "A")
		ts.Set(t2, "B")
		ts.Set(t2, "C") // Overwrite

		wantEntries := []TimeSeriesEntry[string]{
			{T: t1, Val: "A"},
			{T: t2, Val: "C"},
		}
		if diff := cmp.Diff(wantEntries, ts.Entries); diff != "" {
			t.Errorf("Entries mismatch after overwrite (-want +got):\n%s", diff)
		}
	})
}

func TestTimeSeries_GetLastBeforeOrEqual(t *testing.T) {
	t0 := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Second)
	t2 := t0.Add(2 * time.Second)
	t3 := t0.Add(3 * time.Second)
	t4 := t0.Add(4 * time.Second)

	t.Run("empty", func(t *testing.T) {
		ts := NewTimeSeries[string]()
		if _, ok := ts.GetLastBeforeOrEqual(t0); ok {
			t.Errorf("GetLastBeforeOrEqual on empty series should return false")
		}
	})

	t.Run("lookups", func(t *testing.T) {
		ts := NewTimeSeries[string]()
		ts.Set(t1, "A")
		ts.Set(t2, "B")
		ts.Set(t3, "C")

		testCases := []struct {
			name      string
			queryTime time.Time
			wantVal   string
			wantOk    bool
		}{
			{name: "before first entry", queryTime: t0, wantVal: "", wantOk: false},
			{name: "exact match first", queryTime: t1, wantVal: "A", wantOk: true},
			{name: "between first and second", queryTime: t1.Add(500 * time.Millisecond), wantVal: "A", wantOk: true},
			{name: "exact match second", queryTime: t2, wantVal: "B", wantOk: true},
			{name: "after last entry", queryTime: t4, wantVal: "C", wantOk: true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotVal, gotOk := ts.GetLastBeforeOrEqual(tc.queryTime)
				if gotOk != tc.wantOk || gotVal != tc.wantVal {
					t.Errorf("GetLastBeforeOrEqual(%s) = (%q, %v), want (%q, %v)", tc.queryTime, gotVal, gotOk, tc.wantVal, tc.wantOk)
				}
			})
		}
	})
}

func TestTimeSeries_GetFirstAfterOrEqual(t *testing.T) {
	t0 := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Second)
	t2 := t0.Add(2 * time.Second)
	t3 := t0.Add(3 * time.Second)
	t4 := t0.Add(4 * time.Second)

	t.Run("empty", func(t *testing.T) {
		ts := NewTimeSeries[string]()
		if _, ok := ts.GetFirstAfterOrEqual(t0); ok {
			t.Errorf("GetFirstAfterOrEqual on empty series should return false")
		}
	})

	t.Run("lookups", func(t *testing.T) {
		ts := NewTimeSeries[string]()
		ts.Set(t1, "A")
		ts.Set(t2, "B")
		ts.Set(t3, "C")

		testCases := []struct {
			name      string
			queryTime time.Time
			wantVal   string
			wantOk    bool
		}{
			{name: "before first entry", queryTime: t0, wantVal: "A", wantOk: true},
			{name: "exact match first", queryTime: t1, wantVal: "A", wantOk: true},
			{name: "between first and second", queryTime: t1.Add(500 * time.Millisecond), wantVal: "B", wantOk: true},
			{name: "exact match second", queryTime: t2, wantVal: "B", wantOk: true},
			{name: "after last entry", queryTime: t4, wantVal: "", wantOk: false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotVal, gotOk := ts.GetFirstAfterOrEqual(tc.queryTime)
				if gotOk != tc.wantOk || gotVal != tc.wantVal {
					t.Errorf("GetFirstAfterOrEqual(%s) = (%q, %v), want (%q, %v)", tc.queryTime, gotVal, gotOk, tc.wantVal, tc.wantOk)
				}
			})
		}
	})
}

func TestTimeSeries_Get(t *testing.T) {
	t0 := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Second)
	t2 := t0.Add(2 * time.Second)
	t3 := t0.Add(3 * time.Second)
	t4 := t0.Add(4 * time.Second)

	t.Run("empty", func(t *testing.T) {
		ts := NewTimeSeries[string]()
		if _, ok := ts.Get(t0); ok {
			t.Errorf("Get on empty series should return false")
		}
	})

	t.Run("lookups with fallback", func(t *testing.T) {
		ts := NewTimeSeries[string]()
		ts.Set(t2, "A")
		ts.Set(t3, "B")

		testCases := []struct {
			name      string
			queryTime time.Time
			wantVal   string
			wantOk    bool
		}{
			{name: "fallback to earliest (t0 < t2)", queryTime: t0, wantVal: "A", wantOk: true},
			{name: "fallback to earliest (t1 < t2)", queryTime: t1, wantVal: "A", wantOk: true},
			{name: "exact match", queryTime: t2, wantVal: "A", wantOk: true},
			{name: "active before t3", queryTime: t2.Add(500 * time.Millisecond), wantVal: "A", wantOk: true},
			{name: "exact match B", queryTime: t3, wantVal: "B", wantOk: true},
			{name: "active after t3", queryTime: t4, wantVal: "B", wantOk: true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotVal, gotOk := ts.Get(tc.queryTime)
				if gotOk != tc.wantOk || gotVal != tc.wantVal {
					t.Errorf("Get(%s) = (%q, %v), want (%q, %v)", tc.queryTime, gotVal, gotOk, tc.wantVal, tc.wantOk)
				}
			})
		}
	})
}

func TestTimeSeries_AlternatingValues(t *testing.T) {
	t0 := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Second)
	t2 := t0.Add(2 * time.Second)
	t3 := t0.Add(3 * time.Second)

	ts := NewTimeSeries[string]()
	ts.Set(t1, "A")
	ts.Set(t2, "B")
	ts.Set(t3, "A") // Alternates back to A

	wantEntries := []TimeSeriesEntry[string]{
		{T: t1, Val: "A"},
		{T: t2, Val: "B"},
		{T: t3, Val: "A"},
	}
	if diff := cmp.Diff(wantEntries, ts.Entries); diff != "" {
		t.Errorf("Entries mismatch for alternating values (-want +got):\n%s", diff)
	}

	// Verify alternating lookups are correct
	t.Run("query t1.5", func(t *testing.T) {
		val, _ := ts.Get(t1.Add(500 * time.Millisecond))
		if val != "A" {
			t.Errorf("expected A, got %q", val)
		}
	})
	t.Run("query t2.5", func(t *testing.T) {
		val, _ := ts.Get(t2.Add(500 * time.Millisecond))
		if val != "B" {
			t.Errorf("expected B, got %q", val)
		}
	})
	t.Run("query t3.5", func(t *testing.T) {
		val, _ := ts.Get(t3.Add(500 * time.Millisecond))
		if val != "A" {
			t.Errorf("expected A, got %q", val)
		}
	})
}
