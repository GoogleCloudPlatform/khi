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
	"math"
	"sync"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FlatValueKind indicates the concrete type of a packed value in FlatStructStore.
type FlatValueKind uint8

const (
	// FlatValueKindNull represents a null value.
	FlatValueKindNull FlatValueKind = iota
	// FlatValueKindBool represents a boolean value.
	FlatValueKindBool
	// FlatValueKindInt64 represents an int64 value.
	FlatValueKindInt64
	// FlatValueKindDouble represents a double precision floating point value.
	FlatValueKindDouble
	// FlatValueKindStringID represents an interned string ID reference.
	FlatValueKindStringID
	// FlatValueKindStructID represents an interned struct ID reference.
	FlatValueKindStructID
	// FlatValueKindTimestamp represents a timestamp value (seconds and nanoseconds).
	FlatValueKindTimestamp
	// FlatValueKindList represents a repeated list of packed values.
	FlatValueKindList
	// FlatValueKindStructValue represents an uninterned inline nested struct.
	FlatValueKindStructValue
)

// FlatStructStore provides pointer-free, Structure-of-Arrays (SoA) storage for interned structs.
// All stored data is packed into primitive scalar slices without pointers, ensuring that the Go
// runtime allocates them in noscan memory blocks and completely skips them during GC sweeps.
type FlatStructStore struct {
	mu sync.RWMutex

	// Struct-level metadata indexed directly by struct ID.
	structFieldPathSetIDs []uint32
	structValueOffsets    []uint32
	structValueCounts     []uint32
	structPresent         []bool

	// Packed value data (SoA).
	valueKinds []uint8
	valueData  []uint64
	valueAux   []uint32

	count int
}

// NewFlatStructStore instantiates an empty FlatStructStore.
func NewFlatStructStore() *FlatStructStore {
	return &FlatStructStore{}
}

// ensureStructCapacity expands struct metadata slices to accommodate the specified struct ID.
func (s *FlatStructStore) ensureStructCapacity(id uint32) {
	required := int(id) + 1
	if required <= len(s.structPresent) {
		return
	}
	newCap := len(s.structPresent) * 2
	if newCap < required {
		newCap = required
	}
	newFieldPathSetIDs := make([]uint32, newCap)
	copy(newFieldPathSetIDs, s.structFieldPathSetIDs)
	s.structFieldPathSetIDs = newFieldPathSetIDs

	newValueOffsets := make([]uint32, newCap)
	copy(newValueOffsets, s.structValueOffsets)
	s.structValueOffsets = newValueOffsets

	newValueCounts := make([]uint32, newCap)
	copy(newValueCounts, s.structValueCounts)
	s.structValueCounts = newValueCounts

	newPresent := make([]bool, newCap)
	copy(newPresent, s.structPresent)
	s.structPresent = newPresent
}

func (s *FlatStructStore) reserveSlots(count uint32) uint32 {
	offset := uint32(len(s.valueKinds))
	targetLen := offset + count
	if targetLen > uint32(cap(s.valueKinds)) {
		newCap := uint32(cap(s.valueKinds)) * 2
		if newCap < targetLen {
			newCap = targetLen + 1024
		}
		newKinds := make([]uint8, targetLen, newCap)
		copy(newKinds, s.valueKinds)
		s.valueKinds = newKinds

		newData := make([]uint64, targetLen, newCap)
		copy(newData, s.valueData)
		s.valueData = newData

		newAux := make([]uint32, targetLen, newCap)
		copy(newAux, s.valueAux)
		s.valueAux = newAux
	} else {
		s.valueKinds = s.valueKinds[:targetLen]
		s.valueData = s.valueData[:targetLen]
		s.valueAux = s.valueAux[:targetLen]
	}
	return offset
}

// Store encodes and saves an interned struct by its ID, fieldPathSetID, and slice of values.
func (s *FlatStructStore) Store(id uint32, fieldPathSetID uint32, values []*pb.InternedValue) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storeLocked(id, fieldPathSetID, values)
}

