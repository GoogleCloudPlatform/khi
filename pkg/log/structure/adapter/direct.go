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

// DirectAdapter implements ReaderDataAdapter to pass StructureData directly into the Reader.
type DirectAdapter struct {
	sd structuredata.StructureData
}

func Direct(sd structuredata.StructureData) *DirectAdapter {
	return &DirectAdapter{sd: sd}
}

// GetReaderBackedByStore implements structure.ReaderDataAdapter.
func (d *DirectAdapter) GetReaderBackedByStore(store structuredatastore.StructureDataStore) (*structure.Reader, error) {
	sd, err := store.StoreStructureData(d.sd)
	if err != nil {
		return nil, err
	}
	return structure.NewReader(sd), nil
}

var _ structure.ReaderDataAdapter = (*DirectAdapter)(nil)
