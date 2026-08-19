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

package workbench

import (
	"errors"
	"testing"

	khifile "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/google/go-cmp/cmp"
)

func TestWorkbench_Lifecycle(t *testing.T) {
	wb := NewWorkbench("user-session-1", "inspection-100")

	if got := wb.ID(); got != "user-session-1" {
		t.Errorf("ID() = %q, want %q", got, "user-session-1")
	}

	if got := wb.InspectionID(); got != "inspection-100" {
		t.Errorf("InspectionID() = %q, want %q", got, "inspection-100")
	}

	if wb.IsClosed() {
		t.Errorf("expected new workbench not to be closed")
	}

	// Close marks as closed
	wb.Close()
	if !wb.IsClosed() {
		t.Errorf("expected workbench to be closed after Close()")
	}
}

func TestWorkbench_ReadStructYAML(t *testing.T) {
	str1ID := uint32(1)
	str1Val := "message"
	str2ID := uint32(2)
	str2Val := "hello world"

	fs1ID := uint32(1)
	struct1ID := uint32(1)

	chunk := &khifilev6.InterningPoolChunk{
		Strings: []*khifilev6.InternString{
			{Id: &str1ID, Value: &str1Val},
			{Id: &str2ID, Value: &str2Val},
		},
		FieldPathSets: []*khifilev6.InternFieldPathSet{
			{Id: &fs1ID, FieldPathStringIds: []uint32{str1ID}},
		},
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
		name      string
		setupWb   func() *Workbench
		structID  uint32
		wantErrIs error
		wantYAML  string
	}{
		{
			name: "successfully decodes struct to YAML",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.internPool.IngestChunk(chunk)
				return wb
			},
			structID: struct1ID,
			wantYAML: "message: hello world\n",
		},
		{
			name: "successfully decodes struct with multi-chunk ingestion",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				c1 := &khifilev6.InterningPoolChunk{Strings: chunk.Strings}
				c2 := &khifilev6.InterningPoolChunk{FieldPathSets: chunk.FieldPathSets}
				c3 := &khifilev6.InterningPoolChunk{Structs: chunk.Structs}
				wb.internPool.IngestChunk(c1)
				wb.internPool.IngestChunk(c2)
				wb.internPool.IngestChunk(c3)
				return wb
			},
			structID: struct1ID,
			wantYAML: "message: hello world\n",
		},
		{
			name: "returns ErrStructNotFound when struct ID does not exist",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.internPool.IngestChunk(chunk)
				return wb
			},
			structID:  999,
			wantErrIs: ErrStructNotFound,
		},
		{
			name: "returns ErrStructNotFound when intern pool is empty",
			setupWb: func() *Workbench {
				return NewWorkbench("wb-1", "insp-1")
			},
			structID:  struct1ID,
			wantErrIs: ErrStructNotFound,
		},
		{
			name: "returns ErrWorkbenchClosed when workbench is closed",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.internPool.IngestChunk(chunk)
				wb.Close()
				return wb
			},
			structID:  struct1ID,
			wantErrIs: ErrWorkbenchClosed,
		},
		{
			name: "returns pre-serialized YAML from SearchIndex.StructYAMLs when available",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.searchIndex = &SearchIndex{
					StructYAMLs: map[uint32]string{
						struct1ID: "message: cached in index\n",
					},
				}
				return wb
			},
			structID: struct1ID,
			wantYAML: "message: cached in index\n",
		},
		{
			name: "falls back to intern pool when struct ID is not found in SearchIndex.StructYAMLs",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.internPool.IngestChunk(chunk)
				wb.searchIndex = &SearchIndex{
					StructYAMLs: map[uint32]string{
						999: "message: other\n",
					},
				}
				return wb
			},
			structID: struct1ID,
			wantYAML: "message: hello world\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := tc.setupWb()
			gotYAML, err := wb.ReadStructYAML(tc.structID)
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("ReadStructYAML(%d) error = %v, want %v", tc.structID, err, tc.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadStructYAML(%d) unexpected error = %v", tc.structID, err)
			}
			if diff := cmp.Diff(tc.wantYAML, gotYAML); diff != "" {
				t.Errorf("ReadStructYAML(%d) YAML mismatch (-want +got):\n%s", tc.structID, diff)
			}
		})
	}
}