func (s *FlatStructStore) storeLocked(id uint32, fieldPathSetID uint32, values []*pb.InternedValue) {
	s.ensureStructCapacity(id)
	if !s.structPresent[id] {
		s.structPresent[id] = true
		s.count++
	}

	s.structFieldPathSetIDs[id] = fieldPathSetID
	valCount := uint32(len(values))
	s.structValueCounts[id] = valCount

	if valCount == 0 {
		s.structValueOffsets[id] = 0
		return
	}

	startOffset := s.reserveSlots(valCount)
	s.structValueOffsets[id] = startOffset

	// Encode values into the reserved slots, appending any nested child lists at the end
	for i, v := range values {
		slot := startOffset + uint32(i)
		s.encodeValueAt(slot, v)
	}
}

// StoreProto extracts metadata from a pb.InternedStruct message and stores it flatly.
func (s *FlatStructStore) StoreProto(structObj *pb.InternedStruct) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storeProtoLocked(structObj)
}

func (s *FlatStructStore) storeProtoLocked(structObj *pb.InternedStruct) {
	if structObj == nil || structObj.Id == nil {
		return
	}
	fieldSetID := uint32(0)
	if structObj.FieldPathSetId != nil {
		fieldSetID = *structObj.FieldPathSetId
	}
	s.storeLocked(*structObj.Id, fieldSetID, structObj.Values)
}

// StoreProtoBatch extracts metadata from a slice of pb.InternedStruct messages and stores them in a single batch.
// It acquires the write lock once and pre-allocates slice capacities, significantly reducing lock contention
// and re-allocation overhead when processing chunks concurrently.
func (s *FlatStructStore) StoreProtoBatch(structObjs []*pb.InternedStruct) {
	if len(structObjs) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var maxID uint32
	var hasStructs bool
	for _, obj := range structObjs {
		if obj != nil && obj.Id != nil {
			if *obj.Id > maxID {
				maxID = *obj.Id
			}
			hasStructs = true
		}
	}
	if hasStructs {
		s.ensureStructCapacity(maxID)
	}

	for _, obj := range structObjs {
		if obj == nil || obj.Id == nil {
			continue
		}
		id := *obj.Id
		if !s.structPresent[id] {
			s.structPresent[id] = true
			s.count++
		}

		fieldSetID := uint32(0)
		if obj.FieldPathSetId != nil {
			fieldSetID = *obj.FieldPathSetId
		}
		s.structFieldPathSetIDs[id] = fieldSetID

		valCount := uint32(len(obj.Values))
		s.structValueCounts[id] = valCount

		if valCount == 0 {
			s.structValueOffsets[id] = 0
			continue
		}

		startOffset := s.reserveSlots(valCount)
		s.structValueOffsets[id] = startOffset

		for i, v := range obj.Values {
			slot := startOffset + uint32(i)
			s.encodeValueAt(slot, v)
		}
	}
}

