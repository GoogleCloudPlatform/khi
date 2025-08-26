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

package common

import (
	"fmt"
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	JST := time.FixedZone("JST", 9*60*60)
	testCases := []struct {
		Input    string
		Expected time.Time
		Error    bool
	}{
		{
			Input:    "2023-01-02T03:04:05Z",
			Expected: time.Date(2023, time.January, 2, 3, 4, 5, 0, time.UTC),
		},
		{
			Input:    "2023-01-02T03:04:05+00:00",
			Expected: time.Date(2023, time.January, 2, 3, 4, 5, 0, time.UTC),
		},
		{
			Input:    "2023-01-02T03:04:05+09:00",
			Expected: time.Date(2023, time.January, 2, 3, 4, 5, 0, JST),
		},
		{
			Input: "2023-01-02T03:04:05+09:00-invalid-string",
			Error: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%s", testCase.Input), func(t *testing.T) {
			result, err := ParseTime(testCase.Input)
			if testCase.Error {
				if err == nil {
					t.Errorf("expect the call ending with an error. But no error returned.")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error was returned\n%s", err)
				}
				if result.Unix() != testCase.Expected.Unix() {
					t.Errorf("the result is not matching\nexpect:\n%s\nactual:\n%s\n", testCase.Expected, result)
				}
			}
		})
	}
}
