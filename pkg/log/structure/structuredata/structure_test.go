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

	"github.com/google/go-cmp/cmp"
)

func TestYamlNodeStructuredData(t *testing.T) {
	type valueTestCase struct {
		field          string
		value          any
		isValueOfValue bool
		wantErr        bool
	}
	type testCase struct {
		yaml         string
		expectedKeys []string
		expectedType StructuredDataFieldType
		valueTests   []valueTestCase
	}
	tests := []testCase{
		{
			yaml: `
            - value1
            - value2
            `,
			expectedKeys: []string{"0", "1"},
			expectedType: StructuredTypeArray,
			valueTests: []valueTestCase{
				{
					field:          "0",
					value:          "value1",
					isValueOfValue: true,
					wantErr:        false,
				},
				{
					field:          "-1",
					value:          "",
					isValueOfValue: true,
					wantErr:        true,
				},
				{
					field:          "3",
					value:          "",
					isValueOfValue: true,
					wantErr:        true,
				},
				{
					field:          "foo",
					value:          "",
					isValueOfValue: true,
					wantErr:        true,
				},
			},
		},
		{
			yaml: `{
                key1: value1,
                key2: value2
            }`,
			expectedKeys: []string{"key1", "key2"},
			expectedType: StructuredTypeMap,
			valueTests: []valueTestCase{
				{
					field:          "key1",
					value:          "value1",
					isValueOfValue: true,
					wantErr:        false,
				},
				{
					field:          "key3",
					value:          "",
					isValueOfValue: true,
					wantErr:        true,
				},
			},
		},
		{
			yaml: `
            value
            `,
			expectedKeys: []string{""},
			expectedType: StructuredTypeScalar,
			valueTests: []valueTestCase{
				{
					field:          "",
					value:          "value",
					isValueOfValue: false,
					wantErr:        false,
				},
				{
					field:          "foo",
					value:          "value",
					isValueOfValue: false,
					wantErr:        true,
				},
			},
		},
		{
			yaml: `
bool: true
int: 3
float: 3.14
null: ~
`,
			expectedKeys: []string{"bool", "int", "float", "null"},
			expectedType: StructuredTypeMap,
			valueTests: []valueTestCase{
				{
					field:          "bool",
					value:          true,
					isValueOfValue: true,
					wantErr:        false,
				},
				{
					field:          "int",
					value:          3,
					isValueOfValue: true,
					wantErr:        false,
				},
				{
					field:          "float",
					value:          3.14,
					isValueOfValue: true,
					wantErr:        false,
				},
				{
					field:          "null",
					value:          nil,
					isValueOfValue: true,
					wantErr:        false,
				},
			},
		},
		{
			yaml:         "",
			expectedKeys: []string{},
			expectedType: StructuredTypeMap,
			valueTests: []valueTestCase{
				{
					field:   "foo",
					value:   "",
					wantErr: true,
				},
			},
		},
		{
			yaml:         "  ",
			expectedKeys: []string{},
			expectedType: StructuredTypeMap,
			valueTests: []valueTestCase{
				{
					field:   "foo",
					value:   "",
					wantErr: true,
				},
			},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("Test case %d", i), func(t *testing.T) {
			sd, err := DataFromYaml(tc.yaml)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("Type", func(t *testing.T) {
				ty, err := sd.Type()
				if err != nil {
					t.Fatal(err)
				}
				if ty != tc.expectedType {
					t.Errorf("unexpected type: %v\nexpected type: %v", ty, tc.expectedType)
				}
			})
			t.Run("Keys", func(t *testing.T) {
				keys, err := sd.Keys()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !cmp.Equal(keys, tc.expectedKeys) {
					t.Errorf("unexpected keys: %v\nexpected keys: %v", keys, tc.expectedKeys)
				}
			})

			for _, vtc := range tc.valueTests {
				t.Run(fmt.Sprintf("Value-%s", vtc.field), func(t *testing.T) {
					value, err := sd.Value(vtc.field)
					if vtc.wantErr {
						if err == nil {
							t.Fatalf("wanted an error but no err")
						} else {
							return
						}
					}
					if err != nil {
						t.Fatal(err)
					}
					if vtc.isValueOfValue {
						st := value.(StructureData)
						value, err = st.Value("")
						if err != nil {
							t.Fatal(err)
						}
					}
					if diff := cmp.Diff(value, vtc.value); diff != "" {
						t.Errorf("value result is not matching expected result\n%s", diff)
					}
				})
			}
		})
	}
}