// encodeValueAt serializes an InternedValue into the designated slot.
func (s *FlatStructStore) encodeValueAt(slot uint32, v *pb.InternedValue) {
	if v == nil || v.Kind == nil {
		s.valueKinds[slot] = uint8(FlatValueKindNull)
		s.valueData[slot] = 0
		s.valueAux[slot] = 0
		return
	}

	switch k := v.Kind.(type) {
	case *pb.InternedValue_NullValue:
		s.valueKinds[slot] = uint8(FlatValueKindNull)
		s.valueData[slot] = 0
		s.valueAux[slot] = 0
	case *pb.InternedValue_BoolValue:
		s.valueKinds[slot] = uint8(FlatValueKindBool)
		if k.BoolValue {
			s.valueData[slot] = 1
		} else {
			s.valueData[slot] = 0
		}
		s.valueAux[slot] = 0
	case *pb.InternedValue_Int64Value:
		s.valueKinds[slot] = uint8(FlatValueKindInt64)
		s.valueData[slot] = uint64(k.Int64Value)
		s.valueAux[slot] = 0
	case *pb.InternedValue_DoubleValue:
		s.valueKinds[slot] = uint8(FlatValueKindDouble)
		s.valueData[slot] = math.Float64bits(k.DoubleValue)
		s.valueAux[slot] = 0
	case *pb.InternedValue_StringValue:
		s.valueKinds[slot] = uint8(FlatValueKindStringID)
		s.valueData[slot] = uint64(k.StringValue)
		s.valueAux[slot] = 0
	case *pb.InternedValue_StructId:
		s.valueKinds[slot] = uint8(FlatValueKindStructID)
		s.valueData[slot] = uint64(k.StructId)
		s.valueAux[slot] = 0
	case *pb.InternedValue_TimestampValue:
		s.valueKinds[slot] = uint8(FlatValueKindTimestamp)
		if k.TimestampValue != nil {
			s.valueData[slot] = uint64(k.TimestampValue.Seconds)
			s.valueAux[slot] = uint32(k.TimestampValue.Nanos)
		} else {
			s.valueData[slot] = 0
			s.valueAux[slot] = 0
		}
	case *pb.InternedValue_ListValue:
		list := k.ListValue.GetValues()
		listCount := uint32(len(list))
		s.valueKinds[slot] = uint8(FlatValueKindList)
		s.valueAux[slot] = listCount

		if listCount == 0 {
			s.valueData[slot] = 0
			return
		}

		listOffset := s.reserveSlots(listCount)
		s.valueData[slot] = uint64(listOffset)

		// Encode list items
		for i, item := range list {
			itemSlot := listOffset + uint32(i)
			s.encodeValueAt(itemSlot, item)
		}
	case *pb.InternedValue_StructValue:
		if k.StructValue != nil {
			if k.StructValue.Id != nil {
				s.storeProtoLocked(k.StructValue)
				s.valueKinds[slot] = uint8(FlatValueKindStructID)
				s.valueData[slot] = uint64(*k.StructValue.Id)
				s.valueAux[slot] = 0
			} else {
				nestedValues := k.StructValue.GetValues()
				nestedCount := uint32(len(nestedValues))
				fieldSetID := uint32(0)
				if k.StructValue.FieldPathSetId != nil {
					fieldSetID = *k.StructValue.FieldPathSetId
				}
				s.valueKinds[slot] = uint8(FlatValueKindStructValue)
				s.valueAux[slot] = fieldSetID

				if nestedCount == 0 {
					s.valueData[slot] = 0
					return
				}

				nestedOffset := s.reserveSlots(nestedCount)
				s.valueData[slot] = (uint64(nestedCount) << 32) | uint64(nestedOffset)

				for i, item := range nestedValues {
					s.encodeValueAt(nestedOffset+uint32(i), item)
				}
			}
		} else {
			s.valueKinds[slot] = uint8(FlatValueKindNull)
			s.valueData[slot] = 0
			s.valueAux[slot] = 0
		}
	default:
		s.valueKinds[slot] = uint8(FlatValueKindNull)
		s.valueData[slot] = 0
		s.valueAux[slot] = 0
	}
}

// Has reports whether the specified struct ID exists in the store.
func (s *FlatStructStore) Has(id uint32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int(id) < len(s.structPresent) && s.structPresent[id]
}

// Len returns the total count of stored structs.
func (s *FlatStructStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

// AllIDs returns a slice of all stored struct IDs in ascending order.
func (s *FlatStructStore) AllIDs() []uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]uint32, 0, s.count)
	for id, ok := range s.structPresent {
		if ok {
			ids = append(ids, uint32(id))
		}
	}
	return ids
}

// GetFieldPathSetID returns the FieldPathSet ID associated with the struct ID.
func (s *FlatStructStore) GetFieldPathSetID(id uint32) (uint32, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if int(id) >= len(s.structPresent) || !s.structPresent[id] {
		return 0, false
	}
	return s.structFieldPathSetIDs[id], true
}

// GetValueSpan returns the fieldPathSetID, start offset, and value count for a struct.
func (s *FlatStructStore) GetValueSpan(id uint32) (fieldPathSetID uint32, offset uint32, count uint32, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if int(id) >= len(s.structPresent) || !s.structPresent[id] {
		return 0, 0, 0, false
	}
	return s.structFieldPathSetIDs[id], s.structValueOffsets[id], s.structValueCounts[id], true
}

