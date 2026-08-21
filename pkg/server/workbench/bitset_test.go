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
	"testing"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestBuildSparseBitset(t *testing.T) {
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
			got := BuildSparseBitset(tc.ids)
			if diff := cmp.Diff(tc.wantBitset, got, protocmp.Transform()); diff != "" {
				t.Errorf("BuildSparseBitset() bitset mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEncodeFilterResultBitset(t *testing.T) {
	testCases := []struct {
		name       string
		totalCount int
		matchedIDs map[uint32]struct{}
		wantMode   apiv1.FilterResultMode
		wantBitset *apiv1.SparseBitset
	}{
		{
			name:       "empty dataset and empty matched",
			totalCount: 0,
			matchedIDs: map[uint32]struct{}{},
			wantMode:   apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE,
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{},
				Masks:   []uint32{},
			},
		},
		{
			name:       "0% matched selects INCLUDE mode with empty bitset",
			totalCount: 5,
			matchedIDs: map[uint32]struct{}{},
			wantMode:   apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE,
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{},
				Masks:   []uint32{},
			},
		},
		{
			name:       "100% matched selects EXCLUDE mode with empty bitset",
			totalCount: 4,
			matchedIDs: map[uint32]struct{}{
				1: {}, 2: {}, 3: {}, 4: {},
			},
			wantMode: apiv1.FilterResultMode_FILTER_RESULT_MODE_EXCLUDE,
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{},
				Masks:   []uint32{},
			},
		},
		{
			name:       "minority matched (<= 50%) selects INCLUDE mode",
			totalCount: 65,
			matchedIDs: map[uint32]struct{}{
				1:  {}, // block 0, bit 1 -> 1 << 1 = 0x2
				31: {}, // block 0, bit 31 -> 1 << 31 = 0x80000000
				64: {}, // block 2, bit 0 -> 1 << 0 = 0x1
			},
			wantMode: apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE,
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{0, 2},
				Masks:   []uint32{0x80000002, 0x1},
			},
		},
		{
			name:       "majority matched (> 50%) selects EXCLUDE mode",
			totalCount: 5,
			matchedIDs: map[uint32]struct{}{
				1: {}, 2: {}, 3: {}, 4: {},
			},
			wantMode: apiv1.FilterResultMode_FILTER_RESULT_MODE_EXCLUDE,
			// Excluded item is 5 (block 0, bit 5 -> 1 << 5 = 0x20)
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{0},
				Masks:   []uint32{1 << 5},
			},
		},
		{
			name:       "boundary spanning multiple blocks",
			totalCount: 96,
			matchedIDs: map[uint32]struct{}{
				32: {}, // block 1, bit 0
				63: {}, // block 1, bit 31
				96: {}, // block 3, bit 0
			},
			wantMode: apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE,
			wantBitset: &apiv1.SparseBitset{
				Indices: []uint32{1, 3},
				Masks:   []uint32{0x80000001, 0x1},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotMode, gotBitset := EncodeFilterResultBitset(tc.totalCount, tc.matchedIDs)
			if gotMode != tc.wantMode {
				t.Errorf("EncodeFilterResultBitset mode mismatch (-want +got):\n%s", cmp.Diff(tc.wantMode, gotMode))
			}
			if diff := cmp.Diff(tc.wantBitset, gotBitset, protocmp.Transform()); diff != "" {
				t.Errorf("EncodeFilterResultBitset bitset mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
