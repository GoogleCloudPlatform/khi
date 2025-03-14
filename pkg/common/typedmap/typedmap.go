// Copyright 2025 Google LLC
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

package typedmap

import "sync"

// TypedKey represents a key with associated type information.
// The type parameter T indicates what type this key will retrieve.
type TypedKey[T any] struct {
	key string
}

// Key returns the string identifier of this key.
func (k TypedKey[T]) Key() string {
	return k.key
}

// NewTypedKey creates a new typed key with the given string identifier.
func NewTypedKey[T any](key string) TypedKey[T] {
	return TypedKey[T]{key: key}
}

// ReadableTypedMap is an interface for read operations on typed maps.
// Both TypedMap and ReadonlyTypedMap implement this interface.
type ReadableTypedMap interface {
	// load retrieves a value by its string key
	load(key string) (interface{}, bool)
}

// TypedMap is a thread-safe map with type-safe operations.
type TypedMap struct {
	container sync.Map
}

// ReadonlyTypedMap is a read-only view of a TypedMap.
type ReadonlyTypedMap struct {
	source *TypedMap
}

// load implements ReadableTypedMap interface.
func (m *TypedMap) load(key string) (interface{}, bool) {
	return m.container.Load(key)
}

// load implements ReadableTypedMap interface.
func (m *ReadonlyTypedMap) load(key string) (interface{}, bool) {
	return m.source.load(key)
}

// NewTypedMap creates a new empty TypedMap.
func NewTypedMap() *TypedMap {
	return &TypedMap{}
}

// Set stores a value with type safety.
// The key's type parameter must match the value's type.
func Set[T any](m *TypedMap, key TypedKey[T], value T) {
	m.container.Store(key.key, value)
}

// AsReadonly returns a read-only view of this map.
func (m *TypedMap) AsReadonly() *ReadonlyTypedMap {
	return &ReadonlyTypedMap{
		source: m,
	}
}

// Clone creates a new TypedMap with the same contents.
func (m *TypedMap) Clone() *TypedMap {
	cloned := NewTypedMap()

	m.container.Range(func(key, value interface{}) bool {
		cloned.container.Store(key, value)
		return true
	})

	return cloned
}

// Clone creates a new ReadonlyTypedMap with the same contents.
func (m *ReadonlyTypedMap) Clone() *ReadonlyTypedMap {
	cloned := m.source.Clone()
	return cloned.AsReadonly()
}

// Keys returns all keys in the map as a slice of strings
func (m *TypedMap) Keys() []string {
	var keys []string
	m.container.Range(func(key, value interface{}) bool {
		if strKey, ok := key.(string); ok {
			keys = append(keys, strKey)
		}
		return true
	})
	return keys
}

// Keys returns all keys in the map as a slice of strings
func (m *ReadonlyTypedMap) Keys() []string {
	return m.source.Keys()
}

// Get retrieves a value in a type-safe way.
// Works with both TypedMap and ReadonlyTypedMap.
func Get[T any](m ReadableTypedMap, key TypedKey[T]) (T, bool) {
	var zero T
	v, ok := m.load(key.key)
	if !ok {
		return zero, false
	}

	// Type assertion
	typed, ok := v.(T)
	if !ok {
		return zero, false
	}

	return typed, true
}

// GetOrDefault retrieves a value or returns the default if not found.
// Works with both TypedMap and ReadonlyTypedMap.
func GetOrDefault[T any](m ReadableTypedMap, key TypedKey[T], defaultValue T) T {
	v, ok := Get[T](m, key)
	if !ok {
		return defaultValue
	}
	return v
}
