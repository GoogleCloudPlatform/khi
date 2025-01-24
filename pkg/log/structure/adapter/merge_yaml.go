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
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/merger"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredata"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
)

// MergeYamlAdapter implements ReaderDataAdapter to get the Reader of a merged Yaml from 2 yaml strings.
type MergeYamlAdapter struct {
	prevYaml            string
	currentYaml         string
	mergeConfigResolver *merger.MergeConfigResolver
}

func MergeYaml(prevYaml string, currentYaml string, mergeConfigResolver *merger.MergeConfigResolver) *MergeYamlAdapter {
	return &MergeYamlAdapter{
		prevYaml:            prevYaml,
		currentYaml:         currentYaml,
		mergeConfigResolver: mergeConfigResolver,
	}
}

// GetReaderBackedByStore implements structure.ReaderDataAdapter.
func (y *MergeYamlAdapter) GetReaderBackedByStore(store structuredatastore.StructureDataStore) (*structure.Reader, error) {
	prevStructureData, err := structuredata.DataFromYaml(y.prevYaml)
	if err != nil {
		return nil, err
	}
	currentStructureData, err := structuredata.DataFromYaml(y.currentYaml)
	if err != nil {
		return nil, err
	}
	merged := merger.NewStrategicMergedStructureData("", prevStructureData, currentStructureData, y.mergeConfigResolver)
	storeRef, err := store.StoreStructureData(merged)
	if err != nil {
		return nil, err
	}
	return structure.NewReader(storeRef), nil
}

var _ structure.ReaderDataAdapter = (*MergeYamlAdapter)(nil)
