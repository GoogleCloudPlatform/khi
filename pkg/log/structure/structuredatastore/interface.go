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

import "github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredata"

// structure.Reader read its data from StructureDataStorageRef.
// Implementations may store its data in storage or compressed on its demand
type StructureDataStorageRef interface {
	// Get the current structuredata.StructureData
	// This may read the data from storage or compressed memory by its implementation
	Get() (structuredata.StructureData, error)

	// GetStore returns the reference to the store hold the actual data of this reference.
	GetStore() StructureDataStore
}

// StructureDataStore is a factory instanciating StructureDataStore
type StructureDataStore interface {
	// Store the given StructureData and return the reference to it.
	StoreStructureData(sd structuredata.StructureData) (StructureDataStorageRef, error)
}
