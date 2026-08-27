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

package sparsebitset

import (
	"testing"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/RoaringBitmap/roaring/v2"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestEncode(t *testing.T) {
	testCases := []struct {
		name       string
		ids        []uint32
		wantBitset *apiv1.SparseBitset
	}{
		{
			name: "empty slice",
			ids:  []uint32{},
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{},
				Masks:   []uint32{},
			},
		},
		{
			name: "single block boundary IDs",
			ids:  []uint32{0, 1, 31},
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{0},
				Masks:   []uint32{(1 << 0) | (1 << 1) | (1 << 31)},
			},
		},
		{
			name: "multiple blocks with sparse gaps",
			ids:  []uint32{1, 32, 64},
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{0, 1, 2},
				Masks:   []uint32{1 << 1, 1 << 0, 1 << 0},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Encode(tc.ids)
			if diff := cmp.Diff(tc.wantBitset, got, protocmp.Transform()); diff != "" {
				t.Errorf("Encode() bitset mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEncodeFilterResult(t *testing.T) {
	testCases := []struct {
		name       string
		totalCount int
		matchedIDs *roaring.Bitmap
		wantMode   apiv1.FilterResultMode
		wantBitset *apiv1.SparseBitset
	}{
		{
			name:       "empty dataset and empty matched",
			totalCount: 0,
			matchedIDs: roaring.NewBitmap(),
			wantMode:   apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE,
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{},
				Masks:   []uint32{},
			},
		},
		{
			name:       "0% matched selects INCLUDE mode with empty bitset",
			totalCount: 5,
			matchedIDs: roaring.NewBitmap(),
			wantMode:   apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE,
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{},
				Masks:   []uint32{},
			},
		},
		{
			name:       "100% matched selects EXCLUDE mode with empty bitset",
			totalCount: 4,
			matchedIDs: roaring.BitmapOf(1, 2, 3, 4),
			wantMode:   apiv1.FilterResultMode_FILTER_RESULT_MODE_EXCLUDE,
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{},
				Masks:   []uint32{},
			},
		},
		{
			name:       "below 50% matched selects INCLUDE mode with matched IDs",
			totalCount: 10,
			matchedIDs: roaring.BitmapOf(2, 4),
			wantMode:   apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE,
			wantBitset: Encode([]uint32{2, 4}),
		},
		{
			name:       "above 50% matched selects EXCLUDE mode with inverted IDs",
			totalCount: 10,
			matchedIDs: roaring.BitmapOf(1, 2, 3, 4, 5, 6),
			wantMode:   apiv1.FilterResultMode_FILTER_RESULT_MODE_EXCLUDE,
			wantBitset: Encode([]uint32{7, 8, 9, 10}),
		},
		{
			name:       "exactly 50% matched selects INCLUDE mode",
			totalCount: 4,
			matchedIDs: roaring.BitmapOf(1, 3),
			wantMode:   apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE,
			wantBitset: Encode([]uint32{1, 3}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotMode, gotBitset := EncodeFilterResult(tc.totalCount, tc.matchedIDs)
			if gotMode != tc.wantMode {
				t.Errorf("EncodeFilterResult() mode mismatch: got %v, want %v", gotMode, tc.wantMode)
			}
			if diff := cmp.Diff(tc.wantBitset, gotBitset, protocmp.Transform()); diff != "" {
				t.Errorf("EncodeFilterResult() bitset mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	testCases := []struct {
		name     string
		input    *apiv1.SparseBitset
		wantBits []uint32
	}{
		{
			name:     "nil bitset returns empty bitmap",
			input:    nil,
			wantBits: []uint32{},
		},
		{
			name: "empty bitset returns empty bitmap",
			input: &apiv1.SparseBitset{
				Indices: []uint32{},
				Masks:   []uint32{},
			},
			wantBits: []uint32{},
		},
		{
			name: "single block decoded correctly",
			input: &apiv1.SparseBitset{
				Indices: []uint32{0},
				Masks:   []uint32{(1 << 0) | (1 << 5)},
			},
			wantBits: []uint32{0, 5},
		},
		{
			name: "multiple sparse blocks decoded correctly",
			input: &apiv1.SparseBitset{
				Indices: []uint32{1, 3},
				Masks:   []uint32{1 << 0, 1 << 31},
			},
			wantBits: []uint32{32, 3*32 + 31},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Decode(tc.input)
			want := roaring.NewBitmap()
			want.AddMany(tc.wantBits)

			if !got.Equals(want) {
				diff := cmp.Diff(want.ToArray(), got.ToArray())
				t.Errorf("Decode() bitmap mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
