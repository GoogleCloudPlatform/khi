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

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	pbv6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestReadonlyInternPool_IngestAndResolve(t *testing.T) {
	testCases := []struct {
		name         string
		chunk        *pbv6.InterningPoolChunk
		queryStrID   uint32
		wantStr      string
		queryFsID    uint32
		wantFieldSet []uint32
		queryStruct  uint32
		wantStruct   *pb.InternedStruct
	}{
		{
			name: "basic ingestion and resolution",
			chunk: &pbv6.InterningPoolChunk{
				Strings: []*pbv6.InternString{
					{Id: proto.Uint32(5), Value: proto.String("foo")},
					{Id: proto.Uint32(10), Value: proto.String("bar")},
				},
				FieldPathSets: []*pbv6.InternFieldPathSet{
					{Id: proto.Uint32(2), FieldPathStringIds: []uint32{5, 10}},
				},
				Structs: []*pb.InternedStruct{
					{
						Id:             proto.Uint32(1),
						FieldPathSetId: proto.Uint32(2),
						Values: []*pb.InternedValue{
							{Kind: &pb.InternedValue_StringValue{StringValue: 5}},
							{Kind: &pb.InternedValue_TimestampValue{TimestampValue: timestamppb.New(timestamppb.Now().AsTime())}},
						},
					},
				},
			},
			queryStrID:   5,
			wantStr:      "foo",
			queryFsID:    2,
			wantFieldSet: []uint32{5, 10},
			queryStruct:  1,
			wantStruct: &pb.InternedStruct{
				Id:             proto.Uint32(1),
				FieldPathSetId: proto.Uint32(2),
			},
		},
		{
			name: "server-only string resolution with base offset",
			chunk: &pbv6.InterningPoolChunk{
				Strings: []*pbv6.InternString{
					{Id: proto.Uint32(id.ServerStringIDBase + 1), Value: proto.String("server-str-1")},
					{Id: proto.Uint32(id.ServerStringIDBase + 5), Value: proto.String("server-str-5")},
				},
			},
			queryStrID:   id.ServerStringIDBase + 5,
			wantStr:      "server-str-5",
			queryFsID:    999,
			wantFieldSet: nil,
			queryStruct:  999,
			wantStruct:   nil,
		},
		{
			name:         "querying non-existent IDs returns empty/nil",
			chunk:        &pbv6.InterningPoolChunk{},
			queryStrID:   999,
			wantStr:      "",
			queryFsID:    999,
			wantFieldSet: nil,
			queryStruct:  999,
			wantStruct:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := NewReadonlyInternPool()
			pool.IngestChunk(tc.chunk)

			gotStr := pool.ResolveStringFromID(tc.queryStrID)
			if gotStr != tc.wantStr {
				t.Errorf("ResolveStringFromID(%d) = %q, want %q", tc.queryStrID, gotStr, tc.wantStr)
			}

			gotFs := pool.ResolveFieldSetFromID(tc.queryFsID)
			if diff := cmp.Diff(tc.wantFieldSet, gotFs); diff != "" {
				t.Errorf("ResolveFieldSetFromID(%d) mismatch (-want +got):\n%s", tc.queryFsID, diff)
			}

			if tc.wantStruct != nil {
				gotStruct := pool.ResolveStructFromID(tc.queryStruct)
				if gotStruct == nil {
					t.Fatalf("ResolveStructFromID(%d) = nil, want non-nil", tc.queryStruct)
				}
				if *gotStruct.Id != *tc.wantStruct.Id || *gotStruct.FieldPathSetId != *tc.wantStruct.FieldPathSetId {
					t.Errorf("ResolveStructFromID(%d) ID/FieldPathSet mismatch", tc.queryStruct)
				}
			} else if pool.HasStruct(tc.queryStruct) {
				t.Errorf("HasStruct(%d) = true, want false", tc.queryStruct)
			}
		})
	}
}

