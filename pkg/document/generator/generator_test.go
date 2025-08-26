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
)

func TestGenerateDocumentString(t *testing.T) {
	testCases := []struct {
		name                     string
		destinationString        string
		templateString           string
		templateName             string
		ignoreNonMatchingSection bool
		want                     string
		wantErr                  error
	}{
		{
			name:              "with empty destination string",
			destinationString: ``,
			templateString: `{{define "testTemplate"}}
<!-- BEGIN GENERATED PART: test-id -->
Generated content 1
<!-- END GENERATED PART: test-id -->
{{end}}`,
			templateName:             "testTemplate",
			ignoreNonMatchingSection: false,
			want: `<!-- BEGIN GENERATED PART: test-id -->
Generated content 1
<!-- END GENERATED PART: test-id -->
`,
		},
		{
			name: "with a non-empty destination string",
			destinationString: `
<!-- BEGIN GENERATED PART: test-id-1 -->
Generated content 1
<!-- END GENERATED PART: test-id-1 -->
This is additional string amended after content generation.
This is another line of amended line.`,
			templateString: `{{define "testTemplate"}}
<!-- BEGIN GENERATED PART: test-id-1 -->
Generated content 1
<!-- END GENERATED PART: test-id-1 -->
<!-- BEGIN GENERATED PART: test-id-2 -->
Generated content 2
<!-- END GENERATED PART: test-id-2 -->
{{end}}`,
			templateName:             "testTemplate",
			ignoreNonMatchingSection: false,
			want: `
<!-- BEGIN GENERATED PART: test-id-1 -->
Generated content 1
<!-- END GENERATED PART: test-id-1 -->
This is additional string amended after content generation.
This is another line of amended line.
<!-- BEGIN GENERATED PART: test-id-2 -->
Generated content 2
<!-- END GENERATED PART: test-id-2 -->
`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gen, err := newDocumentGeneratorFromStringTemplate(tc.templateString)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			actual, err := gen.generateDocumentString(tc.destinationString, tc.templateName, nil, tc.ignoreNonMatchingSection)
			if tc.wantErr != nil {
				if diff := cmp.Diff(tc.wantErr, err); diff != "" {
					t.Errorf("Error mismatch (-want +got):\n%s", diff)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if diff := cmp.Diff(tc.want, actual); diff != "" {
					t.Errorf("generateDocumentString() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestConcatAmendedContents(t *testing.T) {
	testCases := []struct {
		name                              string
		generated                         []*DocumentSection
		prev                              []*DocumentSection
		ignoreNonMatchingGeneratedSection bool
		wantResult                        string
		wantErr                           string
	}{
		{
			name: "no amended sections",
			generated: []*DocumentSection{
				{ID: "generated-1", Type: SectionTypeGenerated, Body: "Generated 1"},
				{ID: "generated-2", Type: SectionTypeGenerated, Body: "Generated 2"},
			},
			prev:                              []*DocumentSection{},
			ignoreNonMatchingGeneratedSection: false,
			wantResult:                        "Generated 1\nGenerated 2\n",
		},
		{
			name: "single amended section at beginning",
			generated: []*DocumentSection{
				{ID: "generated-1", Type: SectionTypeGenerated, Body: "Generated 1"},
			},
			prev: []*DocumentSection{
				{ID: "amended-1", Type: SectionTypeAmend, Body: "Amended 1"},
			},
			ignoreNonMatchingGeneratedSection: false,
			wantResult:                        "Amended 1\nGenerated 1\n",
		},
		{
			name: "new generated section and a single amended section",
			generated: []*DocumentSection{
				{ID: "generated-1", Type: SectionTypeGenerated, Body: "Generated 1"},
				{ID: "generated-2", Type: SectionTypeGenerated, Body: "Generated 2"},
			},
			prev: []*DocumentSection{
				{ID: "generated-1", Type: SectionTypeGenerated, Body: "Generated 1"},
				{ID: "amended-1", Type: SectionTypeAmend, Body: "Amended 1"},
			},
			ignoreNonMatchingGeneratedSection: false,
			wantResult:                        "Generated 1\nAmended 1\nGenerated 2\n",
		},
		{
			name: "multiple amended sections",
			generated: []*DocumentSection{
				{ID: "generated-1", Body: "Generated 1"},
				{ID: "generated-2", Body: "Generated 2"},
			},
			prev: []*DocumentSection{
				{ID: "amended-1", Type: SectionTypeAmend, Body: "Amended 1"},
				{ID: "generated-1", Body: "Generated 1"},
				{ID: "amended-2", Type: SectionTypeAmend, Body: "Amended 2"},
				{ID: "generated-2", Body: "Generated 2"},
				{ID: "amended-3", Type: SectionTypeAmend, Body: "Amended 3"},
			},
			ignoreNonMatchingGeneratedSection: false,
			wantResult:                        "Amended 1\nGenerated 1\nAmended 2\nGenerated 2\nAmended 3\n",
		},
		{
			name:      "no generated sections",
			generated: []*DocumentSection{},
			prev: []*DocumentSection{
				{ID: "amended-1", Type: SectionTypeAmend, Body: "Amended 1"},
				{ID: "generated-1", Body: "Generated 1"},
				{ID: "amended-2", Type: SectionTypeAmend, Body: "Amended 2"},
			},
			ignoreNonMatchingGeneratedSection: false,
			wantErr:                           "previous amended sections belongs to other generated sections is not used. Unused ids [generated-1]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := concatAmendedContents(tc.generated, tc.prev, tc.ignoreNonMatchingGeneratedSection)
			if tc.wantErr != "" {
				if diff := cmp.Diff(tc.wantErr, err.Error()); diff != "" {
					t.Errorf("Error message mismatch (-want +got):\n%s", diff)
				}

			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if diff := cmp.Diff(tc.wantResult, actual); diff != "" {
					t.Errorf("concatAmendedContents() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
