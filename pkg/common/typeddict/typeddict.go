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

// Package typeddict provides a generic, concurrent, type-safe dictionary.
// It acts as a wrapper around sync.Map to enforce that all values are of a specific type,
// preventing runtime type errors.
package typeddict

import (
	"fmt"
	"sync"
)

// TypedDict is a generic, concurrent, type-safe dictionary.
// It internally uses a sync.Map for the container and another for key-specific mutexes,
// ensuring thread-safe operations on individual keys.
type TypedDict[T any] struct {
	container sync.Map
	lockers   sync.Map
}

// NewTypedDict creates and returns a new, empty TypedDict.
func NewTypedDict[T any]() *TypedDict[T] {
	return &TypedDict[T]{
		container: sync.Map{},
		lockers:   sync.Map{},
	}
}

func (m *TypedDict[T]) lockKey(key string) func() {
	mutexAny, _ := m.lockers.LoadOrStore(key, &sync.Mutex{})
	mutex := mutexAny.(*sync.Mutex)
	mutex.Lock()
	return func() {
		mutex.Unlock()
	}
}

// Keys returns all keys in the map as a slice of strings
func (m *TypedDict[T]) Keys() []string {
	keys := []string{}
	m.container.Range(func(key, _ any) bool {
		keys = append(keys, key.(string))
		return true
	})
	return keys
}

// Get retrieves a value from the dictionary for the given key.
// It returns the value and true if the key is found, otherwise it returns the zero value of type T and false.
// It panics if the stored value is not of the expected type T, which indicates a bug or misuse.
func Get[T any](dict *TypedDict[T], key string) (T, bool) {
	value, ok := dict.container.Load(key)
	if ok {
		typed, ok := value.(T)
		if !ok {
			panic(fmt.Sprintf("expected dict value at %s is convertible to %T, but got %T.\nThis error rarely happens unless users forcibly casting the key types or a bug in KHI.\n Please report a bug. https://github.com/GoogleCloudPlatform/khi/issues", key, *new(T), value))
		}
		return typed, true
	}
	return *new(T), false
}

// GetOrDefault retrieves a value for a key, or returns the provided default value if the key is not found.
func GetOrDefault[T any](m *TypedDict[T], key string, defaultValue T) T {
	v, ok := Get(m, key)
	if !ok {
		return defaultValue
	}
	return v
}

// GetOrSetFunc retrieves the value for a key, or if the key is not found,
// it generates a new value using the provided genFunc, stores it, and returns the new value.
// The operation is atomic for each key.
func GetOrSetFunc[T any](m *TypedDict[T], key string, genFunc func() T) T {
	defer m.lockKey(key)()
	v, found := Get(m, key)
	if !found {
		v = genFunc()
		m.container.Store(key, v)
	}
	return v
}

// Set stores a key-value pair in the dictionary.
// If the key already exists, its value is overwritten.
func Set[T any](m *TypedDict[T], key string, value T) {
	defer m.lockKey(key)()
	m.container.Store(key, value)
}

// Delete removes the key-value pair associated with the given key from the dictionary.
func Delete[T any](m *TypedDict[T], key string) {
	defer m.lockKey(key)()
	m.container.Delete(key)
}
