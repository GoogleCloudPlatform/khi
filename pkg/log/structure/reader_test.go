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

package structure

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredata"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
)

// testYamlAdapter is the copy of YamlAdapter in adapter package just for generating test structuredata from yaml string in test.
// adapters should be defined in adapter package and they depend on this package. To prevent this cyclic import and testing on its package, this test only purposed duplicated type is defined.
type testYamlAdapter struct {
	input string
}

// Returns adapter for parsing source yaml.
func testYaml(sourceYaml string) *testYamlAdapter {
	return &testYamlAdapter{
		input: sourceYaml,
	}
}

// GetReaderBackedByStore implements StructureDataAdapter.
func (y *testYamlAdapter) GetReaderBackedByStore(store structuredatastore.StructureDataStore) (*Reader, error) {
	sd, err := structuredata.DataFromYaml(y.input)
	if err != nil {
		return nil, err
	}
	sdstore, err := store.StoreStructureData(sd)
	if err != nil {
		return nil, err
	}
	return NewReader(sdstore), nil
}

var _ ReaderDataAdapter = (*testYamlAdapter)(nil)

func TestReader(t *testing.T) {
	readerFactory := NewReaderFactory(&structuredatastore.OnMemoryStructureDataStore{})
	type childTestCase struct {
		readerPath  string
		wantError   bool
		resultCount int
		filters     [][]ReaderFilter
	}
	testCases := []struct {
		Input     string
		Testcases []childTestCase
	}{
		{
			Input: `foo:
  bar:
    qux: quux`,
			Testcases: []childTestCase{
				{
					readerPath:  "non-exist",
					wantError:   false,
					resultCount: 0,
				},
				{
					readerPath:  "foo[].bar.qux",
					wantError:   false,
					resultCount: 0,
				},
				{
					readerPath:  "foo.bar",
					wantError:   false,
					resultCount: 1,
				},
				{
					readerPath:  "",
					wantError:   false,
					resultCount: 1,
				},
			},
		}, {
			Input: `foo:
- bar:
    qux:
    - apple
- bar:
    qux:
    - banana
    - grape`,
			Testcases: []childTestCase{
				{
					readerPath:  "foo[].bar",
					wantError:   false,
					resultCount: 2,
				},
				{
					readerPath:  "foo[].bar.qux[]",
					wantError:   false,
					resultCount: 3,
				},
				{
					readerPath:  "foo[].bar.qux",
					wantError:   false,
					resultCount: 2,
				},
			},
		},
		{
			Input: `foo:
- bar: 1
  qux: 
  - apple
  - banana
- bar: 2
  qux:
  - grape`,
			Testcases: []childTestCase{
				{
					readerPath:  "foo[].qux",
					wantError:   false,
					resultCount: 2,
				},
				{
					readerPath:  "foo[].qux",
					wantError:   false,
					resultCount: 1,
					filters: [][]ReaderFilter{
						{
							EqualFilter("bar", 1),
						},
					},
				},
				{
					readerPath:  "foo[].qux[]",
					wantError:   false,
					resultCount: 2,
					filters: [][]ReaderFilter{
						{
							EqualFilter("bar", 1),
						},
					},
				},
			},
		},
		{
			Input: `~`,
			Testcases: []childTestCase{
				{
					readerPath:  "foo",
					wantError:   false,
					resultCount: 0,
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Input-%d", i), func(t *testing.T) {
			rootReader, err := readerFactory.NewReader(testYaml(tc.Input))
			if err != nil {
				t.Fatal(err)
			}
			for _, ctc := range tc.Testcases {
				t.Run(ctc.readerPath, func(t *testing.T) {
					readers, err := rootReader.Reader(ctc.readerPath, ctc.filters...)
					if ctc.wantError {
						if err != nil {
							return
						}
						t.Fatal("no error returned")
					} else {
						if len(readers) != ctc.resultCount {
							t.Errorf("the result is not matching with the expected count: expected:%d,actual:%d\n \nvalues:%v", ctc.resultCount, len(readers), readers)

						}
					}
				})
			}
		})
	}
}

func TestReaderSingle(t *testing.T) {
	readerFactory := NewReaderFactory(&structuredatastore.OnMemoryStructureDataStore{})
	testCases := []struct {
		input   string
		path    string
		wantErr bool
	}{
		{
			input: `data:
  foo:
    bar: quux`,
			path:    "data.foo.bar",
			wantErr: false,
		},
		{
			input: `data:
  foo:
    bar: quux`,
			path:    "data.bar",
			wantErr: true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			reader, err := readerFactory.NewReader(testYaml(tc.input))
			if err != nil {
				t.Fatal(err)
			}

			singleReader, err := reader.ReaderSingle(tc.path)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Expected an error returned. no error returned")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if singleReader == nil {
					t.Errorf("result reader is nil")
				}
			}
		})
	}
}

