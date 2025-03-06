// Copyright 2025 Google LLC
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

package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToGithubAnchorHash(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{
			input: "Simple Text",
			want:  "simple-text",
		},
		{
			input: "Simple_Text",
			want:  "simpletext",
		},
		{
			input: " simple text ",
			want:  "simple-text",
		},
		{
			input: "Simple(Text)",
			want:  "simpletext",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual := ToGithubAnchorHash(tc.input)
			assert.Equal(t, tc.want, actual)
		})
	}
}
