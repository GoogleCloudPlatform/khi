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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestFlatStructStore_RoundTrip(t *testing.T) {
	testCases := []struct {
		name       string
		input      *pb.InternedStruct
		wantStruct *pb.InternedStruct
	}{
		{
			name: "all scalar value kinds",
			input: &pb.InternedStruct{
				Id:             proto.Uint32(1),
				FieldPathSetId: proto.Uint32(10),
				Values: []*pb.InternedValue{
					{Kind: &pb.InternedValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
					{Kind: &pb.InternedValue_BoolValue{BoolValue: true}},
					{Kind: &pb.InternedValue_BoolValue{BoolValue: false}},
					{Kind: &pb.InternedValue_Int64Value{Int64Value: -12345}},
					{Kind: &pb.InternedValue_DoubleValue{DoubleValue: 3.14159265}},
					{Kind: &pb.InternedValue_StringValue{StringValue: 42}},
					{Kind: &pb.InternedValue_StructId{StructId: 99}},
					{
						Kind: &pb.InternedValue_TimestampValue{
							TimestampValue: &timestamppb.Timestamp{
								Seconds: 1700000000,
								Nanos:   500000,
							},
						},
					},
				},
			},
			wantStruct: &pb.InternedStruct{
				Id:             proto.Uint32(1),
				FieldPathSetId: proto.Uint32(10),
				Values: []*pb.InternedValue{
					{Kind: &pb.InternedValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
					{Kind: &pb.InternedValue_BoolValue{BoolValue: true}},
					{Kind: &pb.InternedValue_BoolValue{BoolValue: false}},
					{Kind: &pb.InternedValue_Int64Value{Int64Value: -12345}},
					{Kind: &pb.InternedValue_DoubleValue{DoubleValue: 3.14159265}},
					{Kind: &pb.InternedValue_StringValue{StringValue: 42}},
					{Kind: &pb.InternedValue_StructId{StructId: 99}},
					{
						Kind: &pb.InternedValue_TimestampValue{
							TimestampValue: &timestamppb.Timestamp{
								Seconds: 1700000000,
								Nanos:   500000,
							},
						},
					},
				},
			},
		},
		{
			name: "nested list value",
			input: &pb.InternedStruct{
				Id:             proto.Uint32(2),
				FieldPathSetId: proto.Uint32(20),
				Values: []*pb.InternedValue{
					{
						Kind: &pb.InternedValue_ListValue{
							ListValue: &pb.InternedListValue{
								Values: []*pb.InternedValue{
									{Kind: &pb.InternedValue_StringValue{StringValue: 101}},
									{Kind: &pb.InternedValue_Int64Value{Int64Value: 202}},
									{Kind: &pb.InternedValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
								},
							},
						},
					},
				},
			},
			wantStruct: &pb.InternedStruct{
				Id:             proto.Uint32(2),
				FieldPathSetId: proto.Uint32(20),
				Values: []*pb.InternedValue{
					{
						Kind: &pb.InternedValue_ListValue{
							ListValue: &pb.InternedListValue{
								Values: []*pb.InternedValue{
									{Kind: &pb.InternedValue_StringValue{StringValue: 101}},
									{Kind: &pb.InternedValue_Int64Value{Int64Value: 202}},
									{Kind: &pb.InternedValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "empty struct",
			input: &pb.InternedStruct{
				Id:             proto.Uint32(3),
				FieldPathSetId: proto.Uint32(30),
				Values:         nil,
			},
			wantStruct: &pb.InternedStruct{
				Id:             proto.Uint32(3),
				FieldPathSetId: proto.Uint32(30),
				Values:         []*pb.InternedValue{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewFlatStructStore()
			store.StoreProto(tc.input)

			if !store.Has(*tc.input.Id) {
				t.Fatalf("store.Has(%d) = false, want true", *tc.input.Id)
			}

			got := store.ResolveStruct(*tc.input.Id)
			if diff := cmp.Diff(tc.wantStruct, got, protocmp.Transform()); diff != "" {
				t.Errorf("ResolveStruct() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFlatStructStore_MetadataAndAccessors(t *testing.T) {
	testCases := []struct {
		name       string
		storeSetup func(s *FlatStructStore)
		queryID    uint32
		wantFound  bool
		wantFSID   uint32
		wantCount  uint32
	}{
		{
			name: "existing struct ID",
			storeSetup: func(s *FlatStructStore) {
				s.Store(5, 100, []*pb.InternedValue{
					{Kind: &pb.InternedValue_StringValue{StringValue: 1}},
					{Kind: &pb.InternedValue_StringValue{StringValue: 2}},
				})
			},
			queryID:   5,
			wantFound: true,
			wantFSID:  100,
			wantCount: 2,
		},
		{
			name: "non-existing struct ID",
			storeSetup: func(s *FlatStructStore) {
				s.Store(5, 100, nil)
			},
			queryID:   999,
			wantFound: false,
			wantFSID:  0,
			wantCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewFlatStructStore()
			tc.storeSetup(store)

			fsID, offset, count, ok := store.GetValueSpan(tc.queryID)
			if ok != tc.wantFound {
				t.Fatalf("GetValueSpan(%d) found = %v, want %v", tc.queryID, ok, tc.wantFound)
			}
			if ok {
				if fsID != tc.wantFSID {
					t.Errorf("fsID = %d, want %d", fsID, tc.wantFSID)
				}
				if count != tc.wantCount {
					t.Errorf("count = %d, want %d", count, tc.wantCount)
				}
				_ = offset
			}
		})
	}
}

func TestFlatStructStore_StoreProtoBatch(t *testing.T) {
	testCases := []struct {
		name        string
		inputs      []*pb.InternedStruct
		wantStructs map[uint32]*pb.InternedStruct
	}{
		{
			name: "multiple structs in a batch",
			inputs: []*pb.InternedStruct{
				{
					Id:             proto.Uint32(10),
					FieldPathSetId: proto.Uint32(1),
					Values: []*pb.InternedValue{
						{Kind: &pb.InternedValue_StringValue{StringValue: 100}},
					},
				},
				{
					Id:             proto.Uint32(20),
					FieldPathSetId: proto.Uint32(2),
					Values: []*pb.InternedValue{
						{Kind: &pb.InternedValue_Int64Value{Int64Value: 42}},
					},
				},
			},
			wantStructs: map[uint32]*pb.InternedStruct{
				10: {
					Id:             proto.Uint32(10),
					FieldPathSetId: proto.Uint32(1),
					Values: []*pb.InternedValue{
						{Kind: &pb.InternedValue_StringValue{StringValue: 100}},
					},
				},
				20: {
					Id:             proto.Uint32(20),
					FieldPathSetId: proto.Uint32(2),
					Values: []*pb.InternedValue{
						{Kind: &pb.InternedValue_Int64Value{Int64Value: 42}},
					},
				},
			},
		},
		{
			name: "single struct with id 0",
			inputs: []*pb.InternedStruct{
				{
					Id:             proto.Uint32(0),
					FieldPathSetId: proto.Uint32(1),
					Values: []*pb.InternedValue{
						{Kind: &pb.InternedValue_StringValue{StringValue: 50}},
					},
				},
			},
			wantStructs: map[uint32]*pb.InternedStruct{
				0: {
					Id:             proto.Uint32(0),
					FieldPathSetId: proto.Uint32(1),
					Values: []*pb.InternedValue{
						{Kind: &pb.InternedValue_StringValue{StringValue: 50}},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewFlatStructStore()
			store.StoreProtoBatch(tc.inputs)

			for id, want := range tc.wantStructs {
				got := store.ResolveStruct(id)
				if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
					t.Errorf("ResolveStruct(%d) mismatch (-want +got):\n%s", id, diff)
				}
			}
		})
	}
}

func TestFlatStructStore_NestedStructWithID(t *testing.T) {
	testCases := []struct {
		name       string
		input      *pb.InternedStruct
		wantParent *pb.InternedStruct
		wantChild  *pb.InternedStruct
	}{
		{
			name: "inline nested struct with ID does not deadlock",
			input: &pb.InternedStruct{
				Id:             proto.Uint32(1),
				FieldPathSetId: proto.Uint32(10),
				Values: []*pb.InternedValue{
					{
						Kind: &pb.InternedValue_StructValue{
							StructValue: &pb.InternedStruct{
								Id:             proto.Uint32(2),
								FieldPathSetId: proto.Uint32(20),
								Values: []*pb.InternedValue{
									{Kind: &pb.InternedValue_Int64Value{Int64Value: 999}},
								},
							},
						},
					},
				},
			},
			wantParent: &pb.InternedStruct{
				Id:             proto.Uint32(1),
				FieldPathSetId: proto.Uint32(10),
				Values: []*pb.InternedValue{
					{Kind: &pb.InternedValue_StructId{StructId: 2}},
				},
			},
			wantChild: &pb.InternedStruct{
				Id:             proto.Uint32(2),
				FieldPathSetId: proto.Uint32(20),
				Values: []*pb.InternedValue{
					{Kind: &pb.InternedValue_Int64Value{Int64Value: 999}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewFlatStructStore()
			store.StoreProto(tc.input)

			gotParent := store.ResolveStruct(*tc.wantParent.Id)
			if diff := cmp.Diff(tc.wantParent, gotParent, protocmp.Transform()); diff != "" {
				t.Errorf("ResolveStruct(%d) parent mismatch (-want +got):\n%s", *tc.wantParent.Id, diff)
			}

			gotChild := store.ResolveStruct(*tc.wantChild.Id)
			if diff := cmp.Diff(tc.wantChild, gotChild, protocmp.Transform()); diff != "" {
				t.Errorf("ResolveStruct(%d) child mismatch (-want +got):\n%s", *tc.wantChild.Id, diff)
			}
		})
	}
}

func TestFlatStructStore_StoreFromNodes(t *testing.T) {
	testTime := time.Date(2026, 4, 20, 3, 0, 0, 500000, time.UTC)

	testCases := []struct {
		name       string
		id         uint32
		fsID       uint32
		nodes      func(pool *InternPool) []structured.Node
		wantStruct func(pool *InternPool) *pb.InternedStruct
	}{
		{
			name: "all scalar node types",
			id:   1,
			fsID: 10,
			nodes: func(pool *InternPool) []structured.Node {
				return []structured.Node{
					structured.NewStandardScalarNode[any](nil),
					structured.NewStandardScalarNode(true),
					structured.NewStandardScalarNode(false),
					structured.NewStandardScalarNode(12345),
					structured.NewStandardScalarNode(3.14159),
					structured.NewStandardScalarNode("hello world"),
					structured.NewStandardScalarNode(testTime),
				}
			},
			wantStruct: func(pool *InternPool) *pb.InternedStruct {
				return &pb.InternedStruct{
					Id:             proto.Uint32(1),
					FieldPathSetId: proto.Uint32(10),
					Values: []*pb.InternedValue{
						{Kind: &pb.InternedValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
						{Kind: &pb.InternedValue_BoolValue{BoolValue: true}},
						{Kind: &pb.InternedValue_BoolValue{BoolValue: false}},
						{Kind: &pb.InternedValue_Int64Value{Int64Value: 12345}},
						{Kind: &pb.InternedValue_DoubleValue{DoubleValue: 3.14159}},
						{Kind: &pb.InternedValue_StringValue{StringValue: pool.InternString("hello world").id}},
						{
							Kind: &pb.InternedValue_TimestampValue{
								TimestampValue: &timestamppb.Timestamp{
									Seconds: testTime.Unix(),
									Nanos:   int32(testTime.Nanosecond()),
								},
							},
						},
					},
				}
			},
		},
		{
			name: "nested sequence node with mixed values",
			id:   2,
			fsID: 20,
			nodes: func(pool *InternPool) []structured.Node {
				return []structured.Node{
					structured.NewStandardSequenceNode([]structured.Node{
						structured.NewStandardScalarNode(100),
						structured.NewStandardScalarNode("item"),
						structured.NewStandardScalarNode[any](nil),
					}),
				}
			},
			wantStruct: func(pool *InternPool) *pb.InternedStruct {
				return &pb.InternedStruct{
					Id:             proto.Uint32(2),
					FieldPathSetId: proto.Uint32(20),
					Values: []*pb.InternedValue{
						{
							Kind: &pb.InternedValue_ListValue{
								ListValue: &pb.InternedListValue{
									Values: []*pb.InternedValue{
										{Kind: &pb.InternedValue_Int64Value{Int64Value: 100}},
										{Kind: &pb.InternedValue_StringValue{StringValue: pool.InternString("item").id}},
										{Kind: &pb.InternedValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
									},
								},
							},
						},
					},
				}
			},
		},
		{
			name: "sequence containing nested map without deadlock",
			id:   3,
			fsID: 30,
			nodes: func(pool *InternPool) []structured.Node {
				childMap := structured.NewStandardMap(
					[]string{"val"},
					[]structured.Node{structured.NewStandardScalarNode(999)},
				)
				return []structured.Node{
					structured.NewStandardSequenceNode([]structured.Node{
						childMap,
					}),
				}
			},
			wantStruct: func(pool *InternPool) *pb.InternedStruct {
				childMap := structured.NewStandardMap(
					[]string{"val"},
					[]structured.Node{structured.NewStandardScalarNode(999)},
				)
				childRef, _ := ToInternedStruct(childMap, pool)
				return &pb.InternedStruct{
					Id:             proto.Uint32(3),
					FieldPathSetId: proto.Uint32(30),
					Values: []*pb.InternedValue{
						{
							Kind: &pb.InternedValue_ListValue{
								ListValue: &pb.InternedListValue{
									Values: []*pb.InternedValue{
										{Kind: &pb.InternedValue_StructId{StructId: childRef.id}},
									},
								},
							},
						},
					},
				}
			},
		},
		{
			name: "empty map node",
			id:   4,
			fsID: 40,
			nodes: func(pool *InternPool) []structured.Node {
				emptyMap := structured.NewStandardMap(nil, nil)
				return []structured.Node{
					emptyMap,
				}
			},
			wantStruct: func(pool *InternPool) *pb.InternedStruct {
				emptyMap := structured.NewStandardMap(nil, nil)
				childRef, _ := ToInternedStruct(emptyMap, pool)
				return &pb.InternedStruct{
					Id:             proto.Uint32(4),
					FieldPathSetId: proto.Uint32(40),
					Values: []*pb.InternedValue{
						{Kind: &pb.InternedValue_StructId{StructId: childRef.id}},
					},
				}
			},
		},
		{
			name: "empty struct without nodes",
			id:   5,
			fsID: 50,
			nodes: func(pool *InternPool) []structured.Node {
				return nil
			},
			wantStruct: func(pool *InternPool) *pb.InternedStruct {
				return &pb.InternedStruct{
					Id:             proto.Uint32(5),
					FieldPathSetId: proto.Uint32(50),
					Values:         []*pb.InternedValue{},
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idGen := id.NewGenerator()
			pool := NewTestInternPool(idGen)
			store := NewFlatStructStore()

			nodes := tc.nodes(pool)
			if err := store.StoreFromNodes(tc.id, tc.fsID, nodes, pool); err != nil {
				t.Fatalf("StoreFromNodes() unexpected error: %v", err)
			}

			if !store.Has(tc.id) {
				t.Fatalf("store.Has(%d) = false, want true", tc.id)
			}

			got := store.ResolveStruct(tc.id)
			want := tc.wantStruct(pool)
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("ResolveStruct(%d) mismatch (-want +got):\n%s", tc.id, diff)
			}
		})
	}
}