func TestReadString(t *testing.T) {
	readerFactory := NewReaderFactory(&structuredatastore.OnMemoryStructureDataStore{})
	testCases := []struct {
		input     string
		path      string
		wantValue string
		wantErr   bool
	}{
		{
			input:     "data: hello world",
			path:      "data",
			wantValue: "hello world",
			wantErr:   false,
		},
		{
			input:     "data: value",
			path:      "not.existing.path",
			wantValue: "",
			wantErr:   true,
		},
		{
			input:     "data: \n - item1 \n - item2",
			path:      "data",
			wantValue: "",
			wantErr:   true,
		},
		{
			input:     "data: 1234",
			path:      "data",
			wantValue: "",
			wantErr:   true,
		},
		{
			input:     "data: ''",
			path:      "data",
			wantValue: "",
			wantErr:   false,
		},
		{
			input:     "data: null",
			path:      "data",
			wantValue: "",
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("input: %s, path: %s", tc.input, tc.path), func(t *testing.T) {
			reader, err := readerFactory.NewReader(testYaml(tc.input))
			if err != nil {
				t.Fatal(err)
			}

			gotValue, err := reader.ReadString(tc.path)

			if (err != nil) != tc.wantErr {
				t.Errorf("ReadString() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if gotValue != tc.wantValue {
				t.Errorf("ReadString() = %v, want %v", gotValue, tc.wantValue)
			}
		})
	}
}

func TestReadTimeAsString(t *testing.T) {
	readerFactory := NewReaderFactory(&structuredatastore.OnMemoryStructureDataStore{})
	testCases := []struct {
		name          string
		input         string
		expectedValue string
		wantError     bool
	}{{
		name:          "time yaml field as string",
		input:         `time: "2024-01-01T00:00:00+09:00Z"`,
		expectedValue: "2024-01-01T00:00:00+09:00Z",
		wantError:     false,
	}, {

		name:          "time yaml field as time.Time",
		input:         `time: 2024-01-01T00:00:00+09:00Z`,
		expectedValue: "2024-01-01T00:00:00+09:00Z",
		wantError:     false,
	}}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := readerFactory.NewReader(testYaml(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			gotValue, err := reader.ReadTimeAsString("time")
			if (err != nil) != tc.wantError {
				t.Errorf("ReadTimeAsString() error = %v, wantErr %v", err, tc.wantError)
				return
			}
			if gotValue != tc.expectedValue {
				t.Errorf("ReadTimeAsString() = %v, want %v", gotValue, tc.expectedValue)
			}
		})
	}
}

func TestReadStringOrDefault(t *testing.T) {
	readerFactory := NewReaderFactory(&structuredatastore.OnMemoryStructureDataStore{})
	testCases := []struct {
		input        string
		defaultValue string
		path         string
		wantValue    string
	}{
		{
			input:        "data: hello world",
			path:         "data",
			wantValue:    "hello world",
			defaultValue: "not found",
		},
		{
			input:        "data: value",
			path:         "not.existing.path",
			defaultValue: "not found",
			wantValue:    "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("input: %s, path: %s", tc.input, tc.path), func(t *testing.T) {
			reader, err := readerFactory.NewReader(testYaml(tc.input))
			if err != nil {
				t.Fatal(err)
			}

			gotValue := reader.ReadStringOrDefault(tc.path, tc.defaultValue)

			if gotValue != tc.wantValue {
				t.Errorf("ReadString() = %v, want %v", gotValue, tc.wantValue)
			}
		})
	}

}
