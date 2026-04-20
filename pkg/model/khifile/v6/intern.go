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
	"sort"
	"sync"
	"unsafe"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

// InternStringRef represents a reference to an interned string.
// This struct holds a reference to the pool and the ID of the string.
type InternStringRef struct {
	pool *InternPool
	id   uint32
}

// Resolve returns the original string value.
// It delegates to the pool to resolve the string from the stored ID.
func (r *InternStringRef) Resolve() string {
	return r.pool.resolveStringFromID(r.id)
}

// ToProto converts InternStringRef to its proto representation.
func (r *InternStringRef) ToProto() *pb.InternString {
	id := r.id
	val := r.Resolve()
	return &pb.InternString{
		Id:    &id,
		Value: &val,
	}
}

// FieldPathSetRef represents a reference to an interned field path set.
// This struct holds a reference to the pool and the ID of the field path set.
type FieldPathSetRef struct {
	pool *InternPool
	id   uint32
}

// Resolve returns the original list of strings in the set.
// It delegates to the pool to resolve the field path set and then resolves each string ID.
func (r *FieldPathSetRef) Resolve() []string {
	ids := r.pool.resolveFieldSetFromID(r.id)
	res := make([]string, len(ids))
	for i, id := range ids {
		res[i] = r.pool.resolveStringFromID(id)
	}
	return res
}

// ToProto converts FieldPathSetRef to its proto representation.
func (r *FieldPathSetRef) ToProto() *pb.InternFieldPathSet {
	id := r.id
	names := r.pool.resolveFieldSetFromID(r.id)
	return &pb.InternFieldPathSet{
		Id:         &id,
		FieldNames: names,
	}
}

// InternPool manages interning of strings and field path sets to reduce memory usage.
// It uses sync.Map for concurrent access and relies on IDGenerator for generating IDs.
type InternPool struct {
	idGen   *IDGenerator
	strToID sync.Map // map[string]uint32
	idToStr sync.Map // map[uint32]string

	fieldSetToID sync.Map // map[string]uint32 (key is byte representation of []uint32)
	idToFieldSet sync.Map // map[uint32][]uint32
}

// NewInternPool creates a new InternPool with the given IDGenerator.
func NewInternPool(idGen *IDGenerator) *InternPool {
	return &InternPool{
		idGen: idGen,
	}
}

// InternString returns a InternStringRef for the given string.
// If the string is not already interned, it assigns a new ID from IDGenerator and stores it.
func (p *InternPool) InternString(value string) *InternStringRef {
	if id, ok := p.strToID.Load(value); ok {
		return &InternStringRef{pool: p, id: id.(uint32)}
	}

	id := p.idGen.New(IDString)
	p.idToStr.Store(id, value)

	actual, loaded := p.strToID.LoadOrStore(value, id)
	if loaded {
		p.idToStr.Store(id, "")
		return &InternStringRef{pool: p, id: actual.(uint32)}
	}

	return &InternStringRef{pool: p, id: id}
}

// resolveStringFromID returns the string corresponding to the given ID.
// It returns an empty string if the ID is not found.
func (p *InternPool) resolveStringFromID(id uint32) string {
	if value, ok := p.idToStr.Load(id); ok {
		return value.(string)
	}
	return ""
}

// InternFieldSet returns a FieldPathSetRef for the given list of strings.
// It first interns each string to get its ID, and then interns the resulting list of IDs.
// It uses unsafe string cast for fast lookup in fieldSetToID map without allocation.
func (p *InternPool) InternFieldSet(fieldNames []string) *FieldPathSetRef {
	ids := make([]uint32, len(fieldNames))
	for i, name := range fieldNames {
		ids[i] = p.InternString(name).id
	}

	// Zero-allocation lookup using unsafe string.
	keyLookup := fieldSetKey(ids)
	if id, ok := p.fieldSetToID.Load(keyLookup); ok {
		return &FieldPathSetRef{pool: p, id: id.(uint32)}
	}

	id := p.idGen.New(IDFieldSet)

	namesCopy := make([]uint32, len(ids))
	copy(namesCopy, ids)
	p.idToFieldSet.Store(id, namesCopy)
	keyStore := fieldSetKey(namesCopy)

	actual, loaded := p.fieldSetToID.LoadOrStore(keyStore, id)
	if loaded {
		p.idToFieldSet.Store(id, []uint32{})
		return &FieldPathSetRef{pool: p, id: actual.(uint32)}
	}

	return &FieldPathSetRef{pool: p, id: id}
}

// resolveFieldSetFromID returns the field path set corresponding to the given ID.
// It returns nil if the ID is not found.
func (p *InternPool) resolveFieldSetFromID(id uint32) []uint32 {
	if value, ok := p.idToFieldSet.Load(id); ok {
		return value.([]uint32)
	}
	return nil
}

// SortedStringRefs returns an iterator that yields InternStringRefs in the pool, sorted by their original string value.
func (p *InternPool) SortedStringRefs() iter.Seq[*InternStringRef] {
	type entry struct {
		val string
		id  uint32
	}
	var entries []entry

	p.strToID.Range(func(key, value any) bool {
		entries = append(entries, entry{
			val: key.(string),
			id:  value.(uint32),
		})
		return true
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].val < entries[j].val
	})

	return func(yield func(*InternStringRef) bool) {
		for _, e := range entries {
			if !yield(&InternStringRef{pool: p, id: e.id}) {
				return
			}
		}
	}
}

// FieldSetRefs returns an iterator that yields FieldPathSetRefs in the pool, sorted by their ID.
func (p *InternPool) FieldSetRefs() iter.Seq[*FieldPathSetRef] {
	type entry struct {
		id uint32
	}
	var entries []entry

	p.fieldSetToID.Range(func(key, value any) bool {
		entries = append(entries, entry{
			id: value.(uint32),
		})
		return true
	})

	// Sort by ID.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].id < entries[j].id
	})

	return func(yield func(*FieldPathSetRef) bool) {
		for _, e := range entries {
			if !yield(&FieldPathSetRef{pool: p, id: e.id}) {
				return
			}
		}
	}
}

// fieldSetKey casts a slice of uint32 to a string without copying.
// The returned string shares memory with the slice. It is safe to use as a map key
// ONLY if the slice is never modified.
func fieldSetKey(ids []uint32) string {
	if len(ids) == 0 {
		return ""
	}
	byteSlice := unsafe.Slice((*byte)(unsafe.Pointer(&ids[0])), len(ids)*4)
	return unsafe.String(&byteSlice[0], len(byteSlice))
}
