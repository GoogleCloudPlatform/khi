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
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
	"github.com/GoogleCloudPlatform/khi/pkg/parser/yaml/yamlutil"
)

type AnyAdapter struct {
	objectRef any
}

// GetReaderBackedByStore implements structure.ReaderDataAdapter.
func (a *AnyAdapter) GetReaderBackedByStore(store structuredatastore.StructureDataStore) (*structure.Reader, error) {
	yamlString, err := yamlutil.MarshalToYamlString(a.objectRef)
	if err != nil {
		return nil, err
	}
	return Yaml(yamlString).GetReaderBackedByStore(store)
}

func Any(objectRef any) *AnyAdapter {
	return &AnyAdapter{objectRef: objectRef}
}

var _ structure.ReaderDataAdapter = (*AnyAdapter)(nil)