// GetRawValueAt returns the kind, data, and aux fields at the given packed index.
func (s *FlatStructStore) GetRawValueAt(index uint32) (kind FlatValueKind, data uint64, aux uint32) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if int(index) >= len(s.valueKinds) {
		return FlatValueKindNull, 0, 0
	}
	return FlatValueKind(s.valueKinds[index]), s.valueData[index], s.valueAux[index]
}

// ResolveStruct reconstructs a *pb.InternedStruct on-demand from the flat primitive storage.
// It returns nil if the struct ID is not found.
func (s *FlatStructStore) ResolveStruct(id uint32) *pb.InternedStruct {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if int(id) >= len(s.structPresent) || !s.structPresent[id] {
		return nil
	}

	fieldSetID := s.structFieldPathSetIDs[id]
	offset := s.structValueOffsets[id]
	count := s.structValueCounts[id]

	values := make([]*pb.InternedValue, count)
	for i := uint32(0); i < count; i++ {
		values[i] = s.decodeValueAt(offset + i)
	}

	return &pb.InternedStruct{
		Id:             proto.Uint32(id),
		FieldPathSetId: proto.Uint32(fieldSetID),
		Values:         values,
	}
}

// decodeValueAt unpacks a single InternedValue from the internal SoA arrays.
func (s *FlatStructStore) decodeValueAt(slot uint32) *pb.InternedValue {
	if int(slot) >= len(s.valueKinds) {
		return &pb.InternedValue{
			Kind: &pb.InternedValue_NullValue{
				NullValue: structpb.NullValue_NULL_VALUE,
			},
		}
	}

	kind := FlatValueKind(s.valueKinds[slot])
	data := s.valueData[slot]
	aux := s.valueAux[slot]

	switch kind {
	case FlatValueKindNull:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_NullValue{
				NullValue: structpb.NullValue_NULL_VALUE,
			},
		}
	case FlatValueKindBool:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_BoolValue{
				BoolValue: data != 0,
			},
		}
	case FlatValueKindInt64:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_Int64Value{
				Int64Value: int64(data),
			},
		}
	case FlatValueKindDouble:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_DoubleValue{
				DoubleValue: math.Float64frombits(data),
			},
		}
	case FlatValueKindStringID:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_StringValue{
				StringValue: uint32(data),
			},
		}
	case FlatValueKindStructID:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_StructId{
				StructId: uint32(data),
			},
		}
	case FlatValueKindTimestamp:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_TimestampValue{
				TimestampValue: &timestamppb.Timestamp{
					Seconds: int64(data),
					Nanos:   int32(aux),
				},
			},
		}
	case FlatValueKindList:
		listOffset := uint32(data)
		listCount := aux
		items := make([]*pb.InternedValue, listCount)
		for i := uint32(0); i < listCount; i++ {
			items[i] = s.decodeValueAt(listOffset + i)
		}
		return &pb.InternedValue{
			Kind: &pb.InternedValue_ListValue{
				ListValue: &pb.InternedListValue{
					Values: items,
				},
			},
		}
	case FlatValueKindStructValue:
		fieldSetID := aux
		nestedCount := uint32(data >> 32)
		nestedOffset := uint32(data & 0xFFFFFFFF)
		items := make([]*pb.InternedValue, nestedCount)
		for i := uint32(0); i < nestedCount; i++ {
			items[i] = s.decodeValueAt(nestedOffset + i)
		}
		return &pb.InternedValue{
			Kind: &pb.InternedValue_StructValue{
				StructValue: &pb.InternedStruct{
					FieldPathSetId: proto.Uint32(fieldSetID),
					Values:         items,
				},
			},
		}
	default:
		return &pb.InternedValue{
			Kind: &pb.InternedValue_NullValue{
				NullValue: structpb.NullValue_NULL_VALUE,
			},
		}
	}
}
