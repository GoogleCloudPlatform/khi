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
	"slices"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
)

// BuildSparseBitset encodes a slice of uint32 IDs into a compact SparseBitset.
func BuildSparseBitset(ids []uint32) *apiv1.SparseBitset {
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

// EncodeFilterResultBitset encodes matched entity IDs against totalCount sequential 1-indexed IDs into a FilterResultMode and SparseBitset.
// It automatically selects either INCLUDE or EXCLUDE mode to minimize payload size.
func EncodeFilterResultBitset(totalCount int, matchedIDs map[uint32]struct{}) (apiv1.FilterResultMode, *apiv1.SparseBitset) {
	matchedCount := len(matchedIDs)

	if matchedCount <= totalCount/2 {
		targetIDs := make([]uint32, 0, matchedCount)
		for id := range matchedIDs {
			targetIDs = append(targetIDs, id)
		}
		return apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE, BuildSparseBitset(targetIDs)
	}

	excludedCount := totalCount - matchedCount
	if excludedCount < 0 {
		excludedCount = 0
	}
	targetIDs := make([]uint32, 0, excludedCount)
	for id := 1; id <= totalCount; id++ {
		if _, ok := matchedIDs[uint32(id)]; !ok {
			targetIDs = append(targetIDs, uint32(id))
		}
	}
	return apiv1.FilterResultMode_FILTER_RESULT_MODE_EXCLUDE, BuildSparseBitset(targetIDs)
}
