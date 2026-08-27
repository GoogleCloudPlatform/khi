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
	"slices"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/RoaringBitmap/roaring/v2"
)

// Encode encodes a slice of uint32 IDs into a compact SparseBitset.
func Encode(ids []uint32) *apiv1.SparseBitset {
	blockMap := make(map[uint32]uint32)
	for _, id := range ids {
		blockIdx := id / 32
		bitOffset := id % 32
		blockMap[blockIdx] |= (1 << bitOffset)
	}

	indices := make([]uint32, 0, len(blockMap))
	for idx := range blockMap {
		indices = append(indices, idx)
	}
	slices.Sort(indices)

	masks := make([]uint32, 0, len(indices))
	for _, idx := range indices {
		masks = append(masks, blockMap[idx])
	}

	return &apiv1.SparseBitset{
		Indices: indices,
		Masks:   masks,
	}
}

// EncodeFilterResult encodes matched entity IDs against totalCount sequential 1-indexed IDs into a FilterResultMode and SparseBitset.
// It automatically selects either INCLUDE or EXCLUDE mode to minimize payload size.
func EncodeFilterResult(totalCount int, matchedIDs *roaring.Bitmap) (apiv1.FilterResultMode, *apiv1.SparseBitset) {
	if matchedIDs == nil {
		matchedIDs = roaring.NewBitmap()
	}
	matchedCount := int(matchedIDs.GetCardinality())

	if matchedCount <= totalCount/2 {
		return apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE, Encode(matchedIDs.ToArray())
	}

	excludedCount := totalCount - matchedCount
	if excludedCount < 0 {
		excludedCount = 0
	}
	excludedIDs := make([]uint32, 0, excludedCount)
	for id := 1; id <= totalCount; id++ {
		if !matchedIDs.Contains(uint32(id)) {
			excludedIDs = append(excludedIDs, uint32(id))
		}
	}
	return apiv1.FilterResultMode_FILTER_RESULT_MODE_EXCLUDE, Encode(excludedIDs)
}

// Decode decodes a SparseBitset into a roaring.Bitmap of IDs.
func Decode(bitset *apiv1.SparseBitset) *roaring.Bitmap {
	bm := roaring.NewBitmap()
	if bitset == nil {
		return bm
	}
	for i, blockIdx := range bitset.Indices {
		if i >= len(bitset.Masks) {
			break
		}
		mask := bitset.Masks[i]
		base := blockIdx * 32
		for bit := uint32(0); bit < 32; bit++ {
			if (mask & (1 << bit)) != 0 {
				bm.Add(base + bit)
			}
		}
	}
	return bm
}
