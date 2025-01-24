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

package adapter

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/merger"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
)

func TestYamlMergeAdapterTest(t *testing.T) {
	store := structuredatastore.OnMemoryStructureDataStore{}
	yamlAdapter := MergeYaml("foo: hello", "bar: world", &merger.MergeConfigResolver{})
	reader, err := yamlAdapter.GetReaderBackedByStore(&store)
	if err != nil {
		t.Errorf(err.Error())
	}
	if reader.ReadStringOrDefault("foo", "") != "hello" {
		t.Errorf("expected hello world, got %s", reader.ReadStringOrDefault("foo", ""))
	}
	if reader.ReadStringOrDefault("bar", "") != "world" {
		t.Errorf("expected world, got %s", reader.ReadStringOrDefault("bar", ""))
	}
}
