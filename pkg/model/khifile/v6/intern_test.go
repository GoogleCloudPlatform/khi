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

	khifile "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestInternPool_Intern(t *testing.T) {
	idGen := NewIDGenerator()
	pool := NewInternPool(idGen)

	testCases := []struct {
		name   string
		input  string
		wantID uint32
	}{
		{
			name:   "first string",
			input:  "foo",
			wantID: 1,
		},
		{
			name:   "second string",
			input:  "bar",
			wantID: 2,
		},
		{
			name:   "duplicate string",
			input:  "foo",
			wantID: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := pool.InternString(tc.input)
			if ref.id != tc.wantID {
				t.Errorf("Intern(%q) ID = %d, want %d", tc.input, ref.id, tc.wantID)
			}
			got := ref.Resolve()
			if diff := cmp.Diff(tc.input, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInternPool_Intern_InvalidUTF8(t *testing.T) {
	idGen := NewIDGenerator()
	pool := NewInternPool(idGen)

	testCases := []struct {
		name        string
		input       string
		wantID      uint32
		wantResolve string
	}{
		{
			name:        "invalid utf8",
			input:       "foo\xffbar",
			wantID:      1,
			wantResolve: "foo\uFFFDbar",
		},
		{
			name:        "duplicate invalid utf8",
			input:       "foo\xffbar",
			wantID:      1,
			wantResolve: "foo\uFFFDbar",
		},
		{
			name:        "valid replacement string",
			input:       "foo\uFFFDbar",
			wantID:      1,
			wantResolve: "foo\uFFFDbar",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := pool.InternString(tc.input)
			if ref.id != tc.wantID {
				t.Errorf("Intern(%q) ID = %d, want %d", tc.input, ref.id, tc.wantID)
			}
			got := ref.Resolve()
			if diff := cmp.Diff(tc.wantResolve, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInternPool_ResolveStringFromID(t *testing.T) {
	idGen := NewIDGenerator()
	pool := NewInternPool(idGen)
	ref1 := pool.InternString("foo")
	ref2 := pool.InternString("bar")

	testCases := []struct {
		name string
		id   uint32
		want string
	}{
		{
			name: "resolve foo",
			id:   ref1.id,
			want: "foo",
		},
		{
			name: "resolve bar",
			id:   ref2.id,
			want: "bar",
		},
		{
			name: "invalid ID 0",
			id:   0,
			want: "",
		},
		{
			name: "invalid ID large",
			id:   999,
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := pool.resolveStringFromID(tc.id)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("resolveStringFromID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInternStringRef_ToProto(t *testing.T) {
	idGen := NewIDGenerator()
	pool := NewInternPool(idGen)
	ref := pool.InternString("foo")

	got := ref.ToProto()

	wantId := ref.id
	wantVal := "foo"
	want := &pb.InternString{
		Id:    &wantId,
		Value: &wantVal,
	}

	if got.GetId() != want.GetId() {
		t.Errorf("ToProto().GetId() = %d, want %d", got.GetId(), want.GetId())
	}
	if got.GetValue() != want.GetValue() {
		t.Errorf("ToProto().GetValue() = %q, want %q", got.GetValue(), want.GetValue())
	}
}

func TestInternPool_SortedRefs(t *testing.T) {
	idGen := NewIDGenerator()
	pool := NewInternPool(idGen)
	pool.InternString("c")
	pool.InternString("a")
	pool.InternString("b")

	var refs []*InternStringRef
	for ref := range pool.SortedStringRefs() {
		refs = append(refs, ref)
	}

	testCases := []struct {
		name string
		idx  int
		want string
	}{
		{
			name: "first is a",
			idx:  0,
			want: "a",
		},
		{
			name: "second is b",
			idx:  1,
			want: "b",
		},
		{
			name: "third is c",
			idx:  2,
			want: "c",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.idx >= len(refs) {
				t.Fatalf("Index %d out of range", tc.idx)
			}
			got := refs[tc.idx].Resolve()
			if got != tc.want {
				t.Errorf("refs[%d].Resolve() = %q, want %q", tc.idx, got, tc.want)
			}
		})
	}
}

func TestInternPool_InternFieldSet(t *testing.T) {
	idGen := NewIDGenerator()
	pool := NewInternPool(idGen)

	testCases := []struct {
		name   string
		input  []string
		want   []string
		wantID uint32
	}{
		{
			name:   "first set",
			input:  []string{"a", "b"},
			want:   []string{"a", "b"},
			wantID: 1,
		},
		{
			name:   "second set",
			input:  []string{"c"},
			want:   []string{"c"},
			wantID: 2,
		},
		{
			name:   "duplicate set",
			input:  []string{"a", "b"},
			want:   []string{"a", "b"},
			wantID: 1,
		},
		{
			name:   "invalid utf8 set",
			input:  []string{"foo\xffbar"},
			want:   []string{"foo\uFFFDbar"},
			wantID: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := pool.InternFieldSet(tc.input)
			if ref.id != tc.wantID {
				t.Errorf("InternFieldSet(%v) ID = %d, want %d", tc.input, ref.id, tc.wantID)
			}
			got := ref.Resolve()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInternPool_FieldSetRefs(t *testing.T) {
	idGen := NewIDGenerator()
	pool := NewInternPool(idGen)

	pool.InternFieldSet([]string{"a", "b"})
	pool.InternFieldSet([]string{"c"})
	pool.InternFieldSet([]string{"a", "c"})

	var refs []*FieldPathSetRef
	for ref := range pool.FieldSetRefs() {
		refs = append(refs, ref)
	}

	testCases := []struct {
		name string
		idx  int
		want []string
	}{
		{
			name: "first is [a, b]",
			idx:  0,
			want: []string{"a", "b"},
		},
		{
			name: "second is [c]",
			idx:  1,
			want: []string{"c"},
		},
		{
			name: "third is [a, c]",
			idx:  2,
			want: []string{"a", "c"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.idx >= len(refs) {
				t.Fatalf("Index %d out of range", tc.idx)
			}
			got := refs[tc.idx].Resolve()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("refs[%d].Resolve() mismatch (-want +got):\n%s", tc.idx, diff)
			}
		})
	}
}

func TestInternPool_InternStruct(t *testing.T) {
	idGen := NewIDGenerator()
	pool := NewInternPool(idGen)

	fieldSetID1 := pool.InternFieldSet([]string{"foo", "bar"}).id
	fieldSetID2 := pool.InternFieldSet([]string{"baz"}).id

	str1ID := pool.InternString("val1").id
	str2ID := pool.InternString("val2").id

	testCases := []struct {
		name       string
		fieldSetID uint32
		values     []*khifile.InternedValue
		wantID     uint32
	}{
		{
			name:       "first struct",
			fieldSetID: fieldSetID1,
			values: []*khifile.InternedValue{
				{Kind: &khifile.InternedValue_StringValue{StringValue: str1ID}},
				{Kind: &khifile.InternedValue_Int64Value{Int64Value: 100}},
			},
			wantID: 1,
		},
		{
			name:       "second struct with different values",
			fieldSetID: fieldSetID1,
			values: []*khifile.InternedValue{
				{Kind: &khifile.InternedValue_StringValue{StringValue: str2ID}},
				{Kind: &khifile.InternedValue_Int64Value{Int64Value: 100}},
			},
			wantID: 2,
		},
		{
			name:       "duplicate first struct (should deduplicate)",
			fieldSetID: fieldSetID1,
			values: []*khifile.InternedValue{
				{Kind: &khifile.InternedValue_StringValue{StringValue: str1ID}},
				{Kind: &khifile.InternedValue_Int64Value{Int64Value: 100}},
			},
			wantID: 1,
		},
		{
			name:       "third struct with different fieldSetID",
			fieldSetID: fieldSetID2,
			values: []*khifile.InternedValue{
				{Kind: &khifile.InternedValue_StringValue{StringValue: str1ID}},
			},
			wantID: 3,
		},
		{
			name:       "struct with nested struct_id",
			fieldSetID: fieldSetID2,
			values: []*khifile.InternedValue{
				{Kind: &khifile.InternedValue_StructId{StructId: 1}},
			},
			wantID: 4,
		},
		{
			name:       "struct with uninterned nested struct_value",
			fieldSetID: fieldSetID2,
			values: []*khifile.InternedValue{
				{
					Kind: &khifile.InternedValue_StructValue{
						StructValue: &khifile.InternedStruct{
							FieldPathSetId: &fieldSetID1,
							Values: []*khifile.InternedValue{
								{Kind: &khifile.InternedValue_StringValue{StringValue: str1ID}},
								{Kind: &khifile.InternedValue_Int64Value{Int64Value: 100}},
							},
						},
					},
				},
			},
			wantID: 5,
		},
		{
			name:       "duplicate struct with uninterned nested struct_value",
			fieldSetID: fieldSetID2,
			values: []*khifile.InternedValue{
				{
					Kind: &khifile.InternedValue_StructValue{
						StructValue: &khifile.InternedStruct{
							FieldPathSetId: &fieldSetID1,
							Values: []*khifile.InternedValue{
								{Kind: &khifile.InternedValue_StringValue{StringValue: str1ID}},
								{Kind: &khifile.InternedValue_Int64Value{Int64Value: 100}},
							},
						},
					},
				},
			},
			wantID: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := pool.InternStruct(tc.fieldSetID, tc.values)
			if ref.id != tc.wantID {
				t.Errorf("InternStruct() ID = %d, want %d", ref.id, tc.wantID)
			}
			resolved := ref.Resolve()
			if resolved.GetId() != tc.wantID {
				t.Errorf("Resolve().GetId() = %d, want %d", resolved.GetId(), tc.wantID)
			}
			if resolved.GetFieldPathSetId() != tc.fieldSetID {
				t.Errorf("Resolve().GetFieldPathSetId() = %d, want %d", resolved.GetFieldPathSetId(), tc.fieldSetID)
			}
			if diff := cmp.Diff(tc.values, resolved.GetValues(), protocmp.Transform()); diff != "" {
				t.Errorf("Resolve().GetValues() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInternPool_StructRefs(t *testing.T) {
	idGen := NewIDGenerator()
	pool := NewInternPool(idGen)

	fsID := pool.InternFieldSet([]string{"key"}).id
	ref1 := pool.InternStruct(fsID, []*khifile.InternedValue{{Kind: &khifile.InternedValue_Int64Value{Int64Value: 1}}})
	ref2 := pool.InternStruct(fsID, []*khifile.InternedValue{{Kind: &khifile.InternedValue_Int64Value{Int64Value: 2}}})

	// Simulate an orphaned struct ID in idToStruct (e.g. from concurrent InternStruct collision).
	orphanedID := idGen.New(IDStruct)
	pool.idToStruct.Store(orphanedID, (*khifile.InternedStruct)(nil))

	var refs []*InternStructRef
	for ref := range pool.StructRefs() {
		refs = append(refs, ref)
	}

	if len(refs) != 2 {
		t.Fatalf("StructRefs() returned %d refs, want 2 (orphaned IDs should be excluded)", len(refs))
	}

	testCases := []struct {
		name   string
		idx    int
		wantID uint32
	}{
		{
			name:   "first struct is ref1",
			idx:    0,
			wantID: ref1.id,
		},
		{
			name:   "second struct is ref2",
			idx:    1,
			wantID: ref2.id,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.idx >= len(refs) {
				t.Fatalf("Index %d out of range", tc.idx)
			}
			got := refs[tc.idx].id
			if got != tc.wantID {
				t.Errorf("refs[%d].id = %d, want %d", tc.idx, got, tc.wantID)
			}
		})
	}
}

func TestNewInternPoolFromChunk(t *testing.T) {
	str1ID := uint32(1)
	str1Val := "key"
	fs1ID := uint32(1)
	struct1ID := uint32(1)

	chunk := &pb.InterningPoolChunk{
		Strings: []*pb.InternString{
			{Id: &str1ID, Value: &str1Val},
		},
		FieldPathSets: []*pb.InternFieldPathSet{
			{Id: &fs1ID, FieldPathStringIds: []uint32{str1ID}},
		},
		Structs: []*khifile.InternedStruct{
			{
				Id:             &struct1ID,
				FieldPathSetId: &fs1ID,
				Values: []*khifile.InternedValue{
					{Kind: &khifile.InternedValue_StringValue{StringValue: str1ID}},
				},
			},
		},
	}

	testCases := []struct {
		name       string
		poolChunk  *pb.InterningPoolChunk
		queryID    uint32
		wantFound  bool
		wantString string
	}{
		{
			name:       "resolves struct from populated chunk",
			poolChunk:  chunk,
			queryID:    struct1ID,
			wantFound:  true,
			wantString: "key",
		},
		{
			name:       "returns nil for non-existent struct ID",
			poolChunk:  chunk,
			queryID:    999,
			wantFound:  false,
			wantString: "",
		},
		{
			name:       "handles nil chunk gracefully",
			poolChunk:  nil,
			queryID:    struct1ID,
			wantFound:  false,
			wantString: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := NewInternPoolFromChunk(tc.poolChunk)
			gotStruct := pool.ResolveStructFromID(tc.queryID)
			if (gotStruct != nil) != tc.wantFound {
				t.Fatalf("ResolveStructFromID(%d) = %v, wantFound = %v", tc.queryID, gotStruct, tc.wantFound)
			}
			if tc.wantFound {
				if diff := cmp.Diff(tc.poolChunk.Structs[0], gotStruct, protocmp.Transform()); diff != "" {
					t.Errorf("ResolveStructFromID() struct mismatch (-want +got):\n%s", diff)
				}
				resolvedStr := pool.resolveStringFromID(gotStruct.Values[0].GetStringValue())
				if resolvedStr != tc.wantString {
					t.Errorf("resolveStringFromID() = %q, want %q", resolvedStr, tc.wantString)
				}
			}
		})
	}
}

func TestInternPool_IngestChunk(t *testing.T) {
	str1ID := uint32(1)
	str1Val := "status"
	str2ID := uint32(2)
	str2Val := "Running"
	fs1ID := uint32(1)
	struct1ID := uint32(10)

	stringChunk := &pb.InterningPoolChunk{
		Strings: []*pb.InternString{
			{Id: &str1ID, Value: &str1Val},
			{Id: &str2ID, Value: &str2Val},
		},
	}

	fieldPathChunk := &pb.InterningPoolChunk{
		FieldPathSets: []*pb.InternFieldPathSet{
			{Id: &fs1ID, FieldPathStringIds: []uint32{str1ID}},
		},
	}

	structChunk := &pb.InterningPoolChunk{
		Structs: []*khifile.InternedStruct{
			{
				Id:             &struct1ID,
				FieldPathSetId: &fs1ID,
				Values: []*khifile.InternedValue{
					{Kind: &khifile.InternedValue_StringValue{StringValue: str2ID}},
				},
			},
		},
	}

	testCases := []struct {
		name       string
		chunks     []*pb.InterningPoolChunk
		queryID    uint32
		wantFound  bool
		wantString string
	}{
		{
			name:       "sequentially ingests multiple chunks for strings, field paths, and structs",
			chunks:     []*pb.InterningPoolChunk{stringChunk, fieldPathChunk, structChunk},
			queryID:    struct1ID,
			wantFound:  true,
			wantString: "Running",
		},
		{
			name:       "handles nil chunk during multi-chunk ingestion",
			chunks:     []*pb.InterningPoolChunk{stringChunk, nil, fieldPathChunk, structChunk},
			queryID:    struct1ID,
			wantFound:  true,
			wantString: "Running",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := NewInternPool(nil)
			for _, c := range tc.chunks {
				pool.IngestChunk(c)
			}

			gotStruct := pool.ResolveStructFromID(tc.queryID)
			if (gotStruct != nil) != tc.wantFound {
				t.Fatalf("ResolveStructFromID(%d) = %v, wantFound = %v", tc.queryID, gotStruct, tc.wantFound)
			}
			if tc.wantFound {
				resolvedStr := pool.resolveStringFromID(gotStruct.Values[0].GetStringValue())
				if resolvedStr != tc.wantString {
					t.Errorf("resolveStringFromID() = %q, want %q", resolvedStr, tc.wantString)
				}
			}
		})
	}
}

func TestServerInternPool(t *testing.T) {
	testCases := []struct {
		name       string
		clientStrs []string
		serverStrs []string
		checkStr   string
		wantID     uint32
	}{
		{
			name:       "server string not in client pool gets server id",
			clientStrs: []string{"client-only"},
			serverStrs: []string{"server-only"},
			checkStr:   "server-only",
			wantID:     ServerStringIDBase + 1,
		},
		{
			name:       "server string already in client pool reuses client id",
			clientStrs: []string{"shared-string"},
			serverStrs: []string{"shared-string"},
			checkStr:   "shared-string",
			wantID:     1,
		},
		{
			name:       "multiple strings with mixed client reuse",
			clientStrs: []string{"first-client", "second-client"},
			serverStrs: []string{"second-client", "server-first"},
			checkStr:   "server-first",
			wantID:     ServerStringIDBase + 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idGen := NewIDGenerator()
			clientPool := NewInternPool(idGen)
			serverPool := NewServerInternPool(clientPool, idGen)

			for _, s := range tc.clientStrs {
				clientPool.InternString(s)
			}
			for _, s := range tc.serverStrs {
				serverPool.InternString(s)
			}

			ref := serverPool.InternString(tc.checkStr)
			if ref.id != tc.wantID {
				t.Errorf("InternString(%q) id = %d, want %d", tc.checkStr, ref.id, tc.wantID)
			}
			if resolved := ref.Resolve(); resolved != tc.checkStr {
				t.Errorf("Resolve() = %q, want %q", resolved, tc.checkStr)
			}
		})
	}
}
