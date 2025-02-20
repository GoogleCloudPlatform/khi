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

package structuredatastore

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredata"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestLRUStructureDataStoreFactory(t *testing.T) {
	ITEM_COUNT := 10000
	lru := NewLRUStructureDataStoreFactory()
	for i := 0; i < ITEM_COUNT; i++ {
		data := fmt.Sprintf("textPayload: hello-%d\n", i)
		sd, err := structuredata.DataFromYaml(data)
		if err != nil {
			t.Fatal(err)
		}
		d, err := lru.StoreStructureData(sd)
		if err != nil {
			t.Errorf(err.Error())
		}
		_, err = d.Get()
		if err != nil {
			t.Errorf(err.Error())
		}
		// Needs to check Get() call twice to verify it's on the cache not to read from the storage
		sd, err = d.Get()
		if err != nil {
			t.Errorf(err.Error())
		}
		yaml, err := structuredata.ToYaml(sd)
		if err != nil {
			t.Errorf(err.Error())
		}
		if yaml != data {
			t.Errorf("expected %s, got %s", data, yaml)
		}
	}
}
