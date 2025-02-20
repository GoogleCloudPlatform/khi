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

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestParseSetFilter(t *testing.T) {
	var testAliasMap SetFilterAliasToItemsMap = map[string][]string{
		"foobar": {"foo", "bar"},
	}
	testCases := []struct {
		Filter         string
		AllowAny       bool
		AllowSubtract  bool
		ToLowerCase    bool
		ExpectedResult *SetFilterParseResult
	}{
		{
			Filter:        "",
			AllowAny:      false,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				Additives:       []string{},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
		},
		{
			Filter:        " ",
			AllowAny:      false,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				Additives:       []string{},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
		},
		{
			Filter:        " - ",
			AllowAny:      false,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				ValidationError: "Unexpected filter element `-`. Expecting item or alias after `-`.",
			},
		},
		{
			Filter:        " -@any ",
			AllowAny:      true,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				ValidationError: "Unsupported filter element `-@any`",
			},
		},
		{
			Filter:        "foo",
			AllowAny:      false,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				Additives: []string{
					"foo",
				},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
		},
		{
			Filter:        "Foo",
			AllowAny:      false,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				Additives: []string{
					"Foo",
				},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
		},
		{
			Filter:        " foo ",
			AllowAny:      false,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				Additives: []string{
					"foo",
				},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
		},
		{
			// qux is not included in the set. It should be ignroed
			Filter:        "foo bar -qux",
			AllowAny:      false,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				Additives:       []string{"bar", "foo"},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
		},
		{
			Filter:        "foo bar -foo",
			AllowAny:      false,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				Additives:       []string{"bar"},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
		},
		{
			Filter:        "@foobar -foo",
			AllowAny:      false,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				Additives:       []string{"bar"},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
		},
		{
			Filter:        "@invalidalias -foo",
			AllowAny:      true,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				ValidationError: "alias `invalidalias` was not found",
			},
		},
		{
			Filter:        "must,be,splitted,by,space",
			AllowAny:      true,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				ValidationError: "filter value must be whitespace splitted series of [a-zA-Z0-9\\-_]+",
			},
		},
		{
			Filter:        "@any -foo",
			AllowAny:      true,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				Additives:       []string{},
				Subtractives:    []string{"foo"},
				ValidationError: "",
				SubtractMode:    true,
			},
		},
		{
			Filter:        "@any -@foobar bar",
			AllowAny:      true,
			AllowSubtract: true,
			ExpectedResult: &SetFilterParseResult{
				Additives:       []string{},
				Subtractives:    []string{"foo"},
				ValidationError: "",
				SubtractMode:    true,
			},
		},
		{
			Filter:        "@foobar bar Quux",
			AllowAny:      true,
			AllowSubtract: true,
			ToLowerCase:   true,
			ExpectedResult: &SetFilterParseResult{
				Additives:       []string{"quux", "bar", "foo"},
				Subtractives:    []string{},
				ValidationError: "",
			},
		},
		{
			Filter:        "@any bar -Foo",
			AllowAny:      true,
			AllowSubtract: true,
			ToLowerCase:   true,
			ExpectedResult: &SetFilterParseResult{
				Additives:       []string{},
				Subtractives:    []string{"foo"},
				ValidationError: "",
				SubtractMode:    true,
			},
		},
		{
			Filter:        "-foo",
			AllowAny:      false,
			AllowSubtract: false,
			ExpectedResult: &SetFilterParseResult{
				ValidationError: "Subtract filter is not supported in this filter",
				SubtractMode:    false,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase %s", testCase.Filter), func(t *testing.T) {
			result, err := ParseSetFilter(testCase.Filter, testAliasMap, testCase.AllowAny, testCase.AllowSubtract, testCase.ToLowerCase)
			if err != nil {
				t.Errorf("unexpected error\n%s", err)
			}

			if diff := cmp.Diff(testCase.ExpectedResult, result); diff != "" {
				t.Errorf("ParseSetFilter result is not matching with the expected result\n%s", diff)
			}
		})
	}
}
