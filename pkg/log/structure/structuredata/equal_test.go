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

package structuredata

import "testing"

func TestEqualStructureData(t *testing.T) {
	testCases := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name: "basic map",
			a: `foo:
  bar: 10
  qux: 20
`,
			b: `foo:
  bar: 10
  qux: 20
`,
			expected: true,
		},
		{
			name: "basic map with differnt values",
			a: `foo:
  bar: 10
  qux: 20
`,
			b: `foo:
  bar: 10
  qux: 30
`,
			expected: false,
		},
		{
			name: "basic map with differnt keys",
			a: `foo:
  bar: 10
  qux: 20
`,
			b: `foo:
  bar: 10
  quux: 20
`,
			expected: false,
		},
		{
			name: "basic array",
			a: `foo:
  - bar: 10
  - bar: 20
  - bar: 30
`,
			b: `foo:
  - bar: 10
  - bar: 20
  - bar: 30
`, expected: true,
		},
		{
			name: "basic array with different value",
			a: `foo:
  - bar: 10
  - bar: 20
  - bar: 30
`,
			b: `foo:
  - bar: 10
  - bar: 21
  - bar: 30
`, expected: false,
		},
		{
			name: "deeper map",
			a: `foo:
  bar:
    qux: 
      quux: 10
`,
			b: `foo:
  bar:
    qux: 
      quux: 10
`, expected: true,
		},
		{
			name: "deeper map with different value",
			a: `foo:
  bar:
    qux: 
      quux: 10
`,
			b: `foo:
  bar:
    qux: 
      quux: 11
`, expected: false,
		}, {
			name: "with different types",
			a: `foo:
  bar:
    qux: 10`,
			b: `foo:
- bar:
    qux: 10`,
			expected: false,
		},
		{
			name: "with different key length",
			a: `foo:
  - 0
  - 1
`,
			b: `foo:
  - 0
  - 1
  - 2
`,
			expected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			aStructure, err := DataFromYaml(tc.a)
			if err != nil {
				t.Fatal(err)
			}
			bStructure, err := DataFromYaml(tc.b)
			if err != nil {
				t.Fatal(err)
			}
			result, err := EqualStructureData(aStructure, bStructure)
			if err != nil {
				t.Fatal(err)
			}
			if result != tc.expected {
				t.Errorf("expected %t but %t was returned", tc.expected, result)
			}
		})
	}
}
