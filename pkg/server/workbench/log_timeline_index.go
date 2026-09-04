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

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
)

// LogTimelineCSRIndex provides an in-memory reverse index mapping log IDs to timeline IDs using Compressed Sparse Row (CSR) layout.
// Offsets has length maxLogID + 2, and timelineIDs holds contiguous timeline IDs.
type LogTimelineCSRIndex struct {
	offsets     []uint32
	timelineIDs []uint32
}

// NewLogTimelineCSRIndex builds a CSR reverse index from the provided timelines and maximum log ID.
func NewLogTimelineCSRIndex(maxLogID uint32, timelines []*cel.TimelineData) *LogTimelineCSRIndex {
	if maxLogID == 0 || len(timelines) == 0 {
		return &LogTimelineCSRIndex{
			offsets:     make([]uint32, maxLogID+2),
			timelineIDs: nil,
		}
	}

	// Pre-extract unique log IDs per timeline to avoid duplicate bindings.
	timelineUniqueLogs := make([][]uint32, len(timelines))
	counts := make([]uint32, maxLogID+1)

	for i, tl := range timelines {
		if tl == nil {
			continue
		}
		totalItems := len(tl.Events) + len(tl.Revisions)
		if totalItems == 0 {
			continue
		}
		localLogs := make([]uint32, 0, totalItems)
		for _, evt := range tl.Events {
			if evt.LogID > 0 && evt.LogID <= maxLogID {
				localLogs = append(localLogs, evt.LogID)
			}
		}
		for _, rev := range tl.Revisions {
			if rev.LogID > 0 && rev.LogID <= maxLogID {
				localLogs = append(localLogs, rev.LogID)
			}
		}
		if len(localLogs) == 0 {
			continue
		}
		slices.Sort(localLogs)
		localLogs = slices.Compact(localLogs)
		timelineUniqueLogs[i] = localLogs

		for _, logID := range localLogs {
			counts[logID]++
		}
	}

	// Build CSR offsets using prefix sums.
	offsets := make([]uint32, maxLogID+2)
	for i := uint32(1); i <= maxLogID; i++ {
		offsets[i+1] = offsets[i] + counts[i]
	}

	totalEntries := offsets[maxLogID+1]
	timelineIDs := make([]uint32, totalEntries)
	curPositions := make([]uint32, maxLogID+1)
	copy(curPositions, offsets[:maxLogID+1])

	// Populate flat timeline IDs slice.
	for i, tl := range timelines {
		if tl == nil {
			continue
		}
		for _, logID := range timelineUniqueLogs[i] {
			pos := curPositions[logID]
			timelineIDs[pos] = tl.ID
			curPositions[logID]++
		}
	}

	return &LogTimelineCSRIndex{
		offsets:     offsets,
		timelineIDs: timelineIDs,
	}
}

// GetTimelineIDs retrieves all unique timeline IDs containing the specified log ID in O(1) time.
func (idx *LogTimelineCSRIndex) GetTimelineIDs(logID uint32) []uint32 {
	if idx == nil || logID == 0 || int(logID)+1 >= len(idx.offsets) {
		return nil
	}
	start := idx.offsets[logID]
	end := idx.offsets[logID+1]
	if start >= end || end > uint32(len(idx.timelineIDs)) {
		return nil
	}
	return idx.timelineIDs[start:end]
}
