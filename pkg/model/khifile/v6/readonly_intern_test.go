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
					{Id: proto.Uint32(ServerStringIDBase + 1), Value: proto.String("server-str-1")},
					{Id: proto.Uint32(ServerStringIDBase + 5), Value: proto.String("server-str-5")},
				},
			},
			queryStrID:   ServerStringIDBase + 5,
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
