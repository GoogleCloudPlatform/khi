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
	"context"
	"math"
	"time"

	"github.com/RoaringBitmap/roaring/v2"
)

// TimeRangeFilter filters timelines based on whether they have activity within a specified time range.
type TimeRangeFilter struct {
	hasRange bool
	startNs  int64
	endNs    int64
}

var _ TimelineFilter = (*TimeRangeFilter)(nil)

// NewTimeRangeFilter creates a new TimeRangeFilter.
// If both startTime and endTime are nil, no time range filtering is performed.
// If either is non-nil, any missing boundary defaults to math.MinInt64 or math.MaxInt64.
func NewTimeRangeFilter(startTime, endTime *time.Time) *TimeRangeFilter {
	if startTime == nil && endTime == nil {
		return &TimeRangeFilter{
			hasRange: false,
		}
	}
	startNs := int64(math.MinInt64)
	if startTime != nil {
		startNs = startTime.UnixNano()
	}
	endNs := int64(math.MaxInt64)
	if endTime != nil {
		endNs = endTime.UnixNano()
	}
	return &TimeRangeFilter{
		hasRange: true,
		startNs:  startNs,
		endNs:    endNs,
	}
}

// Name returns the display name of this filter stage.
func (f *TimeRangeFilter) Name() string {
	return "Time range filter"
}

// Process evaluates timeline activity within the specified time range and populates the matching timeline IDs into filterCtx.
func (f *TimeRangeFilter) Process(
	ctx context.Context,
	filterCtx *FilterContext,
	index *SearchIndex,
	report ProgressReporter,
) error {
	candidateIDs := filterCtx.TimelineIDs.ToArray()
	totalCandidates := uint32(len(candidateIDs))
	if !f.hasRange {
		filterCtx.HasTimeRange = false
		if report != nil {
			if err := report(f.Name(), totalCandidates, totalCandidates); err != nil {
				return err
			}
		}
		return nil
	}

	filterCtx.HasTimeRange = true
	filterCtx.StartTimeNs = f.startNs
	filterCtx.EndTimeNs = f.endNs

	matched := roaring.NewBitmap()
	if totalCandidates == 0 {
		if report != nil {
			if err := report(f.Name(), 0, 0); err != nil {
				return err
			}
		}
		filterCtx.TimelineIDs = matched
		return nil
	}

	for i, id := range candidateIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if tl := index.TimelineMap[id]; tl != nil && tl.HasActivityInRange(f.startNs, f.endNs) {
			matched.Add(id)
		}
		current := uint32(i + 1)
		if current%1000 == 0 || current == totalCandidates {
			if report != nil {
				if err := report(f.Name(), current, totalCandidates); err != nil {
					return err
				}
			}
		}
	}

	filterCtx.TimelineIDs = matched
	return nil
}
