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
	"fmt"
	"iter"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"google.golang.org/protobuf/proto"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	pbv6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
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

// ID returns the interned string ID.
func (r *InternStringRef) ID() uint32 {
	return r.id
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
// It uses sync.Map for forward key deduplication and slice-based index resolution for ID lookup.
// Newly interned elements are staged and flushed to the writer in chunks when reaching chunkSizeLimit.
type InternPool struct {
	parentPool *InternPool
	idGen      *id.Generator
	idNs       id.Namespace
	writer     *Writer
	chunkType  ChunkType

	strToID      sync.Map // map[string]uint32
	fieldSetToID sync.Map // map[string]uint32 (key is byte representation of []uint32)
	structToID   sync.Map // map[string]uint32

	idToStrMu sync.RWMutex
	idToStr   []string

	idToFieldSetMu sync.RWMutex
	idToFieldSet   [][]uint32

	idToStructMu sync.RWMutex
	idToStruct   []*pb.InternedStruct

	flushMu            sync.Mutex
	chunkSizeLimit     int
	currentBatchSize   int
	unflushedStrings   []*pbv6.InternString
	unflushedFieldSets []*pbv6.InternFieldPathSet
	unflushedStructs   []*pb.InternedStruct
}

var _ ReadonlyPool = (*InternPool)(nil)

// NewInternPool creates a new client InternPool with the given IDGenerator and Writer.
func NewInternPool(idGen *id.Generator, writer *Writer) *InternPool {
	return &InternPool{
		idGen:          idGen,
		idNs:           id.String,
		writer:         writer,
		chunkType:      ChunkTypeInternPool,
		chunkSizeLimit: DefaultChunkSizeLimit,
	}
}

// NewTestInternPool creates a client InternPool with MustNewTestWriter for testing purposes.
func NewTestInternPool(idGen *id.Generator) *InternPool {
	return NewInternPool(idGen, MustNewTestWriter())
}

// NewServerInternPool creates a new server-only InternPool delegating string lookup to parent when available.
func NewServerInternPool(parent *InternPool, idGen *id.Generator, writer *Writer) *InternPool {
	return &InternPool{
		parentPool:     parent,
		idGen:          idGen,
		idNs:           id.ServerString,
		writer:         writer,
		chunkType:      ChunkTypeServerInternPool,
		chunkSizeLimit: DefaultChunkSizeLimit,
	}
}

// NewTestServerInternPool creates a server InternPool with MustNewTestWriter for testing purposes.
func NewTestServerInternPool(parent *InternPool, idGen *id.Generator) *InternPool {
	return NewServerInternPool(parent, idGen, MustNewTestWriter())
}

// NewInternPoolFromChunk creates an InternPool pre-populated with entries from an InterningPoolChunk.
func NewInternPoolFromChunk(chunk *pbv6.InterningPoolChunk) *InternPool {
	pool := NewInternPool(nil, MustNewTestWriter())
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
			p.storeString(*str.Id, *str.Value)
			p.strToID.Store(*str.Value, *str.Id)
		}
	}
	for _, fs := range chunk.FieldPathSets {
		if fs.Id != nil {
			p.storeFieldSet(*fs.Id, fs.FieldPathStringIds)
			p.fieldSetToID.Store(fieldSetKey(fs.FieldPathStringIds), *fs.Id)
		}
	}
	for _, s := range chunk.Structs {
		if s.Id != nil {
			p.storeStruct(*s.Id, s)
			if s.FieldPathSetId != nil {
				p.structToID.Store(structKey(*s.FieldPathSetId, s.Values), *s.Id)
			}
		}
	}
}

func (p *InternPool) strIndex(idVal uint32) int {
	if p.idNs == id.ServerString {
		if idVal <= id.ServerStringIDBase {
			return -1
		}
		return int(idVal - id.ServerStringIDBase - 1)
	}
	if idVal == 0 {
		return -1
	}
	return int(idVal - 1)
}

func (p *InternPool) storeString(idVal uint32, value string) {
	idx := p.strIndex(idVal)
	if idx < 0 {
		return
	}
	p.idToStrMu.Lock()
	defer p.idToStrMu.Unlock()
	if idx >= len(p.idToStr) {
		p.idToStr = slices.Grow(p.idToStr, idx+1-len(p.idToStr))[:idx+1]
	}
	p.idToStr[idx] = value
}

