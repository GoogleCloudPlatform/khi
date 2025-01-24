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

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredata"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
)

func TestDirectAdapter(t *testing.T) {
	store := structuredatastore.OnMemoryStructureDataStore{}
	sd, err := structuredata.DataFromYaml("textPayload: hello world")
	if err != nil {
		t.Errorf(err.Error())
	}
	direct := Direct(sd)
	reader, err := direct.GetReaderBackedByStore(&store)
	if err != nil {
		t.Errorf(err.Error())
	}
	if reader.ReadStringOrDefault("textPayload", "") != "hello world" {
		t.Errorf("expected hello world, got %s", reader.ReadStringOrDefault("textPayload", ""))
	}
}
