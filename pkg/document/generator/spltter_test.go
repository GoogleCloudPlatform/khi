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

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestSplitToDocumentSections(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expected       []*DocumentSection
		expectedErrMsg string
	}{
		{
			name: "single generated section",
			input: `<!-- BEGIN GENERATED PART:generated-id-1-->
Generated content 1
<!-- END GENERATED PART:generated-id-1-->
`,
			expected: []*DocumentSection{
				{
					Type: SectionTypeGenerated,
					ID:   "generated-id-1",
					Body: "<!-- BEGIN GENERATED PART:generated-id-1-->\nGenerated content 1\n<!-- END GENERATED PART:generated-id-1-->",
				},
			},
		},
		{
			name: "multiple generated sections",
			input: `<!-- BEGIN GENERATED PART:generated-id-1-->
Generated content 1
<!-- END GENERATED PART:generated-id-1-->

<!-- BEGIN GENERATED PART:generated-id-2-->
Generated content 2
<!-- END GENERATED PART:generated-id-2-->
`,
			expected: []*DocumentSection{
				{
					Type: SectionTypeGenerated,
					ID:   "generated-id-1",
					Body: "<!-- BEGIN GENERATED PART:generated-id-1-->\nGenerated content 1\n<!-- END GENERATED PART:generated-id-1-->",
				},
				{
					Type: SectionTypeAmend,
					ID:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // Hash of amend content
					Body: "",
				},
				{
					Type: SectionTypeGenerated,
					ID:   "generated-id-2",
					Body: "<!-- BEGIN GENERATED PART:generated-id-2-->\nGenerated content 2\n<!-- END GENERATED PART:generated-id-2-->",
				},
			},
		},
		{
			name: "generated section with amend",
			input: `<!-- BEGIN GENERATED PART:generated-id-1-->
Generated content 1
<!-- END GENERATED PART:generated-id-1-->

Amend content 1

<!-- BEGIN GENERATED PART:generated-id-2-->
Generated content 2
<!-- END GENERATED PART:generated-id-2-->

Amend content 2
`,
			expected: []*DocumentSection{
				{
					Type: SectionTypeGenerated,
					ID:   "generated-id-1",
					Body: "<!-- BEGIN GENERATED PART:generated-id-1-->\nGenerated content 1\n<!-- END GENERATED PART:generated-id-1-->",
				},
				{
					Type: SectionTypeAmend,
					ID:   "8425062b6f9c5ce9895ebb6fcd8d3c58c68887c14c8628a33e8f604dac84e919", // Hash of amend content
					Body: "\nAmend content 1\n",
				},
				{
					Type: SectionTypeGenerated,
					ID:   "generated-id-2",
					Body: "<!-- BEGIN GENERATED PART:generated-id-2-->\nGenerated content 2\n<!-- END GENERATED PART:generated-id-2-->",
				},
				{
					Type: SectionTypeAmend,
					ID:   "76b08fe06ecd34111bc58e645360d32593cdfdb797d70f84e7ed0c3cf3103374", // Hash of amend content
					Body: "\nAmend content 2\n",
				},
			},
		},
		{
			name: "mismatched end tag",
			input: `<!-- BEGIN GENERATED PART:generated-id-1-->
Generated content 1
<!-- END GENERATED PART:generated-id-2-->
`,
			expectedErrMsg: "invalid end of section. section id generated-id-2 ended but the id is not matching with the previous section id generated-id-1. line 3",
		},
		{
			name: "end tag without begin tag",
			input: `<!-- END GENERATED PART:generated-id-1-->
`,
			expectedErrMsg: "invalid end of section. section id generated-id-1 ended but not began. line 1",
		},
		{
			name:     "empty input",
			input:    ``,
			expected: nil,
		},
		{
			name: "only amend",
			input: `
Amend content
`,
			expected: []*DocumentSection{
				{
					Type: SectionTypeAmend,
					ID:   "1ff547697cd3b7542f7bb024812b201fc77e31bd605f95f247f41b585286c464",
					Body: "\nAmend content\n",
				},
			},
		},
		// Test case for begin tag without matching end tag -- generated section at the end
		{
			name: "begin tag without end tag",
			input: `
Some amend content

<!-- BEGIN GENERATED PART:generated-id-1-->
Generated content at end
`,
			expectedErrMsg: "invalid end of section. section id generated-id-1 began but not ended",
		},
		{
			name: "begin tag appears twice",
			input: `<!-- BEGIN GENERATED PART:generated-id-1-->
Generated content 1
<!-- BEGIN GENERATED PART:generated-id-2-->
Generated content 2
<!-- END GENERATED PART:generated-id-2-->
`,
			expectedErrMsg: "invalid begin of section. section began twice. line 3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := SplitToDocumentSections(tc.input)

			if tc.expectedErrMsg != "" {
				if diff := cmp.Diff(tc.expectedErrMsg, err.Error()); diff != "" {
					t.Errorf("Error message mismatch (-want +got):\n%s", diff)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}

				if diff := cmp.Diff(tc.expected, actual); diff != "" {
					t.Errorf("Sections do not match (-want +got):\n%s", diff)
				}
			}

		})

	}
}

func TestReadIdFromGeneratedSectionComment(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "<!-- BEGIN GENERATED PART:my-id-->",
			expected: "my-id",
		},
		{
			input:    "<!-- END GENERATED PART: my-id -->",
			expected: "my-id",
		},
		{
			input:    "<!-- BEGIN GENERATED PART:  my-id  -->",
			expected: "my-id",
		},
		{
			input:    "<!-- BEGIN GENERATED PART:my-id", // Missing suffix
			expected: "my-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual := readIdFromGeneratedSectionComment(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
