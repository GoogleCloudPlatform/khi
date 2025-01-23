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

package queryutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestTimeRangeQuerySection(t *testing.T) {
	testCases := []struct {
		StartTime      time.Time
		EndTime        time.Time
		IncludeEnd     bool
		ExpectedResult string
	}{
		{
			StartTime:  time.Date(2023, 3, 1, 12, 0, 0, 0, time.UTC),
			EndTime:    time.Date(2023, 3, 1, 14, 0, 0, 0, time.UTC),
			IncludeEnd: false,
			ExpectedResult: `timestamp >= "2023-03-01T12:00:00+0000"
timestamp < "2023-03-01T14:00:00+0000"`,
		},
		{
			StartTime:  time.Date(2023, 3, 1, 12, 0, 0, 0, time.UTC),
			EndTime:    time.Date(2023, 3, 1, 14, 0, 0, 0, time.UTC),
			IncludeEnd: true,
			ExpectedResult: `timestamp >= "2023-03-01T12:00:00+0000"
timestamp <= "2023-03-01T14:00:00+0000"`,
		},
		{
			StartTime:  time.Date(2023, 3, 1, 12, 0, 0, 0, time.FixedZone("Asia/Tokyo", 9*60*60)),
			EndTime:    time.Date(2023, 3, 1, 14, 0, 0, 0, time.FixedZone("Asia/Tokyo", 9*60*60)),
			IncludeEnd: true,
			ExpectedResult: `timestamp >= "2023-03-01T12:00:00+0900"
timestamp <= "2023-03-01T14:00:00+0900"`,
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%d-include-end-%v", i, testCase.IncludeEnd), func(t *testing.T) {
			timeRange := TimeRangeQuerySection(testCase.StartTime, testCase.EndTime, testCase.IncludeEnd)
			if timeRange != testCase.ExpectedResult {
				t.Errorf("result is not matching with the expected status\nexpected:\n%s\nactual:\n%s", testCase.ExpectedResult, timeRange)
			}
		})
	}
}

func TestWrapDoubleQuoteForStringArray(t *testing.T) {
	testCases := []struct {
		Input    []string
		Expected []string
	}{
		{
			Input:    []string{"foo", "bar"},
			Expected: []string{`"foo"`, `"bar"`},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Input[0], func(t *testing.T) {
			actual := WrapDoubleQuoteForStringArray(testCase.Input)
			if diff := cmp.Diff(testCase.Expected, actual); diff != "" {
				t.Errorf("The generated result is not matching with the expected\n%s", diff)
			}
		})
	}
}

func TestSplitToChildGroups(t *testing.T) {
	testCases := []struct {
		Name          string
		Input         []string
		InputMaxCount int
		Expected      [][]string
	}{
		{
			Name:          "empty",
			Input:         []string{},
			InputMaxCount: 10,
			Expected:      [][]string{},
		},
		{
			Name:          "a result",
			Input:         []string{"foo", "bar", "qux"},
			InputMaxCount: 10,
			Expected: [][]string{
				{"foo", "bar", "qux"},
			},
		},
		{
			Name:          "multi result",
			Input:         []string{"foo", "bar", "qux"},
			InputMaxCount: 2,
			Expected: [][]string{
				{"foo", "bar"},
				{"qux"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			actual := SplitToChildGroups(testCase.Input, testCase.InputMaxCount)
			if diff := cmp.Diff(testCase.Expected, actual); diff != "" {
				t.Errorf("The generated result is not matching with the expected\n%s", diff)
			}
		})
	}
}
