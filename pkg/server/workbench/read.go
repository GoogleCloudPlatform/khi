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
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/RoaringBitmap/roaring/v2"
)

// TimelineSegment represents a single node in a hierarchical timeline path.
type TimelineSegment struct {
	Type string
	Name string
}

// FormatTimelinePath formats timeline segments as "[Type] Name > [Type] Name".
func FormatTimelinePath(segments []TimelineSegment) string {
	if len(segments) == 0 {
		return ""
	}
	parts := make([]string, len(segments))
	for i, seg := range segments {
		parts[i] = fmt.Sprintf("[%s] %s", seg.Type, seg.Name)
	}
	return strings.Join(parts, " > ")
}

// BuildTimelineQuery builds a CEL query matching the given timeline hierarchy path.
func BuildTimelineQuery(segments []TimelineSegment) string {
	if len(segments) == 0 {
		return ""
	}
	parts := make([]string, 0, len(segments))
	for _, seg := range segments {
		key := strings.ToLower(seg.Type)
		escapedKey := strings.ReplaceAll(key, `\`, `\\`)
		escapedKey = strings.ReplaceAll(escapedKey, `"`, `\"`)
		escapedVal := strings.ReplaceAll(seg.Name, `\`, `\\`)
		escapedVal = strings.ReplaceAll(escapedVal, `"`, `\"`)
		parts = append(parts, fmt.Sprintf(`path["%s"] == "%s"`, escapedKey, escapedVal))
	}
	return strings.Join(parts, " && ")
}

// TimelineSegments reconstructs the timeline hierarchy segments from root to the given timeline ID.
func (s *SearchIndex) TimelineSegments(timelineID uint32) ([]TimelineSegment, bool) {
	if s == nil || s.TimelineMap == nil {
		return nil, false
	}

	cur, exists := s.TimelineMap[timelineID]
	if !exists || cur == nil {
		return nil, false
	}

	var segments []TimelineSegment
	visited := make(map[uint32]bool)

	for cur != nil {
		if visited[cur.ID] {
			break
		}
		visited[cur.ID] = true
		segments = append(segments, TimelineSegment{
			Type: cur.TimelineType,
			Name: cur.Name,
		})
		if cur.ParentID == 0 {
			break
		}
		cur = s.TimelineMap[cur.ParentID]
	}

	slices.Reverse(segments)
	return segments, true
}

// TimelineSegments returns the hierarchical timeline segments for the specified timeline ID.
func (w *Workbench) TimelineSegments(timelineID uint32) ([]TimelineSegment, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.closed {
		return nil, ErrWorkbenchClosed
	}
	if w.searchIndex == nil {
		return nil, fmt.Errorf("search index is not ready")
	}

	segments, ok := w.searchIndex.TimelineSegments(timelineID)
	if !ok {
		return nil, fmt.Errorf("timeline ID %d not found", timelineID)
	}
	return segments, nil
}

// InspectionTimeRange returns the inspection time range from HeaderMetadata or indexed timestamps.
func (w *Workbench) InspectionTimeRange() (time.Time, time.Time) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, chunk := range w.metadataChunks {
		if chunk == nil {
			continue
		}
		for _, item := range chunk.GetMetadata() {
			if header := item.GetHeader(); header != nil {
				startSec := header.GetStartTimeUnixSeconds()
				endSec := header.GetEndTimeUnixSeconds()
				if startSec > 0 && endSec > 0 {
					return time.Unix(startSec, 0).UTC(), time.Unix(endSec, 0).UTC()
				}
			}
		}
	}

	if w.searchIndex == nil {
		return time.Time{}, time.Time{}
	}

	var minNs int64 = -1
	var maxNs int64 = -1

	update := func(ts int64) {
		if ts <= 0 {
			return
		}
		if minNs == -1 || ts < minNs {
			minNs = ts
		}
		if maxNs == -1 || ts > maxNs {
			maxNs = ts
		}
	}

	for i := range w.searchIndex.Logs {
		update(w.searchIndex.Logs[i].Timestamp)
	}

	for _, tl := range w.searchIndex.Timelines {
		if tl == nil {
			continue
		}
		for _, evt := range tl.Events {
			update(evt.Timestamp)
		}
		for _, rev := range tl.Revisions {
			update(rev.ChangedTime)
		}
	}

	if minNs == -1 || maxNs == -1 {
		return time.Time{}, time.Time{}
	}

	return time.Unix(0, minNs).UTC(), time.Unix(0, maxNs).UTC()
}

// Filter specifies CEL and time range filter parameters for timeline reading.
type Filter struct {
	TimelineQuery               string     `json:"timelineQuery,omitempty" jsonschema:"CEL expression to select timelines to include."`
	TimelineExclusionQuery      string     `json:"timelineExclusionQuery,omitempty" jsonschema:"CEL expression to select timelines to exclude along with their descendants."`
	LogQuery                    string     `json:"logQuery,omitempty" jsonschema:"CEL expression to filter logs."`
	StartTime                   *time.Time `json:"startTime,omitempty" jsonschema:"Inclusive start of the time range in RFC3339 format."`
	EndTime                     *time.Time `json:"endTime,omitempty" jsonschema:"Inclusive end of the time range in RFC3339 format."`
	ExcludeTimelinesWithoutLogs *bool      `json:"excludeTimelinesWithoutLogs,omitempty" jsonschema:"Whether to exclude timelines that have no logs matching the filter."`
}

// AppliedFilter represents the concrete filter settings after defaults and time range have been resolved.
type AppliedFilter struct {
	TimelineQuery               string
	TimelineExclusionQuery      string
	LogQuery                    string
	StartTime                   time.Time
	EndTime                     time.Time
	ExcludeTimelinesWithoutLogs bool
}

// ToPipelineParams converts an AppliedFilter to FilterPipelineParams for execution.
func (a AppliedFilter) ToPipelineParams() FilterPipelineParams {
	var start, end *time.Time
	if !a.StartTime.IsZero() {
		start = &a.StartTime
	}
	if !a.EndTime.IsZero() {
		end = &a.EndTime
	}
	return FilterPipelineParams{
		TimelineQuery:          a.TimelineQuery,
		TimelineExclusionQuery: a.TimelineExclusionQuery,
		LogQuery:               a.LogQuery,
		ExcludeNoLogs:          a.ExcludeTimelinesWithoutLogs,
		FilterStartTime:        start,
		FilterEndTime:          end,
	}
}

// FilterOutput contains the applied filter configuration and matching bitmap results.
type FilterOutput struct {
	Applied     AppliedFilter
	TimelineIDs *roaring.Bitmap
	LogIDs      *roaring.Bitmap
}

// ValidateFilter validates the CEL expressions within a Filter.
// If invalid, it returns the offending field name and the validation error.
func ValidateFilter(filter Filter) (string, error) {
	if err := cel.ValidateTimelineQuery(filter.TimelineQuery); err != nil {
		return "filter.timelineQuery", err
	}
	if err := cel.ValidateTimelineQuery(filter.TimelineExclusionQuery); err != nil {
		return "filter.timelineExclusionQuery", err
	}
	if err := cel.ValidateLogQuery(filter.LogQuery); err != nil {
		return "filter.logQuery", err
	}
	return "", nil
}

// ResolveFilter fills default values for time range and boolean options in Filter.
func (w *Workbench) ResolveFilter(filter Filter) AppliedFilter {
	startTime, endTime := w.InspectionTimeRange()
	if filter.StartTime != nil {
		startTime = filter.StartTime.UTC()
	}
	if filter.EndTime != nil {
		endTime = filter.EndTime.UTC()
	}

	excludeNoLogs := strings.TrimSpace(filter.LogQuery) != ""
	if filter.ExcludeTimelinesWithoutLogs != nil {
		excludeNoLogs = *filter.ExcludeTimelinesWithoutLogs
	}

	return AppliedFilter{
		TimelineQuery:               filter.TimelineQuery,
		TimelineExclusionQuery:      filter.TimelineExclusionQuery,
		LogQuery:                    filter.LogQuery,
		StartTime:                   startTime,
		EndTime:                     endTime,
		ExcludeTimelinesWithoutLogs: excludeNoLogs,
	}
}

// ExecuteFilter validates and executes the filter pipeline, returning matching timeline and log IDs.
func (w *Workbench) ExecuteFilter(ctx context.Context, filter Filter) (*FilterOutput, error) {
	if field, valErr := ValidateFilter(filter); valErr != nil {
		return nil, fmt.Errorf("invalid %s: %w", field, valErr)
	}

	applied := w.ResolveFilter(filter)

	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return nil, ErrWorkbenchClosed
	}
	if w.searchIndex == nil {
		w.mu.RUnlock()
		return nil, fmt.Errorf("search index is not ready")
	}
	index := w.searchIndex
	w.mu.RUnlock()

	params := applied.ToPipelineParams()
	pipeline := NewDefaultPipeline(params)
	filterCtx, err := pipeline.ExecuteToFilterContext(ctx, index, nil)
	if err != nil {
		return nil, err
	}

	return &FilterOutput{
		Applied:     applied,
		TimelineIDs: filterCtx.TimelineIDs,
		LogIDs:      filterCtx.LogIDs,
	}, nil
}
