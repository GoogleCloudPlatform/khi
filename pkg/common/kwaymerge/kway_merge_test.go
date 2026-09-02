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
	"cmp"
	"strings"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
)

type itemWithSource struct {
	key    int
	source string
}

func TestMerge(t *testing.T) {
	t.Parallel()

	t.Run("integers", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name   string
			input  [][]int
			want   []int
			cmpFun func(a, b int) int
		}{
			{
				name:   "empty input slices",
				input:  [][]int{},
				want:   []int{},
				cmpFun: cmp.Compare[int],
			},
			{
				name:   "all inner slices empty",
				input:  [][]int{{}, {}, {}},
				want:   []int{},
				cmpFun: cmp.Compare[int],
			},
			{
				name:   "single slice",
				input:  [][]int{{1, 3, 5}},
				want:   []int{1, 3, 5},
				cmpFun: cmp.Compare[int],
			},
			{
				name:   "single non-empty with empty slices",
				input:  [][]int{{}, {2, 4, 6}, {}},
				want:   []int{2, 4, 6},
				cmpFun: cmp.Compare[int],
			},
			{
				name:   "disjoint sorted slices",
				input:  [][]int{{1, 2, 3}, {4, 5, 6}},
				want:   []int{1, 2, 3, 4, 5, 6},
				cmpFun: cmp.Compare[int],
			},
			{
				name:   "interleaved sorted slices",
				input:  [][]int{{1, 4, 7}, {2, 5, 8}, {3, 6, 9}},
				want:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
				cmpFun: cmp.Compare[int],
			},
			{
				name:   "different length slices with empty slices",
				input:  [][]int{{}, {1, 10}, {2, 3, 4, 5, 6}, {}, {7}, {8, 9}},
				want:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				cmpFun: cmp.Compare[int],
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got := Merge(tc.input, tc.cmpFun)
				if diff := gocmp.Diff(tc.want, got); diff != "" {
					t.Errorf("Merge() mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("stability with duplicate keys", func(t *testing.T) {
		t.Parallel()
		input := [][]itemWithSource{
			{
				{key: 1, source: "s0_first"},
				{key: 2, source: "s0_item"},
				{key: 3, source: "s0_last"},
			},
			{
				{key: 1, source: "s1_first"},
				{key: 2, source: "s1_item"},
			},
			{
				{key: 2, source: "s2_item"},
				{key: 3, source: "s2_last"},
			},
		}
		want := []itemWithSource{
			{key: 1, source: "s0_first"},
			{key: 1, source: "s1_first"},
			{key: 2, source: "s0_item"},
			{key: 2, source: "s1_item"},
			{key: 2, source: "s2_item"},
			{key: 3, source: "s0_last"},
			{key: 3, source: "s2_last"},
		}

		got := Merge(input, func(a, b itemWithSource) int {
			return cmp.Compare(a.key, b.key)
		})

		if diff := gocmp.Diff(want, got, gocmp.AllowUnexported(itemWithSource{})); diff != "" {
			t.Errorf("Merge() stability mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("strings", func(t *testing.T) {
		t.Parallel()
		input := [][]string{
			{"apple", "cherry"},
			{"banana", "date"},
			{"elderberry"},
		}
		want := []string{"apple", "banana", "cherry", "date", "elderberry"}

		got := Merge(input, strings.Compare)
		if diff := gocmp.Diff(want, got); diff != "" {
			t.Errorf("Merge() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("timestamps", func(t *testing.T) {
		t.Parallel()
		base := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
		input := [][]time.Time{
			{base.Add(1 * time.Minute), base.Add(4 * time.Minute)},
			{base.Add(2 * time.Minute), base.Add(5 * time.Minute)},
			{base.Add(3 * time.Minute)},
		}
		want := []time.Time{
			base.Add(1 * time.Minute),
			base.Add(2 * time.Minute),
			base.Add(3 * time.Minute),
			base.Add(4 * time.Minute),
			base.Add(5 * time.Minute),
		}

		got := Merge(input, func(a, b time.Time) int {
			return a.Compare(b)
		})
		if diff := gocmp.Diff(want, got); diff != "" {
			t.Errorf("Merge() mismatch (-want +got):\n%s", diff)
		}
	})
}
