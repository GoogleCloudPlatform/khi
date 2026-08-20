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

	"github.com/RoaringBitmap/roaring/v2"
)

// IncludeDescendantsFilter expands the matching timeline set by recursively including all child and descendant timelines.
type IncludeDescendantsFilter struct{}

var _ TimelineFilter = (*IncludeDescendantsFilter)(nil)

// NewIncludeDescendantsFilter creates a new IncludeDescendantsFilter.
func NewIncludeDescendantsFilter() *IncludeDescendantsFilter {
	return &IncludeDescendantsFilter{}
}

// Name returns the display name of this filter stage.
func (f *IncludeDescendantsFilter) Name() string {
	return "Include descendants"
}

// Process recursively includes all descendant timeline IDs for each timeline currently in filterCtx.TimelineIDs.
func (f *IncludeDescendantsFilter) Process(
	ctx context.Context,
	filterCtx *FilterContext,
	index *SearchIndex,
	report ProgressReporter,
) error {
	descendantTimelineIDs := roaring.NewBitmap()
	var markDescendants func(id uint32)
	markDescendants = func(id uint32) {
		if descendantTimelineIDs.Contains(id) {
			return
		}
		descendantTimelineIDs.Add(id)
		if tl, ok := index.TimelineMap[id]; ok {
			for _, childID := range tl.ChildrenIDs {
				markDescendants(childID)
			}
		}
	}

	for _, id := range filterCtx.TimelineIDs.ToArray() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		markDescendants(id)
	}

	filterCtx.TimelineIDs = descendantTimelineIDs

	if report != nil {
		if err := report(f.Name(), uint32(filterCtx.TimelineIDs.GetCardinality()), uint32(len(index.Timelines))); err != nil {
			return err
		}
	}

	return nil
}

// IncludeAncestorsFilter expands the matching timeline set by recursively including all parent and ancestor timelines.
type IncludeAncestorsFilter struct{}

var _ TimelineFilter = (*IncludeAncestorsFilter)(nil)

// NewIncludeAncestorsFilter creates a new IncludeAncestorsFilter.
func NewIncludeAncestorsFilter() *IncludeAncestorsFilter {
	return &IncludeAncestorsFilter{}
}

// Name returns the display name of this filter stage.
func (f *IncludeAncestorsFilter) Name() string {
	return "Include ancestors"
}

// Process recursively includes all parent timeline IDs for each timeline currently in filterCtx.TimelineIDs.
func (f *IncludeAncestorsFilter) Process(
	ctx context.Context,
	filterCtx *FilterContext,
	index *SearchIndex,
	report ProgressReporter,
) error {
	ancestorTimelineIDs := roaring.NewBitmap()
	var markAncestors func(id uint32)
	markAncestors = func(id uint32) {
		if ancestorTimelineIDs.Contains(id) {
			return
		}
		ancestorTimelineIDs.Add(id)
		if tl, ok := index.TimelineMap[id]; ok && tl.ParentID != 0 {
			markAncestors(tl.ParentID)
		}
	}

	for _, id := range filterCtx.TimelineIDs.ToArray() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		markAncestors(id)
	}

	filterCtx.TimelineIDs = ancestorTimelineIDs

	if report != nil {
		if err := report(f.Name(), uint32(filterCtx.TimelineIDs.GetCardinality()), uint32(len(index.Timelines))); err != nil {
			return err
		}
	}

	return nil
}

// ExcludeNoLogsFilter filters out timelines that do not directly contain any logs matching the current filter criteria.
type ExcludeNoLogsFilter struct {
	enabled bool
}

var _ TimelineFilter = (*ExcludeNoLogsFilter)(nil)

// NewExcludeNoLogsFilter creates a new ExcludeNoLogsFilter.
func NewExcludeNoLogsFilter(enabled bool) *ExcludeNoLogsFilter {
	return &ExcludeNoLogsFilter{enabled: enabled}
}

// Name returns the display name of this filter stage.
func (f *ExcludeNoLogsFilter) Name() string {
	return "Exclude no logs"
}

// Process filters out timelines from filterCtx.TimelineIDs if enabled and if they contain zero matching logs in filterCtx.LogIDs.
func (f *ExcludeNoLogsFilter) Process(
	ctx context.Context,
	filterCtx *FilterContext,
	index *SearchIndex,
	report ProgressReporter,
) error {
	if !f.enabled {
		if report != nil {
			return report(f.Name(), uint32(filterCtx.TimelineIDs.GetCardinality()), uint32(len(index.Timelines)))
		}
		return nil
	}

	retainedTimelineIDs := roaring.NewBitmap()
	for _, id := range filterCtx.TimelineIDs.ToArray() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if tl, ok := index.TimelineMap[id]; ok {
			hasLogs := false
			tl.ForEachLogID(func(logID uint32) bool {
				if filterCtx.LogIDs.Contains(logID) {
					hasLogs = true
					return false
				}
				return true
			})
			if hasLogs {
				retainedTimelineIDs.Add(id)
			}
		}
	}

	filterCtx.TimelineIDs = retainedTimelineIDs

	if report != nil {
		if err := report(f.Name(), uint32(filterCtx.TimelineIDs.GetCardinality()), uint32(len(index.Timelines))); err != nil {
			return err
		}
	}

	return nil
}
