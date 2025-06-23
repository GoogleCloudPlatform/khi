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
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// MetadataLabelsKey is a type-safe key for metadata label values.
type MetadataLabelsKey[T any] = typedmap.TypedKey[T]

// MetadataKey is a type-safe key for metadata.
type MetadataKey[T Metadata] = typedmap.TypedKey[T]

// NewMetadataLabelsKey creates a type-safe metadata key.
func NewMetadataLabelsKey[T any](key string) MetadataLabelsKey[T] {
	return typedmap.NewTypedKey[T](key)
}

// NewMetadataKey creates a type-safe metadata key.
func NewMetadataKey[T Metadata](key string) MetadataKey[T] {
	return typedmap.NewTypedKey[T](key)
}

// Metadata represents serializable data with labels.
type Metadata interface {
	// Converts to serializable format
	ToSerializable() interface{}
	// Returns associated labels
	Labels() *typedmap.ReadonlyTypedMap
}

func GetSerializableSubsetMapFromMetadataSet(metadataSet *typedmap.ReadonlyTypedMap, predicate func(m Metadata) bool) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for _, key := range metadataSet.Keys() {
		metadata, _ := typedmap.Get(metadataSet, NewMetadataKey[Metadata](key))
		if predicate(metadata) {
			result[key] = metadata.ToSerializable()
		}
	}
	return result, nil
}
