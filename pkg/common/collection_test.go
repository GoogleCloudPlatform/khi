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

package common // Or the package your function belongs to

import (
	"testing"

	"sort" // Import the sort package

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestDedupStringArray(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"Empty array", []string{}, []string{}},
		{"Unique elements (sorted)", []string{"hello", "world"}, []string{"hello", "world"}},
		{"Duplicates (sorted)", []string{"apple", "banana", "apple", "orange"}, []string{"apple", "banana", "orange"}},
		{"Duplicates (not sorted)", []string{"orange", "apple", "banana", "apple", "orange"}, []string{"apple", "banana", "orange"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DedupStringArray(tc.input)

			// Sort expected output for consistent comparison
			sort.Strings(tc.expected)

			if diff := cmp.Diff(result, tc.expected); diff != "" {
				t.Errorf("DedupeStringArray failed. Difference:\n%s", diff)
			}
		})
	}
}

func TestSortByStringDistance(t *testing.T) {
	tests := []struct {
		query    string
		strings  []string
		expected []string
	}{
		{
			query: "foo",
			strings: []string{
				"foooo",
				"foo",
				"fooo",
				"fo",
			},
			expected: []string{
				"foo",
				"fooo",
				"foooo",
				"fo",
			},
		},
		{
			query: "foo",
			strings: []string{
				"foooo",
				"fooo",
				"fo",
				"foo",
			},
			expected: []string{
				"foo",
				"fooo",
				"foooo",
				"fo",
			},
		},
		{
			query: "foo",
			strings: []string{
				"foooo",
				"fooo",
				"f",
				"foo",
				"fob",
				"boo",
			},
			expected: []string{
				"foo",
				"fooo",
				"foooo",
				"fob",
				"boo",
				"f",
			},
		},
	}

	for _, tt := range tests {
		actual := SortForAutocomplete(tt.query, tt.strings)
		if diff := cmp.Diff(actual, tt.expected); diff != "" {
			t.Errorf("%v", actual)
			t.Errorf("TestSortByStringDistance failed. Difference:\n%s", diff)
		}
	}
}
