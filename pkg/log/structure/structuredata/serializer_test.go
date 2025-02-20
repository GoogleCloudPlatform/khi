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

import (
	"fmt"
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestYamlString(t *testing.T) {
	testCases := []struct {
		input     string
		wantValue string
		wantErr   bool
	}{
		{
			input: "data: hello world",
			wantValue: `data: hello world
`,
		},
		{
			input: `foo:
  bar:
   - apple
   - banana
   - grape
`,
			wantValue: `foo:
  bar:
  - apple
  - banana
  - grape
`,
		},
		{
			input: `foo:
  bar: ~`,
			wantValue: `foo:
  bar: null
`,
		},
		{
			input: `10`,
			wantValue: `10
`,
		},
		{
			input: `foo: hello`,
			wantValue: `foo: hello
`,
		},
		{
			input: `foo:
- bool: true
- int: 100
- float: 3.14
`,
			wantValue: `foo:
- bool: true
- int: 100
- float: 3.14
`,
		},
		{ // field with invalid characters directly used in golang struct
			input: `foo:
  bar:
    k:{"foo/bar"}: qux
    k:{"foo/qux"}: quux`,
			wantValue: `foo:
  bar:
    k:{"foo/bar"}: qux
    k:{"foo/qux"}: quux
`,
		},
		{ // field with invalid characters directly used in golang struct
			input: `foo:
  bar:
    k:{"foo/bar","apple"}: qux
    k:{"foo/qux","banana"}: quux`,
			wantValue: `foo:
  bar:
    k:{"foo/bar","apple"}: qux
    k:{"foo/qux","banana"}: quux
`,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("input: %s", tc.input), func(t *testing.T) {
			input, err := DataFromYaml(tc.input)
			if err != nil {
				t.Fatal(err)
			}

			gotValue, err := ToYaml(input)

			if (err != nil) != tc.wantErr {
				t.Errorf("ReadString() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if gotValue != tc.wantValue {
				t.Errorf("ReadString() = \n%v, want \n%v", gotValue, tc.wantValue)
			}
		})
	}
}

func TestYamlStringOrder(t *testing.T) {
	// Test the reader is keeping the order of map fields
	fields := []string{}
	for i := 0; i < 10; i++ {
		fields = append(fields, fmt.Sprintf("field-%d", i))
	}
	shufleFields(fields)
	yaml := ""
	for i, field := range fields {
		yaml += fmt.Sprintf("%s: %d\n", field, i)
	}

	structure, err := DataFromYaml(yaml)
	if err != nil {
		t.Fatal(err)
	}

	recoveredYaml, err := ToYaml(structure)
	if err != nil {
		t.Fatal(err)
	}

	if yaml != recoveredYaml {
		t.Errorf("Result is not matching with the input YAML data\nINPUT:\n\n%s\n\nOUTPUT:\n\n%s", yaml, recoveredYaml)
	}
}

func TestJsonString(t *testing.T) {
	testCases := []struct {
		input     string
		wantValue string
		wantErr   bool
	}{
		{
			input:     "data: hello world",
			wantValue: `{"data":"hello world"}`,
		},
		{
			input: `foo:
  bar:
   - apple
   - banana
   - grape
`,
			wantValue: `{"foo":{"bar":["apple","banana","grape"]}}`,
		},
		{
			input: `foo:
  bar:
    qux: apple
`,
			wantValue: `{"foo":{"bar":{"qux":"apple"}}}`,
		},
		// Scalar values at the path
		{
			input: `foo:
  bar: ~`,
			wantValue: `{"foo":{"bar":null}}`,
		},
		{
			input:     `foo: hello`,
			wantValue: `{"foo":"hello"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("input: %s", tc.input), func(t *testing.T) {
			st, err := DataFromYaml(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			gotValue, err := ToJson(st)

			if (err != nil) != tc.wantErr {
				t.Errorf("ReadString() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if gotValue != tc.wantValue {
				t.Errorf("ReadString() = \n%v, want \n%v", gotValue, tc.wantValue)
			}
		})
	}
}
