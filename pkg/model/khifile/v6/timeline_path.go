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

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"google.golang.org/protobuf/proto"
)

// TimelinePath represents a single node in the timeline hierarchy tree.
// Its identity is uniquely managed by TimelinePathPool, so pointer equality (==)
// guarantees logical equality.
type TimelinePath struct {
	// ID is the unique identifier for this timeline path.
	ID uint32
	// Parent is a pointer to the parent path. It is nil for root paths.
	Parent *TimelinePath
	// Name is a reference to the interned string name of this path segment.
	Name *InternStringRef
	// Type is the style definition for this timeline path segment.
	Type *pb.TimelineType
}

// PathSegment represents a single level of a timeline path to be appended or retrieved.
type PathSegment struct {
	// Name is the string representation of the segment.
	Name string
	// Type is the style definition for this segment.
	Type *pb.TimelineType
}

// timelinePathKey is used internally as a composite key to deduplicate TimelinePath instances.
type timelinePathKey struct {
	// parentID is the ID of the parent TimelinePath, or 0 for root paths.
	parentID uint32
	// nameID is the interned string ID.
	nameID uint32
	// typeID is the TimelineType ID.
	typeID uint32
}

// TimelinePathPool guarantees the uniqueness of TimelinePath instances.
// It is safe for concurrent use by multiple goroutines.
type TimelinePathPool struct {
	// idGen is used to generate unique IDs for new TimelinePath instances.
	idGen *IDGenerator
	// stringPool is used to intern string names of path segments to StringRefs.
	stringPool *InternPool
	// paths is a concurrent map caching timelinePathKey to *TimelinePath.
	paths sync.Map // map[timelinePathKey]*TimelinePath
	// idToPath is a concurrent map caching ID to *TimelinePath.
	idToPath sync.Map // map[uint32]*TimelinePath
}

// NewTimelinePathPool creates a new pool for deduplicating TimelinePath instances.
func NewTimelinePathPool(idGen *IDGenerator, sp *InternPool) *TimelinePathPool {
	return &TimelinePathPool{
		idGen:      idGen,
		stringPool: sp,
	}
}

// Get retrieves or creates a multi-level TimelinePath starting from the given parent.
// If parent is nil, it starts from the root.
// This method provides a convenient way to build deep paths in a single call.
func (p *TimelinePathPool) Get(parent *TimelinePath, segments ...PathSegment) *TimelinePath {
	current := parent
	for _, seg := range segments {
		current = p.getOrCreateSingle(current, seg.Name, seg.Type)
	}
	return current
}

// getOrCreateSingle handles the thread-safe retrieval or creation of a single TimelinePath segment.
func (p *TimelinePathPool) getOrCreateSingle(parent *TimelinePath, name string, t *pb.TimelineType) *TimelinePath {
	if t == nil || t.Id == nil {
		panic("TimelineType and its ID must not be nil. TimelineType must be correctly registered to assign the right ID.")
	}

	nameRef := p.stringPool.InternString(name)

	var parentID uint32
	if parent != nil {
		parentID = parent.ID
	}

	key := timelinePathKey{
		parentID: parentID,
		nameID:   nameRef.id,
		typeID:   *t.Id,
	}

	// 1. Fast Path: Load from cache.
	if val, ok := p.paths.Load(key); ok {
		return val.(*TimelinePath)
	}

	// 2. Create new path instance with a fresh ID.
	newID := p.idGen.New(IDTimelinePath)
	newPath := &TimelinePath{
		ID:     newID,
		Parent: parent,
		Name:   nameRef,
		Type:   proto.Clone(t).(*pb.TimelineType),
	}
	p.idToPath.Store(newID, newPath)

	// 3. Atomic store or retrieve if another goroutine won the race.
	actual, loaded := p.paths.LoadOrStore(key, newPath)
	if loaded {
		p.idToPath.Store(newID, (*TimelinePath)(nil))
	}
	return actual.(*TimelinePath)
}

// Paths returns an iterator over all TimelinePath instances present in the pool.
func (p *TimelinePathPool) Paths() iter.Seq[*TimelinePath] {
	return func(yield func(*TimelinePath) bool) {
		p.paths.Range(func(key, value any) bool {
			return yield(value.(*TimelinePath))
		})
	}
}

// ResolvePathFromID returns the TimelinePath corresponding to the given ID.
// It returns nil if the ID is not found or is an orphaned ID.
func (p *TimelinePathPool) ResolvePathFromID(id uint32) *TimelinePath {
	if value, ok := p.idToPath.Load(id); ok {
		return value.(*TimelinePath)
	}
	return nil
}