func TestReadonlyInternPool_ClientAndServerStrings(t *testing.T) {
	type snapshot struct {
		Client []string
		Server []string
	}
	testCases := []struct {
		name  string
		chunk *pbv6.InterningPoolChunk
		want  snapshot
	}{
		{
			name:  "empty pool returns nil slices",
			chunk: &pbv6.InterningPoolChunk{},
			want:  snapshot{},
		},
		{
			name: "returns copied client and server string slices",
			chunk: &pbv6.InterningPoolChunk{
				Strings: []*pbv6.InternString{
					{Id: proto.Uint32(1), Value: proto.String("client-a")},
					{Id: proto.Uint32(3), Value: proto.String("client-b")},
					{Id: proto.Uint32(id.ServerStringIDBase), Value: proto.String("server-0")},
					{Id: proto.Uint32(id.ServerStringIDBase + 1), Value: proto.String("server-1")},
				},
			},
			want: snapshot{
				Client: []string{"", "client-a", "", "client-b"},
				Server: []string{"server-0", "server-1"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := NewReadonlyInternPool()
			pool.IngestChunk(tc.chunk)

			got := snapshot{
				Client: pool.ClientStrings(),
				Server: pool.ServerStrings(),
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ClientStrings/ServerStrings mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadonlyInternPool_ResolveStructStringField(t *testing.T) {
	type stringFieldResult struct {
		Value string
		OK    bool
	}

	pool := NewReadonlyInternPool()
	pool.IngestChunk(&pbv6.InterningPoolChunk{
		Strings: []*pbv6.InternString{
			{Id: proto.Uint32(1), Value: proto.String("insertId")},
			{Id: proto.Uint32(2), Value: proto.String("ins-123")},
			{Id: proto.Uint32(3), Value: proto.String("metadata\x00uid")},
			{Id: proto.Uint32(4), Value: proto.String("pod-uid-1")},
			{Id: proto.Uint32(5), Value: proto.String("empty")},
			{Id: proto.Uint32(6), Value: proto.String("")},
			{Id: proto.Uint32(7), Value: proto.String("count")},
			{Id: proto.Uint32(8), Value: proto.String("nestedRef")},
			{Id: proto.Uint32(9), Value: proto.String("nestedInline")},
			{Id: proto.Uint32(10), Value: proto.String("name")},
			{Id: proto.Uint32(11), Value: proto.String("annotations\x00kubectl.kubernetes.io/restartedAt")},
			{Id: proto.Uint32(12), Value: proto.String("2026-01-01T00:00:00Z")},
			{Id: proto.Uint32(id.ServerStringIDBase + 1), Value: proto.String("server-val")},
		},
		FieldPathSets: []*pbv6.InternFieldPathSet{
			{Id: proto.Uint32(1), FieldPathStringIds: []uint32{1, 3, 5, 7, 8, 9, 11}},
			{Id: proto.Uint32(2), FieldPathStringIds: []uint32{10}},
		},
		Structs: []*pb.InternedStruct{
			{
				Id: proto.Uint32(2), FieldPathSetId: proto.Uint32(2),
				Values: []*pb.InternedValue{{Kind: &pb.InternedValue_StringValue{StringValue: id.ServerStringIDBase + 1}}},
			},
			{
				Id: proto.Uint32(1), FieldPathSetId: proto.Uint32(1),
				Values: []*pb.InternedValue{
					{Kind: &pb.InternedValue_StringValue{StringValue: 2}},
					{Kind: &pb.InternedValue_StringValue{StringValue: 4}},
					{Kind: &pb.InternedValue_StringValue{StringValue: 6}},
					{Kind: &pb.InternedValue_Int64Value{Int64Value: 42}},
					{Kind: &pb.InternedValue_StructId{StructId: 2}},
					{Kind: &pb.InternedValue_StructValue{StructValue: &pb.InternedStruct{
						FieldPathSetId: proto.Uint32(2),
						Values:         []*pb.InternedValue{{Kind: &pb.InternedValue_StringValue{StringValue: 4}}},
					}}},
					{Kind: &pb.InternedValue_StringValue{StringValue: 12}},
				},
			},
		},
	})

	testCases := []struct {
		name      string
		structID  uint32
		fieldPath []string
		want      stringFieldResult
	}{
		{
			name:      "resolves top-level string field",
			structID:  1,
			fieldPath: []string{"insertId"},
			want:      stringFieldResult{Value: "ins-123", OK: true},
		},
		{
			name:      "resolves flattened nested string field",
			structID:  1,
			fieldPath: []string{"metadata", "uid"},
			want:      stringFieldResult{Value: "pod-uid-1", OK: true},
		},
		{
			name:      "resolves segment containing literal dots",
			structID:  1,
			fieldPath: []string{"annotations", "kubectl.kubernetes.io/restartedAt"},
			want:      stringFieldResult{Value: "2026-01-01T00:00:00Z", OK: true},
		},
		{
			name:      "resolves nested StructId with server string value",
			structID:  1,
			fieldPath: []string{"nestedRef", "name"},
			want:      stringFieldResult{Value: "server-val", OK: true},
		},
		{
			name:      "resolves inline StructValue",
			structID:  1,
			fieldPath: []string{"nestedInline", "name"},
			want:      stringFieldResult{Value: "pod-uid-1", OK: true},
		},
		{
			name:      "distinguishes empty string from missing field",
			structID:  1,
			fieldPath: []string{"empty"},
			want:      stringFieldResult{Value: "", OK: true},
		},
		{
			name:      "returns false for non-string scalar",
			structID:  1,
			fieldPath: []string{"count"},
			want:      stringFieldResult{Value: "", OK: false},
		},
		{
			name:      "returns false when querying map prefix without leaf",
			structID:  1,
			fieldPath: []string{"metadata"},
			want:      stringFieldResult{Value: "", OK: false},
		},
		{
			name:      "returns false when traversing through scalar field",
			structID:  1,
			fieldPath: []string{"count", "sub"},
			want:      stringFieldResult{Value: "", OK: false},
		},
		{
			name:      "returns false for missing field",
			structID:  1,
			fieldPath: []string{"unknown"},
			want:      stringFieldResult{Value: "", OK: false},
		},
		{
			name:      "returns false for zero structID",
			structID:  0,
			fieldPath: []string{"insertId"},
			want:      stringFieldResult{Value: "", OK: false},
		},
		{
			name:      "returns false for non-existent structID",
			structID:  999,
			fieldPath: []string{"insertId"},
			want:      stringFieldResult{Value: "", OK: false},
		},
		{
			name:      "returns false for empty fieldPath",
			structID:  1,
			fieldPath: nil,
			want:      stringFieldResult{Value: "", OK: false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val, ok := pool.ResolveStructStringField(tc.structID, tc.fieldPath...)
			got := stringFieldResult{Value: val, OK: ok}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ResolveStructStringField(%d, %v) mismatch (-want +got):\n%s", tc.structID, tc.fieldPath, diff)
			}
		})
	}
}
