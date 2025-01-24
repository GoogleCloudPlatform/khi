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

package metadata

import (
	"encoding/json"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/google/go-cmp/cmp"
)

// Metadata test utility used from tests in another package

type MockedMetadataFactory struct {
	InstanciateResult Metadata
}

// Instanciate implements MetadataFactory.
func (m *MockedMetadataFactory) Instanciate() Metadata {
	return m.InstanciateResult
}

// MockedMetadataFactory implments MetadataFactory
var _ MetadataFactory = (*MockedMetadataFactory)(nil)

type debugMetadata struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
	Qux bool   `json:"-"`
}

// Labels implements Metadata.
func (*debugMetadata) Labels() *task.LabelSet {
	return task.NewLabelSet()
}

var _ Metadata = (*debugMetadata)(nil)

func (d *debugMetadata) ToSerializable() interface{} {
	return d
}
func TestMetadataMapIsSerializable(t *testing.T) {
	metadata := NewSet()
	metadata.LoadOrStore("testdata", &MockedMetadataFactory{InstanciateResult: &debugMetadata{
		Foo: "foo-value",
		Bar: 100,
		Qux: false,
	}})

	md, err := metadata.ToMap()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	resultInBytes, err := json.Marshal(md)

	if err != nil {
		t.Errorf("Unexpected error during marshaling metadata\n%v", err)
	}
	if string(resultInBytes) != "{\"testdata\":{\"foo\":\"foo-value\",\"bar\":100}}" {
		t.Errorf("Unexpected result of marshaling a metadata:%s", string(resultInBytes))
	}
}

func TestMetadataLoadOrStoreShouldReturnTheFirstInstance(t *testing.T) {
	metadata := NewSet()
	metadata.LoadOrStore("foo",
		&MockedMetadataFactory{
			&debugMetadata{
				Foo: "foo-value",
				Bar: 100,
				Qux: false,
			}})

	result := metadata.LoadOrStore("foo", &MockedMetadataFactory{
		InstanciateResult: &debugMetadata{
			Foo: "wrong-value",
			Bar: 200,
			Qux: false,
		}})

	if diff := cmp.Diff(result, &debugMetadata{Foo: "foo-value",
		Bar: 100,
		Qux: false}); diff != "" {
		t.Errorf("The 2nd MetadataSet LoadOrStore is different from the expected:\n%s", diff)
	}
}
