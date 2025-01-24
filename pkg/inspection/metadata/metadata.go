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
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

type Metadata interface {
	ToSerializable() interface{}
	Labels() *task.LabelSet
}

type MetadataFactory interface {
	Instanciate() Metadata
}

// MetadataSet is a type containing data used in frontend except the actual inspection data.
type MetadataSet struct {
	mapKeys map[string]interface{}
	rawMap  sync.Map
	mapLock sync.Mutex
}

func NewSet() *MetadataSet {
	return &MetadataSet{rawMap: sync.Map{}, mapKeys: map[string]interface{}{}}
}

func (m *MetadataSet) LoadOrStore(id string, factory MetadataFactory) Metadata {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	data, _ := m.rawMap.LoadOrStore(id, factory.Instanciate())
	m.mapKeys[id] = struct{}{}
	return data.(Metadata)
}

func (m *MetadataSet) ToMap(filters ...task.LabelFilter) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for key := range m.mapKeys {
		metadataAny, _ := m.rawMap.Load(key)
		md, converted := metadataAny.(Metadata)
		if !converted {
			return nil, fmt.Errorf("failed to convert a medatata to Metadata interface")
		}
		shouldInclude := true
		for _, filter := range filters {
			if !filter.Filter(md.Labels()) {
				shouldInclude = false
				break
			}
		}
		if !shouldInclude {
			continue
		}
		serializableValue := md.ToSerializable()
		if serializableValue != nil {
			result[key] = serializableValue
		}
	}
	return result, nil
}
