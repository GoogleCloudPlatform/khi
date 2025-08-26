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

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBasicGrouper(t *testing.T) {
	type inputStruct struct {
		Key   string
		Value int
	}
	testCases := []struct {
		name     string
		input    []inputStruct
		expected map[string][]inputStruct
	}{
		{
			name:     "empty input",
			input:    []inputStruct{},
			expected: map[string][]inputStruct{},
		},
		{
			name: "single group",
			input: []inputStruct{
				{
					Key:   "groupA",
					Value: 1,
				},
				{
					Key:   "groupA",
					Value: 2,
				},
			},
			expected: map[string][]inputStruct{
				"groupA": {
					{
						Key:   "groupA",
						Value: 1,
					},
					{
						Key:   "groupA",
						Value: 2,
					},
				},
			},
		},
		{
			name: "multiple groups",
			input: []inputStruct{
				{
					Key:   "groupA",
					Value: 1,
				},
				{
					Key:   "groupB",
					Value: 2,
				},
				{
					Key:   "groupA",
					Value: 3,
				},
			},
			expected: map[string][]inputStruct{
				"groupA": {
					{
						Key:   "groupA",
						Value: 1,
					},
					{
						Key:   "groupA",
						Value: 3,
					},
				},
				"groupB": {
					{
						Key:   "groupB",
						Value: 2,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			grouper := NewBasicGrouper(func(input inputStruct) string {
				return input.Key
			})
			actual := grouper.Group(tc.input)
			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Errorf("Group() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
