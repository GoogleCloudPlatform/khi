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

package structurev2

import (
	"testing"
	"time"
	"unique"
)

func toInternedStringArray(original []string) []unique.Handle[string] {
	result := make([]unique.Handle[string], len(original))
	for i := 0; i < len(original); i++ {
		result[i] = unique.Make(original[i])
	}
	return result
}

func TestYAMLNodeSerializer(t *testing.T) {
	testCase := []struct {
		Name     string
		Input    Node
		Expected string
	}{
		{
			Name: "scalar types",
			Input: &StandardMapNode{
				keys: toInternedStringArray([]string{"nil", "bool", "int", "float", "string", "time"}),
				values: []Node{
					NewStandardScalarNode[any](nil),
					NewStandardScalarNode(true),
					NewStandardScalarNode(42),
					NewStandardScalarNode(3.14),
					NewStandardScalarNode("foo"),
					NewStandardScalarNode(time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
			Expected: `nil: null
bool: true
int: 42
float: 3.140000
string: foo
time: 2022-01-01T00:00:00Z
`,
		},
		{
			Name: "simple map",
			Input: &StandardMapNode{
				keys: toInternedStringArray([]string{"foo", "bar"}),
				values: []Node{
					NewStandardScalarNode(42),
					NewStandardScalarNode(3.14)},
			},
			Expected: `foo: 42
bar: 3.140000
`,
		},
		{
			Name: "simple sequence",
			Input: &StandardSequenceNode{
				value: []Node{
					NewStandardScalarNode(42),
					NewStandardScalarNode(3.14)},
			},
			Expected: `- 42
- 3.140000
`,
		},
		{
			Name: "complex nested type",
			Input: &StandardMapNode{
				keys: toInternedStringArray([]string{"foo", "bar"}),
				values: []Node{
					&StandardMapNode{
						keys: toInternedStringArray([]string{"baz", "qux"}),
						values: []Node{
							NewStandardScalarNode(42),
							NewStandardScalarNode(3.14)},
					},
					&StandardSequenceNode{
						value: []Node{
							NewStandardScalarNode(42),
							NewStandardScalarNode(3.14)},
					},
				},
			},
			Expected: `foo:
    baz: 42
    qux: 3.140000
bar:
    - 42
    - 3.140000
`,
		},
	}

	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			reader := NewNodeReader(tc.Input)
			serialized, err := reader.Serialize("", &YAMLNodeSerializer{})
			if err != nil {
				t.Errorf("failed to serialize the given node structure: %s", err.Error())
			}
			if string(serialized) != tc.Expected {
				t.Errorf("expected serialized output to be %s but got %s", tc.Expected, serialized)
			}
		})
	}
}

func TestJSONNodeSerializer(t *testing.T) {
	testCase := []struct {
		Name     string
		Input    Node
		Expected string
	}{
		{
			Name: "scalar types",
			Input: &StandardMapNode{
				keys: toInternedStringArray([]string{"nil", "bool", "int", "float", "string", "time"}),
				values: []Node{
					NewStandardScalarNode[any](nil),
					NewStandardScalarNode(true),
					NewStandardScalarNode(42),
					NewStandardScalarNode(3.14),
					NewStandardScalarNode("foo"),
					NewStandardScalarNode(time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
			Expected: `{"nil":null,"bool":true,"int":42,"float":3.14,"string":"foo","time":"2022-01-01T00:00:00Z"}`,
		},
		{
			Name: "simple map",
			Input: &StandardMapNode{
				keys: toInternedStringArray([]string{"foo", "bar"}),
				values: []Node{
					NewStandardScalarNode(42),
					NewStandardScalarNode(3.14)},
			},
			Expected: `{"foo":42,"bar":3.14}`,
		},
		{
			Name: "simple sequence",
			Input: &StandardSequenceNode{
				value: []Node{
					NewStandardScalarNode(42),
					NewStandardScalarNode(3.14),
				},
			},
			Expected: `[42,3.14]`,
		},
		{
			Name: "complex nested type",
			Input: &StandardMapNode{
				keys: toInternedStringArray([]string{"foo", "bar"}),
				values: []Node{
					&StandardMapNode{
						keys: toInternedStringArray([]string{"baz", "qux"}),
						values: []Node{
							NewStandardScalarNode(42),
							NewStandardScalarNode(3.14),
						},
					},
					&StandardSequenceNode{
						value: []Node{
							NewStandardScalarNode(42),
							NewStandardScalarNode(3.14),
						},
					},
				},
			},
			Expected: `{"foo":{"baz":42,"qux":3.14},"bar":[42,3.14]}`,
		},
		{
			Name: "map containing \" in key and values",
			Input: &StandardMapNode{
				keys: toInternedStringArray([]string{"foo\"bar"}),
				values: []Node{
					NewStandardScalarNode("qux\"quux"),
				},
			},
			Expected: `{"foo\"bar":"qux\"quux"}`,
		},
	}

	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			reader := NewNodeReader(tc.Input)
			serialized, err := reader.Serialize("", &JSONNodeSerializer{})
			if err != nil {
				t.Errorf("failed to serialize the given node structure: %s", err.Error())
			}
			if string(serialized) != tc.Expected {
				t.Errorf("expected serialized output to be %s but got %s", tc.Expected, serialized)
			}
		})
	}
}
