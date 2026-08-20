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
	descendantTimelineIDs := make(map[uint32]struct{})
	var markDescendants func(id uint32)
	markDescendants = func(id uint32) {
		if _, exists := descendantTimelineIDs[id]; exists {
			return
		}
		descendantTimelineIDs[id] = struct{}{}
		if tl, ok := index.TimelineMap[id]; ok {
			for _, childID := range tl.ChildrenIDs {
				markDescendants(childID)
			}
		}
	}

	for id := range filterCtx.TimelineIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		markDescendants(id)
	}

	filterCtx.TimelineIDs = descendantTimelineIDs

	if report != nil {
		if err := report(f.Name(), uint32(len(filterCtx.TimelineIDs)), uint32(len(index.Timelines))); err != nil {
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
	ancestorTimelineIDs := make(map[uint32]struct{})
	var markAncestors func(id uint32)
	markAncestors = func(id uint32) {
		if _, exists := ancestorTimelineIDs[id]; exists {
			return
		}
		ancestorTimelineIDs[id] = struct{}{}
		if tl, ok := index.TimelineMap[id]; ok && tl.ParentID != 0 {
			markAncestors(tl.ParentID)
		}
	}

	for id := range filterCtx.TimelineIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		markAncestors(id)
	}

	filterCtx.TimelineIDs = ancestorTimelineIDs

	if report != nil {
		if err := report(f.Name(), uint32(len(filterCtx.TimelineIDs)), uint32(len(index.Timelines))); err != nil {
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
			return report(f.Name(), uint32(len(filterCtx.TimelineIDs)), uint32(len(index.Timelines)))
		}
		return nil
	}

	retainedTimelineIDs := make(map[uint32]struct{})
	for id := range filterCtx.TimelineIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if tl, ok := index.TimelineMap[id]; ok {
			hasLogs := false
			tl.ForEachLogID(func(logID uint32) bool {
				if _, exists := filterCtx.LogIDs[logID]; exists {
					hasLogs = true
					return false
				}
				return true
			})
			if hasLogs {
				retainedTimelineIDs[id] = struct{}{}
			}
		}
	}

	filterCtx.TimelineIDs = retainedTimelineIDs

	if report != nil {
		if err := report(f.Name(), uint32(len(filterCtx.TimelineIDs)), uint32(len(index.Timelines))); err != nil {
			return err
		}
	}

	return nil
}
