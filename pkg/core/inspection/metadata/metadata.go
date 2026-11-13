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

package inspectionmetadata

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/filter"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// MapContextKey is the context key to access the metadata map for the current inspection run.
var MapContextKey = typedmap.NewTypedKey[*typedmap.ReadonlyTypedMap]("khi.google.com/inspection/metadata-map")

// MetadataLabelsKey is a type-safe key for metadata label values.
type MetadataLabelsKey[T any] = typedmap.TypedKey[T]

// MetadataKey is a type-safe key for metadata.
type MetadataKey[T Metadata] = typedmap.TypedKey[T]

// Metadata represents serializable data with labels.
type Metadata interface {
	// ToSerializable converts the metadata into a JSON-serializable representation.
	ToSerializable() interface{}
	// Labels returns the read-only label map associated with the metadata.
	Labels() *typedmap.ReadonlyTypedMap
}

// NewMetadataLabelsKey creates a type-safe metadata key.
func NewMetadataLabelsKey[T any](key string) MetadataLabelsKey[T] {
	return typedmap.NewTypedKey[T](key)
}

// NewMetadataKey creates a type-safe metadata key.
func NewMetadataKey[T Metadata](key string) MetadataKey[T] {
	return typedmap.NewTypedKey[T](key)
}

// GetSerializableSubsetMapFromMetadataSet returns a map of serialized metadata entries matching the given label filter.
func GetSerializableSubsetMapFromMetadataSet[T any](metadataSet *typedmap.ReadonlyTypedMap, filter filter.TypedMapFilter[T]) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for _, key := range metadataSet.Keys() {
		metadata, found := typedmap.Get(metadataSet, NewMetadataLabelsKey[Metadata](key))
		if !found {
			return nil, fmt.Errorf("unreachable. expected metadata not found")
		}
		metadataLabel, found := typedmap.Get(metadata.Labels(), filter.KeyToFilter())
		if !filter.ShouldInclude(found, metadataLabel) {
			continue
		}
		result[key] = metadata.ToSerializable()
	}
	return result, nil
}
