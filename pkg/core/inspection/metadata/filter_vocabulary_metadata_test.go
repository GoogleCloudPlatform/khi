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

package inspectionmetadata

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTimelineTypeVocabulary_PathKey(t *testing.T) {
	testCases := []struct {
		name  string
		label string
		want  string
	}{
		{
			name:  "camelCase label",
			label: "apiVersion",
			want:  "apiversion",
		},
		{
			name:  "mixed case with spaces",
			label: "Managed Airflow",
			want:  "managed airflow",
		},
		{
			name:  "already lowercase",
			label: "namespace",
			want:  "namespace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := TimelineTypeVocabulary{Label: tc.label}
			if got := v.PathKey(); got != tc.want {
				t.Errorf("PathKey() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFilterVocabularyMetadata_SetAndGetClonesSlices(t *testing.T) {
	m := NewFilterVocabularyMetadata()
	input := FilterVocabulary{
		TimelineTypes: []TimelineTypeVocabulary{
			{Label: "kind", NameCount: 2, ExampleNames: []string{"node", "pod"}},
		},
		LogTypes: []string{"k8s_audit"},
	}
	m.Set(input)

	input.TimelineTypes[0].ExampleNames[0] = "mutated"
	input.LogTypes[0] = "mutated"

	got := m.Get()
	got.TimelineTypes[0].ExampleNames[1] = "mutated-after-get"
	got.LogTypes[0] = "mutated-after-get"

	want := FilterVocabulary{
		TimelineTypes: []TimelineTypeVocabulary{
			{Label: "kind", NameCount: 2, ExampleNames: []string{"node", "pod"}},
		},
		LogTypes: []string{"k8s_audit"},
	}
	if diff := cmp.Diff(want, m.Get()); diff != "" {
		t.Errorf("Get() mismatch (-want +got):\n%s", diff)
	}
}
