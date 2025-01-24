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
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredata"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
)

// YamlAdapter implements ReaderDataAdapter to get the Reader from YAML string.
type YamlAdapter struct {
	sourceYaml string
}

// Returns adapter for parsing source yaml.
func Yaml(sourceYaml string) *YamlAdapter {
	return &YamlAdapter{
		sourceYaml: sourceYaml,
	}
}

// GetReaderBackedByStore implements StructureDataAdapter.
func (y *YamlAdapter) GetReaderBackedByStore(store structuredatastore.StructureDataStore) (*structure.Reader, error) {
	sd, err := structuredata.DataFromYaml(y.sourceYaml)
	if err != nil {
		return nil, err
	}
	sdstore, err := store.StoreStructureData(sd)
	if err != nil {
		return nil, err
	}
	return structure.NewReader(sdstore), nil
}

var _ structure.ReaderDataAdapter = (*YamlAdapter)(nil)
