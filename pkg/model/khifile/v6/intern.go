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
	"encoding/binary"
	"iter"
	"math"
	"sort"
	"strings"
	"sync"
	"unsafe"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	pbv6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
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
func (r *InternStringRef) ToProto() *pbv6.InternString {
	id := r.id
	val := r.Resolve()
	return &pbv6.InternString{
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
func (r *FieldPathSetRef) ToProto() *pbv6.InternFieldPathSet {
	id := r.id
	names := r.pool.resolveFieldSetFromID(r.id)
	return &pbv6.InternFieldPathSet{
		Id:                 &id,
		FieldPathStringIds: names,
	}
}

// InternStructRef represents a reference to an interned struct.
// This struct holds a reference to the pool and the ID of the struct.
type InternStructRef struct {
	pool *InternPool
	id   uint32
}

// ID returns the interned struct ID.
func (r *InternStructRef) ID() uint32 {
	return r.id
}

// Resolve returns the underlying pb.InternedStruct protobuf message.
func (r *InternStructRef) Resolve() *pb.InternedStruct {
	return r.pool.resolveStructFromID(r.id)
}

// ToProto converts InternStructRef to its proto representation.
func (r *InternStructRef) ToProto() *pb.InternedStruct {
	return r.Resolve()
}

// InternPool manages interning of strings, field path sets, and structs to reduce memory usage.
// It uses sync.Map for concurrent access and relies on IDGenerator for generating IDs.
type InternPool struct {
	idGen   *IDGenerator
	strToID sync.Map // map[string]uint32
	idToStr sync.Map // map[uint32]string

	fieldSetToID sync.Map // map[string]uint32 (key is byte representation of []uint32)
	idToFieldSet sync.Map // map[uint32][]uint32

	structToID sync.Map // map[string]uint32
	idToStruct sync.Map // map[uint32]*pb.InternedStruct
}

// NewInternPool creates a new InternPool with the given IDGenerator.
func NewInternPool(idGen *IDGenerator) *InternPool {
	return &InternPool{
		idGen: idGen,
	}
}

// NewInternPoolFromChunk creates an InternPool pre-populated with entries from an InterningPoolChunk.
func NewInternPoolFromChunk(chunk *pbv6.InterningPoolChunk) *InternPool {
	pool := NewInternPool(nil)
	pool.IngestChunk(chunk)
	return pool
}

// IngestChunk adds all strings, field path sets, and structs from an InterningPoolChunk into the pool.
func (p *InternPool) IngestChunk(chunk *pbv6.InterningPoolChunk) {
	if chunk == nil {
		return
	}
	for _, str := range chunk.Strings {
		if str.Id != nil && str.Value != nil {
			p.idToStr.Store(*str.Id, *str.Value)
			p.strToID.Store(*str.Value, *str.Id)
		}
	}
	for _, fs := range chunk.FieldPathSets {
		if fs.Id != nil {
			p.idToFieldSet.Store(*fs.Id, fs.FieldPathStringIds)
			p.fieldSetToID.Store(fieldSetKey(fs.FieldPathStringIds), *fs.Id)
		}
	}
	for _, s := range chunk.Structs {
		if s.Id != nil {
			p.idToStruct.Store(*s.Id, s)
			if s.FieldPathSetId != nil {
				p.structToID.Store(structKey(*s.FieldPathSetId, s.Values), *s.Id)
			}
		}
	}
}

// InternString returns a InternStringRef for the given string.
// If the string is not already interned, it assigns a new ID from IDGenerator and stores it.
func (p *InternPool) InternString(value string) *InternStringRef {
	if id, ok := p.strToID.Load(value); ok {
		return &InternStringRef{pool: p, id: id.(uint32)}
	}

	// Call ToValidUTF8 for every calls are costly and majority of value are expected not to contain invalid utf8, so check it after the first lookup.
	value = strings.ToValidUTF8(value, "\uFFFD")
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

// ResolveStringFromID returns the string corresponding to the given ID.
// It returns an empty string if the ID is not found.
func (p *InternPool) ResolveStringFromID(id uint32) string {
	return p.resolveStringFromID(id)
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

// InternStruct returns an InternStructRef for the given fieldPathSetID and values.
// It checks if an identical struct is already interned, and if not, assigns a new ID and stores it.
func (p *InternPool) InternStruct(fieldPathSetID uint32, values []*pb.InternedValue) *InternStructRef {
	key := structKey(fieldPathSetID, values)
	if id, ok := p.structToID.Load(key); ok {
		return &InternStructRef{pool: p, id: id.(uint32)}
	}

	id := p.idGen.New(IDStruct)
	pbStruct := &pb.InternedStruct{
		Id:             &id,
		FieldPathSetId: &fieldPathSetID,
		Values:         values,
	}

	p.idToStruct.Store(id, pbStruct)
	actual, loaded := p.structToID.LoadOrStore(key, id)
	if loaded {
		p.idToStruct.Store(id, (*pb.InternedStruct)(nil))
		return &InternStructRef{pool: p, id: actual.(uint32)}
	}

	return &InternStructRef{pool: p, id: id}
}

// ResolveStructFromID returns the InternedStruct corresponding to the given ID.
// It returns nil if the ID is not found.
func (p *InternPool) ResolveStructFromID(id uint32) *pb.InternedStruct {
	return p.resolveStructFromID(id)
}

// resolveStructFromID returns the InternedStruct corresponding to the given ID.
// It returns nil if the ID is not found.
func (p *InternPool) resolveStructFromID(id uint32) *pb.InternedStruct {
	if value, ok := p.idToStruct.Load(id); ok {
		return value.(*pb.InternedStruct)
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

// StructRefs returns an iterator that yields InternStructRefs in the pool, sorted by their ID.
func (p *InternPool) StructRefs() iter.Seq[*InternStructRef] {
	type entry struct {
		id uint32
	}
	var entries []entry

	p.structToID.Range(func(key, value any) bool {
		entries = append(entries, entry{
			id: value.(uint32),
		})
		return true
	})

	// Sort by ID.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].id < entries[j].id
	})

	return func(yield func(*InternStructRef) bool) {
		for _, e := range entries {
			if !yield(&InternStructRef{pool: p, id: e.id}) {
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

// structKey computes a deterministic binary key from fieldPathSetID and values.
func structKey(fieldPathSetID uint32, values []*pb.InternedValue) string {
	buf := make([]byte, 0, 4+len(values)*9)
	buf = binary.LittleEndian.AppendUint32(buf, fieldPathSetID)
	for _, v := range values {
		buf = appendInternedValueKey(buf, v)
	}
	return unsafe.String(unsafe.SliceData(buf), len(buf))
}

// appendInternedValueKey appends a deterministic byte representation of InternedValue.
func appendInternedValueKey(buf []byte, v *pb.InternedValue) []byte {
	if v == nil || v.Kind == nil {
		return append(buf, 0x00)
	}
	switch k := v.Kind.(type) {
	case *pb.InternedValue_NullValue:
		return append(buf, 0x01)
	case *pb.InternedValue_BoolValue:
		if k.BoolValue {
			return append(buf, 0x02, 0x01)
		}
		return append(buf, 0x02, 0x00)
	case *pb.InternedValue_Int64Value:
		buf = append(buf, 0x03)
		return binary.LittleEndian.AppendUint64(buf, uint64(k.Int64Value))
	case *pb.InternedValue_DoubleValue:
		buf = append(buf, 0x04)
		return binary.LittleEndian.AppendUint64(buf, math.Float64bits(k.DoubleValue))
	case *pb.InternedValue_StringValue:
		buf = append(buf, 0x05)
		return binary.LittleEndian.AppendUint32(buf, k.StringValue)
	case *pb.InternedValue_StructId:
		buf = append(buf, 0x06)
		return binary.LittleEndian.AppendUint32(buf, k.StructId)
	case *pb.InternedValue_StructValue:
		if k.StructValue != nil {
			if k.StructValue.GetId() > 0 {
				buf = append(buf, 0x06)
				return binary.LittleEndian.AppendUint32(buf, k.StructValue.GetId())
			}
			buf = append(buf, 0x09)
			buf = binary.LittleEndian.AppendUint32(buf, k.StructValue.GetFieldPathSetId())
			for _, elem := range k.StructValue.GetValues() {
				buf = appendInternedValueKey(buf, elem)
			}
			return buf
		}
		buf = append(buf, 0x06)
		return binary.LittleEndian.AppendUint32(buf, 0)
	case *pb.InternedValue_TimestampValue:
		buf = append(buf, 0x07)
		buf = binary.LittleEndian.AppendUint64(buf, uint64(k.TimestampValue.GetSeconds()))
		return binary.LittleEndian.AppendUint32(buf, uint32(k.TimestampValue.GetNanos()))
	case *pb.InternedValue_ListValue:
		buf = append(buf, 0x08)
		listVals := k.ListValue.GetValues()
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(listVals)))
		for _, elem := range listVals {
			buf = appendInternedValueKey(buf, elem)
		}
		return buf
	default:
		return append(buf, 0xFF)
	}
}
