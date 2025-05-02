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
)

func TestYAMLNodeSerializer(t *testing.T) {
	testCase := []struct {
		Name     string
		Input    Node
		Expected string
	}{
		{
			Name: "scalar types",
			Input: &StandardMapNode{
				keys: []string{"nil", "bool", "int", "float", "string", "time"},
				values: []Node{
					&StandardScalarNode[any]{value: nil},
					&StandardScalarNode[bool]{value: true},
					&StandardScalarNode[int]{value: 42},
					&StandardScalarNode[float64]{value: 3.14},
					&StandardScalarNode[string]{value: "foo"},
					&StandardScalarNode[time.Time]{value: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
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
				keys: []string{"foo", "bar"},
				values: []Node{
					&StandardScalarNode[int]{value: 42},
					&StandardScalarNode[float64]{value: 3.14},
				},
			},
			Expected: `foo: 42
bar: 3.140000
`,
		},
		{
			Name: "simple sequence",
			Input: &StandardSequenceNode{
				value: []Node{
					&StandardScalarNode[int]{value: 42},
					&StandardScalarNode[float64]{value: 3.14},
				},
			},
			Expected: `- 42
- 3.140000
`,
		},
		{
			Name: "complex nested type",
			Input: &StandardMapNode{
				keys: []string{"foo", "bar"},
				values: []Node{
					&StandardMapNode{
						keys: []string{"baz", "qux"},
						values: []Node{
							&StandardScalarNode[int]{value: 42},
							&StandardScalarNode[float64]{value: 3.14},
						},
					},
					&StandardSequenceNode{
						value: []Node{
							&StandardScalarNode[int]{value: 4},
							&StandardScalarNode[float64]{value: 3.14},
						},
					},
				},
			},
			Expected: `foo:
    baz: 42
    qux: 3.140000
bar:
    - 4
    - 3.140000
`,
		},
	}

	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			reader := NewNodeReader(tc.Input)
			serialized, err := reader.Serialize(&YAMLNodeSerializer{})
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
				keys: []string{"nil", "bool", "int", "float", "string", "time"},
				values: []Node{
					&StandardScalarNode[any]{value: nil},
					&StandardScalarNode[bool]{value: true},
					&StandardScalarNode[int]{value: 42},
					&StandardScalarNode[float64]{value: 3.14},
					&StandardScalarNode[string]{value: "foo"},
					&StandardScalarNode[time.Time]{value: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			Expected: `{"nil":null,"bool":true,"int":42,"float":3.14,"string":"foo","time":"2022-01-01T00:00:00Z"}`,
		},
		{
			Name: "simple map",
			Input: &StandardMapNode{
				keys: []string{"foo", "bar"},
				values: []Node{
					&StandardScalarNode[int]{value: 42},
					&StandardScalarNode[float64]{value: 3.14},
				},
			},
			Expected: `{"foo":42,"bar":3.14}`,
		},
		{
			Name: "simple sequence",
			Input: &StandardSequenceNode{
				value: []Node{
					&StandardScalarNode[int]{value: 42},
					&StandardScalarNode[float64]{value: 3.14},
				},
			},
			Expected: `[42,3.14]`,
		},
		{
			Name: "complex nested type",
			Input: &StandardMapNode{
				keys: []string{"foo", "bar"},
				values: []Node{
					&StandardMapNode{
						keys: []string{"baz", "qux"},
						values: []Node{
							&StandardScalarNode[int]{value: 42},
							&StandardScalarNode[float64]{value: 3.14},
						},
					},
					&StandardSequenceNode{
						value: []Node{
							&StandardScalarNode[int]{value: 4},
							&StandardScalarNode[float64]{value: 3.14},
						},
					},
				},
			},
			Expected: `{"foo":{"baz":42,"qux":3.14},"bar":[4,3.14]}`,
		},
	}

	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			reader := NewNodeReader(tc.Input)
			serialized, err := reader.Serialize(&JSONNodeSerializer{})
			if err != nil {
				t.Errorf("failed to serialize the given node structure: %s", err.Error())
			}
			if string(serialized) != tc.Expected {
				t.Errorf("expected serialized output to be %s but got %s", tc.Expected, serialized)
			}
		})
	}
}
