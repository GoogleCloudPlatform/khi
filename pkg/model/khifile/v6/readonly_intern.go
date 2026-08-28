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
	"sync"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	pbv6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

// ReadonlyPool defines the common read-only lookup interface implemented by InternPool and ReadonlyInternPool.
type ReadonlyPool interface {
	ResolveStringFromID(id uint32) string
	ResolveFieldSetFromID(id uint32) []uint32
	ResolveStructFromID(id uint32) *pb.InternedStruct
}

// ReadonlyInternPool provides read-only, memory-optimized access to interned strings, field path sets, and structs.
// It stores strings and field sets directly in compact slice arrays indexed by ID, eliminating map hashing and
// pointer indirection, and delegates struct storage to a pointer-free FlatStructStore.
type ReadonlyInternPool struct {
	mu            sync.RWMutex
	clientStrings []string
	serverStrings []string
	fieldSets     [][]uint32
	flatStructs   *FlatStructStore
}

var _ ReadonlyPool = (*ReadonlyInternPool)(nil)

// NewReadonlyInternPool instantiates an empty ReadonlyInternPool.
func NewReadonlyInternPool() *ReadonlyInternPool {
	return &ReadonlyInternPool{
		flatStructs: NewFlatStructStore(),
	}
}

// IngestChunk adds strings, field path sets, and structs from an InterningPoolChunk into the pool.
// Client strings (< ServerStringIDBase) and server strings (>= ServerStringIDBase) are stored
// in separate slices to avoid multi-gigabyte sparse allocations from the 0x80000000 offset.
func (p *ReadonlyInternPool) IngestChunk(chunk *pbv6.InterningPoolChunk) {
	if chunk == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// 1. Strings
	for _, str := range chunk.Strings {
		if str == nil || str.Id == nil || str.Value == nil {
			continue
		}
		id := *str.Id
		if id >= ServerStringIDBase {
			sIdx := id - ServerStringIDBase
			p.serverStrings = ensureCapacity(p.serverStrings, sIdx)
			p.serverStrings[sIdx] = *str.Value
		} else {
			p.clientStrings = ensureCapacity(p.clientStrings, id)
			p.clientStrings[id] = *str.Value
		}
	}

	// 2. FieldPathSets
	for _, fs := range chunk.FieldPathSets {
		if fs == nil || fs.Id == nil {
			continue
		}
		p.fieldSets = ensureCapacity(p.fieldSets, *fs.Id)
		p.fieldSets[*fs.Id] = fs.FieldPathStringIds
	}

	// 3. Structs
	p.flatStructs.StoreProtoBatch(chunk.Structs)
}

// ensureCapacity expands the slice if needed so that maxIndex is a valid index.
func ensureCapacity[T any](slice []T, maxIndex uint32) []T {
	required := int(maxIndex) + 1
	if required <= len(slice) {
		return slice
	}
	newCap := len(slice) * 2
	if newCap < required {
		newCap = required
	}
	newSlice := make([]T, newCap)
	copy(newSlice, slice)
	return newSlice
}

// ResolveStringFromID returns the string corresponding to the given string ID, or empty string if not found.
func (p *ReadonlyInternPool) ResolveStringFromID(id uint32) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if id >= ServerStringIDBase {
		sIdx := id - ServerStringIDBase
		if int(sIdx) < len(p.serverStrings) {
			return p.serverStrings[sIdx]
		}
		return ""
	}
	if int(id) < len(p.clientStrings) {
		return p.clientStrings[id]
	}
	return ""
}

// ResolveFieldSetFromID returns the field path string IDs corresponding to the given field set ID.
func (p *ReadonlyInternPool) ResolveFieldSetFromID(id uint32) []uint32 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if int(id) < len(p.fieldSets) {
		return p.fieldSets[id]
	}
	return nil
}

// ResolveStructFromID resolves an interned struct by its ID, reconstructing its Protobuf representation.
func (p *ReadonlyInternPool) ResolveStructFromID(id uint32) *pb.InternedStruct {
	return p.flatStructs.ResolveStruct(id)
}

// FlatStructStore returns the underlying FlatStructStore.
func (p *ReadonlyInternPool) FlatStructStore() *FlatStructStore {
	return p.flatStructs
}

// HasStruct checks whether a struct with the given ID exists in the pool.
func (p *ReadonlyInternPool) HasStruct(id uint32) bool {
	return p.flatStructs.Has(id)
}

// AllStructIDs returns a slice of all stored struct IDs in the pool.
func (p *ReadonlyInternPool) AllStructIDs() []uint32 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.flatStructs.AllIDs()
}