func (p *InternPool) storeFieldSet(idVal uint32, fieldSet []uint32) {
	if idVal == 0 {
		return
	}
	idx := int(idVal - 1)
	p.idToFieldSetMu.Lock()
	defer p.idToFieldSetMu.Unlock()
	if idx >= len(p.idToFieldSet) {
		p.idToFieldSet = slices.Grow(p.idToFieldSet, idx+1-len(p.idToFieldSet))[:idx+1]
	}
	p.idToFieldSet[idx] = fieldSet
}

func (p *InternPool) storeStruct(idVal uint32, s *pb.InternedStruct) {
	if idVal == 0 {
		return
	}
	idx := int(idVal - 1)
	p.idToStructMu.Lock()
	defer p.idToStructMu.Unlock()
	if idx >= len(p.idToStruct) {
		p.idToStruct = slices.Grow(p.idToStruct, idx+1-len(p.idToStruct))[:idx+1]
	}
	p.idToStruct[idx] = s
}

// InternString returns a InternStringRef for the given string.
// If the string is not already interned, it assigns a new ID from IDGenerator and stores it.
func (p *InternPool) InternString(value string) *InternStringRef {
	if id, ok := p.strToID.Load(value); ok {
		return &InternStringRef{pool: p, id: id.(uint32)}
	}
	if p.parentPool != nil {
		if id, ok := p.parentPool.strToID.Load(value); ok {
			return &InternStringRef{pool: p.parentPool, id: id.(uint32)}
		}
	}

	// Call ToValidUTF8 for every calls are costly and majority of value are expected not to contain invalid utf8, so check it after the first lookup.
	value = strings.ToValidUTF8(value, "\uFFFD")
	if id, ok := p.strToID.Load(value); ok {
		return &InternStringRef{pool: p, id: id.(uint32)}
	}
	if p.parentPool != nil {
		if id, ok := p.parentPool.strToID.Load(value); ok {
			return &InternStringRef{pool: p.parentPool, id: id.(uint32)}
		}
	}

	id := p.idGen.New(p.idNs)
	// Clone string before storing in the pool to prevent pinning large underlying byte buffers (such as protojson buffers).
	cloned := strings.Clone(value)
	p.storeString(id, cloned)

	actual, loaded := p.strToID.LoadOrStore(cloned, id)
	if loaded {
		p.storeString(id, "")
		return &InternStringRef{pool: p, id: actual.(uint32)}
	}

	p.stageString(id, cloned)

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
	idx := p.strIndex(id)
	if idx >= 0 {
		p.idToStrMu.RLock()
		var val string
		if idx < len(p.idToStr) {
			val = p.idToStr[idx]
		}
		p.idToStrMu.RUnlock()
		if val != "" {
			return val
		}
	}
	if p.parentPool != nil {
		return p.parentPool.resolveStringFromID(id)
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
	if idVal, ok := p.fieldSetToID.Load(keyLookup); ok {
		return &FieldPathSetRef{pool: p, id: idVal.(uint32)}
	}

	newID := p.idGen.New(id.FieldSet)

	namesCopy := make([]uint32, len(ids))
	copy(namesCopy, ids)
	p.storeFieldSet(newID, namesCopy)
	keyStore := fieldSetKey(namesCopy)

	actual, loaded := p.fieldSetToID.LoadOrStore(keyStore, newID)
	if loaded {
		p.storeFieldSet(newID, nil)
		return &FieldPathSetRef{pool: p, id: actual.(uint32)}
	}

	p.stageFieldSet(newID, namesCopy)

	return &FieldPathSetRef{pool: p, id: newID}
}

// ResolveFieldSetFromID returns the field path set corresponding to the given ID.
// It returns nil if the ID is not found.
func (p *InternPool) ResolveFieldSetFromID(idVal uint32) []uint32 {
	return p.resolveFieldSetFromID(idVal)
}

// resolveFieldSetFromID returns the field path set corresponding to the given ID.
// It returns nil if the ID is not found.
func (p *InternPool) resolveFieldSetFromID(id uint32) []uint32 {
	if id == 0 {
		return nil
	}
	idx := int(id - 1)
	p.idToFieldSetMu.RLock()
	defer p.idToFieldSetMu.RUnlock()
	if idx < len(p.idToFieldSet) {
		return p.idToFieldSet[idx]
	}
	return nil
}

// InternStruct returns an InternStructRef for the given fieldPathSetID and values.
// It checks if an identical struct is already interned, and if not, assigns a new ID and stores it.
func (p *InternPool) InternStruct(fieldPathSetID uint32, values []*pb.InternedValue) *InternStructRef {
	key := structKey(fieldPathSetID, values)
	return p.internStructWithKey(fieldPathSetID, values, key)
}

func (p *InternPool) internStructWithKey(fieldPathSetID uint32, values []*pb.InternedValue, key string) *InternStructRef {
	if idVal, ok := p.structToID.Load(key); ok {
		return &InternStructRef{pool: p, id: idVal.(uint32)}
	}

	newID := p.idGen.New(id.Struct)
	s := &pb.InternedStruct{
		Id:             &newID,
		FieldPathSetId: &fieldPathSetID,
		Values:         values,
	}
	p.storeStruct(newID, s)

	actual, loaded := p.structToID.LoadOrStore(key, newID)
	if loaded {
		p.storeStruct(newID, nil)
		return &InternStructRef{pool: p, id: actual.(uint32)}
	}

	p.stageStruct(s)

	return &InternStructRef{pool: p, id: newID}
}

func (p *InternPool) stageString(idVal uint32, val string) {
	item := &pbv6.InternString{
		Id:    &idVal,
		Value: &val,
	}
	itemSize := proto.Size(item) + 8

	p.flushMu.Lock()
	defer p.flushMu.Unlock()

	p.unflushedStrings = append(p.unflushedStrings, item)
	p.currentBatchSize += itemSize

	if p.currentBatchSize >= p.chunkSizeLimit {
		_ = p.flushLocked()
	}
}

func (p *InternPool) stageFieldSet(idVal uint32, ids []uint32) {
	item := &pbv6.InternFieldPathSet{
		Id:                 &idVal,
		FieldPathStringIds: ids,
	}
	itemSize := proto.Size(item) + 8

	p.flushMu.Lock()
	defer p.flushMu.Unlock()

	p.unflushedFieldSets = append(p.unflushedFieldSets, item)
	p.currentBatchSize += itemSize

	if p.currentBatchSize >= p.chunkSizeLimit {
		_ = p.flushLocked()
	}
}

func (p *InternPool) stageStruct(s *pb.InternedStruct) {
	itemSize := proto.Size(s) + 8

	p.flushMu.Lock()
	defer p.flushMu.Unlock()

	p.unflushedStructs = append(p.unflushedStructs, s)
	p.currentBatchSize += itemSize

	if p.currentBatchSize >= p.chunkSizeLimit {
		_ = p.flushLocked()
	}
}

// SetChunkSizeLimit overrides the default chunk size limit for testing or performance tuning.
func (p *InternPool) SetChunkSizeLimit(limit int) {
	p.flushMu.Lock()
	defer p.flushMu.Unlock()
	p.chunkSizeLimit = limit
}

// Flush writes any pending unflushed intern pool items directly to the writer.
func (p *InternPool) Flush() error {
	p.flushMu.Lock()
	defer p.flushMu.Unlock()
	return p.flushLocked()
}

func (p *InternPool) flushLocked() error {
	if len(p.unflushedStrings) == 0 && len(p.unflushedFieldSets) == 0 && len(p.unflushedStructs) == 0 {
		return nil
	}

	chunk := &pbv6.InterningPoolChunk{
		Strings:       p.unflushedStrings,
		FieldPathSets: p.unflushedFieldSets,
		Structs:       p.unflushedStructs,
	}

	p.unflushedStrings = nil
	p.unflushedFieldSets = nil
	p.unflushedStructs = nil
	p.currentBatchSize = 0

	rawChunk, err := CompressChunk(p.chunkType, chunk)
	if err != nil {
		return fmt.Errorf("failed to compress intern pool chunk: %w", err)
	}

	if err := p.writer.WriteRawChunk(rawChunk); err != nil {
		return fmt.Errorf("failed to write intern pool raw chunk: %w", err)
	}

	return nil
}

// ResolveStructFromID returns the InternedStruct corresponding to the given ID.
// It returns nil if the ID is not found.
func (p *InternPool) ResolveStructFromID(id uint32) *pb.InternedStruct {
	return p.resolveStructFromID(id)
}

// resolveStructFromID returns the InternedStruct corresponding to the given ID.
// It returns nil if the ID is not found.
func (p *InternPool) resolveStructFromID(id uint32) *pb.InternedStruct {
	if id == 0 {
		return nil
	}
	idx := int(id - 1)
	p.idToStructMu.RLock()
	defer p.idToStructMu.RUnlock()
	if idx < len(p.idToStruct) {
		return p.idToStruct[idx]
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
	var ids []uint32
	p.structToID.Range(func(key, value any) bool {
		ids = append(ids, value.(uint32))
		return true
	})
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	return func(yield func(*InternStructRef) bool) {
		for _, id := range ids {
			if !yield(&InternStructRef{pool: p, id: id}) {
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
