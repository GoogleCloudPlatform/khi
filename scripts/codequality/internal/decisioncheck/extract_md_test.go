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

package decisioncheck

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestExtractMarkdownSections verifies that substantive H1..H4 sections are extracted while H5+, non-heading hash tokens, fenced headings, and empty sections are skipped.
func TestExtractMarkdownSections(t *testing.T) {
	doc := `# Demo Package

Introductory overview of Demo Package.

## Pipeline Diagram

` + "```mermaid" + `
# Heading inside backtick fence
graph TD
    Ingester --> Mapper
` + "```" + `

## Task Descriptions

### Core Tasks

- **demoLogIngesterTask**: Ingests logs.

#### Deep Subtask

Details with #hashtag and bare hashes:
##

##### Ignored H5

Ignored body under H5.

## Blank Only

## Empty Backtick Fence

` + "```go" + `
` + "```" + `

## Empty Tilde Fence

~~~
~~~

## Tilde Code Sample

~~~bash
# Heading inside tilde fence
echo ok
~~~

## After Tilde

Final section body.
`

	gotSections := ExtractMarkdownSections("pkg/demo", "pkg/demo/README.md", doc)

	wantSections := []ExtractedItem{
		{
			Target:    "pkg/demo",
			File:      "pkg/demo/README.md",
			Line:      1,
			StartLine: 1,
			EndLine:   3,
			Heading:   "Demo Package",
		},
		{
			Target:    "pkg/demo",
			File:      "pkg/demo/README.md",
			Line:      5,
			StartLine: 5,
			EndLine:   11,
			Heading:   "Pipeline Diagram",
		},
		{
			Target:    "pkg/demo",
			File:      "pkg/demo/README.md",
			Line:      15,
			StartLine: 15,
			EndLine:   17,
			Heading:   "Core Tasks",
		},
		{
			Target:    "pkg/demo",
			File:      "pkg/demo/README.md",
			Line:      19,
			StartLine: 19,
			EndLine:   26,
			Heading:   "Deep Subtask",
		},
		{
			Target:    "pkg/demo",
			File:      "pkg/demo/README.md",
			Line:      40,
			StartLine: 40,
			EndLine:   45,
			Heading:   "Tilde Code Sample",
		},
		{
			Target:    "pkg/demo",
			File:      "pkg/demo/README.md",
			Line:      47,
			StartLine: 47,
			EndLine:   49,
			Heading:   "After Tilde",
		},
	}
	if diff := cmp.Diff(wantSections, gotSections); diff != "" {
		t.Errorf("ExtractMarkdownSections() mismatch (-want +got):\n%s", diff)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	testCases := []struct {
		name   string
		inputs []string
		want   string
	}{
		// Skips leading empty strings and returns the first non-empty candidate.
		{
			name:   "returns first non-empty value after empty entries",
			inputs: []string{"", "fallback", "later"},
			want:   "fallback",
		},
		// Returns an empty string when every candidate value is empty.
		{
			name:   "returns empty when all inputs are empty",
			inputs: []string{"", ""},
			want:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := firstNonEmpty(tc.inputs...)

			if got != tc.want {
				t.Errorf("firstNonEmpty(%q) = %q, want %q", tc.inputs, got, tc.want)
			}
		})
	}
}
