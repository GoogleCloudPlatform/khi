// Copyright 2026 Google LLC
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

package khifilev6

import (
	"iter"
	"sync"
)

// TimelineRegistry manages the registration and retrieval of TimelineBuilders mapped by their unique TimelinePath.
// It is safe for concurrent use by multiple goroutines.
type TimelineRegistry struct {
	// idGen is used to generate unique IDs for new TimelineBuilder (TimelineItems) instances.
	idGen *IDGenerator
	// internPool is passed to new TimelineBuilder instances for string interning.
	internPool *InternPool
	// builders is a concurrent map caching *TimelinePath to its *TimelineBuilder.
	builders sync.Map // map[*TimelinePath]*TimelineBuilder
	// idToBuilder is a concurrent map caching ID to *TimelineBuilder.
	idToBuilder sync.Map // map[uint32]*TimelineBuilder
}

// NewTimelineRegistry creates a new registry for managing TimelineBuilders.
func NewTimelineRegistry(idGen *IDGenerator, sp *InternPool) *TimelineRegistry {
	return &TimelineRegistry{
		idGen:      idGen,
		internPool: sp,
	}
}

// GetBuilder retrieves the TimelineBuilder associated with the given path.
// Note: The returned TimelineBuilder is a shared instance. Callers must use its Mutex when mutating its state.
func (r *TimelineRegistry) GetBuilder(path *TimelinePath) *TimelineBuilder {
	// 1. Fast Path: Load from cache.
	if val, ok := r.builders.Load(path); ok {
		return val.(*TimelineBuilder)
	}

	// 2. Create new builder instance.
	newID := r.idGen.New(IDTimelineItems)
	newBuilder := &TimelineBuilder{
		Path:            path,
		TimelineItemsID: newID,
		internPool:      r.internPool,
	}
	r.idToBuilder.Store(newID, newBuilder)

	// 3. Atomic store or retrieve if another goroutine won the race.
	actual, loaded := r.builders.LoadOrStore(path, newBuilder)
	if loaded {
		r.idToBuilder.Store(newID, (*TimelineBuilder)(nil))
	}
	return actual.(*TimelineBuilder)
}

// SetAlias configures aliasPath to be an alias of targetPath.
// Any subsequent request for a builder for aliasPath will return the builder for targetPath.
// Panics if aliasPath already has a builder attached (i.e., GetBuilder was already called on it).
func (r *TimelineRegistry) SetAlias(aliasPath, targetPath *TimelinePath) {
	if aliasPath == targetPath {
		return
	}

	targetBuilder := r.GetBuilder(targetPath)

	// Register the alias link.
	// Fail-fast: if aliasPath already has a builder attached, it's too late to alias it.
	actual, loaded := r.builders.LoadOrStore(aliasPath, targetBuilder)
	if loaded && actual != targetBuilder {
		panic("TimelineRegistry: Cannot set alias on a path that already has a builder attached")
	}
}

// Builders returns an iterator over all unique TimelineBuilders present in the registry.
func (r *TimelineRegistry) Builders() iter.Seq[*TimelineBuilder] {
	return func(yield func(*TimelineBuilder) bool) {
		seenBuilders := make(map[*TimelineBuilder]bool)
		r.builders.Range(func(key, value any) bool {
			builder := value.(*TimelineBuilder)
			if seenBuilders[builder] {
				return true
			}
			seenBuilders[builder] = true
			return yield(builder)
		})
	}
}

// GetBuilderIfExists returns the TimelineBuilder attached to the path,
// and a boolean indicating whether the builder exists.
func (r *TimelineRegistry) GetBuilderIfExists(path *TimelinePath) (*TimelineBuilder, bool) {
	if val, ok := r.builders.Load(path); ok {
		return val.(*TimelineBuilder), true
	}
	return nil, false
}

// ResolveBuilderFromID returns the TimelineBuilder corresponding to the given ID.
// It returns nil if the ID is not found or is an orphaned ID.
func (r *TimelineRegistry) ResolveBuilderFromID(id uint32) *TimelineBuilder {
	if value, ok := r.idToBuilder.Load(id); ok {
		return value.(*TimelineBuilder)
	}
	return nil
}
